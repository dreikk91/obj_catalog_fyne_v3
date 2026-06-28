package caslobject

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// CreateDraft creates an object, device, lines, images, rooms and their bindings.
func CreateDraft(
	ctx context.Context,
	provider contracts.CASLObjectEditorProvider,
	draft contracts.CASLObjectEditorSnapshot,
) (string, int64, error) {
	if provider == nil {
		return "", 0, fmt.Errorf("casl provider не налаштовано")
	}
	object := draft.Object
	objID, err := provider.CreateCASLObject(ctx, contracts.CASLGuardObjectCreate{
		Name: object.Name, Address: object.Address, Long: object.Long, Lat: object.Lat,
		Description: object.Description, Contract: object.Contract, ManagerID: object.ManagerID,
		Note: object.Note, StartDate: object.StartDate, Status: object.ObjectStatus,
		ObjectType: object.ObjectType, IDRequest: object.IDRequest, ReactingPultID: object.ReactingPultID,
		GeoZoneID: object.GeoZoneID, BusinessCoeff: object.BusinessCoeff,
	})
	if err != nil {
		return "", 0, fmt.Errorf("створення об'єкта: %w", err)
	}
	objectID, err := strconv.ParseInt(strings.TrimSpace(objID), 10, 64)
	if err != nil {
		return objID, 0, fmt.Errorf("casl повернув некоректний obj_id %q", objID)
	}
	device := object.Device
	inUse, err := provider.IsCASLDeviceNumberInUse(ctx, device.Number)
	if err != nil {
		return objID, objectID, fmt.Errorf("перевірка номера приладу: %w", err)
	}
	if inUse {
		return objID, objectID, fmt.Errorf("номер приладу %d вже зайнятий", device.Number)
	}
	deviceID, err := provider.CreateCASLDevice(ctx, contracts.CASLDeviceCreate{
		Number: device.Number, Name: device.Name, DeviceType: device.Type, Timeout: device.Timeout,
		SIM1: device.SIM1, SIM2: device.SIM2, TechnicianID: device.TechnicianID,
		Units: device.Units, Requisites: device.Requisites, ChangeDate: device.ChangeDate,
		ReglamentDate: device.ReglamentDate, MoreAlarmTime: device.MoreAlarmTime,
		IgnoringAlarmTime: device.IgnoringAlarmTime, LicenceKey: device.LicenceKey, PasswRemote: device.PasswRemote,
	})
	if err != nil {
		return objID, objectID, fmt.Errorf("створення приладу: %w", err)
	}
	for _, line := range device.Lines {
		if err := provider.CreateCASLDeviceLine(ctx, contracts.CASLDeviceLineMutation{
			DeviceID: deviceID, LineNumber: line.LineNumber, GroupNumber: line.GroupNumber,
			AdapterType: line.AdapterType, AdapterNumber: line.AdapterNumber,
			Description: line.Description, LineType: line.LineType, IsBlocked: line.IsBlocked,
		}); err != nil {
			return objID, objectID, fmt.Errorf("не вдалося створити зону #%d: %w", line.LineNumber, err)
		}
	}
	for _, raw := range object.Images {
		imageType, payload, ok := DraftImage(raw)
		if ok {
			if err := provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
				ObjID: objID, ImageType: imageType, ImageData: payload,
			}); err != nil {
				return objID, objectID, fmt.Errorf("фото об'єкта: %w", err)
			}
		}
	}
	for _, room := range object.Rooms {
		if err := provider.CreateCASLRoom(ctx, contracts.CASLRoomCreate{
			ObjID: objID, Name: room.Name, Description: room.Description, RTSP: room.RTSP,
		}); err != nil {
			return objID, objectID, fmt.Errorf("не вдалося створити приміщення %q: %w", room.Name, err)
		}
	}
	reloaded, err := provider.GetCASLObjectEditorSnapshot(ctx, objectID)
	if err != nil {
		return objID, objectID, fmt.Errorf("оновлення створеного об'єкта: %w", err)
	}
	roomIDs := make(map[string]string, len(reloaded.Object.Rooms))
	for _, room := range reloaded.Object.Rooms {
		roomIDs[strings.ToLower(strings.TrimSpace(room.Name))] = room.RoomID
	}
	lineNumbers := make(map[int]struct{}, len(reloaded.Object.Device.Lines))
	for _, line := range reloaded.Object.Device.Lines {
		lineNumbers[line.LineNumber] = struct{}{}
	}
	for _, room := range object.Rooms {
		roomID := roomIDs[strings.ToLower(strings.TrimSpace(room.Name))]
		if roomID == "" {
			return objID, objectID, fmt.Errorf("casl не повернув room_id для приміщення %q", room.Name)
		}
		for _, user := range room.Users {
			if err := provider.AddCASLUserToRoom(ctx, contracts.CASLAddUserToRoomRequest{
				ObjID: objID, RoomID: roomID, UserID: user.UserID,
				Priority: user.Priority, HozNum: user.HozNum,
			}); err != nil {
				return objID, objectID, fmt.Errorf("не вдалося додати користувача до %q: %w", room.Name, err)
			}
		}
		for _, raw := range room.Images {
			imageType, payload, ok := DraftImage(raw)
			if ok {
				if err := provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
					ObjID: objID, RoomID: roomID, ImageType: imageType, ImageData: payload,
				}); err != nil {
					return objID, objectID, fmt.Errorf("фото приміщення %q: %w", room.Name, err)
				}
			}
		}
		for _, line := range room.Lines {
			if _, exists := lineNumbers[line.LineNumber]; !exists {
				return objID, objectID, fmt.Errorf("зона #%d не була створена", line.LineNumber)
			}
			if err := provider.AddCASLLineToRoom(ctx, contracts.CASLLineToRoomBinding{
				ObjID: objID, DeviceID: deviceID, LineNumber: line.LineNumber, RoomID: roomID,
			}); err != nil {
				return objID, objectID, fmt.Errorf("не вдалося прив'язати зону #%d до %q: %w", line.LineNumber, room.Name, err)
			}
		}
	}
	return objID, objectID, nil
}

// DraftImage extracts an image type and base64 payload from a data URI.
func DraftImage(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return "", "", false
	}
	comma := strings.Index(raw, ",")
	if comma < 0 {
		return "", "", false
	}
	header := strings.ToLower(raw[:comma])
	imageType := "jpg"
	for _, candidate := range []string{"png", "webp", "gif", "bmp", "svg"} {
		if strings.Contains(header, "image/"+candidate) {
			imageType = candidate
			break
		}
	}
	return imageType, strings.TrimSpace(raw[comma+1:]), true
}

// DictionaryOptions returns sorted display labels and their raw CASL values.
func DictionaryOptions(dictionary map[string]any, keys ...string) ([]string, map[string]string) {
	values := map[string]string{}
	for _, key := range keys {
		raw, exists := dictionary[key]
		if !exists {
			continue
		}
		normalizeOptions(raw, values)
		if len(values) > 0 {
			break
		}
	}
	labels := make([]string, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels, values
}

// DeviceTypeOptions returns all built-in and user-defined CASL device types with human labels.
func DeviceTypeOptions(dictionary map[string]any) ([]string, map[string]string) {
	_, rawValues := DictionaryOptions(dictionary, "device_types", "user_device_types")
	rawByType := make(map[string]string, len(rawValues)+len(builtInDeviceTypes))
	setType := func(raw, label string) {
		for existing := range rawByType {
			if strings.EqualFold(existing, raw) {
				delete(rawByType, existing)
			}
		}
		rawByType[raw] = label
	}
	for raw, label := range builtInDeviceTypes {
		setType(raw, label)
	}
	for label, raw := range rawValues {
		setType(raw, label)
	}
	if devices, ok := dictionary["devices"].([]any); ok {
		for _, item := range devices {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if raw := firstString(entry, "type", "device_type", "value"); raw != "" {
				exists := false
				for existing := range rawByType {
					if strings.EqualFold(existing, raw) {
						exists = true
						break
					}
				}
				if !exists {
					setType(raw, DeviceTypeDisplayName(raw))
				}
			}
		}
	}
	values := make(map[string]string, len(rawByType))
	for raw, label := range rawByType {
		human := strings.TrimSpace(label)
		if human == "" || human == raw {
			human = DeviceTypeDisplayName(raw)
		}
		values[human] = raw
	}
	labels := make([]string, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels, values
}

// DeviceTypeDisplayName returns the operator-facing name used by the original CASL frontend.
func DeviceTypeDisplayName(raw string) string {
	if label, ok := deviceTypeNames[strings.ToUpper(strings.TrimSpace(raw))]; ok {
		return label
	}
	value := strings.TrimPrefix(strings.TrimSpace(raw), "TYPE_DEVICE_")
	value = strings.NewReplacer("_", " ", "-", " ").Replace(value)
	return strings.TrimSpace(value)
}

// AdapterTypesForDevice applies the original CASL adapter restrictions.
func AdapterTypesForDevice(deviceType string) []string {
	switch strings.ToUpper(strings.TrimSpace(deviceType)) {
	case "TYPE_DEVICE_DUNAY_4L", "TYPE_DEVICE_CASL":
		return []string{"SYS", "AD3L", "AD6L", "AD6WL", "UTS4"}
	case "TYPE_DEVICE_DUNAY_8(16)32_DUNAY_G1R",
		"TYPE_DEVICE_DUNAY_PSPN",
		"TYPE_DEVICE_DUNAY_PSPN_ECOM",
		"TYPE_DEVICE_DUNAY_8_32",
		"TYPE_DEVICE_DUNAY_16_32":
		return []string{"SYS", "TM", "AD3", "AD8", "RK4", "RL2", "RL4", "KJ", "KLK16", "KLK8", "KS16", "KS8", "KLPT", "SM16", "SM8", "TMR", "TMB", "TML"}
	default:
		return []string{"SYS"}
	}
}

var deviceTypeNames = map[string]string{
	"TYPE_DEVICE_CASL":                    "CASL",
	"TYPE_DEVICE_DUNAY_8L":                "Дунай-8L",
	"TYPE_DEVICE_DUNAY_16L":               "Дунай-16L",
	"TYPE_DEVICE_DUNAY_4L":                "Дунай-4L",
	"TYPE_DEVICE_LUN":                     "Лунь",
	"TYPE_DEVICE_AJAX":                    "Ajax",
	"TYPE_DEVICE_AJAX_SIA":                "Ajax (SIA)",
	"TYPE_DEVICE_BRON_SIA":                "Bron (SIA)",
	"TYPE_DEVICE_CASL_PLUS":               "CASL+",
	"TYPE_DEVICE_DOZOR_4":                 "Дозор-4",
	"TYPE_DEVICE_DOZOR_8":                 "Дозор-8",
	"TYPE_DEVICE_DOZOR_8MG":               "Дозор-8MG",
	"TYPE_DEVICE_DUNAY_8_32":              "Дунай-8/32",
	"TYPE_DEVICE_DUNAY_16_32":             "Дунай-16/32",
	"TYPE_DEVICE_DUNAY_4_3":               "Дунай-4.3",
	"TYPE_DEVICE_DUNAY_4_3S":              "Дунай-4.3.1S",
	"TYPE_DEVICE_DUNAY_8(16)32_DUNAY_G1R": "128 + G1R",
	"TYPE_DEVICE_DUNAY_STK":               "Дунай-СТК",
	"TYPE_DEVICE_DUNAY_4.2":               "4.2 + G1R",
	"TYPE_DEVICE_VBD4":                    "ВБД4 + G1R",
	"TYPE_DEVICE_VBD6_2":                  "ВБД6-2 + G1R",
	"TYPE_DEVICE_DUNAY_PSPN":              "ПСПН (R.COM)",
	"TYPE_DEVICE_DUNAY_PSPN_ECOM":         "ПСПН (ECOM)",
	"TYPE_DEVICE_VBD4_ECOM":               "ВБД4",
	"TYPE_DEVICE_VBD_16":                  "ВБД6-16",
}

var builtInDeviceTypes = map[string]string{
	"TYPE_DEVICE_CASL":                    "CASL",
	"TYPE_DEVICE_Dunay_8L":                "Дунай-8L",
	"TYPE_DEVICE_Dunay_16L":               "Дунай-16L",
	"TYPE_DEVICE_Dunay_4L":                "Дунай-4L",
	"TYPE_DEVICE_Lun":                     "Лунь",
	"TYPE_DEVICE_Ajax":                    "Ajax",
	"TYPE_DEVICE_Ajax_SIA":                "Ajax (SIA)",
	"TYPE_DEVICE_Bron_SIA":                "Bron (SIA)",
	"TYPE_DEVICE_CASL_PLUS":               "CASL+",
	"TYPE_DEVICE_Dozor_4":                 "Дозор-4",
	"TYPE_DEVICE_Dozor_8":                 "Дозор-8",
	"TYPE_DEVICE_Dozor_8MG":               "Дозор-8MG",
	"TYPE_DEVICE_Dunay_8_32":              "Дунай-8/32",
	"TYPE_DEVICE_Dunay_16_32":             "Дунай-16/32",
	"TYPE_DEVICE_Dunay_4_3":               "Дунай-4.3",
	"TYPE_DEVICE_Dunay_4_3S":              "Дунай-4.3.1S",
	"TYPE_DEVICE_Dunay_8(16)32_Dunay_G1R": "128 + G1R",
	"TYPE_DEVICE_Dunay_STK":               "Дунай-СТК",
	"TYPE_DEVICE_Dunay_4.2":               "4.2 + G1R",
	"TYPE_DEVICE_VBDb_2":                  "ВБД6-2 + G1R",
	"TYPE_DEVICE_VBD4":                    "ВБД4 + G1R",
	"TYPE_DEVICE_Dunay_PSPN":              "ПСПН (R.COM)",
	"TYPE_DEVICE_Dunay_PSPN_ECOM":         "ПСПН (ECOM)",
	"TYPE_DEVICE_VBD4_ECOM":               "ВБД4",
	"TYPE_DEVICE_VBD_16":                  "ВБД6-16",
	"FULL_SURGARD":                        "Full Surgard",
	"MAKS_PRO":                            "Макс-ПРО (MAKS_PRO)",
	"SATEL":                               "Satel",
	"Инттел":                              "Інттел",
	"\"Макс-ПРО\"":                        "Макс-ПРО",
}

func normalizeOptions(raw any, result map[string]string) {
	switch value := raw.(type) {
	case map[string]any:
		for key, item := range value {
			label := strings.TrimSpace(fmt.Sprint(item))
			if label == "" || strings.HasPrefix(label, "map[") {
				label = key
			}
			result[label] = key
		}
	case map[string]string:
		for key, label := range value {
			if strings.TrimSpace(label) == "" {
				label = key
			}
			result[label] = key
		}
	case []any:
		for _, item := range value {
			switch entry := item.(type) {
			case string:
				result[entry] = entry
			case map[string]any:
				rawValue := firstString(entry, "value", "type", "id", "code")
				label := firstString(entry, "label", "name", "title", "description")
				if rawValue != "" {
					if label == "" {
						label = rawValue
					}
					result[label] = rawValue
				}
			}
		}
	case []string:
		for _, entry := range value {
			result[entry] = entry
		}
	}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(values[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

// NormalizeUAPhone validates and formats a Ukrainian phone number for CASL.
func NormalizeUAPhone(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	digits := digitsOnly(value)
	switch {
	case len(digits) == 12 && strings.HasPrefix(digits, "380"):
		value = "+" + digits
	case len(digits) == 11 && strings.HasPrefix(digits, "80"):
		value = "+3" + digits
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		value = "+38" + digits
	case len(digits) == 9:
		value = "+380" + digits
	}
	digits = digitsOnly(value)
	if len(digits) != 12 || !strings.HasPrefix(digits, "380") {
		return "", fmt.Errorf("має бути українським номером у міжнародному форматі")
	}
	return fmt.Sprintf("+%s (%s) %s-%s-%s", digits[:2], digits[2:5], digits[5:8], digits[8:10], digits[10:12]), nil
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
