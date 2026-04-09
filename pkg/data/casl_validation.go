package data

import (
	"errors"
	"fmt"
	"strings"
)

func validateCASLPults(items []caslPult) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLPult(item, idx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLPult(item caslPult, idx int) error {
	if strings.TrimSpace(item.PultID) == "" {
		return fmt.Errorf("casl read_pult: data[%d].pult_id is required", idx)
	}
	return nil
}

func validateCASLUsers(items []caslUser) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLUser(item, fmt.Sprintf("data[%d]", idx)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLUser(item caslUser, path string) error {
	return validateCASLUserInScope(item, path, "casl read_user")
}

func validateCASLUserInScope(item caslUser, path string, scope string) error {
	errs := make([]error, 0, len(item.PhoneNumbers)+1)
	if strings.TrimSpace(item.UserID) == "" {
		errs = append(errs, fmt.Errorf("%s: %s.user_id is required", scope, path))
	}
	for idx, phone := range item.PhoneNumbers {
		if phone.Active && strings.TrimSpace(phone.Number) == "" {
			errs = append(errs, fmt.Errorf("%s: %s.phone_numbers[%d].number is required for active phone", scope, path, idx))
		}
	}
	return errors.Join(errs...)
}

func validateCASLDevices(items []caslDevice) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLDevice(item, fmt.Sprintf("data[%d]", idx), "casl read_device"); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLDevice(item caslDevice, path string, scope string) error {
	errs := make([]error, 0, len(item.Lines)+2)
	if strings.TrimSpace(item.DeviceID.String()) == "" {
		errs = append(errs, fmt.Errorf("%s: %s.device_id is required", scope, path))
	}
	if item.Number.Int64() <= 0 {
		errs = append(errs, fmt.Errorf("%s: %s.number must be > 0", scope, path))
	}
	for idx, line := range item.Lines {
		if line.ID.Int64() <= 0 && line.Number.Int64() <= 0 {
			errs = append(errs, fmt.Errorf("%s: %s.lines[%d] must contain line id or number", scope, path, idx))
		}
	}
	return errors.Join(errs...)
}

func validateCASLRooms(items []caslRoom, path string, scope string) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLRoom(item, fmt.Sprintf("%s.rooms[%d]", path, idx), scope); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLRoom(item caslRoom, path string, scope string) error {
	errs := make([]error, 0, len(item.Users)+1)
	if strings.TrimSpace(item.RoomID) == "" {
		errs = append(errs, fmt.Errorf("%s: %s.room_id is required", scope, path))
	}
	for idx, user := range item.Users {
		if err := validateCASLUserInScope(user, fmt.Sprintf("%s.users[%d]", path, idx), scope); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLGuardObjects(items []caslGrdObject) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLGuardObject(item, fmt.Sprintf("data[%d]", idx), "casl read_grd_object"); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLGuardObject(item caslGrdObject, path string, scope string) error {
	errs := make([]error, 0, len(item.Rooms)+3)
	if strings.TrimSpace(item.ObjID) == "" {
		errs = append(errs, fmt.Errorf("%s: %s.obj_id is required", scope, path))
	}
	if item.DeviceID.Int64() <= 0 && item.DeviceNumber.Int64() <= 0 {
		errs = append(errs, fmt.Errorf("%s: %s must contain device_id or device_number", scope, path))
	}
	if err := validateCASLRooms(item.Rooms, path, scope); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateCASLConnections(items []caslConnectionRecord) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLConnection(item, idx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLConnection(item caslConnectionRecord, idx int) error {
	errs := make([]error, 0, 2)
	if err := validateCASLGuardObject(item.GuardedObject, fmt.Sprintf("data[%d].guardedObject", idx), "casl read_connections"); err != nil {
		errs = append(errs, err)
	}
	if strings.TrimSpace(item.Device.DeviceID.String()) != "" || item.Device.Number.Int64() > 0 {
		if err := validateCASLDevice(item.Device, fmt.Sprintf("data[%d].device", idx), "casl read_connections"); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLObjectEditorResponse(resp caslObjectEditorFullResponse) error {
	errs := make([]error, 0, len(resp.Rooms)+3)
	if strings.TrimSpace(resp.Name) == "" {
		errs = append(errs, errors.New("casl get_grd_object_full: name is required"))
	}
	if strings.TrimSpace(resp.PultID.String()) == "" {
		errs = append(errs, errors.New("casl get_grd_object_full: pult_id is required"))
	}
	if resp.Device.Number.Int64() <= 0 && strings.TrimSpace(firstCASLString(resp.Device.ID.String(), resp.Device.DeviceID.String())) == "" {
		errs = append(errs, errors.New("casl get_grd_object_full: device identity is required"))
	}
	for idx, room := range resp.Rooms {
		if err := validateCASLObjectEditorRoom(room, idx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLObjectEditorRoom(room caslObjectEditorRoom, idx int) error {
	scope := "casl get_grd_object_full"
	path := fmt.Sprintf("rooms[%d]", idx)
	errs := make([]error, 0, len(room.Users)+len(room.Lines)+1)

	if strings.TrimSpace(room.RoomID.String()) == "" {
		errs = append(errs, fmt.Errorf("%s: %s.room_id is required", scope, path))
	}
	for userIdx, user := range room.Users {
		if strings.TrimSpace(user.UserID.String()) == "" {
			errs = append(errs, fmt.Errorf("%s: %s.users[%d].user_id is required", scope, path, userIdx))
		}
	}
	for lineNumber, line := range room.Lines {
		linePath := fmt.Sprintf("%s.lines[%q]", path, lineNumber)
		if parseCASLID(lineNumber) <= 0 {
			errs = append(errs, fmt.Errorf("%s: %s key must be a positive line number", scope, linePath))
		}
		if strings.TrimSpace(line.AdapterType.String()) == "" {
			errs = append(errs, fmt.Errorf("%s: %s.adapter_type is required", scope, linePath))
		}
		if line.GroupNumber.Int64() <= 0 {
			errs = append(errs, fmt.Errorf("%s: %s.group_number must be > 0", scope, linePath))
		}
	}

	return errors.Join(errs...)
}

func validateCASLObjectEvents(items []caslObjectEvent, scope string) error {
	errs := make([]error, 0, len(items))
	for idx, item := range items {
		if err := validateCASLObjectEvent(item, idx, scope); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateCASLObjectEvent(item caslObjectEvent, idx int, scope string) error {
	errs := make([]error, 0, 2)
	if item.Time.Int64() <= 0 {
		errs = append(errs, fmt.Errorf("%s: data[%d].time must be > 0", scope, idx))
	}

	hasSemanticMarker := strings.TrimSpace(item.Type) != "" ||
		strings.TrimSpace(item.Code.String()) != "" ||
		strings.TrimSpace(item.EventCode.String()) != "" ||
		strings.TrimSpace(item.Action.String()) != "" ||
		strings.TrimSpace(item.DictName.String()) != "" ||
		strings.TrimSpace(item.TypeEvent.String()) != "" ||
		strings.TrimSpace(item.UserAction.String()) != "" ||
		strings.TrimSpace(item.MgrAction.String()) != "" ||
		strings.TrimSpace(item.PPKAction.String()) != "" ||
		strings.TrimSpace(item.ContactID.String()) != "" ||
		item.PPKNum.Int64() > 0 ||
		strings.TrimSpace(item.DeviceID.String()) != "" ||
		strings.TrimSpace(item.ObjID.String()) != ""
	if !hasSemanticMarker {
		errs = append(errs, fmt.Errorf("%s: data[%d] has no event identity fields", scope, idx))
	}

	return errors.Join(errs...)
}

func validateCASLRealtimeObjectEvent(item CASLObjectEvent, scope string) error {
	errs := make([]error, 0, 2)
	if item.Time <= 0 {
		errs = append(errs, fmt.Errorf("%s: time must be > 0", scope))
	}

	hasSemanticMarker := strings.TrimSpace(item.Type) != "" ||
		strings.TrimSpace(item.Code) != "" ||
		strings.TrimSpace(item.Action) != "" ||
		strings.TrimSpace(item.ContactID) != "" ||
		item.Number > 0 ||
		item.PPKNum > 0 ||
		strings.TrimSpace(item.DeviceID) != "" ||
		strings.TrimSpace(item.ObjID) != ""
	if !hasSemanticMarker {
		errs = append(errs, fmt.Errorf("%s: event payload has no identity fields", scope))
	}

	return errors.Join(errs...)
}

func validateCASLDeviceState(state caslDeviceState, scope string) error {
	hasSignal := state.Power.Int64() != 0 ||
		state.Accum.Int64() != 0 ||
		state.Door.Int64() != 0 ||
		state.Online.Int64() != 0 ||
		state.LastPingDate.Int64() != 0 ||
		state.Lines != nil ||
		state.Groups != nil ||
		state.Adapters != nil
	if !hasSignal {
		return fmt.Errorf("%s: state payload is empty", scope)
	}
	return nil
}

func validateCASLStatsAlarmsData(stats caslStatsAlarmsData, scope string) error {
	errs := make([]error, 0, 2)
	if strings.TrimSpace(stats.DeviceID) == "" {
		errs = append(errs, fmt.Errorf("%s: data.device_id is required", scope))
	}
	if strings.TrimSpace(stats.ObjectID) == "" {
		errs = append(errs, fmt.Errorf("%s: data.obj_id is required", scope))
	}
	return errors.Join(errs...)
}
