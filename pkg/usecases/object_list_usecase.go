package usecases

import "obj_catalog_fyne_v3/pkg/models"

// ObjectListRepository описує мінімальне джерело об'єктів для use case списку.
type ObjectListRepository interface {
	GetObjects() []models.Object
}

// ObjectListUseCase інкапсулює сценарій отримання списку об'єктів.
type ObjectListUseCase struct {
	repository ObjectListRepository
}

func NewObjectListUseCase(repository ObjectListRepository) *ObjectListUseCase {
	return &ObjectListUseCase{repository: repository}
}

// FetchObjects повертає копію списку об'єктів для подальшої обробки у ViewModel.
func (uc *ObjectListUseCase) FetchObjects() []models.Object {
	if uc == nil || uc.repository == nil {
		return nil
	}
	objects := uc.repository.GetObjects()
	return append([]models.Object(nil), objects...)
}
