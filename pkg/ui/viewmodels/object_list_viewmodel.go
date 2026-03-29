package viewmodels

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

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
	CurrentSource        string
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
	CountBridge           int
	CountCASL             int
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
	currentSource := NormalizeObjectSourceFilter(input.CurrentSource)

	filtered := make([]models.Object, 0, len(input.AllObjects))
	countAll := 0
	countAlarm := 0
	countOffline := 0
	countDisarmed := 0
	countBridge := 0
	countCASL := 0

	terms := splitSearchTerms(query)

	for _, obj := range input.AllObjects {
		source := ObjectSourceByID(obj.ID)
		if !matchesSearchTerms(obj, source, terms) {
			continue
		}

		countAll++
		if source == ObjectSourceCASL {
			countCASL++
		} else {
			countBridge++
		}
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
		if !statusMatch {
			continue
		}
		if !sourceMatchesFilter(source, currentSource) {
			continue
		}
		filtered = append(filtered, obj)
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
		CountBridge:    countBridge,
		CountCASL:      countCASL,
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

func splitSearchTerms(query string) []string {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func matchesSearchTerms(obj models.Object, source string, terms []string) bool {
	if len(terms) == 0 {
		return true
	}

	idText := strconv.Itoa(obj.ID)
	nameText := strings.ToLower(strings.TrimSpace(obj.Name))
	addressText := strings.ToLower(strings.TrimSpace(obj.Address))
	contractText := strings.ToLower(strings.TrimSpace(obj.ContractNum))
	phoneText := strings.ToLower(strings.TrimSpace(obj.Phone))
	sim1Text := strings.ToLower(strings.TrimSpace(obj.SIM1))
	sim2Text := strings.ToLower(strings.TrimSpace(obj.SIM2))
	sourceText := strings.ToLower(strings.TrimSpace(source))

	sim1Digits := digitsOnly(sim1Text)
	sim2Digits := digitsOnly(sim2Text)
	phoneDigits := digitsOnly(phoneText)

	for _, term := range terms {
		switch {
		case strings.HasPrefix(term, "src:"):
			required := NormalizeObjectSourceFilter(strings.TrimSpace(strings.TrimPrefix(term, "src:")))
			if !sourceMatchesFilter(source, required) {
				return false
			}
		case strings.HasPrefix(term, "sim:"):
			simQuery := digitsOnly(strings.TrimSpace(strings.TrimPrefix(term, "sim:")))
			if simQuery == "" {
				return false
			}
			if !strings.Contains(sim1Digits, simQuery) && !strings.Contains(sim2Digits, simQuery) {
				return false
			}
		default:
			digitsTerm := digitsOnly(term)
			isNumericTerm := digitsTerm != "" && isDigitsOnlyTerm(term)
			if isNumericTerm && len(digitsTerm) >= 4 {
				if strings.Contains(sim1Digits, digitsTerm) || strings.Contains(sim2Digits, digitsTerm) || strings.Contains(phoneDigits, digitsTerm) || strings.Contains(idText, digitsTerm) {
					continue
				}
			}
			if strings.Contains(idText, term) ||
				strings.Contains(nameText, term) ||
				strings.Contains(addressText, term) ||
				strings.Contains(contractText, term) ||
				strings.Contains(phoneText, term) ||
				strings.Contains(sim1Text, term) ||
				strings.Contains(sim2Text, term) ||
				strings.Contains(sourceText, term) {
				continue
			}
			return false
		}
	}
	return true
}

func digitsOnly(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isDigitsOnlyTerm(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
