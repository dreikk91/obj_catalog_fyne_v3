//go:build qt

package qtui

import (
	"context"
	"reflect"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type caslObjectEditDiffStub struct {
	contracts.CASLObjectEditorProvider
	calls         []string
	lineMutations []contracts.CASLDeviceLineMutation
	bindings      []contracts.CASLLineToRoomBinding
}

func TestFormatCASLPhoneForDisplay(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"380753163889":        "+38 (075) 316-38-89",
		"+380931234567":       "+38 (093) 123-45-67",
		"+38 (067) 344-74-85": "+38 (067) 344-74-85",
	}
	for input, want := range tests {
		if got := formatCASLPhoneForDisplay(input); got != want {
			t.Fatalf("formatCASLPhoneForDisplay(%q) = %q, want %q", input, got, want)
		}
	}
}

func (s *caslObjectEditDiffStub) UpdateCASLObject(context.Context, contracts.CASLGuardObjectUpdate) error {
	s.calls = append(s.calls, "object")
	return nil
}

func (s *caslObjectEditDiffStub) UpdateCASLDevice(context.Context, contracts.CASLDeviceUpdate) error {
	s.calls = append(s.calls, "device")
	return nil
}

func (s *caslObjectEditDiffStub) UpdateCASLRoom(context.Context, contracts.CASLRoomUpdate) error {
	s.calls = append(s.calls, "room")
	return nil
}

func (s *caslObjectEditDiffStub) UpdateCASLDeviceLine(_ context.Context, mutation contracts.CASLDeviceLineMutation) error {
	s.calls = append(s.calls, "line")
	s.lineMutations = append(s.lineMutations, mutation)
	return nil
}

func (s *caslObjectEditDiffStub) CreateCASLDeviceLine(_ context.Context, mutation contracts.CASLDeviceLineMutation) error {
	s.calls = append(s.calls, "create")
	s.lineMutations = append(s.lineMutations, mutation)
	return nil
}

func (s *caslObjectEditDiffStub) DeleteCASLDeviceLine(context.Context, string, int) error {
	s.calls = append(s.calls, "delete")
	return nil
}

func (s *caslObjectEditDiffStub) AddCASLLineToRoom(_ context.Context, binding contracts.CASLLineToRoomBinding) error {
	s.calls = append(s.calls, "bind")
	s.bindings = append(s.bindings, binding)
	return nil
}

func (s *caslObjectEditDiffStub) RemoveCASLLineFromRoom(_ context.Context, binding contracts.CASLLineToRoomBinding) error {
	s.calls = append(s.calls, "unbind")
	s.bindings = append(s.bindings, binding)
	return nil
}

func TestCASLObjectSaveExistingUpdatesOnlyChangedLine(t *testing.T) {
	lineID := int64(11)
	original := contracts.CASLGuardObjectDetails{
		ObjID: "29",
		Device: contracts.CASLDeviceDetails{
			DeviceID: "28",
			Number:   1007,
			Lines: []contracts.CASLDeviceLineDetails{{
				LineID: &lineID, LineNumber: 1, Description: "Старий опис", LineType: "NORMAL", RoomID: "36",
			}},
		},
		Rooms: []contracts.CASLRoomDetails{{RoomID: "36", Name: "Офіс"}},
	}
	current := cloneCASLGuardObject(original)
	current.Device.Lines[0].Description = "Новий опис"
	provider := &caslObjectEditDiffStub{}
	state := &caslObjectDialogState{
		provider: provider,
		original: original,
		snapshot: contracts.CASLObjectEditorSnapshot{Object: current},
	}

	if _, err := state.saveExisting(context.Background()); err != nil {
		t.Fatalf("saveExisting() error = %v", err)
	}
	if len(provider.calls) != 1 || provider.calls[0] != "line" {
		t.Fatalf("calls = %v, want only line update", provider.calls)
	}
}

func TestCASLObjectSaveExistingUsesOriginalRoomLinksForBindingDiff(t *testing.T) {
	lineID := int64(11)
	original := contracts.CASLGuardObjectDetails{
		ObjID: "29",
		Device: contracts.CASLDeviceDetails{
			DeviceID: "28",
			Number:   1007,
			Lines: []contracts.CASLDeviceLineDetails{{
				LineID: &lineID, LineNumber: 1, Description: "Старий опис", LineType: "NORMAL",
			}},
		},
		Rooms: []contracts.CASLRoomDetails{{
			RoomID: "36",
			Name:   "Офіс",
			Lines:  []contracts.CASLRoomLineLink{{LineNumber: 1}},
		}},
	}
	current := cloneCASLGuardObject(original)
	current.Device.Lines[0].Description = "Новий опис"
	current.Device.Lines[0].RoomID = "36"
	provider := &caslObjectEditDiffStub{}
	state := &caslObjectDialogState{
		provider: provider,
		original: original,
		snapshot: contracts.CASLObjectEditorSnapshot{Object: current},
	}

	if _, err := state.saveExisting(context.Background()); err != nil {
		t.Fatalf("saveExisting() error = %v", err)
	}
	if len(provider.calls) != 1 || provider.calls[0] != "line" {
		t.Fatalf("calls = %v, want only line update without duplicate binding", provider.calls)
	}
}

func TestCASLObjectSaveExistingRenumbersLineByRecreatingAndRebinding(t *testing.T) {
	lineID := int64(11)
	original := contracts.CASLGuardObjectDetails{
		ObjID: "29",
		Device: contracts.CASLDeviceDetails{
			DeviceID: "28",
			Number:   1007,
			Lines: []contracts.CASLDeviceLineDetails{{
				LineID: &lineID, LineNumber: 5, Description: "Зона", LineType: "NORMAL", RoomID: "36",
			}},
		},
		Rooms: []contracts.CASLRoomDetails{{
			RoomID: "36",
			Name:   "Офіс",
			Lines:  []contracts.CASLRoomLineLink{{LineNumber: 5}},
		}},
	}
	current := cloneCASLGuardObject(original)
	current.Device.Lines[0].LineNumber = 10
	provider := &caslObjectEditDiffStub{}
	state := &caslObjectDialogState{
		provider: provider,
		original: original,
		snapshot: contracts.CASLObjectEditorSnapshot{Object: current},
	}

	if _, err := state.saveExisting(context.Background()); err != nil {
		t.Fatalf("saveExisting() error = %v", err)
	}
	wantCalls := []string{"unbind", "delete", "create", "bind"}
	if !reflect.DeepEqual(provider.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", provider.calls, wantCalls)
	}
	if len(provider.lineMutations) != 1 {
		t.Fatalf("line mutations = %d, want 1", len(provider.lineMutations))
	}
	mutation := provider.lineMutations[0]
	if mutation.LineID != nil || mutation.LineNumber != 10 {
		t.Fatalf("line mutation = %+v, want a newly created line number 10", mutation)
	}
}

func TestCASLObjectSaveExistingMovesLineByUnbindingBeforeBinding(t *testing.T) {
	lineID := int64(11)
	original := contracts.CASLGuardObjectDetails{
		ObjID: "29",
		Device: contracts.CASLDeviceDetails{
			DeviceID: "28",
			Number:   1007,
			Lines: []contracts.CASLDeviceLineDetails{{
				LineID: &lineID, LineNumber: 1, Description: "Зона", LineType: "NORMAL", RoomID: "36",
			}},
		},
		Rooms: []contracts.CASLRoomDetails{
			{RoomID: "36", Name: "Офіс"},
			{RoomID: "37", Name: "Склад"},
		},
	}
	current := cloneCASLGuardObject(original)
	current.Device.Lines[0].RoomID = "37"
	provider := &caslObjectEditDiffStub{}
	state := &caslObjectDialogState{
		provider: provider,
		original: original,
		snapshot: contracts.CASLObjectEditorSnapshot{Object: current},
	}

	if _, err := state.saveExisting(context.Background()); err != nil {
		t.Fatalf("saveExisting() error = %v", err)
	}
	if len(provider.calls) != 2 || provider.calls[0] != "unbind" || provider.calls[1] != "bind" {
		t.Fatalf("calls = %v, want unbind then bind", provider.calls)
	}
}
