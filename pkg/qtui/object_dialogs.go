//go:build qt

package qtui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/simcommands"
	"obj_catalog_fyne_v3/pkg/simoperator"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type objectEditDialogState struct {
	parent   *qt.QWidget
	provider contracts.AdminObjectDialogProvider
	card     contracts.AdminObjectCard
	isNew    bool

	vm                 *viewmodels.ObjectCardViewModel
	formVM             *viewmodels.ObjectCardFormViewModel
	loadVM             *viewmodels.ObjectCardLoadViewModel
	refsVM             *viewmodels.ObjectCardReferencesViewModel
	channelVM          *viewmodels.ObjectChannelFlowViewModel
	simVM              *viewmodels.SIMPhoneUsageViewModel
	dateVM             *viewmodels.ObjectDateFieldViewModel
	channelLabelToCode map[string]int64
	channelCodeToLabel map[int64]string
	dirty              bool

	statusLabel *qt.QLabel

	objn         *qt.QLineEdit
	shortName    *qt.QLineEdit
	fullName     *qt.QLineEdit
	address      *qt.QLineEdit
	phones       *qt.QLineEdit
	contract     *qt.QLineEdit
	startDate    *qt.QLineEdit
	location     *qt.QTextEdit
	notes        *qt.QTextEdit
	sim1         *qt.QLineEdit
	sim2         *qt.QLineEdit
	sim1Usage    *qt.QLabel
	sim2Usage    *qt.QLabel
	hidden       *qt.QLineEdit
	channel      *qt.QComboBox
	ppk          *qt.QComboBox
	objectType   *qt.QComboBox
	region       *qt.QComboBox
	subServerA   *qt.QComboBox
	subServerB   *qt.QComboBox
	testEnabled  *qt.QCheckBox
	testInterval *qt.QLineEdit

	createPersonals   *objectCreatePersonalsTab
	createZones       *objectCreateZonesTab
	createCoordinates *objectCreateCoordinatesTab
}

type objectEditSIMLookup struct {
	base contracts.AdminObjectSIMLookupService
}

func (l objectEditSIMLookup) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]viewmodels.SIMPhoneUsage, error) {
	items, err := l.base.FindObjectsBySIMPhone(phone, excludeObjN)
	if err != nil {
		return nil, err
	}
	return viewmodels.SIMPhoneUsagesFromContracts(items), nil
}

func ShowObjectEditDialog(parent *qt.QWidget, provider contracts.AdminObjectDialogProvider, card contracts.AdminObjectCard) (contracts.AdminObjectCard, bool) {
	state := newObjectEditDialogState(parent, provider, card, false)
	if err := state.loadReferences(); err != nil {
		qt.QMessageBox_Critical(parent, "Редагування об'єкта", err.Error())
		return card, false
	}
	state.fillCard(card)
	return state.exec()
}

func ShowObjectCreateDialog(parent *qt.QWidget, provider contracts.AdminObjectDialogProvider) (contracts.AdminObjectCard, []string, bool) {
	state := newObjectEditDialogState(parent, provider, contracts.AdminObjectCard{}, true)
	if err := state.loadReferences(); err != nil {
		qt.QMessageBox_Critical(parent, "Створення об'єкта", err.Error())
		return contracts.AdminObjectCard{}, nil, false
	}
	state.fillDefaults()
	card, accepted := state.exec()
	if !accepted {
		return contracts.AdminObjectCard{}, nil, false
	}
	result, err := viewmodels.NewObjectWizardViewModel().CreateObjectWithRelatedData(
		qtObjectWizardPersistence{provider: provider},
		card,
		state.createPersonals.viewModelItems(),
		state.createZones.viewModelItems(),
		state.createCoordinates.coordinates(),
	)
	if err != nil {
		qt.QMessageBox_Critical(parent, "Створення об'єкта", err.Error())
		return contracts.AdminObjectCard{}, nil, false
	}
	return card, result.Warnings, true
}

func newObjectEditDialogState(parent *qt.QWidget, provider contracts.AdminObjectDialogProvider, card contracts.AdminObjectCard, isNew bool) *objectEditDialogState {
	state := &objectEditDialogState{
		parent:             parent,
		provider:           provider,
		card:               card,
		isNew:              isNew,
		vm:                 viewmodels.NewObjectCardViewModel(),
		formVM:             viewmodels.NewObjectCardFormViewModel(),
		loadVM:             viewmodels.NewObjectCardLoadViewModel(),
		refsVM:             viewmodels.NewObjectCardReferencesViewModel(),
		channelVM:          viewmodels.NewObjectChannelFlowViewModel(),
		simVM:              viewmodels.NewSIMPhoneUsageViewModel(),
		dateVM:             viewmodels.NewObjectDateFieldViewModel(),
		channelLabelToCode: viewmodels.DefaultObjectChannelLabelToCode(),
		channelCodeToLabel: viewmodels.DefaultObjectChannelCodeToLabel(),
		statusLabel:        qt.NewQLabel3("Готово"),
		objn:               newLineEdit(""),
		shortName:          newLineEdit(""),
		fullName:           newLineEdit(""),
		address:            newLineEdit(""),
		phones:             newLineEdit(""),
		contract:           newLineEdit(""),
		startDate:          newLineEdit(""),
		location:           qt.NewQTextEdit2(),
		notes:              qt.NewQTextEdit2(),
		sim1:               newLineEdit(""),
		sim2:               newLineEdit(""),
		sim1Usage:          qt.NewQLabel3(""),
		sim2Usage:          qt.NewQLabel3(""),
		hidden:             newLineEdit(""),
		channel:            qt.NewQComboBox2(),
		ppk:                qt.NewQComboBox2(),
		objectType:         qt.NewQComboBox2(),
		region:             qt.NewQComboBox2(),
		subServerA:         qt.NewQComboBox2(),
		subServerB:         qt.NewQComboBox2(),
		testEnabled:        qt.NewQCheckBox3("Контролювати тестові повідомлення"),
		testInterval:       newLineEdit(""),
	}
	state.objn.SetValidator(qt.NewQIntValidator2(1, 2147483647).QValidator)
	state.hidden.SetValidator(qt.NewQIntValidator2(1, 9999).QValidator)
	state.testInterval.SetValidator(qt.NewQIntValidator2(1, 1000000).QValidator)
	return state
}

func (s *objectEditDialogState) loadReferences() error {
	if s.provider == nil {
		return fmt.Errorf("поточне джерело даних не підтримує редагування картки об'єкта")
	}
	if err := s.refsVM.LoadFromProvider(s.provider); err != nil {
		return err
	}
	fillComboBox(s.objectType, s.refsVM.ObjectTypeOptions())
	fillComboBox(s.region, s.refsVM.RegionOptions())
	fillComboBox(s.channel, viewmodels.ObjectChannelOptions())
	fillComboBox(s.subServerA, s.refsVM.SubServerOptions())
	fillComboBox(s.subServerB, s.refsVM.SubServerOptions())
	return nil
}

func (s *objectEditDialogState) fillCard(card contracts.AdminObjectCard) {
	presentation := s.loadVM.BuildPresentation(card, s.refsVM, s.channelCodeToLabel)
	s.objn.SetText(presentation.ObjNText)
	s.objn.SetReadOnly(true)
	s.shortName.SetText(presentation.ShortName)
	s.fullName.SetText(presentation.FullName)
	s.vm.ResetNameSync(presentation.ShortName, presentation.FullName)
	s.address.SetText(presentation.Address)
	s.phones.SetText(presentation.Phones)
	s.contract.SetText(presentation.Contract)
	s.startDate.SetText(presentation.StartDate)
	s.location.SetPlainText(presentation.Location)
	s.notes.SetPlainText(presentation.Notes)
	setComboText(s.channel, presentation.ChannelLabel)
	s.vm.SetChannelCode(presentation.ChannelCode)
	s.refreshPPKOptions(presentation.PPKID)
	s.sim1.SetText(presentation.GSMPhone1)
	s.sim2.SetText(presentation.GSMPhone2)
	s.checkSIMUsage(1)
	s.checkSIMUsage(2)
	s.hidden.SetText(presentation.GSMHiddenNText)
	s.testEnabled.SetChecked(presentation.TestControlEnabled)
	s.testInterval.SetText(presentation.TestIntervalMinText)
	setComboText(s.objectType, presentation.ObjectTypeLabel)
	setComboText(s.region, presentation.RegionLabel)
	setComboText(s.subServerA, presentation.SubServerALabel)
	setComboText(s.subServerB, presentation.SubServerBLabel)
	s.updateTestControls()
	s.updateChannelSpecificControls()
}

func (s *objectEditDialogState) fillDefaults() {
	presentation := viewmodels.NewObjectCardDefaultsViewModel().BuildPresentation(
		s.formVM.Defaults(),
		s.refsVM,
		s.channelCodeToLabel,
		time.Now().Format("02.01.2006"),
	)
	s.objn.SetText(presentation.ObjNText)
	s.objn.SetReadOnly(false)
	s.shortName.SetText(presentation.ShortName)
	s.fullName.SetText(presentation.FullName)
	s.vm.ResetNameSync(presentation.ShortName, presentation.FullName)
	s.address.SetText(presentation.Address)
	s.phones.SetText(presentation.Phones)
	s.contract.SetText(presentation.Contract)
	s.startDate.SetText(presentation.StartDateText)
	s.location.SetPlainText(presentation.Location)
	s.notes.SetPlainText(presentation.Notes)
	setComboText(s.channel, presentation.ChannelLabel)
	s.vm.SetChannelCode(presentation.ChannelCode)
	s.refreshPPKOptions(0)
	s.sim1.SetText(presentation.GSMPhone1)
	s.sim2.SetText(presentation.GSMPhone2)
	s.hidden.SetText(presentation.GSMHiddenNText)
	s.testEnabled.SetChecked(presentation.TestControlEnabled)
	s.testInterval.SetText(presentation.TestIntervalMinText)
	setComboText(s.objectType, presentation.ObjectTypeLabel)
	setComboText(s.region, presentation.RegionLabel)
	setComboText(s.subServerA, presentation.SubServerALabel)
	setComboText(s.subServerB, presentation.SubServerBLabel)
	s.updateTestControls()
	s.updateChannelSpecificControls()
}

func (s *objectEditDialogState) exec() (contracts.AdminObjectCard, bool) {
	dialog := qt.NewQDialog(s.parent)
	title := "Редагування об'єкта МІСТ"
	if s.isNew {
		title = "Створення об'єкта МІСТ"
	}
	dialog.SetWindowTitle(title)
	dialog.Resize(820, 680)
	allowClose := false

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(func() {
		if _, err := s.buildCardFromUI(); err != nil {
			s.statusLabel.SetText(err.Error())
			qt.QMessageBox_Information(dialog.QWidget, title, err.Error())
			return
		}
		allowClose = true
		dialog.Accept()
	})
	buttons.OnRejected(func() {
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

	footer := qt.NewQHBoxLayout2()
	footer.AddWidget(s.statusLabel.QWidget)
	footer.AddStretch()
	footer.AddWidget(buttons.QWidget)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(s.buildTabs().QWidget)
	layout.AddLayout(footer.QLayout)
	dialog.SetLayout(layout.QLayout)

	s.wireEvents()

	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return s.card, false
	}
	updated, err := s.buildCardFromUI()
	if err != nil {
		qt.QMessageBox_Critical(s.parent, title, err.Error())
		return s.card, false
	}
	if !s.isNew {
		updated.ObjUIN = s.card.ObjUIN
		updated.GrpN = s.card.GrpN
	}
	return updated, true
}

func (s *objectEditDialogState) buildTabs() *qt.QTabWidget {
	tabs := qt.NewQTabWidget2()
	var loadPersonals func()
	var loadZones func()
	var loadAdditional func()
	tabs.AddTab(s.buildObjectTab(), "Об'єкт")
	if s.isNew {
		s.createPersonals = newObjectCreatePersonalsTab(s.parent)
		tabs.AddTab(s.createPersonals.widget(), "В/О")
	} else {
		var personalsWidget *qt.QWidget
		personalsWidget, loadPersonals = newObjectPersonalsEditTab(s.parent, s.provider, s.card.ObjN, s.statusLabel)
		tabs.AddTab(personalsWidget, "В/О")
	}
	tabs.AddTab(buildPlaceholderWidget("Зображення будуть перенесені окремо."), "Зображення")
	if s.isNew {
		s.createZones = newObjectCreateZonesTab(s.parent)
		tabs.AddTab(s.createZones.widget(), "Зони")
	} else {
		var zonesWidget *qt.QWidget
		zonesWidget, loadZones = newObjectZonesEditTab(s.parent, s.provider, s.card.ObjN, s.statusLabel)
		tabs.AddTab(zonesWidget, "Зони")
	}
	if s.isNew {
		s.createCoordinates = newObjectCreateCoordinatesTab(s.address.Text)
		tabs.AddTab(s.createCoordinates.widget(), "Додатково")
	} else {
		var additionalWidget *qt.QWidget
		additionalWidget, loadAdditional = newObjectAdditionalEditTab(s.parent, s.provider, s.card.ObjN, s.statusLabel, s.address.Text)
		tabs.AddTab(additionalWidget, "Додатково")
	}
	tabs.OnCurrentChanged(func(index int) {
		switch index {
		case 1:
			if loadPersonals != nil {
				loadPersonals()
			}
		case 3:
			if loadZones != nil {
				loadZones()
			}
		case 4:
			if loadAdditional != nil {
				loadAdditional()
			}
		}
	})
	return tabs
}

func (s *objectEditDialogState) buildObjectTab() *qt.QWidget {
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddWidget(s.buildMainForm())
	layout.AddWidget(s.buildTechnicalForm())
	layout.AddStretch()
	content.SetLayout(layout.QLayout)

	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

func (s *objectEditDialogState) buildMainForm() *qt.QWidget {
	widget := qt.NewQWidget2()
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	s.location.SetMinimumHeight(64)
	s.notes.SetMinimumHeight(96)
	s.startDate.SetPlaceholderText("дд.мм.рррр")
	dateButton := qt.NewQPushButton3("Обрати")
	dateButton.SetToolTip("Вибрати дату в календарі")
	dateButton.OnClicked(func() {
		if selected, ok := showObjectDatePicker(s.parent, s.dateVM, s.startDate.Text()); ok {
			s.startDate.SetText(selected)
			s.dirty = true
		}
	})

	form.AddRow3("№ об'єкта", s.objn.QWidget)
	form.AddRow3("Коротка назва", s.shortName.QWidget)
	form.AddRow3("Тип", s.objectType.QWidget)
	form.AddRow3("Повна назва", s.fullName.QWidget)
	form.AddRow3("Телефони", s.phones.QWidget)
	form.AddRow3("Договір", s.contract.QWidget)
	form.AddRow3("Дата", horizontalWidgets(s.startDate.QWidget, dateButton.QWidget))
	form.AddRow3("Адреса", s.address.QWidget)
	form.AddRow3("Розташування", s.location.QWidget)
	form.AddRow3("Інформація", s.notes.QWidget)
	form.AddRow3("Район", s.region.QWidget)
	form.AddRow3("Канал", s.channel.QWidget)
	form.AddRow3("ППК", s.ppk.QWidget)
	widget.SetLayout(form.QLayout)
	return widget
}

func (s *objectEditDialogState) buildTechnicalForm() *qt.QWidget {
	widget := qt.NewQWidget2()
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	s.sim1Usage.SetWordWrap(true)
	s.sim2Usage.SetWordWrap(true)
	s.hidden.SetPlaceholderText("Прихований номер для GPRS")
	s.testInterval.SetPlaceholderText("хв.")

	form.AddRow3("SIM 1", stackedWidgets(s.sim1.QWidget, s.sim1Usage.QWidget))
	form.AddRow3("SIM 2", stackedWidgets(s.sim2.QWidget, s.sim2Usage.QWidget))
	form.AddRow3("GSM hidden", s.hidden.QWidget)
	form.AddRow3("Контроль GPRS/тестів", s.testEnabled.QWidget)
	form.AddRow3("Інтервал, хв", s.testInterval.QWidget)
	form.AddRow3("Підсервер A", s.subServerA.QWidget)
	form.AddRow3("Підсервер B", s.subServerB.QWidget)
	widget.SetLayout(form.QLayout)
	return widget
}

func (s *objectEditDialogState) wireEvents() {
	s.shortName.OnTextEdited(func(text string) {
		s.dirty = true
		fullName, ok := s.vm.OnShortNameChanged(text)
		if ok {
			s.fullName.SetText(fullName)
		}
	})
	s.fullName.OnTextEdited(func(text string) {
		s.dirty = true
		s.vm.OnFullNameChanged(text, s.shortName.Text())
	})
	s.channel.OnCurrentTextChanged(func(_ string) {
		s.dirty = true
		change := s.channelVM.ResolveChange(s.channel.CurrentText(), s.ppk.CurrentText(), s.channelLabelToCode, s.refsVM.PPKID)
		s.vm.SetChannelCode(change.ChannelCode)
		s.refreshPPKOptions(change.PreferredPPKID)
		s.updateChannelSpecificControls()
	})
	s.testEnabled.OnStateChanged(func(_ int) {
		s.dirty = true
		s.updateTestControls()
	})
	s.sim1.OnTextEdited(func(_ string) {
		s.dirty = true
		s.checkSIMUsage(1)
	})
	s.sim2.OnTextEdited(func(_ string) {
		s.dirty = true
		s.checkSIMUsage(2)
	})
	for _, edit := range []*qt.QLineEdit{
		s.objn, s.address, s.phones, s.contract, s.startDate, s.hidden, s.testInterval,
	} {
		edit.OnTextEdited(func(_ string) { s.dirty = true })
	}
	s.location.OnTextChanged(func() { s.dirty = true })
	s.notes.OnTextChanged(func() { s.dirty = true })
	for _, combo := range []*qt.QComboBox{s.ppk, s.objectType, s.region, s.subServerA, s.subServerB} {
		combo.OnCurrentIndexChanged(func(_ int) { s.dirty = true })
	}
}

func (s *objectEditDialogState) refreshPPKOptions(preferredID int64) {
	selected := s.refsVM.RefreshPPKOptions(preferredID)
	fillComboBox(s.ppk, s.refsVM.PPKOptions())
	setComboText(s.ppk, selected)
}

func (s *objectEditDialogState) updateChannelSpecificControls() {
	s.hidden.SetEnabled(s.vm.ShouldShowHiddenNumber())
}

func (s *objectEditDialogState) updateTestControls() {
	s.testInterval.SetEnabled(s.testEnabled.IsChecked())
}

func (s *objectEditDialogState) checkSIMUsage(slot int) {
	if s.provider == nil {
		return
	}
	rawPhone := s.sim1.Text()
	label := s.sim1Usage
	if slot == 2 {
		rawPhone = s.sim2.Text()
		label = s.sim2Usage
	}
	var exclude *int64
	if !s.isNew {
		exclude = &s.card.ObjN
	}
	text := s.simVM.ResolveUsageText(objectEditSIMLookup{base: s.provider}, rawPhone, exclude)
	label.SetText(strings.TrimSpace(text))
}

func (s *objectEditDialogState) buildCardFromUI() (contracts.AdminObjectCard, error) {
	input, err := s.formVM.BuildInput(viewmodels.ObjectCardFormSnapshot{
		ObjNRaw:            s.objn.Text(),
		ShortName:          s.shortName.Text(),
		FullName:           s.fullName.Text(),
		Address:            s.address.Text(),
		Phones:             s.phones.Text(),
		Contract:           s.contract.Text(),
		StartDate:          s.startDate.Text(),
		Location:           s.location.ToPlainText(),
		Notes:              s.notes.ToPlainText(),
		GSMPhone1:          s.sim1.Text(),
		GSMPhone2:          s.sim2.Text(),
		GSMHiddenNRaw:      s.hidden.Text(),
		ChannelLabel:       s.channel.CurrentText(),
		TestControlEnabled: s.testEnabled.IsChecked(),
		TestIntervalMinRaw: s.testInterval.Text(),
		ObjectTypeLabel:    s.objectType.CurrentText(),
		RegionLabel:        s.region.CurrentText(),
		PPKLabel:           s.ppk.CurrentText(),
		SubServerALabel:    s.subServerA.CurrentText(),
		SubServerBLabel:    s.subServerB.CurrentText(),
	}, s.refsVM, s.channelLabelToCode)
	if err != nil {
		return contracts.AdminObjectCard{}, err
	}
	card, err := s.vm.ValidateAndBuildCard(input)
	if err != nil {
		return contracts.AdminObjectCard{}, err
	}
	if rawDate := strings.TrimSpace(card.StartDate); rawDate != "" {
		parsed, ok := s.dateVM.Parse(rawDate)
		if !ok {
			return contracts.AdminObjectCard{}, fmt.Errorf("некоректна дата, використовуйте формат дд.мм.рррр")
		}
		card.StartDate = s.dateVM.FormatForDisplay(parsed)
	}
	if strings.TrimSpace(card.StartDate) == "" {
		card.StartDate = time.Now().Format("02.01.2006")
	}
	return card, nil
}

func (s *objectEditDialogState) confirmDiscard(parent *qt.QWidget) bool {
	if !s.hasUnsavedChanges() {
		return true
	}
	return qt.QMessageBox_Question(
		parent,
		"Незбережені зміни",
		"Закрити картку та відкинути незбережені зміни?",
	) == qt.QMessageBox__Yes
}

func (s *objectEditDialogState) hasUnsavedChanges() bool {
	if s.dirty {
		return true
	}
	if !s.isNew {
		return false
	}
	return (s.createPersonals != nil && s.createPersonals.hasChanges()) ||
		(s.createZones != nil && s.createZones.hasChanges()) ||
		(s.createCoordinates != nil && s.createCoordinates.hasChanges())
}

func ShowSIMManagementDialog(
	parent *qt.QWidget,
	object models.Object,
	usageText string,
	vfService contracts.AdminObjectVodafoneService,
	ksService contracts.AdminObjectKyivstarService,
	sendSMS func(models.Object, string),
) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("SIM-карти об'єкта")
	dialog.Resize(640, 480)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Об'єкт", qt.NewQLabel3(fmt.Sprintf("<b>%s</b> №%s", htmlEscape(strings.TrimSpace(object.Name)), viewmodels.ObjectDisplayNumber(object))).QWidget)

	// SIM 1 Layout with Action Buttons if supported
	sim1Widget := qt.NewQWidget2()
	sim1Layout := qt.NewQHBoxLayout(sim1Widget)
	sim1Layout.SetContentsMargins(0, 0, 0, 0)
	sim1Layout.AddWidget(qt.NewQLabel3(emptyDash(object.SIM1)).QWidget)
	if strings.TrimSpace(object.SIM1) != "" && sendSMS != nil {
		btn := qt.NewQPushButton3("SMS")
		btn.SetToolTip("Надіслати SMS через Omnicell на SIM 1")
		btn.OnClicked(func() {
			sendSMS(object, object.SIM1)
		})
		sim1Layout.AddWidget(btn.QWidget)
	}
	sim1Operator := simoperator.Detect(object.SIM1)
	if sim1Operator == simoperator.Vodafone && vfService != nil {
		btn := qt.NewQPushButton3("Vodafone M2M")
		btn.OnClicked(func() {
			ShowVodafoneSIMDialog(dialog.QWidget, vfService, object.SIM1, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim1Layout.AddWidget(btn.QWidget)
	} else if sim1Operator == simoperator.Kyivstar && ksService != nil {
		btn := qt.NewQPushButton3("Kyivstar M2M")
		btn.OnClicked(func() {
			ShowKyivstarSIMDialog(dialog.QWidget, ksService, object.SIM1, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim1Layout.AddWidget(btn.QWidget)
	}
	sim1Layout.AddStretch()
	sim1Widget.SetLayout(sim1Layout.QLayout)
	form.AddRow3("SIM 1", sim1Widget)

	// SIM 2 Layout with Action Buttons if supported
	sim2Widget := qt.NewQWidget2()
	sim2Layout := qt.NewQHBoxLayout(sim2Widget)
	sim2Layout.SetContentsMargins(0, 0, 0, 0)
	sim2Layout.AddWidget(qt.NewQLabel3(emptyDash(object.SIM2)).QWidget)
	if strings.TrimSpace(object.SIM2) != "" && sendSMS != nil {
		btn := qt.NewQPushButton3("SMS")
		btn.SetToolTip("Надіслати SMS через Omnicell на SIM 2")
		btn.OnClicked(func() {
			sendSMS(object, object.SIM2)
		})
		sim2Layout.AddWidget(btn.QWidget)
	}
	sim2Operator := simoperator.Detect(object.SIM2)
	if sim2Operator == simoperator.Vodafone && vfService != nil {
		btn := qt.NewQPushButton3("Vodafone M2M")
		btn.OnClicked(func() {
			ShowVodafoneSIMDialog(dialog.QWidget, vfService, object.SIM2, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim2Layout.AddWidget(btn.QWidget)
	} else if sim2Operator == simoperator.Kyivstar && ksService != nil {
		btn := qt.NewQPushButton3("Kyivstar M2M")
		btn.OnClicked(func() {
			ShowKyivstarSIMDialog(dialog.QWidget, ksService, object.SIM2, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim2Layout.AddWidget(btn.QWidget)
	}
	sim2Layout.AddStretch()
	sim2Widget.SetLayout(sim2Layout.QLayout)
	form.AddRow3("SIM 2", sim2Widget)

	usage := qt.NewQTextEdit3(strings.TrimSpace(usageText))
	usage.SetReadOnly(true)
	usage.SetMinimumHeight(180)
	if strings.TrimSpace(usageText) == "" {
		usage.SetPlainText("Збігів використання SIM-номерів не знайдено.")
	}

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok)
	buttons.OnAccepted(dialog.Accept)

	layout.AddLayout(form.QLayout)
	layout.AddWidget(usage.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	dialog.Exec()
}

func ShowSIMSMSDialog(parent *qt.QWidget, object models.Object, phone string, cfg config.OmnicellConfig) ([]simcommands.SMSCommand, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("SMS на SIM")
	dialog.Resize(720, 680)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Об'єкт", qt.NewQLabel3(fmt.Sprintf("<b>%s</b> №%s", htmlEscape(strings.TrimSpace(object.Name)), viewmodels.ObjectDisplayNumber(object))).QWidget)
	form.AddRow3("Номер", qt.NewQLabel3(htmlEscape(strings.TrimSpace(phone))).QWidget)

	profile := qt.NewQComboBox2()
	profile.AddItems([]string{simcommands.ProfileMCAGSM4, simcommands.ProfileMCAGSM, simcommands.ProfileFreeSMS})
	setComboTextFallback(profile, cfg.MCADefaultMessageProfile, simcommands.ProfileMCAGSM4)
	form.AddRow3("Тип", profile.QWidget)

	objectNumber := spinBox(1, 999999)
	objectNumber.SetValue(viewmodels.NumericObjectDisplayNumber(object))
	form.AddRow3("Об'єктовий номер", objectNumber.QWidget)
	hiddenNumber := spinBox(0, 999999)
	hiddenNumber.SetValue(int(object.GSMHiddenN))
	form.AddRow3("Прихований номер", hiddenNumber.QWidget)

	primaryAPN := newLineEdit(cfg.MCAPrimaryAPN)
	reserveAPN := newLineEdit(cfg.MCAReserveAPN)
	primaryIP := newLineEdit(cfg.MCAPrimaryIP)
	reserveIP := newLineEdit(cfg.MCAReserveIP)
	primaryModulePort := spinBox(1, 9999)
	reserveModulePort := spinBox(1, 9999)
	primaryReceiverPort := spinBox(1, 9999)
	reserveReceiverPort := spinBox(1, 9999)
	primaryInterval := spinBox(1, 240)
	reserveInterval := spinBox(1, 240)
	inputConfirm := qt.NewQCheckBox3("Підтвердження")

	primaryModulePort.SetValue(cfg.MCAPrimaryModulePort)
	reserveModulePort.SetValue(cfg.MCAReserveModulePort)
	primaryReceiverPort.SetValue(cfg.MCAPrimaryReceiverPort)
	reserveReceiverPort.SetValue(cfg.MCAReserveReceiverPort)
	primaryInterval.SetValue(cfg.MCAPrimaryTestInterval)
	reserveInterval.SetValue(cfg.MCAReserveTestInterval)
	inputConfirm.SetChecked(cfg.MCAInput1ConfirmMode)

	form.AddRow3("APN основний", primaryAPN.QWidget)
	form.AddRow3("APN резервний", reserveAPN.QWidget)
	form.AddRow3("IP основний", primaryIP.QWidget)
	form.AddRow3("IP резервний", reserveIP.QWidget)
	form.AddRow3("Порт модуля основний", primaryModulePort.QWidget)
	form.AddRow3("Порт модуля резервний", reserveModulePort.QWidget)
	form.AddRow3("Порт ПЦПС основний", primaryReceiverPort.QWidget)
	form.AddRow3("Порт ПЦПС резервний", reserveReceiverPort.QWidget)
	form.AddRow3("Тест основний, хв", primaryInterval.QWidget)
	form.AddRow3("Тест резервний, хв", reserveInterval.QWidget)
	form.AddRow3("Вхід 1", inputConfirm.QWidget)

	message := qt.NewQTextEdit2()
	message.SetMinimumHeight(160)
	form.AddRow3("Повідомлення", message.QWidget)

	buildConfig := func() simcommands.MCAGSM4Config {
		return simcommands.MCAGSM4Config{
			ObjectNumber:        objectNumber.Value(),
			HiddenNumber:        hiddenNumber.Value(),
			PrimaryAPN:          primaryAPN.Text(),
			ReserveAPN:          reserveAPN.Text(),
			PrimaryIP:           primaryIP.Text(),
			ReserveIP:           reserveIP.Text(),
			PrimaryModulePort:   primaryModulePort.Value(),
			ReserveModulePort:   reserveModulePort.Value(),
			PrimaryReceiverPort: primaryReceiverPort.Value(),
			ReserveReceiverPort: reserveReceiverPort.Value(),
			PrimaryTestInterval: primaryInterval.Value(),
			ReserveTestInterval: reserveInterval.Value(),
			Input1ConfirmMode:   inputConfirm.IsChecked(),
		}
	}
	buildGSMConfig := func() simcommands.MCAGSM4Config {
		gsmCfg := simcommands.DefaultMCAGSMConfig()
		gsmCfg.ObjectNumber = objectNumber.Value()
		gsmCfg.HiddenNumber = hiddenNumber.Value()
		gsmCfg.PrimaryAPN = primaryAPN.Text()
		gsmCfg.ReserveAPN = reserveAPN.Text()
		gsmCfg.PrimaryIP = primaryIP.Text()
		gsmCfg.ReserveIP = reserveIP.Text()
		return gsmCfg
	}
	renderPreview := func(commands []simcommands.SMSCommand) string {
		lines := make([]string, 0, len(commands)*2)
		for _, command := range commands {
			lines = append(lines, command.Title+":", command.Text)
		}
		return strings.Join(lines, "\n")
	}
	updatePreview := func() {
		switch profile.CurrentText() {
		case simcommands.ProfileMCAGSM4:
			message.SetReadOnly(true)
			commands, err := simcommands.BuildMCAGSM4Messages(buildConfig())
			if err != nil {
				message.SetPlainText("Помилка шаблону: " + err.Error())
				return
			}
			message.SetPlainText(renderPreview(commands))
		case simcommands.ProfileMCAGSM:
			message.SetReadOnly(true)
			commands, err := simcommands.BuildMCAGSMMessages(buildGSMConfig())
			if err != nil {
				message.SetPlainText("Помилка шаблону: " + err.Error())
				return
			}
			message.SetPlainText(renderPreview(commands))
		default:
			message.SetReadOnly(false)
			if strings.TrimSpace(message.ToPlainText()) == "" {
				message.SetPlaceholderText("Текст SMS")
			}
		}
	}
	profile.OnCurrentTextChanged(func(string) { updatePreview() })
	for _, edit := range []*qt.QLineEdit{primaryAPN, reserveAPN, primaryIP, reserveIP} {
		edit.OnTextChanged(func(string) { updatePreview() })
	}
	for _, spin := range []*qt.QSpinBox{objectNumber, hiddenNumber, primaryModulePort, reserveModulePort, primaryReceiverPort, reserveReceiverPort, primaryInterval, reserveInterval} {
		spin.OnValueChanged(func(int) { updatePreview() })
	}
	inputConfirm.OnToggled(func(bool) { updatePreview() })
	updatePreview()

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)

	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return nil, false
	}
	switch profile.CurrentText() {
	case simcommands.ProfileMCAGSM4:
		commands, err := simcommands.BuildMCAGSM4Messages(buildConfig())
		if err != nil {
			qt.QMessageBox_Critical(parent, "SMS", err.Error())
			return nil, false
		}
		return commands, true
	case simcommands.ProfileMCAGSM:
		commands, err := simcommands.BuildMCAGSMMessages(buildGSMConfig())
		if err != nil {
			qt.QMessageBox_Critical(parent, "SMS", err.Error())
			return nil, false
		}
		return commands, true
	}
	text := strings.TrimSpace(message.ToPlainText())
	if text == "" {
		qt.QMessageBox_Information(parent, "SMS", "Текст повідомлення порожній.")
		return nil, false
	}
	return []simcommands.SMSCommand{{Title: profile.CurrentText(), Text: text}}, true
}

func newLineEdit(value string) *qt.QLineEdit {
	edit := qt.NewQLineEdit3(strings.TrimSpace(value))
	edit.SetClearButtonEnabled(true)
	return edit
}

func newSpinBox(value int, min int, max int) *qt.QSpinBox {
	spin := qt.NewQSpinBox2()
	spin.SetRange(min, max)
	spin.SetValue(value)
	return spin
}

func fillComboBox(combo *qt.QComboBox, options []string) {
	if combo == nil {
		return
	}
	combo.Clear()
	combo.AddItems(options)
	if combo.Count() > 0 && combo.CurrentIndex() < 0 {
		combo.SetCurrentIndex(0)
	}
}

func stackedWidgets(widgets ...*qt.QWidget) *qt.QWidget {
	container := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(container)
	layout.SetContentsMargins(0, 0, 0, 0)
	for _, widget := range widgets {
		if widget != nil {
			layout.AddWidget(widget)
		}
	}
	container.SetLayout(layout.QLayout)
	return container
}

func horizontalWidgets(widgets ...*qt.QWidget) *qt.QWidget {
	container := qt.NewQWidget2()
	layout := qt.NewQHBoxLayout2()
	layout.SetContentsMargins(0, 0, 0, 0)
	for _, widget := range widgets {
		if widget != nil {
			layout.AddWidget(widget)
		}
	}
	container.SetLayout(layout.QLayout)
	return container
}

func showObjectDatePicker(parent *qt.QWidget, dateVM *viewmodels.ObjectDateFieldViewModel, raw string) (string, bool) {
	if dateVM == nil {
		dateVM = viewmodels.NewObjectDateFieldViewModel()
	}
	initial := dateVM.ResolvePickerInitial(raw, time.Now())
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Вибір дати")

	calendar := qt.NewQCalendarWidget2()
	calendar.SetSelectedDate(*qt.NewQDate2(initial.Year(), int(initial.Month()), initial.Day()))
	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)
	calendar.OnActivated(func(_ qt.QDate) { dialog.Accept() })

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(calendar.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return "", false
	}
	selected := calendar.SelectedDate()
	if selected == nil {
		return "", false
	}
	value := time.Date(selected.Year(), time.Month(selected.Month()), selected.Day(), 0, 0, 0, 0, time.Local)
	return dateVM.FormatForDisplay(value), true
}

func buildPlaceholderWidget(text string) *qt.QWidget {
	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)
	label := qt.NewQLabel3(strings.TrimSpace(text))
	label.SetWordWrap(true)
	layout.AddWidget(label.QWidget)
	layout.AddStretch()
	widget.SetLayout(layout.QLayout)
	return widget
}

func emptyDash(value string) string {
	if text := strings.TrimSpace(value); text != "" {
		return text
	}
	return "-"
}

func ObjectSIMUsageText(lookup viewmodels.SIMPhoneUsageLookup, object models.Object) string {
	exclude := int64(object.ID)
	vm := viewmodels.NewSIMPhoneUsageViewModel()
	parts := make([]string, 0, 2)
	for _, sim := range []struct {
		label string
		phone string
	}{
		{label: "SIM 1", phone: object.SIM1},
		{label: "SIM 2", phone: object.SIM2},
	} {
		phone := strings.TrimSpace(sim.phone)
		if phone == "" {
			parts = append(parts, sim.label+": номер не задано")
			continue
		}
		text := vm.ResolveUsageText(lookup, phone, &exclude)
		if strings.TrimSpace(text) == "" {
			text = "номер вільний"
		}
		parts = append(parts, sim.label+" "+phone+": "+text)
	}
	return strings.Join(parts, "\n")
}

func contactPositionText(contact models.Contact) string {
	position := strings.TrimSpace(contact.Position)
	switch strings.ToUpper(position) {
	case "IN_CHARGE":
		return "Відповідальна особа"
	case "OWNER":
		return "Власник"
	case "ADMIN", "MANAGER":
		return "Адміністратор"
	case "USER":
		return "Користувач"
	}
	if position != "" {
		if _, err := strconv.Atoi(position); err == nil {
			return "Відповідальна особа " + position
		}
		return position
	}
	if contact.Priority > 0 {
		return fmt.Sprintf("Відповідальна особа %d", contact.Priority)
	}
	return "-"
}
