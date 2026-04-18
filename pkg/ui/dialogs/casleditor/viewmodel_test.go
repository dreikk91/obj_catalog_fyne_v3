package casleditor

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"testing"
)

func TestValidateRoomUserHozNum(t *testing.T) {
	t.Parallel()

	vm := &EditorViewModel{
		RoomSelected: 0,
		RoomUsersLocal: []contracts.CASLRoomUserLink{
			{UserID: "1", HozNum: "10"},
			{UserID: "2", HozNum: ""},
		},
		Snapshot: contracts.CASLObjectEditorSnapshot{
			Object: contracts.CASLGuardObjectDetails{
				Rooms: []contracts.CASLRoomDetails{
					{
						RoomID: "r1",
						Users: []contracts.CASLRoomUserLink{
							{UserID: "1", HozNum: "10"},
							{UserID: "2", HozNum: ""},
						},
					},
					{
						RoomID: "r2",
						Users: []contracts.CASLRoomUserLink{
							{UserID: "3", HozNum: "11"},
						},
					},
				},
			},
		},
	}

	if err := vm.ValidateRoomUserHozNum(1, "12"); err != nil {
		t.Fatalf("ValidateRoomUserHozNum valid value returned error: %v", err)
	}
	if err := vm.ValidateRoomUserHozNum(1, "11"); err == nil {
		t.Fatal("ValidateRoomUserHozNum duplicate value expected error")
	}
	if err := vm.ValidateRoomUserHozNum(1, "129"); err == nil {
		t.Fatal("ValidateRoomUserHozNum out of range expected error")
	}
}

func TestValidateCASLLineNumberUnique(t *testing.T) {
	t.Parallel()

	lines := []contracts.CASLDeviceLineDetails{
		{LineNumber: 1},
		{LineNumber: 2},
	}

	if err := ValidateCASLLineNumberUnique(lines, 3, -1); err != nil {
		t.Fatalf("ValidateCASLLineNumberUnique unique value returned error: %v", err)
	}
	if err := ValidateCASLLineNumberUnique(lines, 2, -1); err == nil {
		t.Fatal("ValidateCASLLineNumberUnique duplicate value expected error")
	}
	if err := ValidateCASLLineNumberUnique(lines, 2, 1); err != nil {
		t.Fatalf("ValidateCASLLineNumberUnique excluded current row returned error: %v", err)
	}
}

func TestPendingAutoRoomLineBindings(t *testing.T) {
	t.Parallel()

	vm := &EditorViewModel{
		creating: true, // Use internal field via same package test
		Snapshot: contracts.CASLObjectEditorSnapshot{
			Object: contracts.CASLGuardObjectDetails{
				ObjID: "obj-1",
				Device: contracts.CASLDeviceDetails{
					DeviceID: "dev-1",
					Lines: []contracts.CASLDeviceLineDetails{
						{LineNumber: 1},
						{LineNumber: 2},
					},
				},
				Rooms: []contracts.CASLRoomDetails{
					{RoomID: "room-1"},
				},
			},
		},
	}

	got := vm.PendingAutoRoomLineBindings()
	if len(got) != 2 {
		t.Fatalf("PendingAutoRoomLineBindings() len = %d, want 2", len(got))
	}
	if got[0].RoomID != "room-1" || got[0].DeviceID != "dev-1" || got[0].ObjID != "obj-1" || got[0].LineNumber != 1 {
		t.Fatalf("PendingAutoRoomLineBindings() first binding = %+v", got[0])
	}

	vm.Snapshot.Object.Rooms[0].Lines = []contracts.CASLRoomLineLink{{LineNumber: 1}}
	if got := vm.PendingAutoRoomLineBindings(); len(got) != 0 {
		t.Fatalf("PendingAutoRoomLineBindings() expected no bindings for non-empty room, got %+v", got)
	}
}

func TestMutationToDetails(t *testing.T) {
	t.Parallel()

	vm := &EditorViewModel{}
	lineID := int64(15)
	mutation := contracts.CASLDeviceLineMutation{
		LineID:        &lineID,
		LineNumber:    3,
		GroupNumber:   2,
		AdapterType:   "SYS",
		AdapterNumber: 5,
		Description:   "Вхідні двері",
		LineType:      "NORMAL",
		IsBlocked:     true,
	}

	got := vm.mutationToDetails(mutation, "room-7")
	if got.LineID != mutation.LineID {
		t.Fatal("mutationToDetails() should preserve line ID")
	}
	if got.RoomID != "room-7" {
		t.Fatal("mutationToDetails() should set room binding")
	}
	if got.LineNumber != mutation.LineNumber || got.GroupNumber != mutation.GroupNumber || got.AdapterType != mutation.AdapterType || got.AdapterNumber != mutation.AdapterNumber || got.Description != mutation.Description || got.LineType != mutation.LineType || got.IsBlocked != mutation.IsBlocked {
		t.Fatalf("mutationToDetails() did not copy mutation fields correctly: %+v", got)
	}
}
