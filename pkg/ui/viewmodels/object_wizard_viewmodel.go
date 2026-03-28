package viewmodels

import (
	"fmt"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectWizardPersistence описує мінімальні backend-операції майстра створення об'єкта.
type ObjectWizardPersistence interface {
	CreateObject(card contracts.AdminObjectCard) error
	AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error
	AddObjectZone(objn int64, zone contracts.AdminObjectZone) error
	SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error
}

// ObjectWizardCreateResult містить підсумок команди створення об'єкта в майстрі.
type ObjectWizardCreateResult struct {
	ObjN          int64
	StatusMessage string
	Warnings      []string
}

// ObjectWizardViewModel інкапсулює сценарій створення об'єкта з пов'язаними даними.
type ObjectWizardViewModel struct{}

func NewObjectWizardViewModel() *ObjectWizardViewModel {
	return &ObjectWizardViewModel{}
}

// ObjectWizardStepValidationInput описує мінімальні дані для валідації кроку майстра.
type ObjectWizardStepValidationInput struct {
	Step              int
	ObjNRaw           string
	ShortName         string
	SelectedObjTypeID int64
	CardBuildErr      error
}

func (vm *ObjectWizardViewModel) ValidateStep(input ObjectWizardStepValidationInput) error {
	if input.Step < 0 {
		return nil
	}

	objnRaw := strings.TrimSpace(input.ObjNRaw)
	if objnRaw == "" {
		return fmt.Errorf("вкажіть об'єктовий номер")
	}
	if _, err := strconv.ParseInt(objnRaw, 10, 64); err != nil {
		return fmt.Errorf("некоректний об'єктовий номер")
	}
	if strings.TrimSpace(input.ShortName) == "" {
		return fmt.Errorf("вкажіть коротку назву об'єкта")
	}
	if input.SelectedObjTypeID <= 0 {
		return fmt.Errorf("виберіть тип об'єкта")
	}
	if input.CardBuildErr != nil {
		return input.CardBuildErr
	}
	return nil
}

func (vm *ObjectWizardViewModel) CreateObjectWithRelatedData(
	persistence ObjectWizardPersistence,
	card contracts.AdminObjectCard,
	personals []contracts.AdminObjectPersonal,
	zones []contracts.AdminObjectZone,
	coords contracts.AdminObjectCoordinates,
) (ObjectWizardCreateResult, error) {
	if err := persistence.CreateObject(card); err != nil {
		return ObjectWizardCreateResult{
			StatusMessage: "Не вдалося створити об'єкт",
		}, err
	}

	warnings := make([]string, 0, 4)
	for idx, item := range personals {
		if err := persistence.AddObjectPersonal(card.ObjN, item); err != nil {
			warnings = append(warnings, fmt.Sprintf("В/О #%d не додано: %v", idx+1, err))
		}
	}
	for idx, zone := range zones {
		if err := persistence.AddObjectZone(card.ObjN, zone); err != nil {
			warnings = append(warnings, fmt.Sprintf("Зона #%d не додана: %v", idx+1, err))
		}
	}

	coords.Latitude = strings.TrimSpace(coords.Latitude)
	coords.Longitude = strings.TrimSpace(coords.Longitude)
	if err := persistence.SaveObjectCoordinates(card.ObjN, coords); err != nil {
		warnings = append(warnings, fmt.Sprintf("Координати не збережено: %v", err))
	}

	return ObjectWizardCreateResult{
		ObjN:          card.ObjN,
		StatusMessage: "Новий об'єкт створено",
		Warnings:      warnings,
	}, nil
}
