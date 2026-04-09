// Package ui - робоча область з деталями об'єкта
package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/simoperator"
	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

const workAreaJournalTabIndex = 3

// WorkAreaPanel - структура робочої області
type WorkAreaPanel struct {
	Container       *fyne.Container
	Data            contracts.WorkAreaProvider
	ViewModel       *viewmodels.WorkAreaViewModel
	CaseHistoryVM   *viewmodels.WorkAreaCaseHistoryViewModel
	HeaderVM        *viewmodels.WorkAreaHeaderViewModel
	DeviceVM        *viewmodels.WorkAreaDeviceViewModel
	GroupSectionsVM *viewmodels.WorkAreaGroupSectionsViewModel
	ExportVM        *viewmodels.WorkAreaExportViewModel
	DeviceStateVM   *viewmodels.WorkAreaDeviceStateViewModel
	ExternalVM      *viewmodels.WorkAreaExternalStateViewModel
	CurrentObject   *models.Object
	Window          fyne.Window
	ZonesData       binding.UntypedList
	ContactsData    binding.UntypedList
	EventsData      binding.UntypedList

	// Стан завантаження
	Zones     []models.Zone
	Contacts  []models.Contact
	Events    []models.Event
	IsLoading bool

	eventsLoadedObjectID  int
	eventsLoadingObjectID int

	// UI елементи
	HeaderName    *widget.Label
	HeaderAddress *widget.Label
	HeaderStatus  *canvas.Text
	ExportPDFBtn  *widget.Button
	ExportXLSXBtn *widget.Button
	CopyExcelBtn  *widget.Button
	Tabs          *container.AppTabs

	// Лейбли інформації про прилад
	DeviceTypeLabel      *widget.Label
	PanelMarkLabel       *widget.Label // Added PanelMarkLabel
	GroupsLabel          *widget.Label
	GSMLabel             *widget.Label
	PowerLabel           *widget.Label
	SIMLabel             *widget.Label
	SIM1Label            *widget.Label
	SIM2Label            *widget.Label
	AutoTestLabel        *widget.Label
	GuardLabel           *widget.Label
	ChanLabel            *widget.Label
	PhoneLabel           *widget.Label
	AkbLabel             *widget.Label
	TestControlLabel     *widget.Label
	SignalLabel          *widget.Label
	LastTestLabel        *widget.Label
	LastTestTimeLabel    *widget.Label
	LastMessageTimeLabel *widget.Label
	SummaryStateCaption  *canvas.Text
	SummaryStatePanel    *canvas.Rectangle
	SummarySectionPanels []*canvas.Rectangle
	TestLogsBtn          *widget.Button
	Notes1Label          *widget.Label
	Location1Label       *widget.Label

	// Кнопки копіювання
	CopyNameBtn     *widget.Button
	CopyAddressBtn  *widget.Button
	CopySimBtn      *widget.Button
	CopySIM1Btn     *widget.Button
	CopySIM2Btn     *widget.Button
	VodafoneSIM1Btn *widget.Button
	VodafoneSIM2Btn *widget.Button
	CopyPhonesBtn   *widget.Button
	CopyNotesBtn    *widget.Button
	CopyLocationBtn *widget.Button

	// Таблиці
	ZonesTable           *widget.Table
	ContactsList         *widget.List
	EventsList           *widget.List
	CaseHistoryAccordion *widget.Accordion
	CaseHistorySection   *fyne.Container
	ZonesContent         *fyne.Container
	ContactsContent      *fyne.Container
	ZonesFlatContent     fyne.CanvasObject
	ContactsFlatContent  fyne.CanvasObject
	EventsWidthGuide     *canvas.Rectangle
}

// NewWorkAreaPanel створює робочу область
func NewWorkAreaPanel(provider contracts.WorkAreaProvider, window fyne.Window) *WorkAreaPanel {
	panel := &WorkAreaPanel{
		Data:            provider,
		ViewModel:       viewmodels.NewWorkAreaViewModel(),
		CaseHistoryVM:   viewmodels.NewWorkAreaCaseHistoryViewModel(),
		HeaderVM:        viewmodels.NewWorkAreaHeaderViewModel(),
		DeviceVM:        viewmodels.NewWorkAreaDeviceViewModel(),
		GroupSectionsVM: viewmodels.NewWorkAreaGroupSectionsViewModel(),
		ExportVM:        viewmodels.NewWorkAreaExportViewModel(),
		DeviceStateVM:   viewmodels.NewWorkAreaDeviceStateViewModel(),
		ExternalVM:      viewmodels.NewWorkAreaExternalStateViewModel(),
		Window:          window,
		ZonesData:       binding.NewUntypedList(),
		ContactsData:    binding.NewUntypedList(),
		EventsData:      binding.NewUntypedList(),
	}

	// Шапка
	themeSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)

	// Назва об'єкта: використовуємо Label з перенесенням рядків,
	// щоб довгі назви (до ~200 символів) коректно відображались у межах правої вкладки.
	panel.HeaderName = widget.NewLabelWithData(panel.HeaderVM.HeaderNameBinding())
	panel.HeaderName.TextStyle = fyne.TextStyle{Bold: true}
	panel.HeaderName.Wrapping = fyne.TextWrapWord

	// Адреса об'єкта також може бути довгою — вмикаємо перенесення.
	panel.HeaderAddress = widget.NewLabelWithData(panel.HeaderVM.HeaderAddressBinding())
	panel.HeaderAddress.Wrapping = fyne.TextWrapWord
	panel.HeaderStatus = canvas.NewText("", appTheme.ColorNormal)
	panel.HeaderStatus.TextSize = themeSize + 1
	panel.HeaderStatus.TextStyle = fyne.TextStyle{Bold: true}

	panel.CopyNameBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	panel.CopyAddressBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	panel.initExportButtons()
	panel.CopyExcelBtn = widget.NewButton("Копіювати рядок для Excel", func() {
		if panel.CurrentObject == nil {
			ShowToast(panel.Window, "Спочатку оберіть об'єкт")
			return
		}

		row := panel.ExportVM.BuildExcelRowTSV(*panel.CurrentObject, panel.Contacts)
		panel.Window.Clipboard().SetContent(row)
		ShowToast(panel.Window, "Рядок для Excel скопійовано")
	})
	panel.CopyExcelBtn.Disable()

	header := container.NewVBox(
		container.NewBorder(nil, nil, nil, panel.CopyNameBtn, panel.HeaderName),
		container.NewBorder(nil, nil, nil, panel.CopyAddressBtn, panel.HeaderAddress),
		panel.HeaderStatus,
		container.NewHBox(
			widget.NewLabel("Експорт:"),
			panel.ExportPDFBtn,
			panel.ExportXLSXBtn,
			panel.CopyExcelBtn,
		),
		widget.NewSeparator(),
	)

	// Вкладки
	panel.Tabs = container.NewAppTabs(
		container.NewTabItem("📊 Стан", panel.createSummaryTab()),
		container.NewTabItem("🔌 Зони", panel.createZonesTab()),
		container.NewTabItem("👥 Відповідальні", panel.createContactsTab()),
		container.NewTabItem("📜 Журнал", panel.createEventsTab()),
	)
	panel.Tabs.OnSelected = func(item *container.TabItem) {
		if item == nil {
			return
		}
		panel.ensureCurrentObjectEventsLoaded()
	}

	panel.Container = container.NewBorder(
		header,
		nil, nil, nil,
		panel.Tabs,
	)

	return panel
}

func (w *WorkAreaPanel) initExportButtons() {
	w.ExportPDFBtn = widget.NewButton("PDF", func() {
		w.exportSelectedObject("pdf")
	})
	w.ExportXLSXBtn = widget.NewButton("XLSX", func() {
		w.exportSelectedObject("xlsx")
	})
	w.ExportPDFBtn.Disable()
	w.ExportXLSXBtn.Disable()
}

func (w *WorkAreaPanel) createSummaryTab() fyne.CanvasObject {
	w.DeviceTypeLabel = widget.NewLabelWithData(w.DeviceStateVM.DeviceTypeBinding())
	w.PanelMarkLabel = widget.NewLabelWithData(w.DeviceStateVM.PanelMarkBinding())
	w.GroupsLabel = widget.NewLabelWithData(w.DeviceStateVM.GroupsBinding())
	w.GroupsLabel.Wrapping = fyne.TextWrapWord
	w.PowerLabel = widget.NewLabelWithData(w.DeviceStateVM.PowerBinding())
	w.SIMLabel = widget.NewLabelWithData(w.DeviceStateVM.SIMBinding())
	w.SIM1Label = widget.NewLabelWithData(w.DeviceStateVM.SIM1Binding())
	w.SIM2Label = widget.NewLabelWithData(w.DeviceStateVM.SIM2Binding())
	w.AutoTestLabel = widget.NewLabelWithData(w.DeviceStateVM.AutoTestBinding())
	w.GuardLabel = widget.NewLabelWithData(w.DeviceStateVM.GuardBinding())
	w.GuardLabel.TextStyle = fyne.TextStyle{Bold: true}
	w.CopySimBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	w.CopySIM1Btn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	w.CopySIM2Btn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	w.VodafoneSIM1Btn = widget.NewButton("Vodafone", nil)
	w.VodafoneSIM2Btn = widget.NewButton("Vodafone", nil)
	w.ChanLabel = widget.NewLabelWithData(w.DeviceStateVM.ChannelBinding())
	w.PhoneLabel = widget.NewLabelWithData(w.DeviceStateVM.PhoneBinding())
	w.CopyPhonesBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	w.AkbLabel = widget.NewLabelWithData(w.DeviceStateVM.AkbBinding())
	w.TestControlLabel = widget.NewLabelWithData(w.DeviceStateVM.TestControlBinding())
	w.SignalLabel = widget.NewLabelWithData(w.ExternalVM.SignalBinding())
	w.LastTestLabel = widget.NewLabelWithData(w.ExternalVM.LastTestBinding())
	w.LastTestTimeLabel = widget.NewLabelWithData(w.ExternalVM.LastTestTimeBinding())
	w.LastMessageTimeLabel = widget.NewLabelWithData(w.ExternalVM.LastMessageTimeBinding())
	w.TestLogsBtn = widget.NewButtonWithIcon("Тестові повідомлення", fyneTheme.HistoryIcon(), nil)

	w.Notes1Label = widget.NewLabelWithData(w.DeviceStateVM.NotesBinding())
	w.Notes1Label.Wrapping = fyne.TextWrapWord
	w.CopyNotesBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)

	w.Location1Label = widget.NewLabelWithData(w.DeviceStateVM.LocationBinding())
	w.Location1Label.Wrapping = fyne.TextWrapWord
	w.CopyLocationBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)

	for _, label := range []*widget.Label{
		w.DeviceTypeLabel,
		w.PanelMarkLabel,
		w.GroupsLabel,
		w.PowerLabel,
		w.SIMLabel,
		w.SIM1Label,
		w.SIM2Label,
		w.AutoTestLabel,
		w.GuardLabel,
		w.ChanLabel,
		w.PhoneLabel,
		w.AkbLabel,
		w.TestControlLabel,
		w.SignalLabel,
		w.LastTestLabel,
		w.LastTestTimeLabel,
		w.LastMessageTimeLabel,
	} {
		label.Wrapping = fyne.TextWrapWord
	}

	w.SummarySectionPanels = nil

	w.SummaryStateCaption = canvas.NewText("Оперативний стан", appTheme.ColorSectionTitle)
	w.SummaryStateCaption.TextSize = fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText) - 1

	w.SummaryStatePanel = canvas.NewRectangle(color.Transparent)
	w.SummaryStatePanel.CornerRadius = 14
	w.SummaryStatePanel.StrokeWidth = 1

	statusSummary := container.NewStack(
		w.SummaryStatePanel,
		container.NewPadded(
			container.NewAdaptiveGrid(
				2,
				container.NewVBox(
					w.SummaryStateCaption,
					w.GuardLabel,
				),
				container.NewVBox(
					w.PowerLabel,
					w.SignalLabel,
					w.LastMessageTimeLabel,
				),
			),
		),
	)

	groupsScroll := container.NewScroll(w.GroupsLabel)
	groupsScroll.SetMinSize(fyne.NewSize(0, 120))

	notesScroll := container.NewScroll(w.Notes1Label)
	notesScroll.SetMinSize(fyne.NewSize(0, 84))

	locationScroll := container.NewScroll(w.Location1Label)
	locationScroll.SetMinSize(fyne.NewSize(0, 68))

	deviceSection, devicePanel := makeWorkAreaSummarySection(
		"Прилад",
		w.DeviceTypeLabel,
		w.PanelMarkLabel,
		w.ChanLabel,
		w.AkbLabel,
	)

	communicationSection, communicationPanel := makeWorkAreaSummarySection(
		"Зв'язок",
		makeWorkAreaActionRow(w.SIMLabel, w.CopySimBtn),
		makeWorkAreaActionRow(w.SIM1Label, w.VodafoneSIM1Btn, w.CopySIM1Btn),
		makeWorkAreaActionRow(w.SIM2Label, w.VodafoneSIM2Btn, w.CopySIM2Btn),
		makeWorkAreaActionRow(w.PhoneLabel, w.CopyPhonesBtn),
	)

	controlSection, controlPanel := makeWorkAreaSummarySection(
		"Контроль і активність",
		w.AutoTestLabel,
		w.TestControlLabel,
		w.LastTestLabel,
		w.LastTestTimeLabel,
		w.TestLogsBtn,
	)

	groupsSection, groupsPanel := makeWorkAreaSummarySection("Групи", groupsScroll)
	locationSection, locationPanel := makeWorkAreaSummarySection(
		"Розташування",
		makeWorkAreaActionRow(locationScroll, w.CopyLocationBtn),
	)
	notesSection, notesPanel := makeWorkAreaSummarySection(
		"Примітки",
		makeWorkAreaActionRow(notesScroll, w.CopyNotesBtn),
	)

	w.SummarySectionPanels = append(
		w.SummarySectionPanels,
		devicePanel,
		communicationPanel,
		controlPanel,
		groupsPanel,
		locationPanel,
		notesPanel,
	)
	w.updateSummaryThemeColors()

	deviceInfo := container.NewVBox(
		statusSummary,
		container.NewAdaptiveGrid(2, deviceSection, communicationSection),
		container.NewAdaptiveGrid(2, controlSection, groupsSection),
		container.NewAdaptiveGrid(2, locationSection, notesSection),
	)

	return container.NewScroll(
		container.New(
			&summaryTabContentLayout{},
			container.NewPadded(deviceInfo),
		),
	)
}

func makeWorkAreaSummarySection(title string, content ...fyne.CanvasObject) (fyne.CanvasObject, *canvas.Rectangle) {
	themeVariant := fyne.CurrentApp().Settings().ThemeVariant()
	themeColors := fyne.CurrentApp().Settings().Theme()

	bg := canvas.NewRectangle(themeColors.Color(fyneTheme.ColorNameInputBackground, themeVariant))
	bg.CornerRadius = 12
	bg.StrokeColor = themeColors.Color(fyneTheme.ColorNameSeparator, themeVariant)
	bg.StrokeWidth = 1

	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	items := make([]fyne.CanvasObject, 0, len(content)+2)
	items = append(items, titleLabel, widget.NewSeparator())
	items = append(items, content...)

	return container.NewStack(
		bg,
		container.NewPadded(container.NewVBox(items...)),
	), bg
}

func makeWorkAreaActionRow(content fyne.CanvasObject, actions ...fyne.CanvasObject) fyne.CanvasObject {
	visibleActions := make([]fyne.CanvasObject, 0, len(actions))
	for _, action := range actions {
		if action != nil {
			visibleActions = append(visibleActions, action)
		}
	}
	if len(visibleActions) == 0 {
		return content
	}
	return container.NewBorder(nil, nil, nil, container.NewHBox(visibleActions...), content)
}

func (w *WorkAreaPanel) updateSummaryStatusAccent(status models.ObjectStatus) {
	if w == nil || w.SummaryStatePanel == nil {
		return
	}

	base := color.NRGBAModel.Convert(GetStatusColor(status)).(color.NRGBA)
	fillAlpha := uint8(32)
	strokeAlpha := uint8(96)
	if IsDarkMode() {
		fillAlpha = 52
		strokeAlpha = 128
	}

	w.SummaryStatePanel.FillColor = color.NRGBA{R: base.R, G: base.G, B: base.B, A: fillAlpha}
	w.SummaryStatePanel.StrokeColor = color.NRGBA{R: base.R, G: base.G, B: base.B, A: strokeAlpha}
	w.SummaryStatePanel.Refresh()
}

func (w *WorkAreaPanel) updateSummaryThemeColors() {
	if w == nil {
		return
	}

	themeVariant := fyne.CurrentApp().Settings().ThemeVariant()
	themeColors := fyne.CurrentApp().Settings().Theme()
	fillColor := themeColors.Color(fyneTheme.ColorNameInputBackground, themeVariant)
	strokeColor := themeColors.Color(fyneTheme.ColorNameSeparator, themeVariant)
	captionColor := themeColors.Color(fyneTheme.ColorNamePlaceHolder, themeVariant)

	if w.SummaryStateCaption != nil {
		w.SummaryStateCaption.Color = captionColor
		w.SummaryStateCaption.Refresh()
	}

	for _, panel := range w.SummarySectionPanels {
		if panel == nil {
			continue
		}
		panel.FillColor = fillColor
		panel.StrokeColor = strokeColor
		panel.Refresh()
	}
}

type summaryTabContentLayout struct{}

func (l *summaryTabContentLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		minSize := obj.MinSize()
		height := minSize.Height
		if size.Height > height {
			height = size.Height
		}
		obj.Resize(fyne.NewSize(size.Width, height))
		obj.Move(fyne.NewPos(0, 0))
	}
}

func (l *summaryTabContentLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}

func (w *WorkAreaPanel) createZonesTab() fyne.CanvasObject {
	w.ZonesTable = widget.NewTable(
		func() (int, int) {
			if w.ZonesData != nil {
				return w.ZonesData.Length(), 5
			}
			return 0, 5
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Data")
			label.Wrapping = fyne.TextWrapWord
			btn := widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
			btn.Hide()
			return container.NewBorder(nil, nil, nil, btn, label)
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			container := obj.(*fyne.Container)
			label := container.Objects[0].(*widget.Label)
			btn := container.Objects[1].(*widget.Button)

			zone, ok := w.zoneByRow(id.Row)
			if !ok {
				label.SetText("")
				btn.Hide()
				return
			}

			label.Truncation = fyne.TextTruncateOff
			if id.Col == 1 || id.Col == 2 {
				label.Wrapping = fyne.TextWrapWord
			} else {
				label.Wrapping = fyne.TextWrapOff
			}

			var text string
			switch id.Col {
			case 0:
				text = "№" + strconv.Itoa(zone.Number)
			case 1:
				text = zone.Name
			case 2:
				text = zone.SensorType
			case 3:
				text = zone.GetStatusDisplay()
			case 4:
				text = ""
				label.Hide()
				btn.Show()
				btn.OnTapped = func() {
					copyText := fmt.Sprintf("Зона %d: %s (%s)", zone.Number, zone.Name, w.CurrentObject.Name)
					w.Window.Clipboard().SetContent(copyText)
					ShowToast(w.Window, "Скопійовано зону")
				}
				return
			}

			label.SetText(text)
			label.Show()
			if id.Col == 1 {
				btn.Show()
				btn.OnTapped = func() {
					w.Window.Clipboard().SetContent(zone.Name)
					ShowToast(w.Window, "Скопійовано назву зони")
				}
			} else {
				btn.Hide()
			}
		},
	)

	w.ZonesTable.SetColumnWidth(0, zoneTableNumberColumnWidth)
	w.ZonesTable.SetColumnWidth(1, zoneTableNameDefaultWidth)
	w.ZonesTable.SetColumnWidth(2, zoneTableTypeColumnWidth)
	w.ZonesTable.SetColumnWidth(3, zoneTableStatusColumnWidth)
	w.ZonesTable.SetColumnWidth(4, zoneTableCopyColumnWidth)

	w.ZonesFlatContent = container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		container.New(&zonesTableLayout{
			table: w.ZonesTable,
			zones: func() []models.Zone { return w.Zones },
		}, w.ZonesTable),
	)
	w.ZonesContent = container.NewMax(w.ZonesFlatContent)
	w.refreshZonesTableLayout()

	return w.ZonesContent
}

type zonesTableLayout struct {
	table         *widget.Table
	zones         func() []models.Zone
	lastNameWidth float32
}

func (l *zonesTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	available := zoneTableNameColumnWidth(size.Width)

	if l.lastNameWidth != available {
		l.table.SetColumnWidth(1, available)
		if l.zones != nil {
			updateZoneTableRowHeights(l.table, l.zones(), available)
		}
		l.lastNameWidth = available
		l.table.Refresh()
	}

	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func (l *zonesTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(450, 200)
}

func (w *WorkAreaPanel) buildGroupedZonesAccordion(
	sections []viewmodels.WorkAreaGroupSection,
) fyne.CanvasObject {
	items := make([]*widget.AccordionItem, 0, len(sections))
	for _, section := range sections {
		title := w.GroupSectionsVM.FormatSectionTitle(section.Group)
		items = append(items, widget.NewAccordionItem(title, w.buildGroupedZonesSection(section)))
	}
	accordion := widget.NewAccordion(items...)
	if len(items) > 0 {
		accordion.Open(0)
	}
	return container.NewScroll(accordion)
}

func (w *WorkAreaPanel) buildGroupedZonesSection(
	section viewmodels.WorkAreaGroupSection,
) fyne.CanvasObject {
	if len(section.Zones) == 0 {
		return container.NewPadded(widget.NewLabel("Немає зон у цій групі"))
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(section.Zones), 5
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Data")
			label.Truncation = fyne.TextTruncateOff
			label.Wrapping = fyne.TextWrapWord
			btn := widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
			btn.Hide()
			return container.NewBorder(nil, nil, nil, btn, label)
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			cell := obj.(*fyne.Container)
			label := cell.Objects[0].(*widget.Label)
			btn := cell.Objects[1].(*widget.Button)
			zone := section.Zones[id.Row]
			label.Truncation = fyne.TextTruncateOff
			if id.Col == 1 || id.Col == 2 {
				label.Wrapping = fyne.TextWrapWord
			} else {
				label.Wrapping = fyne.TextWrapOff
			}

			var text string
			switch id.Col {
			case 0:
				text = "№" + strconv.Itoa(zone.Number)
			case 1:
				text = zone.Name
			case 2:
				text = zone.SensorType
			case 3:
				text = zone.GetStatusDisplay()
			case 4:
				text = ""
				label.Hide()
				btn.Show()
				btn.OnTapped = func() {
					objectName := ""
					if w.CurrentObject != nil {
						objectName = strings.TrimSpace(w.CurrentObject.Name)
					}
					copyText := fmt.Sprintf("Зона %d: %s", zone.Number, zone.Name)
					if objectName != "" {
						copyText += " (" + objectName + ")"
					}
					w.Window.Clipboard().SetContent(copyText)
					ShowToast(w.Window, "Скопійовано зону")
				}
				return
			default:
				text = ""
			}

			label.SetText(text)
			label.Show()
			if id.Col == 1 {
				btn.Show()
				btn.OnTapped = func() {
					w.Window.Clipboard().SetContent(zone.Name)
					ShowToast(w.Window, "Скопійовано назву зони")
				}
			} else {
				btn.Hide()
			}
		},
	)
	table.SetColumnWidth(0, zoneTableNumberColumnWidth)
	table.SetColumnWidth(1, groupedZoneTableNameWidth)
	table.SetColumnWidth(2, groupedZoneTableTypeWidth)
	table.SetColumnWidth(3, groupedZoneTableStatusWidth)
	table.SetColumnWidth(4, zoneTableCopyColumnWidth)
	updateGroupedZoneTableRowHeights(table, section.Zones)

	return container.NewPadded(table)
}

func (w *WorkAreaPanel) createContactsTab() fyne.CanvasObject {
	w.ContactsList = widget.NewListWithData(
		w.ContactsData,
		func() fyne.CanvasObject {
			nameLabel := widget.NewLabel("Name")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			phoneLabel := widget.NewLabel("Phone")
			copyNameBtn := widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
			copyPhoneBtn := widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)

			return container.NewVBox(
				container.NewBorder(nil, nil, nil, copyNameBtn, nameLabel),
				container.NewBorder(nil, nil, nil, copyPhoneBtn, phoneLabel),
				widget.NewSeparator(),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			data, ok := item.(binding.Untyped)
			if !ok {
				return
			}
			value, err := data.Get()
			if err != nil {
				return
			}
			contact, ok := value.(models.Contact)
			if !ok {
				return
			}
			vbox := obj.(*fyne.Container)

			nameRow := vbox.Objects[0].(*fyne.Container)
			nameLabel := nameRow.Objects[0].(*widget.Label)
			nameBtn := nameRow.Objects[1].(*widget.Button)
			nameLabel.SetText(fmt.Sprintf("👤 %s (%s)", contact.Name, contact.Position))
			nameBtn.OnTapped = func() {
				w.Window.Clipboard().SetContent(contact.Name)
				ShowToast(w.Window, "Скопійовано ім'я")
			}

			phoneRow := vbox.Objects[1].(*fyne.Container)
			phoneLabel := phoneRow.Objects[0].(*widget.Label)
			phoneBtn := phoneRow.Objects[1].(*widget.Button)
			phoneLabel.SetText("📞 " + contact.Phone)
			phoneBtn.OnTapped = func() {
				w.Window.Clipboard().SetContent(contact.Phone)
				ShowToast(w.Window, "Скопійовано телефон")
			}
		},
	)
	w.ContactsFlatContent = w.ContactsList
	w.ContactsContent = container.NewMax(w.ContactsFlatContent)
	return w.ContactsContent
}

func (w *WorkAreaPanel) buildGroupedContactsAccordion(
	sections []viewmodels.WorkAreaGroupSection,
) fyne.CanvasObject {
	items := make([]*widget.AccordionItem, 0, len(sections))
	for _, section := range sections {
		title := w.GroupSectionsVM.FormatSectionTitle(section.Group)
		items = append(items, widget.NewAccordionItem(title, w.buildGroupedContactsSection(section)))
	}
	accordion := widget.NewAccordion(items...)
	if len(items) > 0 {
		accordion.Open(0)
	}
	return container.NewScroll(accordion)
}

func (w *WorkAreaPanel) buildGroupedContactsSection(
	section viewmodels.WorkAreaGroupSection,
) fyne.CanvasObject {
	if len(section.Contacts) == 0 {
		return container.NewPadded(widget.NewLabel("Немає відповідальних у цій групі"))
	}

	rows := make([]fyne.CanvasObject, 0, len(section.Contacts)*2)
	for _, contact := range section.Contacts {
		name := contact.Name
		if strings.TrimSpace(contact.Position) != "" {
			name += " (" + contact.Position + ")"
		}

		phone := strings.TrimSpace(contact.Phone)
		if phone == "" {
			phone = "—"
		}

		rows = append(rows,
			widget.NewLabel(fmt.Sprintf("%d. %s", contact.Priority, name)),
			widget.NewLabel("📞 "+phone),
		)
	}

	return container.NewPadded(container.NewVBox(rows...))
}

func (w *WorkAreaPanel) createEventsTab() fyne.CanvasObject {
	eventsList := widget.NewListWithData(
		w.EventsData,
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Подія", color.White)
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			data, ok := item.(binding.Untyped)
			if !ok {
				return
			}
			value, err := data.Get()
			if err != nil {
				return
			}
			event, ok := value.(models.Event)
			if !ok {
				return
			}

			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtContainer := stack.Objects[1].(*fyne.Container)
			txt := txtContainer.Objects[0].(*canvas.Text)

			textColor, rowColor := eventRowColors(event.SC1)

			bg.FillColor = rowColor
			bg.Refresh()

			txt.Color = textColor

			txt.Text = formatWorkAreaEventRowText(event)
			txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
			txt.Refresh()
		},
	)

	eventsScroll, eventsWidthGuide := newHorizontalJournalScroll(eventsList)
	w.EventsList = eventsList
	w.EventsWidthGuide = eventsWidthGuide
	return eventsScroll
}

// SetObject встановлює об'єкт та запускає фонове завантаження деталей
func (w *WorkAreaPanel) SetObject(object models.Object) {
	w.CurrentObject = &object
	if w.ExportPDFBtn != nil {
		w.ExportPDFBtn.Enable()
	}
	if w.ExportXLSXBtn != nil {
		w.ExportXLSXBtn.Enable()
	}
	if w.CopyExcelBtn != nil {
		w.CopyExcelBtn.Enable()
	}

	// Оновлюємо базову інфу
	w.HeaderVM.ApplyObject(object)
	w.HeaderStatus.Text = object.GetStatusDisplay()
	w.HeaderStatus.Color = GetStatusColor(object.Status)
	w.HeaderStatus.Refresh()
	w.updateSummaryStatusAccent(object.Status)

	// Налаштовуємо дії копіювання
	w.CopyNameBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(object.Name)
		ShowToast(w.Window, "Скопійовано назву об'єкта")
	}
	w.CopyAddressBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(object.Address)
		ShowToast(w.Window, "Скопійовано адресу")
	}

	// Очищуємо старі деталі та показуємо завантаження
	w.Zones = nil
	w.Contacts = nil
	w.Events = nil
	w.eventsLoadedObjectID = 0
	w.eventsLoadingObjectID = 0

	w.updateDeviceInfo()
	w.refreshTabs()
	w.ensureCurrentObjectEventsLoaded()

	// Запускаємо асинхронне завантаження
	go w.loadObjectDetails(object.ID)
}

func (w *WorkAreaPanel) exportSelectedObject(format string) {
	if w.CurrentObject == nil {
		ShowToast(w.Window, "Спочатку оберіть об'єкт")
		return
	}

	obj := *w.CurrentObject
	zones := append([]models.Zone(nil), w.Zones...)
	contacts := append([]models.Contact(nil), w.Contacts...)
	events := append([]models.Event(nil), w.Events...)
	eventsLoaded := w.eventsLoadedObjectID == obj.ID

	if w.ExportPDFBtn != nil {
		w.ExportPDFBtn.Disable()
	}
	if w.ExportXLSXBtn != nil {
		w.ExportXLSXBtn.Disable()
	}

	go func() {
		if !eventsLoaded {
			uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
			events = w.ViewModel.LoadObjectEvents(w.Data, obj.ID, uiCfg.ObjectLogLimit)
		}
		externalData := w.ViewModel.LoadExternalData(w.Data, obj.ID)
		exportData := w.ExportVM.BuildObjectExportData(obj, zones, contacts, events, externalData)
		uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
		exportDir := uiCfg.ExportDir

		var (
			filePath string
			err      error
		)

		switch strings.ToLower(format) {
		case "pdf":
			filePath, err = objexport.ExportObjectToPDF(exportData, exportDir)
		case "xlsx":
			filePath, err = objexport.ExportObjectToXLSX(exportData, exportDir)
		default:
			err = fmt.Errorf("unsupported export format: %s", format)
		}

		fyne.Do(func() {
			if w.ExportPDFBtn != nil {
				w.ExportPDFBtn.Enable()
			}
			if w.ExportXLSXBtn != nil {
				w.ExportXLSXBtn.Enable()
			}

			if err != nil {
				dialog.ShowError(err, w.Window)
				return
			}

			ShowToast(w.Window, "Експорт завершено")
			dialogs.ShowInfoDialog(w.Window, "Експорт виконано", "Файл створено:\n"+filePath)
		})
	}()
}

func (w *WorkAreaPanel) loadObjectDetails(id int) {
	details := w.ViewModel.LoadObjectBaseDetails(w.Data, id)

	fyne.Do(func() {
		// Перевіряємо, чи користувач досі на цьому ж об'єкті
		if !w.ViewModel.CanApplyDetails(w.CurrentObject, id) {
			return
		}

		if details.FullObject != nil {
			w.CurrentObject = details.FullObject
			w.updateDeviceInfo()
		}

		w.Zones = details.Zones
		w.Contacts = details.Contacts

		w.refreshTabs()
	})
}

func (w *WorkAreaPanel) refreshTabs() {
	w.syncZonesDataBinding()
	w.syncContactsDataBinding()
	w.syncEventsDataBinding()

	w.rebuildZonesContent()
	w.rebuildContactsContent()
	if w.EventsList != nil {
		w.EventsList.Refresh()
	}
}

func (w *WorkAreaPanel) rebuildZonesContent() {
	if w == nil || w.ZonesContent == nil {
		return
	}

	if w.GroupSectionsVM != nil && w.GroupSectionsVM.ShouldUseGroupedZones(w.CurrentObject, w.Zones) {
		sections := w.GroupSectionsVM.BuildZoneSections(w.CurrentObject, w.Zones)
		if len(sections) > 0 {
			w.ZonesContent.Objects = []fyne.CanvasObject{w.buildGroupedZonesAccordion(sections)}
			w.ZonesContent.Refresh()
			return
		}
	}

	if w.ZonesFlatContent != nil {
		w.ZonesContent.Objects = []fyne.CanvasObject{w.ZonesFlatContent}
	}
	w.ZonesContent.Refresh()
	if w.ZonesTable != nil {
		w.ZonesTable.Refresh()
	}
}

func (w *WorkAreaPanel) rebuildContactsContent() {
	if w == nil || w.ContactsContent == nil {
		return
	}

	if w.GroupSectionsVM != nil && w.GroupSectionsVM.ShouldUseGroupedContacts(w.CurrentObject, w.Contacts) {
		sections := w.GroupSectionsVM.BuildContactSections(w.CurrentObject, w.Contacts)
		if len(sections) > 0 {
			w.ContactsContent.Objects = []fyne.CanvasObject{w.buildGroupedContactsAccordion(sections)}
			w.ContactsContent.Refresh()
			return
		}
	}

	if w.ContactsFlatContent != nil {
		w.ContactsContent.Objects = []fyne.CanvasObject{w.ContactsFlatContent}
	}
	w.ContactsContent.Refresh()
	if w.ContactsList != nil {
		w.ContactsList.Refresh()
	}
}

func (w *WorkAreaPanel) syncZonesDataBinding() {
	if w == nil || w.ZonesData == nil {
		return
	}
	_ = SetUntypedList(w.ZonesData, w.Zones)
	w.refreshZonesTableLayout()
}

func (w *WorkAreaPanel) syncContactsDataBinding() {
	if w == nil || w.ContactsData == nil {
		return
	}
	_ = SetUntypedList(w.ContactsData, w.Contacts)
}

func (w *WorkAreaPanel) syncEventsDataBinding() {
	if w == nil || w.EventsData == nil {
		return
	}
	_ = SetUntypedList(w.EventsData, w.Events)
	fyne.Do(func() {
		ensureJournalListMinWidth(w.EventsWidthGuide, workAreaEventRowTexts(w.Events), 0, fyne.TextStyle{})
		if w.EventsList != nil {
			w.EventsList.Refresh()
		}
	})
}

func (w *WorkAreaPanel) refreshCaseHistoryAccordion() {
	if w == nil || w.CaseHistoryAccordion == nil || w.CaseHistorySection == nil || w.CaseHistoryVM == nil {
		return
	}

	groups := w.CaseHistoryVM.BuildGroups(w.CurrentObject, w.Events)
	if len(groups) == 0 {
		w.CaseHistoryAccordion.Items = nil
		w.CaseHistoryAccordion.Refresh()
		w.CaseHistorySection.Hide()
		return
	}

	items := make([]*widget.AccordionItem, 0, len(groups))
	for _, group := range groups {
		rows := make([]fyne.CanvasObject, 0, len(group.Events))
		for _, event := range group.Events {
			line := event.GetDateTimeDisplay()
			if event.ZoneNumber > 0 {
				line += " | Зона " + strconv.Itoa(event.ZoneNumber)
			}
			line += " | " + event.GetTypeDisplay()
			if details := strings.TrimSpace(event.Details); details != "" {
				line += " — " + details
			}

			label := widget.NewLabel(line)
			label.Wrapping = fyne.TextWrapWord
			rows = append(rows, label)
		}
		items = append(items, widget.NewAccordionItem(
			group.Title,
			container.NewPadded(container.NewVBox(rows...)),
		))
	}

	w.CaseHistoryAccordion.Items = items
	w.CaseHistoryAccordion.Refresh()
	w.CaseHistorySection.Show()
}

func (w *WorkAreaPanel) isJournalTabSelected() bool {
	return w != nil && w.Tabs != nil && w.Tabs.SelectedIndex() == workAreaJournalTabIndex
}

func (w *WorkAreaPanel) ensureCurrentObjectEventsLoaded() {
	if w == nil || w.CurrentObject == nil || w.Data == nil || w.ViewModel == nil {
		return
	}
	if !w.isJournalTabSelected() {
		return
	}

	objectID := w.CurrentObject.ID
	if w.eventsLoadedObjectID == objectID || w.eventsLoadingObjectID == objectID {
		return
	}

	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	eventLimit := uiCfg.ObjectLogLimit
	w.eventsLoadingObjectID = objectID

	go func(id int) {
		events := w.ViewModel.LoadObjectEvents(w.Data, id, eventLimit)
		fyne.Do(func() {
			if w.eventsLoadingObjectID == id {
				w.eventsLoadingObjectID = 0
			}
			if !w.ViewModel.CanApplyDetails(w.CurrentObject, id) {
				return
			}

			w.Events = events
			w.eventsLoadedObjectID = id
			w.syncEventsDataBinding()
			if w.EventsList != nil {
				w.EventsList.Refresh()
			}
		})
	}(objectID)
}

func (w *WorkAreaPanel) zoneByRow(row int) (models.Zone, bool) {
	if w == nil || w.ZonesData == nil || row < 0 || row >= w.ZonesData.Length() {
		return models.Zone{}, false
	}
	value, err := w.ZonesData.GetValue(row)
	if err != nil {
		return models.Zone{}, false
	}
	zone, ok := value.(models.Zone)
	return zone, ok
}

func (w *WorkAreaPanel) refreshZonesTableLayout() {
	if w == nil || w.ZonesTable == nil {
		return
	}
	updateZoneTableRowHeights(w.ZonesTable, w.Zones, zoneTableNameColumnWidth(w.ZonesTable.Size().Width))
}

// RefreshCurrentObjectEvents оновлює тільки журнал подій для поточного об'єкта.
// Використовується для "онлайн" автооновлення без перезавантаження всіх деталей.
func (w *WorkAreaPanel) RefreshCurrentObjectEvents() {
	if w == nil || w.CurrentObject == nil || w.Data == nil || w.ViewModel == nil {
		return
	}
	if !w.isJournalTabSelected() {
		return
	}

	objectID := w.CurrentObject.ID
	if w.eventsLoadedObjectID != objectID {
		w.ensureCurrentObjectEventsLoaded()
		return
	}
	if w.eventsLoadingObjectID == objectID {
		return
	}

	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	eventLimit := uiCfg.ObjectLogLimit

	go func(id int) {
		events := w.ViewModel.LoadObjectEvents(w.Data, id, eventLimit)
		fyne.Do(func() {
			if !w.ViewModel.CanApplyDetails(w.CurrentObject, id) {
				return
			}
			w.Events = events
			w.syncEventsDataBinding()
			if w.EventsList != nil {
				w.EventsList.Refresh()
			}
		})
	}(objectID)
}

func (w *WorkAreaPanel) updateDeviceInfo() {
	if w.CurrentObject == nil {
		return
	}
	obj := w.CurrentObject

	presentation := w.DeviceVM.BuildObjectPresentation(*obj)
	w.DeviceStateVM.Apply(presentation)
	w.CopySimBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(presentation.SIMCopyText)
		ShowToast(w.Window, "Скопійовано SIM")
	}
	w.CopySIM1Btn.OnTapped = func() {
		if strings.TrimSpace(presentation.SIM1Value) == "" {
			ShowToast(w.Window, "SIM1 не вказана")
			return
		}
		w.Window.Clipboard().SetContent(presentation.SIM1Value)
		ShowToast(w.Window, "Скопійовано SIM1")
	}
	w.CopySIM2Btn.OnTapped = func() {
		if strings.TrimSpace(presentation.SIM2Value) == "" {
			ShowToast(w.Window, "SIM2 не вказана")
			return
		}
		w.Window.Clipboard().SetContent(presentation.SIM2Value)
		ShowToast(w.Window, "Скопійовано SIM2")
	}

	configureSIMButton := func(btn *widget.Button, simValue string) {
		if btn == nil {
			return
		}
		simValue = strings.TrimSpace(simValue)
		objectNumber := strings.TrimSpace(viewmodels.ObjectDisplayNumber(*obj))
		operator := simoperator.Detect(simValue)
		btn.SetText(simoperator.Label(operator))
		btn.OnTapped = func() {
			admin := w.resolveAdminProvider()
			if admin == nil {
				dialogs.ShowInfoDialog(w.Window, simoperator.Label(operator), "Сервіс оператора недоступний.")
				return
			}
			switch operator {
			case simoperator.Vodafone:
				dialogs.ShowVodafoneSIMDialog(w.Window, admin, simValue, objectNumber, obj.Name)
			case simoperator.Kyivstar:
				dialogs.ShowKyivstarSIMDialog(w.Window, admin, simValue, objectNumber, obj.Name)
			default:
				dialogs.ShowInfoDialog(w.Window, "SIM API", "Оператор номера не підтримується.")
			}
		}
		if operator == simoperator.Unknown {
			btn.Disable()
			return
		}
		btn.Enable()
	}

	configureSIMButton(w.VodafoneSIM1Btn, presentation.SIM1Value)
	configureSIMButton(w.VodafoneSIM2Btn, presentation.SIM2Value)

	// Скидаємо динамічні дані перед завантаженням нових
	loading := w.DeviceVM.BuildLoadingExternalPresentation()
	w.ExternalVM.Apply(loading)

	// Рівень сигналу та останній тест
	go func() {
		externalData := w.ViewModel.LoadExternalData(w.Data, obj.ID)
		fyne.Do(func() {
			if w.CurrentObject == nil || w.CurrentObject.ID != obj.ID {
				return
			}
			external := w.DeviceVM.BuildExternalPresentation(
				externalData.Signal,
				externalData.TestMessage,
				externalData.LastTest,
				externalData.LastMessage,
			)
			w.ExternalVM.Apply(external)
		})
	}()

	w.TestLogsBtn.OnTapped = func() {
		w.showTestMessages(strconv.Itoa(obj.ID))
	}

	w.CopyPhonesBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(presentation.PhoneCopyText)
		ShowToast(w.Window, "Скопійовано телефон(и)")
	}

	w.CopyNotesBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(presentation.NotesCopyText)
		ShowToast(w.Window, "Скопійовано примітку")
	}

	w.CopyLocationBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(presentation.LocationCopyText)
		ShowToast(w.Window, "Скопійовано розташування")
	}
}

func (w *WorkAreaPanel) resolveAdminProvider() contracts.AdminProvider {
	if w == nil || w.Data == nil {
		return nil
	}
	if provider, ok := any(w.Data).(contracts.AdminProvider); ok {
		return provider
	}
	if resolver, ok := any(w.Data).(interface {
		AdminProvider() contracts.AdminProvider
	}); ok {
		admin := resolver.AdminProvider()
		if admin != nil {
			return admin
		}
	}
	return nil
}

func (w *WorkAreaPanel) showTestMessages(objectID string) {
	dialogs.ShowTestMessagesDialog(w.Window, w.Data, objectID)
}

func (w *WorkAreaPanel) OnThemeChanged(fontSize float32) {
	if w.HeaderStatus != nil {
		w.HeaderStatus.TextSize = fontSize + 3
		w.HeaderStatus.Refresh()
	}
	if w.SummaryStateCaption != nil {
		w.SummaryStateCaption.TextSize = fontSize - 1
	}
	w.updateSummaryThemeColors()
	if w.CurrentObject != nil {
		w.updateSummaryStatusAccent(w.CurrentObject.Status)
	}
	// Віджети (Labels, Tables) оновляться автоматично через Refresh
	if w.ZonesTable != nil {
		w.ZonesTable.Refresh()
	}
	if w.ContactsList != nil {
		w.ContactsList.Refresh()
	}
	if w.EventsList != nil {
		ensureJournalListMinWidth(w.EventsWidthGuide, workAreaEventRowTexts(w.Events), fontSize, fyne.TextStyle{})
		w.EventsList.Refresh()
	}
	if w.ZonesContent != nil {
		w.ZonesContent.Refresh()
	}
	if w.ContactsContent != nil {
		w.ContactsContent.Refresh()
	}
}

func formatWorkAreaEventRowText(event models.Event) string {
	text := event.GetDateTimeDisplay() + " " + getEventIcon(event.Type)
	if event.ZoneNumber > 0 {
		text += " | Зона " + strconv.Itoa(event.ZoneNumber)
	}
	text += " | " + event.GetTypeDisplay()
	if event.Details != "" {
		text += " — " + event.Details
	}
	return text
}

func workAreaEventRowTexts(events []models.Event) []string {
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, formatWorkAreaEventRowText(event))
	}
	return texts
}
