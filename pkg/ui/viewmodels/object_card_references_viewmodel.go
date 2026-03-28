package viewmodels

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

const emptyOptionLabel = "—"

// ObjectCardReferenceProvider описує мінімальний контракт джерела довідників для форми об'єкта.
type ObjectCardReferenceProvider interface {
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	ListObjectDistricts() ([]contracts.DictionaryItem, error)
	ListPPKConstructor() ([]contracts.PPKConstructorItem, error)
	ListSubServers() ([]contracts.AdminSubServer, error)
}

// ObjectCardReferencesViewModel зберігає та форматує довідники для форми картки об'єкта.
type ObjectCardReferencesViewModel struct {
	objectTypeOptions []string
	regionOptions     []string
	ppkOptions        []string
	subServerOptions  []string

	objectTypeIDs  map[string]int64
	regionIDs      map[string]int64
	ppkIDs         map[string]int64
	subServerBinds map[string]string

	allPPKItems []contracts.PPKConstructorItem
}

func NewObjectCardReferencesViewModel() *ObjectCardReferencesViewModel {
	return &ObjectCardReferencesViewModel{
		objectTypeIDs:  map[string]int64{},
		regionIDs:      map[string]int64{},
		ppkIDs:         map[string]int64{},
		subServerBinds: map[string]string{},
		allPPKItems:    make([]contracts.PPKConstructorItem, 0),
	}
}

func (vm *ObjectCardReferencesViewModel) Load(
	typeItems []contracts.DictionaryItem,
	regionItems []contracts.DictionaryItem,
	ppkItems []contracts.PPKConstructorItem,
	subServerItems []contracts.AdminSubServer,
) {
	vm.objectTypeIDs = map[string]int64{}
	vm.objectTypeOptions = make([]string, 0, len(typeItems))
	for _, item := range typeItems {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprintf("Тип %d", item.ID)
		}
		label = fmt.Sprintf("%s [%d]", label, item.ID)
		vm.objectTypeOptions = append(vm.objectTypeOptions, label)
		vm.objectTypeIDs[label] = item.ID
	}

	vm.regionIDs = map[string]int64{emptyOptionLabel: 0}
	vm.regionOptions = []string{emptyOptionLabel}
	for _, item := range regionItems {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprintf("Район %d", item.ID)
		}
		label = fmt.Sprintf("%s [%d]", label, item.ID)
		vm.regionOptions = append(vm.regionOptions, label)
		vm.regionIDs[label] = item.ID
	}

	vm.allPPKItems = append(vm.allPPKItems[:0], ppkItems...)
	vm.RefreshPPKOptions(0)

	vm.subServerBinds = map[string]string{emptyOptionLabel: ""}
	vm.subServerOptions = []string{emptyOptionLabel}
	for _, item := range subServerItems {
		bind := strings.TrimSpace(item.Bind)
		if bind == "" {
			continue
		}
		name := strings.TrimSpace(item.Info)
		if name == "" {
			name = strings.TrimSpace(item.Host)
		}
		if name == "" {
			name = fmt.Sprintf("Підсервер %d", item.ID)
		}
		label := fmt.Sprintf("%s (%s) [%s]", name, subServerTypeLabel(item.Type), bind)
		vm.subServerOptions = append(vm.subServerOptions, label)
		vm.subServerBinds[label] = bind
	}
}

func (vm *ObjectCardReferencesViewModel) LoadFromProvider(provider ObjectCardReferenceProvider) error {
	typeItems, err := provider.ListObjectTypes()
	if err != nil {
		return fmt.Errorf("не вдалося завантажити типи об'єктів: %w", err)
	}
	regionItems, err := provider.ListObjectDistricts()
	if err != nil {
		return fmt.Errorf("не вдалося завантажити райони: %w", err)
	}
	ppkItems, err := provider.ListPPKConstructor()
	if err != nil {
		return fmt.Errorf("не вдалося завантажити довідник ППК: %w", err)
	}
	subServerItems, err := provider.ListSubServers()
	if err != nil {
		return fmt.Errorf("не вдалося завантажити довідник підсерверів: %w", err)
	}

	vm.Load(typeItems, regionItems, ppkItems, subServerItems)
	return nil
}

func subServerTypeLabel(subServerType int64) string {
	switch subServerType {
	case 2:
		return "GPRS"
	case 4:
		return "AVD"
	default:
		if subServerType > 0 {
			return fmt.Sprintf("%d", subServerType)
		}
		return emptyOptionLabel
	}
}

func (vm *ObjectCardReferencesViewModel) RefreshPPKOptions(preferredID int64) string {
	vm.ppkIDs = map[string]int64{emptyOptionLabel: 0}
	vm.ppkOptions = []string{emptyOptionLabel}
	for _, item := range vm.allPPKItems {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprintf("ППК %d", item.ID)
		}
		label = fmt.Sprintf("%s [%d]", label, item.ID)
		vm.ppkOptions = append(vm.ppkOptions, label)
		vm.ppkIDs[label] = item.ID
	}

	selected := emptyOptionLabel
	preferredIDs := make([]int64, 0, 3)
	if preferredID > 0 {
		preferredIDs = append(preferredIDs, preferredID)
		if preferredID > 100 {
			preferredIDs = append(preferredIDs, preferredID-100)
		}
		if preferredID < 100 {
			preferredIDs = append(preferredIDs, preferredID+100)
		}
	}
	for _, wantedID := range preferredIDs {
		for _, option := range vm.ppkOptions {
			if vm.ppkIDs[option] == wantedID {
				selected = option
				return selected
			}
		}
	}
	return selected
}

func (vm *ObjectCardReferencesViewModel) ObjectTypeOptions() []string {
	return append([]string(nil), vm.objectTypeOptions...)
}

func (vm *ObjectCardReferencesViewModel) RegionOptions() []string {
	return append([]string(nil), vm.regionOptions...)
}

func (vm *ObjectCardReferencesViewModel) PPKOptions() []string {
	return append([]string(nil), vm.ppkOptions...)
}

func (vm *ObjectCardReferencesViewModel) SubServerOptions() []string {
	return append([]string(nil), vm.subServerOptions...)
}

func (vm *ObjectCardReferencesViewModel) ObjectTypeID(label string) int64 {
	return vm.objectTypeIDs[label]
}

func (vm *ObjectCardReferencesViewModel) RegionID(label string) int64 {
	return vm.regionIDs[label]
}

func (vm *ObjectCardReferencesViewModel) PPKID(label string) int64 {
	return vm.ppkIDs[label]
}

func (vm *ObjectCardReferencesViewModel) SubServerBind(label string) string {
	return strings.TrimSpace(vm.subServerBinds[label])
}

func (vm *ObjectCardReferencesViewModel) ObjectTypeLabelByID(id int64) string {
	for _, option := range vm.objectTypeOptions {
		if vm.objectTypeIDs[option] == id {
			return option
		}
	}
	if len(vm.objectTypeOptions) > 0 {
		return vm.objectTypeOptions[0]
	}
	return ""
}

func (vm *ObjectCardReferencesViewModel) RegionLabelByID(id int64) string {
	for _, option := range vm.regionOptions {
		if vm.regionIDs[option] == id {
			return option
		}
	}
	if len(vm.regionOptions) > 0 {
		return vm.regionOptions[0]
	}
	return ""
}

func (vm *ObjectCardReferencesViewModel) RegionLabelByIDExact(id int64) (string, bool) {
	for _, option := range vm.regionOptions {
		if vm.regionIDs[option] == id {
			return option, true
		}
	}
	return "", false
}

func (vm *ObjectCardReferencesViewModel) SubServerLabelByBind(bind string) string {
	bind = strings.TrimSpace(bind)
	if bind == "" {
		return emptyOptionLabel
	}
	for _, option := range vm.subServerOptions {
		if vm.subServerBinds[option] == bind {
			return option
		}
	}
	return emptyOptionLabel
}
