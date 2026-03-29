package viewmodels

import (
	"fmt"
	"strings"
)

// ObjectWizardReviewPersonalItem - рядок рев'ю для відповідальної особи.
type ObjectWizardReviewPersonalItem struct {
	Number   int64
	FullName string
	Phones   string
	IsAdmin  bool
}

// ObjectWizardReviewZoneItem - рядок рев'ю для зони.
type ObjectWizardReviewZoneItem struct {
	Number      int64
	Description string
}

// ObjectWizardReviewInput містить зріз поточного стану майстра для побудови підсумку.
type ObjectWizardReviewInput struct {
	ObjN       string
	ShortName  string
	FullName   string
	ObjectType string
	Region     string
	HiddenN    string
	Address    string
	Phones     string
	Contract   string
	StartDate  string
	Location   string
	Notes      string

	Channel    string
	PPK        string
	SubServerA string
	SubServerB string
	SIM1       string
	SIM2       string

	TestControlEnabled bool
	TestIntervalMin    string

	Personals []ObjectWizardReviewPersonalItem
	Zones     []ObjectWizardReviewZoneItem

	Latitude  string
	Longitude string

	CardValidationErr error
}

// ObjectWizardReviewViewModel інкапсулює форматування підсумку майстра.
type ObjectWizardReviewViewModel struct{}

func NewObjectWizardReviewViewModel() *ObjectWizardReviewViewModel {
	return &ObjectWizardReviewViewModel{}
}

func (vm *ObjectWizardReviewViewModel) BuildText(input ObjectWizardReviewInput) string {
	lines := []string{
		"1) Дані об'єкта",
		fmt.Sprintf("№ об'єкта: %s", fallbackText(input.ObjN, "—")),
		fmt.Sprintf("Коротка назва: %s", fallbackText(input.ShortName, "—")),
		fmt.Sprintf("Повна назва: %s", fallbackText(input.FullName, "—")),
		fmt.Sprintf("Тип: %s", fallbackText(input.ObjectType, "—")),
		fmt.Sprintf("Район: %s", fallbackText(input.Region, "—")),
		fmt.Sprintf("Прихований №: %s", fallbackText(input.HiddenN, "—")),
		fmt.Sprintf("Адреса: %s", fallbackText(input.Address, "—")),
		fmt.Sprintf("Телефони: %s", fallbackText(input.Phones, "—")),
		fmt.Sprintf("Договір: %s", fallbackText(input.Contract, "—")),
		fmt.Sprintf("Дата: %s", fallbackText(input.StartDate, "—")),
		fmt.Sprintf("Розташування: %s", fallbackText(input.Location, "—")),
		fmt.Sprintf("Інформація: %s", fallbackText(input.Notes, "—")),
		"",
		"2) Параметри пристрою",
		fmt.Sprintf("Канал: %s", fallbackText(input.Channel, "—")),
		fmt.Sprintf("ППК: %s", fallbackText(input.PPK, "—")),
		fmt.Sprintf("Підсервер A: %s", fallbackText(input.SubServerA, "—")),
		fmt.Sprintf("Підсервер B: %s", fallbackText(input.SubServerB, "—")),
		fmt.Sprintf("SIM 1: %s", fallbackText(input.SIM1, "—")),
		fmt.Sprintf("SIM 2: %s", fallbackText(input.SIM2, "—")),
		fmt.Sprintf("Контроль тестів: %t", input.TestControlEnabled),
		fmt.Sprintf("Інтервал тесту, хв: %s", fallbackText(input.TestIntervalMin, "—")),
		"",
		"3) Зв'язані дані",
		fmt.Sprintf("В/О: %d", len(input.Personals)),
		fmt.Sprintf("Зони: %d", len(input.Zones)),
		fmt.Sprintf("Координати: %s / %s", fallbackText(input.Latitude, "0"), fallbackText(input.Longitude, "0")),
	}

	if input.CardValidationErr != nil {
		lines = append(lines, "")
		lines = append(lines, "Увага: перед створенням виправте:")
		lines = append(lines, "- "+input.CardValidationErr.Error())
	}

	if len(input.Personals) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Список В/О:")
		for i, item := range input.Personals {
			role := "Оператор"
			if item.IsAdmin {
				role = "Адмін"
			}
			lines = append(lines, fmt.Sprintf("%d) #%d %s, %s, %s",
				i+1,
				item.Number,
				fallbackText(item.FullName, "(без ПІБ)"),
				role,
				fallbackText(item.Phones, "—"),
			))
		}
	}

	if len(input.Zones) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Список зон:")
		for i, zone := range input.Zones {
			lines = append(lines, fmt.Sprintf("%d) ZONEN=%d, %s",
				i+1,
				zone.Number,
				fallbackText(zone.Description, "—"),
			))
		}
	}

	return strings.Join(lines, "\n")
}

func fallbackText(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
