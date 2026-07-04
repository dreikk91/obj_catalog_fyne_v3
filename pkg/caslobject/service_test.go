package caslobject

import (
	"context"
	"reflect"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type createDraftProviderStub struct {
	contracts.CASLObjectEditorProvider
	calls []string
}

func (stub *createDraftProviderStub) CreateCASLObject(context.Context, contracts.CASLGuardObjectCreate) (string, error) {
	stub.calls = append(stub.calls, "object")
	return "42", nil
}

func (stub *createDraftProviderStub) IsCASLDeviceNumberInUse(context.Context, int64) (bool, error) {
	stub.calls = append(stub.calls, "check-device")
	return false, nil
}

func (stub *createDraftProviderStub) CreateCASLDevice(context.Context, contracts.CASLDeviceCreate) (string, error) {
	stub.calls = append(stub.calls, "device")
	return "77", nil
}

func (stub *createDraftProviderStub) CreateCASLDeviceLine(context.Context, contracts.CASLDeviceLineMutation) error {
	stub.calls = append(stub.calls, "line")
	return nil
}

func (stub *createDraftProviderStub) CreateCASLImage(_ context.Context, request contracts.CASLImageCreateRequest) error {
	if request.RoomID == "" {
		stub.calls = append(stub.calls, "object-image")
	} else {
		stub.calls = append(stub.calls, "room-image")
	}
	return nil
}

func (stub *createDraftProviderStub) CreateCASLRoom(context.Context, contracts.CASLRoomCreate) error {
	stub.calls = append(stub.calls, "room")
	return nil
}

func (stub *createDraftProviderStub) GetCASLObjectEditorSnapshot(context.Context, int64) (contracts.CASLObjectEditorSnapshot, error) {
	stub.calls = append(stub.calls, "reload")
	return contracts.CASLObjectEditorSnapshot{
		Object: contracts.CASLGuardObjectDetails{
			Rooms: []contracts.CASLRoomDetails{{RoomID: "91", Name: "Office"}},
			Device: contracts.CASLDeviceDetails{
				Lines: []contracts.CASLDeviceLineDetails{{LineNumber: 1}},
			},
		},
	}, nil
}

func (stub *createDraftProviderStub) AddCASLUserToRoom(context.Context, contracts.CASLAddUserToRoomRequest) error {
	stub.calls = append(stub.calls, "user")
	return nil
}

func (stub *createDraftProviderStub) AddCASLLineToRoom(context.Context, contracts.CASLLineToRoomBinding) error {
	stub.calls = append(stub.calls, "binding")
	return nil
}

func TestCreateDraftObjectRunsCompleteCreationFlow(t *testing.T) {
	provider := &createDraftProviderStub{}
	draft := contracts.CASLObjectEditorSnapshot{
		Object: contracts.CASLGuardObjectDetails{
			Name:   "Object",
			Images: []string{"data:image/png;base64,cG5n"},
			Device: contracts.CASLDeviceDetails{
				Number: 1001,
				Lines:  []contracts.CASLDeviceLineDetails{{LineNumber: 1}},
			},
			Rooms: []contracts.CASLRoomDetails{{
				Name:   "Office",
				Images: []string{"data:image/jpeg;base64,anBn"},
				Users:  []contracts.CASLRoomUserLink{{UserID: "5", Priority: 1}},
				Lines:  []contracts.CASLRoomLineLink{{LineNumber: 1}},
			}},
		},
	}

	objID, objectID, err := CreateDraft(context.Background(), provider, draft)
	if err != nil {
		t.Fatalf("CreateDraftObject() error = %v", err)
	}
	if objID != "42" || objectID != 42 {
		t.Fatalf("CreateDraftObject() ids = %q, %d", objID, objectID)
	}
	want := []string{
		"object", "check-device", "device", "line", "object-image",
		"room", "reload", "user", "room-image", "binding",
	}
	if !reflect.DeepEqual(provider.calls, want) {
		t.Fatalf("calls = %#v, want %#v", provider.calls, want)
	}
}

func TestNormalizeUAPhone(t *testing.T) {
	tests := map[string]string{
		"050 123 45 67": "+38 (050) 123-45-67",
		"380631234567":  "+38 (063) 123-45-67",
		"+380931112233": "+38 (093) 111-22-33",
		"+380671234567": "+38 (067) 123-45-67",
		"380671234567":  "+38 (067) 123-45-67",
		"0671234567":    "+38 (067) 123-45-67",
	}
	for input, want := range tests {
		got, err := NormalizeUAPhone(input)
		if err != nil {
			t.Fatalf("NormalizeUAPhone(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeUAPhone(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := NormalizeUAPhone("123"); err == nil {
		t.Fatal("NormalizeUAPhone() expected validation error")
	}
}

func TestNormalizeImageType(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"jpg":        "jpeg",
		".JPEG":      "jpeg",
		"image/jpeg": "jpeg",
		"png":        "png",
		"gif":        "gif",
		"bmp":        "bmp",
	}
	for input, want := range tests {
		got, err := NormalizeImageType(input)
		if err != nil {
			t.Fatalf("NormalizeImageType(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeImageType(%q) = %q, want %q", input, got, want)
		}
	}
	for _, input := range []string{"webp", "svg", ""} {
		if _, err := NormalizeImageType(input); err == nil {
			t.Fatalf("NormalizeImageType(%q) expected error", input)
		}
	}
}

func TestDraftImageUsesCASLImageTypes(t *testing.T) {
	t.Parallel()

	imageType, payload, ok := DraftImage("data:image/jpg;base64,ZmFrZQ==")
	if !ok || imageType != "jpeg" || payload != "ZmFrZQ==" {
		t.Fatalf("DraftImage(jpg) = %q, %q, %v", imageType, payload, ok)
	}
	if _, _, ok := DraftImage("data:image/webp;base64,ZmFrZQ=="); ok {
		t.Fatal("DraftImage(webp) must reject unsupported CASL image type")
	}
}

func TestDeviceTypeOptionsUsesHumanNames(t *testing.T) {
	dictionary := map[string]any{
		"device_types": map[string]any{
			"TYPE_DEVICE_DUNAY_4L": "TYPE_DEVICE_DUNAY_4L",
		},
		"devices": []any{
			map[string]any{"type": "TYPE_DEVICE_CUSTOM_PANEL"},
		},
	}

	_, values := DeviceTypeOptions(dictionary)
	if values["Дунай-4L"] != "TYPE_DEVICE_DUNAY_4L" {
		t.Fatalf("Dunay option = %q", values["Дунай-4L"])
	}
	if values["CUSTOM PANEL"] != "TYPE_DEVICE_CUSTOM_PANEL" {
		t.Fatalf("custom option = %q", values["CUSTOM PANEL"])
	}
	_, builtIns := DeviceTypeOptions(nil)
	if builtIns["Дунай-4L"] != "TYPE_DEVICE_Dunay_4L" {
		t.Fatalf("built-in Dunay option = %q", builtIns["Дунай-4L"])
	}
}

func TestAdapterTypesForDevice(t *testing.T) {
	if got := AdapterTypesForDevice("TYPE_DEVICE_CUSTOM_PANEL"); !reflect.DeepEqual(got, []string{"SYS"}) {
		t.Fatalf("custom adapters = %#v", got)
	}
	wantDunay := []string{"SYS", "AD3L", "AD6L", "AD6WL", "UTS4"}
	if got := AdapterTypesForDevice("TYPE_DEVICE_DUNAY_4L"); !reflect.DeepEqual(got, wantDunay) {
		t.Fatalf("Dunay adapters = %#v, want %#v", got, wantDunay)
	}
}
