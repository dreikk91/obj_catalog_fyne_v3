package viewmodels

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/models"
)

// ObjectListUseCase описує мінімальний use case для завантаження об'єктів у список.
type ObjectListUseCase interface {
	FetchObjects() []models.Object
}

// ObjectListFilterInput описує вхідні дані фільтрації списку об'єктів.
type ObjectListFilterInput struct {
	AllObjects           []models.Object
	Query                string
	CurrentFilter        string
	PreviousSelectedID   int
	HadPreviousSelection bool
	LastNotifiedID       int
	HasNotifiedSelection bool
}

// ObjectListFilterOutput описує результат фільтрації та вибору.
type ObjectListFilterOutput struct {
	Filtered              []models.Object
	CountAll              int
	CountAlarm            int
	CountOffline          int
	CountDisarmed         int
	NewSelectedRow        int
	SelectedObject        models.Object
	HasSelectedObject     bool
	ShouldNotifySelection bool
}

// ObjectListViewModel інкапсулює бізнес-правила фільтрації/вибору списку об'єктів.
type ObjectListViewModel struct{}

func NewObjectListViewModel() *ObjectListViewModel {
	return &ObjectListViewModel{}
}

func (vm *ObjectListViewModel) LoadObjects(useCase ObjectListUseCase) []models.Object {
	if useCase == nil {
		return nil
	}
	objects := useCase.FetchObjects()
	return append([]models.Object(nil), objects...)
}

func (vm *ObjectListViewModel) NormalizeFilter(selected string) string {
	clean := strings.TrimSpace(selected)
	if idx := strings.Index(clean, " ("); idx != -1 {
		clean = clean[:idx]
	}
	return clean
}

func (vm *ObjectListViewModel) BuildFilterOptions(countAll int, countAlarm int, countOffline int, countDisarmed int) []string {
	return []string{
		fmt.Sprintf("Всі (%d)", countAll),
		fmt.Sprintf("Є тривоги (%d)", countAlarm),
		fmt.Sprintf("Нема зв'язку (%d)", countOffline),
		fmt.Sprintf("Знято з охорони (%d)", countDisarmed),
	}
}

func (vm *ObjectListViewModel) ApplyFilters(input ObjectListFilterInput) ObjectListFilterOutput {
	query := strings.ToLower(strings.TrimSpace(input.Query))
	currentFilter := strings.TrimSpace(input.CurrentFilter)

	filtered := make([]models.Object, 0, len(input.AllObjects))
	countAll := 0
	countAlarm := 0
	countOffline := 0
	countDisarmed := 0

	for _, obj := range input.AllObjects {
		matchSearch := true
		if query != "" {
			idText := strings.ToLower(fmt.Sprintf("%d", obj.ID))
			matchSearch = strings.Contains(idText, query) ||
				strings.Contains(strings.ToLower(obj.Name), query) ||
				strings.Contains(strings.ToLower(obj.Address), query) ||
				strings.Contains(strings.ToLower(obj.ContractNum), query) ||
				strings.Contains(strings.ToLower(obj.SIM1), query) ||
				strings.Contains(strings.ToLower(obj.SIM2), query) ||
				strings.Contains(strings.ToLower(obj.Phone), query)
		}
		if !matchSearch {
			continue
		}

		countAll++
		if obj.Status == models.StatusFire || obj.Status == models.StatusFault {
			countAlarm++
		}
		if obj.IsConnState == 0 && obj.GuardState != 0 {
			countOffline++
		}
		if obj.GuardState == 0 {
			countDisarmed++
		}

		statusMatch := true
		switch currentFilter {
		case "Є тривоги":
			if obj.Status != models.StatusFire && obj.Status != models.StatusFault {
				statusMatch = false
			}
		case "Нема зв'язку":
			if !(obj.IsConnState == 0 && obj.GuardState != 0) {
				statusMatch = false
			}
		case "Знято з охорони":
			if obj.GuardState != 0 {
				statusMatch = false
			}
		}
		if statusMatch {
			filtered = append(filtered, obj)
		}
	}

	newSelectedRow := -1
	if input.HadPreviousSelection {
		for i := range filtered {
			if filtered[i].ID == input.PreviousSelectedID {
				newSelectedRow = i
				break
			}
		}
	}
	if newSelectedRow == -1 && len(filtered) > 0 {
		newSelectedRow = 0
	}

	out := ObjectListFilterOutput{
		Filtered:       filtered,
		CountAll:       countAll,
		CountAlarm:     countAlarm,
		CountOffline:   countOffline,
		CountDisarmed:  countDisarmed,
		NewSelectedRow: newSelectedRow,
	}
	if newSelectedRow >= 0 {
		out.SelectedObject = filtered[newSelectedRow]
		out.HasSelectedObject = true
		if !input.HasNotifiedSelection || out.SelectedObject.ID != input.LastNotifiedID ||
			(!input.HadPreviousSelection || out.SelectedObject.ID != input.PreviousSelectedID) {
			out.ShouldNotifySelection = true
		}
	}

	return out
}
