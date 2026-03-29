package usecases

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

type objectListRepoStub struct {
	objects []models.Object
}

func (s *objectListRepoStub) GetObjects() []models.Object {
	return append([]models.Object(nil), s.objects...)
}

func TestObjectListUseCase_FetchObjectsReturnsCopy(t *testing.T) {
	stub := &objectListRepoStub{
		objects: []models.Object{
			{ID: 1, Name: "Obj 1"},
			{ID: 2, Name: "Obj 2"},
		},
	}
	uc := NewObjectListUseCase(stub)

	got := uc.FetchObjects()
	if len(got) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(got))
	}
	got[0].Name = "Changed"
	if stub.objects[0].Name != "Obj 1" {
		t.Fatalf("use case must return copy, repository data changed")
	}
}

func TestObjectListUseCase_FetchObjectsNilRepository(t *testing.T) {
	uc := NewObjectListUseCase(nil)
	got := uc.FetchObjects()
	if len(got) != 0 {
		t.Fatalf("expected empty result for nil repository, got %d", len(got))
	}
}
