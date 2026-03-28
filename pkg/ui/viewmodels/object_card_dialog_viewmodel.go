package viewmodels

import "obj_catalog_fyne_v3/pkg/contracts"

// ObjectCardPersistence описує мінімально необхідний бекенд-контракт для збереження картки об'єкта.
type ObjectCardPersistence interface {
	GetObjectCard(objn int64) (contracts.AdminObjectCard, error)
	CreateObject(card contracts.AdminObjectCard) error
	UpdateObject(card contracts.AdminObjectCard) error
}

// ObjectCardSaveResult - результат команди збереження картки.
type ObjectCardSaveResult struct {
	ObjN          int64
	StatusMessage string
}

// ObjectCardDialogViewModel інкапсулює сценарій збереження картки (create/update).
type ObjectCardDialogViewModel struct{}

func NewObjectCardDialogViewModel() *ObjectCardDialogViewModel {
	return &ObjectCardDialogViewModel{}
}

func (vm *ObjectCardDialogViewModel) SaveObject(
	persistence ObjectCardPersistence,
	editObjN *int64,
	card contracts.AdminObjectCard,
) (ObjectCardSaveResult, error) {
	isEdit := editObjN != nil && *editObjN > 0
	if isEdit {
		loaded, err := persistence.GetObjectCard(*editObjN)
		if err != nil {
			return ObjectCardSaveResult{StatusMessage: "Не вдалося перезавантажити картку"}, err
		}
		card.ObjUIN = loaded.ObjUIN
		if err := persistence.UpdateObject(card); err != nil {
			return ObjectCardSaveResult{StatusMessage: "Не вдалося зберегти зміни об'єкта"}, err
		}
		return ObjectCardSaveResult{
			ObjN:          card.ObjN,
			StatusMessage: "Картку об'єкта оновлено",
		}, nil
	}

	if err := persistence.CreateObject(card); err != nil {
		return ObjectCardSaveResult{StatusMessage: "Не вдалося створити об'єкт"}, err
	}
	return ObjectCardSaveResult{
		ObjN:          card.ObjN,
		StatusMessage: "Новий об'єкт створено",
	}, nil
}
