package viewmodels

import (
	"fmt"
	"sort"
	"strings"

	"obj_catalog_fyne_v3/pkg/models"
)

type WorkAreaGroupSection struct {
	Group    WorkAreaGroupSectionGroup
	Zones    []models.Zone
	Contacts []models.Contact
}

type WorkAreaGroupSectionGroup struct {
	ID        string
	Number    int
	Name      string
	StateText string
}

type WorkAreaGroupSectionsViewModel struct{}

func NewWorkAreaGroupSectionsViewModel() *WorkAreaGroupSectionsViewModel {
	return &WorkAreaGroupSectionsViewModel{}
}

func (vm *WorkAreaGroupSectionsViewModel) BuildZoneSections(
	object *models.Object,
	zones []models.Zone,
) []WorkAreaGroupSection {
	sections := vm.buildSections(object, zones, nil)
	for i := range sections {
		sort.SliceStable(sections[i].Zones, func(left, right int) bool {
			return sections[i].Zones[left].Number < sections[i].Zones[right].Number
		})
	}
	return sections
}

func (vm *WorkAreaGroupSectionsViewModel) BuildContactSections(
	object *models.Object,
	contacts []models.Contact,
) []WorkAreaGroupSection {
	sections := vm.buildSections(object, nil, contacts)
	for i := range sections {
		sort.SliceStable(sections[i].Contacts, func(left, right int) bool {
			if sections[i].Contacts[left].Priority == sections[i].Contacts[right].Priority {
				return sections[i].Contacts[left].Name < sections[i].Contacts[right].Name
			}
			return sections[i].Contacts[left].Priority < sections[i].Contacts[right].Priority
		})
	}
	return sections
}

func (vm *WorkAreaGroupSectionsViewModel) ShouldUseGroupedZones(
	object *models.Object,
	zones []models.Zone,
) bool {
	return vm.shouldUseGroupedLayout(object, zones, nil)
}

func (vm *WorkAreaGroupSectionsViewModel) ShouldUseGroupedContacts(
	object *models.Object,
	contacts []models.Contact,
) bool {
	return vm.shouldUseGroupedLayout(object, nil, contacts)
}

func (vm *WorkAreaGroupSectionsViewModel) FormatSectionTitle(group WorkAreaGroupSectionGroup) string {
	titleParts := []string{}

	groupLabel := "Група"
	if group.Number > 0 {
		groupLabel = fmt.Sprintf("Група %d", group.Number)
	}
	titleParts = append(titleParts, groupLabel)

	if name := strings.TrimSpace(group.Name); name != "" {
		titleParts = append(titleParts, name)
	}
	if state := strings.TrimSpace(group.StateText); state != "" {
		titleParts = append(titleParts, state)
	}

	return strings.Join(titleParts, " | ")
}

func (vm *WorkAreaGroupSectionsViewModel) buildSections(
	object *models.Object,
	zones []models.Zone,
	contacts []models.Contact,
) []WorkAreaGroupSection {
	knownGroups := normalizeObjectGroups(object)
	if len(knownGroups) == 0 {
		knownGroups = append(knownGroups, groupsFromZones(zones)...)
		knownGroups = appendMissingGroups(knownGroups, groupsFromContacts(contacts))
	}

	if len(knownGroups) <= 1 {
		return nil
	}

	sectionIndexByID := make(map[string]int, len(knownGroups))
	ordered := make([]WorkAreaGroupSection, 0, len(knownGroups))
	for _, group := range knownGroups {
		groupID := normalizeGroupID(group.ID, group.Number)
		section := WorkAreaGroupSection{
			Group: WorkAreaGroupSectionGroup{
				ID:        groupID,
				Number:    group.Number,
				Name:      displayObjectGroupName(group),
				StateText: displayObjectGroupState(group),
			},
			Zones:    []models.Zone{},
			Contacts: []models.Contact{},
		}
		ordered = append(ordered, section)
		sectionIndexByID[groupID] = len(ordered) - 1
	}

	for _, zone := range zones {
		groupID := normalizeGroupID(zone.GroupID, zone.GroupNumber)
		sectionIndex, ok := sectionIndexByID[groupID]
		if !ok {
			continue
		}
		ordered[sectionIndex].Zones = append(ordered[sectionIndex].Zones, zone)
	}

	for _, contact := range contacts {
		groupID := normalizeGroupID(contact.GroupID, contact.GroupNumber)
		sectionIndex, ok := sectionIndexByID[groupID]
		if !ok {
			continue
		}
		ordered[sectionIndex].Contacts = append(ordered[sectionIndex].Contacts, contact)
	}

	sort.SliceStable(ordered, func(left, right int) bool {
		if ordered[left].Group.Number == ordered[right].Group.Number {
			return ordered[left].Group.Name < ordered[right].Group.Name
		}
		return ordered[left].Group.Number < ordered[right].Group.Number
	})

	return ordered
}

func (vm *WorkAreaGroupSectionsViewModel) shouldUseGroupedLayout(
	object *models.Object,
	zones []models.Zone,
	contacts []models.Contact,
) bool {
	if object != nil && len(object.Groups) > 1 {
		return true
	}
	if len(distinctZoneGroupKeys(zones)) > 1 {
		return true
	}
	if len(distinctContactGroupKeys(contacts)) > 1 {
		return true
	}
	return false
}

func normalizeObjectGroups(object *models.Object) []models.ObjectGroup {
	if object == nil || len(object.Groups) == 0 {
		return nil
	}
	groups := make([]models.ObjectGroup, 0, len(object.Groups))
	for _, group := range object.Groups {
		group.ID = normalizeGroupID(group.ID, group.Number)
		if group.Name == "" {
			group.Name = displayObjectGroupName(group)
		}
		groups = append(groups, group)
	}
	return groups
}

func groupsFromZones(zones []models.Zone) []models.ObjectGroup {
	groups := make([]models.ObjectGroup, 0, len(zones))
	seen := map[string]struct{}{}
	for _, zone := range zones {
		groupID := normalizeGroupID(zone.GroupID, zone.GroupNumber)
		if groupID == "" {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		groups = append(groups, models.ObjectGroup{
			ID:        groupID,
			Number:    zone.GroupNumber,
			Name:      strings.TrimSpace(zone.GroupName),
			StateText: strings.TrimSpace(zone.GroupStateText),
		})
	}
	return groups
}

func groupsFromContacts(contacts []models.Contact) []models.ObjectGroup {
	groups := make([]models.ObjectGroup, 0, len(contacts))
	seen := map[string]struct{}{}
	for _, contact := range contacts {
		groupID := normalizeGroupID(contact.GroupID, contact.GroupNumber)
		if groupID == "" {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		groups = append(groups, models.ObjectGroup{
			ID:        groupID,
			Number:    contact.GroupNumber,
			Name:      strings.TrimSpace(contact.GroupName),
			StateText: strings.TrimSpace(contact.GroupStateText),
		})
	}
	return groups
}

func appendMissingGroups(
	base []models.ObjectGroup,
	candidates []models.ObjectGroup,
) []models.ObjectGroup {
	seen := make(map[string]struct{}, len(base))
	for _, group := range base {
		seen[normalizeGroupID(group.ID, group.Number)] = struct{}{}
	}
	for _, group := range candidates {
		groupID := normalizeGroupID(group.ID, group.Number)
		if _, ok := seen[groupID]; ok {
			continue
		}
		base = append(base, group)
		seen[groupID] = struct{}{}
	}
	return base
}

func distinctZoneGroupKeys(zones []models.Zone) map[string]struct{} {
	distinct := map[string]struct{}{}
	for _, zone := range zones {
		groupID := normalizeGroupID(zone.GroupID, zone.GroupNumber)
		if groupID == "" {
			continue
		}
		distinct[groupID] = struct{}{}
	}
	return distinct
}

func distinctContactGroupKeys(contacts []models.Contact) map[string]struct{} {
	distinct := map[string]struct{}{}
	for _, contact := range contacts {
		groupID := normalizeGroupID(contact.GroupID, contact.GroupNumber)
		if groupID == "" {
			continue
		}
		distinct[groupID] = struct{}{}
	}
	return distinct
}

func normalizeGroupID(raw string, number int) string {
	if value := strings.TrimSpace(raw); value != "" {
		return value
	}
	if number > 0 {
		return fmt.Sprintf("group:%d", number)
	}
	return ""
}

func displayObjectGroupName(group models.ObjectGroup) string {
	if name := strings.TrimSpace(group.Name); name != "" {
		return name
	}
	if name := strings.TrimSpace(group.RoomName); name != "" {
		return name
	}
	if name := strings.TrimSpace(group.PremiseName); name != "" {
		return name
	}
	return ""
}

func displayObjectGroupState(group models.ObjectGroup) string {
	if state := strings.TrimSpace(group.StateText); state != "" {
		return state
	}
	if group.Armed {
		return "ПІД ОХОРОНОЮ"
	}
	return ""
}
