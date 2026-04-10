package viewmodels

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
)

const (
	FilterAll           = "Всі"
	FilterAlarm         = "Є тривоги"
	FilterOffline       = "Нема зв'язку"
	FilterMonitoringOff = "Знято зі спостереження"
	FilterDebug         = "В режимі налагодження"
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
	CountMonitoringOff    int
	CountDebug            int
	CountBridge           int
	CountPhoenix          int
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
	return slices.Clone(objects)
}

func NormalizeObjectListFilter(selected string) string {
	clean := utils.StripCountSuffix(selected)
	switch clean {
	case FilterAlarm, FilterOffline, FilterMonitoringOff, FilterDebug:
		return clean
	default:
		return FilterAll
	}
}

func (vm *ObjectListViewModel) BuildFilterOptions(countAll int, countAlarm int, countOffline int, countMonitoringOff int, countDebug int) []string {
	return []string{
		fmt.Sprintf("%s (%d)", FilterAll, countAll),
		fmt.Sprintf("%s (%d)", FilterAlarm, countAlarm),
		fmt.Sprintf("%s (%d)", FilterOffline, countOffline),
		fmt.Sprintf("%s (%d)", FilterMonitoringOff, countMonitoringOff),
		fmt.Sprintf("%s (%d)", FilterDebug, countDebug),
	}
}

func (vm *ObjectListViewModel) ApplyFilters(input ObjectListFilterInput) ObjectListFilterOutput {
	query := strings.ToLower(strings.TrimSpace(input.Query))
	currentFilter := NormalizeObjectListFilter(input.CurrentFilter)
	currentSource := NormalizeObjectSourceFilter(input.CurrentSource)

	filtered := make([]models.Object, 0, len(input.AllObjects))
	countAll := 0
	countAlarm := 0
	countOffline := 0
	countMonitoringOff := 0
	countDebug := 0
	countBridge := 0
	countPhoenix := 0
	countCASL := 0

	terms := strings.Fields(query)

	for _, obj := range input.AllObjects {
		source := ObjectSourceByID(obj.ID)
		if !matchesSearchTerms(obj, source, terms) {
			continue
		}

		countAll++
		switch source {
		case ObjectSourceCASL:
			countCASL++
		case ObjectSourcePhoenix:
			countPhoenix++
		default:
			countBridge++
		}
		if obj.Status == models.StatusFire || obj.Status == models.StatusFault {
			countAlarm++
		}
		if obj.IsConnState == 0 && obj.GuardState != 0 {
			countOffline++
		}
		if isMonitoringOffObject(obj, source) {
			countMonitoringOff++
		}
		if isDebugObject(obj, source) {
			countDebug++
		}

		statusMatch := true
		switch currentFilter {
		case FilterAlarm:
			if obj.Status != models.StatusFire && obj.Status != models.StatusFault {
				statusMatch = false
			}
		case FilterOffline:
			if !(obj.IsConnState == 0 && obj.GuardState != 0) {
				statusMatch = false
			}
		case FilterMonitoringOff:
			if !isMonitoringOffObject(obj, source) {
				statusMatch = false
			}
		case FilterDebug:
			if !isDebugObject(obj, source) {
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
		if newSelectedRow == -1 && len(filtered) > 0 {
			newSelectedRow = 0
		}
	}
	out := ObjectListFilterOutput{
		Filtered:           filtered,
		CountAll:           countAll,
		CountAlarm:         countAlarm,
		CountOffline:       countOffline,
		CountMonitoringOff: countMonitoringOff,
		CountDebug:         countDebug,
		CountBridge:        countBridge,
		CountPhoenix:       countPhoenix,
		CountCASL:          countCASL,
		NewSelectedRow:     newSelectedRow,
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

func isMonitoringOffObject(obj models.Object, source string) bool {
	switch NormalizeObjectSourceFilter(source) {
	case ObjectSourcePhoenix, ObjectSourceCASL:
		return obj.BlockedArmedOnOff == 1
	default:
		return obj.GuardState == 0
	}
}

func isDebugObject(obj models.Object, source string) bool {
	switch NormalizeObjectSourceFilter(source) {
	case ObjectSourceCASL:
		return false
	case ObjectSourcePhoenix:
		return obj.BlockedArmedOnOff == 2
	default:
		return obj.BlockedArmedOnOff == 2
	}
}

func matchesSearchTerms(obj models.Object, source string, terms []string) bool {
	if len(terms) == 0 {
		return true
	}

	idText := strconv.Itoa(obj.ID)
	displayNumberText := strings.ToLower(strings.TrimSpace(ObjectDisplayNumber(obj)))
	nameText := strings.ToLower(strings.TrimSpace(obj.Name))
	addressText := strings.ToLower(strings.TrimSpace(obj.Address))
	contractText := strings.ToLower(strings.TrimSpace(obj.ContractNum))
	phoneText := strings.ToLower(strings.TrimSpace(obj.Phone))
	sim1Text := strings.ToLower(strings.TrimSpace(obj.SIM1))
	sim2Text := strings.ToLower(strings.TrimSpace(obj.SIM2))
	sourceText := strings.ToLower(strings.TrimSpace(source))

	sim1Digits := utils.DigitsOnly(sim1Text)
	sim2Digits := utils.DigitsOnly(sim2Text)
	phoneDigits := utils.DigitsOnly(phoneText)

	for _, term := range terms {
		switch {
		case strings.HasPrefix(term, "src:"):
			required := NormalizeObjectSourceFilter(strings.TrimSpace(strings.TrimPrefix(term, "src:")))
			if !sourceMatchesFilter(source, required) {
				return false
			}
		case strings.HasPrefix(term, "sim:"):
			simQuery := utils.DigitsOnly(strings.TrimSpace(strings.TrimPrefix(term, "sim:")))
			if simQuery == "" {
				return false
			}
			if !strings.Contains(sim1Digits, simQuery) && !strings.Contains(sim2Digits, simQuery) {
				return false
			}
		default:
			digitsTerm := utils.DigitsOnly(term)
			isNumericTerm := digitsTerm != "" && utils.IsDigitsOnlyTerm(term)
			if isNumericTerm && len(digitsTerm) >= 4 {
				if strings.Contains(sim1Digits, digitsTerm) || strings.Contains(sim2Digits, digitsTerm) || strings.Contains(phoneDigits, digitsTerm) || strings.Contains(idText, digitsTerm) {
					continue
				}
				if strings.Contains(displayNumberText, digitsTerm) {
					continue
				}
			}
			if strings.Contains(idText, term) ||
				strings.Contains(displayNumberText, term) ||
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

// GetRowColors визначає кольори тексту та фону для рядка списку об'єктів.
// Логіка перенесена з View (MVVM).
func (vm *ObjectListViewModel) GetRowColors(item models.Object, isDark bool) (textColor, rowColor color.NRGBA) {
	selectObjectColor := utils.SelectObjectColorNRGBA
	if isDark {
		selectObjectColor = utils.SelectObjectColorNRGBADark
	}

	// Спеціальні випадки для Phoenix
	if ids.IsPhoenixObjectID(item.ID) &&
		item.BlockedArmedOnOff == 1 &&
		item.AlarmState == 0 &&
		item.TechAlarmState == 0 &&
		item.Status == models.StatusNormal {
		if isDark {
			return color.NRGBA{R: 232, G: 239, B: 246, A: 255}, color.NRGBA{R: 54, G: 74, B: 92, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 79, G: 109, B: 135, A: 255}
	}

	if ids.IsPhoenixObjectID(item.ID) &&
		item.BlockedArmedOnOff == 0 &&
		item.GuardState == 0 &&
		item.AlarmState == 0 &&
		item.TechAlarmState == 0 &&
		item.Status == models.StatusNormal {
		if isDark {
			return color.NRGBA{R: 225, G: 244, B: 255, A: 255}, color.NRGBA{R: 37, G: 96, B: 128, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 67, G: 156, B: 199, A: 255}
	}

	// Пріоритети кольорів (зверху вниз):
	// 1) блокування, 2) тривога, 3) технічна/пожежна несправність,
	// 4) втрата зв'язку, 5) проблема приписки/конфігурації, 6) інші стани.
	if item.BlockedArmedOnOff == 1 {
		// Тимчасово знято із спостереження.
		if isDark {
			return color.NRGBA{R: 230, G: 220, B: 245, A: 255}, color.NRGBA{R: 98, G: 52, B: 125, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 144, G: 64, B: 196, A: 255}
	}
	if item.BlockedArmedOnOff == 2 {
		// Режим налагодження.
		if isDark {
			return color.NRGBA{R: 238, G: 236, B: 195, A: 255}, color.NRGBA{R: 95, G: 96, B: 42, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 128, G: 128, B: 0, A: 255}
	}

	if item.AlarmState > 0 || item.Status == models.StatusFire {
		return selectObjectColor(1)
	}

	if item.TechAlarmState > 0 || item.Status == models.StatusFault {
		return selectObjectColor(2)
	}

	if item.IsConnState == 0 || item.Status == models.StatusOffline {
		if isDark {
			return color.NRGBA{R: 255, G: 250, B: 180, A: 255}, color.NRGBA{R: 90, G: 90, B: 20, A: 255}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}, color.NRGBA{R: 225, G: 235, B: 35, A: 255}
	}

	if ids.IsCASLObjectID(item.ID) && !item.HasAssignment {
		if isDark {
			return color.NRGBA{R: 240, G: 243, B: 255, A: 255}, color.NRGBA{R: 52, G: 70, B: 98, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 77, G: 112, B: 168, A: 255}
	}

	if !ids.IsCASLObjectID(item.ID) && !ids.IsPhoenixObjectID(item.ID) &&
		strings.TrimSpace(item.SubServerA) == "" && strings.TrimSpace(item.SubServerB) == "" {
		// Для МІСТ/БД підсервери мають бути заповнені.
		return color.NRGBA{R: 210, G: 0, B: 0, A: 255}, color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}

	return utils.ChangeItemColorNRGBA(item.AlarmState, item.GuardState, item.TechAlarmState, item.IsConnState, isDark)
}
