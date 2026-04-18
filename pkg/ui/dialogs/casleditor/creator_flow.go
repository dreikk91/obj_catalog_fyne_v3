package casleditor

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func (vm *EditorViewModel) DraftObject(data ObjectUpdateData) error {
	if err := vm.ValidateObjectForm(data); err != nil {
		return err
	}
	vm.Snapshot.Object.Name = strings.TrimSpace(data.Name)
	vm.Snapshot.Object.Address = strings.TrimSpace(data.Address)
	vm.Snapshot.Object.Lat = strings.TrimSpace(data.Lat)
	vm.Snapshot.Object.Long = strings.TrimSpace(data.Long)
	vm.Snapshot.Object.Description = strings.TrimSpace(data.Description)
	vm.Snapshot.Object.Contract = strings.TrimSpace(data.Contract)
	vm.Snapshot.Object.ManagerID = strings.TrimSpace(data.ManagerID)
	vm.Snapshot.Object.Note = strings.TrimSpace(data.Note)
	vm.Snapshot.Object.StartDate = data.StartDate
	vm.Snapshot.Object.ObjectStatus = strings.TrimSpace(data.Status)
	vm.Snapshot.Object.ObjectType = strings.TrimSpace(data.ObjectType)
	vm.Snapshot.Object.IDRequest = strings.TrimSpace(data.IDRequest)
	vm.Snapshot.Object.ReactingPultID = strings.TrimSpace(data.ReactingPultID)
	vm.Snapshot.Object.GeoZoneID = data.GeoZoneID
	vm.Snapshot.Object.BusinessCoeff = data.BusinessCoeff
	vm.emitDataChanged()
	return nil
}

func (vm *EditorViewModel) DraftDevice(data DeviceUpdateData) error {
	if err := vm.ValidateDeviceDraft(data); err != nil {
		return err
	}
	vm.Snapshot.Object.Device.Number = data.Number
	vm.Snapshot.Object.Device.Name = strings.TrimSpace(data.Name)
	vm.Snapshot.Object.Device.Type = strings.TrimSpace(data.Type)
	vm.Snapshot.Object.Device.Timeout = data.Timeout
	vm.Snapshot.Object.Device.SIM1 = strings.TrimSpace(data.SIM1)
	vm.Snapshot.Object.Device.SIM2 = strings.TrimSpace(data.SIM2)
	vm.Snapshot.Object.Device.TechnicianID = strings.TrimSpace(data.TechnicianID)
	vm.Snapshot.Object.Device.Units = strings.TrimSpace(data.Units)
	vm.Snapshot.Object.Device.Requisites = strings.TrimSpace(data.Requisites)
	vm.Snapshot.Object.Device.ChangeDate = data.ChangeDate
	vm.Snapshot.Object.Device.ReglamentDate = data.ReglamentDate
	vm.Snapshot.Object.Device.LicenceKey = strings.TrimSpace(data.LicenceKey)
	vm.Snapshot.Object.Device.PasswRemote = strings.TrimSpace(data.RemotePass)
	vm.emitDataChanged()
	return nil
}

func (vm *EditorViewModel) DraftSelectedRoom(data RoomUpdateData) error {
	if vm.RoomSelected < 0 || vm.RoomSelected >= len(vm.Snapshot.Object.Rooms) {
		return fmt.Errorf("оберіть приміщення")
	}
	if err := ValidateRoomDraftData(data); err != nil {
		return err
	}
	name := strings.TrimSpace(data.Name)
	vm.Snapshot.Object.Rooms[vm.RoomSelected].Name = name
	vm.Snapshot.Object.Rooms[vm.RoomSelected].Description = strings.TrimSpace(data.Description)
	vm.Snapshot.Object.Rooms[vm.RoomSelected].RTSP = strings.TrimSpace(data.RTSP)
	vm.initDictionaries()
	vm.emitDataChanged()
	return nil
}

func ValidateRoomDraftData(data RoomUpdateData) error {
	if strings.TrimSpace(data.Name) == "" {
		return fmt.Errorf("вкажіть назву приміщення")
	}
	if strings.TrimSpace(data.Description) == "" {
		return fmt.Errorf("вкажіть опис приміщення")
	}
	return nil
}

func (vm *EditorViewModel) ValidateDeviceDraft(data DeviceUpdateData) error {
	if data.Number <= 0 {
		return fmt.Errorf("вкажіть номер приладу")
	}
	if strings.TrimSpace(data.Name) == "" {
		return fmt.Errorf("вкажіть назву приладу")
	}
	if strings.TrimSpace(data.Type) == "" {
		return fmt.Errorf("вкажіть тип приладу")
	}
	if data.Timeout <= 0 {
		return fmt.Errorf("вкажіть timeout приладу")
	}
	if data.SIM1 != "" {
		if _, err := NormalizeCASLEditorSIM(data.SIM1); err != nil {
			return fmt.Errorf("sim1: %w", err)
		}
	}
	if data.SIM2 != "" {
		if _, err := NormalizeCASLEditorSIM(data.SIM2); err != nil {
			return fmt.Errorf("sim2: %w", err)
		}
	}
	return nil
}

func (vm *EditorViewModel) ValidateDraftRooms() error {
	if len(vm.Snapshot.Object.Rooms) == 0 {
		return fmt.Errorf("додайте хоча б одне приміщення")
	}
	seen := make(map[string]struct{}, len(vm.Snapshot.Object.Rooms))
	for _, room := range vm.Snapshot.Object.Rooms {
		if err := ValidateRoomDraftData(RoomUpdateData{
			Name:        room.Name,
			Description: room.Description,
			RTSP:        room.RTSP,
		}); err != nil {
			return err
		}
		name := strings.TrimSpace(room.Name)
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("назви приміщень мають бути унікальними")
		}
		seen[key] = struct{}{}
	}
	for idx, line := range vm.Snapshot.Object.Device.Lines {
		if line.LineNumber <= 0 {
			return fmt.Errorf("зона #%d має некоректний номер", idx+1)
		}
		if err := ValidateCASLLineNumberRange(line.LineNumber); err != nil {
			return fmt.Errorf("зона #%d: %w", idx+1, err)
		}
		if err := ValidateCASLLineDescription(line.Description); err != nil {
			return fmt.Errorf("зона #%d: %w", idx+1, err)
		}
		if err := ValidateCASLLineNumberUnique(vm.Snapshot.Object.Device.Lines, line.LineNumber, idx); err != nil {
			return err
		}
	}
	for idx := range vm.RoomUsersLocal {
		if err := vm.ValidateRoomUserHozNum(idx, vm.RoomUsersLocal[idx].HozNum); err != nil {
			return err
		}
	}
	return nil
}

func (vm *EditorViewModel) CommitCreationWizard() {
	if !vm.creating {
		return
	}
	vm.setStatus("Створення об'єкта CASL...")

	draft := vm.Snapshot
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		objID, objectID, err := vm.createDraftObject(ctx, draft.Object)
		if err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		deviceID, err := vm.createDraftDevice(ctx, draft.Object.Device)
		if err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		if err := vm.createDraftLines(ctx, deviceID, draft.Object.Device.Lines); err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		if err := vm.createDraftObjectImages(ctx, objID, draft.Object.Images); err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		if err := vm.createDraftRooms(ctx, objID, draft.Object.Rooms); err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		roomMap, lineMap, err := vm.reloadCreatedMappings(ctx, objectID)
		if err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		if err := vm.bindDraftRooms(ctx, objID, deviceID, draft.Object.Rooms, roomMap, lineMap); err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка створення")
				vm.showError(err)
			})
			return
		}

		fyne.Do(func() {
			vm.creating = false
			vm.objectID = objectID
			vm.setStatus("Об'єкт створено")
			if vm.onChanged != nil {
				vm.onChanged()
			}
			vm.showAlert("CASL", "Майстер успішно створив об'єкт.")
			if vm.win != nil {
				vm.win.Close()
			}
		})
	}()
}

func (vm *EditorViewModel) createDraftObject(ctx context.Context, object contracts.CASLGuardObjectDetails) (string, int64, error) {
	objID, err := vm.provider.CreateCASLObject(ctx, contracts.CASLGuardObjectCreate{
		Name:           strings.TrimSpace(object.Name),
		Address:        strings.TrimSpace(object.Address),
		Long:           strings.TrimSpace(object.Long),
		Lat:            strings.TrimSpace(object.Lat),
		Description:    strings.TrimSpace(object.Description),
		Contract:       strings.TrimSpace(object.Contract),
		ManagerID:      strings.TrimSpace(object.ManagerID),
		Note:           strings.TrimSpace(object.Note),
		StartDate:      object.StartDate,
		Status:         strings.TrimSpace(object.ObjectStatus),
		ObjectType:     strings.TrimSpace(object.ObjectType),
		IDRequest:      strings.TrimSpace(object.IDRequest),
		ReactingPultID: strings.TrimSpace(object.ReactingPultID),
		GeoZoneID:      object.GeoZoneID,
		BusinessCoeff:  object.BusinessCoeff,
	})
	if err != nil {
		return "", 0, err
	}
	parsedID, err := strconv.ParseInt(strings.TrimSpace(objID), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("casl повернув некоректний obj_id %q", objID)
	}
	return strings.TrimSpace(objID), parsedID, nil
}

func (vm *EditorViewModel) createDraftDevice(ctx context.Context, device contracts.CASLDeviceDetails) (string, error) {
	inUse, err := vm.provider.IsCASLDeviceNumberInUse(ctx, device.Number)
	if err != nil {
		return "", err
	}
	if inUse {
		return "", fmt.Errorf("номер приладу %d вже зайнятий", device.Number)
	}
	return vm.provider.CreateCASLDevice(ctx, contracts.CASLDeviceCreate{
		Number:            device.Number,
		Name:              strings.TrimSpace(device.Name),
		DeviceType:        strings.TrimSpace(device.Type),
		Timeout:           device.Timeout,
		SIM1:              strings.TrimSpace(device.SIM1),
		SIM2:              strings.TrimSpace(device.SIM2),
		TechnicianID:      strings.TrimSpace(device.TechnicianID),
		Units:             strings.TrimSpace(device.Units),
		Requisites:        strings.TrimSpace(device.Requisites),
		ChangeDate:        device.ChangeDate,
		ReglamentDate:     device.ReglamentDate,
		MoreAlarmTime:     device.MoreAlarmTime,
		IgnoringAlarmTime: device.IgnoringAlarmTime,
		LicenceKey:        strings.TrimSpace(device.LicenceKey),
		PasswRemote:       strings.TrimSpace(device.PasswRemote),
	})
}

func (vm *EditorViewModel) createDraftLines(ctx context.Context, deviceID string, lines []contracts.CASLDeviceLineDetails) error {
	for _, line := range lines {
		if err := vm.provider.CreateCASLDeviceLine(ctx, contracts.CASLDeviceLineMutation{
			DeviceID:      deviceID,
			LineNumber:    line.LineNumber,
			GroupNumber:   line.GroupNumber,
			AdapterType:   strings.TrimSpace(line.AdapterType),
			AdapterNumber: line.AdapterNumber,
			Description:   strings.TrimSpace(line.Description),
			LineType:      strings.TrimSpace(line.LineType),
			IsBlocked:     line.IsBlocked,
		}); err != nil {
			return fmt.Errorf("не вдалося створити зону #%d: %w", line.LineNumber, err)
		}
	}
	return nil
}

func (vm *EditorViewModel) createDraftObjectImages(ctx context.Context, objID string, images []string) error {
	for _, raw := range images {
		imageType, encoded, ok := draftImageUpload(raw)
		if !ok {
			continue
		}
		if err := vm.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
			ObjID:     objID,
			ImageType: imageType,
			ImageData: encoded,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (vm *EditorViewModel) createDraftRooms(ctx context.Context, objID string, rooms []contracts.CASLRoomDetails) error {
	for _, room := range rooms {
		if err := vm.provider.CreateCASLRoom(ctx, contracts.CASLRoomCreate{
			ObjID:       objID,
			Name:        strings.TrimSpace(room.Name),
			Description: strings.TrimSpace(room.Description),
			RTSP:        strings.TrimSpace(room.RTSP),
		}); err != nil {
			return fmt.Errorf("не вдалося створити приміщення %q: %w", room.Name, err)
		}
	}
	return nil
}

func (vm *EditorViewModel) reloadCreatedMappings(ctx context.Context, objectID int64) (map[string]string, map[int]string, error) {
	snapshot, err := vm.provider.GetCASLObjectEditorSnapshot(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}
	roomMap := make(map[string]string, len(snapshot.Object.Rooms))
	for _, room := range snapshot.Object.Rooms {
		name := strings.TrimSpace(room.Name)
		if name == "" {
			continue
		}
		roomMap[strings.ToLower(name)] = room.RoomID
	}
	lineMap := make(map[int]string, len(snapshot.Object.Device.Lines))
	for _, line := range snapshot.Object.Device.Lines {
		lineMap[line.LineNumber] = snapshot.Object.Device.DeviceID
	}
	return roomMap, lineMap, nil
}

func (vm *EditorViewModel) bindDraftRooms(ctx context.Context, objID string, deviceID string, rooms []contracts.CASLRoomDetails, roomMap map[string]string, lineMap map[int]string) error {
	for _, room := range rooms {
		roomID := roomMap[strings.ToLower(strings.TrimSpace(room.Name))]
		if roomID == "" {
			return fmt.Errorf("casl не повернув room_id для приміщення %q", room.Name)
		}
		for _, user := range room.Users {
			if err := vm.provider.AddCASLUserToRoom(ctx, contracts.CASLAddUserToRoomRequest{
				ObjID:    objID,
				RoomID:   roomID,
				UserID:   user.UserID,
				Priority: user.Priority,
				HozNum:   strings.TrimSpace(user.HozNum),
			}); err != nil {
				return fmt.Errorf("не вдалося додати користувача до %q: %w", room.Name, err)
			}
		}
		for _, image := range room.Images {
			imageType, encoded, ok := draftImageUpload(image)
			if !ok {
				continue
			}
			if err := vm.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
				ObjID:     objID,
				RoomID:    roomID,
				ImageType: imageType,
				ImageData: encoded,
			}); err != nil {
				return fmt.Errorf("не вдалося завантажити фото приміщення %q: %w", room.Name, err)
			}
		}
		for _, line := range room.Lines {
			if _, ok := lineMap[line.LineNumber]; !ok {
				return fmt.Errorf("зона #%d не була створена", line.LineNumber)
			}
			if err := vm.provider.AddCASLLineToRoom(ctx, contracts.CASLLineToRoomBinding{
				ObjID:      objID,
				DeviceID:   deviceID,
				LineNumber: line.LineNumber,
				RoomID:     roomID,
			}); err != nil {
				return fmt.Errorf("не вдалося прив'язати зону #%d до %q: %w", line.LineNumber, room.Name, err)
			}
		}
	}
	return nil
}

func draftImageUpload(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return "", "", false
	}
	commaIdx := strings.Index(raw, ",")
	if commaIdx < 0 {
		return "", "", false
	}
	header := strings.ToLower(strings.TrimSpace(raw[:commaIdx]))
	payload := strings.TrimSpace(raw[commaIdx+1:])
	imageType := "jpg"
	switch {
	case strings.Contains(header, "image/png"):
		imageType = "png"
	case strings.Contains(header, "image/webp"):
		imageType = "webp"
	case strings.Contains(header, "image/gif"):
		imageType = "gif"
	case strings.Contains(header, "image/bmp"):
		imageType = "bmp"
	case strings.Contains(header, "image/svg"):
		imageType = "svg"
	}
	return imageType, payload, true
}

func (vm *EditorViewModel) ReplaceObjectDraftImages(images []string) {
	vm.Snapshot.Object.Images = slices.Clone(images)
	vm.emitDataChanged()
}

func (vm *EditorViewModel) ReplaceSelectedRoomDraftImages(images []string) {
	if vm.RoomSelected < 0 || vm.RoomSelected >= len(vm.Snapshot.Object.Rooms) {
		return
	}
	vm.Snapshot.Object.Rooms[vm.RoomSelected].Images = slices.Clone(images)
	vm.emitDataChanged()
}
