package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
)

type caslObjectEditorFullResponse struct {
	Status         string                     `json:"status"`
	Error          string                     `json:"error"`
	Name           string                     `json:"name"`
	Address        string                     `json:"address"`
	Lat            string                     `json:"lat"`
	Long           string                     `json:"long"`
	Description    string                     `json:"description"`
	PultID         caslText                   `json:"pult_id"`
	ReactingPultID caslText                   `json:"reacting_pult_id"`
	Contract       string                     `json:"contract"`
	UserID         caslText                   `json:"user_id"`
	Note           string                     `json:"note"`
	StartDate      caslInt64                  `json:"start_date"`
	ObjectType     string                     `json:"object_type"`
	IDRequest      string                     `json:"id_request"`
	GeoZoneID      caslInt64                  `json:"geo_zone_id"`
	BusinessCoeff  caslNullableFloat64        `json:"bissnes_coeff"`
	Rooms          []caslObjectEditorRoom     `json:"rooms"`
	Device         caslObjectEditorDeviceStub `json:"device"`
	Devices        json.RawMessage            `json:"devices"`
	ObjectStatus   caslText                   `json:"obj_status"`
	Images         []caslText                 `json:"images"`
}

type caslObjectEditorRoom struct {
	RoomID      caslText                            `json:"room_id"`
	Name        string                              `json:"name"`
	Description string                              `json:"description"`
	Images      []caslText                          `json:"images"`
	RTSP        string                              `json:"rtsp"`
	Users       []caslObjectEditorRoomUser          `json:"users"`
	Lines       map[string]caslObjectEditorRoomLine `json:"lines"`
}

type caslObjectEditorRoomUser struct {
	UserID   caslText  `json:"user_id"`
	Priority caslInt64 `json:"priority"`
	HozNum   caslText  `json:"hoz_num"`
}

type caslObjectEditorRoomLine struct {
	AdapterType   caslText  `json:"adapter_type"`
	GroupNumber   caslInt64 `json:"group_number"`
	AdapterNumber caslInt64 `json:"adapter_number"`
}

type caslObjectEditorDeviceStub struct {
	ID                caslText                 `json:"id"`
	DeviceID          caslText                 `json:"device_id"`
	ObjID             caslText                 `json:"obj_id"`
	Number            caslInt64                `json:"number"`
	Name              caslText                 `json:"name"`
	Type              caslText                 `json:"type"`
	DeviceType        caslText                 `json:"device_type"`
	Timeout           caslInt64                `json:"timeout"`
	SIM1              caslText                 `json:"sim1"`
	SIM2              caslText                 `json:"sim2"`
	TechnicianID      caslText                 `json:"technician_id"`
	Technician        caslObjectEditorUserStub `json:"technician"`
	Units             caslText                 `json:"units"`
	Requisites        caslText                 `json:"requisites"`
	ChangeDate        caslInt64                `json:"change_date"`
	ReglamentDate     caslInt64                `json:"reglament_date"`
	MoreAlarmTime     []any                    `json:"moreAlarmTime"`
	IgnoringAlarmTime []any                    `json:"ignoringAlarmTime"`
	LicenceKey        caslText                 `json:"licence_key"`
	PasswRemote       caslText                 `json:"passw_remote"`
	LastPingDate      caslInt64                `json:"lastPingDate"`
	Lines             json.RawMessage          `json:"lines"`
}

type caslObjectEditorUserStub struct {
	UserID caslText `json:"user_id"`
}

func (p *CASLCloudProvider) GetCASLObjectEditorSnapshot(ctx context.Context, objectID int64) (contracts.CASLObjectEditorSnapshot, error) {
	users, pults, dictionary, err := p.loadCASLObjectEditorReferences(ctx)
	if err != nil {
		return contracts.CASLObjectEditorSnapshot{}, err
	}

	if objectID <= 0 {
		return contracts.CASLObjectEditorSnapshot{
			Users:      users,
			Pults:      pults,
			Dictionary: dictionary,
		}, nil
	}

	objID, err := p.resolveCASLEditorObjectID(ctx, objectID)
	if err != nil {
		return contracts.CASLObjectEditorSnapshot{}, err
	}
	fullObject, err := p.getCASLObjectFull(ctx, objID)
	if err != nil {
		return contracts.CASLObjectEditorSnapshot{}, err
	}

	deviceRaw, _ := p.findCASLDeviceRaw(ctx, fullObject.Device.DeviceID, fullObject.ObjID, fullObject.Device.Number)
	if len(deviceRaw) > 0 {
		fullObject.Device = overlayCASLDeviceDetails(fullObject.Device, mapCASLDeviceDetails(deviceRaw))
	}
	assignRoomIDsToLines(&fullObject)

	return contracts.CASLObjectEditorSnapshot{
		Object:     fullObject,
		Users:      users,
		Pults:      pults,
		Dictionary: dictionary,
	}, nil
}

func (p *CASLCloudProvider) CreateCASLObject(ctx context.Context, create contracts.CASLGuardObjectCreate) (string, error) {
	payload := map[string]any{
		"type":             "create_grd_object",
		"name":             strings.TrimSpace(create.Name),
		"address":          strings.TrimSpace(create.Address),
		"long":             strings.TrimSpace(create.Long),
		"lat":              strings.TrimSpace(create.Lat),
		"description":      strings.TrimSpace(create.Description),
		"contract":         strings.TrimSpace(create.Contract),
		"manager_id":       strings.TrimSpace(create.ManagerID),
		"note":             strings.TrimSpace(create.Note),
		"start_date":       create.StartDate,
		"status":           strings.TrimSpace(create.Status),
		"object_type":      strings.TrimSpace(create.ObjectType),
		"id_request":       strings.TrimSpace(create.IDRequest),
		"reacting_pult_id": strings.TrimSpace(create.ReactingPultID),
		"geo_zone_id":      create.GeoZoneID,
		"bissnes_coeff":    create.BusinessCoeff,
	}
	response, err := p.ExecuteCASLCommand(ctx, payload, true)
	if err != nil {
		return "", err
	}
	p.invalidateCASLEditorCaches()
	return strings.TrimSpace(asString(response["obj_id"])), nil
}

func (p *CASLCloudProvider) UpdateCASLObject(ctx context.Context, update contracts.CASLGuardObjectUpdate) error {
	payload := map[string]any{
		"type":             "update_grd_object",
		"obj_id":           strings.TrimSpace(update.ObjID),
		"name":             strings.TrimSpace(update.Name),
		"address":          strings.TrimSpace(update.Address),
		"long":             strings.TrimSpace(update.Long),
		"lat":              strings.TrimSpace(update.Lat),
		"description":      strings.TrimSpace(update.Description),
		"contract":         strings.TrimSpace(update.Contract),
		"manager_id":       strings.TrimSpace(update.ManagerID),
		"note":             strings.TrimSpace(update.Note),
		"start_date":       update.StartDate,
		"status":           strings.TrimSpace(update.Status),
		"object_type":      strings.TrimSpace(update.ObjectType),
		"id_request":       strings.TrimSpace(update.IDRequest),
		"reacting_pult_id": strings.TrimSpace(update.ReactingPultID),
		"geo_zone_id":      update.GeoZoneID,
		"bissnes_coeff":    update.BusinessCoeff,
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) UpdateCASLRoom(ctx context.Context, update contracts.CASLRoomUpdate) error {
	payload := map[string]any{
		"type":        "update_grd_room",
		"obj_id":      strings.TrimSpace(update.ObjID),
		"room_id":     strings.TrimSpace(update.RoomID),
		"name":        strings.TrimSpace(update.Name),
		"description": strings.TrimSpace(update.Description),
		"rtsp":        strings.TrimSpace(update.RTSP),
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) CreateCASLRoom(ctx context.Context, create contracts.CASLRoomCreate) error {
	payload := map[string]any{
		"type":        "create_grd_room",
		"obj_id":      strings.TrimSpace(create.ObjID),
		"name":        strings.TrimSpace(create.Name),
		"description": strings.TrimSpace(create.Description),
		"rtsp":        strings.TrimSpace(create.RTSP),
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) ReadCASLDeviceNumbers(ctx context.Context) ([]int64, error) {
	response, err := p.ExecuteCASLCommand(ctx, map[string]any{"type": "read_devices_numbers"}, true)
	if err != nil {
		return nil, err
	}
	rawItems, _ := response["data"].([]any)
	numbers := make([]int64, 0, len(rawItems))
	for _, item := range rawItems {
		value := int64(parseCASLAnyInt(item))
		if value > 0 {
			numbers = append(numbers, value)
		}
	}
	sort.SliceStable(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })
	return numbers, nil
}

func (p *CASLCloudProvider) IsCASLDeviceNumberInUse(ctx context.Context, deviceNumber int64) (bool, error) {
	response, err := p.ExecuteCASLCommand(ctx, map[string]any{
		"type":          "is_device_number_in_use",
		"device_number": deviceNumber,
	}, true)
	if err != nil {
		return false, err
	}
	raw := response["data"]
	if value, ok := raw.(bool); ok {
		return value, nil
	}
	switch strings.ToLower(strings.TrimSpace(asString(raw))) {
	case "1", "true", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (p *CASLCloudProvider) CreateCASLDevice(ctx context.Context, create contracts.CASLDeviceCreate) (string, error) {
	payload := map[string]any{
		"type":              "create_device",
		"number":            create.Number,
		"name":              strings.TrimSpace(create.Name),
		"device_type":       strings.TrimSpace(create.DeviceType),
		"timeout":           create.Timeout,
		"sim1":              strings.TrimSpace(create.SIM1),
		"sim2":              strings.TrimSpace(create.SIM2),
		"technician_id":     strings.TrimSpace(create.TechnicianID),
		"units":             strings.TrimSpace(create.Units),
		"requisites":        strings.TrimSpace(create.Requisites),
		"change_date":       zeroToEmpty(create.ChangeDate),
		"reglament_date":    zeroToEmpty(create.ReglamentDate),
		"moreAlarmTime":     normalizeCASLAnySlice(create.MoreAlarmTime),
		"ignoringAlarmTime": normalizeCASLAnySlice(create.IgnoringAlarmTime),
		"licence_key":       strings.TrimSpace(create.LicenceKey),
		"passw_remote":      strings.TrimSpace(create.PasswRemote),
	}
	response, err := p.ExecuteCASLCommand(ctx, payload, true)
	if err != nil {
		return "", err
	}

	deviceID := strings.TrimSpace(asString(response["device_id"]))
	if _, err := p.ExecuteCASLCommand(ctx, map[string]any{
		"type":          "created_new_device",
		"device_number": strconv.FormatInt(create.Number, 10),
	}, true); err != nil {
		return "", err
	}

	p.invalidateCASLEditorCaches()
	return deviceID, nil
}

func (p *CASLCloudProvider) UpdateCASLDevice(ctx context.Context, update contracts.CASLDeviceUpdate) error {
	payload := map[string]any{
		"type":              "update_device",
		"device_id":         strings.TrimSpace(update.DeviceID),
		"number":            update.Number,
		"name":              strings.TrimSpace(update.Name),
		"device_type":       strings.TrimSpace(update.DeviceType),
		"timeout":           update.Timeout,
		"sim1":              strings.TrimSpace(update.SIM1),
		"sim2":              strings.TrimSpace(update.SIM2),
		"technician_id":     strings.TrimSpace(update.TechnicianID),
		"units":             strings.TrimSpace(update.Units),
		"requisites":        strings.TrimSpace(update.Requisites),
		"change_date":       update.ChangeDate,
		"reglament_date":    update.ReglamentDate,
		"moreAlarmTime":     normalizeCASLAnySlice(update.MoreAlarmTime),
		"ignoringAlarmTime": normalizeCASLAnySlice(update.IgnoringAlarmTime),
		"licence_key":       strings.TrimSpace(update.LicenceKey),
		"passw_remote":      strings.TrimSpace(update.PasswRemote),
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) UpdateCASLDeviceLine(ctx context.Context, update contracts.CASLDeviceLineMutation) error {
	payload := caslDeviceLinePayload("update_device_line", update)
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) CreateCASLDeviceLine(ctx context.Context, create contracts.CASLDeviceLineMutation) error {
	payload := caslDeviceLinePayload("create_device_line", create)
	payload["line_id"] = nil
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) AddCASLLineToRoom(ctx context.Context, binding contracts.CASLLineToRoomBinding) error {
	payload := map[string]any{
		"type":        "add_line_to_room",
		"obj_id":      strings.TrimSpace(binding.ObjID),
		"device_id":   strings.TrimSpace(binding.DeviceID),
		"line_number": binding.LineNumber,
		"room_id":     strings.TrimSpace(binding.RoomID),
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) AddCASLUserToRoom(ctx context.Context, request contracts.CASLAddUserToRoomRequest) error {
	payload := map[string]any{
		"type":     "add_user_to_room",
		"obj_id":   strings.TrimSpace(request.ObjID),
		"room_id":  strings.TrimSpace(request.RoomID),
		"user_id":  strings.TrimSpace(request.UserID),
		"priority": request.Priority,
		"hoz_num":  caslNullableString(request.HozNum),
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) RemoveCASLUserFromRoom(ctx context.Context, request contracts.CASLRemoveUserFromRoomRequest) error {
	payload := map[string]any{
		"type":    "remove_user_from_room",
		"obj_id":  strings.TrimSpace(request.ObjID),
		"room_id": strings.TrimSpace(request.RoomID),
		"user_id": strings.TrimSpace(request.UserID),
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) UpdateCASLRoomUserPriorities(ctx context.Context, objectID int64, items []contracts.CASLRoomUserPriority) error {
	objID, err := p.resolveCASLEditorObjectID(ctx, objectID)
	if err != nil {
		return err
	}

	payloadItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payloadItems = append(payloadItems, map[string]any{
			"user_id":  parseCASLAnyInt(item.UserID),
			"room_id":  parseCASLAnyInt(item.RoomID),
			"priority": item.Priority,
			"hoz_num":  caslNullableString(item.HozNum),
		})
	}
	payload := map[string]any{
		"type":                 "upd_priority_user_in_room",
		"obj_id":               objID,
		"usersPriorityByRooms": payloadItems,
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) CreateCASLUser(ctx context.Context, request contracts.CASLUserCreateRequest) (contracts.CASLUserProfile, error) {
	payloadPhones := make([]map[string]any, 0, len(request.PhoneNumbers))
	for _, phone := range request.PhoneNumbers {
		number := strings.TrimSpace(phone.Number)
		if number == "" {
			continue
		}
		payloadPhones = append(payloadPhones, map[string]any{
			"active": phone.Active,
			"number": number,
		})
	}

	payload := map[string]any{
		"type":          "create_user",
		"email":         strings.TrimSpace(request.Email),
		"passw":         strings.TrimSpace(request.Password),
		"last_name":     strings.TrimSpace(request.LastName),
		"first_name":    strings.TrimSpace(request.FirstName),
		"middle_name":   strings.TrimSpace(request.MiddleName),
		"onebox_id":     strings.TrimSpace(request.OneboxID),
		"role":          strings.TrimSpace(request.Role),
		"tag":           strings.TrimSpace(request.Tag),
		"phone_numbers": payloadPhones,
		"device_ids":    append([]string(nil), request.DeviceIDs...),
	}
	response, err := p.ExecuteCASLCommand(ctx, payload, true)
	if err != nil {
		return contracts.CASLUserProfile{}, err
	}

	p.invalidateCASLEditorCaches()

	createdID := strings.TrimSpace(asString(response["user_id"]))
	usersRaw, err := p.ReadUsersRaw(ctx, 0, caslReadLimit)
	if err != nil {
		return contracts.CASLUserProfile{}, err
	}
	for _, raw := range usersRaw {
		user := mapCASLUserProfile(raw)
		if createdID != "" && user.UserID == createdID {
			return user, nil
		}
		if caslUserMatchesCreateRequest(user, request) {
			return user, nil
		}
	}
	if createdID != "" {
		return contracts.CASLUserProfile{UserID: createdID}, nil
	}
	return contracts.CASLUserProfile{}, nil
}

func (p *CASLCloudProvider) CreateCASLImage(ctx context.Context, request contracts.CASLImageCreateRequest) error {
	payload := map[string]any{
		"type":       "create_image",
		"obj_id":     strings.TrimSpace(request.ObjID),
		"image_type": strings.TrimSpace(request.ImageType),
		"image_data": strings.TrimSpace(request.ImageData),
	}
	if roomID := strings.TrimSpace(request.RoomID); roomID != "" {
		payload["room_id"] = roomID
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) DeleteCASLImage(ctx context.Context, request contracts.CASLImageDeleteRequest) error {
	payload := map[string]any{
		"type":     "delete_image",
		"image_id": strings.TrimSpace(request.ImageID),
	}
	if roomID := strings.TrimSpace(request.RoomID); roomID != "" {
		payload["room_id"] = roomID
	}
	if _, err := p.ExecuteCASLCommand(ctx, payload, true); err != nil {
		return err
	}
	p.invalidateCASLEditorCaches()
	return nil
}

func (p *CASLCloudProvider) FetchCASLImagePreview(ctx context.Context, imageID string) ([]byte, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, fmt.Errorf("empty image id")
	}

	token, err := p.ensureToken(ctx)
	if err != nil {
		return nil, err
	}

	path := "/images/" + url.PathEscape(imageID) + "/" + url.PathEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("casl create image preview request: %w", err)
	}
	req.Header.Set("Accept", "image/*")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("casl image preview request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("casl read image preview response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("casl image preview unexpected http status: %d", resp.StatusCode)
	}
	return body, nil
}

func zeroToEmpty(value int64) any {
	if value == 0 {
		return ""
	}
	return value
}

func caslDeviceLinePayload(command string, update contracts.CASLDeviceLineMutation) map[string]any {
	payload := map[string]any{
		"type":           command,
		"device_id":      strings.TrimSpace(update.DeviceID),
		"line_number":    update.LineNumber,
		"group_number":   update.GroupNumber,
		"adapter_type":   strings.TrimSpace(update.AdapterType),
		"adapter_number": update.AdapterNumber,
		"description":    strings.TrimSpace(update.Description),
		"line_type":      strings.TrimSpace(update.LineType),
		"isBlocked":      update.IsBlocked,
	}
	if update.LineID != nil {
		payload["line_id"] = *update.LineID
	}
	return payload
}

func (p *CASLCloudProvider) resolveCASLEditorObjectID(ctx context.Context, objectID int64) (string, error) {
	if objectID <= 0 {
		return "", fmt.Errorf("casl object editor: empty object id")
	}

	if objectID < int64(ids.CASLObjectIDNamespaceStart) || objectID > int64(ids.CASLObjectIDNamespaceEnd) {
		return strconv.FormatInt(objectID, 10), nil
	}

	record, found, err := p.resolveObjectRecord(ctx, int(objectID))
	if err != nil {
		return "", err
	}
	if !found || strings.TrimSpace(record.ObjID) == "" {
		return "", fmt.Errorf("casl object editor: raw obj_id not found for internal id %d", objectID)
	}
	return strings.TrimSpace(record.ObjID), nil
}

func (p *CASLCloudProvider) loadCASLObjectEditorReferences(ctx context.Context) ([]contracts.CASLUserProfile, []contracts.CASLPultRef, map[string]any, error) {
	usersRaw, err := p.readUsers(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	users := make([]contracts.CASLUserProfile, 0, len(usersRaw))
	for _, raw := range usersRaw {
		users = append(users, mapCASLUserProfileFromUser(raw))
	}
	sort.SliceStable(users, func(i, j int) bool {
		return caslUserDisplayName(users[i]) < caslUserDisplayName(users[j])
	})

	pultsRaw, err := p.readPultsPublic(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	pults := make([]contracts.CASLPultRef, 0, len(pultsRaw))
	for _, raw := range pultsRaw {
		pults = append(pults, contracts.CASLPultRef{
			PultID:   strings.TrimSpace(raw.PultID),
			Name:     strings.TrimSpace(raw.Name),
			Nickname: strings.TrimSpace(raw.Nickname),
		})
	}
	sort.SliceStable(pults, func(i, j int) bool {
		return pults[i].PultID < pults[j].PultID
	})

	dictionary, err := p.ReadDictionary(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return users, pults, dictionary, nil
}

func (p *CASLCloudProvider) getCASLObjectFull(ctx context.Context, objID string) (contracts.CASLGuardObjectDetails, error) {
	var resp caslObjectEditorFullResponse
	if err := p.postCommand(ctx, map[string]any{
		"type":   "get_grd_object_full",
		"obj_id": strings.TrimSpace(objID),
	}, &resp, true); err != nil {
		return contracts.CASLGuardObjectDetails{}, err
	}
	if !statusIsOK(resp.Status) {
		return contracts.CASLGuardObjectDetails{}, fmt.Errorf("casl get_grd_object_full status=%q error=%q", resp.Status, resp.Error)
	}
	if err := validateCASLObjectEditorResponse(resp); err != nil {
		return contracts.CASLGuardObjectDetails{}, err
	}

	rooms := make([]contracts.CASLRoomDetails, 0, len(resp.Rooms))
	for _, room := range resp.Rooms {
		roomImages := make([]string, 0, len(room.Images))
		for _, imageValue := range room.Images {
			if value := strings.TrimSpace(imageValue.String()); value != "" && !strings.EqualFold(value, "null") {
				roomImages = append(roomImages, value)
			}
		}

		roomUsers := make([]contracts.CASLRoomUserLink, 0, len(room.Users))
		for _, user := range room.Users {
			roomUsers = append(roomUsers, contracts.CASLRoomUserLink{
				UserID:   strings.TrimSpace(user.UserID.String()),
				Priority: int(user.Priority.Int64()),
				HozNum:   strings.TrimSpace(user.HozNum.String()),
			})
		}

		roomLines := make([]contracts.CASLRoomLineLink, 0, len(room.Lines))
		keys := make([]string, 0, len(room.Lines))
		for key := range room.Lines {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			roomLine := room.Lines[key]
			roomLines = append(roomLines, contracts.CASLRoomLineLink{
				LineNumber:    parseCASLID(key),
				AdapterType:   strings.TrimSpace(roomLine.AdapterType.String()),
				GroupNumber:   int(roomLine.GroupNumber.Int64()),
				AdapterNumber: int(roomLine.AdapterNumber.Int64()),
			})
		}

		rooms = append(rooms, contracts.CASLRoomDetails{
			RoomID:      strings.TrimSpace(room.RoomID.String()),
			Name:        strings.TrimSpace(room.Name),
			Description: strings.TrimSpace(room.Description),
			Images:      roomImages,
			RTSP:        strings.TrimSpace(room.RTSP),
			Users:       roomUsers,
			Lines:       roomLines,
		})
	}

	images := make([]string, 0, len(resp.Images))
	for _, imageID := range resp.Images {
		if value := strings.TrimSpace(imageID.String()); value != "" {
			images = append(images, value)
		}
	}

	deviceDetails := mapCASLObjectEditorDeviceStub(strings.TrimSpace(objID), resp.Device)
	if raw, ok := findCASLDeviceMapInAny(resp.Devices, deviceDetails.DeviceID, deviceDetails.ObjID, deviceDetails.Number); ok {
		deviceDetails = overlayCASLDeviceDetails(deviceDetails, mapCASLDeviceDetails(raw))
	}

	return contracts.CASLGuardObjectDetails{
		ObjID:          strings.TrimSpace(objID),
		Name:           strings.TrimSpace(resp.Name),
		Address:        strings.TrimSpace(resp.Address),
		Lat:            strings.TrimSpace(resp.Lat),
		Long:           strings.TrimSpace(resp.Long),
		Description:    strings.TrimSpace(resp.Description),
		PultID:         strings.TrimSpace(resp.PultID.String()),
		ReactingPultID: strings.TrimSpace(resp.ReactingPultID.String()),
		Contract:       strings.TrimSpace(resp.Contract),
		ManagerID:      strings.TrimSpace(resp.UserID.String()),
		Note:           strings.TrimSpace(resp.Note),
		StartDate:      resp.StartDate.Int64(),
		ObjectType:     strings.TrimSpace(resp.ObjectType),
		IDRequest:      strings.TrimSpace(resp.IDRequest),
		GeoZoneID:      resp.GeoZoneID.Int64(),
		BusinessCoeff:  resp.BusinessCoeff.Float64Ptr(),
		Rooms:          rooms,
		Device:         deviceDetails,
		ObjectStatus:   strings.TrimSpace(resp.ObjectStatus.String()),
		Images:         images,
	}, nil
}

func (p *CASLCloudProvider) findCASLDeviceRaw(ctx context.Context, deviceID string, objID string, number int64) (map[string]any, bool) {
	devicesRaw, err := p.ReadDevices(ctx, 0, caslReadLimit)
	if err != nil {
		return nil, false
	}

	allDevices := make([]map[string]any, 0, len(devicesRaw))
	for _, raw := range devicesRaw {
		allDevices = append(allDevices, extractCASLDeviceMapsFromAny(raw)...)
	}
	return selectCASLDeviceMap(allDevices, deviceID, objID, number)
}

func mapCASLObjectEditorDeviceStub(objID string, stub caslObjectEditorDeviceStub) contracts.CASLDeviceDetails {
	deviceID := strings.TrimSpace(firstCASLString(stub.ID.String(), stub.DeviceID.String()))
	resolvedObjID := strings.TrimSpace(firstCASLString(stub.ObjID.String(), objID))
	technicianID := strings.TrimSpace(firstCASLString(stub.TechnicianID.String(), stub.Technician.UserID.String()))

	return contracts.CASLDeviceDetails{
		DeviceID:          deviceID,
		ObjID:             resolvedObjID,
		Number:            stub.Number.Int64(),
		Name:              strings.TrimSpace(stub.Name.String()),
		Type:              strings.TrimSpace(firstCASLString(stub.Type.String(), stub.DeviceType.String())),
		Timeout:           stub.Timeout.Int64(),
		SIM1:              strings.TrimSpace(stub.SIM1.String()),
		SIM2:              strings.TrimSpace(stub.SIM2.String()),
		TechnicianID:      technicianID,
		Units:             strings.TrimSpace(stub.Units.String()),
		Requisites:        strings.TrimSpace(stub.Requisites.String()),
		ChangeDate:        stub.ChangeDate.Int64(),
		ReglamentDate:     stub.ReglamentDate.Int64(),
		MoreAlarmTime:     normalizeCASLAnySlice(stub.MoreAlarmTime),
		IgnoringAlarmTime: normalizeCASLAnySlice(stub.IgnoringAlarmTime),
		LicenceKey:        strings.TrimSpace(stub.LicenceKey.String()),
		PasswRemote:       strings.TrimSpace(stub.PasswRemote.String()),
		LastPingDate:      stub.LastPingDate.Int64(),
		Lines:             decodeCASLEditorDeviceLines(stub.Lines),
	}
}

func mapCASLDeviceDetails(raw map[string]any) contracts.CASLDeviceDetails {
	lines := decodeCASLEditorDeviceLines(raw["lines"])
	return contracts.CASLDeviceDetails{
		DeviceID:          strings.TrimSpace(asString(raw["device_id"])),
		ObjID:             strings.TrimSpace(asString(raw["obj_id"])),
		Number:            int64(parseCASLAnyInt(raw["number"])),
		Name:              strings.TrimSpace(asString(raw["name"])),
		Type:              strings.TrimSpace(firstCASLString(raw["type"], raw["device_type"])),
		Timeout:           int64(parseCASLAnyInt(raw["timeout"])),
		SIM1:              strings.TrimSpace(asString(raw["sim1"])),
		SIM2:              strings.TrimSpace(asString(raw["sim2"])),
		TechnicianID:      strings.TrimSpace(asString(raw["technician_id"])),
		Units:             strings.TrimSpace(asString(raw["units"])),
		Requisites:        strings.TrimSpace(asString(raw["requisites"])),
		ChangeDate:        parseCASLAnyTime(raw["change_date"]).UnixMilli(),
		ReglamentDate:     parseCASLAnyTime(raw["reglament_date"]).UnixMilli(),
		MoreAlarmTime:     normalizeCASLAnySlice(raw["moreAlarmTime"]),
		IgnoringAlarmTime: normalizeCASLAnySlice(raw["ignoringAlarmTime"]),
		LicenceKey:        strings.TrimSpace(asString(raw["licence_key"])),
		PasswRemote:       strings.TrimSpace(asString(raw["passw_remote"])),
		LastPingDate:      int64(parseCASLAnyInt(raw["lastPingDate"])),
		Lines:             lines,
	}
}

func findCASLDeviceMapInAny(raw any, deviceID string, objID string, number int64) (map[string]any, bool) {
	return selectCASLDeviceMap(extractCASLDeviceMapsFromAny(raw), deviceID, objID, number)
}

func selectCASLDeviceMap(items []map[string]any, deviceID string, objID string, number int64) (map[string]any, bool) {
	deviceID = strings.TrimSpace(deviceID)
	objID = strings.TrimSpace(objID)

	for _, raw := range items {
		if deviceID != "" && strings.TrimSpace(firstCASLString(raw["device_id"], raw["id"])) == deviceID {
			return raw, true
		}
	}
	for _, raw := range items {
		if objID != "" && strings.TrimSpace(asString(raw["obj_id"])) == objID {
			return raw, true
		}
	}
	for _, raw := range items {
		if number > 0 && int64(parseCASLAnyInt(raw["number"])) == number {
			return raw, true
		}
	}
	if len(items) == 0 {
		return nil, false
	}
	return items[0], true
}

func extractCASLDeviceMapsFromAny(raw any) []map[string]any {
	switch typed := raw.(type) {
	case nil:
		return nil
	case json.RawMessage:
		if len(typed) == 0 {
			return nil
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil
		}
		return extractCASLDeviceMapsFromAny(decoded)
	case []byte:
		if len(typed) == 0 {
			return nil
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil
		}
		return extractCASLDeviceMapsFromAny(decoded)
	case []any:
		result := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, extractCASLDeviceMapsFromAny(item)...)
		}
		return result
	case map[string]any:
		if looksLikeCASLDeviceMap(typed) {
			return []map[string]any{typed}
		}
		if nested, ok := typed["devices"]; ok {
			return extractCASLDeviceMapsFromAny(nested)
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		result := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			result = append(result, extractCASLDeviceMapsFromAny(typed[key])...)
		}
		return result
	default:
		return nil
	}
}

func looksLikeCASLDeviceMap(raw map[string]any) bool {
	if len(raw) == 0 {
		return false
	}
	if firstCASLString(raw["room_id"], raw["group_id"], raw["line_id"]) != "" {
		return false
	}

	hasIdentity := strings.TrimSpace(firstCASLString(raw["device_id"], raw["id"], raw["obj_id"])) != "" || parseCASLAnyInt(raw["number"]) > 0
	if !hasIdentity {
		return false
	}

	if parseCASLAnyInt(raw["timeout"]) > 0 {
		return true
	}
	if raw["lines"] != nil {
		return true
	}
	if strings.TrimSpace(firstCASLString(raw["sim1"], raw["sim2"], raw["technician_id"], raw["licence_key"], raw["passw_remote"])) != "" {
		return true
	}
	return strings.TrimSpace(firstCASLString(raw["type"], raw["device_type"])) != ""
}

func overlayCASLDeviceDetails(base contracts.CASLDeviceDetails, overlay contracts.CASLDeviceDetails) contracts.CASLDeviceDetails {
	if strings.TrimSpace(overlay.DeviceID) != "" {
		base.DeviceID = overlay.DeviceID
	}
	if strings.TrimSpace(overlay.ObjID) != "" {
		base.ObjID = overlay.ObjID
	}
	if overlay.Number > 0 {
		base.Number = overlay.Number
	}
	if strings.TrimSpace(overlay.Name) != "" {
		base.Name = overlay.Name
	}
	if strings.TrimSpace(overlay.Type) != "" {
		base.Type = overlay.Type
	}
	if overlay.Timeout > 0 {
		base.Timeout = overlay.Timeout
	}
	if strings.TrimSpace(overlay.SIM1) != "" {
		base.SIM1 = overlay.SIM1
	}
	if strings.TrimSpace(overlay.SIM2) != "" {
		base.SIM2 = overlay.SIM2
	}
	if strings.TrimSpace(overlay.TechnicianID) != "" {
		base.TechnicianID = overlay.TechnicianID
	}
	if strings.TrimSpace(overlay.Units) != "" {
		base.Units = overlay.Units
	}
	if strings.TrimSpace(overlay.Requisites) != "" {
		base.Requisites = overlay.Requisites
	}
	if overlay.ChangeDate > 0 {
		base.ChangeDate = overlay.ChangeDate
	}
	if overlay.ReglamentDate > 0 {
		base.ReglamentDate = overlay.ReglamentDate
	}
	if len(overlay.MoreAlarmTime) > 0 {
		base.MoreAlarmTime = overlay.MoreAlarmTime
	}
	if len(overlay.IgnoringAlarmTime) > 0 {
		base.IgnoringAlarmTime = overlay.IgnoringAlarmTime
	}
	if strings.TrimSpace(overlay.LicenceKey) != "" {
		base.LicenceKey = overlay.LicenceKey
	}
	if strings.TrimSpace(overlay.PasswRemote) != "" {
		base.PasswRemote = overlay.PasswRemote
	}
	if overlay.LastPingDate > 0 {
		base.LastPingDate = overlay.LastPingDate
	}
	if len(overlay.Lines) > 0 {
		base.Lines = overlay.Lines
	}
	return base
}

func assignRoomIDsToLines(object *contracts.CASLGuardObjectDetails) {
	if object == nil || len(object.Device.Lines) == 0 || len(object.Rooms) == 0 {
		return
	}
	lineToRoom := make(map[int]string, len(object.Device.Lines))
	for _, room := range object.Rooms {
		roomID := strings.TrimSpace(room.RoomID)
		if roomID == "" {
			continue
		}
		for _, line := range room.Lines {
			if line.LineNumber > 0 {
				lineToRoom[line.LineNumber] = roomID
			}
		}
	}
	for idx := range object.Device.Lines {
		if roomID := strings.TrimSpace(lineToRoom[object.Device.Lines[idx].LineNumber]); roomID != "" {
			object.Device.Lines[idx].RoomID = roomID
		}
	}
}

func decodeCASLEditorDeviceLines(raw any) []contracts.CASLDeviceLineDetails {
	if raw == nil {
		return nil
	}

	body, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var generic any
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil
	}

	decoded := decodeCASLLinePayloads(generic)
	lines := make([]contracts.CASLDeviceLineDetails, 0, len(decoded))
	for _, item := range decoded {
		lines = append(lines, mapCASLDecodedLineToEditorLine(item))
	}
	return lines
}

func mapCASLUserProfile(raw map[string]any) contracts.CASLUserProfile {
	phones := make([]contracts.CASLPhoneNumber, 0, 2)
	if rawPhones, ok := raw["phone_numbers"].([]any); ok {
		for _, item := range rawPhones {
			phoneMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			phones = append(phones, contracts.CASLPhoneNumber{
				Active: parseCASLAnyInt(phoneMap["active"]) > 0 || strings.EqualFold(strings.TrimSpace(asString(phoneMap["active"])), "true"),
				Number: strings.TrimSpace(asString(phoneMap["number"])),
			})
		}
	}

	return contracts.CASLUserProfile{
		UserID:       strings.TrimSpace(asString(raw["user_id"])),
		Email:        strings.TrimSpace(asString(raw["email"])),
		LastName:     strings.TrimSpace(asString(raw["last_name"])),
		FirstName:    strings.TrimSpace(asString(raw["first_name"])),
		MiddleName:   strings.TrimSpace(asString(raw["middle_name"])),
		Role:         strings.TrimSpace(asString(raw["role"])),
		Tag:          strings.TrimSpace(asString(raw["tag"])),
		PhoneNumbers: phones,
	}
}

func mapCASLUserProfileFromUser(raw caslUser) contracts.CASLUserProfile {
	phones := make([]contracts.CASLPhoneNumber, 0, len(raw.PhoneNumbers))
	for _, item := range raw.PhoneNumbers {
		phones = append(phones, contracts.CASLPhoneNumber{
			Active: item.Active,
			Number: strings.TrimSpace(item.Number),
		})
	}

	return contracts.CASLUserProfile{
		UserID:       strings.TrimSpace(raw.UserID),
		Email:        strings.TrimSpace(raw.Email),
		LastName:     strings.TrimSpace(raw.LastName),
		FirstName:    strings.TrimSpace(raw.FirstName),
		MiddleName:   strings.TrimSpace(raw.MiddleName),
		Role:         strings.TrimSpace(raw.Role),
		Tag:          strings.TrimSpace(raw.Tag.String()),
		PhoneNumbers: phones,
	}
}

func caslUserDisplayName(user contracts.CASLUserProfile) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{user.LastName, user.FirstName, user.MiddleName} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "Користувач #" + strings.TrimSpace(user.UserID)
	}
	return strings.Join(parts, " ")
}

func caslUserMatchesCreateRequest(user contracts.CASLUserProfile, request contracts.CASLUserCreateRequest) bool {
	if strings.TrimSpace(user.LastName) != strings.TrimSpace(request.LastName) {
		return false
	}
	if strings.TrimSpace(user.FirstName) != strings.TrimSpace(request.FirstName) {
		return false
	}
	if strings.TrimSpace(user.MiddleName) != strings.TrimSpace(request.MiddleName) {
		return false
	}
	if strings.TrimSpace(request.Tag) != "" && strings.TrimSpace(user.Tag) != strings.TrimSpace(request.Tag) {
		return false
	}
	return true
}

func firstCASLString(values ...any) string {
	for _, value := range values {
		text := strings.TrimSpace(asString(value))
		if text != "" {
			return text
		}
	}
	return ""
}

func firstCASLAny(values ...any) any {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		return value
	}
	return nil
}

func normalizeCASLAnySlice(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return append([]any(nil), typed...)
	default:
		return []any{}
	}
}

func caslNullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func (p *CASLCloudProvider) invalidateCASLEditorCaches() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cachedObjects = nil
	p.cachedObjectsAt = time.Time{}
	p.objectByInternalID = make(map[int]caslGrdObject)

	p.deviceByDeviceID = make(map[string]caslDevice)
	p.deviceByObjectID = make(map[string]caslDevice)
	p.deviceByNumber = make(map[int64]caslDevice)
	p.cachedDevicesAt = time.Time{}

	p.cachedUsers = make(map[string]caslUser)
	p.cachedUsersAt = time.Time{}

	p.cachedGroupStats = make(map[string]map[int]int)
	p.cachedGroupStatsAt = time.Time{}
}
