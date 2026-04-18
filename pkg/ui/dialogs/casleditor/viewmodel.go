package casleditor

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"obj_catalog_fyne_v3/pkg/contracts"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type ObjectUpdateData struct {
	Name           string
	Address        string
	Lat            string
	Long           string
	Description    string
	Contract       string
	ManagerID      string
	Note           string
	StartDate      int64
	Status         string
	ObjectType     string
	IDRequest      string
	ReactingPultID string
	GeoZoneID      int64
	BusinessCoeff  *float64
}

type DeviceUpdateData struct {
	Number        int64
	Name          string
	Type          string
	Timeout       int64
	SIM1          string
	SIM2          string
	TechnicianID  string
	Units         string
	Requisites    string
	ChangeDate    int64
	ReglamentDate int64
	LicenceKey    string
	RemotePass    string
}

type RoomUpdateData struct {
	Name        string
	Description string
	RTSP        string
}

type LineMutationData struct {
	LineNumber    int
	Description   string
	LineType      string
	GroupNumber   int
	AdapterType   string
	AdapterNumber int
	IsBlocked     bool
	RoomID        string
}

type EditorViewModel struct {
	provider  contracts.CASLObjectEditorProvider
	objectID  int64
	creating  bool
	onChanged func()

	Snapshot   contracts.CASLObjectEditorSnapshot
	WizardStep int

	// Option mappings
	ManagerOptionToID    map[string]string
	PultOptionToID       map[string]string
	TechOptionToID       map[string]string
	UserOptionToID       map[string]string
	RoomOptionToID       map[string]string
	GeoZoneOptionToID    map[string]int64
	DeviceTypeOptionToID map[string]string
	LineTypeOptionToID   map[string]string
	AdapterOptionToID    map[string]string
	AdapterOptions       []string
	ManagerOptions       []string
	PultOptions          []string
	TechOptions          []string
	RoomOptions          []string
	GeoZoneOptions       []string
	AllUserOptions       []string
	DeviceTypeOptions    []string
	LineTypeOptions      []string

	// Selection state
	RoomSelected     int
	RoomUserSelected int
	LineSelected     int
	RoomUsersLocal   []contracts.CASLRoomUserLink

	// UI update callbacks
	dataChangedListeners     []func()
	statusUpdateListeners    []func(string)
	headerUpdateListeners    []func(string)
	validationStateListeners []func()
	roomSelectionListeners   []func(int)
	lineSelectionListeners   []func(int)
	alertListeners           []func(title, msg string)
	errorListeners           []func(err error)

	// Internal state
	win                   fyne.Window
	deviceNumbers         []int64
	pendingLineNumber     int
	pendingRoomName       string
	pendingPrepareNewLine bool
	pendingFocusQuickLine bool
	pendingFocusRoomUsers bool
	autoBindingRoomLines  bool
	draftRoomSeq          int
	geoZoneCache          []CASLGeoZoneOption
	OnChanged             func() // Callback for external changes
}

func NewEditorViewModel(win fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onChanged func()) *EditorViewModel {
	return &EditorViewModel{
		win:                  win,
		provider:             provider,
		objectID:             objectID,
		creating:             objectID <= 0,
		onChanged:            onChanged,
		WizardStep:           1,
		ManagerOptionToID:    map[string]string{},
		PultOptionToID:       map[string]string{},
		TechOptionToID:       map[string]string{},
		UserOptionToID:       map[string]string{},
		RoomOptionToID:       map[string]string{},
		GeoZoneOptionToID:    map[string]int64{},
		DeviceTypeOptionToID: map[string]string{},
		LineTypeOptionToID:   map[string]string{},
		AdapterOptionToID:    map[string]string{},
		RoomSelected:         -1,
		RoomUserSelected:     -1,
		LineSelected:         -1,
	}
}

func (vm *EditorViewModel) AddDataChangedListener(fn func()) {
	if fn != nil {
		vm.dataChangedListeners = append(vm.dataChangedListeners, fn)
	}
}

func (vm *EditorViewModel) AddStatusUpdateListener(fn func(string)) {
	if fn != nil {
		vm.statusUpdateListeners = append(vm.statusUpdateListeners, fn)
	}
}

func (vm *EditorViewModel) AddHeaderUpdateListener(fn func(string)) {
	if fn != nil {
		vm.headerUpdateListeners = append(vm.headerUpdateListeners, fn)
	}
}

func (vm *EditorViewModel) AddValidationStateListener(fn func()) {
	if fn != nil {
		vm.validationStateListeners = append(vm.validationStateListeners, fn)
	}
}

func (vm *EditorViewModel) AddRoomSelectionListener(fn func(int)) {
	if fn != nil {
		vm.roomSelectionListeners = append(vm.roomSelectionListeners, fn)
	}
}

func (vm *EditorViewModel) AddLineSelectionListener(fn func(int)) {
	if fn != nil {
		vm.lineSelectionListeners = append(vm.lineSelectionListeners, fn)
	}
}

func (vm *EditorViewModel) AddAlertListener(fn func(title, msg string)) {
	if fn != nil {
		vm.alertListeners = append(vm.alertListeners, fn)
	}
}

func (vm *EditorViewModel) AddErrorListener(fn func(err error)) {
	if fn != nil {
		vm.errorListeners = append(vm.errorListeners, fn)
	}
}

func (vm *EditorViewModel) ObjectID() int64                              { return vm.objectID }
func (vm *EditorViewModel) IsCreating() bool                             { return vm.creating }
func (vm *EditorViewModel) Provider() contracts.CASLObjectEditorProvider { return vm.provider }

func (vm *EditorViewModel) UserOptionByID(id string, mapping map[string]string) string {
	return optionLabelByValue(id, mapping)
}

func (vm *EditorViewModel) TechOptionByID(id string) string {
	return optionLabelByValue(id, vm.TechOptionToID)
}

func (vm *EditorViewModel) RoomOptionByID(id string) string {
	return optionLabelByValue(id, vm.RoomOptionToID)
}

func (vm *EditorViewModel) RoomNameByID(id string) string {
	for _, r := range vm.Snapshot.Object.Rooms {
		if r.RoomID == id {
			return FirstNonEmpty(r.Name, r.RoomID)
		}
	}
	return ""
}

func (vm *EditorViewModel) Reload() {
	vm.ReloadWithID(vm.objectID)
}

func (vm *EditorViewModel) ReloadWithID(objID int64) {
	vm.objectID = objID
	vm.creating = objID <= 0
	vm.setStatus("Завантаження...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		snap, err := vm.provider.GetCASLObjectEditorSnapshot(ctx, vm.objectID)
		if err != nil {
			fyne.Do(func() {
				vm.setStatus("Помилка завантаження")
				vm.showError(err)
			})
			return
		}

		nums, _ := vm.provider.ReadCASLDeviceNumbers(ctx)

		fyne.Do(func() {
			vm.Snapshot = snap
			vm.deviceNumbers = nums
			vm.RoomSelected = -1
			vm.LineSelected = -1
			vm.RoomUsersLocal = nil
			vm.ensureDefaultDraftRoom()
			vm.initDictionaries()
			vm.refreshHeader()
			vm.setStatus("Готово.")

			if vm.pendingLineNumber > 0 {
				if idx := vm.findLineIndexByNumber(vm.pendingLineNumber); idx >= 0 {
					vm.LineSelected = idx
				}
				vm.pendingLineNumber = 0
			}
			if strings.TrimSpace(vm.pendingRoomName) != "" {
				if idx := vm.findRoomIndexByName(vm.pendingRoomName); idx >= 0 {
					vm.RoomSelected = idx
				}
				vm.pendingRoomName = ""
			}

			if len(vm.Snapshot.Object.Rooms) > 0 {
				if vm.RoomSelected < 0 || vm.RoomSelected >= len(vm.Snapshot.Object.Rooms) {
					vm.RoomSelected = 0
				}
			}
			if len(vm.Snapshot.Object.Device.Lines) > 0 {
				if vm.LineSelected < 0 || vm.LineSelected >= len(vm.Snapshot.Object.Device.Lines) {
					vm.LineSelected = 0
				}
			}

			vm.emitDataChanged()
			vm.emitValidationStateChanged()
		})
	}()
}

func (vm *EditorViewModel) SubmitObject(data ObjectUpdateData) {
	if !vm.HasObject() {
		if err := vm.DraftObject(data); err != nil {
			vm.showError(err)
		}
		return
	}
	update := contracts.CASLGuardObjectUpdate{
		ObjID:          vm.Snapshot.Object.ObjID,
		Name:           data.Name,
		Address:        data.Address,
		Long:           data.Long,
		Lat:            data.Lat,
		Description:    data.Description,
		Contract:       data.Contract,
		ManagerID:      data.ManagerID,
		Note:           data.Note,
		StartDate:      data.StartDate,
		Status:         data.Status,
		ObjectType:     data.ObjectType,
		IDRequest:      data.IDRequest,
		ReactingPultID: data.ReactingPultID,
		GeoZoneID:      data.GeoZoneID,
		BusinessCoeff:  data.BusinessCoeff,
	}
	vm.runMutation("Збереження об'єкта...", func(ctx context.Context) error {
		return vm.provider.UpdateCASLObject(ctx, update)
	})
}

func (vm *EditorViewModel) RunMutation(started string, fn func(ctx context.Context) error) {
	vm.runMutation(started, fn)
}

func (vm *EditorViewModel) runMutation(started string, fn func(ctx context.Context) error) {
	vm.runMutationWithHooks(started, fn, nil, nil)
}

func (vm *EditorViewModel) runMutationWithHooks(started string, fn func(ctx context.Context) error, onSuccess func(), onError func(error)) {
	vm.setStatus(started)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := fn(ctx)
		fyne.Do(func() {
			if err != nil {
				if onError != nil {
					onError(err)
				}
				vm.setStatus("Помилка")
				vm.showError(err)
				return
			}
			if onSuccess != nil {
				onSuccess()
			}
			vm.setStatus("Збережено")
			if vm.onChanged != nil {
				vm.onChanged()
			}
			vm.Reload()
		})
	}()
}

func (vm *EditorViewModel) HasObject() bool {
	return strings.TrimSpace(vm.Snapshot.Object.ObjID) != "" || vm.objectID > 0
}

func (vm *EditorViewModel) HasDevice() bool {
	return strings.TrimSpace(vm.Snapshot.Object.Device.DeviceID) != ""
}

func (vm *EditorViewModel) setStatus(msg string) {
	for _, fn := range vm.statusUpdateListeners {
		fn(msg)
	}
}

func (vm *EditorViewModel) showError(err error) {
	if len(vm.errorListeners) > 0 {
		for _, fn := range vm.errorListeners {
			fn(err)
		}
		return
	}
	dialog.ShowError(err, vm.win)
}

func (vm *EditorViewModel) showAlert(title, msg string) {
	if len(vm.alertListeners) > 0 {
		for _, fn := range vm.alertListeners {
			fn(title, msg)
		}
		return
	}
	dialog.ShowInformation(title, msg, vm.win)
}

func (vm *EditorViewModel) ensureDefaultDraftRoom() {
	if !vm.creating || vm.HasObject() || len(vm.Snapshot.Object.Rooms) > 0 {
		return
	}
	vm.Snapshot.Object.Rooms = []contracts.CASLRoomDetails{
		{
			RoomID:      vm.nextDraftRoomID(),
			Name:        "Приміщення",
			Description: "Без опису",
			Users:       nil,
			Lines:       nil,
		},
	}
	if vm.RoomSelected < 0 {
		vm.RoomSelected = 0
	}
}

func (vm *EditorViewModel) nextDraftRoomID() string {
	vm.draftRoomSeq++
	return fmt.Sprintf("draft-room-%d", vm.draftRoomSeq)
}

func (vm *EditorViewModel) findLineIndexByNumber(num int) int {
	for i, l := range vm.Snapshot.Object.Device.Lines {
		if l.LineNumber == num {
			return i
		}
	}
	return -1
}

func (vm *EditorViewModel) findRoomIndexByName(name string) int {
	name = strings.TrimSpace(name)
	for i, r := range vm.Snapshot.Object.Rooms {
		if strings.TrimSpace(r.Name) == name {
			return i
		}
	}
	return -1
}

func (vm *EditorViewModel) SelectRoom(index int) {
	if index < 0 || index >= len(vm.Snapshot.Object.Rooms) {
		return
	}
	vm.RoomSelected = index
	room := vm.Snapshot.Object.Rooms[index]
	vm.RoomUsersLocal = slices.Clone(room.Users)
	vm.emitRoomSelectionChanged(index)
}

func (vm *EditorViewModel) SelectLine(index int) {
	if index < 0 || index >= len(vm.Snapshot.Object.Device.Lines) {
		return
	}
	vm.LineSelected = index
	vm.emitLineSelectionChanged(index)
}

func (vm *EditorViewModel) SelectedRoom() (contracts.CASLRoomDetails, bool) {
	if vm.RoomSelected < 0 || vm.RoomSelected >= len(vm.Snapshot.Object.Rooms) {
		return contracts.CASLRoomDetails{}, false
	}
	return vm.Snapshot.Object.Rooms[vm.RoomSelected], true
}

func (vm *EditorViewModel) SelectedLine() (contracts.CASLDeviceLineDetails, bool) {
	if vm.LineSelected < 0 || vm.LineSelected >= len(vm.Snapshot.Object.Device.Lines) {
		return contracts.CASLDeviceLineDetails{}, false
	}
	return vm.Snapshot.Object.Device.Lines[vm.LineSelected], true
}

func (vm *EditorViewModel) SubmitDevice(data DeviceUpdateData) {
	if !vm.HasDevice() {
		if err := vm.DraftDevice(data); err != nil {
			vm.showError(err)
		}
		return
	}
	update := contracts.CASLDeviceUpdate{
		DeviceID:          vm.Snapshot.Object.Device.DeviceID,
		Number:            data.Number,
		Name:              data.Name,
		DeviceType:        data.Type,
		Timeout:           data.Timeout,
		SIM1:              data.SIM1,
		SIM2:              data.SIM2,
		TechnicianID:      data.TechnicianID,
		Units:             data.Units,
		Requisites:        data.Requisites,
		ChangeDate:        data.ChangeDate,
		ReglamentDate:     data.ReglamentDate,
		LicenceKey:        data.LicenceKey,
		PasswRemote:       data.RemotePass,
		MoreAlarmTime:     vm.Snapshot.Object.Device.MoreAlarmTime,
		IgnoringAlarmTime: vm.Snapshot.Object.Device.IgnoringAlarmTime,
	}
	vm.runMutation("Збереження обладнання...", func(ctx context.Context) error {
		return vm.provider.UpdateCASLDevice(ctx, update)
	})
}

func (vm *EditorViewModel) SaveRoom(data RoomUpdateData) {
	room, ok := vm.SelectedRoom()
	if !ok {
		return
	}
	if !vm.HasObject() {
		if err := vm.DraftSelectedRoom(data); err != nil {
			vm.showError(err)
		}
		return
	}
	update := contracts.CASLRoomUpdate{
		ObjID:       vm.Snapshot.Object.ObjID,
		RoomID:      room.RoomID,
		Name:        data.Name,
		Description: data.Description,
		RTSP:        data.RTSP,
	}
	vm.runMutation("Збереження приміщення...", func(ctx context.Context) error {
		return vm.provider.UpdateCASLRoom(ctx, update)
	})
}

func (vm *EditorViewModel) CreateRoom(data RoomUpdateData) {
	if err := ValidateRoomDraftData(data); err != nil {
		vm.showError(err)
		return
	}
	if !vm.HasObject() {
		vm.Snapshot.Object.Rooms = append(vm.Snapshot.Object.Rooms, contracts.CASLRoomDetails{
			RoomID:      vm.nextDraftRoomID(),
			Name:        data.Name,
			Description: data.Description,
			RTSP:        data.RTSP,
			Users:       nil,
			Lines:       nil,
		})
		vm.RoomSelected = len(vm.Snapshot.Object.Rooms) - 1
		vm.initDictionaries()
		vm.emitDataChanged()
		return
	}
	create := contracts.CASLRoomCreate{
		ObjID:       vm.Snapshot.Object.ObjID,
		Name:        data.Name,
		Description: data.Description,
		RTSP:        data.RTSP,
	}
	vm.runMutationWithHooks("Створення приміщення...", func(ctx context.Context) error {
		return vm.provider.CreateCASLRoom(ctx, create)
	}, func() {
		vm.pendingRoomName = data.Name
		vm.pendingFocusRoomUsers = true
	}, nil)
}

func (vm *EditorViewModel) SaveLine(data LineMutationData) {
	line, ok := vm.SelectedLine()
	if !ok {
		return
	}
	if err := ValidateCASLLineNumberRange(data.LineNumber); err != nil {
		vm.showError(err)
		return
	}
	if err := ValidateCASLLineDescription(data.Description); err != nil {
		vm.showError(err)
		return
	}
	if err := ValidateCASLLineNumberUnique(vm.Snapshot.Object.Device.Lines, data.LineNumber, vm.LineSelected); err != nil {
		vm.showError(err)
		return
	}
	mutation := contracts.CASLDeviceLineMutation{
		DeviceID:      vm.Snapshot.Object.Device.DeviceID,
		LineID:        line.LineID,
		LineNumber:    data.LineNumber,
		GroupNumber:   data.GroupNumber,
		AdapterType:   data.AdapterType,
		AdapterNumber: data.AdapterNumber,
		Description:   data.Description,
		LineType:      data.LineType,
		IsBlocked:     data.IsBlocked,
	}
	if !vm.HasDevice() {
		vm.Snapshot.Object.Device.Lines[vm.LineSelected] = vm.mutationToDetails(mutation, line.RoomID)
		vm.initDictionaries()
		vm.emitDataChanged()
		return
	}
	vm.runMutation("Збереження зони...", func(ctx context.Context) error {
		return vm.provider.UpdateCASLDeviceLine(ctx, mutation)
	})
}

func (vm *EditorViewModel) CreateLine(data LineMutationData) {
	if err := ValidateCASLLineNumberRange(data.LineNumber); err != nil {
		vm.showError(err)
		return
	}
	if err := ValidateCASLLineDescription(data.Description); err != nil {
		vm.showError(err)
		return
	}
	if err := ValidateCASLLineNumberUnique(vm.Snapshot.Object.Device.Lines, data.LineNumber, -1); err != nil {
		vm.showError(err)
		return
	}
	mutation := contracts.CASLDeviceLineMutation{
		DeviceID:      vm.Snapshot.Object.Device.DeviceID,
		LineNumber:    data.LineNumber,
		GroupNumber:   data.GroupNumber,
		AdapterType:   data.AdapterType,
		AdapterNumber: data.AdapterNumber,
		Description:   data.Description,
		LineType:      data.LineType,
		IsBlocked:     data.IsBlocked,
	}
	if !vm.HasDevice() {
		vm.Snapshot.Object.Device.Lines = append(vm.Snapshot.Object.Device.Lines, vm.mutationToDetails(mutation, ""))
		vm.initDictionaries()
		vm.emitDataChanged()
		return
	}
	vm.pendingPrepareNewLine = true
	vm.runMutation("Створення зони...", func(ctx context.Context) error {
		return vm.provider.CreateCASLDeviceLine(ctx, mutation)
	})
}

func (vm *EditorViewModel) BindLineToRoom(lineNumber int, roomID string) {
	if !vm.HasObject() || !vm.HasDevice() {
		vm.attachLineToRoomDraft(roomID, lineNumber)
		return
	}
	binding := contracts.CASLLineToRoomBinding{
		ObjID:      vm.Snapshot.Object.ObjID,
		DeviceID:   vm.Snapshot.Object.Device.DeviceID,
		LineNumber: lineNumber,
		RoomID:     roomID,
	}
	vm.runMutation("Прив'язка зони до приміщення...", func(ctx context.Context) error {
		return vm.provider.AddCASLLineToRoom(ctx, binding)
	})
}

func (vm *EditorViewModel) AddUserToRoom(userID string) {
	room, ok := vm.SelectedRoom()
	if !ok {
		return
	}
	if !vm.HasObject() {
		vm.RoomUsersLocal = append(vm.RoomUsersLocal, contracts.CASLRoomUserLink{
			UserID:   userID,
			Priority: len(vm.RoomUsersLocal) + 1,
		})
		vm.syncRoomUsersToSnapshot()
		return
	}
	request := contracts.CASLAddUserToRoomRequest{
		ObjID:    vm.Snapshot.Object.ObjID,
		RoomID:   room.RoomID,
		UserID:   userID,
		Priority: len(vm.RoomUsersLocal) + 1,
	}
	vm.runMutation("Додавання користувача...", func(ctx context.Context) error {
		return vm.provider.AddCASLUserToRoom(ctx, request)
	})
}

func (vm *EditorViewModel) RemoveUserFromRoom(index int) {
	room, ok := vm.SelectedRoom()
	if !ok || index < 0 || index >= len(vm.RoomUsersLocal) {
		return
	}
	if !vm.HasObject() {
		vm.RoomUsersLocal = append(vm.RoomUsersLocal[:index], vm.RoomUsersLocal[index+1:]...)
		vm.syncRoomUsersToSnapshot()
		return
	}
	request := contracts.CASLRemoveUserFromRoomRequest{
		ObjID:  vm.Snapshot.Object.ObjID,
		RoomID: room.RoomID,
		UserID: vm.RoomUsersLocal[index].UserID,
	}
	vm.runMutation("Видалення користувача...", func(ctx context.Context) error {
		return vm.provider.RemoveCASLUserFromRoom(ctx, request)
	})
}

func (vm *EditorViewModel) UpdateRoomUserHozNum(index int, raw string) {
	if index < 0 || index >= len(vm.RoomUsersLocal) {
		return
	}
	cleaned := DigitsOnly(raw)
	if vm.RoomUsersLocal[index].HozNum != cleaned {
		vm.RoomUsersLocal[index].HozNum = cleaned
		vm.syncRoomUsersToSnapshot()
	}
}

func (vm *EditorViewModel) SaveRoomUserPriorities() {
	room, ok := vm.SelectedRoom()
	if !ok || !vm.HasObject() {
		return
	}
	items := make([]contracts.CASLRoomUserPriority, 0, len(vm.RoomUsersLocal))
	for idx, user := range vm.RoomUsersLocal {
		items = append(items, contracts.CASLRoomUserPriority{
			UserID:   user.UserID,
			RoomID:   room.RoomID,
			Priority: idx + 1,
			HozNum:   user.HozNum,
		})
	}
	vm.runMutation("Збереження порядку...", func(ctx context.Context) error {
		return vm.provider.UpdateCASLRoomUserPriorities(ctx, vm.objectID, items)
	})
}

func (vm *EditorViewModel) CreateUserAndAddToRoom() {
	room, ok := vm.SelectedRoom()
	if !ok {
		vm.showAlert("Користувач", "Оберіть приміщення.")
		return
	}

	lastNameEntry := widget.NewEntry()
	firstNameEntry := widget.NewEntry()
	middleNameEntry := widget.NewEntry()
	tagEntry := widget.NewEntry()
	emailEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	phone1Entry := widget.NewEntry()
	phone2Entry := widget.NewEntry()
	roleSelect := widget.NewSelect([]string{"IN_CHARGE", "MANAGER", "TECHNICIAN", "ADMIN"}, nil)
	roleSelect.SetSelected("IN_CHARGE")
	phone1Entry.SetPlaceHolder("+38 (050) 329-92-04")
	phone2Entry.SetPlaceHolder("+38 (050) 329-92-04")

	BindDebouncedPhoneFormatter(phone1Entry, 250*time.Millisecond, nil)
	BindDebouncedPhoneFormatter(phone2Entry, 250*time.Millisecond, nil)

	form := widget.NewForm(
		widget.NewFormItem("Прізвище", lastNameEntry),
		widget.NewFormItem("Ім'я", firstNameEntry),
		widget.NewFormItem("По батькові", middleNameEntry),
		widget.NewFormItem("Tag", tagEntry),
		widget.NewFormItem("Role", roleSelect),
		widget.NewFormItem("Email", emailEntry),
		widget.NewFormItem("Password", passwordEntry),
		widget.NewFormItem("Телефон 1", phone1Entry),
		widget.NewFormItem("Телефон 2", phone2Entry),
	)

	dialog.ShowCustomConfirm("Новий користувач CASL", "Створити", "Скасувати", form, func(confirmed bool) {
		if !confirmed {
			return
		}
		phone1, err := NormalizeCASLEditorUserPhone(phone1Entry.Text)
		if err != nil {
			vm.showError(fmt.Errorf("телефон 1: %w", err))
			return
		}
		phone2, err := NormalizeCASLEditorUserPhone(phone2Entry.Text)
		if err != nil {
			vm.showError(fmt.Errorf("телефон 2: %w", err))
			return
		}
		request := contracts.CASLUserCreateRequest{
			LastName:   strings.TrimSpace(lastNameEntry.Text),
			FirstName:  strings.TrimSpace(firstNameEntry.Text),
			MiddleName: strings.TrimSpace(middleNameEntry.Text),
			Tag:        strings.TrimSpace(tagEntry.Text),
			Role:       strings.TrimSpace(roleSelect.Selected),
			Email:      strings.TrimSpace(emailEntry.Text),
			Password:   strings.TrimSpace(passwordEntry.Text),
			PhoneNumbers: []contracts.CASLPhoneNumber{
				{Active: true, Number: phone1},
				{Active: false, Number: phone2},
			},
		}

		vm.setStatus("Створення користувача...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			user, err := vm.provider.CreateCASLUser(ctx, request)
			if err == nil && vm.HasObject() && strings.TrimSpace(user.UserID) != "" {
				err = vm.provider.AddCASLUserToRoom(ctx, contracts.CASLAddUserToRoomRequest{
					ObjID:    vm.Snapshot.Object.ObjID,
					RoomID:   room.RoomID,
					UserID:   user.UserID,
					Priority: len(vm.RoomUsersLocal) + 1,
				})
			}

			fyne.Do(func() {
				if err != nil {
					vm.setStatus("Помилка створення користувача")
					vm.showError(err)
					return
				}
				if strings.TrimSpace(user.UserID) != "" {
					vm.Snapshot.Users = append(vm.Snapshot.Users, user)
					vm.RoomUsersLocal = append(vm.RoomUsersLocal, contracts.CASLRoomUserLink{
						UserID:   user.UserID,
						Priority: len(vm.RoomUsersLocal) + 1,
					})
					vm.syncRoomUsersToSnapshot()
					vm.initDictionaries()
					vm.emitDataChanged()
				}
				vm.setStatus("Користувача створено")
			})
		}()
	}, vm.win)
}

func (vm *EditorViewModel) ValidateObjectForm(data ObjectUpdateData) error {
	if utf8.RuneCountInString(data.Name) < 2 {
		return fmt.Errorf("назва занадто коротка")
	}
	if utf8.RuneCountInString(data.Address) < 5 {
		return fmt.Errorf("адреса занадто коротка")
	}
	if utf8.RuneCountInString(data.Description) < 3 {
		return fmt.Errorf("опис занадто короткий")
	}
	return nil
}

func (vm *EditorViewModel) PickObjectCoordinatesOnMap() {
	if ShowObjectCoordinatesPicker == nil {
		vm.showAlert("Карта", "Функція вибору на карті недоступна.")
		return
	}
	ShowObjectCoordinatesPicker(
		vm.win,
		strings.TrimSpace(vm.Snapshot.Object.Lat),
		strings.TrimSpace(vm.Snapshot.Object.Long),
		"Вибір координат об'єкта",
		strings.TrimSpace(vm.Snapshot.Object.Address),
		func(lat, lon string) {
			vm.Snapshot.Object.Lat = strings.TrimSpace(lat)
			vm.Snapshot.Object.Long = strings.TrimSpace(lon)
			vm.emitDataChanged()
		},
	)
}

func (vm *EditorViewModel) UploadObjectImage() {
	if !vm.HasObject() {
		vm.openImagePicker("Додати фото об'єкта", func(imageType, encoded string) {
			vm.ReplaceObjectDraftImages(append(vm.Snapshot.Object.Images, "data:image/"+imageType+";base64,"+encoded))
		})
		return
	}
	vm.openImagePicker("Додати фото об'єкта", func(imageType, encoded string) {
		vm.runMutation("Завантаження фото...", func(ctx context.Context) error {
			return vm.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
				ObjID:     vm.Snapshot.Object.ObjID,
				ImageType: imageType,
				ImageData: encoded,
			})
		})
	})
}

func (vm *EditorViewModel) UploadRoomImage() {
	room, ok := vm.SelectedRoom()
	if !ok {
		return
	}
	if !vm.HasObject() {
		vm.openImagePicker("Додати фото приміщення", func(imageType, encoded string) {
			images := append(slices.Clone(room.Images), "data:image/"+imageType+";base64,"+encoded)
			vm.ReplaceSelectedRoomDraftImages(images)
		})
		return
	}
	vm.openImagePicker("Додати фото приміщення", func(imageType, encoded string) {
		vm.runMutation("Завантаження фото...", func(ctx context.Context) error {
			return vm.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
				ObjID:     vm.Snapshot.Object.ObjID,
				RoomID:    room.RoomID,
				ImageType: imageType,
				ImageData: encoded,
			})
		})
	})
}

func (vm *EditorViewModel) DeleteObjectImage(imageID string) {
	if !vm.HasObject() {
		images := slices.Clone(vm.Snapshot.Object.Images)
		for idx, raw := range images {
			if raw == imageID {
				vm.ReplaceObjectDraftImages(append(images[:idx], images[idx+1:]...))
				return
			}
		}
		return
	}
	vm.runMutation("Видалення фото об'єкта...", func(ctx context.Context) error {
		return vm.provider.DeleteCASLImage(ctx, contracts.CASLImageDeleteRequest{
			ObjID:   vm.Snapshot.Object.ObjID,
			ImageID: imageID,
		})
	})
}

func (vm *EditorViewModel) DeleteRoomImage(imageID string) {
	room, ok := vm.SelectedRoom()
	if !ok {
		return
	}
	if !vm.HasObject() {
		images := slices.Clone(room.Images)
		for idx, raw := range images {
			if raw == imageID {
				vm.ReplaceSelectedRoomDraftImages(append(images[:idx], images[idx+1:]...))
				return
			}
		}
		return
	}
	vm.runMutation("Видалення фото приміщення...", func(ctx context.Context) error {
		return vm.provider.DeleteCASLImage(ctx, contracts.CASLImageDeleteRequest{
			ObjID:   vm.Snapshot.Object.ObjID,
			RoomID:  room.RoomID,
			ImageID: imageID,
		})
	})
}

func (vm *EditorViewModel) ObjectScopedImages() []string {
	return vm.Snapshot.Object.Images
}

func (vm *EditorViewModel) ShowImagePreview(title, raw string) {
	win := fyne.CurrentApp().NewWindow(title)
	res, _ := caslImageResource(raw, "preview")
	if res != nil {
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillContain
		win.SetContent(img)
	} else {
		win.SetContent(widget.NewLabel("Не вдалося завантажити зображення"))
	}
	win.Resize(fyne.NewSize(800, 600))
	win.Show()
}

func (vm *EditorViewModel) InitDictionaries() { vm.initDictionaries() }

func (vm *EditorViewModel) initDictionaries() {
	dict := vm.Snapshot.Dictionary
	vm.DeviceTypeOptions, vm.DeviceTypeOptionToID = labeledOptionMap(caslDeviceTypeOptionsMap(dict))
	vm.LineTypeOptions, vm.LineTypeOptionToID = labeledOptionMap(caslLineTypeOptionsMap(dict))
	vm.AdapterOptions, vm.AdapterOptionToID = labeledOptionMap(caslAdapterTypeOptionsMap(dict))

	vm.ManagerOptionToID = map[string]string{}
	vm.ManagerOptions = nil
	for _, u := range vm.Snapshot.Users {
		if u.Role == "MANAGER" || u.Role == "ADMIN" {
			label := fmt.Sprintf("%s %s", u.LastName, u.FirstName)
			vm.ManagerOptionToID[label] = u.UserID
			vm.ManagerOptions = append(vm.ManagerOptions, label)
		}
	}
	sort.Strings(vm.ManagerOptions)

	vm.PultOptionToID = map[string]string{}
	vm.PultOptions = nil
	for _, p := range vm.Snapshot.Pults {
		label := p.Name
		if p.Nickname != "" {
			label = fmt.Sprintf("%s (%s)", p.Name, p.Nickname)
		}
		vm.PultOptionToID[label] = p.PultID
		vm.PultOptions = append(vm.PultOptions, label)
	}
	sort.Strings(vm.PultOptions)

	vm.TechOptionToID = map[string]string{}
	vm.TechOptions = nil
	for _, u := range vm.Snapshot.Users {
		if strings.Contains(u.Role, "TECH") || u.Role == "ADMIN" {
			label := fmt.Sprintf("%s %s", u.LastName, u.FirstName)
			vm.TechOptionToID[label] = u.UserID
			vm.TechOptions = append(vm.TechOptions, label)
		}
	}
	sort.Strings(vm.TechOptions)

	vm.UserOptionToID = map[string]string{}
	vm.AllUserOptions = make([]string, 0, len(vm.Snapshot.Users))
	for _, u := range vm.Snapshot.Users {
		label := fmt.Sprintf("%s %s (%s)", u.LastName, u.FirstName, u.UserID)
		vm.UserOptionToID[label] = u.UserID
		vm.AllUserOptions = append(vm.AllUserOptions, label)
	}
	sort.Strings(vm.AllUserOptions)

	vm.RoomOptions, vm.RoomOptionToID = vm.roomOptionsMap()
	vm.refreshGeoZoneOptions()
}

func (vm *EditorViewModel) roomOptionsMap() ([]string, map[string]string) {
	mapping := map[string]string{}
	for _, r := range vm.Snapshot.Object.Rooms {
		label := FirstNonEmpty(r.Name, "Приміщення #"+r.RoomID)
		mapping[label] = r.RoomID
	}
	return labeledOptionMap(mapping)
}

func (vm *EditorViewModel) refreshGeoZoneOptions() {
	vm.GeoZoneOptionToID = map[string]int64{}
	vm.GeoZoneOptions = nil
	for _, item := range vm.geoZoneCache {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprintf("Група реагування #%d", item.ID)
		}
		vm.GeoZoneOptionToID[label] = item.ID
		vm.GeoZoneOptions = append(vm.GeoZoneOptions, label)
	}
	sort.Strings(vm.GeoZoneOptions)
}

func (vm *EditorViewModel) refreshHeader() {
	title := fmt.Sprintf("Об'єкт #%d: %s", vm.objectID, FirstNonEmpty(vm.Snapshot.Object.Name, "Без назви"))
	if vm.creating {
		title = "Створення нового об'єкта CASL"
	}
	for _, fn := range vm.headerUpdateListeners {
		fn(title)
	}
}

func (vm *EditorViewModel) DisplayLineType(raw string) string {
	return caslLineTypeDisplayName(raw)
}

func (vm *EditorViewModel) DisplayAdapterType(raw string) string {
	return humanizeCASLAdapterType(raw)
}

func (vm *EditorViewModel) UserLabelByID(id string) string {
	for _, u := range vm.Snapshot.Users {
		if u.UserID == id {
			return fmt.Sprintf("%s %s", u.LastName, u.FirstName)
		}
	}
	return id
}

func (vm *EditorViewModel) RoomUserDetailsText(link contracts.CASLRoomUserLink) string {
	for _, u := range vm.Snapshot.Users {
		if u.UserID == link.UserID {
			role := TranslateRole(u.Role)
			phones := PhoneNumbersText(u.PhoneNumbers)
			text := role
			if phones != "" {
				text += " | " + phones
			}
			if link.HozNum != "" {
				text += " | Гос. номер: " + link.HozNum
			}
			return text
		}
	}
	return ""
}

func (vm *EditorViewModel) MoveRoomUserUp() {
	if vm.RoomUserSelected <= 0 || vm.RoomUserSelected >= len(vm.RoomUsersLocal) {
		return
	}
	vm.RoomUsersLocal[vm.RoomUserSelected-1], vm.RoomUsersLocal[vm.RoomUserSelected] = vm.RoomUsersLocal[vm.RoomUserSelected], vm.RoomUsersLocal[vm.RoomUserSelected-1]
	vm.RoomUserSelected--
	vm.syncRoomUsersToSnapshot()
}

func (vm *EditorViewModel) MoveRoomUserDown() {
	if vm.RoomUserSelected < 0 || vm.RoomUserSelected >= len(vm.RoomUsersLocal)-1 {
		return
	}
	vm.RoomUsersLocal[vm.RoomUserSelected+1], vm.RoomUsersLocal[vm.RoomUserSelected] = vm.RoomUsersLocal[vm.RoomUserSelected], vm.RoomUsersLocal[vm.RoomUserSelected+1]
	vm.RoomUserSelected++
	vm.syncRoomUsersToSnapshot()
}

func (vm *EditorViewModel) syncRoomUsersToSnapshot() {
	if vm.RoomSelected >= 0 && vm.RoomSelected < len(vm.Snapshot.Object.Rooms) {
		vm.Snapshot.Object.Rooms[vm.RoomSelected].Users = slices.Clone(vm.RoomUsersLocal)
		vm.emitDataChanged()
	}
}

func (vm *EditorViewModel) LineDetailsByNumber(num int) (contracts.CASLDeviceLineDetails, bool) {
	for _, l := range vm.Snapshot.Object.Device.Lines {
		if l.LineNumber == num {
			return l, true
		}
	}
	return contracts.CASLDeviceLineDetails{}, false
}

func (vm *EditorViewModel) UnblockDevice() {
	if !vm.HasDevice() {
		return
	}
	vm.runMutation("Розблокування приладу...", func(ctx context.Context) error {
		return vm.provider.UnblockCASLDevice(ctx, vm.Snapshot.Object.Device.DeviceID)
	})
}

func (vm *EditorViewModel) ToggleDeviceBlock(reason string, hours int) {
	if !vm.HasDevice() {
		return
	}
	until := time.Now().Unix() + int64(hours*3600)
	request := contracts.CASLDeviceBlockRequest{
		DeviceID:     vm.Snapshot.Object.Device.DeviceID,
		DeviceNumber: vm.Snapshot.Object.Device.Number,
		TimeUnblock:  until,
		Message:      reason,
	}
	vm.runMutation("Блокування приладу...", func(ctx context.Context) error {
		return vm.provider.BlockCASLDevice(ctx, request)
	})
}

func (vm *EditorViewModel) openImagePicker(title string, onLoaded func(string, string)) {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		data, _ := os.ReadFile(reader.URI().Path())
		imageType, encoded := caslEncodeImageUpload(reader.URI().Name(), data)
		onLoaded(imageType, encoded)
	}, vm.win)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png"}))
	fileDialog.Show()
}

func (vm *EditorViewModel) attachLineToRoomDraft(roomID string, lineNum int) {
	for idx := range vm.Snapshot.Object.Rooms {
		r := &vm.Snapshot.Object.Rooms[idx]
		filtered := r.Lines[:0]
		for _, l := range r.Lines {
			if l.LineNumber != lineNum {
				filtered = append(filtered, l)
			}
		}
		r.Lines = filtered
		if r.RoomID == roomID {
			for _, dl := range vm.Snapshot.Object.Device.Lines {
				if dl.LineNumber == lineNum {
					r.Lines = append(r.Lines, contracts.CASLRoomLineLink{
						LineNumber:    dl.LineNumber,
						AdapterType:   dl.AdapterType,
						GroupNumber:   dl.GroupNumber,
						AdapterNumber: dl.AdapterNumber,
					})
					break
				}
			}
		}
	}
	for idx := range vm.Snapshot.Object.Device.Lines {
		if vm.Snapshot.Object.Device.Lines[idx].LineNumber == lineNum {
			vm.Snapshot.Object.Device.Lines[idx].RoomID = roomID
		}
	}
	vm.initDictionaries()
	vm.emitDataChanged()
}

func (vm *EditorViewModel) mutationToDetails(m contracts.CASLDeviceLineMutation, roomID string) contracts.CASLDeviceLineDetails {
	return contracts.CASLDeviceLineDetails{
		LineID:        m.LineID,
		LineNumber:    m.LineNumber,
		GroupNumber:   m.GroupNumber,
		AdapterType:   m.AdapterType,
		AdapterNumber: m.AdapterNumber,
		Description:   m.Description,
		LineType:      m.LineType,
		IsBlocked:     m.IsBlocked,
		RoomID:        roomID,
	}
}

func (vm *EditorViewModel) ValidateRoomUserHozNum(index int, raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("гос. номер має містити лише цифри")
	}
	if number < 1 || number > 128 {
		return fmt.Errorf("гос. номер має бути в межах 1..128")
	}
	for roomIdx, room := range vm.Snapshot.Object.Rooms {
		users := room.Users
		if roomIdx == vm.RoomSelected {
			users = vm.RoomUsersLocal
		}
		for userIdx, user := range users {
			if roomIdx == vm.RoomSelected && userIdx == index {
				continue
			}
			if strings.TrimSpace(user.HozNum) == value {
				return fmt.Errorf("гос. номер %s вже використовується", value)
			}
		}
	}
	return nil
}

func (vm *EditorViewModel) PendingAutoRoomLineBindings() []contracts.CASLLineToRoomBinding {
	if !vm.creating || len(vm.Snapshot.Object.Rooms) != 1 {
		return nil
	}
	room := vm.Snapshot.Object.Rooms[0]
	if len(room.Lines) > 0 {
		return nil
	}
	var bindings []contracts.CASLLineToRoomBinding
	for _, l := range vm.Snapshot.Object.Device.Lines {
		bindings = append(bindings, contracts.CASLLineToRoomBinding{
			ObjID:      vm.Snapshot.Object.ObjID,
			DeviceID:   vm.Snapshot.Object.Device.DeviceID,
			LineNumber: l.LineNumber,
			RoomID:     room.RoomID,
		})
	}
	return bindings
}

func (vm *EditorViewModel) LoadGeoZones() {
	provider, ok := vm.provider.(caslGeoZoneAccessProvider)
	if !ok {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		raw, err := provider.ReadManagers(ctx, 0, 1000)
		if err != nil {
			return
		}
		options := parseCASLGeoZoneOptions(raw)
		fyne.Do(func() {
			vm.geoZoneCache = options
			vm.refreshGeoZoneOptions()
			vm.emitDataChanged()
		})
	}()
}

func parseCASLGeoZoneOptions(raw []map[string]any) []CASLGeoZoneOption {
	result := make([]CASLGeoZoneOption, 0, len(raw))
	for _, item := range raw {
		id := int64(parseCASLAnyIntLocal(item["mgr_id"]))
		if id <= 0 {
			continue
		}

		name := strings.TrimSpace(asStringAny(item["name"]))
		number := strings.TrimSpace(asStringAny(item["number"]))
		phone := strings.TrimSpace(asStringAny(item["phone_number"]))
		label := name
		if number != "" && number != "<nil>" {
			label = fmt.Sprintf("%s #%s", FirstNonEmpty(name, "Група реагування"), number)
		}
		if label == "" {
			label = fmt.Sprintf("Група реагування #%d", id)
		}
		if phone != "" && phone != "<nil>" {
			label = fmt.Sprintf("%s | %s", label, phone)
		}

		result = append(result, CASLGeoZoneOption{
			ID:   id,
			Name: label,
		})
	}
	return result
}

func (vm *EditorViewModel) emitDataChanged() {
	for _, fn := range vm.dataChangedListeners {
		fn()
	}
}

func (vm *EditorViewModel) emitValidationStateChanged() {
	for _, fn := range vm.validationStateListeners {
		fn()
	}
}

func (vm *EditorViewModel) emitRoomSelectionChanged(index int) {
	for _, fn := range vm.roomSelectionListeners {
		fn(index)
	}
}

func (vm *EditorViewModel) emitLineSelectionChanged(index int) {
	for _, fn := range vm.lineSelectionListeners {
		fn(index)
	}
}
