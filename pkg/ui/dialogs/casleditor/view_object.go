package casleditor

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type ObjectView struct {
	vm *EditorViewModel

	Container *fyne.Container

	nameEntry         *widget.Entry
	addressEntry      *widget.Entry
	latEntry          *widget.Entry
	longEntry         *widget.Entry
	descriptionEntry  *widget.Entry
	contractEntry     *widget.Entry
	managerSelect     *widget.Select
	noteEntry         *widget.Entry
	startDateEntry    *widget.DateEntry
	statusEntry       *widget.Entry
	typeEntry         *widget.Entry
	requestIDEntry    *widget.Entry
	pultSelect        *widget.Select
	geoZoneSelect     *widget.Select
	businessEntry     *widget.Entry
	nameStatus        *canvas.Text
	addressStatus     *canvas.Text
	descriptionStatus *canvas.Text
	imagesBox         *fyne.Container
	roomTabsHost      *fyne.Container
	saveBtn           *widget.Button
}

var (
	objectNameValidator fyne.StringValidator = func(value string) error {
		if len([]rune(strings.TrimSpace(value))) < 2 {
			return fmt.Errorf("вкажіть назву об'єкта")
		}
		return nil
	}
	objectAddressValidator fyne.StringValidator = func(value string) error {
		if len([]rune(strings.TrimSpace(value))) < 5 {
			return fmt.Errorf("вкажіть адресу об'єкта")
		}
		return nil
	}
	objectDescriptionValidator fyne.StringValidator = func(value string) error {
		if len([]rune(strings.TrimSpace(value))) < 3 {
			return fmt.Errorf("вкажіть опис об'єкта")
		}
		return nil
	}
	roomNameValidator fyne.StringValidator = func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("обов'язково")
		}
		return nil
	}
	roomDescriptionValidator fyne.StringValidator = func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("обов'язково")
		}
		return nil
	}
)

func NewObjectView(vm *EditorViewModel) *ObjectView {
	v := &ObjectView{
		vm: vm,

		nameEntry:         widget.NewEntry(),
		addressEntry:      widget.NewEntry(),
		latEntry:          widget.NewEntry(),
		longEntry:         widget.NewEntry(),
		descriptionEntry:  widget.NewMultiLineEntry(),
		contractEntry:     widget.NewEntry(),
		managerSelect:     widget.NewSelect(nil, nil),
		noteEntry:         widget.NewMultiLineEntry(),
		startDateEntry:    widget.NewDateEntry(),
		statusEntry:       widget.NewEntry(),
		typeEntry:         widget.NewEntry(),
		requestIDEntry:    widget.NewEntry(),
		pultSelect:        widget.NewSelect(nil, nil),
		geoZoneSelect:     widget.NewSelect(nil, nil),
		businessEntry:     widget.NewEntry(),
		nameStatus:        newValidationStatusText(),
		addressStatus:     newValidationStatusText(),
		descriptionStatus: newValidationStatusText(),
		imagesBox:         container.NewGridWrap(fyne.NewSize(140, 118)),
		roomTabsHost:      container.NewMax(),
	}

	v.descriptionEntry.SetMinRowsVisible(4)
	v.noteEntry.SetMinRowsVisible(3)
	v.startDateEntry.SetPlaceHolder("Оберіть дату старту")
	v.pultSelect.OnChanged = func(selected string) {
		v.vm.Snapshot.Object.ReactingPultID = v.vm.PultOptionToID[selected]
	}

	v.saveBtn = widget.NewButton("Зберегти об'єкт", v.handleSubmit)

	v.setupLayout()
	v.bind()

	return v
}

func (v *ObjectView) setupLayout() {
	pickCoordsBtn := widget.NewButton("Відкрити карту", v.vm.PickObjectCoordinatesOnMap)
	pickCoordsBtn.Importance = widget.LowImportance

	photos := newWizardPanel("Фото об'єкта", container.NewVScroll(v.imagesBox))
	photosWrapper := container.NewBorder(nil, nil, nil, nil, photos)

	objectFields := container.NewVBox(
		newWizardFieldWithStatus("Назва", v.nameEntry, v.nameStatus),
		newWizardFieldWithStatus("Адреса", v.addressEntry, v.addressStatus),
		container.NewGridWithColumns(3,
			newWizardField("Широта", v.latEntry),
			newWizardField("Довгота", v.longEntry),
			newWizardField("Карта", pickCoordsBtn),
		),
		newWizardFieldWithStatus("Опис", v.descriptionEntry, v.descriptionStatus),
		newWizardField("Договір", v.contractEntry),
		newWizardField("Менеджер", v.managerSelect),
		newWizardField("Примітка", v.noteEntry),
		newWizardField("Дата запуску", v.startDateEntry),
		container.NewGridWithColumns(2,
			newWizardField("Статус об'єкта", v.statusEntry),
			newWizardField("Тип об'єкта", v.typeEntry),
		),
		container.NewGridWithColumns(2,
			newWizardField("ID заявки", v.requestIDEntry),
			newWizardField("Бізнес-коефіцієнт", v.businessEntry),
		),
		container.NewGridWithColumns(2,
			newWizardField("Пульт реагування", v.pultSelect),
			newWizardField("Зона реагування", v.geoZoneSelect),
		),
	)

	centerCard := container.NewPadded(newWizardPanel("Створення географічного об'єкта", container.NewVScroll(objectFields)))
	rightCard := container.NewPadded(v.buildRoomsPanel())

	leftCenter := container.NewHSplit(
		container.NewPadded(photosWrapper),
		centerCard,
	)
	leftCenter.SetOffset(0.2)

	mainContent := container.NewHSplit(leftCenter, rightCard)
	mainContent.SetOffset(0.7)

	v.Container = container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), v.saveBtn),
		nil,
		nil,
		FixedMinHeightArea(700, mainContent),
	)
}

func (v *ObjectView) bind() {
	v.vm.AddDataChangedListener(func() {
		obj := v.vm.Snapshot.Object
		v.nameEntry.SetText(obj.Name)
		v.addressEntry.SetText(obj.Address)
		v.latEntry.SetText(obj.Lat)
		v.longEntry.SetText(obj.Long)
		v.descriptionEntry.SetText(obj.Description)
		v.contractEntry.SetText(obj.Contract)
		v.noteEntry.SetText(obj.Note)
		v.statusEntry.SetText(obj.ObjectStatus)
		v.typeEntry.SetText(obj.ObjectType)
		v.requestIDEntry.SetText(obj.IDRequest)
		v.businessEntry.SetText(float64PtrToString(obj.BusinessCoeff))

		SetCASLEditorDateEntry(v.startDateEntry, CaslDatePtr(obj.StartDate))

		v.managerSelect.Options = v.vm.ManagerOptions
		v.managerSelect.SetSelected(v.vm.UserOptionByID(obj.ManagerID, v.vm.ManagerOptionToID))

		v.pultSelect.Options = v.vm.PultOptions
		selectedPult := optionLabelByValue(obj.ReactingPultID, v.vm.PultOptionToID)
		if selectedPult == "" && len(v.vm.PultOptions) > 0 {
			selectedPult = v.vm.PultOptions[0]
			v.vm.Snapshot.Object.ReactingPultID = v.vm.PultOptionToID[selectedPult]
		}
		v.pultSelect.SetSelected(selectedPult)

		v.geoZoneSelect.Options = v.vm.GeoZoneOptions
		selectedGeo := optionLabelByInt64(obj.GeoZoneID, v.vm.GeoZoneOptionToID)
		if selectedGeo == "" && len(v.vm.GeoZoneOptions) > 0 && obj.GeoZoneID <= 0 {
			selectedGeo = v.vm.GeoZoneOptions[0]
			v.vm.Snapshot.Object.GeoZoneID = v.vm.GeoZoneOptionToID[selectedGeo]
		}
		v.geoZoneSelect.SetSelected(selectedGeo)

		if v.vm.HasObject() {
			v.saveBtn.SetText("Зберегти об'єкт")
		} else {
			v.saveBtn.SetText("Зберегти чернетку")
		}

		v.refreshImages()
		v.refreshRoomTabs()
		v.refreshValidation()
	})

	v.nameEntry.OnChanged = func(string) { v.refreshValidation() }
	v.addressEntry.OnChanged = func(string) { v.refreshValidation() }
	v.descriptionEntry.OnChanged = func(string) { v.refreshValidation() }
	v.pultSelect.OnChanged = func(selected string) {
		v.vm.Snapshot.Object.ReactingPultID = v.vm.PultOptionToID[selected]
		v.refreshValidation()
	}
	v.geoZoneSelect.OnChanged = func(selected string) {
		v.vm.Snapshot.Object.GeoZoneID = v.vm.GeoZoneOptionToID[selected]
	}
}

func (v *ObjectView) handleSubmit() {
	if err := v.CommitDraft(); err != nil {
		v.vm.showError(err)
		return
	}
	if !v.vm.HasObject() {
		return
	}
	v.vm.SubmitObject(v.collectData())
}

func (v *ObjectView) collectData() ObjectUpdateData {
	biz, _ := ParseCASLEditorFloatPtr(v.businessEntry.Text)
	start, _ := DateEntryUnixMilli(v.startDateEntry)

	return ObjectUpdateData{
		Name:           v.nameEntry.Text,
		Address:        v.addressEntry.Text,
		Lat:            v.latEntry.Text,
		Long:           v.longEntry.Text,
		Description:    v.descriptionEntry.Text,
		Contract:       v.contractEntry.Text,
		ManagerID:      v.vm.ManagerOptionToID[v.managerSelect.Selected],
		Note:           v.noteEntry.Text,
		StartDate:      start,
		Status:         v.statusEntry.Text,
		ObjectType:     v.typeEntry.Text,
		IDRequest:      v.requestIDEntry.Text,
		ReactingPultID: v.vm.PultOptionToID[v.pultSelect.Selected],
		GeoZoneID:      v.vm.GeoZoneOptionToID[v.geoZoneSelect.Selected],
		BusinessCoeff:  biz,
	}
}

func (v *ObjectView) Title() string { return "Крок 1. Дані об'єкта" }

func (v *ObjectView) ProgressLabel() string {
	return "Створення географічного об'єкта"
}

func (v *ObjectView) Content() fyne.CanvasObject { return v.Container }

func (v *ObjectView) CommitDraft() error {
	return v.vm.DraftObject(v.collectData())
}

func (v *ObjectView) refreshValidation() {
	data := ObjectUpdateData{
		Name:        v.nameEntry.Text,
		Address:     v.addressEntry.Text,
		Description: v.descriptionEntry.Text,
	}
	setValidationStatus(v.nameStatus, objectNameValidator(v.nameEntry.Text))
	setValidationStatus(v.addressStatus, objectAddressValidator(v.addressEntry.Text))
	setValidationStatus(v.descriptionStatus, objectDescriptionValidator(v.descriptionEntry.Text))
	if err := v.vm.ValidateObjectForm(data); err != nil {
		v.saveBtn.Disable()
	} else {
		v.saveBtn.Enable()
	}
}

func (v *ObjectView) refreshImages() {
	SetCASLImageSlots(v.imagesBox, v.vm.ObjectScopedImages(), "Об'єкт", v.vm.Provider(), v.vm.DeleteObjectImage, v.vm.ShowImagePreview, v.vm.UploadObjectImage)
}

func (v *ObjectView) buildRoomsPanel() fyne.CanvasObject {
	return newWizardPanel("Приміщення", v.roomTabsHost)
}

func (v *ObjectView) refreshRoomTabs() {
	rooms := v.vm.Snapshot.Object.Rooms
	tabButtons := make([]fyne.CanvasObject, 0, len(rooms)+1)
	for idx, room := range rooms {
		index := idx
		active := index == v.vm.RoomSelected
		title := roomTabTitle(room)
		tabBtn := newCompactWizardTab(title, active, func() {
			v.vm.SelectRoom(index)
			v.refreshRoomTabs()
		})
		var deleteAction fyne.CanvasObject = canvas.NewText("", color.Transparent)
		if !v.vm.HasObject() {
			deleteAction = newWizardHeaderAction(theme.CancelIcon(), wizardDanger, func() {
				v.removeDraftRoom(index)
			})
		}
		tabButtons = append(tabButtons, container.NewHBox(
			tabBtn,
			deleteAction,
		))
	}
	tabButtons = append(tabButtons, newWizardHeaderAction(theme.ContentAddIcon(), wizardAccent, func() {
		v.addDraftRoom()
	}))

	var content fyne.CanvasObject
	if len(rooms) == 0 {
		content = widget.NewLabel("Додайте хоча б одне приміщення. Назва і опис обов'язкові.")
	} else {
		if v.vm.RoomSelected < 0 || v.vm.RoomSelected >= len(rooms) {
			v.vm.RoomSelected = 0
		}
		content = v.buildSelectedRoomEditor(v.vm.RoomSelected, rooms[v.vm.RoomSelected])
	}
	v.roomTabsHost.Objects = []fyne.CanvasObject{
		container.NewVBox(
			container.New(newWizardWrapLayout(8, 8), tabButtons...),
			content,
		),
	}
	v.roomTabsHost.Refresh()
}

func (v *ObjectView) buildSelectedRoomEditor(index int, room contracts.CASLRoomDetails) fyne.CanvasObject {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(room.Name)
	nameEntry.SetPlaceHolder("Назва приміщення")
	nameEntry.SetMinRowsVisible(1)
	nameEntry.Resize(fyne.NewSize(180, nameEntry.MinSize().Height))

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetText(room.Description)
	descEntry.SetMinRowsVisible(3)

	rtspEntry := widget.NewEntry()
	rtspEntry.SetText(room.RTSP)

	nameError := newValidationStatusText()
	descError := newValidationStatusText()
	syncValidation := func() {
		nameMsg, descMsg := roomFieldValidationMessages(RoomUpdateData{
			Name:        nameEntry.Text,
			Description: descEntry.Text,
			RTSP:        rtspEntry.Text,
		})
		nameError.Text = nameMsg
		descError.Text = descMsg
		nameError.Refresh()
		descError.Refresh()
	}
	updateDraft := func() {
		v.vm.Snapshot.Object.Rooms[index].Name = strings.TrimSpace(nameEntry.Text)
		v.vm.Snapshot.Object.Rooms[index].Description = strings.TrimSpace(descEntry.Text)
		v.vm.Snapshot.Object.Rooms[index].RTSP = strings.TrimSpace(rtspEntry.Text)
		v.vm.initDictionaries()
		syncValidation()
		v.refreshRoomTabs()
	}
	nameEntry.OnChanged = func(string) { updateDraft() }
	descEntry.OnChanged = func(string) { updateDraft() }
	rtspEntry.OnChanged = func(string) { updateDraft() }
	syncValidation()

	photosBox := container.NewGridWrap(fyne.NewSize(96, 96))
	SetCASLImageSlots(photosBox, room.Images, "Приміщення", v.vm.Provider(), func(imageID string) {
		v.vm.SelectRoom(index)
		v.vm.DeleteRoomImage(imageID)
	}, v.vm.ShowImagePreview, func() {
		v.vm.SelectRoom(index)
		v.vm.UploadRoomImage()
	})

	return container.NewVBox(
		container.NewGridWithColumns(
			2,
			newWizardFieldWithStatus("Назва", fixedWidthField(nameEntry, 180), nameError),
			newWizardField("Посилання на камеру (RTSP)", rtspEntry),
		),
		newWizardFieldWithStatus("Опис", descEntry, descError),
		newWizardPanel("Фото приміщення", photosBox),
	)
}

func (v *ObjectView) addDraftRoom() {
	if v.vm.HasObject() {
		return
	}
	v.vm.Snapshot.Object.Rooms = append(v.vm.Snapshot.Object.Rooms, contracts.CASLRoomDetails{
		RoomID:      v.vm.nextDraftRoomID(),
		Name:        "",
		Description: "",
		RTSP:        "",
		Users:       nil,
		Lines:       nil,
	})
	v.vm.RoomSelected = len(v.vm.Snapshot.Object.Rooms) - 1
	v.vm.initDictionaries()
	v.vm.emitDataChanged()
}

func (v *ObjectView) removeDraftRoom(index int) {
	if v.vm.HasObject() || index < 0 || index >= len(v.vm.Snapshot.Object.Rooms) {
		return
	}
	roomID := v.vm.Snapshot.Object.Rooms[index].RoomID
	for lineIndex := range v.vm.Snapshot.Object.Device.Lines {
		if v.vm.Snapshot.Object.Device.Lines[lineIndex].RoomID == roomID {
			v.vm.Snapshot.Object.Device.Lines[lineIndex].RoomID = ""
		}
	}
	v.vm.Snapshot.Object.Rooms = append(v.vm.Snapshot.Object.Rooms[:index], v.vm.Snapshot.Object.Rooms[index+1:]...)
	if v.vm.RoomSelected >= len(v.vm.Snapshot.Object.Rooms) {
		v.vm.RoomSelected = len(v.vm.Snapshot.Object.Rooms) - 1
	}
	v.vm.initDictionaries()
	v.vm.emitDataChanged()
}

func roomTabTitle(room contracts.CASLRoomDetails) string {
	name := strings.TrimSpace(room.Name)
	if name == "" {
		return "Нове приміщення"
	}
	runes := []rune(name)
	if len(runes) <= 20 {
		return name
	}
	return string(runes[:20])
}

func roomFieldValidationMessages(data RoomUpdateData) (string, string) {
	nameMsg := validationMessageFromError(roomNameValidator(data.Name))
	descMsg := validationMessageFromError(roomDescriptionValidator(data.Description))
	return nameMsg, descMsg
}

func fixedWidthField(input fyne.CanvasObject, width float32) fyne.CanvasObject {
	spacer := widget.NewLabel("")
	spacer.Resize(fyne.NewSize(width, 1))
	bg := container.NewWithoutLayout(spacer)
	return container.NewStack(bg, input)
}
