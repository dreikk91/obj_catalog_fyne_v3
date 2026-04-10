package dialogs

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestNormalizeCASLEditorSIM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "formatted", input: "+38 (050) 123-45-67", want: "+38 (050) 123-45-67"},
		{name: "digits 12", input: "380501234567", want: "+38 (050) 123-45-67"},
		{name: "digits 10", input: "0501234567", want: "+38 (050) 123-45-67"},
		{name: "digits 11", input: "80501234567", want: "+38 (050) 123-45-67"},
		{name: "empty", input: "", want: ""},
		{name: "partial", input: "+38 (050) 123-__", wantErr: true},
		{name: "invalid", input: "12345", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeCASLEditorSIM(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeCASLEditorSIM(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeCASLEditorSIM(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeCASLEditorSIM(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCASLEditorLicenceForSave(t *testing.T) {
	t.Parallel()

	got, err := normalizeCASLEditorLicenceForSave("123-456-789-012-345-678")
	if err != nil {
		t.Fatalf("normalizeCASLEditorLicenceForSave() unexpected error: %v", err)
	}
	if got != "123;456;789;012;345;678" {
		t.Fatalf("normalizeCASLEditorLicenceForSave() = %q", got)
	}

	if _, err := normalizeCASLEditorLicenceForSave("123-456"); err == nil {
		t.Fatal("normalizeCASLEditorLicenceForSave() expected error for short key")
	}
}

func TestValidateRoomUserHozNum(t *testing.T) {
	t.Parallel()

	state := &caslObjectEditorState{
		roomSelected: 0,
		roomUsersLocal: []contracts.CASLRoomUserLink{
			{UserID: "1", HozNum: "10"},
			{UserID: "2", HozNum: ""},
		},
		snapshot: contracts.CASLObjectEditorSnapshot{
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

	if err := state.validateRoomUserHozNum(1, "12"); err != nil {
		t.Fatalf("validateRoomUserHozNum valid value returned error: %v", err)
	}
	if err := state.validateRoomUserHozNum(1, "11"); err == nil {
		t.Fatal("validateRoomUserHozNum duplicate value expected error")
	}
	if err := state.validateRoomUserHozNum(1, "129"); err == nil {
		t.Fatal("validateRoomUserHozNum out of range expected error")
	}
}
