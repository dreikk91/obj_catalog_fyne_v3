package dialogs

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type objectWizardDialogState struct {
	provider         contracts.AdminObjectWizardProvider
	personalsStateVM *viewmodels.ObjectWizardPersonalsStateViewModel
	zonesStateVM     *viewmodels.ObjectWizardZonesStateViewModel
	objectCardVM     *viewmodels.ObjectCardViewModel
	formVM           *viewmodels.ObjectCardFormViewModel
	defaultsVM       *viewmodels.ObjectCardDefaultsViewModel
	refsVM           *viewmodels.ObjectCardReferencesViewModel
	wizardVM         *viewmodels.ObjectWizardViewModel
	initVM           *viewmodels.ObjectWizardInitViewModel
	reviewVM         *viewmodels.ObjectWizardReviewViewModel
	personalsVM      *viewmodels.ObjectWizardPersonalsTableViewModel
	personalsFlow    *viewmodels.ObjectWizardPersonalsFlowViewModel
	zonesStepVM      *viewmodels.ObjectWizardZonesStepViewModel
	zonesFlowVM      *viewmodels.ObjectWizardZonesFlowViewModel
	coordsFlowVM     *viewmodels.ObjectWizardCoordinatesFlowViewModel
	channelFlowVM    *viewmodels.ObjectChannelFlowViewModel
	simUsageVM       *viewmodels.SIMPhoneUsageViewModel
	simStateVM       *viewmodels.ObjectWizardSIMUsageStateViewModel
	dateVM           *viewmodels.ObjectDateFieldViewModel

	shortNameBinding binding.String
	fullNameBinding  binding.String

	channelLabelToCode map[string]int64
	channelCodeToLabel map[int64]string

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
	hiddenNEntry      *widget.Entry
	hiddenNRow        *fyne.Container

	testControlCheck  *widget.Check
	testIntervalEntry *widget.Entry

	objectTypeSelect *widget.Select
	regionSelect     *widget.Select
	ppkSelect        *widget.Select
	subServerASelect *widget.Select
	subServerBSelect *widget.Select

	latitudeEntry  *widget.Entry
	longitudeEntry *widget.Entry
}

func (s *objectWizardDialogState) onFullNameBindingChanged() {
	fullName, err := s.fullNameBinding.Get()
	if err != nil {
		return
	}
	shortName, err := s.shortNameBinding.Get()
	if err != nil {
		return
	}
	s.objectCardVM.OnFullNameChanged(fullName, shortName)
}

func (s *objectWizardDialogState) onShortNameBindingChanged() {
	shortName, err := s.shortNameBinding.Get()
	if err != nil {
		return
	}
	fullName, shouldApply := s.objectCardVM.OnShortNameChanged(shortName)
	if !shouldApply {
		return
	}
	_ = s.fullNameBinding.Set(fullName)
}

func (s *objectWizardDialogState) enableTestControls(enabled bool) {
	if enabled {
		s.testIntervalEntry.Enable()
		return
	}
	s.testIntervalEntry.Disable()
}

func (s *objectWizardDialogState) updateChannelSpecificControls() {
	if s.objectCardVM.ShouldShowHiddenNumber() {
		s.hiddenNRow.Show()
		s.hiddenNEntry.Enable()
		return
	}
	s.hiddenNRow.Hide()
	s.hiddenNEntry.Disable()
}

func (s *objectWizardDialogState) refreshPPKOptionsByChannel(preferredID int64) {
	selected := s.refsVM.RefreshPPKOptions(preferredID)
	s.ppkSelect.Options = s.refsVM.PPKOptions()
	s.ppkSelect.Refresh()
	s.ppkSelect.SetSelected(selected)
}

func (s *objectWizardDialogState) onChannelChanged() {
	change := s.channelFlowVM.ResolveChange(
		s.channelCodeSelect.Selected,
		s.ppkSelect.Selected,
		s.channelLabelToCode,
		s.refsVM.PPKID,
	)
	s.objectCardVM.SetChannelCode(change.ChannelCode)
	s.updateChannelSpecificControls()
	s.refreshPPKOptionsByChannel(change.PreferredPPKID)
}

func (s *objectWizardDialogState) checkSIMUsage(rawPhone string, slot int) {
	text := s.simUsageVM.ResolveUsageText(s.provider, rawPhone, nil)
	if slot == 2 {
		s.simStateVM.SetSIM2(text)
		return
	}
	s.simStateVM.SetSIM1(text)
}

func (s *objectWizardDialogState) loadReferenceData() error {
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

func (s *objectWizardDialogState) fillDefaults() {
	defaults := s.formVM.Defaults()
	presentation := s.defaultsVM.BuildPresentation(
		defaults,
		s.refsVM,
		s.channelCodeToLabel,
		s.dateVM.FormatForDisplay(time.Now()),
	)

	s.objnEntry.SetText(presentation.ObjNText)
	s.shortNameEntry.SetText(presentation.ShortName)
	s.fullNameEntry.SetText(presentation.FullName)
	s.objectCardVM.ResetNameSync(presentation.ShortName, presentation.FullName)
	s.addressEntry.SetText(presentation.Address)
	s.phonesEntry.SetText(presentation.Phones)
	s.contractEntry.SetText(presentation.Contract)
	s.dateEntry.SetText(presentation.StartDateText)
	s.locationEntry.SetText(presentation.Location)
	s.notesEntry.SetText(presentation.Notes)
	s.channelCodeSelect.SetSelected(presentation.ChannelLabel)
	s.objectCardVM.SetChannelCode(presentation.ChannelCode)
	s.sim1Entry.SetText(presentation.GSMPhone1)
	s.sim2Entry.SetText(presentation.GSMPhone2)
	s.simStateVM.Clear()
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
	s.latitudeEntry.SetText("")
	s.longitudeEntry.SetText("")
	s.personalsStateVM.Reset()
	s.zonesStateVM.Reset()
}

func (s *objectWizardDialogState) buildCardFromUI() (contracts.AdminObjectCard, error) {
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
	return s.objectCardVM.ValidateAndBuildCard(input)
}

func (s *objectWizardDialogState) validateStep(step int) error {
	_, cardErr := s.buildCardFromUI()
	return s.wizardVM.ValidateStep(viewmodels.ObjectWizardStepValidationInput{
		Step:              step,
		ObjNRaw:           s.objnEntry.Text,
		ShortName:         s.shortNameEntry.Text,
		SelectedObjTypeID: s.refsVM.ObjectTypeID(s.objectTypeSelect.Selected),
		CardBuildErr:      cardErr,
	})
}

func (s *objectWizardDialogState) buildReviewText() string {
	_, cardBuildErr := s.buildCardFromUI()

	personals := s.personalsStateVM.Items()
	reviewPersonals := make([]viewmodels.ObjectWizardReviewPersonalItem, 0, len(personals))
	for _, it := range personals {
		reviewPersonals = append(reviewPersonals, viewmodels.ObjectWizardReviewPersonalItem{
			Number:   it.Number,
			FullName: s.personalsStateVM.FullName(it),
			Phones:   it.Phones,
			IsAdmin:  it.Access1 > 0,
		})
	}

	zones := s.zonesStateVM.Items()
	reviewZones := make([]viewmodels.ObjectWizardReviewZoneItem, 0, len(zones))
	for i, it := range zones {
		reviewZones = append(reviewZones, viewmodels.ObjectWizardReviewZoneItem{
			Number:      s.zonesStateVM.EffectiveNumberAt(i),
			Description: it.Description,
		})
	}

	return s.reviewVM.BuildText(viewmodels.ObjectWizardReviewInput{
		ObjN:               s.objnEntry.Text,
		ShortName:          s.shortNameEntry.Text,
		FullName:           s.fullNameEntry.Text,
		ObjectType:         s.objectTypeSelect.Selected,
		Region:             s.regionSelect.Selected,
		HiddenN:            s.hiddenNEntry.Text,
		Address:            s.addressEntry.Text,
		Phones:             s.phonesEntry.Text,
		Contract:           s.contractEntry.Text,
		StartDate:          s.dateEntry.Text,
		Location:           s.locationEntry.Text,
		Notes:              s.notesEntry.Text,
		Channel:            s.channelCodeSelect.Selected,
		PPK:                s.ppkSelect.Selected,
		SubServerA:         s.subServerASelect.Selected,
		SubServerB:         s.subServerBSelect.Selected,
		SIM1:               s.sim1Entry.Text,
		SIM2:               s.sim2Entry.Text,
		TestControlEnabled: s.testControlCheck.Checked,
		TestIntervalMin:    s.testIntervalEntry.Text,
		Personals:          reviewPersonals,
		Zones:              reviewZones,
		Latitude:           s.latitudeEntry.Text,
		Longitude:          s.longitudeEntry.Text,
		CardValidationErr:  cardBuildErr,
	})
}

func (s *objectWizardDialogState) buildObjectDataStep(dateRow fyne.CanvasObject) fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("Об'єктовий номер:", s.objnEntry),
		widget.NewFormItem("Коротка назва:", s.shortNameEntry),
		widget.NewFormItem("Повна назва:", s.fullNameEntry),
		widget.NewFormItem("Тип об'єкта:", s.objectTypeSelect),
		widget.NewFormItem("Район:", s.regionSelect),
		widget.NewFormItem("Канал:", s.channelCodeSelect),
		widget.NewFormItem("", s.hiddenNRow),
		widget.NewFormItem("Адреса:", s.addressEntry),
		widget.NewFormItem("Телефони:", s.phonesEntry),
		widget.NewFormItem("Договір:", s.contractEntry),
		widget.NewFormItem("Дата:", dateRow),
		widget.NewFormItem("Розташування:", s.locationEntry),
		widget.NewFormItem("Інформація:", s.notesEntry),
	)
}

func (s *objectWizardDialogState) buildDeviceParamsStep() fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("ППК:", s.ppkSelect),
		widget.NewFormItem("Підсервер A:", s.subServerASelect),
		widget.NewFormItem("Підсервер B:", s.subServerBSelect),
		widget.NewFormItem("SIM 1:", container.NewVBox(s.sim1Entry, s.sim1UsageLabel)),
		widget.NewFormItem("SIM 2:", container.NewVBox(s.sim2Entry, s.sim2UsageLabel)),
		widget.NewFormItem(
			"Контроль GPRS:",
			container.NewHBox(
				s.testControlCheck,
				widget.NewLabel("Інтервал (хв.):"),
				container.NewGridWrap(fyne.NewSize(100, 36), s.testIntervalEntry),
			),
		),
	)
}

func (s *objectWizardDialogState) buildAdditionalInfoStep(mapPickBtn *widget.Button, clearCoordsBtn *widget.Button) fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("Широта:", s.latitudeEntry),
		widget.NewFormItem("Довгота:", s.longitudeEntry),
		widget.NewFormItem("", container.NewHBox(mapPickBtn, clearCoordsBtn)),
	)
}

func (s *objectWizardDialogState) buildPersonalsStep(win fyne.Window, statusLabel *widget.Label) (fyne.CanvasObject, *widget.Table) {
	personalFullName := s.personalsStateVM.FullName

	personalTable := widget.NewTable(
		func() (int, int) { return s.personalsStateVM.Count() + 1, 6 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.SetText(s.personalsVM.HeaderText(id.Col))
				return
			}
			idx := id.Row - 1
			item, ok := s.personalsStateVM.At(idx)
			if !ok {
				lbl.SetText("")
				return
			}
			lbl.SetText(s.personalsVM.CellText(item, personalFullName(item), id.Col))
		},
	)
	personalTable.SetColumnWidth(0, 60)
	personalTable.SetColumnWidth(1, 270)
	personalTable.SetColumnWidth(2, 180)
	personalTable.SetColumnWidth(3, 170)
	personalTable.SetColumnWidth(4, 100)
	personalTable.SetColumnWidth(5, 230)
	personalTable.OnSelected = func(id widget.TableCellID) {
		s.personalsFlow.SelectTableRow(s.personalsStateVM, id.Row)
	}

	applyPersonalAction := func(out viewmodels.ObjectWizardPersonalsActionResult) {
		if out.RefreshTable {
			personalTable.Refresh()
		}
		if out.StatusText != "" {
			statusLabel.SetText(out.StatusText)
		}
	}

	addPersonalBtn := widget.NewButton("Додати", func() {
		showObjectPersonalEditor(win, s.provider, "Додати В/О", contracts.AdminObjectPersonal{
			Number: s.personalsFlow.NextNumber(s.personalsStateVM),
			IsRang: true,
		}, func(item contracts.AdminObjectPersonal) error {
			out := s.personalsFlow.ApplyAdd(s.personalsStateVM, item)
			applyPersonalAction(out)
			return nil
		}, statusLabel, nil)
	})
	editPersonalBtn := widget.NewButton("Змінити", func() {
		prompt := s.personalsFlow.PrepareEdit(s.personalsStateVM)
		if !prompt.CanEdit {
			statusLabel.SetText(prompt.StatusText)
			return
		}
		showObjectPersonalEditor(win, s.provider, "Редагування В/О", prompt.Initial, func(item contracts.AdminObjectPersonal) error {
			out := s.personalsFlow.ApplyUpdate(s.personalsStateVM, prompt.SelectedIdx, item)
			applyPersonalAction(out)
			return nil
		}, statusLabel, nil)
	})
	deletePersonalBtn := widget.NewButton("Видалити", func() {
		prompt := s.personalsFlow.PrepareDelete(s.personalsStateVM)
		if !prompt.CanDelete {
			statusLabel.SetText(prompt.StatusText)
			return
		}
		dialog.ShowConfirm(
			"Підтвердження",
			prompt.ConfirmText,
			func(ok bool) {
				if !ok {
					return
				}
				out := s.personalsFlow.ApplyDelete(s.personalsStateVM, prompt.SelectedIdx)
				applyPersonalAction(out)
			},
			win,
		)
	})

	step := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addPersonalBtn, editPersonalBtn, deletePersonalBtn),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		personalTable,
	)
	return step, personalTable
}

func (s *objectWizardDialogState) buildZonesStep(win fyne.Window, statusLabel *widget.Label) (fyne.CanvasObject, func(targetZoneNumber int64, focusQuickName bool)) {
	quickZoneNameEntry := widget.NewEntry()
	quickZoneNameEntry.SetPlaceHolder("Назва зони (Enter -> наступна зона)")
	selectedZoneLabel := widget.NewLabel("Зона: —")

	effectiveZoneNumberAt := s.zonesStateVM.EffectiveNumberAt

	updateSelectedZoneLabel := func() {
		selectedZoneLabel.SetText(s.zonesStateVM.SelectedLabel())
	}

	zoneTable := widget.NewTable(
		func() (int, int) { return s.zonesStateVM.Count() + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.SetText(s.zonesStepVM.HeaderText(id.Col))
				return
			}
			idx := id.Row - 1
			it, ok := s.zonesStateVM.At(idx)
			if !ok {
				lbl.SetText("")
				return
			}
			lbl.SetText(s.zonesStepVM.CellText(effectiveZoneNumberAt(idx), it.Description, id.Col))
		},
	)
	zoneTable.StickyRowCount = 1
	zoneTable.StickyColumnCount = 1
	applyZoneTableLayout := func() {
		zoneTable.SetColumnWidth(0, 120)
		zoneTable.SetColumnWidth(1, 120)
		zoneTable.SetColumnWidth(2, 520)
	}
	applyZoneTableLayout()

	var selectZoneByNumber func(zoneNumber int64, focusQuickName bool)
	var refreshZoneTable func(targetZoneNumber int64, focusQuickName bool)
	applyZoneAction := func(out viewmodels.ObjectWizardZonesActionResult) {
		if out.Err != nil && out.ShowErrorDialog {
			dialog.ShowError(out.Err, win)
		}
		if out.RefreshTable && refreshZoneTable != nil {
			refreshZoneTable(out.TargetZoneNumber, out.FocusQuickName)
		}
		if out.StatusText != "" {
			statusLabel.SetText(out.StatusText)
		}
	}

	refreshZoneTable = func(targetZoneNumber int64, focusQuickName bool) {
		zoneTable.Refresh()
		applyZoneTableLayout()
		if selectZoneByNumber != nil {
			selectZoneByNumber(targetZoneNumber, focusQuickName)
		}
	}
	selectZoneByNumber = func(zoneNumber int64, focusQuickName bool) {
		if !s.zonesStateVM.SelectByNumber(zoneNumber) {
			zoneTable.UnselectAll()
			quickZoneNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		targetRow := s.zonesStateVM.Selected()
		zoneTable.Select(widget.TableCellID{Row: targetRow + 1, Col: 0})
		quickZoneNameEntry.SetText(s.zonesStateVM.SelectedDescription())
		updateSelectedZoneLabel()
		if focusQuickName {
			focusIfOnCanvas(win, quickZoneNameEntry)
		}
	}

	zoneTable.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			s.zonesStateVM.SetSelected(-1)
			quickZoneNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		idx := id.Row - 1
		if !s.zonesStateVM.SetSelected(idx) {
			s.zonesStateVM.SetSelected(-1)
			quickZoneNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		quickZoneNameEntry.SetText(s.zonesStateVM.SelectedDescription())
		updateSelectedZoneLabel()
		focusIfOnCanvas(win, quickZoneNameEntry)
	}

	moveToNextZone := func() {
		out := s.zonesFlowVM.MoveToNext(s.zonesStateVM, quickZoneNameEntry.Text)
		applyZoneAction(out)
	}
	quickZoneNameEntry.OnSubmitted = func(string) {
		moveToNextZone()
	}

	addZoneBtn := widget.NewButton("Додати", func() {
		out := s.zonesFlowVM.AddZone(s.zonesStateVM)
		applyZoneAction(out)
	})

	editZoneBtn := widget.NewButton("Змінити", func() {
		out := s.zonesFlowVM.StartEdit(s.zonesStateVM)
		applyZoneAction(out)
	})

	deleteZoneBtn := widget.NewButton("Видалити", func() {
		prompt := s.zonesFlowVM.PrepareDelete(s.zonesStateVM)
		if !prompt.CanDelete {
			statusLabel.SetText(prompt.StatusText)
			return
		}
		dialog.ShowConfirm(
			"Підтвердження",
			prompt.ConfirmText,
			func(ok bool) {
				if !ok {
					return
				}
				out := s.zonesFlowVM.ApplyDelete(s.zonesStateVM, prompt.TargetZoneNumber)
				applyZoneAction(out)
			},
			win,
		)
	})

	defaultZoneFillCount := func() int64 {
		return s.zonesFlowVM.DefaultFillCount(s.zonesStateVM)
	}

	fillZonesBtn := widget.NewButton("Заповнити", func() {
		showZoneFillDialog(win, defaultZoneFillCount(), func(count int64) {
			out := s.zonesFlowVM.Fill(s.zonesStateVM, count)
			applyZoneAction(out)
		}, statusLabel)
	})

	clearZonesBtn := widget.NewButton("Очистити", func() {
		dialog.ShowConfirm(
			"Підтвердження",
			s.zonesFlowVM.ClearConfirmText(),
			func(ok bool) {
				if !ok {
					return
				}
				out := s.zonesFlowVM.Clear(s.zonesStateVM)
				applyZoneAction(out)
			},
			win,
		)
	})

	refreshZonesBtn := widget.NewButton("Оновити", func() {
		refreshZoneTable(0, false)
		statusLabel.SetText(s.zonesFlowVM.RefreshStatus(s.zonesStateVM))
	})
	nextZoneBtn := widget.NewButton("Enter -> Наступна", moveToNextZone)

	step := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addZoneBtn, editZoneBtn, deleteZoneBtn, fillZonesBtn, clearZonesBtn, layout.NewSpacer(), refreshZonesBtn),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(widget.NewLabel("Швидке введення:"), layout.NewSpacer(), selectedZoneLabel),
				nextZoneBtn,
				quickZoneNameEntry,
			),
		),
		nil,
		nil,
		zoneTable,
	)
	return step, refreshZoneTable
}

func ShowNewObjectWizardDialog(parent fyne.Window, provider contracts.AdminObjectWizardProvider, onSaved func(objn int64)) {
	win := fyne.CurrentApp().NewWindow("Майстер створення об'єкта")
	win.Resize(fyne.NewSize(1024, 768))

	statusLabel := widget.NewLabel("Крок 1/6: дані об'єкта")
	personalsStateVM := viewmodels.NewObjectWizardPersonalsStateViewModel()
	zonesStateVM := viewmodels.NewObjectWizardZonesStateViewModel()

	objnEntry := widget.NewEntry()
	shortNameBinding := binding.NewString()
	fullNameBinding := binding.NewString()
	shortNameEntry := widget.NewEntryWithData(shortNameBinding)
	fullNameEntry := widget.NewEntryWithData(fullNameBinding)
	addressEntry := widget.NewEntry()
	phonesEntry := widget.NewEntry()
	contractEntry := widget.NewEntry()
	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("дд.мм.рррр")
	locationEntry := widget.NewMultiLineEntry()
	locationEntry.SetMinRowsVisible(2)
	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetMinRowsVisible(4)
	latitudeEntry := widget.NewEntry()
	latitudeEntry.SetPlaceHolder("Широта (LATITUDE)")
	longitudeEntry := widget.NewEntry()
	longitudeEntry.SetPlaceHolder("Довгота (LONGITUDE)")

	channelCodeSelect := widget.NewSelect(viewmodels.ObjectChannelOptions(), nil)
	channelLabelToCode := viewmodels.DefaultObjectChannelLabelToCode()
	channelCodeToLabel := viewmodels.DefaultObjectChannelCodeToLabel()

	sim1Entry := widget.NewEntry()
	sim2Entry := widget.NewEntry()
	simStateVM := viewmodels.NewObjectWizardSIMUsageStateViewModel()
	sim1UsageLabel := widget.NewLabelWithData(simStateVM.SIM1Binding())
	sim1UsageLabel.Wrapping = fyne.TextWrapWord
	sim2UsageLabel := widget.NewLabelWithData(simStateVM.SIM2Binding())
	sim2UsageLabel.Wrapping = fyne.TextWrapWord
	hiddenNEntry := widget.NewEntry()
	hiddenNEntry.SetPlaceHolder("Прихований номер (до 4 цифр)")
	hiddenNRow := container.NewHBox(
		widget.NewLabel("Для каналу GPRS (5):"),
		container.NewGridWrap(fyne.NewSize(150, 36), hiddenNEntry),
	)

	testControlCheck := widget.NewCheck("Контролювати тестові повідомлення", nil)
	testIntervalEntry := widget.NewEntry()
	testIntervalEntry.SetPlaceHolder("хв.")

	objectTypeSelect := widget.NewSelect(nil, nil)
	regionSelect := widget.NewSelect(nil, nil)
	ppkSelect := widget.NewSelect(nil, nil)
	subServerASelect := widget.NewSelect(nil, nil)
	subServerBSelect := widget.NewSelect(nil, nil)

	objectCardVM := viewmodels.NewObjectCardViewModel()
	formVM := viewmodels.NewObjectCardFormViewModel()
	defaultsVM := viewmodels.NewObjectCardDefaultsViewModel()
	refsVM := viewmodels.NewObjectCardReferencesViewModel()
	wizardVM := viewmodels.NewObjectWizardViewModel()
	initVM := viewmodels.NewObjectWizardInitViewModel()
	reviewVM := viewmodels.NewObjectWizardReviewViewModel()
	personalsVM := viewmodels.NewObjectWizardPersonalsTableViewModel()
	personalsFlow := viewmodels.NewObjectWizardPersonalsFlowViewModel(personalsVM)
	zonesStepVM := viewmodels.NewObjectWizardZonesStepViewModel()
	zonesFlowVM := viewmodels.NewObjectWizardZonesFlowViewModel(zonesStepVM)
	coordsFlowVM := viewmodels.NewObjectWizardCoordinatesFlowViewModel()
	channelFlowVM := viewmodels.NewObjectChannelFlowViewModel()
	simUsageVM := viewmodels.NewSIMPhoneUsageViewModel()
	dateVM := viewmodels.NewObjectDateFieldViewModel()

	state := &objectWizardDialogState{
		provider:         provider,
		personalsStateVM: personalsStateVM,
		zonesStateVM:     zonesStateVM,
		objectCardVM:     objectCardVM,
		formVM:           formVM,
		defaultsVM:       defaultsVM,
		refsVM:           refsVM,
		wizardVM:         wizardVM,
		initVM:           initVM,
		reviewVM:         reviewVM,
		personalsVM:      personalsVM,
		personalsFlow:    personalsFlow,
		zonesStepVM:      zonesStepVM,
		zonesFlowVM:      zonesFlowVM,
		coordsFlowVM:     coordsFlowVM,
		channelFlowVM:    channelFlowVM,
		simUsageVM:       simUsageVM,
		simStateVM:       simStateVM,
		dateVM:           dateVM,

		shortNameBinding: shortNameBinding,
		fullNameBinding:  fullNameBinding,

		channelLabelToCode: channelLabelToCode,
		channelCodeToLabel: channelCodeToLabel,

		objnEntry:      objnEntry,
		shortNameEntry: shortNameEntry,
		fullNameEntry:  fullNameEntry,
		addressEntry:   addressEntry,
		phonesEntry:    phonesEntry,
		contractEntry:  contractEntry,
		dateEntry:      dateEntry,
		locationEntry:  locationEntry,
		notesEntry:     notesEntry,

		channelCodeSelect: channelCodeSelect,
		sim1Entry:         sim1Entry,
		sim2Entry:         sim2Entry,
		sim1UsageLabel:    sim1UsageLabel,
		sim2UsageLabel:    sim2UsageLabel,
		hiddenNEntry:      hiddenNEntry,
		hiddenNRow:        hiddenNRow,

		testControlCheck:  testControlCheck,
		testIntervalEntry: testIntervalEntry,

		objectTypeSelect: objectTypeSelect,
		regionSelect:     regionSelect,
		ppkSelect:        ppkSelect,
		subServerASelect: subServerASelect,
		subServerBSelect: subServerBSelect,

		latitudeEntry:  latitudeEntry,
		longitudeEntry: longitudeEntry,
	}

	state.fullNameBinding.AddListener(binding.NewDataListener(state.onFullNameBindingChanged))
	state.shortNameBinding.AddListener(binding.NewDataListener(state.onShortNameBindingChanged))

	testControlCheck.OnChanged = state.enableTestControls

	channelCodeSelect.OnChanged = func(_ string) {
		state.onChannelChanged()
	}

	sim1Entry.OnChanged = func(text string) {
		state.checkSIMUsage(text, 1)
	}
	sim2Entry.OnChanged = func(text string) {
		state.checkSIMUsage(text, 2)
	}
	buildCardFromUI := state.buildCardFromUI

	openDatePicker := func() {
		initial := state.dateVM.ResolvePickerInitial(dateEntry.Text, time.Now())

		var pickerDlg dialog.Dialog
		calendar := xwidget.NewCalendar(initial, func(selected time.Time) {
			dateEntry.SetText(state.dateVM.FormatForDisplay(selected))
			if pickerDlg != nil {
				pickerDlg.Hide()
			}
		})
		pickerDlg = dialog.NewCustom("Вибір дати", "Закрити", container.NewPadded(calendar), win)
		pickerDlg.Show()
	}
	datePickBtn := widget.NewButton("...", openDatePicker)
	datePickBtn.Importance = widget.LowImportance
	dateRow := container.NewBorder(nil, nil, nil, datePickBtn, dateEntry)

	validateStep := state.validateStep
	step3, personalTable := state.buildPersonalsStep(win, statusLabel)
	step4, refreshZoneTable := state.buildZonesStep(win, statusLabel)

	mapPickBtn := widget.NewButton("Вибрати на карті", func() {
		lat, lon := state.coordsFlowVM.PreparePickerInput(latitudeEntry.Text, longitudeEntry.Text)
		showCoordinatesMapPicker(
			win,
			lat,
			lon,
			func(lat, lon string) {
				out := state.coordsFlowVM.ApplyPicked(lat, lon)
				latitudeEntry.SetText(out.Latitude)
				longitudeEntry.SetText(out.Longitude)
				statusLabel.SetText(out.Status)
			},
		)
	})
	clearCoordsBtn := widget.NewButton("Очистити", func() {
		out := state.coordsFlowVM.Clear()
		latitudeEntry.SetText(out.Latitude)
		longitudeEntry.SetText(out.Longitude)
		statusLabel.SetText(out.Status)
	})

	reviewText := widget.NewTextGrid()
	reviewText.SetText("")
	reviewScroll := container.NewScroll(reviewText)
	reviewScroll.SetMinSize(fyne.NewSize(0, 420))

	refreshReview := func() {
		reviewText.SetText(state.buildReviewText())
	}

	step1 := state.buildObjectDataStep(dateRow)
	step2 := state.buildDeviceParamsStep()
	step5 := state.buildAdditionalInfoStep(mapPickBtn, clearCoordsBtn)

	step6 := container.NewBorder(
		widget.NewLabel("Підсумок введених даних перед створенням"),
		nil, nil, nil,
		reviewScroll,
	)

	steps := container.NewAppTabs(
		container.NewTabItem("1. Дані об'єкта", container.NewPadded(step1)),
		container.NewTabItem("2. Параметри пристрою", container.NewPadded(step2)),
		container.NewTabItem("3. Відповідальні", container.NewPadded(step3)),
		container.NewTabItem("4. Зони", container.NewPadded(step4)),
		container.NewTabItem("5. Додаткова інфо", container.NewPadded(step5)),
		container.NewTabItem("6. Підтвердження", container.NewPadded(step6)),
	)

	stepTitles := []string{
		"дані об'єкта",
		"параметри пристрою",
		"відповідальні особи",
		"зони",
		"додаткова інформація",
		"підтвердження",
	}
	flowVM := viewmodels.NewObjectWizardFlowViewModel(stepTitles)
	submitVM := viewmodels.NewObjectWizardSubmitViewModel(flowVM, wizardVM)

	updateStepState := func() {
		steps.SelectIndex(flowVM.CurrentStep())
		statusLabel.SetText(flowVM.StatusText())
		if flowVM.IsLastStep() {
			refreshReview()
		}
	}

	backBtn := widget.NewButton("Назад", nil)
	nextBtn := widget.NewButton("Далі", nil)
	createBtn := widget.NewButton("Створити", nil)
	cancelBtn := widget.NewButton("Скасувати", func() { win.Close() })

	refreshButtons := func() {
		if !flowVM.CanGoBack() {
			backBtn.Disable()
		} else {
			backBtn.Enable()
		}
		if !flowVM.CanGoNext() {
			nextBtn.Disable()
		} else {
			nextBtn.Enable()
		}
		if flowVM.CanCreate() {
			createBtn.Enable()
		} else {
			createBtn.Disable()
		}
	}

	backBtn.OnTapped = func() {
		if !flowVM.GoBack() {
			return
		}
		updateStepState()
		refreshButtons()
	}

	nextBtn.OnTapped = func() {
		moved, err := flowVM.GoNext(validateStep)
		if err != nil {
			statusLabel.SetText(err.Error())
			return
		}
		if !moved {
			return
		}
		updateStepState()
		refreshButtons()
	}

	createBtn.OnTapped = func() {
		out, err := submitVM.Submit(viewmodels.ObjectWizardSubmitInput{
			ValidateStep: validateStep,
			BuildCard:    buildCardFromUI,
			Persistence:  provider,
			Personals:    personalsStateVM.Items(),
			Zones:        zonesStateVM.Items(),
			Coordinates: contracts.AdminObjectCoordinates{
				Latitude:  latitudeEntry.Text,
				Longitude: longitudeEntry.Text,
			},
		})
		if err != nil {
			if out.ShowErrorDialog {
				dialog.ShowError(err, win)
			}
			statusLabel.SetText(out.StatusMessage)
			return
		}

		if out.WarningMessage != "" {
			dialog.ShowInformation(
				"Створено з попередженнями",
				out.WarningMessage,
				win,
			)
		} else {
			statusLabel.SetText(out.StatusMessage)
		}
		if onSaved != nil {
			onSaved(out.Result.ObjN)
		}
		win.Close()
	}

	content := container.NewBorder(
		nil,
		container.NewHBox(statusLabel, layout.NewSpacer(), backBtn, nextBtn, createBtn, cancelBtn),
		nil, nil,
		steps,
	)
	win.SetContent(content)

	initResult := state.initVM.Initialize(viewmodels.ObjectWizardInitInput{
		LoadReferenceData: state.loadReferenceData,
		FillDefaults:      state.fillDefaults,
	})
	for _, issue := range initResult.Issues {
		if issue.ShowErrorDialog && issue.Err != nil {
			dialog.ShowError(issue.Err, win)
		}
		if issue.StatusMessage != "" {
			statusLabel.SetText(issue.StatusMessage)
		}
	}
	updateStepState()
	refreshButtons()
	personalTable.Refresh()
	refreshZoneTable(0, false)

	win.Show()
}
