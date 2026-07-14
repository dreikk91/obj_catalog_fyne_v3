package data

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

const simPhoneLookupTimeout = 20 * time.Second

type combinedAdminProvider struct {
	contracts.AdminProvider
	lookup contracts.AdminObjectSIMLookupService
}

func (p combinedAdminProvider) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	if p.lookup == nil {
		return nil, errors.New("SIM lookup is not configured")
	}
	return p.lookup.FindObjectsBySIMPhone(phone, excludeObjN)
}

// FindObjectsBySIMPhone шукає використання SIM-номера в усіх підключених джерелах.
func (p *CombinedDataProvider) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	if p == nil {
		return nil, errors.New("combined provider is nil")
	}
	if len(p.sources) == 0 {
		return nil, errors.New("SIM lookup sources are not configured")
	}
	normalized := normalizeUASimPhone(phone)
	if len(normalized) != 10 || !strings.HasPrefix(normalized, "0") {
		return nil, nil
	}

	type lookupResult struct {
		items  []contracts.AdminSIMPhoneUsage
		source string
		err    error
	}
	results := make(chan lookupResult, len(p.sources))
	for i := range p.sources {
		source := &p.sources[i]
		go func() {
			label := simSourceDisplayName(source.Name)
			lookup, ok := source.Provider.(contracts.AdminObjectSIMLookupService)
			if !ok {
				results <- lookupResult{source: label, err: errors.New("джерело не підтримує перевірку SIM")}
				return
			}
			items, err := lookup.FindObjectsBySIMPhone(normalized, excludeObjN)
			results <- lookupResult{items: items, source: label, err: err}
		}()
	}

	var (
		usages []contracts.AdminSIMPhoneUsage
		errs   []error
	)
	for range p.sources {
		result := <-results
		if result.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", result.source, result.err))
		}
		for _, item := range result.items {
			if strings.TrimSpace(item.Source) == "" {
				item.Source = result.source
			}
			usages = append(usages, item)
		}
	}

	sort.SliceStable(usages, func(i, j int) bool {
		if usages[i].Source != usages[j].Source {
			return usages[i].Source < usages[j].Source
		}
		if usages[i].DisplayNumber != usages[j].DisplayNumber {
			return usages[i].DisplayNumber < usages[j].DisplayNumber
		}
		if usages[i].ObjN != usages[j].ObjN {
			return usages[i].ObjN < usages[j].ObjN
		}
		return usages[i].Slot < usages[j].Slot
	})
	return usages, errors.Join(errs...)
}

// FindObjectsBySIMPhone шукає SIM безпосередньо в таблицях Phoenix.
func (p *PhoenixDataProvider) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	if p == nil || p.db == nil {
		return nil, errors.New("Phoenix provider is not configured")
	}
	normalized := normalizeUASimPhone(phone)
	if len(normalized) != 10 || !strings.HasPrefix(normalized, "0") {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()
	var rows []phoenixPanelSIMRow
	if err := p.db.SelectContext(ctx, &rows, phoenixObjectSIMListQuery); err != nil {
		return nil, fmt.Errorf("failed to lookup Phoenix SIM phone usage: %w", err)
	}
	p.objectMu.RLock()
	namesByPanel := make(map[string]string, len(p.cachedObjects))
	for _, object := range p.cachedObjects {
		namesByPanel[strings.TrimSpace(object.DisplayNumber)] = strings.TrimSpace(object.Name)
	}
	p.objectMu.RUnlock()

	usages := make([]contracts.AdminSIMPhoneUsage, 0, 4)
	for _, row := range rows {
		panelID := strings.TrimSpace(row.PanelID)
		if panelID == "" {
			continue
		}
		objectID := int64(p.registerPanelID(panelID))
		if excludeObjN != nil && objectID == *excludeObjN {
			continue
		}
		if normalizeUASimPhone(nullString(row.Sim1Number)) == normalized {
			usages = append(usages, contracts.AdminSIMPhoneUsage{ObjN: objectID, DisplayNumber: panelID, Name: namesByPanel[panelID], Slot: "SIM 1"})
		}
		if normalizeUASimPhone(nullString(row.Sim2Number)) == normalized {
			usages = append(usages, contracts.AdminSIMPhoneUsage{ObjN: objectID, DisplayNumber: panelID, Name: namesByPanel[panelID], Slot: "SIM 2"})
		}
	}
	return usages, nil
}

// FindObjectsBySIMPhone шукає SIM у поточному наборі об'єктів і приладів CASL Cloud.
func (p *CASLCloudProvider) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	if p == nil {
		return nil, errors.New("CASL provider is not configured")
	}
	normalized := normalizeUASimPhone(phone)
	if len(normalized) != 10 || !strings.HasPrefix(normalized, "0") {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), simPhoneLookupTimeout)
	defer cancel()
	records, err := p.loadObjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load CASL objects for SIM lookup: %w", err)
	}
	_, err = p.loadDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load CASL devices for SIM lookup: %w", err)
	}

	objects := make([]models.Object, 0, len(records))
	for _, record := range records {
		device, ok := p.resolveDeviceForObject(record)
		if !ok {
			objects = append(objects, mapCASLGrdObjectToObject(record, nil))
			continue
		}
		objects = append(objects, mapCASLGrdObjectToObject(record, &device))
	}
	return findSIMPhoneUsagesInObjects(objects, normalized, excludeObjN), nil
}

func findSIMPhoneUsagesInObjects(objects []models.Object, normalizedPhone string, excludeObjN *int64) []contracts.AdminSIMPhoneUsage {
	usages := make([]contracts.AdminSIMPhoneUsage, 0, 4)
	for _, object := range objects {
		objectID := int64(object.ID)
		if excludeObjN != nil && objectID == *excludeObjN {
			continue
		}
		appendUsage := func(slot string) {
			usages = append(usages, contracts.AdminSIMPhoneUsage{
				ObjN:          objectID,
				DisplayNumber: strings.TrimSpace(object.DisplayNumber),
				Name:          strings.TrimSpace(object.Name),
				Slot:          slot,
			})
		}
		if normalizeUASimPhone(object.SIM1) == normalizedPhone {
			appendUsage("SIM 1")
		}
		if normalizeUASimPhone(object.SIM2) == normalizedPhone {
			appendUsage("SIM 2")
		}
	}
	return usages
}

func simSourceDisplayName(name string) string {
	source := frontendSourceFromProviderName(name)
	if source != contracts.FrontendSourceUnknown {
		return source.DisplayName()
	}
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return contracts.FrontendSourceUnknown.DisplayName()
}
