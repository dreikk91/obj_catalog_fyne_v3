package viewmodels

import (
	"errors"
	"strings"
	"testing"
)

func TestObjectWizardReviewViewModel_BuildText_BaseAndFallbacks(t *testing.T) {
	vm := NewObjectWizardReviewViewModel()

	text := vm.BuildText(ObjectWizardReviewInput{
		ObjN:               "1001",
		ShortName:          "Об'єкт 1",
		FullName:           "Повна назва",
		ObjectType:         "Тип А",
		Region:             "Район 1",
		Address:            "Адреса 1",
		Phones:             "0501112233",
		Contract:           "Договір",
		StartDate:          "01.01.2026",
		Channel:            "5 - GPRS",
		TestControlEnabled: true,
		TestIntervalMin:    "9",
	})

	required := []string{
		"1) Дані об'єкта",
		"№ об'єкта: 1001",
		"Координати: 0 / 0",
		"Контроль тестів: true",
		"Інтервал тесту, хв: 9",
	}
	for _, item := range required {
		if !strings.Contains(text, item) {
			t.Fatalf("review must contain %q\n%s", item, text)
		}
	}
}

func TestObjectWizardReviewViewModel_BuildText_IncludesValidationError(t *testing.T) {
	vm := NewObjectWizardReviewViewModel()
	text := vm.BuildText(ObjectWizardReviewInput{
		CardValidationErr: errors.New("некоректний об'єктовий номер"),
	})

	if !strings.Contains(text, "Увага: перед створенням виправте:") {
		t.Fatalf("missing warning section:\n%s", text)
	}
	if !strings.Contains(text, "- некоректний об'єктовий номер") {
		t.Fatalf("missing validation error message:\n%s", text)
	}
}

func TestObjectWizardReviewViewModel_BuildText_IncludesPersonalsAndZones(t *testing.T) {
	vm := NewObjectWizardReviewViewModel()
	text := vm.BuildText(ObjectWizardReviewInput{
		Personals: []ObjectWizardReviewPersonalItem{
			{Number: 1, FullName: "Іван Петренко", Phones: "0501", IsAdmin: true},
			{Number: 2, FullName: "Олег Коваль", Phones: "0502", IsAdmin: false},
		},
		Zones: []ObjectWizardReviewZoneItem{
			{Number: 1, Description: "Шлейф 1"},
			{Number: 2, Description: "Шлейф 2"},
		},
	})

	required := []string{
		"Список В/О:",
		"1) #1 Іван Петренко, Адмін, 0501",
		"2) #2 Олег Коваль, Оператор, 0502",
		"Список зон:",
		"1) ZONEN=1, Шлейф 1",
		"2) ZONEN=2, Шлейф 2",
	}
	for _, item := range required {
		if !strings.Contains(text, item) {
			t.Fatalf("review must contain %q\n%s", item, text)
		}
	}
}
