package dialogs

import (
	"context"
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type caslObjectEditorState struct {
	win       fyne.Window
	provider  contracts.CASLObjectEditorProvider
	objectID  int64
	onChanged func()

	headerLabel *widget.Label
	statusLabel *widget.Label
	snapshot    contracts.CASLObjectEditorSnapshot

	managerOptionToID    map[string]string
	pultOptionToID       map[string]string
	techOptionToID       map[string]string
	userOptionToID       map[string]string
	roomOptionToID       map[string]string
	deviceTypeOptionToID map[string]string
	lineTypeOptionToID   map[string]string
	adapterOptionToID    map[string]string
	allUserOptions       []string
	deviceTypeOptions    []string
	lineTypeOptions      []string
	adapterOptions       []string

	objectNameEntry        *widget.Entry
	objectAddressEntry     *widget.Entry
	objectLatEntry         *widget.Entry
	objectLongEntry        *widget.Entry
	objectDescriptionEntry *widget.Entry
	objectContractEntry    *widget.Entry
	objectManagerSelect    *widget.Select
	objectNoteEntry        *widget.Entry
	objectStartDateEntry   *widget.DateEntry
	objectStatusEntry      *widget.Entry
	objectTypeEntry        *widget.Entry
	objectRequestIDEntry   *widget.Entry
	objectPultSelect       *widget.Select
	objectGeoZoneEntry     *widget.Entry
	objectBusinessEntry    *widget.Entry
	objectImagesBox        *fyne.Container
	objectSaveBtn          *widget.Button

	roomList            *widget.List
	roomNameEntry       *widget.Entry
	roomDescEntry       *widget.Entry
	roomRTSPEntry       *widget.Entry
	roomUserSearchEntry *widget.Entry
	roomUserSelect      *widget.Select
	roomUsersList       *widget.List
	roomSelected        int
	roomUserSelected    int
	roomUsersLocal      []contracts.CASLRoomUserLink
	roomImagesBox       *fyne.Container

	deviceNumberEntry        *widget.Entry
	deviceNameEntry          *widget.Entry
	deviceTypeSelect         *widget.Select
	deviceTimeoutEntry       *widget.Entry
	deviceSIM1Entry          *widget.Entry
	deviceSIM2Entry          *widget.Entry
	deviceTechnicianSelect   *widget.Select
	deviceUnitsEntry         *widget.Entry
	deviceRequisitesEntry    *widget.Entry
	deviceChangeDateEntry    *widget.DateEntry
	deviceReglamentDateEntry *widget.DateEntry
	deviceLicenceEntry       *widget.Entry
	deviceRemotePassEntry    *widget.Entry
	deviceSaveBtn            *widget.Button

	lineList               *widget.List
	lineDescriptionEntry   *widget.Entry
	lineNumberEntry        *widget.Entry
	lineTypeEntry          *widget.SelectEntry
	lineGroupNumberEntry   *widget.Entry
	lineAdapterTypeSelect  *widget.Select
	lineAdapterNumberEntry *widget.Entry
	lineBlockedCheck       *widget.Check
	lineRoomSelect         *widget.Select
	quickLineNameEntry     *widget.Entry
	quickLineTypeEntry     *widget.SelectEntry
	quickLineHintLabel     *widget.Label
	lineSelected           int
	pendingLineNumber      int
	pendingFocusQuickLine  bool
}

func ShowCASLObjectEditorDialog(parent fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onChanged func()) {
	if provider == nil {
		ShowInfoDialog(parent, "Недоступно", "CASL-редактор недоступний.")
		return
	}

	state := newCASLObjectEditorState(parent, provider, objectID, onChanged)
	state.reload()
	state.win.Show()
}

func newCASLObjectEditorState(parent fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onChanged func()) *caslObjectEditorState {
	title := fmt.Sprintf("CASL: Редагування об'єкта #%d", objectID)
	if objectID <= 0 {
		title = "CASL: Створення нового об'єкта"
	}
	win := fyne.CurrentApp().NewWindow(title)
	win.Resize(fyne.NewSize(1024, 768))

	s := &caslObjectEditorState{
		win:                      win,
		provider:                 provider,
		objectID:                 objectID,
		onChanged:                onChanged,
		headerLabel:              widget.NewLabel(""),
		statusLabel:              widget.NewLabel("Завантаження..."),
		managerOptionToID:        map[string]string{},
		pultOptionToID:           map[string]string{},
		techOptionToID:           map[string]string{},
		userOptionToID:           map[string]string{},
		roomOptionToID:           map[string]string{},
		deviceTypeOptionToID:     map[string]string{},
		lineTypeOptionToID:       map[string]string{},
		adapterOptionToID:        map[string]string{},
		roomSelected:             -1,
		roomUserSelected:         -1,
		lineSelected:             -1,
		objectNameEntry:          widget.NewEntry(),
		objectAddressEntry:       widget.NewEntry(),
		objectLatEntry:           widget.NewEntry(),
		objectLongEntry:          widget.NewEntry(),
		objectDescriptionEntry:   widget.NewMultiLineEntry(),
		objectContractEntry:      widget.NewEntry(),
		objectManagerSelect:      widget.NewSelect(nil, nil),
		objectNoteEntry:          widget.NewMultiLineEntry(),
		objectStartDateEntry:     widget.NewDateEntry(),
		objectStatusEntry:        widget.NewEntry(),
		objectTypeEntry:          widget.NewEntry(),
		objectRequestIDEntry:     widget.NewEntry(),
		objectPultSelect:         widget.NewSelect(nil, nil),
		objectGeoZoneEntry:       widget.NewEntry(),
		objectBusinessEntry:      widget.NewEntry(),
		objectImagesBox:          container.NewGridWrap(fyne.NewSize(220, 190)),
		roomNameEntry:            widget.NewEntry(),
		roomDescEntry:            widget.NewMultiLineEntry(),
		roomRTSPEntry:            widget.NewEntry(),
		roomUserSearchEntry:      widget.NewEntry(),
		roomUserSelect:           widget.NewSelect(nil, nil),
		roomImagesBox:            container.NewGridWrap(fyne.NewSize(220, 190)),
		deviceNumberEntry:        widget.NewEntry(),
		deviceNameEntry:          widget.NewEntry(),
		deviceTypeSelect:         widget.NewSelect(nil, nil),
		deviceTimeoutEntry:       widget.NewEntry(),
		deviceSIM1Entry:          widget.NewEntry(),
		deviceSIM2Entry:          widget.NewEntry(),
		deviceTechnicianSelect:   widget.NewSelect(nil, nil),
		deviceUnitsEntry:         widget.NewEntry(),
		deviceRequisitesEntry:    widget.NewEntry(),
		deviceChangeDateEntry:    widget.NewDateEntry(),
		deviceReglamentDateEntry: widget.NewDateEntry(),
		deviceLicenceEntry:       widget.NewEntry(),
		deviceRemotePassEntry:    widget.NewEntry(),
		lineDescriptionEntry:     widget.NewEntry(),
		lineNumberEntry:          widget.NewEntry(),
		lineTypeEntry:            widget.NewSelectEntry(nil),
		lineGroupNumberEntry:     widget.NewEntry(),
		lineAdapterTypeSelect:    widget.NewSelect(nil, nil),
		lineAdapterNumberEntry:   widget.NewEntry(),
		lineBlockedCheck:         widget.NewCheck("Заблокована", nil),
		lineRoomSelect:           widget.NewSelect(nil, nil),
		quickLineNameEntry:       widget.NewEntry(),
		quickLineTypeEntry:       widget.NewSelectEntry(nil),
		quickLineHintLabel:       widget.NewLabel(""),
	}

	s.objectDescriptionEntry.SetMinRowsVisible(4)
	s.objectNoteEntry.SetMinRowsVisible(3)
	s.roomDescEntry.SetMinRowsVisible(3)
	s.objectStartDateEntry.SetPlaceHolder("Оберіть дату старту")
	s.deviceChangeDateEntry.SetPlaceHolder("Оберіть дату")
	s.deviceReglamentDateEntry.SetPlaceHolder("Оберіть дату")
	s.roomUserSearchEntry.SetPlaceHolder("Пошук користувача: ПІБ, ID, телефон")
	s.quickLineNameEntry.SetPlaceHolder("Введіть назву нової зони та натисніть Enter")
	s.quickLineTypeEntry.SetText("NORMAL")
	s.quickLineNameEntry.OnSubmitted = func(string) {
		s.createQuickLine()
	}
	s.roomUserSearchEntry.OnChanged = func(value string) {
		s.refreshRoomUserOptions(value)
	}

	s.initLists()

	refreshBtn := widget.NewButton("Оновити", s.reload)
	closeBtn := widget.NewButton("Закрити", func() { s.win.Close() })

	tabs := container.NewAppTabs(
		container.NewTabItem("Об'єкт", s.buildObjectTab()),
		container.NewTabItem("Зв'язки", s.buildRoomsTab()),
		container.NewTabItem("Обладнання", s.buildDeviceTab()),
		container.NewTabItem("Зони", s.buildLinesTab()),
	)

	s.win.SetContent(container.NewBorder(
		container.NewVBox(
			s.headerLabel,
			container.NewHBox(refreshBtn, layout.NewSpacer(), s.statusLabel, closeBtn),
			widget.NewSeparator(),
		),
		nil, nil, nil,
		tabs,
	))
	s.refreshWindowPresentation()

	return s
}

func (s *caslObjectEditorState) initLists() {
	s.roomList = widget.NewList(
		func() int { return len(s.snapshot.Object.Rooms) },
		func() fyne.CanvasObject { return newCASLListRowTemplate() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(s.snapshot.Object.Rooms) {
				setCASLListRow(obj, "", "")
				return
			}
			room := s.snapshot.Object.Rooms[id]
			setCASLListRow(
				obj,
				fmt.Sprintf("%s [%s]", firstNonEmpty(room.Name, "Без назви"), room.RoomID),
				fmt.Sprintf("Користувачів: %d | Зон: %d", len(room.Users), len(room.Lines)),
			)
		},
	)
	s.roomList.OnSelected = func(id widget.ListItemID) {
		s.selectRoom(id)
	}
	s.applyRoomListHeights()

	s.roomUsersList = widget.NewList(
		func() int { return len(s.roomUsersLocal) },
		func() fyne.CanvasObject { return newCASLListRowTemplate() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(s.roomUsersLocal) {
				setCASLListRow(obj, "", "")
				return
			}
			item := s.roomUsersLocal[id]
			setCASLListRow(
				obj,
				fmt.Sprintf("%d. %s", id+1, s.userLabelByID(item.UserID)),
				s.roomUserDetailsText(item),
			)
		},
	)
	s.roomUsersList.OnSelected = func(id widget.ListItemID) {
		s.roomUserSelected = id
	}
	s.applyRoomUsersListHeights()

	s.lineList = widget.NewList(
		func() int { return len(s.snapshot.Object.Device.Lines) },
		func() fyne.CanvasObject { return newCASLListRowTemplate() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(s.snapshot.Object.Device.Lines) {
				setCASLListRow(obj, "", "")
				return
			}
			line := s.snapshot.Object.Device.Lines[id]
			roomLabel := "без приміщення"
			if roomName := s.roomNameByID(line.RoomID); roomName != "" {
				roomLabel = roomName
			}
			blockState := "активна"
			if line.IsBlocked {
				blockState = "заблокована"
			}
			setCASLListRow(
				obj,
				fmt.Sprintf("#%d %s", line.LineNumber, firstNonEmpty(line.Description, "Немає назви")),
				fmt.Sprintf("Тип: %s | Група: %d | %s | %s", s.displayLineType(line.LineType), line.GroupNumber, roomLabel, blockState),
			)
		},
	)
	s.lineList.OnSelected = func(id widget.ListItemID) {
		s.selectLine(id)
	}
	s.applyLineListHeights()
}

func (s *caslObjectEditorState) buildObjectTab() fyne.CanvasObject {
	s.objectSaveBtn = widget.NewButton("Зберегти об'єкт", s.submitObject)
	addImageBtn := widget.NewButton("Додати фото", s.uploadObjectImage)
	pickCoordsBtn := widget.NewButton("Вибрати на карті", s.pickObjectCoordinatesOnMap)
	objectImagesScroll := container.NewVScroll(s.objectImagesBox)
	objectImagesArea := fixedMinHeightArea(560, objectImagesScroll)

	mainForm := widget.NewForm(
		widget.NewFormItem("Назва", s.objectNameEntry),
		widget.NewFormItem("Адреса", s.objectAddressEntry),
		widget.NewFormItem("Договір", s.objectContractEntry),
		widget.NewFormItem("Тип об'єкта", s.objectTypeEntry),
		widget.NewFormItem("Статус", s.objectStatusEntry),
		widget.NewFormItem("ID заявки", s.objectRequestIDEntry),
	)
	controlForm := widget.NewForm(
		widget.NewFormItem("Менеджер", s.objectManagerSelect),
		widget.NewFormItem("Реагуючий пульт", s.objectPultSelect),
		widget.NewFormItem("Дата старту", s.objectStartDateEntry),
		widget.NewFormItem("Geo zone ID", s.objectGeoZoneEntry),
		widget.NewFormItem("Business coeff", s.objectBusinessEntry),
	)
	locationForm := widget.NewForm(
		widget.NewFormItem("Широта", s.objectLatEntry),
		widget.NewFormItem("Довгота", s.objectLongEntry),
		widget.NewFormItem("", container.NewHBox(pickCoordsBtn)),
		widget.NewFormItem("Опис", s.objectDescriptionEntry),
		widget.NewFormItem("Примітка", s.objectNoteEntry),
	)
	content := container.NewVBox(
		widget.NewCard("Основне", "Базова інформація про об'єкт", mainForm),
		widget.NewCard("Керування", "Відповідальні та службові параметри", controlForm),
		widget.NewCard("Локація та нотатки", "Координати, опис та робочі примітки", locationForm),
	)
	imagesCard := widget.NewCard("Фото об'єкта", "Поточні фото та швидке додавання нових", container.NewBorder(
		nil,
		container.NewHBox(addImageBtn),
		nil, nil,
		objectImagesArea,
	))
	split := container.NewHSplit(imagesCard, container.NewScroll(content))
	split.SetOffset(0.24)
	return container.NewBorder(nil, container.NewHBox(layout.NewSpacer(), s.objectSaveBtn), nil, nil, split)
}

func (s *caslObjectEditorState) buildRoomsTab() fyne.CanvasObject {
	saveRoomBtn := widget.NewButton("Зберегти приміщення", s.saveRoom)
	createRoomBtn := widget.NewButton("Нове приміщення", s.createRoom)
	addImageBtn := widget.NewButton("Додати фото", s.uploadRoomImage)
	roomImagesScroll := container.NewVScroll(s.roomImagesBox)
	roomImagesArea := fixedMinHeightArea(700, roomImagesScroll)
	addUserBtn := widget.NewButton("Додати користувача", s.addUserToRoom)
	removeUserBtn := widget.NewButton("Видалити користувача", s.removeUserFromRoom)
	upBtn := widget.NewButton("Вгору", s.moveRoomUserUp)
	downBtn := widget.NewButton("Вниз", s.moveRoomUserDown)
	saveOrderBtn := widget.NewButton("Зберегти порядок", s.saveRoomUserPriorities)
	createUserBtn := widget.NewButton("Створити користувача", s.createUserAndAddToRoom)

	detailsCard := widget.NewCard("Дані приміщення", "Редагування назви, опису та RTSP", container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Назва", s.roomNameEntry),
			widget.NewFormItem("Опис", s.roomDescEntry),
			widget.NewFormItem("RTSP", s.roomRTSPEntry),
		),
		container.NewHBox(saveRoomBtn, createRoomBtn),
	))

	usersCard := widget.NewCard("Користувачі приміщення", "Пошук, додавання і порядок користувачів", container.NewBorder(
		container.NewVBox(
			s.roomUserSearchEntry,
			s.roomUserSelect,
			container.NewHBox(addUserBtn, createUserBtn),
		),
		container.NewHBox(upBtn, downBtn, saveOrderBtn, layout.NewSpacer(), removeUserBtn),
		nil, nil,
		s.roomUsersList,
	))

	photosCard := widget.NewCard("Фото приміщення", "Фото вибраної кімнати", container.NewBorder(
		nil,
		container.NewHBox(addImageBtn),
		nil, nil,
		roomImagesArea,
	))

	center := container.NewVSplit(detailsCard, usersCard)
	center.SetOffset(0.34)

	left := widget.NewCard("Приміщення", "Список кімнат об'єкта", s.roomList)
	leftCenter := container.NewHSplit(left, center)
	leftCenter.SetOffset(0.30)

	root := container.NewHSplit(leftCenter, photosCard)
	root.SetOffset(0.72)
	return container.NewVScroll(root)
}

func (s *caslObjectEditorState) buildDeviceTab() fyne.CanvasObject {
	s.deviceSaveBtn = widget.NewButton("Зберегти обладнання", s.submitDevice)

	mainForm := widget.NewForm(
		widget.NewFormItem("Номер", s.deviceNumberEntry),
		widget.NewFormItem("Назва", s.deviceNameEntry),
		widget.NewFormItem("Тип пристрою", s.deviceTypeSelect),
		widget.NewFormItem("Timeout", s.deviceTimeoutEntry),
		widget.NewFormItem("Технік", s.deviceTechnicianSelect),
	)
	serviceForm := widget.NewForm(
		widget.NewFormItem("SIM1", s.deviceSIM1Entry),
		widget.NewFormItem("SIM2", s.deviceSIM2Entry),
		widget.NewFormItem("Units", s.deviceUnitsEntry),
		widget.NewFormItem("Requisites", s.deviceRequisitesEntry),
	)
	dateGrid := container.NewGridWithColumns(2,
		widget.NewCard("Дата зміни", "Коли обладнання востаннє змінювали", s.deviceChangeDateEntry),
		widget.NewCard("Дата регламенту", "Планова дата сервісного регламенту", s.deviceReglamentDateEntry),
	)
	accessForm := widget.NewForm(
		widget.NewFormItem("Licence key", s.deviceLicenceEntry),
		widget.NewFormItem("Remote password", s.deviceRemotePassEntry),
	)
	content := container.NewVBox(
		widget.NewCard("Основне", "Ідентифікація і тип обладнання", mainForm),
		widget.NewCard("Зв'язок та обслуговування", "SIM-карти, сервіс і службові реквізити", serviceForm),
		widget.NewCard("Сервісні дати", "Окремо винесені дати, щоб поля не з'їжджали по висоті", dateGrid),
		widget.NewCard("Доступ", "Ліцензія та пароль віддаленого доступу", accessForm),
	)
	return container.NewBorder(nil, container.NewHBox(layout.NewSpacer(), s.deviceSaveBtn), nil, nil, container.NewScroll(content))
}

func (s *caslObjectEditorState) buildLinesTab() fyne.CanvasObject {
	saveBtn := widget.NewButton("Зберегти зону", s.saveLine)
	createBtn := widget.NewButton("Створити зону", s.createLine)
	bindBtn := widget.NewButton("Прив'язати до приміщення", s.bindLineToRoom)
	quickCreateBtn := widget.NewButton("Створити швидко", s.createQuickLine)

	quickCard := widget.NewCard("Швидке створення", "Введіть назву зони, натисніть Enter і продовжуйте вводити наступну", container.NewVBox(
		s.quickLineHintLabel,
		widget.NewForm(
			widget.NewFormItem("Назва нової зони", s.quickLineNameEntry),
			widget.NewFormItem("Тип нової зони", s.quickLineTypeEntry),
		),
		container.NewHBox(quickCreateBtn, layout.NewSpacer(), widget.NewLabel("Enter = створити наступну зону")),
	))

	editorCard := widget.NewCard("Вибрана зона", "Редагування параметрів існуючої або ручне створення нової", container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Номер зони", s.lineNumberEntry),
			widget.NewFormItem("Назва / опис", s.lineDescriptionEntry),
			widget.NewFormItem("Тип зони", s.lineTypeEntry),
			widget.NewFormItem("Group number", s.lineGroupNumberEntry),
			widget.NewFormItem("Adapter type", s.lineAdapterTypeSelect),
			widget.NewFormItem("Adapter number", s.lineAdapterNumberEntry),
			widget.NewFormItem("Приміщення", s.lineRoomSelect),
			widget.NewFormItem("", s.lineBlockedCheck),
		),
		container.NewHBox(saveBtn, createBtn, bindBtn),
	))

	left := widget.NewCard("Зони обладнання", "Список зон приладу", s.lineList)
	right := container.NewScroll(container.NewVBox(quickCard, editorCard))
	split := container.NewHSplit(left, right)
	split.SetOffset(0.36)
	return split
}

func (s *caslObjectEditorState) reload() {
	s.setStatus("Завантаження...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		snapshot, err := s.provider.GetCASLObjectEditorSnapshot(ctx, s.objectID)
		fyne.Do(func() {
			if err != nil {
				s.setStatus("Помилка завантаження")
				dialog.ShowError(err, s.win)
				return
			}

			s.snapshot = snapshot
			s.rebuildOptions()
			s.fillObjectForm()
			s.fillDeviceForm()
			s.refreshQuickLineHint()
			s.refreshObjectImages()
			s.refreshWindowPresentation()
			s.roomList.Refresh()
			s.lineList.Refresh()
			s.applyRoomListHeights()
			s.applyRoomUsersListHeights()
			s.applyLineListHeights()
			if len(s.snapshot.Object.Rooms) == 0 {
				s.refreshRoomImages(nil)
			}

			if s.pendingLineNumber > 0 {
				if idx := s.findLineIndexByNumber(s.pendingLineNumber); idx >= 0 {
					s.lineSelected = idx
				}
				s.pendingLineNumber = 0
			}
			if len(s.snapshot.Object.Rooms) > 0 {
				if s.roomSelected < 0 || s.roomSelected >= len(s.snapshot.Object.Rooms) {
					s.roomSelected = 0
				}
				s.roomList.Select(s.roomSelected)
			}
			if len(s.snapshot.Object.Device.Lines) > 0 {
				if s.lineSelected < 0 || s.lineSelected >= len(s.snapshot.Object.Device.Lines) {
					s.lineSelected = 0
				}
				s.lineList.Select(s.lineSelected)
			}
			if s.pendingFocusQuickLine {
				s.pendingFocusQuickLine = false
				s.win.Canvas().Focus(s.quickLineNameEntry)
			}

			s.setStatus("Готово")
		})
	}()
}

func (s *caslObjectEditorState) rebuildOptions() {
	s.managerOptionToID = map[string]string{"": ""}
	s.techOptionToID = map[string]string{"": ""}
	s.userOptionToID = map[string]string{"": ""}
	s.pultOptionToID = map[string]string{"": ""}
	s.roomOptionToID = map[string]string{"": ""}
	s.deviceTypeOptionToID = map[string]string{"": ""}
	s.lineTypeOptionToID = map[string]string{}
	s.adapterOptionToID = map[string]string{}

	managerOptions := []string{""}
	technicianOptions := []string{""}
	userOptions := []string{""}
	for _, user := range s.snapshot.Users {
		label := fmt.Sprintf("%s [%s]", caslProfileName(user), user.UserID)
		userOptions = append(userOptions, label)
		s.userOptionToID[label] = user.UserID

		if strings.EqualFold(strings.TrimSpace(user.Role), "MANAGER") {
			managerOptions = append(managerOptions, label)
			s.managerOptionToID[label] = user.UserID
		}

		if strings.EqualFold(strings.TrimSpace(user.Role), "TECHNICIAN") {
			technicianOptions = append(technicianOptions, label)
			s.techOptionToID[label] = user.UserID
		}
	}
	s.allUserOptions = append([]string(nil), userOptions...)
	s.objectManagerSelect.Options = managerOptions
	s.objectManagerSelect.Refresh()
	s.deviceTechnicianSelect.Options = technicianOptions
	s.deviceTechnicianSelect.Refresh()
	s.refreshRoomUserOptions(s.roomUserSearchEntry.Text)

	pultOptions := []string{""}
	for _, pult := range s.snapshot.Pults {
		label := fmt.Sprintf("%s [%s]", firstNonEmpty(pult.Name, pult.Nickname, "Пульт"), pult.PultID)
		pultOptions = append(pultOptions, label)
		s.pultOptionToID[label] = pult.PultID
	}
	s.objectPultSelect.Options = pultOptions
	s.objectPultSelect.Refresh()

	roomOptions := []string{""}
	for _, room := range s.snapshot.Object.Rooms {
		label := fmt.Sprintf("%s [%s]", room.Name, room.RoomID)
		roomOptions = append(roomOptions, label)
		s.roomOptionToID[label] = room.RoomID
	}
	s.lineRoomSelect.Options = roomOptions
	s.lineRoomSelect.Refresh()

	deviceTypeMap := caslDeviceTypeOptionsMap(s.snapshot.Dictionary)
	for rawType, label := range deviceTypeMap {
		if strings.TrimSpace(label) == "" || strings.TrimSpace(label) == strings.TrimSpace(rawType) {
			deviceTypeMap[rawType] = s.displayDeviceType(rawType)
		}
	}
	if rawType := strings.TrimSpace(s.snapshot.Object.Device.Type); rawType != "" {
		ensureOptionMapping(deviceTypeMap, rawType, s.displayDeviceType(rawType))
	}
	s.deviceTypeOptions, s.deviceTypeOptionToID = labeledOptionMap(deviceTypeMap)
	s.deviceTypeSelect.Options = s.deviceTypeOptions
	s.deviceTypeSelect.Refresh()

	lineTypeMap := caslLineTypeOptionsMap(s.snapshot.Dictionary)
	for _, line := range s.snapshot.Object.Device.Lines {
		if rawType := strings.TrimSpace(line.LineType); rawType != "" {
			setRawOptionLabel(lineTypeMap, rawType, s.displayLineType(rawType))
		}
	}
	s.lineTypeOptions, s.lineTypeOptionToID = labeledOptionMap(lineTypeMap)
	s.lineTypeEntry.SetOptions(s.lineTypeOptions)
	s.quickLineTypeEntry.SetOptions(s.lineTypeOptions)
	defaultLineTypeLabel := optionLabelByValue("NORMAL", s.lineTypeOptionToID)
	if strings.TrimSpace(defaultLineTypeLabel) == "" {
		defaultLineTypeLabel = optionLabelByValue("ZONE_NORMAL", s.lineTypeOptionToID)
	}
	if strings.TrimSpace(defaultLineTypeLabel) == "" {
		defaultLineTypeLabel = optionLabelByValue("EMPTY", s.lineTypeOptionToID)
	}
	if strings.TrimSpace(defaultLineTypeLabel) == "" && len(s.lineTypeOptions) > 0 {
		defaultLineTypeLabel = s.lineTypeOptions[0]
	}
	if strings.TrimSpace(s.quickLineTypeEntry.Text) == "" || !containsString(s.lineTypeOptions, s.quickLineTypeEntry.Text) {
		s.quickLineTypeEntry.SetText(defaultLineTypeLabel)
	}

	adapterMap := caslAdapterTypeOptionsMap(s.snapshot.Dictionary)
	for _, line := range s.snapshot.Object.Device.Lines {
		if rawType := strings.TrimSpace(line.AdapterType); rawType != "" {
			adapterMap[rawType] = s.displayAdapterType(rawType)
		}
	}
	setRawOptionLabel(adapterMap, "SYS", s.displayAdapterType("SYS"))
	s.adapterOptions, s.adapterOptionToID = labeledOptionMap(adapterMap)
	s.lineAdapterTypeSelect.Options = s.adapterOptions
	s.lineAdapterTypeSelect.Refresh()
	defaultAdapterLabel := optionLabelByValue("SYS", s.adapterOptionToID)
	if strings.TrimSpace(defaultAdapterLabel) == "" && len(s.adapterOptions) > 0 {
		defaultAdapterLabel = s.adapterOptions[0]
	}
	if len(s.adapterOptions) > 0 && !containsString(s.adapterOptions, s.lineAdapterTypeSelect.Selected) {
		s.lineAdapterTypeSelect.SetSelected(defaultAdapterLabel)
	}
}

func (s *caslObjectEditorState) fillObjectForm() {
	obj := s.snapshot.Object
	s.objectNameEntry.SetText(obj.Name)
	s.objectAddressEntry.SetText(obj.Address)
	s.objectLatEntry.SetText(obj.Lat)
	s.objectLongEntry.SetText(obj.Long)
	s.objectDescriptionEntry.SetText(obj.Description)
	s.objectContractEntry.SetText(obj.Contract)
	s.objectNoteEntry.SetText(obj.Note)
	s.objectStartDateEntry.SetDate(caslDatePtr(obj.StartDate))
	s.objectStatusEntry.SetText(obj.ObjectStatus)
	s.objectTypeEntry.SetText(obj.ObjectType)
	s.objectRequestIDEntry.SetText(obj.IDRequest)
	s.objectGeoZoneEntry.SetText(int64ToString(obj.GeoZoneID))
	s.objectBusinessEntry.SetText(float64PtrToString(obj.BusinessCoeff))
	s.objectManagerSelect.SetSelected(s.userOptionByID(obj.ManagerID, s.managerOptionToID))
	s.objectPultSelect.SetSelected(s.pultOptionByID(obj.ReactingPultID))
	if !s.hasObject() {
		if s.objectStatusEntry.Text == "" {
			s.objectStatusEntry.SetText("Включено")
		}
		if s.objectPultSelect.Selected == "" && len(s.objectPultSelect.Options) > 1 {
			s.objectPultSelect.SetSelected(s.objectPultSelect.Options[1])
		}
	}
	s.refreshObjectImages()
}

func (s *caslObjectEditorState) fillDeviceForm() {
	device := s.snapshot.Object.Device
	s.deviceNumberEntry.SetText(int64ToString(device.Number))
	s.deviceNameEntry.SetText(device.Name)
	s.deviceTypeSelect.SetSelected(optionLabelByValue(device.Type, s.deviceTypeOptionToID))
	s.deviceTimeoutEntry.SetText(int64ToString(device.Timeout))
	s.deviceSIM1Entry.SetText(device.SIM1)
	s.deviceSIM2Entry.SetText(device.SIM2)
	s.deviceTechnicianSelect.SetSelected(s.userOptionByID(device.TechnicianID, s.techOptionToID))
	s.deviceUnitsEntry.SetText(device.Units)
	s.deviceRequisitesEntry.SetText(device.Requisites)
	s.deviceChangeDateEntry.SetDate(caslDatePtr(device.ChangeDate))
	s.deviceReglamentDateEntry.SetDate(caslDatePtr(device.ReglamentDate))
	s.deviceLicenceEntry.SetText(device.LicenceKey)
	s.deviceRemotePassEntry.SetText(device.PasswRemote)
	if !s.hasDevice() && s.deviceTimeoutEntry.Text == "" {
		s.deviceTimeoutEntry.SetText("3600")
	}
}

func (s *caslObjectEditorState) selectRoom(index int) {
	if index < 0 || index >= len(s.snapshot.Object.Rooms) {
		return
	}
	s.roomSelected = index
	room := s.snapshot.Object.Rooms[index]
	s.roomNameEntry.SetText(room.Name)
	s.roomDescEntry.SetText(room.Description)
	s.roomRTSPEntry.SetText(room.RTSP)
	s.roomUsersLocal = append([]contracts.CASLRoomUserLink(nil), room.Users...)
	s.roomUsersList.Refresh()
	s.refreshRoomUserOptions(s.roomUserSearchEntry.Text)
	s.refreshRoomImages(room.Images)
	if len(s.roomUsersLocal) > 0 {
		s.roomUserSelected = 0
		s.roomUsersList.Select(0)
	} else {
		s.roomUserSelected = -1
	}
}

func (s *caslObjectEditorState) selectLine(index int) {
	if index < 0 || index >= len(s.snapshot.Object.Device.Lines) {
		return
	}
	s.lineSelected = index
	line := s.snapshot.Object.Device.Lines[index]
	s.lineDescriptionEntry.SetText(line.Description)
	s.lineNumberEntry.SetText(strconv.Itoa(line.LineNumber))
	s.lineTypeEntry.SetText(optionLabelByValue(line.LineType, s.lineTypeOptionToID))
	s.lineGroupNumberEntry.SetText(strconv.Itoa(line.GroupNumber))
	s.lineAdapterTypeSelect.SetSelected(optionLabelByValue(line.AdapterType, s.adapterOptionToID))
	s.lineAdapterNumberEntry.SetText(strconv.Itoa(line.AdapterNumber))
	s.lineBlockedCheck.SetChecked(line.IsBlocked)
	s.lineRoomSelect.SetSelected(s.roomOptionByID(line.RoomID))
}

func (s *caslObjectEditorState) submitObject() {
	if !s.hasObject() {
		s.createObject()
		return
	}
	s.saveObject()
}

func (s *caslObjectEditorState) saveObject() {
	update, err := s.currentObjectUpdate()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}

	s.runMutation("Збереження об'єкта...", func(ctx context.Context) error {
		return s.provider.UpdateCASLObject(ctx, update)
	})
}

func (s *caslObjectEditorState) createObject() {
	create, err := s.currentObjectCreate()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}

	var createdObjID string
	s.runMutationWithSuccess("Створення об'єкта...", func(ctx context.Context) error {
		objID, createErr := s.provider.CreateCASLObject(ctx, create)
		if createErr != nil {
			return createErr
		}
		if strings.TrimSpace(objID) == "" {
			return fmt.Errorf("casl не повернув obj_id нового об'єкта")
		}
		createdObjID = strings.TrimSpace(objID)
		return nil
	}, func() {
		s.snapshot.Object.ObjID = createdObjID
		if parsedID, parseErr := strconv.ParseInt(createdObjID, 10, 64); parseErr == nil {
			s.objectID = parsedID
		}
		s.refreshWindowPresentation()
	})
}

func (s *caslObjectEditorState) saveRoom() {
	update, err := s.currentRoomUpdate()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.runMutation("Збереження приміщення...", func(ctx context.Context) error {
		return s.provider.UpdateCASLRoom(ctx, update)
	})
}

func (s *caslObjectEditorState) createRoom() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт, а потім додавайте приміщення.") {
		return
	}
	create := contracts.CASLRoomCreate{
		ObjID:       s.snapshot.Object.ObjID,
		Name:        strings.TrimSpace(s.roomNameEntry.Text),
		Description: strings.TrimSpace(s.roomDescEntry.Text),
		Images:      nil,
		RTSP:        strings.TrimSpace(s.roomRTSPEntry.Text),
	}
	if create.Name == "" {
		ShowInfoDialog(s.win, "Некоректні дані", "Вкажіть назву нового приміщення.")
		return
	}
	s.runMutation("Створення приміщення...", func(ctx context.Context) error {
		return s.provider.CreateCASLRoom(ctx, create)
	})
}

func (s *caslObjectEditorState) currentObjectCreate() (contracts.CASLGuardObjectCreate, error) {
	startDate, err := dateEntryUnixMilli(s.objectStartDateEntry)
	if err != nil {
		return contracts.CASLGuardObjectCreate{}, fmt.Errorf("дата старту: %w", err)
	}
	geoZoneID, err := parseCASLEditorInt64(s.objectGeoZoneEntry.Text)
	if err != nil {
		return contracts.CASLGuardObjectCreate{}, fmt.Errorf("geo zone: %w", err)
	}
	businessCoeff, err := parseCASLEditorFloatPtr(s.objectBusinessEntry.Text)
	if err != nil {
		return contracts.CASLGuardObjectCreate{}, fmt.Errorf("business coeff: %w", err)
	}
	name := strings.TrimSpace(s.objectNameEntry.Text)
	if name == "" {
		return contracts.CASLGuardObjectCreate{}, fmt.Errorf("вкажіть назву об'єкта")
	}
	return contracts.CASLGuardObjectCreate{
		Name:           name,
		Address:        strings.TrimSpace(s.objectAddressEntry.Text),
		Long:           strings.TrimSpace(s.objectLongEntry.Text),
		Lat:            strings.TrimSpace(s.objectLatEntry.Text),
		Description:    strings.TrimSpace(s.objectDescriptionEntry.Text),
		Contract:       strings.TrimSpace(s.objectContractEntry.Text),
		ManagerID:      s.managerOptionToID[s.objectManagerSelect.Selected],
		Note:           strings.TrimSpace(s.objectNoteEntry.Text),
		StartDate:      startDate,
		Status:         strings.TrimSpace(s.objectStatusEntry.Text),
		ObjectType:     strings.TrimSpace(s.objectTypeEntry.Text),
		IDRequest:      strings.TrimSpace(s.objectRequestIDEntry.Text),
		ReactingPultID: s.pultOptionToID[s.objectPultSelect.Selected],
		GeoZoneID:      geoZoneID,
		BusinessCoeff:  businessCoeff,
	}, nil
}

func (s *caslObjectEditorState) currentObjectUpdate() (contracts.CASLGuardObjectUpdate, error) {
	startDate, err := dateEntryUnixMilli(s.objectStartDateEntry)
	if err != nil {
		return contracts.CASLGuardObjectUpdate{}, fmt.Errorf("дата старту: %w", err)
	}
	geoZoneID, err := parseCASLEditorInt64(s.objectGeoZoneEntry.Text)
	if err != nil {
		return contracts.CASLGuardObjectUpdate{}, fmt.Errorf("geo zone: %w", err)
	}
	businessCoeff, err := parseCASLEditorFloatPtr(s.objectBusinessEntry.Text)
	if err != nil {
		return contracts.CASLGuardObjectUpdate{}, fmt.Errorf("business coeff: %w", err)
	}
	return contracts.CASLGuardObjectUpdate{
		ObjID:          s.snapshot.Object.ObjID,
		Name:           strings.TrimSpace(s.objectNameEntry.Text),
		Address:        strings.TrimSpace(s.objectAddressEntry.Text),
		Long:           strings.TrimSpace(s.objectLongEntry.Text),
		Lat:            strings.TrimSpace(s.objectLatEntry.Text),
		Description:    strings.TrimSpace(s.objectDescriptionEntry.Text),
		Contract:       strings.TrimSpace(s.objectContractEntry.Text),
		ManagerID:      s.managerOptionToID[s.objectManagerSelect.Selected],
		Note:           strings.TrimSpace(s.objectNoteEntry.Text),
		StartDate:      startDate,
		Status:         strings.TrimSpace(s.objectStatusEntry.Text),
		ObjectType:     strings.TrimSpace(s.objectTypeEntry.Text),
		IDRequest:      strings.TrimSpace(s.objectRequestIDEntry.Text),
		ReactingPultID: s.pultOptionToID[s.objectPultSelect.Selected],
		GeoZoneID:      geoZoneID,
		BusinessCoeff:  businessCoeff,
	}, nil
}

func (s *caslObjectEditorState) currentRoomUpdate() (contracts.CASLRoomUpdate, error) {
	room, ok := s.selectedRoom()
	if !ok {
		return contracts.CASLRoomUpdate{}, fmt.Errorf("не вибрано приміщення")
	}
	return contracts.CASLRoomUpdate{
		ObjID:       s.snapshot.Object.ObjID,
		RoomID:      room.RoomID,
		Name:        strings.TrimSpace(s.roomNameEntry.Text),
		Description: strings.TrimSpace(s.roomDescEntry.Text),
		RTSP:        strings.TrimSpace(s.roomRTSPEntry.Text),
	}, nil
}

func (s *caslObjectEditorState) uploadObjectImage() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт, а потім додавайте фото.") {
		return
	}
	if len(s.objectScopedImages()) >= 3 {
		ShowInfoDialog(s.win, "Ліміт фото", "Для об'єкта доступно максимум 3 фото.")
		return
	}
	s.openCASLImagePicker("Додати фото об'єкта", func(imageType string, encoded string) {
		s.runMutation("Завантаження фото об'єкта...", func(ctx context.Context) error {
			return s.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
				ObjID:     s.snapshot.Object.ObjID,
				ImageType: imageType,
				ImageData: encoded,
			})
		})
	})
}

func (s *caslObjectEditorState) uploadRoomImage() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення, а потім додавайте фото.") {
		return
	}
	room, ok := s.selectedRoom()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть приміщення у списку.")
		return
	}
	if len(room.Images) >= 3 {
		ShowInfoDialog(s.win, "Ліміт фото", "Для приміщення доступно максимум 3 фото.")
		return
	}
	s.openCASLImagePicker("Додати фото приміщення", func(imageType string, encoded string) {
		s.runMutation("Завантаження фото приміщення...", func(ctx context.Context) error {
			return s.provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
				ObjID:     s.snapshot.Object.ObjID,
				RoomID:    room.RoomID,
				ImageType: imageType,
				ImageData: encoded,
			})
		})
	})
}

func (s *caslObjectEditorState) deleteObjectImage(imageID string) {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт.") {
		return
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		ShowInfoDialog(s.win, "Недоступно", "CASL не повернув image_id для цього фото, тому видалення недоступне.")
		return
	}
	s.runMutation("Видалення фото об'єкта...", func(ctx context.Context) error {
		return s.provider.DeleteCASLImage(ctx, contracts.CASLImageDeleteRequest{
			ObjID:   s.snapshot.Object.ObjID,
			ImageID: imageID,
		})
	})
}

func (s *caslObjectEditorState) deleteRoomImage(imageID string) {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення.") {
		return
	}
	room, ok := s.selectedRoom()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть приміщення у списку.")
		return
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		ShowInfoDialog(s.win, "Недоступно", "CASL не повернув image_id для цього фото, тому видалення недоступне.")
		return
	}
	s.runMutation("Видалення фото приміщення...", func(ctx context.Context) error {
		return s.provider.DeleteCASLImage(ctx, contracts.CASLImageDeleteRequest{
			ObjID:   s.snapshot.Object.ObjID,
			RoomID:  room.RoomID,
			ImageID: imageID,
		})
	})
}

func (s *caslObjectEditorState) openCASLImagePicker(title string, onLoaded func(imageType string, encoded string)) {
	if runtime.GOOS == "windows" {
		path, err := pickCASLImageFilePath(title)
		if err != nil {
			dialog.ShowError(err, s.win)
			return
		}
		if strings.TrimSpace(path) == "" {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			dialog.ShowError(err, s.win)
			return
		}
		imageType, encoded := caslEncodeImageUpload(filepath.Base(path), data)
		if strings.TrimSpace(encoded) == "" || strings.TrimSpace(imageType) == "" {
			dialog.ShowError(fmt.Errorf("не вдалося підготувати зображення"), s.win)
			return
		}
		onLoaded(imageType, encoded)
		return
	}

	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.win)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			dialog.ShowError(readErr, s.win)
			return
		}
		imageType, encoded := caslEncodeImageUpload(reader.URI().Name(), data)
		if strings.TrimSpace(encoded) == "" || strings.TrimSpace(imageType) == "" {
			dialog.ShowError(fmt.Errorf("не вдалося підготувати зображення"), s.win)
			return
		}
		onLoaded(imageType, encoded)
	}, s.win)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png", ".webp", ".bmp", ".gif"}))
	fileDialog.Show()
}

func (s *caslObjectEditorState) submitDevice() {
	if !s.hasDevice() {
		s.createDevice()
		return
	}
	s.saveDevice()
}

func (s *caslObjectEditorState) createDevice() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт, а потім обладнання.") {
		return
	}

	create, err := s.currentDeviceCreate()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}

	var createdDeviceID string
	s.runMutationWithSuccess("Створення обладнання...", func(ctx context.Context) error {
		inUse, useErr := s.provider.IsCASLDeviceNumberInUse(ctx, create.Number)
		if useErr != nil {
			return useErr
		}
		if inUse {
			return fmt.Errorf("номер приладу %d вже зайнятий", create.Number)
		}
		deviceID, createErr := s.provider.CreateCASLDevice(ctx, create)
		if createErr != nil {
			return createErr
		}
		createdDeviceID = strings.TrimSpace(deviceID)
		return nil
	}, func() {
		s.snapshot.Object.Device.DeviceID = createdDeviceID
		s.snapshot.Object.Device.Number = create.Number
		s.refreshWindowPresentation()
	})
}

func (s *caslObjectEditorState) saveDevice() {
	number, err := parseCASLEditorInt64(s.deviceNumberEntry.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("номер обладнання: %w", err), s.win)
		return
	}
	timeout, err := parseCASLEditorInt64(s.deviceTimeoutEntry.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("timeout: %w", err), s.win)
		return
	}
	changeDate, err := dateEntryUnixMilli(s.deviceChangeDateEntry)
	if err != nil {
		dialog.ShowError(fmt.Errorf("change date: %w", err), s.win)
		return
	}
	reglamentDate, err := dateEntryUnixMilli(s.deviceReglamentDateEntry)
	if err != nil {
		dialog.ShowError(fmt.Errorf("reglament date: %w", err), s.win)
		return
	}

	update := contracts.CASLDeviceUpdate{
		DeviceID:          s.snapshot.Object.Device.DeviceID,
		Number:            number,
		Name:              strings.TrimSpace(s.deviceNameEntry.Text),
		DeviceType:        mappedOptionValue(s.deviceTypeSelect.Selected, s.deviceTypeOptionToID),
		Timeout:           timeout,
		SIM1:              strings.TrimSpace(s.deviceSIM1Entry.Text),
		SIM2:              strings.TrimSpace(s.deviceSIM2Entry.Text),
		TechnicianID:      s.techOptionToID[s.deviceTechnicianSelect.Selected],
		Units:             strings.TrimSpace(s.deviceUnitsEntry.Text),
		Requisites:        strings.TrimSpace(s.deviceRequisitesEntry.Text),
		ChangeDate:        changeDate,
		ReglamentDate:     reglamentDate,
		LicenceKey:        strings.TrimSpace(s.deviceLicenceEntry.Text),
		PasswRemote:       strings.TrimSpace(s.deviceRemotePassEntry.Text),
		MoreAlarmTime:     s.snapshot.Object.Device.MoreAlarmTime,
		IgnoringAlarmTime: s.snapshot.Object.Device.IgnoringAlarmTime,
	}
	s.runMutation("Збереження обладнання...", func(ctx context.Context) error {
		return s.provider.UpdateCASLDevice(ctx, update)
	})
}

func (s *caslObjectEditorState) currentDeviceCreate() (contracts.CASLDeviceCreate, error) {
	number, err := parseCASLEditorInt64(s.deviceNumberEntry.Text)
	if err != nil {
		return contracts.CASLDeviceCreate{}, fmt.Errorf("номер обладнання: %w", err)
	}
	if number <= 0 {
		return contracts.CASLDeviceCreate{}, fmt.Errorf("вкажіть номер обладнання")
	}
	timeout, err := parseCASLEditorInt64(s.deviceTimeoutEntry.Text)
	if err != nil {
		return contracts.CASLDeviceCreate{}, fmt.Errorf("timeout: %w", err)
	}
	changeDate, err := dateEntryUnixMilli(s.deviceChangeDateEntry)
	if err != nil {
		return contracts.CASLDeviceCreate{}, fmt.Errorf("change date: %w", err)
	}
	reglamentDate, err := dateEntryUnixMilli(s.deviceReglamentDateEntry)
	if err != nil {
		return contracts.CASLDeviceCreate{}, fmt.Errorf("reglament date: %w", err)
	}
	return contracts.CASLDeviceCreate{
		Number:            number,
		Name:              strings.TrimSpace(s.deviceNameEntry.Text),
		DeviceType:        mappedOptionValue(s.deviceTypeSelect.Selected, s.deviceTypeOptionToID),
		Timeout:           timeout,
		SIM1:              strings.TrimSpace(s.deviceSIM1Entry.Text),
		SIM2:              strings.TrimSpace(s.deviceSIM2Entry.Text),
		TechnicianID:      s.techOptionToID[s.deviceTechnicianSelect.Selected],
		Units:             strings.TrimSpace(s.deviceUnitsEntry.Text),
		Requisites:        strings.TrimSpace(s.deviceRequisitesEntry.Text),
		ChangeDate:        changeDate,
		ReglamentDate:     reglamentDate,
		LicenceKey:        strings.TrimSpace(s.deviceLicenceEntry.Text),
		PasswRemote:       strings.TrimSpace(s.deviceRemotePassEntry.Text),
		MoreAlarmTime:     s.snapshot.Object.Device.MoreAlarmTime,
		IgnoringAlarmTime: s.snapshot.Object.Device.IgnoringAlarmTime,
	}, nil
}

func (s *caslObjectEditorState) saveLine() {
	if !s.ensureDeviceCreated("Спочатку створіть обладнання, а потім редагуйте зони.") {
		return
	}
	line, ok := s.selectedLine()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть зону у списку.")
		return
	}
	mutation, err := s.currentLineMutation()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	mutation.LineID = line.LineID
	s.runMutation("Збереження зони...", func(ctx context.Context) error {
		return s.provider.UpdateCASLDeviceLine(ctx, mutation)
	})
}

func (s *caslObjectEditorState) createLine() {
	if !s.ensureDeviceCreated("Спочатку створіть обладнання, а потім додавайте зони.") {
		return
	}
	mutation, err := s.currentLineMutation()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	mutation.LineID = nil
	s.runMutation("Створення зони...", func(ctx context.Context) error {
		return s.provider.CreateCASLDeviceLine(ctx, mutation)
	})
}

func (s *caslObjectEditorState) createQuickLine() {
	if !s.ensureDeviceCreated("Спочатку створіть обладнання, а потім додавайте зони.") {
		return
	}
	name := strings.TrimSpace(s.quickLineNameEntry.Text)
	if name == "" {
		ShowInfoDialog(s.win, "Некоректні дані", "Вкажіть назву нової зони.")
		return
	}

	template := s.defaultLineTemplate()
	lineNumber := s.nextAvailableLineNumber()
	mutation := contracts.CASLDeviceLineMutation{
		DeviceID:      s.snapshot.Object.Device.DeviceID,
		LineNumber:    lineNumber,
		GroupNumber:   template.GroupNumber,
		AdapterType:   template.AdapterType,
		AdapterNumber: template.AdapterNumber,
		Description:   name,
		LineType:      mappedOptionValue(strings.TrimSpace(s.quickLineTypeEntry.Text), s.lineTypeOptionToID),
		IsBlocked:     false,
	}
	if strings.TrimSpace(mutation.LineType) == "" {
		mutation.LineType = "NORMAL"
	}
	if strings.TrimSpace(mutation.AdapterType) == "" {
		mutation.AdapterType = "SYS"
	}
	if mutation.GroupNumber <= 0 {
		mutation.GroupNumber = 1
	}

	s.pendingLineNumber = lineNumber
	s.pendingFocusQuickLine = true
	s.runMutationWithSuccess("Створення нової зони...", func(ctx context.Context) error {
		return s.provider.CreateCASLDeviceLine(ctx, mutation)
	}, func() {
		s.quickLineNameEntry.SetText("")
		s.refreshQuickLineHint()
	})
}

func (s *caslObjectEditorState) bindLineToRoom() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення, а потім прив'язуйте зони.") {
		return
	}
	if !s.ensureDeviceCreated("Спочатку створіть обладнання, а потім прив'язуйте зони.") {
		return
	}
	line, ok := s.selectedLine()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть зону у списку.")
		return
	}
	roomID := s.roomOptionToID[s.lineRoomSelect.Selected]
	if strings.TrimSpace(roomID) == "" {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть приміщення для прив'язки.")
		return
	}
	binding := contracts.CASLLineToRoomBinding{
		ObjID:      s.snapshot.Object.ObjID,
		DeviceID:   s.snapshot.Object.Device.DeviceID,
		LineNumber: line.LineNumber,
		RoomID:     roomID,
	}
	s.runMutation("Прив'язка зони до приміщення...", func(ctx context.Context) error {
		return s.provider.AddCASLLineToRoom(ctx, binding)
	})
}

func (s *caslObjectEditorState) addUserToRoom() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення, а потім додавайте користувачів.") {
		return
	}
	room, ok := s.selectedRoom()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть приміщення.")
		return
	}
	userID := s.userOptionToID[s.roomUserSelect.Selected]
	if strings.TrimSpace(userID) == "" {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть користувача.")
		return
	}
	request := contracts.CASLAddUserToRoomRequest{
		ObjID:    s.snapshot.Object.ObjID,
		RoomID:   room.RoomID,
		UserID:   userID,
		Priority: len(s.roomUsersLocal) + 1,
	}
	previous := append([]contracts.CASLRoomUserLink(nil), s.roomUsersLocal...)
	s.roomUsersLocal = append(s.roomUsersLocal, contracts.CASLRoomUserLink{
		UserID:   userID,
		Priority: request.Priority,
	})
	s.syncCurrentRoomUsers()
	s.refreshRoomUsersUI(len(s.roomUsersLocal) - 1)
	s.runMutationWithHooks("Додавання користувача...", func(ctx context.Context) error {
		return s.provider.AddCASLUserToRoom(ctx, request)
	}, nil, func(error) {
		s.roomUsersLocal = previous
		s.syncCurrentRoomUsers()
		s.refreshRoomUsersUI(minListIndex(s.roomUserSelected, len(s.roomUsersLocal)-1))
	})
}

func (s *caslObjectEditorState) removeUserFromRoom() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення.") {
		return
	}
	room, ok := s.selectedRoom()
	if !ok || s.roomUserSelected < 0 || s.roomUserSelected >= len(s.roomUsersLocal) {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть користувача у списку приміщення.")
		return
	}
	request := contracts.CASLRemoveUserFromRoomRequest{
		ObjID:  s.snapshot.Object.ObjID,
		RoomID: room.RoomID,
		UserID: s.roomUsersLocal[s.roomUserSelected].UserID,
	}
	previous := append([]contracts.CASLRoomUserLink(nil), s.roomUsersLocal...)
	nextSelected := s.roomUserSelected
	s.roomUsersLocal = append(s.roomUsersLocal[:s.roomUserSelected], s.roomUsersLocal[s.roomUserSelected+1:]...)
	if nextSelected >= len(s.roomUsersLocal) {
		nextSelected = len(s.roomUsersLocal) - 1
	}
	s.syncCurrentRoomUsers()
	s.refreshRoomUsersUI(nextSelected)
	s.runMutationWithHooks("Видалення користувача...", func(ctx context.Context) error {
		return s.provider.RemoveCASLUserFromRoom(ctx, request)
	}, nil, func(error) {
		s.roomUsersLocal = previous
		s.syncCurrentRoomUsers()
		s.refreshRoomUsersUI(minListIndex(nextSelected, len(s.roomUsersLocal)-1))
	})
}

func (s *caslObjectEditorState) moveRoomUserUp() {
	if s.roomUserSelected <= 0 || s.roomUserSelected >= len(s.roomUsersLocal) {
		return
	}
	s.roomUsersLocal[s.roomUserSelected-1], s.roomUsersLocal[s.roomUserSelected] = s.roomUsersLocal[s.roomUserSelected], s.roomUsersLocal[s.roomUserSelected-1]
	s.roomUserSelected--
	s.syncCurrentRoomUsers()
	s.refreshRoomUsersUI(s.roomUserSelected)
}

func (s *caslObjectEditorState) moveRoomUserDown() {
	if s.roomUserSelected < 0 || s.roomUserSelected >= len(s.roomUsersLocal)-1 {
		return
	}
	s.roomUsersLocal[s.roomUserSelected+1], s.roomUsersLocal[s.roomUserSelected] = s.roomUsersLocal[s.roomUserSelected], s.roomUsersLocal[s.roomUserSelected+1]
	s.roomUserSelected++
	s.syncCurrentRoomUsers()
	s.refreshRoomUsersUI(s.roomUserSelected)
}

func (s *caslObjectEditorState) saveRoomUserPriorities() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення.") {
		return
	}
	room, ok := s.selectedRoom()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть приміщення.")
		return
	}
	items := make([]contracts.CASLRoomUserPriority, 0, len(s.roomUsersLocal))
	for idx, user := range s.roomUsersLocal {
		items = append(items, contracts.CASLRoomUserPriority{
			UserID:   user.UserID,
			RoomID:   room.RoomID,
			Priority: idx + 1,
			HozNum:   user.HozNum,
		})
	}
	s.runMutation("Збереження порядку користувачів...", func(ctx context.Context) error {
		return s.provider.UpdateCASLRoomUserPriorities(ctx, s.objectID, items)
	})
}

func (s *caslObjectEditorState) createUserAndAddToRoom() {
	if !s.ensureObjectCreated("Спочатку створіть об'єкт і приміщення.") {
		return
	}
	room, ok := s.selectedRoom()
	if !ok {
		ShowInfoDialog(s.win, "Не вибрано", "Оберіть приміщення.")
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
		request := contracts.CASLUserCreateRequest{
			LastName:   strings.TrimSpace(lastNameEntry.Text),
			FirstName:  strings.TrimSpace(firstNameEntry.Text),
			MiddleName: strings.TrimSpace(middleNameEntry.Text),
			Tag:        strings.TrimSpace(tagEntry.Text),
			Role:       strings.TrimSpace(roleSelect.Selected),
			Email:      strings.TrimSpace(emailEntry.Text),
			Password:   strings.TrimSpace(passwordEntry.Text),
			PhoneNumbers: []contracts.CASLPhoneNumber{
				{Active: true, Number: strings.TrimSpace(phone1Entry.Text)},
				{Active: false, Number: strings.TrimSpace(phone2Entry.Text)},
			},
		}

		s.setStatus("Створення користувача...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			user, err := s.provider.CreateCASLUser(ctx, request)
			if err == nil && strings.TrimSpace(user.UserID) != "" {
				err = s.provider.AddCASLUserToRoom(ctx, contracts.CASLAddUserToRoomRequest{
					ObjID:    s.snapshot.Object.ObjID,
					RoomID:   room.RoomID,
					UserID:   user.UserID,
					Priority: len(s.roomUsersLocal) + 1,
				})
			}

			fyne.Do(func() {
				if err != nil {
					s.setStatus("Помилка створення користувача")
					dialog.ShowError(err, s.win)
					return
				}
				if strings.TrimSpace(user.UserID) != "" {
					s.snapshot.Users = append(s.snapshot.Users, user)
					s.roomUsersLocal = append(s.roomUsersLocal, contracts.CASLRoomUserLink{
						UserID:   user.UserID,
						Priority: len(s.roomUsersLocal) + 1,
					})
					s.syncCurrentRoomUsers()
					s.refreshRoomUserOptions(s.roomUserSearchEntry.Text)
					s.refreshRoomUsersUI(len(s.roomUsersLocal) - 1)
				}
				s.setStatus("Користувача створено")
				if s.onChanged != nil {
					s.onChanged()
				}
				s.reload()
			})
		}()
	}, s.win)
}

func (s *caslObjectEditorState) currentLineMutation() (contracts.CASLDeviceLineMutation, error) {
	if strings.TrimSpace(s.snapshot.Object.Device.DeviceID) == "" {
		return contracts.CASLDeviceLineMutation{}, fmt.Errorf("спочатку створіть обладнання")
	}
	lineNumber, err := parseCASLEditorInt(s.lineNumberEntry.Text)
	if err != nil {
		return contracts.CASLDeviceLineMutation{}, fmt.Errorf("номер зони: %w", err)
	}
	groupNumber, err := parseCASLEditorInt(s.lineGroupNumberEntry.Text)
	if err != nil {
		return contracts.CASLDeviceLineMutation{}, fmt.Errorf("group number: %w", err)
	}
	adapterNumber, err := parseCASLEditorInt(s.lineAdapterNumberEntry.Text)
	if err != nil {
		return contracts.CASLDeviceLineMutation{}, fmt.Errorf("adapter number: %w", err)
	}
	return contracts.CASLDeviceLineMutation{
		DeviceID:      s.snapshot.Object.Device.DeviceID,
		LineNumber:    lineNumber,
		GroupNumber:   groupNumber,
		AdapterType:   mappedOptionValue(s.lineAdapterTypeSelect.Selected, s.adapterOptionToID),
		AdapterNumber: adapterNumber,
		Description:   strings.TrimSpace(s.lineDescriptionEntry.Text),
		LineType:      mappedOptionValue(strings.TrimSpace(s.lineTypeEntry.Text), s.lineTypeOptionToID),
		IsBlocked:     s.lineBlockedCheck.Checked,
	}, nil
}

func (s *caslObjectEditorState) runMutation(started string, fn func(ctx context.Context) error) {
	s.runMutationWithHooks(started, fn, nil, nil)
}

func (s *caslObjectEditorState) runMutationWithSuccess(started string, fn func(ctx context.Context) error, onSuccess func()) {
	s.runMutationWithHooks(started, fn, onSuccess, nil)
}

func (s *caslObjectEditorState) runMutationWithHooks(started string, fn func(ctx context.Context) error, onSuccess func(), onError func(error)) {
	s.setStatus(started)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := fn(ctx)
		fyne.Do(func() {
			if err != nil {
				if onError != nil {
					onError(err)
				}
				s.setStatus("Помилка")
				dialog.ShowError(err, s.win)
				return
			}
			if onSuccess != nil {
				onSuccess()
			}
			s.setStatus("Збережено")
			if s.onChanged != nil {
				s.onChanged()
			}
			s.reload()
		})
	}()
}

func (s *caslObjectEditorState) selectedRoom() (contracts.CASLRoomDetails, bool) {
	if s.roomSelected < 0 || s.roomSelected >= len(s.snapshot.Object.Rooms) {
		return contracts.CASLRoomDetails{}, false
	}
	return s.snapshot.Object.Rooms[s.roomSelected], true
}

func (s *caslObjectEditorState) selectedLine() (contracts.CASLDeviceLineDetails, bool) {
	if s.lineSelected < 0 || s.lineSelected >= len(s.snapshot.Object.Device.Lines) {
		return contracts.CASLDeviceLineDetails{}, false
	}
	return s.snapshot.Object.Device.Lines[s.lineSelected], true
}

func (s *caslObjectEditorState) hasObject() bool {
	return strings.TrimSpace(s.snapshot.Object.ObjID) != "" || s.objectID > 0
}

func (s *caslObjectEditorState) hasDevice() bool {
	return strings.TrimSpace(s.snapshot.Object.Device.DeviceID) != ""
}

func (s *caslObjectEditorState) ensureObjectCreated(message string) bool {
	if s.hasObject() {
		return true
	}
	ShowInfoDialog(s.win, "Спочатку створіть об'єкт", message)
	return false
}

func (s *caslObjectEditorState) ensureDeviceCreated(message string) bool {
	if s.hasDevice() {
		return true
	}
	ShowInfoDialog(s.win, "Спочатку створіть обладнання", message)
	return false
}

func (s *caslObjectEditorState) setStatus(text string) {
	s.statusLabel.SetText(text)
}

func (s *caslObjectEditorState) refreshWindowPresentation() {
	if s.hasObject() {
		title := fmt.Sprintf("CASL: %s [%s]", firstNonEmpty(s.snapshot.Object.Name, "Об'єкт"), firstNonEmpty(s.snapshot.Object.ObjID, strconv.FormatInt(s.objectID, 10)))
		s.win.SetTitle(title)
		s.headerLabel.SetText(fmt.Sprintf("CASL-редактор для об'єкта %s", firstNonEmpty(s.snapshot.Object.ObjID, strconv.FormatInt(s.objectID, 10))))
	} else {
		s.win.SetTitle("CASL: Створення нового об'єкта")
		s.headerLabel.SetText("Створення нового CASL-об'єкта")
	}
	if s.objectSaveBtn != nil {
		if s.hasObject() {
			s.objectSaveBtn.SetText("Зберегти об'єкт")
		} else {
			s.objectSaveBtn.SetText("Створити об'єкт")
		}
	}
	if s.deviceSaveBtn != nil {
		if s.hasDevice() {
			s.deviceSaveBtn.SetText("Зберегти обладнання")
		} else {
			s.deviceSaveBtn.SetText("Створити обладнання")
		}
	}
}

func (s *caslObjectEditorState) userLabelByID(userID string) string {
	for _, user := range s.snapshot.Users {
		if strings.TrimSpace(user.UserID) == strings.TrimSpace(userID) {
			return fmt.Sprintf("%s [%s]", caslProfileName(user), user.UserID)
		}
	}
	return "Користувач [" + strings.TrimSpace(userID) + "]"
}

func (s *caslObjectEditorState) userOptionByID(userID string, options map[string]string) string {
	for label, id := range options {
		if strings.TrimSpace(id) == strings.TrimSpace(userID) {
			return label
		}
	}
	return ""
}

func (s *caslObjectEditorState) pultOptionByID(pultID string) string {
	for label, id := range s.pultOptionToID {
		if strings.TrimSpace(id) == strings.TrimSpace(pultID) {
			return label
		}
	}
	return ""
}

func (s *caslObjectEditorState) roomOptionByID(roomID string) string {
	for label, id := range s.roomOptionToID {
		if strings.TrimSpace(id) == strings.TrimSpace(roomID) {
			return label
		}
	}
	return ""
}

func (s *caslObjectEditorState) roomNameByID(roomID string) string {
	for _, room := range s.snapshot.Object.Rooms {
		if strings.TrimSpace(room.RoomID) == strings.TrimSpace(roomID) {
			return room.Name
		}
	}
	return ""
}

func (s *caslObjectEditorState) displayLineType(raw string) string {
	return caslLineTypeDisplayNameWithDict(s.snapshot.Dictionary, raw)
}

func (s *caslObjectEditorState) displayAdapterType(raw string) string {
	return caslAdapterTypeDisplayNameWithDict(s.snapshot.Dictionary, raw)
}

func (s *caslObjectEditorState) displayDeviceType(raw string) string {
	return caslDeviceTypeDisplayNameWithDict(s.snapshot.Dictionary, raw)
}

func (s *caslObjectEditorState) currentRoomImages() []string {
	if room, ok := s.selectedRoom(); ok {
		return append([]string(nil), room.Images...)
	}
	return nil
}

func (s *caslObjectEditorState) pickObjectCoordinatesOnMap() {
	showCoordinatesMapPickerWithOptions(
		s.win,
		strings.TrimSpace(s.objectLatEntry.Text),
		strings.TrimSpace(s.objectLongEntry.Text),
		coordinatesMapPickerOptions{
			Title:           "Вибір координат об'єкта",
			InitialAddress:  strings.TrimSpace(s.objectAddressEntry.Text),
			ForceLvivCenter: true,
		},
		func(lat, lon string) {
			s.objectLatEntry.SetText(lat)
			s.objectLongEntry.SetText(lon)
			s.setStatus("Координати вибрано на карті")
		},
	)
}

func (s *caslObjectEditorState) objectScopedImages() []string {
	if len(s.snapshot.Object.Images) == 0 {
		return nil
	}

	roomImageCounts := make(map[string]int)
	for _, room := range s.snapshot.Object.Rooms {
		for _, image := range room.Images {
			image = strings.TrimSpace(image)
			if image == "" {
				continue
			}
			roomImageCounts[image]++
		}
	}

	result := make([]string, 0, len(s.snapshot.Object.Images))
	for _, image := range s.snapshot.Object.Images {
		image = strings.TrimSpace(image)
		if image == "" {
			continue
		}
		if roomImageCounts[image] > 0 {
			roomImageCounts[image]--
			continue
		}
		result = append(result, image)
	}
	return result
}

func (s *caslObjectEditorState) syncCurrentRoomUsers() {
	if s.roomSelected < 0 || s.roomSelected >= len(s.snapshot.Object.Rooms) {
		return
	}
	s.snapshot.Object.Rooms[s.roomSelected].Users = append([]contracts.CASLRoomUserLink(nil), s.roomUsersLocal...)
}

func (s *caslObjectEditorState) refreshRoomUsersUI(selected int) {
	s.roomUsersList.Refresh()
	s.refreshRoomUserOptions(s.roomUserSearchEntry.Text)
	s.applyRoomUsersListHeights()
	if len(s.roomUsersLocal) == 0 {
		s.roomUserSelected = -1
		s.roomUsersList.UnselectAll()
		return
	}
	selected = minListIndex(selected, len(s.roomUsersLocal)-1)
	s.roomUserSelected = selected
	s.roomUsersList.Select(selected)
}

func (s *caslObjectEditorState) refreshObjectImages() {
	setCASLImageStrip(s.objectImagesBox, s.objectScopedImages(), "Немає фото об'єкта", "Об'єкт", s.provider, s.deleteObjectImage, s.showImagePreview)
}

func (s *caslObjectEditorState) refreshRoomImages(images []string) {
	roomLabel := "Приміщення"
	if room, ok := s.selectedRoom(); ok && strings.TrimSpace(room.Name) != "" {
		roomLabel = room.Name
	}
	setCASLImageStrip(s.roomImagesBox, images, "Немає фото приміщення", roomLabel, s.provider, s.deleteRoomImage, s.showImagePreview)
}

func (s *caslObjectEditorState) showImagePreview(title string, raw string) {
	holder := container.NewCenter(widget.NewLabel("Завантаження фото..."))
	scroll := container.NewScroll(holder)
	zoomLabel := widget.NewLabel("Масштаб: 100%")
	zoomOutBtn := widget.NewButton("－", func() {})
	zoomInBtn := widget.NewButton("＋", func() {})
	controlsBar := container.NewHBox(
		zoomOutBtn,
		zoomInBtn,
		layout.NewSpacer(),
		zoomLabel,
	)
	content := container.NewBorder(
		nil,
		controlsBar,
		nil,
		nil,
		container.NewPadded(scroll),
	)
	dlg := dialog.NewCustom(title, "Закрити", content, s.win)
	dlg.Resize(fyne.NewSize(980, 760))
	dlg.Show()

	go func() {
		resource, err := s.resolveImagePreviewResource(raw, "casl-preview")
		fyne.Do(func() {
			if err != nil {
				holder.Objects = []fyne.CanvasObject{
					container.NewCenter(widget.NewLabel("Не вдалося завантажити фото")),
				}
				holder.Refresh()
				return
			}

			baseSize := fyne.NewSize(880, 620)
			zoom := 1.0
			const (
				minZoom  = 0.25
				maxZoom  = 5.0
				zoomStep = 0.15
			)

			image := canvas.NewImageFromResource(resource)
			image.FillMode = canvas.ImageFillContain
			image.ScaleMode = canvas.ImageScaleSmooth
			image.SetMinSize(baseSize)

			updateZoomLabel := func() {
				zoomLabel.SetText(fmt.Sprintf("Масштаб: %d%%", int(math.Round(zoom*100))))
			}
			applyZoom := func(nextZoom float64) {
				if nextZoom < minZoom {
					nextZoom = minZoom
				}
				if nextZoom > maxZoom {
					nextZoom = maxZoom
				}
				zoom = nextZoom
				image.SetMinSize(fyne.NewSize(
					float32(float64(baseSize.Width)*zoom),
					float32(float64(baseSize.Height)*zoom),
				))
				image.Refresh()
				updateZoomLabel()
			}

			interaction := newMapInteractionSurface()
			interaction.onScrolled = func(ev *fyne.ScrollEvent) {
				delta := ev.Scrolled.DY
				if math.Abs(float64(ev.Scrolled.DX)) > math.Abs(float64(delta)) {
					delta = ev.Scrolled.DX
				}
				switch {
				case delta > 0:
					applyZoom(zoom + zoomStep)
				case delta < 0:
					applyZoom(zoom - zoomStep)
				}
			}

			zoomOutBtn.OnTapped = func() {
				applyZoom(zoom - zoomStep)
			}
			zoomInBtn.OnTapped = func() {
				applyZoom(zoom + zoomStep)
			}

			updateZoomLabel()
			holder.Objects = []fyne.CanvasObject{
				container.NewStack(
					container.NewCenter(image),
					interaction,
				),
			}
			holder.Refresh()
		})
	}()
}

func (s *caslObjectEditorState) resolveImagePreviewResource(raw string, name string) (fyne.Resource, error) {
	if resource, err := caslImageResource(raw, name); err == nil {
		return resource, nil
	}
	imageID := caslImageID(raw)
	if imageID == "" || s.provider == nil {
		return nil, fmt.Errorf("unsupported image preview source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	body, err := s.provider.FetchCASLImagePreview(ctx, imageID)
	if err != nil {
		return nil, err
	}
	return caslImageResourceFromBytes(body, name)
}

func (s *caslObjectEditorState) roomUserDetailsText(item contracts.CASLRoomUserLink) string {
	user := s.userByID(item.UserID)
	parts := []string{fmt.Sprintf("Пріоритет: %d", item.Priority)}
	if role := strings.TrimSpace(user.Role); role != "" {
		parts = append(parts, "Роль: "+role)
	}
	phones := make([]string, 0, len(user.PhoneNumbers))
	for _, phone := range user.PhoneNumbers {
		number := strings.TrimSpace(phone.Number)
		if number != "" {
			phones = append(phones, number)
		}
	}
	if len(phones) > 0 {
		parts = append(parts, "Тел: "+strings.Join(phones, ", "))
	}
	if hozNum := strings.TrimSpace(item.HozNum); hozNum != "" {
		parts = append(parts, "Гос. номер: "+hozNum)
	}
	return strings.Join(parts, " | ")
}

func (s *caslObjectEditorState) userByID(userID string) contracts.CASLUserProfile {
	for _, user := range s.snapshot.Users {
		if strings.TrimSpace(user.UserID) == strings.TrimSpace(userID) {
			return user
		}
	}
	return contracts.CASLUserProfile{UserID: strings.TrimSpace(userID)}
}

func (s *caslObjectEditorState) refreshRoomUserOptions(filter string) {
	filter = strings.ToLower(strings.TrimSpace(filter))
	options := []string{""}
	for _, label := range s.allUserOptions {
		if label == "" {
			continue
		}
		userID := s.userOptionToID[label]
		user := s.userByID(userID)
		searchBlob := strings.ToLower(strings.Join([]string{
			label,
			user.Email,
			user.Tag,
			user.Role,
			phoneNumbersText(user.PhoneNumbers),
		}, " "))
		if filter != "" && !strings.Contains(searchBlob, filter) {
			continue
		}
		options = append(options, label)
	}
	s.roomUserSelect.Options = options
	s.roomUserSelect.Refresh()
	if len(options) > 1 {
		if !containsString(options, s.roomUserSelect.Selected) {
			s.roomUserSelect.SetSelected(options[1])
		}
	} else {
		s.roomUserSelect.SetSelected("")
	}
}

func (s *caslObjectEditorState) nextAvailableLineNumber() int {
	used := make(map[int]struct{}, len(s.snapshot.Object.Device.Lines))
	for _, line := range s.snapshot.Object.Device.Lines {
		if line.LineNumber > 0 {
			used[line.LineNumber] = struct{}{}
		}
	}
	for n := 1; ; n++ {
		if _, exists := used[n]; !exists {
			return n
		}
	}
}

func (s *caslObjectEditorState) defaultLineTemplate() contracts.CASLDeviceLineDetails {
	if line, ok := s.selectedLine(); ok {
		return line
	}
	if len(s.snapshot.Object.Device.Lines) > 0 {
		return s.snapshot.Object.Device.Lines[len(s.snapshot.Object.Device.Lines)-1]
	}
	return contracts.CASLDeviceLineDetails{
		GroupNumber:   1,
		AdapterType:   "SYS",
		AdapterNumber: 0,
		LineType:      "NORMAL",
	}
}

func (s *caslObjectEditorState) refreshQuickLineHint() {
	nextNumber := s.nextAvailableLineNumber()
	s.quickLineHintLabel.SetText(fmt.Sprintf("Наступний вільний номер зони: #%d", nextNumber))
}

func (s *caslObjectEditorState) findLineIndexByNumber(lineNumber int) int {
	for idx, line := range s.snapshot.Object.Device.Lines {
		if line.LineNumber == lineNumber {
			return idx
		}
	}
	return -1
}

func (s *caslObjectEditorState) applyRoomListHeights() {
	if s.roomList == nil {
		return
	}
	for idx := range s.snapshot.Object.Rooms {
		s.roomList.SetItemHeight(idx, 60)
	}
}

func (s *caslObjectEditorState) applyRoomUsersListHeights() {
	if s.roomUsersList == nil {
		return
	}
	for idx := range s.roomUsersLocal {
		s.roomUsersList.SetItemHeight(idx, 72)
	}
}

func (s *caslObjectEditorState) applyLineListHeights() {
	if s.lineList == nil {
		return
	}
	for idx := range s.snapshot.Object.Device.Lines {
		s.lineList.SetItemHeight(idx, 60)
	}
}

func newCASLListRowTemplate() fyne.CanvasObject {
	title := widget.NewLabel("")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	subtitle := widget.NewLabel("")
	subtitle.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	return container.NewVBox(title, subtitle)
}

func setCASLListRow(obj fyne.CanvasObject, title string, subtitle string) {
	box, ok := obj.(*fyne.Container)
	if !ok || len(box.Objects) < 2 {
		return
	}
	titleLabel, _ := box.Objects[0].(*widget.Label)
	subtitleLabel, _ := box.Objects[1].(*widget.Label)
	if titleLabel != nil {
		titleLabel.SetText(title)
	}
	if subtitleLabel != nil {
		subtitleLabel.SetText(subtitle)
	}
}

func caslProfileName(user contracts.CASLUserProfile) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{user.LastName, user.FirstName, user.MiddleName} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "Користувач"
	}
	return strings.Join(parts, " ")
}

func parseCASLEditorDate(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	parsed, err := time.ParseInLocation("02.01.2006", value, time.Local)
	if err != nil {
		return 0, err
	}
	return parsed.UnixMilli(), nil
}

func formatCASLEditorDate(raw int64) string {
	if raw <= 0 {
		return ""
	}
	return time.UnixMilli(raw).Format("02.01.2006")
}

func dateEntryUnixMilli(entry *widget.DateEntry) (int64, error) {
	if entry == nil {
		return 0, nil
	}
	if strings.TrimSpace(entry.Text) == "" {
		return 0, nil
	}
	if entry.Date == nil {
		return 0, fmt.Errorf("некоректна дата")
	}
	return time.Date(entry.Date.Year(), entry.Date.Month(), entry.Date.Day(), 0, 0, 0, 0, time.Local).UnixMilli(), nil
}

func caslDatePtr(raw int64) *time.Time {
	if raw <= 0 {
		return nil
	}
	value := time.UnixMilli(raw).Local()
	return &value
}

func parseCASLEditorInt64(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func parseCASLEditorInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseCASLEditorFloatPtr(raw string) (*float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func int64ToString(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func float64PtrToString(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func phoneNumbersText(phones []contracts.CASLPhoneNumber) string {
	values := make([]string, 0, len(phones))
	for _, phone := range phones {
		number := strings.TrimSpace(phone.Number)
		if number != "" {
			values = append(values, number)
		}
	}
	return strings.Join(values, " ")
}

func fixedMinHeightArea(height float32, content fyne.CanvasObject) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, height))
	return container.NewStack(spacer, content)
}

type caslImageTapTarget struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newCASLImageTapTarget(content fyne.CanvasObject, onTap func()) *caslImageTapTarget {
	target := &caslImageTapTarget{
		content: content,
		onTap:   onTap,
	}
	target.ExtendBaseWidget(target)
	return target
}

func (t *caslImageTapTarget) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *caslImageTapTarget) TappedSecondary(*fyne.PointEvent) {}

func (t *caslImageTapTarget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func setCASLImageStrip(box *fyne.Container, images []string, emptyText string, ownerLabel string, provider contracts.CASLObjectEditorProvider, onDelete func(string), onPreview func(string, string)) {
	if box == nil {
		return
	}
	items := make([]fyne.CanvasObject, 0, maxInt(len(images), 1))
	for idx, raw := range images {
		if tile := newCASLImageTile(raw, idx+1, ownerLabel, provider, onDelete, onPreview); tile != nil {
			items = append(items, tile)
		}
	}
	if len(items) == 0 {
		items = append(items, widget.NewCard("", "", container.NewCenter(widget.NewLabel(emptyText))))
	}
	box.Objects = items
	box.Refresh()
}

func newCASLImageTile(raw string, index int, ownerLabel string, provider contracts.CASLObjectEditorProvider, onDelete func(string), onPreview func(string, string)) fyne.CanvasObject {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "null") {
		return nil
	}
	imageID := caslImageID(raw)
	var deleteRow fyne.CanvasObject
	if onDelete != nil && imageID != "" {
		deleteRow = container.NewHBox(
			layout.NewSpacer(),
			widget.NewButton("Видалити", func() { onDelete(imageID) }),
		)
	}
	ownerLabel = firstNonEmpty(ownerLabel, "Фото")
	previewTitle := fmt.Sprintf("%s, фото %d", ownerLabel, index)
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(190, 120))
	previewHolder := container.NewStack(spacer)
	resource, err := caslImageResource(raw, fmt.Sprintf("casl-image-%d", index))
	if err == nil {
		image := canvas.NewImageFromResource(resource)
		image.FillMode = canvas.ImageFillContain
		image.ScaleMode = canvas.ImageScaleFastest
		image.SetMinSize(fyne.NewSize(180, 110))
		previewHolder.Objects = []fyne.CanvasObject{
			newCASLImageTapTarget(container.NewPadded(image), func() {
				if onPreview != nil {
					onPreview(previewTitle, raw)
				}
			}),
		}
	} else if imageID != "" && provider != nil {
		status := widget.NewLabel("Завантаження прев'ю...")
		status.Wrapping = fyne.TextWrapWord
		previewHolder.Objects = []fyne.CanvasObject{
			newCASLImageTapTarget(container.NewCenter(status), func() {
				if onPreview != nil {
					onPreview(previewTitle, raw)
				}
			}),
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			body, fetchErr := provider.FetchCASLImagePreview(ctx, imageID)
			fyne.Do(func() {
				if fetchErr != nil {
					status.SetText("Не вдалося завантажити прев'ю\nID: " + imageID)
					return
				}
				resource, resourceErr := caslImageResourceFromBytes(body, fmt.Sprintf("casl-image-%d", index))
				if resourceErr != nil {
					status.SetText("Некоректне зображення\nID: " + imageID)
					return
				}
				image := canvas.NewImageFromResource(resource)
				image.FillMode = canvas.ImageFillContain
				image.ScaleMode = canvas.ImageScaleFastest
				image.SetMinSize(fyne.NewSize(180, 110))
				previewHolder.Objects = []fyne.CanvasObject{
					newCASLImageTapTarget(container.NewPadded(image), func() {
						if onPreview != nil {
							onPreview(previewTitle, raw)
						}
					}),
				}
				previewHolder.Refresh()
			})
		}()
	} else {
		label := widget.NewLabel(firstNonEmpty(raw, "Немає фото"))
		label.Wrapping = fyne.TextWrapWord
		previewHolder.Objects = []fyne.CanvasObject{container.NewCenter(label)}
	}
	caption := widget.NewLabel(ownerLabel)
	caption.Alignment = fyne.TextAlignCenter
	caption.TextStyle = fyne.TextStyle{Bold: true}
	content := container.NewBorder(caption, deleteRow, nil, nil, previewHolder)
	return widget.NewCard("", "Клікніть для перегляду", content)
}

func caslImageResource(raw string, name string) (fyne.Resource, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty image")
	}
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return nil, fmt.Errorf("unsupported image reference")
	}

	commaIdx := strings.Index(raw, ",")
	if commaIdx <= 5 {
		return nil, fmt.Errorf("invalid data url")
	}

	header := raw[:commaIdx]
	payload := strings.TrimSpace(raw[commaIdx+1:])
	mimeType := "image/jpeg"
	if strings.HasPrefix(strings.ToLower(header), "data:") {
		rest := strings.TrimSpace(header[5:])
		for _, separator := range []string{";", " "} {
			if idx := strings.Index(rest, separator); idx > 0 {
				rest = rest[:idx]
				break
			}
		}
		if parsed := strings.TrimSpace(rest); parsed != "" {
			mimeType = parsed
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, err
		}
	}

	exts, _ := mime.ExtensionsByType(mimeType)
	ext := ".img"
	if len(exts) > 0 && strings.TrimSpace(exts[0]) != "" {
		ext = exts[0]
	}
	return fyne.NewStaticResource(name+ext, decoded), nil
}

func caslImageResourceFromBytes(data []byte, name string) (fyne.Resource, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image data")
	}
	mimeType := http.DetectContentType(data)
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	exts, _ := mime.ExtensionsByType(mimeType)
	ext := ".img"
	if len(exts) > 0 && strings.TrimSpace(exts[0]) != "" {
		ext = exts[0]
	}
	return fyne.NewStaticResource(name+ext, data), nil
}

func caslImageID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return ""
	}
	return raw
}

func caslEncodeImageUpload(fileName string, data []byte) (string, string) {
	if len(data) == 0 {
		return "", ""
	}
	mimeType := strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName))))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}
	imageType := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(mimeType, "image/")))
	switch imageType {
	case "jpeg":
		imageType = "jpg"
	case "svg+xml":
		imageType = "svg"
	}
	return imageType, base64.StdEncoding.EncodeToString(data)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minListIndex(value int, max int) int {
	if max < 0 {
		return -1
	}
	if value < 0 {
		return 0
	}
	if value > max {
		return max
	}
	return value
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func mappedOptionValue(label string, mapping map[string]string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	if value, ok := mapping[label]; ok {
		return strings.TrimSpace(value)
	}
	return label
}

func optionLabelByValue(value string, mapping map[string]string) string {
	value = strings.TrimSpace(value)
	for label, mapped := range mapping {
		if strings.TrimSpace(mapped) == value {
			return label
		}
	}
	return value
}

func setRawOptionLabel(mapping map[string]string, raw string, label string) {
	if mapping == nil {
		return
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = caslLineTypeDisplayName(raw)
	}
	mapping[raw] = label
}

func ensureOptionMapping(mapping map[string]string, key string, fallbackName string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if mapping == nil {
		return
	}
	for _, existing := range mapping {
		if strings.TrimSpace(existing) == key {
			return
		}
	}
	label := strings.TrimSpace(fallbackName)
	if label == "" {
		label = key
	}
	base := label
	suffix := 2
	for {
		if existing, exists := mapping[label]; !exists {
			mapping[label] = key
			return
		} else if strings.TrimSpace(existing) == key {
			return
		}
		label = fmt.Sprintf("%s [%d]", base, suffix)
		suffix++
	}
}

func caslDefaultLineTypes() map[string]string {
	return map[string]string{
		"EMPTY":              "Пустий шлейф",
		"NORMAL":             "Нормальна зона",
		"ZONE_ALARM_ON_KZ":   "Тривожний шлейф",
		"ZONE_ALARM":         "Тривожний шлейф",
		"ZONE_ALM":           "Тривожний шлейф",
		"ALM_BTN":            "Тривожна кнопка",
		"ZONE_FIRE":          "Пожежний шлейф",
		"ZONE_NORMAL":        "Нормальна зона",
		"ZONE_COMMON":        "Нормальна зона",
		"ZONE_DELAY":         "Вхідний шлейф",
		"ZONE_PANIC":         "Тривожна кнопка",
		"UNTYPED_ZONE_ALARM": "Нетипізована тривожна зона",
	}
}

func caslLineTypeDisplayName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "—"
	}
	if label, ok := caslDefaultLineTypes()[raw]; ok {
		return label
	}

	upper := strings.ToUpper(raw)
	switch {
	case upper == "EMPTY":
		return "Пустий шлейф"
	case strings.HasPrefix(strings.ToLower(raw), "fire_pipeline"), strings.Contains(upper, "FIRE"):
		return "Пожежний шлейф"
	case strings.Contains(upper, "PANIC"), strings.Contains(upper, "ALM_BTN"):
		return "Тривожна кнопка"
	case strings.Contains(upper, "ALARM"), strings.Contains(upper, "ZONE_ALM"):
		return "Тривожний шлейф"
	case strings.Contains(upper, "NORMAL"), strings.Contains(upper, "COMMON"):
		return "Нормальна зона"
	case strings.Contains(upper, "UNTYPED"):
		return "Звичайна зона"
	case strings.Contains(upper, "DELAY"), strings.Contains(upper, "ENTRY"):
		return "Вхідний шлейф"
	default:
		return humanizeCASLToken(raw)
	}
}

func caslLineTypeDisplayNameWithDict(dict map[string]any, raw string) string {
	if translated := caslDictionaryTranslateLabel(dict, raw); translated != "" {
		return translated
	}
	return caslLineTypeDisplayName(raw)
}

func humanizeCASLToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	replacer := strings.NewReplacer("_", " ", "-", " ")
	text := strings.TrimSpace(replacer.Replace(raw))
	if text == "" {
		return raw
	}
	return strings.ToUpper(text[:1]) + strings.ToLower(text[1:])
}

func humanizeCASLAdapterType(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch strings.ToUpper(raw) {
	case "SYS":
		return "SYS"
	default:
		return humanizeCASLToken(raw)
	}
}

func caslAdapterTypeDisplayNameWithDict(dict map[string]any, raw string) string {
	if translated := caslDictionaryTranslateLabel(dict, raw); translated != "" {
		return translated
	}
	return humanizeCASLAdapterType(raw)
}

var caslDeviceTypesDisplayNames = map[string]string{
	"TYPE_DEVICE_CASL":                    "CASL",
	"TYPE_DEVICE_DUNAY_8L":                "Дунай-8L",
	"TYPE_DEVICE_DUNAY_16L":               "Дунай-16L",
	"TYPE_DEVICE_DUNAY_4L":                "Дунай-4L",
	"TYPE_DEVICE_LUN":                     "Лунь",
	"TYPE_DEVICE_AJAX":                    "Ajax",
	"TYPE_DEVICE_AJAX_SIA":                "Ajax(SIA)",
	"TYPE_DEVICE_BRON_SIA":                "Bron(SIA)",
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
	"TYPE_DEVICE_VBDB_2":                  "ВБД6-2 + G1R",
	"TYPE_DEVICE_VBD4":                    "ВБД4 + G1R",
	"TYPE_DEVICE_DUNAY_PSPN":              "ПСПН (R.COM)",
	"TYPE_DEVICE_DUNAY_PSPN_ECOM":         "ПСПН (ECOM)",
	"TYPE_DEVICE_VBD4_ECOM":               "ВБД4",
	"TYPE_DEVICE_VBD_16":                  "ВБД6-16",
}

func caslDeviceTypeDisplayName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if label, ok := caslDeviceTypesDisplayNames[strings.ToUpper(raw)]; ok {
		return label
	}
	trimmed := strings.TrimPrefix(raw, "TYPE_DEVICE_")
	if trimmed != raw {
		return humanizeCASLToken(trimmed)
	}
	return humanizeCASLToken(raw)
}

func caslDeviceTypeDisplayNameWithDict(dict map[string]any, raw string) string {
	if translated := caslDictionaryTranslateLabel(dict, raw); translated != "" {
		return translated
	}
	return caslDeviceTypeDisplayName(raw)
}

func labeledOptionMap(mapping map[string]string) ([]string, map[string]string) {
	if len(mapping) == 0 {
		return nil, map[string]string{}
	}
	options := make([]string, 0, len(mapping))
	result := make(map[string]string, len(mapping))
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		ensureOptionMapping(result, key, mapping[key])
	}
	for label := range result {
		options = append(options, label)
	}
	sort.Strings(options)
	return options, result
}

func caslDictionaryOptions(dict map[string]any, keys ...string) ([]string, map[string]string) {
	return labeledOptionMap(caslDictionaryOptionsMap(dict, keys...))
}

func caslLineTypeOptionsMap(dict map[string]any) map[string]string {
	lineTypes := caslDictionaryOptionsMap(dict, "line_types")
	if len(lineTypes) == 0 {
		lineTypes = caslDictionaryOptionsMap(dict, "zone_types")
	}
	if len(lineTypes) == 0 {
		lineTypes = caslDefaultLineTypes()
	}
	for rawType, label := range lineTypes {
		setRawOptionLabel(lineTypes, rawType, firstNonEmpty(label, caslLineTypeDisplayNameWithDict(dict, rawType)))
	}
	return lineTypes
}

func caslAdapterTypeOptionsMap(dict map[string]any) map[string]string {
	adapterTypes := caslDictionaryOptionsMap(dict, "adapters", "adapter_types")
	if len(adapterTypes) == 0 {
		adapterTypes = map[string]string{}
	}
	for rawType, label := range adapterTypes {
		setRawOptionLabel(adapterTypes, rawType, firstNonEmpty(label, caslAdapterTypeDisplayNameWithDict(dict, rawType)))
	}
	return adapterTypes
}

func caslDeviceTypeOptionsMap(dict map[string]any) map[string]string {
	deviceTypes := caslDictionaryOptionsMap(dict, "device_types", "user_device_types")
	if len(deviceTypes) == 0 {
		deviceTypes = map[string]string{}
	}
	if raw, ok := dict["devices"]; ok {
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				rawType := strings.TrimSpace(asStringAny(obj["type"]))
				if rawType == "" {
					continue
				}
				setRawOptionLabel(deviceTypes, rawType, caslDeviceTypeDisplayNameWithDict(dict, rawType))
			}
		}
	}
	for rawType, label := range deviceTypes {
		setRawOptionLabel(deviceTypes, rawType, firstNonEmpty(label, caslDeviceTypeDisplayNameWithDict(dict, rawType)))
	}
	return deviceTypes
}

func caslDictionaryOptionsMap(dict map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		if raw, ok := dict[key]; ok {
			if values := normalizeCASLObjectEditorOptionMap(raw); len(values) > 0 {
				return values
			}
		}
	}
	return map[string]string{}
}

func caslDictionaryTranslateLabel(dict map[string]any, key string) string {
	key = strings.TrimSpace(key)
	if key == "" || len(dict) == 0 {
		return ""
	}
	translateMap := caslDictionaryTranslateMap(dict)
	return strings.TrimSpace(translateMap[key])
}

func caslDictionaryTranslateMap(dict map[string]any) map[string]string {
	if len(dict) == 0 {
		return map[string]string{}
	}
	if translate, ok := dict["translate"]; ok {
		if mapping := caslDictionaryLanguageMap(translate, "uk"); len(mapping) > 0 {
			return mapping
		}
	}
	if nestedRaw, ok := dict["dictionary"]; ok {
		if nested, ok := nestedRaw.(map[string]any); ok {
			if translate, exists := nested["translate"]; exists {
				if mapping := caslDictionaryLanguageMap(translate, "uk"); len(mapping) > 0 {
					return mapping
				}
			}
		}
	}
	return map[string]string{}
}

func caslDictionaryLanguageMap(raw any, lang string) map[string]string {
	root, ok := raw.(map[string]any)
	if !ok || len(root) == 0 {
		return map[string]string{}
	}
	langCandidates := []string{
		strings.TrimSpace(lang),
		strings.ToLower(strings.TrimSpace(lang)),
		strings.ToUpper(strings.TrimSpace(lang)),
		"ua",
		"UA",
		"uk-UA",
		"uk_ua",
	}
	for _, key := range langCandidates {
		if nested, exists := root[key]; exists {
			return normalizeCASLObjectEditorOptionMap(nested)
		}
	}
	return map[string]string{}
}

func asStringAny(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func normalizeCASLObjectEditorOptionMap(raw any) map[string]string {
	switch typed := raw.(type) {
	case map[string]string:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			result[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
		return result
	case map[string]any:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			text := strings.TrimSpace(fmt.Sprint(value))
			if text == "" || text == "<nil>" {
				text = key
			}
			result[key] = text
		}
		return result
	case []string:
		result := make(map[string]string, len(typed))
		for _, value := range typed {
			value = strings.TrimSpace(value)
			if value != "" {
				result[value] = value
			}
		}
		return result
	case []any:
		result := make(map[string]string, len(typed))
		for _, value := range typed {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text == "" || text == "<nil>" {
				continue
			}
			result[text] = text
		}
		return result
	default:
		return map[string]string{}
	}
}
