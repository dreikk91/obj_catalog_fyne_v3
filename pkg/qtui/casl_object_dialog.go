//go:build qt

package qtui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/caslobject"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/geocode"
)

type caslObjectDialogState struct {
	parent   *qt.QWidget
	provider contracts.CASLObjectEditorProvider
	snapshot contracts.CASLObjectEditorSnapshot
	original contracts.CASLGuardObjectDetails
	creating bool

	name               *qt.QLineEdit
	address            *qt.QLineEdit
	latitude           *qt.QLineEdit
	longitude          *qt.QLineEdit
	description        *qt.QTextEdit
	contract           *qt.QLineEdit
	manager            *qt.QComboBox
	note               *qt.QLineEdit
	startDate          *qt.QLineEdit
	status             *qt.QComboBox
	objectType         *qt.QLineEdit
	requestID          *qt.QLineEdit
	reactingPult       *qt.QComboBox
	geoZoneID          *qt.QLineEdit
	businessCoeff      *qt.QLineEdit
	deviceNumber       *qt.QLineEdit
	deviceNumberStatus *qt.QLabel
	deviceName         *qt.QLineEdit
	deviceType         *qt.QComboBox
	deviceTimeout      *qt.QLineEdit
	changeDate         *qt.QLineEdit
	reglamentDate      *qt.QLineEdit
	sim1               *qt.QLineEdit
	sim2               *qt.QLineEdit
	technician         *qt.QComboBox
	units              *qt.QLineEdit
	requisites         *qt.QLineEdit
	licenceKey         *qt.QLineEdit
	remotePassword     *qt.QLineEdit

	roomsTable     *qt.QTableWidget
	linesTable     *qt.QTableWidget
	usersRoom      *qt.QComboBox
	usersTable     *qt.QTableWidget
	imageScope     *qt.QComboBox
	imagesList     *qt.QListWidget
	moreAlarmTable *qt.QTableWidget
	ignoringTable  *qt.QTableWidget
	statusText     *qt.QLabel

	moreAlarmRegimes []caslRegime
	ignoringRegimes  []caslRegime

	managerIDs          map[string]string
	pultIDs             map[string]string
	technicianIDs       map[string]string
	deviceTypes         map[string]string
	lineTypes           map[string]string
	adapterTypes        map[string]string
	dirty               bool
	deviceNumbers       map[int64]struct{}
	deviceNumbersLoaded bool
	deviceNumberTimer   *qt.QTimer
	imageLoadSeq        int
}

type caslRegime struct {
	Day       int
	StartTime string
	StopTime  string
	Cause     string
}

// ShowCASLObjectDialog opens the CASL creation wizard or editor.
func ShowCASLObjectDialog(
	parent *qt.QWidget,
	provider contracts.CASLObjectEditorProvider,
	snapshot contracts.CASLObjectEditorSnapshot,
	creating bool,
) (int64, bool) {
	state := newCASLObjectDialogState(parent, provider, snapshot, creating)
	return state.exec()
}

func newCASLObjectDialogState(
	parent *qt.QWidget,
	provider contracts.CASLObjectEditorProvider,
	snapshot contracts.CASLObjectEditorSnapshot,
	creating bool,
) *caslObjectDialogState {
	state := &caslObjectDialogState{
		parent:             parent,
		provider:           provider,
		snapshot:           snapshot,
		original:           cloneCASLGuardObject(snapshot.Object),
		creating:           creating,
		name:               newLineEdit(snapshot.Object.Name),
		address:            newLineEdit(snapshot.Object.Address),
		latitude:           newLineEdit(snapshot.Object.Lat),
		longitude:          newLineEdit(snapshot.Object.Long),
		description:        qt.NewQTextEdit2(),
		contract:           newLineEdit(snapshot.Object.Contract),
		manager:            qt.NewQComboBox2(),
		note:               newLineEdit(snapshot.Object.Note),
		startDate:          newLineEdit(caslDateText(snapshot.Object.StartDate)),
		status:             qt.NewQComboBox2(),
		objectType:         newLineEdit(snapshot.Object.ObjectType),
		requestID:          newLineEdit(snapshot.Object.IDRequest),
		reactingPult:       qt.NewQComboBox2(),
		geoZoneID:          newLineEdit(formatInt64NonZero(snapshot.Object.GeoZoneID)),
		businessCoeff:      newLineEdit(formatFloatPointer(snapshot.Object.BusinessCoeff)),
		deviceNumber:       newLineEdit(formatInt64NonZero(snapshot.Object.Device.Number)),
		deviceNumberStatus: qt.NewQLabel3("Перевірка номера виконається автоматично"),
		deviceName:         newLineEdit(snapshot.Object.Device.Name),
		deviceType:         qt.NewQComboBox2(),
		deviceTimeout:      newLineEdit(formatInt64NonZero(snapshot.Object.Device.Timeout)),
		changeDate:         newLineEdit(caslDateText(snapshot.Object.Device.ChangeDate)),
		reglamentDate:      newLineEdit(caslDateText(snapshot.Object.Device.ReglamentDate)),
		sim1:               newLineEdit(formatCASLPhoneForDisplay(snapshot.Object.Device.SIM1)),
		sim2:               newLineEdit(formatCASLPhoneForDisplay(snapshot.Object.Device.SIM2)),
		technician:         qt.NewQComboBox2(),
		units:              newLineEdit(snapshot.Object.Device.Units),
		requisites:         newLineEdit(snapshot.Object.Device.Requisites),
		licenceKey:         newLineEdit(snapshot.Object.Device.LicenceKey),
		remotePassword:     newLineEdit(snapshot.Object.Device.PasswRemote),
		roomsTable:         qt.NewQTableWidget3(0, 4),
		linesTable:         qt.NewQTableWidget3(0, 8),
		usersRoom:          qt.NewQComboBox2(),
		usersTable:         qt.NewQTableWidget3(0, 4),
		imageScope:         qt.NewQComboBox2(),
		imagesList:         qt.NewQListWidget2(),
		moreAlarmTable:     qt.NewQTableWidget3(0, 4),
		ignoringTable:      qt.NewQTableWidget3(0, 4),
		statusText:         qt.NewQLabel3("Готово"),
		moreAlarmRegimes:   caslRegimesFromAny(snapshot.Object.Device.MoreAlarmTime),
		ignoringRegimes:    caslRegimesFromAny(snapshot.Object.Device.IgnoringAlarmTime),
		deviceNumbers:      map[int64]struct{}{},
		deviceNumberTimer:  qt.NewQTimer(),
	}
	state.deviceNumberTimer.SetSingleShot(true)
	state.deviceNumberTimer.SetInterval(350)
	state.deviceNumberTimer.OnTimeout(state.updateDeviceNumberSuggestions)
	phonePlaceholder := "+380671234567 / 380671234567 / 0671234567"
	state.sim1.SetPlaceholderText(phonePlaceholder)
	state.sim2.SetPlaceholderText(phonePlaceholder)
	state.description.SetPlainText(snapshot.Object.Description)
	state.prepareOptions()
	state.prepareDefaults()
	state.refreshRooms()
	state.refreshLines()
	state.refreshUsers()
	state.refreshImageScopes()
	state.refreshRegimeTables()
	return state
}

func (s *caslObjectDialogState) prepareOptions() {
	s.managerIDs = map[string]string{}
	s.technicianIDs = map[string]string{}
	for _, user := range s.snapshot.Users {
		label := strings.TrimSpace(strings.Join([]string{user.LastName, user.FirstName}, " "))
		if label == "" {
			label = user.UserID
		}
		if user.Role == "MANAGER" || user.Role == "ADMIN" {
			s.managerIDs[label] = user.UserID
		}
		if strings.Contains(user.Role, "TECH") || user.Role == "ADMIN" {
			s.technicianIDs[label] = user.UserID
		}
	}
	s.pultIDs = map[string]string{}
	for _, pult := range s.snapshot.Pults {
		label := strings.TrimSpace(pult.Name)
		if pult.Nickname != "" {
			label += " (" + pult.Nickname + ")"
		}
		s.pultIDs[label] = pult.PultID
	}
	deviceOptions, deviceTypes := caslobject.DeviceTypeOptions(s.snapshot.Dictionary)
	_, lineTypes := caslobject.DictionaryOptions(s.snapshot.Dictionary, "line_types", "zone_types")
	_, adapterTypes := caslobject.DictionaryOptions(s.snapshot.Dictionary, "adapters", "adapter_types")
	s.deviceTypes = deviceTypes
	s.lineTypes = lineTypes
	s.adapterTypes = adapterTypes

	fillComboBox(s.manager, append([]string{""}, sortedMapKeys(s.managerIDs)...))
	setComboByValue(s.manager, s.managerIDs, s.snapshot.Object.ManagerID)
	fillComboBox(s.reactingPult, append([]string{""}, sortedMapKeys(s.pultIDs)...))
	setComboByValue(s.reactingPult, s.pultIDs, s.snapshot.Object.ReactingPultID)
	fillComboBox(s.technician, append([]string{""}, sortedMapKeys(s.technicianIDs)...))
	setComboByValue(s.technician, s.technicianIDs, s.snapshot.Object.Device.TechnicianID)
	fillEditableCombo(s.deviceType, deviceOptions, s.deviceTypes, s.snapshot.Object.Device.Type)
	if len(s.lineTypes) == 0 {
		s.lineTypes = map[string]string{"NORMAL": "NORMAL"}
	}
	if len(s.adapterTypes) == 0 {
		s.adapterTypes = map[string]string{"SYS": "SYS"}
	}
	s.status.AddItems([]string{"Включено", "Вимкнено"})
	setComboTextFallback(s.status, s.snapshot.Object.ObjectStatus, "Включено")
}

func (s *caslObjectDialogState) prepareDefaults() {
	if !s.creating {
		return
	}
	if strings.TrimSpace(s.deviceTimeout.Text()) == "" {
		s.deviceTimeout.SetText("240")
	}
	if len(s.snapshot.Object.Rooms) == 0 {
		s.snapshot.Object.Rooms = []contracts.CASLRoomDetails{{
			RoomID:      "draft-room-1",
			Name:        "Приміщення",
			Description: "Без опису",
		}}
	}
}

func (s *caslObjectDialogState) exec() (int64, bool) {
	dialog := qt.NewQDialog(s.parent)
	if s.creating {
		dialog.SetWindowTitle("Майстер створення об'єкта CASL")
	} else {
		dialog.SetWindowTitle("Редагування об'єкта CASL")
	}
	dialog.Resize(1040, 760)
	var savedObjectID int64

	tabs := qt.NewQTabWidget2()
	tabs.AddTab(s.buildObjectTab(), "1. Об'єкт")
	tabs.AddTab(s.buildDeviceTab(dialog.QWidget), "2. Прилад")
	tabs.AddTab(s.buildRoomsTab(), "3. Приміщення")
	tabs.AddTab(s.buildLinesTab(), "4. Зони та зв'язки")
	tabs.AddTab(s.buildUsersTab(), "5. Користувачі")
	tabs.AddTab(s.buildImagesTab(), "6. Фото")
	tabs.AddTab(s.buildRegimesTab(), "7. Режими")
	s.wireDirtyTracking()
	s.wirePhoneFormatting()
	s.loadDeviceNumbers()

	saveButton := qt.NewQPushButton3("Створити")
	if !s.creating {
		saveButton.SetText("Зберегти")
	}
	cancelButton := qt.NewQPushButton3("Скасувати")
	allowClose := false
	cancelButton.OnClicked(func() {
		if !s.confirmDiscard(dialog.QWidget) {
			return
		}
		allowClose = true
		dialog.Reject()
	})
	dialog.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		if allowClose || s.confirmDiscard(dialog.QWidget) {
			allowClose = true
			event.Accept()
			super(event)
			return
		}
		event.Ignore()
	})
	saveButton.OnClicked(func() {
		if err := s.readForms(); err != nil {
			s.showError(err)
			return
		}
		saveButton.SetEnabled(false)
		cancelButton.SetEnabled(false)
		s.statusText.SetText("Збереження CASL...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			var (
				objectID int64
				err      error
			)
			if s.creating {
				_, objectID, err = caslobject.CreateDraft(ctx, s.provider, s.snapshot)
			} else {
				objectID, err = s.saveExisting(ctx)
			}
			RunOnMainThread(func() {
				if err != nil {
					saveButton.SetEnabled(true)
					cancelButton.SetEnabled(true)
					s.statusText.SetText("Помилка збереження")
					s.showError(err)
					return
				}
				s.statusText.SetText("Збережено")
				allowClose = true
				savedObjectID = objectID
				dialog.Accept()
			})
		}()
	})

	footer := qt.NewQHBoxLayout2()
	footer.AddWidget(s.statusText.QWidget)
	footer.AddStretch()
	footer.AddWidget(cancelButton.QWidget)
	footer.AddWidget(saveButton.QWidget)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(tabs.QWidget)
	layout.AddLayout(footer.QLayout)
	dialog.SetLayout(layout.QLayout)

	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return 0, false
	}
	if savedObjectID <= 0 {
		return 0, false
	}
	return savedObjectID, true
}

func (s *caslObjectDialogState) wireDirtyTracking() {
	for _, edit := range []*qt.QLineEdit{
		s.name, s.address, s.latitude, s.longitude, s.contract, s.startDate,
		s.note, s.objectType, s.requestID, s.geoZoneID, s.businessCoeff, s.deviceNumber,
		s.deviceName, s.deviceTimeout, s.changeDate, s.reglamentDate, s.sim1, s.sim2,
		s.units, s.requisites, s.licenceKey, s.remotePassword,
	} {
		edit.OnTextEdited(func(_ string) { s.dirty = true })
	}
	for _, edit := range []*qt.QTextEdit{s.description} {
		edit.OnTextChanged(func() { s.dirty = true })
	}
	for _, combo := range []*qt.QComboBox{s.manager, s.status, s.reactingPult, s.deviceType, s.technician} {
		combo.OnCurrentIndexChanged(func(_ int) { s.dirty = true })
	}
	s.deviceNumber.OnTextEdited(func(_ string) {
		s.deviceNumberTimer.Stop()
		s.deviceNumberTimer.Start(350)
	})
}

func (s *caslObjectDialogState) wirePhoneFormatting() {
	for _, field := range []*qt.QLineEdit{s.sim1, s.sim2} {
		field := field
		field.OnEditingFinished(func() {
			formatted, err := caslobject.NormalizeUAPhone(field.Text())
			if err == nil {
				field.SetText(formatted)
			}
		})
	}
}

func (s *caslObjectDialogState) confirmDiscard(parent *qt.QWidget) bool {
	if !s.dirty {
		return true
	}
	return qt.QMessageBox_Question(
		parent,
		"Незбережені зміни",
		"Закрити редактор CASL та відкинути незбережені зміни?",
	) == qt.QMessageBox__Yes
}

func (s *caslObjectDialogState) loadDeviceNumbers() {
	s.deviceNumberStatus.SetText("Завантаження зайнятих номерів...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		numbers, err := s.provider.ReadCASLDeviceNumbers(ctx)
		RunOnMainThread(func() {
			if err != nil {
				s.deviceNumberStatus.SetText("Не вдалося перевірити номери: " + err.Error())
				return
			}
			s.deviceNumbers = make(map[int64]struct{}, len(numbers))
			for _, number := range numbers {
				s.deviceNumbers[number] = struct{}{}
			}
			s.deviceNumbersLoaded = true
			s.updateDeviceNumberSuggestions()
		})
	}()
}

func (s *caslObjectDialogState) updateDeviceNumberSuggestions() {
	raw := strings.TrimSpace(s.deviceNumber.Text())
	number, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || number <= 0 {
		s.deviceNumberStatus.SetText("Введіть додатний номер приладу")
		return
	}
	if !s.deviceNumbersLoaded {
		s.deviceNumberStatus.SetText("Перевірка номера...")
		return
	}
	_, occupied := s.deviceNumbers[number]
	isCurrent := !s.creating && number == s.original.Device.Number
	state := "Номер вільний."
	if occupied && !isCurrent {
		state = "Номер зайнятий."
	} else if isCurrent {
		state = "Поточний номер об'єкта."
	}
	free := make([]string, 0, 8)
	start := max(int64(1), number-5)
	for candidate := start; candidate <= number+12 && len(free) < 8; candidate++ {
		if _, used := s.deviceNumbers[candidate]; !used {
			free = append(free, strconv.FormatInt(candidate, 10))
		}
	}
	s.deviceNumberStatus.SetText(state + " Вільні поруч: " + strings.Join(free, ", "))
}

func (s *caslObjectDialogState) buildObjectTab() *qt.QWidget {
	content := qt.NewQWidget2()
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	s.description.SetMinimumHeight(70)
	form.AddRow3("Назва", s.name.QWidget)
	form.AddRow3("Адреса", s.address.QWidget)
	s.latitude.SetPlaceholderText("Широта")
	s.longitude.SetPlaceholderText("Довгота")
	mapButton := qt.NewQPushButton3("Вибрати на карті")
	mapButton.OnClicked(func() {
		latitude, longitude, accepted := showCASLCoordinatesDialog(
			s.parent,
			s.address.Text(),
			s.latitude.Text(),
			s.longitude.Text(),
		)
		if accepted {
			s.latitude.SetText(latitude)
			s.longitude.SetText(longitude)
			s.dirty = true
		}
	})
	form.AddRow3("Координати", horizontalWidgets(s.latitude.QWidget, s.longitude.QWidget, mapButton.QWidget))
	form.AddRow3("Опис", s.description.QWidget)
	form.AddRow3("Договір", s.contract.QWidget)
	form.AddRow3("Менеджер", s.manager.QWidget)
	form.AddRow3("Примітка", s.note.QWidget)
	form.AddRow3("Дата запуску", s.dateRow(s.startDate))
	form.AddRow3("Статус", s.status.QWidget)
	form.AddRow3("Тип об'єкта", s.objectType.QWidget)
	form.AddRow3("ID заявки", s.requestID.QWidget)
	form.AddRow3("Пульт реагування", s.reactingPult.QWidget)
	form.AddRow3("Geo zone ID", s.geoZoneID.QWidget)
	form.AddRow3("Бізнес-коефіцієнт", s.businessCoeff.QWidget)
	content.SetLayout(form.QLayout)
	return wrapInScrollArea(content)
}

func (s *caslObjectDialogState) buildDeviceTab(dialogParent *qt.QWidget) *qt.QWidget {
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	formWidget := qt.NewQWidget2()
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	s.deviceNumberStatus.SetWordWrap(true)
	form.AddRow3("Номер приладу", stackedWidgets(s.deviceNumber.QWidget, s.deviceNumberStatus.QWidget))
	form.AddRow3("Назва", s.deviceName.QWidget)
	form.AddRow3("Тип", s.deviceType.QWidget)
	form.AddRow3("Timeout, с", s.deviceTimeout.QWidget)
	form.AddRow3("Дата зміни", s.dateRow(s.changeDate))
	form.AddRow3("Дата регламенту", s.dateRow(s.reglamentDate))
	form.AddRow3("SIM 1", s.sim1.QWidget)
	form.AddRow3("SIM 2", s.sim2.QWidget)
	form.AddRow3("Технік", s.technician.QWidget)
	form.AddRow3("Units", s.units.QWidget)
	form.AddRow3("Реквізити", s.requisites.QWidget)
	form.AddRow3("Ліцензійний ключ", s.licenceKey.QWidget)
	form.AddRow3("Пароль віддаленого доступу", s.remotePassword.QWidget)
	formWidget.SetLayout(form.QLayout)
	layout.AddWidget(formWidget)

	if !s.creating {
		blockButton := qt.NewQPushButton3("Блокувати прилад")
		if s.snapshot.Object.DeviceBlocked {
			blockButton.SetText("Розблокувати прилад")
		}
		blockButton.OnClicked(func() {
			if s.snapshot.Object.DeviceBlocked {
				if qt.QMessageBox_Question(dialogParent, "CASL", "Розблокувати прилад об'єкта?") != qt.QMessageBox__Yes {
					return
				}
				s.runDeviceMutation(dialogParent, func(ctx context.Context) error {
					return s.provider.UnblockCASLDevice(ctx, s.snapshot.Object.Device.DeviceID)
				}, "Прилад розблоковано")
				return
			}
			request, ok := showCASLBlockDialog(dialogParent, s.snapshot.Object.Device)
			if !ok {
				return
			}
			s.runDeviceMutation(dialogParent, func(ctx context.Context) error {
				return s.provider.BlockCASLDevice(ctx, request)
			}, "Прилад заблоковано")
		})
		actions := qt.NewQHBoxLayout2()
		actions.AddStretch()
		actions.AddWidget(blockButton.QWidget)
		layout.AddLayout(actions.QLayout)
	}
	layout.AddStretch()
	content.SetLayout(layout.QLayout)
	return wrapInScrollArea(content)
}

func (s *caslObjectDialogState) dateRow(edit *qt.QLineEdit) *qt.QWidget {
	button := qt.NewQPushButton3("Обрати")
	button.SetToolTip("Вибрати дату в календарі")
	button.OnClicked(func() {
		if value, ok := showObjectDatePicker(s.parent, nil, edit.Text()); ok {
			edit.SetText(value)
			s.dirty = true
		}
	})
	edit.SetPlaceholderText("дд.мм.рррр")
	return horizontalWidgets(edit.QWidget, button.QWidget)
}

func (s *caslObjectDialogState) buildRoomsTab() *qt.QWidget {
	s.roomsTable.SetHorizontalHeaderLabels([]string{"ID", "Назва", "Опис", "RTSP"})
	s.roomsTable.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	s.roomsTable.OnCellDoubleClicked(func(row int, _ int) { s.editRoom(row) })
	addButton := qt.NewQPushButton3("Додати")
	editButton := qt.NewQPushButton3("Змінити")
	removeButton := qt.NewQPushButton3("Видалити чернетку")
	addButton.OnClicked(s.addRoom)
	editButton.OnClicked(func() { s.editRoom(s.roomsTable.CurrentRow()) })
	removeButton.OnClicked(s.removeDraftRoom)
	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(editButton.QWidget)
	actions.AddWidget(removeButton.QWidget)
	actions.AddStretch()
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(s.roomsTable.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (s *caslObjectDialogState) buildLinesTab() *qt.QWidget {
	s.linesTable.SetHorizontalHeaderLabels([]string{"№", "Опис", "Тип", "Група", "Адаптер", "№ адапт.", "Блок", "Приміщення"})
	s.linesTable.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	s.linesTable.OnCellDoubleClicked(func(row int, _ int) { s.editLine(row) })
	addButton := qt.NewQPushButton3("Додати")
	editButton := qt.NewQPushButton3("Змінити")
	removeButton := qt.NewQPushButton3("Видалити чернетку")
	addButton.OnClicked(s.addLine)
	editButton.OnClicked(func() { s.editLine(s.linesTable.CurrentRow()) })
	removeButton.OnClicked(s.removeDraftLine)
	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(editButton.QWidget)
	actions.AddWidget(removeButton.QWidget)
	actions.AddStretch()
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(s.linesTable.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (s *caslObjectDialogState) buildUsersTab() *qt.QWidget {
	s.usersTable.SetHorizontalHeaderLabels([]string{"Пріоритет", "Користувач", "Гос. номер", "Роль / телефони"})
	s.usersTable.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	s.usersRoom.OnCurrentIndexChanged(func(_ int) { s.refreshUsersTable() })
	addButton := qt.NewQPushButton3("Додати існуючого")
	createButton := qt.NewQPushButton3("Новий користувач")
	editButton := qt.NewQPushButton3("Змінити гос. номер")
	removeButton := qt.NewQPushButton3("Видалити")
	upButton := qt.NewQPushButton3("Вгору")
	downButton := qt.NewQPushButton3("Вниз")
	addButton.OnClicked(s.addExistingUser)
	createButton.OnClicked(s.createUser)
	editButton.OnClicked(s.editUserHozNumber)
	removeButton.OnClicked(s.removeUser)
	upButton.OnClicked(func() { s.moveUser(-1) })
	downButton.OnClicked(func() { s.moveUser(1) })
	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(createButton.QWidget)
	actions.AddWidget(editButton.QWidget)
	actions.AddWidget(removeButton.QWidget)
	actions.AddWidget(upButton.QWidget)
	actions.AddWidget(downButton.QWidget)
	actions.AddStretch()
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddWidget(s.usersRoom.QWidget)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(s.usersTable.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (s *caslObjectDialogState) buildImagesTab() *qt.QWidget {
	s.imageScope.OnCurrentIndexChanged(func(_ int) { s.refreshImagesList() })
	s.imagesList.OnItemDoubleClicked(func(_ *qt.QListWidgetItem) { s.previewSelectedImage() })
	s.imagesList.SetIconSize(qt.NewQSize2(96, 72))
	addButton := qt.NewQPushButton3("Додати фото")
	previewButton := qt.NewQPushButton3("Переглянути")
	deleteButton := qt.NewQPushButton3("Видалити")
	addButton.OnClicked(s.addImage)
	previewButton.OnClicked(s.previewSelectedImage)
	deleteButton.OnClicked(s.deleteSelectedImage)
	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(previewButton.QWidget)
	actions.AddWidget(deleteButton.QWidget)
	actions.AddStretch()
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddWidget(s.imageScope.QWidget)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(s.imagesList.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (s *caslObjectDialogState) buildRegimesTab() *qt.QWidget {
	tabs := qt.NewQTabWidget2()
	tabs.AddTab(s.buildRegimeTable("Блокування тривог", s.moreAlarmTable, true), "Блокування тривог")
	tabs.AddTab(s.buildRegimeTable("Неробочі години", s.ignoringTable, false), "Неробочі години")
	return tabs.QWidget
}

func (s *caslObjectDialogState) buildRegimeTable(title string, table *qt.QTableWidget, moreAlarm bool) *qt.QWidget {
	table.SetHorizontalHeaderLabels([]string{"День", "Початок", "Кінець", "Причина"})
	table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	table.OnCellDoubleClicked(func(row int, _ int) { s.editRegime(moreAlarm, row) })
	addButton := qt.NewQPushButton3("Додати")
	editButton := qt.NewQPushButton3("Змінити")
	deleteButton := qt.NewQPushButton3("Видалити")
	addButton.OnClicked(func() { s.addRegime(moreAlarm) })
	editButton.OnClicked(func() { s.editRegime(moreAlarm, table.CurrentRow()) })
	deleteButton.OnClicked(func() { s.deleteRegime(moreAlarm, table.CurrentRow()) })
	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(editButton.QWidget)
	actions.AddWidget(deleteButton.QWidget)
	actions.AddStretch()
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddWidget(qt.NewQLabel3(title).QWidget)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(table.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (s *caslObjectDialogState) readForms() error {
	object := &s.snapshot.Object
	var err error
	object.Name = strings.TrimSpace(s.name.Text())
	object.Address = strings.TrimSpace(s.address.Text())
	object.Lat = strings.TrimSpace(s.latitude.Text())
	object.Long = strings.TrimSpace(s.longitude.Text())
	object.Description = strings.TrimSpace(s.description.ToPlainText())
	object.Contract = strings.TrimSpace(s.contract.Text())
	object.ManagerID = selectedComboValue(s.manager, s.managerIDs)
	object.Note = strings.TrimSpace(s.note.Text())
	object.StartDate, err = parseCASLDate(s.startDate.Text())
	if err != nil {
		return fmt.Errorf("дата запуску: %w", err)
	}
	object.ObjectStatus = strings.TrimSpace(s.status.CurrentText())
	object.ObjectType = strings.TrimSpace(s.objectType.Text())
	object.IDRequest = strings.TrimSpace(s.requestID.Text())
	object.ReactingPultID = selectedComboValue(s.reactingPult, s.pultIDs)
	if object.Name == "" || len([]rune(object.Name)) < 2 {
		return fmt.Errorf("вкажіть назву об'єкта")
	}
	if len([]rune(object.Address)) < 5 {
		return fmt.Errorf("адреса має містити щонайменше 5 символів")
	}
	if len([]rune(object.Description)) < 3 {
		return fmt.Errorf("опис має містити щонайменше 3 символи")
	}
	geoZoneID, err := parseOptionalInt64(s.geoZoneID.Text())
	if err != nil {
		return fmt.Errorf("некоректний Geo zone ID")
	}
	object.GeoZoneID = geoZoneID
	coeff, err := parseOptionalFloat(s.businessCoeff.Text())
	if err != nil {
		return fmt.Errorf("некоректний бізнес-коефіцієнт")
	}
	object.BusinessCoeff = coeff

	device := &object.Device
	device.Number, err = parseRequiredInt64(s.deviceNumber.Text(), "вкажіть номер приладу")
	if err != nil {
		return err
	}
	if s.deviceNumbersLoaded {
		_, occupied := s.deviceNumbers[device.Number]
		if occupied && (s.creating || device.Number != s.original.Device.Number) {
			return fmt.Errorf("номер приладу %d вже зайнятий", device.Number)
		}
	}
	device.Name = strings.TrimSpace(s.deviceName.Text())
	device.Type = selectedEditableComboValue(s.deviceType, s.deviceTypes)
	device.Timeout, err = parseRequiredInt64(s.deviceTimeout.Text(), "вкажіть timeout приладу")
	if err != nil {
		return err
	}
	device.SIM1, err = caslobject.NormalizeUAPhone(s.sim1.Text())
	if err != nil {
		return fmt.Errorf("SIM 1: %w", err)
	}
	device.SIM2, err = caslobject.NormalizeUAPhone(s.sim2.Text())
	if err != nil {
		return fmt.Errorf("SIM 2: %w", err)
	}
	device.ChangeDate, err = parseCASLDate(s.changeDate.Text())
	if err != nil {
		return fmt.Errorf("дата зміни: %w", err)
	}
	device.ReglamentDate, err = parseCASLDate(s.reglamentDate.Text())
	if err != nil {
		return fmt.Errorf("дата регламенту: %w", err)
	}
	device.TechnicianID = selectedComboValue(s.technician, s.technicianIDs)
	device.Units = strings.TrimSpace(s.units.Text())
	device.Requisites = strings.TrimSpace(s.requisites.Text())
	device.LicenceKey = strings.TrimSpace(s.licenceKey.Text())
	device.PasswRemote = strings.TrimSpace(s.remotePassword.Text())
	device.MoreAlarmTime = caslRegimesToAny(s.moreAlarmRegimes)
	device.IgnoringAlarmTime = caslRegimesToAny(s.ignoringRegimes)
	if device.Name == "" {
		return fmt.Errorf("вкажіть назву приладу")
	}
	if device.Type == "" {
		return fmt.Errorf("вкажіть тип приладу")
	}
	if len(object.Rooms) == 0 {
		return fmt.Errorf("додайте хоча б одне приміщення")
	}
	return nil
}

func (s *caslObjectDialogState) saveExisting(ctx context.Context) (int64, error) {
	object := s.snapshot.Object
	objectUpdate := caslObjectUpdateFromDetails(object)
	if !reflect.DeepEqual(caslObjectUpdateFromDetails(s.original), objectUpdate) {
		if err := s.provider.UpdateCASLObject(ctx, objectUpdate); err != nil {
			return 0, fmt.Errorf("об'єкт: %w", err)
		}
	}

	device := object.Device
	if device.Number != s.original.Device.Number {
		inUse, err := s.provider.IsCASLDeviceNumberInUse(ctx, device.Number)
		if err != nil {
			return 0, fmt.Errorf("перевірка номера приладу: %w", err)
		}
		if inUse {
			return 0, fmt.Errorf("номер приладу %d вже зайнятий", device.Number)
		}
	}
	deviceUpdate := caslDeviceUpdateFromDetails(device)
	if !reflect.DeepEqual(caslDeviceUpdateFromDetails(s.original.Device), deviceUpdate) {
		if err := s.provider.UpdateCASLDevice(ctx, deviceUpdate); err != nil {
			return 0, fmt.Errorf("прилад: %w", err)
		}
	}

	createdRoom := false
	for _, room := range object.Rooms {
		if strings.HasPrefix(room.RoomID, "draft-room-") {
			if err := s.provider.CreateCASLRoom(ctx, contracts.CASLRoomCreate{
				ObjID: object.ObjID, Name: room.Name, Description: room.Description, RTSP: room.RTSP,
			}); err != nil {
				return 0, fmt.Errorf("приміщення %q: %w", room.Name, err)
			}
			createdRoom = true
			continue
		}
		originalRoom, found := caslRoomByID(s.original.Rooms, room.RoomID)
		if found && caslRoomMetadataEqual(originalRoom, room) {
			continue
		}
		if err := s.provider.UpdateCASLRoom(ctx, contracts.CASLRoomUpdate{
			ObjID: object.ObjID, RoomID: room.RoomID, Name: room.Name, Description: room.Description, RTSP: room.RTSP,
		}); err != nil {
			return 0, fmt.Errorf("приміщення %q: %w", room.Name, err)
		}
	}

	for _, line := range device.Lines {
		mutation := contracts.CASLDeviceLineMutation{
			DeviceID: device.DeviceID, LineID: line.LineID, LineNumber: line.LineNumber,
			GroupNumber: line.GroupNumber, AdapterType: line.AdapterType, AdapterNumber: line.AdapterNumber,
			Description: line.Description, LineType: line.LineType, IsBlocked: line.IsBlocked,
		}
		originalLine, found := caslOriginalLine(line, s.original.Device.Lines)
		if !found {
			if err := s.provider.CreateCASLDeviceLine(ctx, mutation); err != nil {
				return 0, fmt.Errorf("зона #%d: %w", line.LineNumber, err)
			}
		} else if !caslLineDefinitionEqual(originalLine, line) {
			if err := s.provider.UpdateCASLDeviceLine(ctx, mutation); err != nil {
				return 0, fmt.Errorf("зона #%d: %w", line.LineNumber, err)
			}
		}
	}

	roomIDsByName := make(map[string]string, len(object.Rooms))
	for _, room := range object.Rooms {
		if !strings.HasPrefix(room.RoomID, "draft-room-") {
			roomIDsByName[strings.ToLower(strings.TrimSpace(room.Name))] = room.RoomID
		}
	}
	if createdRoom {
		reloaded, err := s.provider.GetCASLObjectEditorSnapshot(ctx, parseCASLObjectID(object.ObjID))
		if err != nil {
			return 0, fmt.Errorf("оновлення приміщень: %w", err)
		}
		for _, room := range reloaded.Object.Rooms {
			roomIDsByName[strings.ToLower(strings.TrimSpace(room.Name))] = room.RoomID
		}
	}

	for _, line := range device.Lines {
		desiredRoomID := strings.TrimSpace(line.RoomID)
		if desiredRoomID == "" {
			desiredRoomID = caslRoomIDForLine(object.Rooms, line.LineNumber)
		}
		if strings.HasPrefix(desiredRoomID, "draft-room-") {
			desiredRoomID = roomIDsByName[strings.ToLower(strings.TrimSpace(s.roomName(line.RoomID)))]
		}
		originalLine, existed := caslOriginalLine(line, s.original.Device.Lines)
		originalRoomID := ""
		originalLineNumber := line.LineNumber
		if existed {
			originalLineNumber = originalLine.LineNumber
			originalRoomID = caslRoomIDForLine(s.original.Rooms, originalLineNumber)
			if originalRoomID == "" {
				originalRoomID = strings.TrimSpace(originalLine.RoomID)
			}
		}
		bindingChanged := !existed || originalRoomID != desiredRoomID || originalLineNumber != line.LineNumber
		if !bindingChanged {
			continue
		}
		if originalRoomID != "" {
			if err := s.provider.RemoveCASLLineFromRoom(ctx, contracts.CASLLineToRoomBinding{
				ObjID: object.ObjID, DeviceID: device.DeviceID, LineNumber: originalLineNumber, RoomID: originalRoomID,
			}); err != nil {
				return 0, fmt.Errorf("відв'язка зони #%d: %w", originalLineNumber, err)
			}
		}
		if desiredRoomID == "" {
			continue
		}
		if err := s.provider.AddCASLLineToRoom(ctx, contracts.CASLLineToRoomBinding{
			ObjID: object.ObjID, DeviceID: device.DeviceID, LineNumber: line.LineNumber, RoomID: desiredRoomID,
		}); err != nil {
			return 0, fmt.Errorf("прив'язка зони #%d: %w", line.LineNumber, err)
		}
	}

	if caslRoomUsersChanged(s.original.Rooms, object.Rooms) {
		if err := s.saveRoomUsers(ctx, roomIDsByName); err != nil {
			return 0, err
		}
	}
	if caslImagesChanged(s.original, object) {
		if err := s.saveImages(ctx, roomIDsByName); err != nil {
			return 0, err
		}
	}
	return parseRequiredInt64(object.ObjID, "CASL повернув некоректний obj_id")
}

func caslObjectUpdateFromDetails(object contracts.CASLGuardObjectDetails) contracts.CASLGuardObjectUpdate {
	return contracts.CASLGuardObjectUpdate{
		ObjID: object.ObjID, Name: object.Name, Address: object.Address, Long: object.Long, Lat: object.Lat,
		Description: object.Description, Contract: object.Contract, ManagerID: object.ManagerID, Note: object.Note,
		StartDate: object.StartDate, Status: object.ObjectStatus, ObjectType: object.ObjectType, IDRequest: object.IDRequest,
		ReactingPultID: object.ReactingPultID, GeoZoneID: object.GeoZoneID, BusinessCoeff: object.BusinessCoeff,
	}
}

func caslDeviceUpdateFromDetails(device contracts.CASLDeviceDetails) contracts.CASLDeviceUpdate {
	return contracts.CASLDeviceUpdate{
		DeviceID: device.DeviceID, Number: device.Number, Name: device.Name, DeviceType: device.Type,
		Timeout: device.Timeout, SIM1: device.SIM1, SIM2: device.SIM2, TechnicianID: device.TechnicianID,
		Units: device.Units, Requisites: device.Requisites, ChangeDate: device.ChangeDate,
		ReglamentDate: device.ReglamentDate, MoreAlarmTime: device.MoreAlarmTime,
		IgnoringAlarmTime: device.IgnoringAlarmTime, LicenceKey: device.LicenceKey, PasswRemote: device.PasswRemote,
	}
}

func caslRoomByID(rooms []contracts.CASLRoomDetails, roomID string) (contracts.CASLRoomDetails, bool) {
	for _, room := range rooms {
		if room.RoomID == roomID {
			return room, true
		}
	}
	return contracts.CASLRoomDetails{}, false
}

func caslRoomIDForLine(rooms []contracts.CASLRoomDetails, lineNumber int) string {
	for _, room := range rooms {
		for _, line := range room.Lines {
			if line.LineNumber == lineNumber {
				return strings.TrimSpace(room.RoomID)
			}
		}
	}
	return ""
}

func caslRoomMetadataEqual(left, right contracts.CASLRoomDetails) bool {
	return strings.TrimSpace(left.Name) == strings.TrimSpace(right.Name) &&
		strings.TrimSpace(left.Description) == strings.TrimSpace(right.Description) &&
		strings.TrimSpace(left.RTSP) == strings.TrimSpace(right.RTSP)
}

func caslOriginalLine(line contracts.CASLDeviceLineDetails, originals []contracts.CASLDeviceLineDetails) (contracts.CASLDeviceLineDetails, bool) {
	for _, original := range originals {
		if line.LineID != nil && original.LineID != nil && *line.LineID == *original.LineID {
			return original, true
		}
	}
	for _, original := range originals {
		if original.LineNumber == line.LineNumber {
			return original, true
		}
	}
	return contracts.CASLDeviceLineDetails{}, false
}

func caslLineDefinitionEqual(left, right contracts.CASLDeviceLineDetails) bool {
	return left.LineNumber == right.LineNumber &&
		left.GroupNumber == right.GroupNumber &&
		strings.TrimSpace(left.AdapterType) == strings.TrimSpace(right.AdapterType) &&
		left.AdapterNumber == right.AdapterNumber &&
		strings.TrimSpace(left.Description) == strings.TrimSpace(right.Description) &&
		strings.TrimSpace(left.LineType) == strings.TrimSpace(right.LineType) &&
		left.IsBlocked == right.IsBlocked
}

func caslRoomUsersChanged(original, current []contracts.CASLRoomDetails) bool {
	for _, room := range current {
		if strings.HasPrefix(room.RoomID, "draft-room-") && len(room.Users) > 0 {
			return true
		}
		previous, found := caslRoomByID(original, room.RoomID)
		if !found || !reflect.DeepEqual(previous.Users, room.Users) {
			return true
		}
	}
	return false
}

func caslImagesChanged(original, current contracts.CASLGuardObjectDetails) bool {
	if !reflect.DeepEqual(original.Images, current.Images) {
		return true
	}
	for _, room := range current.Rooms {
		previous, found := caslRoomByID(original.Rooms, room.RoomID)
		if !found {
			return len(room.Images) > 0
		}
		if !reflect.DeepEqual(previous.Images, room.Images) {
			return true
		}
	}
	return false
}

func (s *caslObjectDialogState) refreshRooms() {
	s.roomsTable.SetRowCount(len(s.snapshot.Object.Rooms))
	for row, room := range s.snapshot.Object.Rooms {
		s.roomsTable.SetItem(row, 0, qt.NewQTableWidgetItem2(room.RoomID))
		s.roomsTable.SetItem(row, 1, qt.NewQTableWidgetItem2(room.Name))
		s.roomsTable.SetItem(row, 2, qt.NewQTableWidgetItem2(room.Description))
		s.roomsTable.SetItem(row, 3, qt.NewQTableWidgetItem2(room.RTSP))
	}
	for column, width := range []int{110, 220, 360, 220} {
		s.roomsTable.SetColumnWidth(column, width)
	}
}

func (s *caslObjectDialogState) refreshLines() {
	lines := s.snapshot.Object.Device.Lines
	s.linesTable.SetRowCount(len(lines))
	for row, line := range lines {
		roomName := s.roomName(line.RoomID)
		values := []string{
			strconv.Itoa(line.LineNumber), line.Description, line.LineType, strconv.Itoa(line.GroupNumber),
			line.AdapterType, strconv.Itoa(line.AdapterNumber), yesNo(line.IsBlocked), roomName,
		}
		for column, value := range values {
			s.linesTable.SetItem(row, column, qt.NewQTableWidgetItem2(value))
		}
	}
	for column, width := range []int{60, 250, 120, 70, 110, 80, 70, 200} {
		s.linesTable.SetColumnWidth(column, width)
	}
}

func (s *caslObjectDialogState) addRoom() {
	room, ok := showCASLRoomDialog(s.parent, contracts.CASLRoomDetails{})
	if !ok {
		return
	}
	room.RoomID = fmt.Sprintf("draft-room-%d", time.Now().UnixNano())
	s.snapshot.Object.Rooms = append(s.snapshot.Object.Rooms, room)
	s.dirty = true
	s.refreshRooms()
	s.refreshLines()
	s.refreshUsers()
	s.refreshImageScopes()
}

func (s *caslObjectDialogState) editRoom(row int) {
	if row < 0 || row >= len(s.snapshot.Object.Rooms) {
		return
	}
	room, ok := showCASLRoomDialog(s.parent, s.snapshot.Object.Rooms[row])
	if !ok {
		return
	}
	s.snapshot.Object.Rooms[row] = room
	s.dirty = true
	s.refreshRooms()
	s.refreshLines()
	s.refreshUsers()
	s.refreshImageScopes()
}

func (s *caslObjectDialogState) removeDraftRoom() {
	row := s.roomsTable.CurrentRow()
	if row < 0 || row >= len(s.snapshot.Object.Rooms) {
		return
	}
	room := s.snapshot.Object.Rooms[row]
	if !strings.HasPrefix(room.RoomID, "draft-room-") {
		s.showError(fmt.Errorf("CASL API не надає безпечного видалення існуючого приміщення"))
		return
	}
	s.snapshot.Object.Rooms = append(s.snapshot.Object.Rooms[:row], s.snapshot.Object.Rooms[row+1:]...)
	s.dirty = true
	for index := range s.snapshot.Object.Device.Lines {
		if s.snapshot.Object.Device.Lines[index].RoomID == room.RoomID {
			s.snapshot.Object.Device.Lines[index].RoomID = ""
		}
	}
	s.refreshRooms()
	s.refreshLines()
	s.refreshUsers()
	s.refreshImageScopes()
}

func (s *caslObjectDialogState) refreshUsers() {
	selectedID := ""
	if index := s.usersRoom.CurrentIndex(); index >= 0 {
		selectedID = s.usersRoom.ItemData(index).ToString()
	}
	s.usersRoom.Clear()
	for _, room := range s.snapshot.Object.Rooms {
		s.usersRoom.AddItem3(room.Name, qt.NewQVariant14(room.RoomID))
	}
	if selectedID != "" {
		for index := 0; index < s.usersRoom.Count(); index++ {
			if s.usersRoom.ItemData(index).ToString() == selectedID {
				s.usersRoom.SetCurrentIndex(index)
				break
			}
		}
	}
	s.refreshUsersTable()
}

func (s *caslObjectDialogState) refreshUsersTable() {
	roomIndex := s.currentUsersRoomIndex()
	if roomIndex < 0 {
		s.usersTable.SetRowCount(0)
		return
	}
	users := s.snapshot.Object.Rooms[roomIndex].Users
	s.usersTable.SetRowCount(len(users))
	for row, link := range users {
		profile := s.userByID(link.UserID)
		name := strings.TrimSpace(strings.Join([]string{profile.LastName, profile.FirstName, profile.MiddleName}, " "))
		details := strings.TrimSpace(profile.Role + " | " + caslPhonesText(profile.PhoneNumbers))
		values := []string{strconv.Itoa(row + 1), name, link.HozNum, details}
		for column, value := range values {
			s.usersTable.SetItem(row, column, qt.NewQTableWidgetItem2(value))
		}
	}
	for column, width := range []int{90, 280, 130, 360} {
		s.usersTable.SetColumnWidth(column, width)
	}
}

func (s *caslObjectDialogState) addExistingUser() {
	roomIndex := s.currentUsersRoomIndex()
	if roomIndex < 0 {
		return
	}
	combo := qt.NewQComboBox2()
	combo.SetEditable(true)
	combo.SetPlaceholderText("Пошук за ПІБ, ID або телефоном")
	userIDs := make(map[string]string, len(s.snapshot.Users))
	options := make([]string, 0, len(s.snapshot.Users))
	for _, user := range s.snapshot.Users {
		label := strings.TrimSpace(strings.Join([]string{user.LastName, user.FirstName, user.MiddleName}, " "))
		if label == "" {
			label = user.UserID
		}
		label += " (" + user.UserID + ") " + caslPhonesText(user.PhoneNumbers)
		combo.AddItem(label)
		userIDs[label] = user.UserID
		options = append(options, label)
	}
	completer := qt.NewQCompleter3(options)
	completer.SetCaseSensitivity(qt.CaseInsensitive)
	completer.SetFilterMode(qt.MatchContains)
	completer.SetCompletionMode(qt.QCompleter__PopupCompletion)
	combo.SetCompleter(completer)
	dialog := qt.NewQDialog(s.parent)
	dialog.SetWindowTitle("Додати користувача CASL")
	form := qt.NewQFormLayout2()
	form.AddRow3("Користувач", combo.QWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return
	}
	userID := userIDs[combo.CurrentText()]
	if userID == "" {
		qt.QMessageBox_Information(s.parent, "CASL", "Оберіть користувача зі списку результатів пошуку.")
		return
	}
	for _, link := range s.snapshot.Object.Rooms[roomIndex].Users {
		if link.UserID == userID {
			return
		}
	}
	s.snapshot.Object.Rooms[roomIndex].Users = append(s.snapshot.Object.Rooms[roomIndex].Users, contracts.CASLRoomUserLink{
		UserID: userID, Priority: len(s.snapshot.Object.Rooms[roomIndex].Users) + 1,
	})
	s.dirty = true
	s.refreshUsersTable()
}

func (s *caslObjectDialogState) createUser() {
	roomIndex := s.currentUsersRoomIndex()
	if roomIndex < 0 {
		return
	}
	request, accepted := showCASLUserDialog(s.parent)
	if !accepted {
		return
	}
	s.statusText.SetText("Створення користувача CASL...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		user, err := s.provider.CreateCASLUser(ctx, request)
		RunOnMainThread(func() {
			if err != nil {
				s.statusText.SetText("Помилка створення користувача")
				s.showError(err)
				return
			}
			s.snapshot.Users = append(s.snapshot.Users, user)
			s.snapshot.Object.Rooms[roomIndex].Users = append(
				s.snapshot.Object.Rooms[roomIndex].Users,
				contracts.CASLRoomUserLink{
					UserID: user.UserID, Priority: len(s.snapshot.Object.Rooms[roomIndex].Users) + 1,
				},
			)
			s.dirty = true
			s.refreshUsersTable()
			s.statusText.SetText("Користувача створено")
		})
	}()
}

func (s *caslObjectDialogState) editUserHozNumber() {
	roomIndex := s.currentUsersRoomIndex()
	row := s.usersTable.CurrentRow()
	if roomIndex < 0 || row < 0 || row >= len(s.snapshot.Object.Rooms[roomIndex].Users) {
		return
	}
	current := s.snapshot.Object.Rooms[roomIndex].Users[row]
	value := qt.QInputDialog_GetText(s.parent, "Гос. номер", "Номер користувача:")
	if strings.TrimSpace(value) == "" {
		return
	}
	current.HozNum = digitsOnly(value)
	s.snapshot.Object.Rooms[roomIndex].Users[row] = current
	s.dirty = true
	s.refreshUsersTable()
}

func (s *caslObjectDialogState) removeUser() {
	roomIndex := s.currentUsersRoomIndex()
	row := s.usersTable.CurrentRow()
	if roomIndex < 0 || row < 0 || row >= len(s.snapshot.Object.Rooms[roomIndex].Users) {
		return
	}
	users := s.snapshot.Object.Rooms[roomIndex].Users
	s.snapshot.Object.Rooms[roomIndex].Users = append(users[:row], users[row+1:]...)
	s.dirty = true
	s.normalizeUserPriorities(roomIndex)
	s.refreshUsersTable()
}

func (s *caslObjectDialogState) moveUser(offset int) {
	roomIndex := s.currentUsersRoomIndex()
	row := s.usersTable.CurrentRow()
	target := row + offset
	if roomIndex < 0 || row < 0 || target < 0 || target >= len(s.snapshot.Object.Rooms[roomIndex].Users) {
		return
	}
	users := s.snapshot.Object.Rooms[roomIndex].Users
	users[row], users[target] = users[target], users[row]
	s.snapshot.Object.Rooms[roomIndex].Users = users
	s.dirty = true
	s.normalizeUserPriorities(roomIndex)
	s.refreshUsersTable()
	s.usersTable.SetCurrentCell(target, 0)
}

func (s *caslObjectDialogState) normalizeUserPriorities(roomIndex int) {
	for index := range s.snapshot.Object.Rooms[roomIndex].Users {
		s.snapshot.Object.Rooms[roomIndex].Users[index].Priority = index + 1
	}
}

func (s *caslObjectDialogState) currentUsersRoomIndex() int {
	roomID := ""
	if index := s.usersRoom.CurrentIndex(); index >= 0 {
		roomID = s.usersRoom.ItemData(index).ToString()
	}
	for index, room := range s.snapshot.Object.Rooms {
		if room.RoomID == roomID {
			return index
		}
	}
	return -1
}

func (s *caslObjectDialogState) userByID(userID string) contracts.CASLUserProfile {
	for _, user := range s.snapshot.Users {
		if user.UserID == userID {
			return user
		}
	}
	return contracts.CASLUserProfile{UserID: userID}
}

func (s *caslObjectDialogState) saveRoomUsers(ctx context.Context, roomIDsByName map[string]string) error {
	originalByRoom := make(map[string][]contracts.CASLRoomUserLink, len(s.original.Rooms))
	for _, room := range s.original.Rooms {
		originalByRoom[room.RoomID] = room.Users
	}
	for _, room := range s.snapshot.Object.Rooms {
		roomID := room.RoomID
		if strings.HasPrefix(roomID, "draft-room-") {
			roomID = roomIDsByName[strings.ToLower(strings.TrimSpace(room.Name))]
		}
		if roomID == "" {
			continue
		}
		original := originalByRoom[room.RoomID]
		currentIDs := make(map[string]struct{}, len(room.Users))
		for _, link := range room.Users {
			currentIDs[link.UserID] = struct{}{}
		}
		for _, link := range original {
			if _, exists := currentIDs[link.UserID]; exists {
				continue
			}
			if err := s.provider.RemoveCASLUserFromRoom(ctx, contracts.CASLRemoveUserFromRoomRequest{
				ObjID: s.snapshot.Object.ObjID, RoomID: roomID, UserID: link.UserID,
			}); err != nil {
				return fmt.Errorf("видалення користувача з %q: %w", room.Name, err)
			}
		}
		originalIDs := make(map[string]struct{}, len(original))
		for _, link := range original {
			originalIDs[link.UserID] = struct{}{}
		}
		priorities := make([]contracts.CASLRoomUserPriority, 0, len(room.Users))
		for index, link := range room.Users {
			if _, exists := originalIDs[link.UserID]; !exists {
				if err := s.provider.AddCASLUserToRoom(ctx, contracts.CASLAddUserToRoomRequest{
					ObjID: s.snapshot.Object.ObjID, RoomID: roomID, UserID: link.UserID,
					Priority: index + 1, HozNum: link.HozNum,
				}); err != nil {
					return fmt.Errorf("додавання користувача до %q: %w", room.Name, err)
				}
			}
			priorities = append(priorities, contracts.CASLRoomUserPriority{
				UserID: link.UserID, RoomID: roomID, Priority: index + 1, HozNum: link.HozNum,
			})
		}
		if len(priorities) > 0 {
			if err := s.provider.UpdateCASLRoomUserPriorities(ctx, parseCASLObjectID(s.snapshot.Object.ObjID), priorities); err != nil {
				return fmt.Errorf("порядок користувачів у %q: %w", room.Name, err)
			}
		}
	}
	return nil
}

func (s *caslObjectDialogState) refreshRegimeTables() {
	s.refreshRegimeTable(s.moreAlarmTable, s.moreAlarmRegimes)
	s.refreshRegimeTable(s.ignoringTable, s.ignoringRegimes)
}

func (s *caslObjectDialogState) refreshRegimeTable(table *qt.QTableWidget, regimes []caslRegime) {
	table.SetRowCount(len(regimes))
	for row, regime := range regimes {
		values := []string{caslWeekdayName(regime.Day), regime.StartTime, regime.StopTime, regime.Cause}
		for column, value := range values {
			table.SetItem(row, column, qt.NewQTableWidgetItem2(value))
		}
	}
	for column, width := range []int{130, 100, 100, 300} {
		table.SetColumnWidth(column, width)
	}
}

func (s *caslObjectDialogState) addRegime(moreAlarm bool) {
	causes := s.regimeCauses(moreAlarm)
	regime, accepted := showCASLRegimeDialog(s.parent, caslRegime{Day: 1}, causes)
	if !accepted {
		return
	}
	if moreAlarm {
		s.moreAlarmRegimes = append(s.moreAlarmRegimes, regime)
	} else {
		s.ignoringRegimes = append(s.ignoringRegimes, regime)
	}
	s.dirty = true
	s.refreshRegimeTables()
}

func (s *caslObjectDialogState) editRegime(moreAlarm bool, row int) {
	regimes := &s.ignoringRegimes
	if moreAlarm {
		regimes = &s.moreAlarmRegimes
	}
	if row < 0 || row >= len(*regimes) {
		return
	}
	updated, accepted := showCASLRegimeDialog(s.parent, (*regimes)[row], s.regimeCauses(moreAlarm))
	if !accepted {
		return
	}
	(*regimes)[row] = updated
	s.dirty = true
	s.refreshRegimeTables()
}

func (s *caslObjectDialogState) deleteRegime(moreAlarm bool, row int) {
	regimes := &s.ignoringRegimes
	if moreAlarm {
		regimes = &s.moreAlarmRegimes
	}
	if row < 0 || row >= len(*regimes) {
		return
	}
	*regimes = append((*regimes)[:row], (*regimes)[row+1:]...)
	s.dirty = true
	s.refreshRegimeTables()
}

func (s *caslObjectDialogState) regimeCauses(moreAlarm bool) []string {
	key := "off_hours_causes"
	if moreAlarm {
		key = "block_causes"
	}
	labels, values := caslobject.DictionaryOptions(s.snapshot.Dictionary, key)
	if len(labels) == 0 {
		return nil
	}
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		result = append(result, values[label])
	}
	slices.Sort(result)
	return result
}

func (s *caslObjectDialogState) refreshImageScopes() {
	selected := s.currentImageScope()
	s.imageScope.Clear()
	s.imageScope.AddItem3("Об'єкт", qt.NewQVariant14("object"))
	for _, room := range s.snapshot.Object.Rooms {
		s.imageScope.AddItem3("Приміщення: "+room.Name, qt.NewQVariant14(room.RoomID))
	}
	for index := 0; index < s.imageScope.Count(); index++ {
		if s.imageScope.ItemData(index).ToString() == selected {
			s.imageScope.SetCurrentIndex(index)
			break
		}
	}
	s.refreshImagesList()
}

func (s *caslObjectDialogState) refreshImagesList() {
	s.imageLoadSeq++
	loadSeq := s.imageLoadSeq
	s.imagesList.Clear()
	for index, raw := range s.currentImages() {
		label := strings.TrimSpace(raw)
		if strings.HasPrefix(strings.ToLower(label), "data:") {
			label = fmt.Sprintf("Нове фото %d", index+1)
		} else {
			label = "CASL image_id: " + label
		}
		item := qt.NewQListWidgetItem2(label)
		item.SetSizeHint(qt.NewQSize2(0, 80))
		s.imagesList.AddItemWithItem(item)
		go s.loadImageThumbnail(loadSeq, index, raw)
	}
}

func (s *caslObjectDialogState) loadImageThumbnail(loadSeq, row int, raw string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	body, err := s.imageBytes(ctx, raw)
	if err != nil {
		return
	}
	RunOnMainThread(func() {
		if loadSeq != s.imageLoadSeq || row < 0 || row >= s.imagesList.Count() {
			return
		}
		pixmap := qt.NewQPixmap()
		if !pixmap.LoadFromDataWithData(body) {
			return
		}
		thumbnail := pixmap.Scaled3(96, 72, qt.KeepAspectRatio, qt.SmoothTransformation)
		s.imagesList.Item(row).SetIcon(qt.NewQIcon2(thumbnail))
	})
}

func (s *caslObjectDialogState) addImage() {
	filename := qt.QFileDialog_GetOpenFileName()
	if strings.TrimSpace(filename) == "" {
		return
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		s.showError(err)
		return
	}
	imageType := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if imageType == "jpeg" {
		imageType = "jpg"
	}
	switch imageType {
	case "jpg", "png", "webp", "gif", "bmp":
	default:
		s.showError(fmt.Errorf("підтримуються JPG, PNG, WEBP, GIF і BMP"))
		return
	}
	images := append(slices.Clone(s.currentImages()), "data:image/"+imageType+";base64,"+base64.StdEncoding.EncodeToString(body))
	s.setCurrentImages(images)
	s.dirty = true
	s.refreshImagesList()
}

func (s *caslObjectDialogState) deleteSelectedImage() {
	row := s.imagesList.CurrentRow()
	images := s.currentImages()
	if row < 0 || row >= len(images) {
		return
	}
	images = append(slices.Clone(images[:row]), images[row+1:]...)
	s.setCurrentImages(images)
	s.dirty = true
	s.refreshImagesList()
}

func (s *caslObjectDialogState) previewSelectedImage() {
	row := s.imagesList.CurrentRow()
	images := s.currentImages()
	if row < 0 || row >= len(images) {
		return
	}
	raw := images[row]
	s.statusText.SetText("Завантаження фото...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		body, err := s.imageBytes(ctx, raw)
		RunOnMainThread(func() {
			if err != nil {
				s.statusText.SetText("Не вдалося завантажити фото")
				s.showError(err)
				return
			}
			showCASLImagePreview(s.parent, body)
			s.statusText.SetText("Готово")
		})
	}()
}

func (s *caslObjectDialogState) imageBytes(ctx context.Context, raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "data:") {
		comma := strings.Index(raw, ",")
		if comma < 0 {
			return nil, fmt.Errorf("некоректне локальне фото")
		}
		return base64.StdEncoding.DecodeString(strings.TrimSpace(raw[comma+1:]))
	}
	return s.provider.FetchCASLImagePreview(ctx, raw)
}

func (s *caslObjectDialogState) currentImageScope() string {
	index := s.imageScope.CurrentIndex()
	if index < 0 {
		return "object"
	}
	return s.imageScope.ItemData(index).ToString()
}

func (s *caslObjectDialogState) currentImages() []string {
	scope := s.currentImageScope()
	if scope == "object" {
		return s.snapshot.Object.Images
	}
	for _, room := range s.snapshot.Object.Rooms {
		if room.RoomID == scope {
			return room.Images
		}
	}
	return nil
}

func (s *caslObjectDialogState) setCurrentImages(images []string) {
	scope := s.currentImageScope()
	if scope == "object" {
		s.snapshot.Object.Images = images
		return
	}
	for index := range s.snapshot.Object.Rooms {
		if s.snapshot.Object.Rooms[index].RoomID == scope {
			s.snapshot.Object.Rooms[index].Images = images
			return
		}
	}
}

func (s *caslObjectDialogState) saveImages(ctx context.Context, roomIDsByName map[string]string) error {
	if err := s.reconcileImages(ctx, "", s.original.Images, s.snapshot.Object.Images); err != nil {
		return fmt.Errorf("фото об'єкта: %w", err)
	}
	originalRooms := make(map[string]contracts.CASLRoomDetails, len(s.original.Rooms))
	for _, room := range s.original.Rooms {
		originalRooms[room.RoomID] = room
	}
	for _, room := range s.snapshot.Object.Rooms {
		roomID := room.RoomID
		if strings.HasPrefix(roomID, "draft-room-") {
			roomID = roomIDsByName[strings.ToLower(strings.TrimSpace(room.Name))]
		}
		if err := s.reconcileImages(ctx, roomID, originalRooms[room.RoomID].Images, room.Images); err != nil {
			return fmt.Errorf("фото приміщення %q: %w", room.Name, err)
		}
	}
	return nil
}

func (s *caslObjectDialogState) reconcileImages(ctx context.Context, roomID string, original []string, current []string) error {
	currentExisting := make(map[string]struct{}, len(current))
	for _, raw := range current {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "data:") {
			currentExisting[strings.TrimSpace(raw)] = struct{}{}
		}
	}
	for _, imageID := range original {
		imageID = strings.TrimSpace(imageID)
		if _, exists := currentExisting[imageID]; exists {
			continue
		}
		if err := s.provider.DeleteCASLImage(ctx, contracts.CASLImageDeleteRequest{
			ObjID: s.snapshot.Object.ObjID, RoomID: roomID, ImageID: imageID,
		}); err != nil {
			return err
		}
	}
	for _, raw := range current {
		imageType, payload, ok := caslobject.DraftImage(raw)
		if !ok {
			continue
		}
		if err := s.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
			ObjID: s.snapshot.Object.ObjID, RoomID: roomID, ImageType: imageType, ImageData: payload,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *caslObjectDialogState) addLine() {
	line, ok := showCASLLineDialog(s.parent, contracts.CASLDeviceLineDetails{
		LineNumber:  s.nextLineNumber(),
		GroupNumber: 1,
		AdapterType: "SYS",
		LineType:    "NORMAL",
	}, s.snapshot.Object.Rooms, s.lineTypes, s.allowedAdapterTypes())
	if !ok {
		return
	}
	if s.lineNumberExists(line.LineNumber, -1) {
		s.showError(fmt.Errorf("зона #%d вже існує", line.LineNumber))
		return
	}
	s.snapshot.Object.Device.Lines = append(s.snapshot.Object.Device.Lines, line)
	s.dirty = true
	s.syncRoomLine(line)
	s.refreshLines()
}

func (s *caslObjectDialogState) editLine(row int) {
	if row < 0 || row >= len(s.snapshot.Object.Device.Lines) {
		return
	}
	line, ok := showCASLLineDialog(s.parent, s.snapshot.Object.Device.Lines[row], s.snapshot.Object.Rooms, s.lineTypes, s.allowedAdapterTypes())
	if !ok {
		return
	}
	if s.lineNumberExists(line.LineNumber, row) {
		s.showError(fmt.Errorf("зона #%d вже існує", line.LineNumber))
		return
	}
	s.snapshot.Object.Device.Lines[row] = line
	s.dirty = true
	s.syncRoomLine(line)
	s.refreshLines()
}

func (s *caslObjectDialogState) removeDraftLine() {
	row := s.linesTable.CurrentRow()
	if row < 0 || row >= len(s.snapshot.Object.Device.Lines) {
		return
	}
	line := s.snapshot.Object.Device.Lines[row]
	if line.LineID != nil {
		s.showError(fmt.Errorf("CASL API не надає безпечного видалення існуючої зони"))
		return
	}
	s.snapshot.Object.Device.Lines = append(s.snapshot.Object.Device.Lines[:row], s.snapshot.Object.Device.Lines[row+1:]...)
	s.dirty = true
	for index := range s.snapshot.Object.Rooms {
		filtered := s.snapshot.Object.Rooms[index].Lines[:0]
		for _, link := range s.snapshot.Object.Rooms[index].Lines {
			if link.LineNumber != line.LineNumber {
				filtered = append(filtered, link)
			}
		}
		s.snapshot.Object.Rooms[index].Lines = filtered
	}
	s.refreshLines()
}

func (s *caslObjectDialogState) syncRoomLine(line contracts.CASLDeviceLineDetails) {
	for index := range s.snapshot.Object.Rooms {
		room := &s.snapshot.Object.Rooms[index]
		filtered := room.Lines[:0]
		for _, link := range room.Lines {
			if link.LineNumber != line.LineNumber {
				filtered = append(filtered, link)
			}
		}
		room.Lines = filtered
		if room.RoomID == line.RoomID {
			room.Lines = append(room.Lines, contracts.CASLRoomLineLink{
				LineNumber: line.LineNumber, AdapterType: line.AdapterType,
				GroupNumber: line.GroupNumber, AdapterNumber: line.AdapterNumber,
			})
		}
	}
}

func (s *caslObjectDialogState) roomName(roomID string) string {
	for _, room := range s.snapshot.Object.Rooms {
		if room.RoomID == roomID {
			return room.Name
		}
	}
	return ""
}

func (s *caslObjectDialogState) allowedAdapterTypes() map[string]string {
	result := map[string]string{}
	deviceType := selectedEditableComboValue(s.deviceType, s.deviceTypes)
	for _, raw := range caslobject.AdapterTypesForDevice(deviceType) {
		result[raw] = raw
	}
	return result
}

func (s *caslObjectDialogState) nextLineNumber() int {
	maximum := 0
	for _, line := range s.snapshot.Object.Device.Lines {
		if line.LineNumber > maximum {
			maximum = line.LineNumber
		}
	}
	return maximum + 1
}

func (s *caslObjectDialogState) lineNumberExists(number int, except int) bool {
	for index, line := range s.snapshot.Object.Device.Lines {
		if index != except && line.LineNumber == number {
			return true
		}
	}
	return false
}

func (s *caslObjectDialogState) runDeviceMutation(parent *qt.QWidget, mutation func(context.Context) error, success string) {
	s.statusText.SetText("Виконується операція CASL...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := mutation(ctx)
		RunOnMainThread(func() {
			if err != nil {
				s.statusText.SetText("Помилка")
				qt.QMessageBox_Critical(parent, "CASL", err.Error())
				return
			}
			s.statusText.SetText(success)
			qt.QMessageBox_Information(parent, "CASL", success)
		})
	}()
}

func (s *caslObjectDialogState) showError(err error) {
	if err != nil {
		qt.QMessageBox_Critical(s.parent, "CASL", err.Error())
	}
}

func showCASLRoomDialog(parent *qt.QWidget, initial contracts.CASLRoomDetails) (contracts.CASLRoomDetails, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Приміщення CASL")
	name := newLineEdit(initial.Name)
	description := newLineEdit(initial.Description)
	rtsp := newLineEdit(initial.RTSP)
	syncDescription := qt.NewQCheckBox3("Опис такий самий, як назва")
	syncDescription.SetChecked(strings.TrimSpace(initial.RoomID) == "")
	name.OnTextEdited(func(text string) {
		if syncDescription.IsChecked() {
			description.SetText(text)
		}
	})
	syncDescription.OnStateChanged(func(_ int) {
		if syncDescription.IsChecked() {
			description.SetText(name.Text())
		}
	})
	form := qt.NewQFormLayout2()
	form.AddRow3("Назва", name.QWidget)
	form.AddRow3("Опис", description.QWidget)
	form.AddRow3("", syncDescription.QWidget)
	form.AddRow3("RTSP", rtsp.QWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return initial, false
	}
	if strings.TrimSpace(name.Text()) == "" || strings.TrimSpace(description.Text()) == "" {
		qt.QMessageBox_Information(parent, "CASL", "Вкажіть назву й опис приміщення.")
		return initial, false
	}
	initial.Name = strings.TrimSpace(name.Text())
	initial.Description = strings.TrimSpace(description.Text())
	initial.RTSP = strings.TrimSpace(rtsp.Text())
	return initial, true
}

func showCASLRegimeDialog(parent *qt.QWidget, initial caslRegime, causes []string) (caslRegime, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Часовий режим CASL")
	day := qt.NewQComboBox2()
	for _, item := range []struct {
		label string
		value int
	}{
		{"Понеділок", 1}, {"Вівторок", 2}, {"Середа", 3}, {"Четвер", 4},
		{"П'ятниця", 5}, {"Субота", 6}, {"Неділя", 0},
	} {
		day.AddItem3(item.label, qt.NewQVariant4(item.value))
		if item.value == initial.Day {
			day.SetCurrentIndex(day.Count() - 1)
		}
	}
	startTime := newLineEdit(initial.StartTime)
	stopTime := newLineEdit(initial.StopTime)
	startTime.SetPlaceholderText("08:00")
	stopTime.SetPlaceholderText("18:00")
	cause := qt.NewQComboBox2()
	cause.SetEditable(true)
	cause.AddItems(causes)
	cause.SetCurrentText(initial.Cause)
	form := qt.NewQFormLayout2()
	form.AddRow3("День", day.QWidget)
	form.AddRow3("Початок", startTime.QWidget)
	form.AddRow3("Кінець", stopTime.QWidget)
	form.AddRow3("Причина", cause.QWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return initial, false
	}
	start := strings.TrimSpace(startTime.Text())
	stop := strings.TrimSpace(stopTime.Text())
	selectedCause := strings.TrimSpace(cause.CurrentText())
	if err := validateCASLRegimeTimes(start, stop, selectedCause); err != nil {
		qt.QMessageBox_Information(parent, "CASL", err.Error())
		return initial, false
	}
	return caslRegime{
		Day:       day.ItemData(day.CurrentIndex()).ToInt(),
		StartTime: start,
		StopTime:  stop,
		Cause:     selectedCause,
	}, true
}

func showCASLUserDialog(parent *qt.QWidget) (contracts.CASLUserCreateRequest, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Новий користувач CASL")
	dialog.Resize(620, 560)
	lastName := newLineEdit("")
	firstName := newLineEdit("")
	middleName := newLineEdit("")
	tag := newLineEdit("")
	role := qt.NewQComboBox2()
	roleValues := map[string]string{
		"Відповідальна особа": "IN_CHARGE",
		"Менеджер":            "MANAGER",
		"Технік":              "TECHNICIAN",
		"Адміністратор":       "ADMIN",
	}
	role.AddItems([]string{"Відповідальна особа", "Менеджер", "Технік", "Адміністратор"})
	email := newLineEdit("")
	password := newLineEdit("")
	password.SetEchoMode(qt.QLineEdit__Password)
	phones := qt.NewQTableWidget3(1, 1)
	phones.SetHorizontalHeaderLabels([]string{"Телефон"})
	phones.SetItem(0, 0, qt.NewQTableWidgetItem2(""))
	phones.SetColumnWidth(0, 460)
	phones.SetMinimumHeight(150)
	addPhone := qt.NewQPushButton3("+")
	addPhone.SetToolTip("Додати телефон")
	removePhone := qt.NewQPushButton3("-")
	removePhone.SetToolTip("Видалити вибраний телефон")
	addPhone.OnClicked(func() {
		row := phones.RowCount()
		phones.SetRowCount(row + 1)
		phones.SetItem(row, 0, qt.NewQTableWidgetItem2(""))
		phones.SetCurrentCell(row, 0)
	})
	removePhone.OnClicked(func() {
		row := phones.CurrentRow()
		if row < 0 || phones.RowCount() <= 1 {
			return
		}
		phones.RemoveRow(row)
	})
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Прізвище", lastName.QWidget)
	form.AddRow3("Ім'я", firstName.QWidget)
	form.AddRow3("По батькові", middleName.QWidget)
	form.AddRow3("Tag", tag.QWidget)
	form.AddRow3("Роль", role.QWidget)
	form.AddRow3("Email", email.QWidget)
	form.AddRow3("Пароль", password.QWidget)
	phoneActions := qt.NewQHBoxLayout2()
	phoneActions.AddWidget(addPhone.QWidget)
	phoneActions.AddWidget(removePhone.QWidget)
	phoneActions.AddStretch()
	phoneWidget := qt.NewQWidget2()
	phoneLayout := qt.NewQVBoxLayout(phoneWidget)
	phoneLayout.SetContentsMargins(0, 0, 0, 0)
	phoneLayout.AddWidget(phones.QWidget)
	phoneLayout.AddLayout(phoneActions.QLayout)
	phoneWidget.SetLayout(phoneLayout.QLayout)
	form.AddRow3("Телефони", phoneWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return contracts.CASLUserCreateRequest{}, false
	}
	if strings.TrimSpace(lastName.Text()) == "" || strings.TrimSpace(firstName.Text()) == "" {
		qt.QMessageBox_Information(parent, "CASL", "Вкажіть прізвище та ім'я користувача.")
		return contracts.CASLUserCreateRequest{}, false
	}
	phoneNumbers := make([]contracts.CASLPhoneNumber, 0, phones.RowCount())
	for row := 0; row < phones.RowCount(); row++ {
		item := phones.Item(row, 0)
		if item == nil || strings.TrimSpace(item.Text()) == "" {
			continue
		}
		number, err := caslobject.NormalizeUAPhone(item.Text())
		if err != nil {
			qt.QMessageBox_Information(parent, "CASL", fmt.Sprintf("Телефон %d: %v", row+1, err))
			return contracts.CASLUserCreateRequest{}, false
		}
		phoneNumbers = append(phoneNumbers, contracts.CASLPhoneNumber{
			Active: len(phoneNumbers) == 0,
			Number: number,
		})
	}
	return contracts.CASLUserCreateRequest{
		LastName:     strings.TrimSpace(lastName.Text()),
		FirstName:    strings.TrimSpace(firstName.Text()),
		MiddleName:   strings.TrimSpace(middleName.Text()),
		Tag:          strings.TrimSpace(tag.Text()),
		Role:         roleValues[role.CurrentText()],
		Email:        strings.TrimSpace(email.Text()),
		Password:     strings.TrimSpace(password.Text()),
		PhoneNumbers: phoneNumbers,
	}, true
}

func showCASLLineDialog(
	parent *qt.QWidget,
	initial contracts.CASLDeviceLineDetails,
	rooms []contracts.CASLRoomDetails,
	lineTypes map[string]string,
	adapterTypes map[string]string,
) (contracts.CASLDeviceLineDetails, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Зона CASL")
	number := newSpinBox(initial.LineNumber, 1, 9999)
	description := newLineEdit(initial.Description)
	lineType := qt.NewQComboBox2()
	fillEditableCombo(lineType, sortedMapKeys(lineTypes), lineTypes, initial.LineType)
	group := newSpinBox(initial.GroupNumber, 0, 9999)
	adapter := qt.NewQComboBox2()
	fillEditableCombo(adapter, sortedMapKeys(adapterTypes), adapterTypes, initial.AdapterType)
	adapterNumber := newSpinBox(initial.AdapterNumber, 0, 9999)
	blocked := qt.NewQCheckBox3("Зона заблокована")
	blocked.SetChecked(initial.IsBlocked)
	room := qt.NewQComboBox2()
	roomIDs := map[string]string{"": ""}
	room.AddItem("")
	for _, item := range rooms {
		room.AddItem(item.Name)
		roomIDs[item.Name] = item.RoomID
		if item.RoomID == initial.RoomID {
			room.SetCurrentText(item.Name)
		}
	}
	form := qt.NewQFormLayout2()
	form.AddRow3("Номер", number.QWidget)
	form.AddRow3("Опис", description.QWidget)
	form.AddRow3("Тип", lineType.QWidget)
	form.AddRow3("Група", group.QWidget)
	form.AddRow3("Адаптер", adapter.QWidget)
	form.AddRow3("№ адаптера", adapterNumber.QWidget)
	form.AddRow3("Приміщення", room.QWidget)
	form.AddRow3("", blocked.QWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return initial, false
	}
	if strings.TrimSpace(description.Text()) == "" {
		qt.QMessageBox_Information(parent, "CASL", "Вкажіть опис зони.")
		return initial, false
	}
	initial.LineNumber = number.Value()
	initial.Description = strings.TrimSpace(description.Text())
	initial.LineType = selectedEditableComboValue(lineType, lineTypes)
	initial.GroupNumber = group.Value()
	initial.AdapterType = selectedEditableComboValue(adapter, adapterTypes)
	initial.AdapterNumber = adapterNumber.Value()
	initial.IsBlocked = blocked.IsChecked()
	initial.RoomID = roomIDs[room.CurrentText()]
	return initial, true
}

func showCASLBlockDialog(parent *qt.QWidget, device contracts.CASLDeviceDetails) (contracts.CASLDeviceBlockRequest, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Блокування об'єкта CASL")
	hours := newSpinBox(0, 0, 24)
	minutes := newSpinBox(30, 0, 59)
	unlimited := qt.NewQCheckBox3("Безстрокове блокування")
	reason := qt.NewQTextEdit2()
	reason.SetMinimumHeight(90)
	status := qt.NewQLabel3("")
	status.SetWordWrap(true)
	unlimited.OnStateChanged(func(_ int) {
		hours.SetEnabled(!unlimited.IsChecked())
		minutes.SetEnabled(!unlimited.IsChecked())
		status.SetText("")
	})
	form := qt.NewQFormLayout2()
	form.AddRow3("Години", hours.QWidget)
	form.AddRow3("Хвилини", minutes.QWidget)
	form.AddRow3("", unlimited.QWidget)
	form.AddRow3("Причина", reason.QWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	var request contracts.CASLDeviceBlockRequest
	buttons.OnAccepted(func() {
		result, err := buildCASLDeviceBlockRequest(
			device,
			hours.Value(),
			minutes.Value(),
			reason.ToPlainText(),
			unlimited.IsChecked(),
			time.Now(),
		)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		request = result
		dialog.Accept()
	})
	buttons.OnRejected(dialog.Reject)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(status.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return contracts.CASLDeviceBlockRequest{}, false
	}
	return request, true
}

func showCASLCoordinatesDialog(parent *qt.QWidget, initialAddress string, initialLat string, initialLon string) (string, string, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Вибір координат CASL")
	dialog.Resize(760, 650)
	address := newLineEdit(initialAddress)
	latitude := newLineEdit(initialLat)
	longitude := newLineEdit(initialLon)
	status := qt.NewQLabel3("Знайдіть адресу, введіть координати або клацніть потрібну точку на карті.")
	status.SetWordWrap(true)
	mapLabel := qt.NewQLabel3("Карта ще не завантажена")
	mapLabel.SetAlignment(qt.AlignCenter)
	mapLabel.SetFixedSize2(700, 360)
	mapLabel.SetFrameShape(qt.QFrame__StyledPanel)
	searchButton := qt.NewQPushButton3("Знайти адресу")
	centerButton := qt.NewQPushButton3("Показати введену точку")
	openMapButton := qt.NewQPushButton3("Відкрити OpenStreetMap")
	var mapSnapshot *geocode.MapSnapshot
	active := true

	showSnapshot := func(snapshot *geocode.MapSnapshot, markerX, markerY int) {
		body := snapshot.PNGWithMarker(markerX, markerY)
		pixmap := qt.NewQPixmap()
		if len(body) == 0 || !pixmap.LoadFromDataWithData(body) {
			status.SetText("Не вдалося відобразити карту.")
			return
		}
		mapSnapshot = snapshot
		mapLabel.SetPixmap(pixmap)
	}
	var loadMap func(float64, float64)
	loadMap = func(lat, lon float64) {
		searchButton.SetEnabled(false)
		centerButton.SetEnabled(false)
		status.SetText("Завантаження карти...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			snapshot, err := geocode.LoadMapSnapshot(ctx, lat, lon, 700, 360, 16)
			RunOnMainThread(func() {
				if !active {
					return
				}
				searchButton.SetEnabled(true)
				centerButton.SetEnabled(true)
				if err != nil {
					status.SetText(err.Error())
					return
				}
				showSnapshot(snapshot, 350, 180)
				status.SetText("Клацніть карту, щоб уточнити координати.")
			})
		}()
	}
	searchButton.OnClicked(func() {
		searchButton.SetEnabled(false)
		centerButton.SetEnabled(false)
		status.SetText("Пошук адреси...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			latText, lonText, err := geocode.SearchAddress(ctx, address.Text())
			var lat, lon float64
			if err == nil {
				lat, lon, err = validateCoordinates(latText, lonText)
			}
			RunOnMainThread(func() {
				if !active {
					return
				}
				searchButton.SetEnabled(true)
				centerButton.SetEnabled(true)
				if err != nil {
					status.SetText(err.Error())
					return
				}
				latitude.SetText(strconv.FormatFloat(lat, 'f', 6, 64))
				longitude.SetText(strconv.FormatFloat(lon, 'f', 6, 64))
				loadMap(lat, lon)
			})
		}()
	})
	centerButton.OnClicked(func() {
		lat, lon, err := validateCoordinates(latitude.Text(), longitude.Text())
		if err != nil {
			status.SetText(err.Error())
			return
		}
		loadMap(lat, lon)
	})
	mapLabel.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		if mapSnapshot == nil {
			super(event)
			return
		}
		x, y := event.X(), event.Y()
		if x < 0 || x >= 700 || y < 0 || y >= 360 {
			return
		}
		lat, lon := mapSnapshot.CoordinateAt(x, y)
		latitude.SetText(strconv.FormatFloat(lat, 'f', 6, 64))
		longitude.SetText(strconv.FormatFloat(lon, 'f', 6, 64))
		showSnapshot(mapSnapshot, x, y)
		status.SetText("Точку вибрано вручну.")
	})
	openMapButton.OnClicked(func() {
		lat := strings.TrimSpace(latitude.Text())
		lon := strings.TrimSpace(longitude.Text())
		target := "https://www.openstreetmap.org"
		if _, _, err := validateCoordinates(lat, lon); err == nil {
			target = fmt.Sprintf("https://www.openstreetmap.org/?mlat=%s&mlon=%s#map=18/%s/%s", lat, lon, lat, lon)
		}
		qt.QDesktopServices_OpenUrl(qt.NewQUrl3(target))
	})
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Адреса", horizontalWidgets(address.QWidget, searchButton.QWidget))
	form.AddRow3("Широта", latitude.QWidget)
	form.AddRow3("Довгота", longitude.QWidget)
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(func() {
		if _, _, err := validateCoordinates(latitude.Text(), longitude.Text()); err != nil {
			status.SetText(err.Error())
			return
		}
		dialog.Accept()
	})
	buttons.OnRejected(dialog.Reject)
	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(openMapButton.QWidget)
	actions.AddWidget(centerButton.QWidget)
	actions.AddStretch()
	actions.AddWidget(buttons.QWidget)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(status.QWidget)
	layout.AddWidget(mapLabel.QWidget)
	attribution := qt.NewQLabel3("Карта: © OpenStreetMap contributors")
	attribution.SetAlignment(qt.AlignRight)
	layout.AddWidget(attribution.QWidget)
	layout.AddLayout(actions.QLayout)
	dialog.SetLayout(layout.QLayout)
	if lat, lon, err := validateCoordinates(initialLat, initialLon); err == nil {
		loadMap(lat, lon)
	}
	result := dialog.Exec()
	active = false
	if result != int(qt.QDialog__Accepted) {
		return initialLat, initialLon, false
	}
	return strings.TrimSpace(latitude.Text()), strings.TrimSpace(longitude.Text()), true
}

func validateCoordinates(latitudeRaw string, longitudeRaw string) (float64, float64, error) {
	latitude, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(latitudeRaw), ",", "."), 64)
	if err != nil || latitude < -90 || latitude > 90 {
		return 0, 0, fmt.Errorf("широта має бути числом у межах -90..90")
	}
	longitude, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(longitudeRaw), ",", "."), 64)
	if err != nil || longitude < -180 || longitude > 180 {
		return 0, 0, fmt.Errorf("довгота має бути числом у межах -180..180")
	}
	return latitude, longitude, nil
}

func wrapInScrollArea(content *qt.QWidget) *qt.QWidget {
	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

func setComboByValue(combo *qt.QComboBox, values map[string]string, selected string) {
	for label, value := range values {
		if value == selected {
			setComboText(combo, label)
			return
		}
	}
}

func selectedComboValue(combo *qt.QComboBox, values map[string]string) string {
	if value, ok := values[combo.CurrentText()]; ok {
		return value
	}
	return strings.TrimSpace(combo.CurrentText())
}

func fillEditableCombo(combo *qt.QComboBox, labels []string, values map[string]string, selected string) {
	combo.SetEditable(true)
	fillComboBox(combo, labels)
	setComboByValue(combo, values, selected)
	if combo.CurrentText() == "" {
		combo.SetCurrentText(selected)
	}
}

func selectedEditableComboValue(combo *qt.QComboBox, values map[string]string) string {
	return selectedComboValue(combo, values)
}

func sortedMapKeys(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	slices.Sort(result)
	return result
}

func parseRequiredInt64(raw string, message string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s", message)
	}
	return value, nil
}

func parseOptionalInt64(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

func parseOptionalFloat(raw string) (*float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(raw), ",", "."), 64)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func formatFloatPointer(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func yesNo(value bool) string {
	if value {
		return "Так"
	}
	return "Ні"
}

func cloneCASLGuardObject(value contracts.CASLGuardObjectDetails) contracts.CASLGuardObjectDetails {
	body, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone contracts.CASLGuardObjectDetails
	if err := json.Unmarshal(body, &clone); err != nil {
		return value
	}
	return clone
}

func parseCASLObjectID(raw string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return value
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

func caslPhonesText(items []contracts.CASLPhoneNumber) string {
	phones := make([]string, 0, len(items))
	for _, item := range items {
		if value := strings.TrimSpace(item.Number); value != "" {
			phones = append(phones, formatCASLPhoneForDisplay(value))
		}
	}
	return strings.Join(phones, ", ")
}

func formatCASLPhoneForDisplay(raw string) string {
	formatted, err := caslobject.NormalizeUAPhone(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return formatted
}

func caslDateText(value int64) string {
	if value <= 0 {
		return ""
	}
	if value < 100000000000 {
		value *= 1000
	}
	return time.UnixMilli(value).Format("02.01.2006")
}

func parseCASLDate(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := time.ParseInLocation("02.01.2006", raw, time.Local)
	if err != nil {
		return 0, fmt.Errorf("використовуйте формат дд.мм.рррр")
	}
	return value.UnixMilli(), nil
}

func caslRegimesFromAny(items []any) []caslRegime {
	result := make([]caslRegime, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, caslRegime{
			Day:       int(anyInt64(item["day"])),
			StartTime: strings.TrimSpace(fmt.Sprint(item["start_time"])),
			StopTime:  strings.TrimSpace(fmt.Sprint(item["stop_time"])),
			Cause:     strings.TrimSpace(fmt.Sprint(item["cause"])),
		})
	}
	return result
}

func caslRegimesToAny(items []caslRegime) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"day": item.Day, "start_time": item.StartTime, "stop_time": item.StopTime, "cause": item.Cause,
		})
	}
	return result
}

func anyInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(value)), 10, 64)
		return parsed
	}
}

func validateCASLRegimeTimes(start string, stop string, cause string) error {
	startValue, err := time.Parse("15:04", start)
	if err != nil {
		return fmt.Errorf("некоректний час початку, використовуйте ГГ:ХХ")
	}
	stopValue, err := time.Parse("15:04", stop)
	if err != nil {
		return fmt.Errorf("некоректний час завершення, використовуйте ГГ:ХХ")
	}
	if cause == "" {
		return fmt.Errorf("вкажіть причину/заборонену дію")
	}
	if cause != "REQUIRED_GROUP_ON" && !startValue.Before(stopValue) {
		return fmt.Errorf("час початку має бути раніше часу завершення")
	}
	return nil
}

func caslWeekdayName(day int) string {
	names := map[int]string{
		0: "Неділя", 1: "Понеділок", 2: "Вівторок", 3: "Середа",
		4: "Четвер", 5: "П'ятниця", 6: "Субота",
	}
	return names[day]
}

func showCASLImagePreview(parent *qt.QWidget, body []byte) {
	if len(body) == 0 {
		qt.QMessageBox_Information(parent, "CASL", "Порожнє зображення.")
		return
	}
	pixmap := qt.NewQPixmap()
	if !pixmap.LoadFromData(&body[0], uint(len(body))) {
		qt.QMessageBox_Information(parent, "CASL", "Не вдалося декодувати зображення.")
		return
	}
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Фото CASL")
	dialog.Resize(900, 650)
	label := qt.NewQLabel3("")
	label.SetAlignment(qt.AlignCenter)
	label.SetPixmap(pixmap.Scaled(860, 580))
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	buttons.OnRejected(dialog.Reject)
	buttons.OnAccepted(dialog.Accept)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(label.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	dialog.Exec()
}
