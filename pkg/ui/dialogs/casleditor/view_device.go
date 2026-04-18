package casleditor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type DeviceView struct {
	vm *EditorViewModel

	Container *fyne.Container

	numberEntry        *widget.Entry
	nameEntry          *widget.Entry
	typeSelect         *widget.Select
	timeoutEntry       *widget.Entry
	sim1Entry          *widget.Entry
	sim2Entry          *widget.Entry
	technicianSelect   *widget.Select
	unitsEntry         *widget.Entry
	requisitesEntry    *widget.Entry
	changeDateEntry    *widget.DateEntry
	reglamentDateEntry *widget.DateEntry
	licenceEntry       *widget.Entry
	remotePassEntry    *widget.Entry
	typeStatus         *canvas.Text
	numberStatus       *canvas.Text
	nameStatus         *canvas.Text
	timeoutStatus      *canvas.Text
	sim1Status         *canvas.Text
	sim2Status         *canvas.Text
	saveBtn            *widget.Button
	blockBtn           *widget.Button
	linesPreviewList   *widget.List
	syncing            bool
	numberTaken        bool
	numberCheckLoading bool
	numberCheckMessage string
	numberCheckSeq     uint64
	lastAutoTimeout    int64
}

var (
	deviceNumberValidator fyne.StringValidator = func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("вкажіть № ППК")
		}
		if _, err := ParseCASLEditorInt64(value); err != nil {
			return fmt.Errorf("№ ППК має містити лише цифри")
		}
		return nil
	}
	deviceNameValidator fyne.StringValidator = func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("вкажіть назву приладу")
		}
		return nil
	}
	deviceTypeValidator fyne.StringValidator = func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("вкажіть тип приладу")
		}
		return nil
	}
	deviceTimeoutValidator fyne.StringValidator = func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("вкажіть timeout")
		}
		timeout, err := ParseCASLEditorInt64(value)
		if err != nil {
			return fmt.Errorf("timeout має містити лише цифри")
		}
		if timeout <= 0 {
			return fmt.Errorf("timeout має бути більше 0")
		}
		return nil
	}
)

func NewDeviceView(vm *EditorViewModel) *DeviceView {
	v := &DeviceView{
		vm: vm,

		numberEntry:        widget.NewEntry(),
		nameEntry:          widget.NewEntry(),
		typeSelect:         widget.NewSelect(nil, nil),
		timeoutEntry:       widget.NewEntry(),
		sim1Entry:          widget.NewEntry(),
		sim2Entry:          widget.NewEntry(),
		technicianSelect:   widget.NewSelect(nil, nil),
		unitsEntry:         widget.NewEntry(),
		requisitesEntry:    widget.NewEntry(),
		changeDateEntry:    widget.NewDateEntry(),
		reglamentDateEntry: widget.NewDateEntry(),
		licenceEntry:       widget.NewEntry(),
		remotePassEntry:    widget.NewEntry(),
		typeStatus:         newValidationStatusText(),
		numberStatus:       newValidationStatusText(),
		nameStatus:         newValidationStatusText(),
		timeoutStatus:      newValidationStatusText(),
		sim1Status:         newValidationStatusText(),
		sim2Status:         newValidationStatusText(),
	}

	v.sim1Entry.SetPlaceHolder("+38 (050) 123-45-67")
	v.sim2Entry.SetPlaceHolder("+38 (050) 123-45-67")
	v.licenceEntry.SetPlaceHolder("123-123-123-123-123-123")

	v.setupLinesPreview()
	v.setupLayout()
	v.bind()

	return v
}

func (v *DeviceView) setupLinesPreview() {
	v.linesPreviewList = widget.NewList(
		func() int { return len(v.vm.Snapshot.Object.Device.Lines) },
		func() fyne.CanvasObject { return newCASLListRowTemplate() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(v.vm.Snapshot.Object.Device.Lines) {
				setCASLListRow(obj, "", "")
				return
			}
			line := v.vm.Snapshot.Object.Device.Lines[id]
			title := "#" + strconv.Itoa(line.LineNumber) + " " + FirstNonEmpty(line.Description, "Немає назви")
			subtitle := "Адаптер: " + FirstNonEmpty(v.vm.DisplayAdapterType(line.AdapterType), line.AdapterType) + "/" + strconv.Itoa(line.AdapterNumber) +
				" | Група: " + strconv.Itoa(line.GroupNumber) +
				" | Тип: " + FirstNonEmpty(v.vm.DisplayLineType(line.LineType), line.LineType)
			setCASLListRow(obj, title, subtitle)
		},
	)
}

func (v *DeviceView) setupLayout() {
	v.saveBtn = widget.NewButton("Зберегти обладнання", v.handleSubmit)
	v.blockBtn = widget.NewButton("Заблокувати прилад", v.handleToggleBlock)
	showLinesCheck := widget.NewCheck("Показати шлейфи", nil)

	identityCard := newWizardPanel("Ідентифікація", container.NewVBox(
		newWizardFieldWithStatus("Тип приладу", v.typeSelect, v.typeStatus),
		newWizardFieldWithStatus("№ ППК", v.numberEntry, v.numberStatus),
		newWizardFieldWithStatus("Назва / примітка", v.nameEntry, v.nameStatus),
		newWizardField("Технік", v.technicianSelect),
	))

	connectivityCard := newWizardPanel("Зв'язок", container.NewVBox(
		newWizardFieldWithStatus("Таймаут, с", v.timeoutEntry, v.timeoutStatus),
		container.NewGridWithColumns(
			2,
			newWizardFieldWithStatus("SIM 1", v.sim1Entry, v.sim1Status),
			newWizardFieldWithStatus("SIM 2", v.sim2Entry, v.sim2Status),
		),
		container.NewGridWithColumns(
			2,
			newWizardField("Одиниці виміру", v.unitsEntry),
			newWizardField("Реквізити", v.requisitesEntry),
		),
	))

	serviceCard := newWizardPanel("Сервіс", container.NewVBox(
		container.NewGridWithColumns(
			2,
			newWizardField("Дата заміни", v.changeDateEntry),
			newWizardField("Дата регламенту", v.reglamentDateEntry),
		),
		newWizardField("Ліцензійний ключ", v.licenceEntry),
		newWizardField("Пароль віддаленого доступу", v.remotePassEntry),
	))

	leftColumn := container.NewVBox(
		identityCard,
		connectivityCard,
		serviceCard,
		showLinesCheck,
	)

	rightColumn := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Шлейфи приладу"),
			widget.NewLabel("Список зон приладу без таблиці."),
		),
		nil,
		nil,
		nil,
		v.linesPreviewList,
	)

	leftCard := newWizardPanel("Створення об'єктового обладнання", container.NewScroll(leftColumn))
	rightCard := newWizardPanel("Шлейфи", rightColumn)
	main := container.NewHSplit(leftCard, rightCard)
	main.SetOffset(1)
	rightCard.Hide()

	showLinesCheck.OnChanged = func(checked bool) {
		if checked {
			rightCard.Show()
			main.SetOffset(0.42)
			return
		}
		rightCard.Hide()
		main.SetOffset(1)
	}

	v.Container = container.NewBorder(
		nil,
		container.NewHBox(v.blockBtn, layout.NewSpacer(), v.saveBtn),
		nil,
		nil,
		main,
	)
}

func (v *DeviceView) bind() {
	v.vm.AddDataChangedListener(func() {
		dev := v.vm.Snapshot.Object.Device
		v.syncing = true
		v.numberEntry.SetText(int64ToString(dev.Number))
		v.nameEntry.SetText(dev.Name)
		v.timeoutEntry.SetText(int64ToString(dev.Timeout))
		v.sim1Entry.SetText(FormatCASLEditorSIMForDisplay(dev.SIM1))
		v.sim2Entry.SetText(FormatCASLEditorSIMForDisplay(dev.SIM2))
		v.unitsEntry.SetText(dev.Units)
		v.requisitesEntry.SetText(dev.Requisites)
		v.licenceEntry.SetText(FormatCASLEditorLicenceForDisplay(dev.LicenceKey))
		v.remotePassEntry.SetText(dev.PasswRemote)

		SetCASLEditorDateEntry(v.changeDateEntry, CaslDatePtr(dev.ChangeDate))
		SetCASLEditorDateEntry(v.reglamentDateEntry, CaslDatePtr(dev.ReglamentDate))

		v.typeSelect.Options = v.vm.DeviceTypeOptions
		v.typeSelect.SetSelected(optionLabelByValue(dev.Type, v.vm.DeviceTypeOptionToID))

		v.technicianSelect.Options = v.vm.TechOptions
		v.technicianSelect.SetSelected(v.vm.TechOptionByID(dev.TechnicianID))

		if v.vm.HasDevice() {
			v.numberEntry.Disable()
			v.saveBtn.SetText("Зберегти обладнання")
			if v.vm.Snapshot.Object.DeviceBlocked {
				v.blockBtn.SetText("Розблокувати прилад")
			} else {
				v.blockBtn.SetText("Заблокувати прилад")
			}
		} else {
			v.numberEntry.Enable()
			v.saveBtn.SetText("Зберегти чернетку обладнання")
		}
		v.syncing = false

		v.linesPreviewList.Refresh()
		v.scheduleDeviceNumberCheck(v.numberEntry.Text)
		v.refreshValidation()
	})

	v.attachPhoneFormatter(v.sim1Entry)
	v.attachPhoneFormatter(v.sim2Entry)

	v.numberEntry.OnChanged = func(value string) {
		if v.syncing {
			return
		}
		digits := DigitsOnly(value)
		if digits != value {
			v.syncing = true
			v.numberEntry.SetText(digits)
			v.syncing = false
			return
		}
		v.syncDraftSnapshot()
		v.scheduleDeviceNumberCheck(value)
		v.refreshValidation()
	}
	v.nameEntry.OnChanged = func(string) {
		if v.syncing {
			return
		}
		v.syncDraftSnapshot()
		v.refreshValidation()
	}
	v.timeoutEntry.OnChanged = func(string) {
		if v.syncing {
			return
		}
		v.syncDraftSnapshot()
		v.refreshValidation()
	}
	v.typeSelect.OnChanged = func(selected string) {
		if v.syncing {
			return
		}
		v.applyDefaultTimeoutForType(selected)
		v.syncDraftSnapshot()
		v.refreshValidation()
	}
	v.technicianSelect.OnChanged = func(string) { v.syncDraftSnapshot() }
	v.unitsEntry.OnChanged = func(string) { v.syncDraftSnapshot() }
	v.requisitesEntry.OnChanged = func(string) { v.syncDraftSnapshot() }
	v.changeDateEntry.OnChanged = func(*time.Time) { v.syncDraftSnapshot() }
	v.reglamentDateEntry.OnChanged = func(*time.Time) { v.syncDraftSnapshot() }
	v.licenceEntry.OnChanged = func(string) { v.syncDraftSnapshot() }
	v.remotePassEntry.OnChanged = func(string) { v.syncDraftSnapshot() }
}

func (v *DeviceView) handleSubmit() {
	if err := v.CommitDraft(); err != nil {
		v.vm.showError(err)
		return
	}
	if !v.vm.HasDevice() {
		return
	}
	v.vm.SubmitDevice(v.collectData())
}

func (v *DeviceView) collectData() DeviceUpdateData {
	num, _ := ParseCASLEditorInt64(v.numberEntry.Text)
	timeout, _ := ParseCASLEditorInt64(v.timeoutEntry.Text)
	sim1, _ := NormalizeCASLEditorSIM(v.sim1Entry.Text)
	sim2, _ := NormalizeCASLEditorSIM(v.sim2Entry.Text)
	licence, _ := NormalizeCASLEditorLicenceForSave(v.licenceEntry.Text)
	change, _ := DateEntryUnixMilli(v.changeDateEntry)
	reg, _ := DateEntryUnixMilli(v.reglamentDateEntry)

	return DeviceUpdateData{
		Number:        num,
		Name:          v.nameEntry.Text,
		Type:          v.vm.DeviceTypeOptionToID[v.typeSelect.Selected],
		Timeout:       timeout,
		SIM1:          sim1,
		SIM2:          sim2,
		TechnicianID:  v.vm.TechOptionToID[v.technicianSelect.Selected],
		Units:         v.unitsEntry.Text,
		Requisites:    v.requisitesEntry.Text,
		ChangeDate:    change,
		ReglamentDate: reg,
		LicenceKey:    licence,
		RemotePass:    v.remotePassEntry.Text,
	}
}

func (v *DeviceView) CommitDraft() error {
	return v.vm.DraftDevice(v.collectData())
}

func (v *DeviceView) handleToggleBlock() {
	v.vm.ToggleDeviceBlock("Блокування оператором", 24)
}

func (v *DeviceView) refreshValidation() {
	typeErr := deviceTypeValidator(v.typeSelect.Selected)
	numberErr := deviceNumberValidator(v.numberEntry.Text)
	nameErr := deviceNameValidator(v.nameEntry.Text)
	timeoutErr := deviceTimeoutValidator(v.timeoutEntry.Text)
	sim1Err := optionalSIMValidator("SIM 1")(v.sim1Entry.Text)
	sim2Err := optionalSIMValidator("SIM 2")(v.sim2Entry.Text)

	setValidationStatus(v.typeStatus, typeErr)
	setValidationStatus(v.numberStatus, numberErr)
	setValidationStatus(v.nameStatus, nameErr)
	setValidationStatus(v.timeoutStatus, timeoutErr)
	setValidationStatus(v.sim1Status, sim1Err)
	setValidationStatus(v.sim2Status, sim2Err)

	if numberErr == nil && !v.vm.HasDevice() {
		switch {
		case v.numberTaken:
			setValidationMessage(v.numberStatus, "* цей № ППК вже зайнятий", wizardDanger)
		case v.numberCheckLoading:
			setValidationMessage(v.numberStatus, "перевірка...", wizardMutedText)
		case strings.TrimSpace(v.numberCheckMessage) == "№ ППК вільний":
			setValidationMessage(v.numberStatus, "", wizardDanger)
		case strings.TrimSpace(v.numberCheckMessage) != "":
			setValidationMessage(v.numberStatus, v.numberCheckMessage, wizardMutedText)
		}
	}

	if typeErr != nil || numberErr != nil || nameErr != nil || timeoutErr != nil || sim1Err != nil || sim2Err != nil || v.numberTaken || (v.numberCheckLoading && !v.vm.HasDevice()) {
		v.saveBtn.Disable()
		return
	}
	v.saveBtn.Enable()
}

func (v *DeviceView) attachPhoneFormatter(entry *widget.Entry) {
	BindDebouncedPhoneFormatter(entry, 250*time.Millisecond, func() {
		if v.syncing {
			return
		}
		v.syncDraftSnapshot()
		v.refreshValidation()
	})
}

func (v *DeviceView) applyDefaultTimeoutForType(selected string) {
	rawType := strings.TrimSpace(v.vm.DeviceTypeOptionToID[selected])
	if rawType == "" {
		rawType = strings.TrimSpace(selected)
	}
	nextDefault := defaultCASLDeviceTimeout(rawType)
	if nextDefault <= 0 {
		return
	}
	current, _ := ParseCASLEditorInt64(v.timeoutEntry.Text)
	if current == 0 || current == v.lastAutoTimeout || current == 240 || current == 1000 {
		v.syncing = true
		v.timeoutEntry.SetText(strconv.FormatInt(nextDefault, 10))
		v.syncing = false
		v.lastAutoTimeout = nextDefault
	}
}

func (v *DeviceView) syncDraftSnapshot() {
	if v.syncing || v.vm.HasDevice() {
		return
	}
	data := v.collectData()
	v.vm.Snapshot.Object.Device.Number = data.Number
	v.vm.Snapshot.Object.Device.Name = strings.TrimSpace(data.Name)
	v.vm.Snapshot.Object.Device.Type = strings.TrimSpace(data.Type)
	v.vm.Snapshot.Object.Device.Timeout = data.Timeout
	v.vm.Snapshot.Object.Device.SIM1 = strings.TrimSpace(data.SIM1)
	v.vm.Snapshot.Object.Device.SIM2 = strings.TrimSpace(data.SIM2)
	v.vm.Snapshot.Object.Device.TechnicianID = strings.TrimSpace(data.TechnicianID)
	v.vm.Snapshot.Object.Device.Units = strings.TrimSpace(data.Units)
	v.vm.Snapshot.Object.Device.Requisites = strings.TrimSpace(data.Requisites)
	v.vm.Snapshot.Object.Device.ChangeDate = data.ChangeDate
	v.vm.Snapshot.Object.Device.ReglamentDate = data.ReglamentDate
	v.vm.Snapshot.Object.Device.LicenceKey = strings.TrimSpace(data.LicenceKey)
	v.vm.Snapshot.Object.Device.PasswRemote = strings.TrimSpace(data.RemotePass)
}

func (v *DeviceView) scheduleDeviceNumberCheck(raw string) {
	v.numberTaken = false
	v.numberCheckMessage = ""
	if v.vm.HasDevice() {
		v.numberCheckLoading = false
		return
	}

	deviceNumber, err := ParseCASLEditorInt64(raw)
	if err != nil || deviceNumber <= 0 {
		v.numberCheckLoading = false
		return
	}

	seq := atomic.AddUint64(&v.numberCheckSeq, 1)
	v.numberCheckLoading = true

	go func(number int64, checkSeq uint64) {
		time.Sleep(250 * time.Millisecond)
		if atomic.LoadUint64(&v.numberCheckSeq) != checkSeq {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		inUse, checkErr := v.vm.Provider().IsCASLDeviceNumberInUse(ctx, number)
		fyne.Do(func() {
			if atomic.LoadUint64(&v.numberCheckSeq) != checkSeq {
				return
			}
			v.numberCheckLoading = false
			if checkErr != nil {
				v.numberCheckMessage = "Не вдалося перевірити № ППК"
				v.numberTaken = false
				v.refreshValidation()
				return
			}
			v.numberTaken = inUse
			if inUse {
				v.numberCheckMessage = "Цей № ППК вже зайнятий"
			} else {
				v.numberCheckMessage = "№ ППК вільний"
			}
			v.refreshValidation()
		})
	}(deviceNumber, seq)
}

func optionalSIMValidator(label string) fyne.StringValidator {
	return func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		if _, err := NormalizeCASLEditorSIM(value); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		return nil
	}
}
