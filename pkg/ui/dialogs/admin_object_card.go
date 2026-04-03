package dialogs

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/simoperator"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

func ShowNewObjectDialog(parent fyne.Window, provider contracts.AdminObjectWizardProvider, onSaved func(objn int64)) {
	ShowNewObjectWizardDialog(parent, provider, onSaved)
}

func ShowEditObjectDialog(parent fyne.Window, provider contracts.AdminObjectCardProvider, objn int64, onSaved func(objn int64)) {
	showObjectCardDialog(parent, provider, &objn, onSaved)
}

type objectCardDialogState struct {
	win       fyne.Window
	provider  contracts.AdminObjectCardProvider
	onSaved   func(objn int64)
	editObjN  *int64
	isEdit    bool
	vm        *viewmodels.ObjectCardViewModel
	formVM    *viewmodels.ObjectCardFormViewModel
	dateVM    *viewmodels.ObjectDateFieldViewModel
	initVM    *viewmodels.ObjectCardInitViewModel
	dialogVM  *viewmodels.ObjectCardDialogViewModel
	submitVM  *viewmodels.ObjectCardSubmitViewModel
	defVM     *viewmodels.ObjectCardDefaultsViewModel
	loadVM    *viewmodels.ObjectCardLoadViewModel
	channelVM *viewmodels.ObjectChannelFlowViewModel
	refsVM    *viewmodels.ObjectCardReferencesViewModel
	simVM     *viewmodels.SIMPhoneUsageViewModel
	simState  *viewmodels.ObjectCardSIMUsageStateViewModel
	vfSIMVM   *viewmodels.VodafoneSIMViewModel
	ksSIMVM   *viewmodels.KyivstarSIMViewModel
	simAPI1VM *viewmodels.SIMOperatorStateViewModel
	simAPI2VM *viewmodels.SIMOperatorStateViewModel

	statusLabel *widget.Label

	shortNameBinding binding.String
	fullNameBinding  binding.String

	objnEntry      *widget.Entry
	shortNameEntry *widget.Entry
	fullNameEntry  *widget.Entry
	addressEntry   *widget.Entry
	phonesEntry    *widget.Entry
	contractEntry  *widget.Entry
	dateEntry      *widget.Entry
	locationEntry  *widget.Entry
	notesEntry     *widget.Entry

	channelCodeSelect *widget.Select
	sim1Entry         *widget.Entry
	sim2Entry         *widget.Entry
	sim1UsageLabel    *widget.Label
	sim2UsageLabel    *widget.Label
	sim1OperatorLabel *widget.Label
	sim2OperatorLabel *widget.Label
	hiddenNEntry      *widget.Entry
	hiddenNCard       *widget.Card

	testControlCheck  *widget.Check
	testIntervalEntry *widget.Entry

	objectTypeSelect *widget.Select
	regionSelect     *widget.Select
	ppkSelect        *widget.Select
	subServerASelect *widget.Select
	subServerBSelect *widget.Select

	channelLabelToCode map[string]int64
	channelCodeToLabel map[int64]string
}

func newObjectCardDialogState(win fyne.Window, provider contracts.AdminObjectCardProvider, editObjN *int64, onSaved func(objn int64)) *objectCardDialogState {
	simState := viewmodels.NewObjectCardSIMUsageStateViewModel()
	simAPI1VM := viewmodels.NewSIMOperatorStateViewModel("SIM API: перевірка за запитом")
	simAPI2VM := viewmodels.NewSIMOperatorStateViewModel("SIM API: перевірка за запитом")
	channelOptions := viewmodels.ObjectChannelOptions()
	channelLabelToCode := viewmodels.DefaultObjectChannelLabelToCode()
	channelCodeToLabel := viewmodels.DefaultObjectChannelCodeToLabel()

	s := &objectCardDialogState{
		win:       win,
		provider:  provider,
		onSaved:   onSaved,
		editObjN:  editObjN,
		isEdit:    editObjN != nil && *editObjN > 0,
		vm:        viewmodels.NewObjectCardViewModel(),
		formVM:    viewmodels.NewObjectCardFormViewModel(),
		dateVM:    viewmodels.NewObjectDateFieldViewModel(),
		initVM:    viewmodels.NewObjectCardInitViewModel(),
		dialogVM:  viewmodels.NewObjectCardDialogViewModel(),
		submitVM:  nil,
		defVM:     viewmodels.NewObjectCardDefaultsViewModel(),
		loadVM:    viewmodels.NewObjectCardLoadViewModel(),
		channelVM: viewmodels.NewObjectChannelFlowViewModel(),
		refsVM:    viewmodels.NewObjectCardReferencesViewModel(),
		simVM:     viewmodels.NewSIMPhoneUsageViewModel(),
		simState:  simState,
		vfSIMVM:   viewmodels.NewVodafoneSIMViewModel(),
		ksSIMVM:   viewmodels.NewKyivstarSIMViewModel(),
		simAPI1VM: simAPI1VM,
		simAPI2VM: simAPI2VM,

		statusLabel: widget.NewLabel("Готово"),

		shortNameBinding: binding.NewString(),
		fullNameBinding:  binding.NewString(),

		objnEntry:     widget.NewEntry(),
		addressEntry:  widget.NewEntry(),
		phonesEntry:   widget.NewEntry(),
		contractEntry: widget.NewEntry(),
		dateEntry:     widget.NewEntry(),
		locationEntry: widget.NewMultiLineEntry(),
		notesEntry:    widget.NewMultiLineEntry(),

		channelCodeSelect: widget.NewSelect(channelOptions, nil),
		sim1Entry:         widget.NewEntry(),
		sim2Entry:         widget.NewEntry(),
		sim1UsageLabel:    widget.NewLabelWithData(simState.SIM1Binding()),
		sim2UsageLabel:    widget.NewLabelWithData(simState.SIM2Binding()),
		sim1OperatorLabel: widget.NewLabelWithData(simAPI1VM.StatusBinding()),
		sim2OperatorLabel: widget.NewLabelWithData(simAPI2VM.StatusBinding()),
		hiddenNEntry:      widget.NewEntry(),

		testControlCheck:  widget.NewCheck("Контролювати тестові повідомлення", nil),
		testIntervalEntry: widget.NewEntry(),

		objectTypeSelect: widget.NewSelect(nil, nil),
		regionSelect:     widget.NewSelect(nil, nil),
		ppkSelect:        widget.NewSelect(nil, nil),
		subServerASelect: widget.NewSelect(nil, nil),
		subServerBSelect: widget.NewSelect(nil, nil),

		channelLabelToCode: channelLabelToCode,
		channelCodeToLabel: channelCodeToLabel,
	}

	s.shortNameEntry = widget.NewEntryWithData(s.shortNameBinding)
	s.fullNameEntry = widget.NewEntryWithData(s.fullNameBinding)

	s.dateEntry.SetPlaceHolder("дд.мм.рррр")
	s.locationEntry.SetMinRowsVisible(2)
	s.notesEntry.SetMinRowsVisible(4)
	s.sim1UsageLabel.Wrapping = fyne.TextWrapWord
	s.sim2UsageLabel.Wrapping = fyne.TextWrapWord
	s.sim1OperatorLabel.Wrapping = fyne.TextWrapWord
	s.sim2OperatorLabel.Wrapping = fyne.TextWrapWord
	s.hiddenNEntry.SetPlaceHolder("Прихований номер (до 4 цифр)")
	s.testIntervalEntry.SetPlaceHolder("хв.")

	s.hiddenNCard = widget.NewCard(
		"Прихований номер",
		"",
		container.NewVBox(
			widget.NewLabel("Номер (до 4 цифр):"),
			s.hiddenNEntry,
		),
	)

	s.submitVM = viewmodels.NewObjectCardSubmitViewModel(s.dialogVM)
	s.wireEvents()
	return s
}

func (s *objectCardDialogState) wireEvents() {
	s.fullNameBinding.AddListener(binding.NewDataListener(s.onFullNameBindingChanged))
	s.shortNameBinding.AddListener(binding.NewDataListener(s.onShortNameBindingChanged))

	s.testControlCheck.OnChanged = s.enableTestControls

	s.channelCodeSelect.OnChanged = func(_ string) {
		change := s.channelVM.ResolveChange(
			s.channelCodeSelect.Selected,
			s.ppkSelect.Selected,
			s.channelLabelToCode,
			s.refsVM.PPKID,
		)
		s.vm.SetChannelCode(change.ChannelCode)
		s.updateChannelSpecificControls()
		s.refreshPPKOptionsByChannel(change.PreferredPPKID)
	}

	s.sim1Entry.OnChanged = func(text string) {
		s.checkSIMUsage(text, 1)
		s.resetSIMOperatorStatus(1)
	}
	s.sim2Entry.OnChanged = func(text string) {
		s.checkSIMUsage(text, 2)
		s.resetSIMOperatorStatus(2)
	}
}

func (s *objectCardDialogState) onFullNameBindingChanged() {
	fullName, err := s.fullNameBinding.Get()
	if err != nil {
		return
	}
	shortName, err := s.shortNameBinding.Get()
	if err != nil {
		return
	}
	s.vm.OnFullNameChanged(fullName, shortName)
}

func (s *objectCardDialogState) onShortNameBindingChanged() {
	shortName, err := s.shortNameBinding.Get()
	if err != nil {
		return
	}
	fullName, shouldApply := s.vm.OnShortNameChanged(shortName)
	if !shouldApply {
		return
	}
	_ = s.fullNameBinding.Set(fullName)
}

func (s *objectCardDialogState) enableTestControls(enabled bool) {
	if enabled {
		s.testIntervalEntry.Enable()
		return
	}
	s.testIntervalEntry.Disable()
}

func (s *objectCardDialogState) updateChannelSpecificControls() {
	if s.vm.ShouldShowHiddenNumber() {
		s.hiddenNCard.Show()
		s.hiddenNEntry.Enable()
		return
	}
	s.hiddenNCard.Hide()
	s.hiddenNEntry.Disable()
}

func (s *objectCardDialogState) refreshPPKOptionsByChannel(preferredID int64) {
	selected := s.refsVM.RefreshPPKOptions(preferredID)
	s.ppkSelect.Options = s.refsVM.PPKOptions()
	s.ppkSelect.Refresh()
	s.ppkSelect.SetSelected(selected)
}

func (s *objectCardDialogState) checkSIMUsage(rawPhone string, slot int) {
	text := s.simVM.ResolveUsageText(s.provider, rawPhone, s.editObjN)
	if slot == 2 {
		s.simState.SetSIM2(text)
		return
	}
	s.simState.SetSIM1(text)
}

func (s *objectCardDialogState) simOperatorState(slot int) *viewmodels.SIMOperatorStateViewModel {
	if slot == 2 {
		return s.simAPI2VM
	}
	return s.simAPI1VM
}

func (s *objectCardDialogState) currentSIMOperator(slot int) simoperator.Operator {
	return simoperator.Detect(strings.TrimSpace(s.currentSIM(slot)))
}

func (s *objectCardDialogState) resetSIMOperatorStatus(slot int) {
	msisdn := strings.TrimSpace(s.currentSIM(slot))
	if msisdn == "" {
		s.simOperatorState(slot).SetStatus("SIM API: SIM не вказана")
		return
	}
	switch s.currentSIMOperator(slot) {
	case simoperator.Vodafone:
		s.simOperatorState(slot).SetStatus("Vodafone: перевірка за запитом")
	case simoperator.Kyivstar:
		s.simOperatorState(slot).SetStatus("Kyivstar: перевірка за запитом")
	default:
		s.simOperatorState(slot).SetStatus("SIM API: оператор номера не підтримується")
	}
}

func (s *objectCardDialogState) currentSIM(slot int) string {
	if slot == 2 {
		return s.sim2Entry.Text
	}
	return s.sim1Entry.Text
}

func (s *objectCardDialogState) runSIMOperatorAction(slot int, startedText string, action func(operator simoperator.Operator, msisdn string) (string, error)) {
	msisdn := strings.TrimSpace(s.currentSIM(slot))
	if msisdn == "" {
		s.simOperatorState(slot).SetStatus("SIM API: SIM не вказана")
		return
	}
	operator := s.currentSIMOperator(slot)
	if operator == simoperator.Unknown {
		s.simOperatorState(slot).SetStatus("SIM API: оператор номера не підтримується")
		return
	}

	s.simOperatorState(slot).SetStatus(startedText)
	go func() {
		text, err := action(operator, msisdn)
		fyne.Do(func() {
			if err != nil {
				s.simOperatorState(slot).SetStatus(err.Error())
				s.statusLabel.SetText(err.Error())
				return
			}
			s.simOperatorState(slot).SetStatus(text)
			s.statusLabel.SetText(text)
		})
	}()
}

func (s *objectCardDialogState) refreshSIMStatus(slot int) {
	startedText := "SIM API: перевірка стану..."
	switch s.currentSIMOperator(slot) {
	case simoperator.Vodafone:
		startedText = "Vodafone: перевірка стану..."
	case simoperator.Kyivstar:
		startedText = "Kyivstar: перевірка стану..."
	}

	s.runSIMOperatorAction(slot, startedText, func(operator simoperator.Operator, msisdn string) (string, error) {
		switch operator {
		case simoperator.Vodafone:
			status, err := s.provider.GetVodafoneSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			return s.vfSIMVM.BuildStatusText(status), nil
		case simoperator.Kyivstar:
			status, err := s.provider.GetKyivstarSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			return s.ksSIMVM.BuildStatusText(status), nil
		default:
			return "", nil
		}
	})
}

func (s *objectCardDialogState) rebootSIM(slot int) {
	startedText := "SIM API: створення заявки..."
	switch s.currentSIMOperator(slot) {
	case simoperator.Vodafone:
		startedText = "Vodafone: створення заявки на перезавантаження..."
	case simoperator.Kyivstar:
		startedText = "Kyivstar: запит на reset..."
	}

	s.runSIMOperatorAction(slot, startedText, func(operator simoperator.Operator, msisdn string) (string, error) {
		switch operator {
		case simoperator.Vodafone:
			result, err := s.provider.RebootVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(result.OrderID) == "" {
				return "Vodafone: заявку на перезавантаження створено", nil
			}
			if strings.TrimSpace(result.State) == "" {
				return "Vodafone: заявку створено, ID " + result.OrderID, nil
			}
			return "Vodafone: заявку створено, ID " + result.OrderID + ", стан " + result.State, nil
		case simoperator.Kyivstar:
			result, err := s.provider.RebootKyivstarSIM(msisdn)
			if err != nil {
				return "", err
			}
			return "Kyivstar: запит на reset відправлено для " + result.MSISDN, nil
		default:
			return "", nil
		}
	})
}

func (s *objectCardDialogState) syncSIMMetadata(slot int) {
	startedText := "SIM API: запис метаданих..."
	switch s.currentSIMOperator(slot) {
	case simoperator.Vodafone:
		startedText = "Vodafone: запис name/comment..."
	case simoperator.Kyivstar:
		startedText = "Kyivstar: запис deviceName/deviceId..."
	}

	s.runSIMOperatorAction(slot, startedText, func(operator simoperator.Operator, msisdn string) (string, error) {
		switch operator {
		case simoperator.Vodafone:
			name, comment, err := s.vfSIMVM.BuildMetadata(msisdn, s.objnEntry.Text, s.shortNameEntry.Text, s.fullNameEntry.Text)
			if err != nil {
				return "", err
			}
			if err := s.provider.UpdateVodafoneSIMMetadata(msisdn, name, comment); err != nil {
				return "", err
			}
			return "Vodafone: name/comment оновлено", nil
		case simoperator.Kyivstar:
			deviceName, deviceID, err := s.ksSIMVM.BuildMetadata(msisdn, s.objnEntry.Text, s.shortNameEntry.Text, s.fullNameEntry.Text)
			if err != nil {
				return "", err
			}
			if err := s.provider.UpdateKyivstarSIMMetadata(msisdn, deviceName, deviceID); err != nil {
				return "", err
			}
			return "Kyivstar: deviceName/deviceId оновлено", nil
		default:
			return "", nil
		}
	})
}

func (s *objectCardDialogState) openSIMDialog(slot int) {
	name := strings.TrimSpace(s.shortNameEntry.Text)
	if name == "" {
		name = strings.TrimSpace(s.fullNameEntry.Text)
	}
	switch s.currentSIMOperator(slot) {
	case simoperator.Vodafone:
		ShowVodafoneSIMDialog(
			s.win,
			s.provider,
			s.currentSIM(slot),
			s.objnEntry.Text,
			name,
		)
	case simoperator.Kyivstar:
		ShowKyivstarSIMDialog(
			s.win,
			s.provider,
			s.currentSIM(slot),
			s.objnEntry.Text,
			name,
		)
	default:
		s.simOperatorState(slot).SetStatus("SIM API: оператор номера не підтримується")
	}
}

func (s *objectCardDialogState) loadReferenceData() error {
	if err := s.refsVM.LoadFromProvider(s.provider); err != nil {
		return err
	}

	s.objectTypeSelect.Options = s.refsVM.ObjectTypeOptions()
	s.objectTypeSelect.Refresh()

	s.regionSelect.Options = s.refsVM.RegionOptions()
	s.regionSelect.Refresh()

	s.refreshPPKOptionsByChannel(s.refsVM.PPKID(s.ppkSelect.Selected))

	s.subServerASelect.Options = s.refsVM.SubServerOptions()
	s.subServerASelect.Refresh()
	s.subServerBSelect.Options = s.refsVM.SubServerOptions()
	s.subServerBSelect.Refresh()

	return nil
}

func (s *objectCardDialogState) fillDefaults() {
	defaults := s.formVM.Defaults()
	presentation := s.defVM.BuildPresentation(
		defaults,
		s.refsVM,
		s.channelCodeToLabel,
		s.dateVM.FormatForDisplay(time.Now()),
	)

	s.objnEntry.SetText(presentation.ObjNText)
	s.shortNameEntry.SetText(presentation.ShortName)
	s.fullNameEntry.SetText(presentation.FullName)
	s.vm.ResetNameSync(presentation.ShortName, presentation.FullName)
	s.addressEntry.SetText(presentation.Address)
	s.phonesEntry.SetText(presentation.Phones)
	s.contractEntry.SetText(presentation.Contract)
	s.dateEntry.SetText(presentation.StartDateText)
	s.locationEntry.SetText(presentation.Location)
	s.notesEntry.SetText(presentation.Notes)
	s.channelCodeSelect.SetSelected(presentation.ChannelLabel)
	s.vm.SetChannelCode(presentation.ChannelCode)
	s.sim1Entry.SetText(presentation.GSMPhone1)
	s.sim2Entry.SetText(presentation.GSMPhone2)
	s.simState.Clear()
	s.resetSIMOperatorStatus(1)
	s.resetSIMOperatorStatus(2)
	s.hiddenNEntry.SetText(presentation.GSMHiddenNText)
	s.testControlCheck.SetChecked(presentation.TestControlEnabled)
	s.testIntervalEntry.SetText(presentation.TestIntervalMinText)
	s.enableTestControls(presentation.TestControlEnabled)
	s.updateChannelSpecificControls()
	s.refreshPPKOptionsByChannel(0)

	s.objectTypeSelect.SetSelected(presentation.ObjectTypeLabel)
	s.regionSelect.SetSelected(presentation.RegionLabel)
	s.subServerASelect.SetSelected(presentation.SubServerALabel)
	s.subServerBSelect.SetSelected(presentation.SubServerBLabel)
}

func (s *objectCardDialogState) loadCard(objn int64) error {
	card, err := s.provider.GetObjectCard(objn)
	if err != nil {
		return err
	}

	presentation := s.loadVM.BuildPresentation(card, s.refsVM, s.channelCodeToLabel)

	s.objnEntry.SetText(presentation.ObjNText)
	s.shortNameEntry.SetText(presentation.ShortName)
	s.fullNameEntry.SetText(presentation.FullName)
	s.vm.ResetNameSync(presentation.ShortName, presentation.FullName)
	s.addressEntry.SetText(presentation.Address)
	s.phonesEntry.SetText(presentation.Phones)
	s.contractEntry.SetText(presentation.Contract)
	s.dateEntry.SetText(presentation.StartDate)
	s.locationEntry.SetText(presentation.Location)
	s.notesEntry.SetText(presentation.Notes)
	s.channelCodeSelect.SetSelected(presentation.ChannelLabel)
	s.vm.SetChannelCode(presentation.ChannelCode)
	s.refreshPPKOptionsByChannel(presentation.PPKID)
	s.sim1Entry.SetText(presentation.GSMPhone1)
	s.sim2Entry.SetText(presentation.GSMPhone2)
	s.checkSIMUsage(presentation.GSMPhone1, 1)
	s.checkSIMUsage(presentation.GSMPhone2, 2)
	s.resetSIMOperatorStatus(1)
	s.resetSIMOperatorStatus(2)
	s.hiddenNEntry.SetText(presentation.GSMHiddenNText)
	s.testControlCheck.SetChecked(presentation.TestControlEnabled)
	s.testIntervalEntry.SetText(presentation.TestIntervalMinText)
	s.enableTestControls(presentation.TestControlEnabled)
	s.objectTypeSelect.SetSelected(presentation.ObjectTypeLabel)
	s.regionSelect.SetSelected(presentation.RegionLabel)
	s.subServerASelect.SetSelected(presentation.SubServerALabel)
	s.subServerBSelect.SetSelected(presentation.SubServerBLabel)
	s.updateChannelSpecificControls()

	return nil
}

func (s *objectCardDialogState) buildCardFromUI() (contracts.AdminObjectCard, error) {
	input, err := s.formVM.BuildInput(viewmodels.ObjectCardFormSnapshot{
		ObjNRaw:            s.objnEntry.Text,
		ShortName:          s.shortNameEntry.Text,
		FullName:           s.fullNameEntry.Text,
		Address:            s.addressEntry.Text,
		Phones:             s.phonesEntry.Text,
		Contract:           s.contractEntry.Text,
		StartDate:          s.dateEntry.Text,
		Location:           s.locationEntry.Text,
		Notes:              s.notesEntry.Text,
		GSMPhone1:          s.sim1Entry.Text,
		GSMPhone2:          s.sim2Entry.Text,
		GSMHiddenNRaw:      s.hiddenNEntry.Text,
		ChannelLabel:       s.channelCodeSelect.Selected,
		TestControlEnabled: s.testControlCheck.Checked,
		TestIntervalMinRaw: s.testIntervalEntry.Text,
		ObjectTypeLabel:    s.objectTypeSelect.Selected,
		RegionLabel:        s.regionSelect.Selected,
		PPKLabel:           s.ppkSelect.Selected,
		SubServerALabel:    s.subServerASelect.Selected,
		SubServerBLabel:    s.subServerBSelect.Selected,
	}, s.refsVM, s.channelLabelToCode)
	if err != nil {
		return contracts.AdminObjectCard{}, err
	}
	return s.vm.ValidateAndBuildCard(input)
}

func (s *objectCardDialogState) openDatePicker() {
	initial := s.dateVM.ResolvePickerInitial(s.dateEntry.Text, time.Now())

	var pickerDlg dialog.Dialog
	calendar := xwidget.NewCalendar(initial, func(selected time.Time) {
		s.dateEntry.SetText(s.dateVM.FormatForDisplay(selected))
		if pickerDlg != nil {
			pickerDlg.Hide()
		}
	})
	pickerDlg = dialog.NewCustom("Вибір дати", "Закрити", container.NewPadded(calendar), s.win)
	pickerDlg.Show()
}

func (s *objectCardDialogState) setRegionByID(regionID int64) bool {
	if label, ok := s.refsVM.RegionLabelByIDExact(regionID); ok {
		s.regionSelect.SetSelected(label)
		return true
	}
	return false
}

func (s *objectCardDialogState) buildDateRow() fyne.CanvasObject {
	datePickBtn := widget.NewButton("...", s.openDatePicker)
	datePickBtn.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, datePickBtn, s.dateEntry)
}

func (s *objectCardDialogState) buildMainInfoForm(dateRow fyne.CanvasObject) *widget.Form {
	objectAndTypeRow := container.NewGridWithColumns(
		2,
		s.shortNameEntry,
		s.objectTypeSelect,
	)
	phoneContractDateRow := container.NewGridWithColumns(
		3,
		s.phonesEntry,
		s.contractEntry,
		dateRow,
	)
	channelAndPPKRow := container.NewGridWithColumns(2, s.channelCodeSelect, s.ppkSelect)

	return widget.NewForm(
		widget.NewFormItem("№ об'єкта:", s.objnEntry),
		widget.NewFormItem("Об'єкт / Тип:", objectAndTypeRow),
		widget.NewFormItem("Повна назва:", s.fullNameEntry),
		widget.NewFormItem("Телефони / Договір / Дата:", phoneContractDateRow),
		widget.NewFormItem("Адреса:", s.addressEntry),
		widget.NewFormItem("Розташування:", s.locationEntry),
		widget.NewFormItem("Інформація:", s.notesEntry),
		widget.NewFormItem("Район:", s.regionSelect),
		widget.NewFormItem("Канал / ППК:", channelAndPPKRow),
	)
}

func (s *objectCardDialogState) buildTestControlCard() fyne.CanvasObject {
	testControlForm := widget.NewForm(
		widget.NewFormItem("Контролювати:", s.testControlCheck),
		widget.NewFormItem("Інтервал, хв.:", container.NewGridWrap(fyne.NewSize(90, 36), s.testIntervalEntry)),
	)
	return widget.NewCard("Контроль GPRS/тестів", "", testControlForm)
}

func (s *objectCardDialogState) buildSIMPhonesCard() fyne.CanvasObject {
	simActions := func(slot int) fyne.CanvasObject {
		return container.NewHBox(
			makeLowButton("Статус", func() { s.refreshSIMStatus(slot) }),
			makeLowButton("Перезавантажити", func() { s.rebootSIM(slot) }),
			makeLowButton("Записати №/назву", func() { s.syncSIMMetadata(slot) }),
			makeLowButton("Блок/розблок", func() { s.openSIMDialog(slot) }),
		)
	}

	simPhonesForm := widget.NewForm(
		widget.NewFormItem("SIM1:", container.NewVBox(s.sim1Entry, s.sim1UsageLabel, s.sim1OperatorLabel, simActions(1))),
		widget.NewFormItem("SIM2:", container.NewVBox(s.sim2Entry, s.sim2UsageLabel, s.sim2OperatorLabel, simActions(2))),
	)
	return widget.NewCard("Телефони", "", simPhonesForm)
}

func (s *objectCardDialogState) buildSubserverCard() fyne.CanvasObject {
	subserverForm := widget.NewForm(
		widget.NewFormItem("Підсервер A:", s.subServerASelect),
		widget.NewFormItem("Підсервер B:", s.subServerBSelect),
	)
	return widget.NewCard("Підсервери", "", subserverForm)
}

func (s *objectCardDialogState) buildObjectTab() fyne.CanvasObject {
	dateRow := s.buildDateRow()
	mainInfoForm := s.buildMainInfoForm(dateRow)
	testControlCard := s.buildTestControlCard()
	simPhonesCard := s.buildSIMPhonesCard()
	subserverCard := s.buildSubserverCard()
	ppkParamsRow := container.NewGridWithColumns(3, testControlCard, simPhonesCard, s.hiddenNCard)

	return container.NewVScroll(container.NewVBox(
		mainInfoForm,
		widget.NewSeparator(),
		makeSectionHeader("Технічні параметри"),
		ppkParamsRow,
		widget.NewSeparator(),
		subserverCard,
	))
}

func (s *objectCardDialogState) placeholderTabText() string {
	if s.isEdit {
		return "Вкладка буде перенесена за формами з D:\\most_output (frmObjChange.dfm)."
	}
	return "Для цієї вкладки спочатку збережіть новий об'єкт."
}

func (s *objectCardDialogState) saveObject() {
	out, err := s.submitVM.Submit(viewmodels.ObjectCardSubmitInput{
		BuildCard:   s.buildCardFromUI,
		Persistence: s.provider,
		EditObjN:    s.editObjN,
	})
	if err != nil {
		if out.ShowErrorDialog {
			dialog.ShowError(err, s.win)
		}
		s.statusLabel.SetText(out.StatusMessage)
		return
	}
	s.statusLabel.SetText(out.StatusMessage)
	if s.onSaved != nil {
		s.onSaved(out.Result.ObjN)
	}
	s.win.Close()
}

func (s *objectCardDialogState) buildPlaceholderTab(text string) fyne.CanvasObject {
	return container.NewPadded(widget.NewLabel(text))
}

func (s *objectCardDialogState) buildRelatedTabs(placeholderText string) (fyne.CanvasObject, fyne.CanvasObject, fyne.CanvasObject) {
	personalTab := s.buildPlaceholderTab(placeholderText)
	zonesTab := s.buildPlaceholderTab(placeholderText)
	additionalTab := s.buildPlaceholderTab(placeholderText)
	if !s.isEdit {
		return personalTab, zonesTab, additionalTab
	}

	personalTab = buildObjectPersonalTab(s.win, s.provider, *s.editObjN, s.statusLabel)
	zonesTab = buildObjectZonesTab(s.win, s.provider, *s.editObjN, s.statusLabel)
	additionalTab = buildObjectAdditionalTab(
		s.win,
		s.provider,
		*s.editObjN,
		s.statusLabel,
		func() string {
			return strings.TrimSpace(s.addressEntry.Text)
		},
		s.setRegionByID,
	)
	return personalTab, zonesTab, additionalTab
}

func (s *objectCardDialogState) buildTabs(objectTab fyne.CanvasObject, placeholderText string) fyne.CanvasObject {
	personalTab, zonesTab, additionalTab := s.buildRelatedTabs(placeholderText)
	return container.NewAppTabs(
		container.NewTabItem("Об'єкт", objectTab),
		container.NewTabItem("В/О", personalTab),
		container.NewTabItem("Зображення", s.buildPlaceholderTab(placeholderText)),
		container.NewTabItem("Зони", zonesTab),
		container.NewTabItem("Додатково", additionalTab),
	)
}

func (s *objectCardDialogState) buildFooter() fyne.CanvasObject {
	saveBtn := makePrimaryButton("Зберегти", s.saveObject)
	cancelBtn := makeLowButton("Відміна", func() { s.win.Close() })
	return container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(s.statusLabel, layout.NewSpacer(), saveBtn, cancelBtn),
	)
}

func (s *objectCardDialogState) buildContent() fyne.CanvasObject {
	objectTab := s.buildObjectTab()
	placeholderText := s.placeholderTabText()

	return container.NewBorder(
		nil,
		s.buildFooter(),
		nil,
		nil,
		s.buildTabs(objectTab, placeholderText),
	)
}

func (s *objectCardDialogState) initializeDialogData() {
	result := s.initVM.Initialize(viewmodels.ObjectCardInitInput{
		EditObjN:          s.editObjN,
		LoadReferenceData: s.loadReferenceData,
		PrepareEditMode: func() {
			s.objnEntry.Disable()
		},
		LoadCard:     s.loadCard,
		FillDefaults: s.fillDefaults,
	})

	for _, issue := range result.Issues {
		if issue.ShowErrorDialog && issue.Err != nil {
			dialog.ShowError(issue.Err, s.win)
		}
		if issue.StatusMessage != "" {
			s.statusLabel.SetText(issue.StatusMessage)
		}
	}
}

func showObjectCardDialog(parent fyne.Window, provider contracts.AdminObjectCardProvider, editObjN *int64, onSaved func(objn int64)) {
	title := "Редагування/Створення об'єкта"

	win := fyne.CurrentApp().NewWindow(title)
	win.Resize(fyne.NewSize(800, 600))

	state := newObjectCardDialogState(win, provider, editObjN, onSaved)
	win.SetContent(state.buildContent())
	state.initializeDialogData()
	win.Show()
}
