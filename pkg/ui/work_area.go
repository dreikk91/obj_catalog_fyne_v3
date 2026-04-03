// Package ui - робоча область з деталями об'єкта
package ui

import (
	"fmt"
	"image/color"
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
	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/utils"
)

// WorkAreaPanel - структура робочої області
type WorkAreaPanel struct {
	Container       *fyne.Container
	Data            contracts.WorkAreaProvider
	ViewModel       *viewmodels.WorkAreaViewModel
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
	ZonesTable          *widget.Table
	ContactsList        *widget.List
	EventsList          *widget.List
	ZonesContent        *fyne.Container
	ContactsContent     *fyne.Container
	ZonesFlatContent    fyne.CanvasObject
	ContactsFlatContent fyne.CanvasObject
}

// NewWorkAreaPanel створює робочу область
func NewWorkAreaPanel(provider contracts.WorkAreaProvider, window fyne.Window) *WorkAreaPanel {
	panel := &WorkAreaPanel{
		Data:            provider,
		ViewModel:       viewmodels.NewWorkAreaViewModel(),
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
	w.PanelMarkLabel = widget.NewLabelWithData(w.DeviceStateVM.PanelMarkBinding()) // Initialized PanelMarkLabel
	w.GroupsLabel = widget.NewLabelWithData(w.DeviceStateVM.GroupsBinding())
	w.GroupsLabel.Wrapping = fyne.TextWrapWord
	// w.GSMLabel = widget.NewLabel("📶 GSM: —")
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

	notesScroll := container.NewScroll(w.Notes1Label)
	notesScroll.SetMinSize(fyne.NewSize(0, 80))

	locationScroll := container.NewScroll(w.Location1Label)
	locationScroll.SetMinSize(fyne.NewSize(0, 60))

	deviceInfo := container.NewVBox(
		widget.NewLabel("📟 ЗАГАЛЬНА ІНФОРМАЦІЯ:"),
		widget.NewSeparator(),
		container.NewHBox(
			container.NewVBox(w.DeviceTypeLabel, w.PanelMarkLabel, w.SignalLabel, w.PowerLabel, w.ChanLabel),
			widget.NewSeparator(),
			container.NewVBox(
				container.NewBorder(nil, nil, nil, container.NewHBox(w.CopySimBtn), w.SIMLabel),
				container.NewBorder(nil, nil, nil, container.NewHBox(w.VodafoneSIM1Btn, w.CopySIM1Btn), w.SIM1Label),
				container.NewBorder(nil, nil, nil, container.NewHBox(w.VodafoneSIM2Btn, w.CopySIM2Btn), w.SIM2Label),
				container.NewBorder(nil, nil, nil, w.CopyPhonesBtn, w.PhoneLabel),
				w.AkbLabel,
				w.AutoTestLabel,
				w.TestControlLabel,
				w.LastTestLabel,
				w.LastTestTimeLabel,
				w.LastMessageTimeLabel,
				w.GuardLabel,
				w.GroupsLabel,
				widget.NewSeparator(),
				w.TestLogsBtn,
			),
		),
		widget.NewSeparator(),
		widget.NewLabel("📌 РОЗТАШУВАННЯ:"),
		container.NewBorder(nil, nil, nil, w.CopyLocationBtn, locationScroll),
		widget.NewLabel("📝 ДОДАТКОВА ІНФОРМАЦІЯ:"),
		container.NewBorder(nil, nil, nil, w.CopyNotesBtn, notesScroll),
	)

	// Додаємо скрол до всієї вкладки
	return container.NewScroll(container.NewPadded(container.NewVBox(deviceInfo)))
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
			label.Truncation = fyne.TextTruncateEllipsis
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

			var text string
			switch id.Col {
			case 0:
				text = "№" + itoa(zone.Number)
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

	w.ZonesTable.SetColumnWidth(0, 50)
	w.ZonesTable.SetColumnWidth(1, 200)
	w.ZonesTable.SetColumnWidth(2, 100)
	w.ZonesTable.SetColumnWidth(3, 100)
	w.ZonesTable.SetColumnWidth(4, 40)

	w.ZonesFlatContent = container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		container.New(&zonesTableLayout{table: w.ZonesTable}, w.ZonesTable),
	)
	w.ZonesContent = container.NewMax(w.ZonesFlatContent)

	return w.ZonesContent
}

type zonesTableLayout struct {
	table         *widget.Table
	lastNameWidth float32
}

func (l *zonesTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Fixed columns: ID(50) + Type(100) + Status(100) + Copy(40) = 290
	fixedWidth := float32(290)
	available := size.Width - fixedWidth - 10
	if available < 150 {
		available = 150
	}

	if l.lastNameWidth != available {
		l.table.SetColumnWidth(1, available)
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
			return len(section.Zones), 4
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Data")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			zone := section.Zones[id.Row]

			switch id.Col {
			case 0:
				label.SetText("№" + itoa(zone.Number))
			case 1:
				label.SetText(zone.Name)
			case 2:
				label.SetText(zone.SensorType)
			case 3:
				label.SetText(zone.GetStatusDisplay())
			default:
				label.SetText("")
			}
		},
	)
	table.SetColumnWidth(0, 50)
	table.SetColumnWidth(1, 220)
	table.SetColumnWidth(2, 140)
	table.SetColumnWidth(3, 110)

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

			// Вибираємо палітру кольорів залежно від теми
			var textColor, rowColor color.NRGBA
			if IsDarkMode() {
				textColor, rowColor = utils.SelectColorNRGBADark(event.SC1)
			} else {
				textColor, rowColor = utils.SelectColorNRGBA(event.SC1)
			}

			bg.FillColor = rowColor
			bg.Refresh()

			txt.Color = textColor

			// Формат: час | Зона N | тип — деталі
			text := event.GetDateTimeDisplay()
			if event.ZoneNumber > 0 {
				text += " | Зона " + itoa(event.ZoneNumber)
			}
			text += " | " + event.GetTypeDisplay()
			if event.Details != "" {
				text += " — " + event.Details
			}
			txt.Text = text
			txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
			txt.Refresh()
		},
	)

	w.EventsList = eventsList
	return eventsList
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

	w.updateDeviceInfo()
	w.refreshTabs()

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

	if w.ExportPDFBtn != nil {
		w.ExportPDFBtn.Disable()
	}
	if w.ExportXLSXBtn != nil {
		w.ExportXLSXBtn.Disable()
	}

	go func() {
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
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	details := w.ViewModel.LoadObjectDetails(w.Data, id, uiCfg.ObjectLogLimit)

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
		w.Events = details.Events

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

// RefreshCurrentObjectEvents оновлює тільки журнал подій для поточного об'єкта.
// Використовується для "онлайн" автооновлення без перезавантаження всіх деталей.
func (w *WorkAreaPanel) RefreshCurrentObjectEvents() {
	if w == nil || w.CurrentObject == nil || w.Data == nil || w.ViewModel == nil {
		return
	}

	objectID := w.CurrentObject.ID
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

	configureVodafoneButton := func(btn *widget.Button, simValue string) {
		if btn == nil {
			return
		}
		simValue = strings.TrimSpace(simValue)
		objectNumber := strings.TrimSpace(viewmodels.ObjectDisplayNumber(*obj))
		btn.OnTapped = func() {
			provider := w.resolveVodafoneProvider()
			if provider == nil {
				dialogs.ShowInfoDialog(w.Window, "Vodafone", "Vodafone сервіс недоступний.")
				return
			}
			dialogs.ShowVodafoneSIMDialog(w.Window, provider, simValue, objectNumber, obj.Name)
		}
		if !isVodafonePhone(simValue) {
			btn.Disable()
			return
		}
		btn.Enable()
	}

	configureVodafoneButton(w.VodafoneSIM1Btn, presentation.SIM1Value)
	configureVodafoneButton(w.VodafoneSIM2Btn, presentation.SIM2Value)

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
		w.showTestMessages(itoa(obj.ID))
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

func (w *WorkAreaPanel) resolveVodafoneProvider() contracts.AdminObjectVodafoneService {
	if w == nil || w.Data == nil {
		return nil
	}
	if provider, ok := any(w.Data).(contracts.AdminObjectVodafoneService); ok {
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

func isVodafonePhone(raw string) bool {
	digits := digitsOnly(raw)
	switch {
	case len(digits) >= 5 && strings.HasPrefix(digits, "38050"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38066"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38075"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38095"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38099"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "050"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "066"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "075"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "095"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "099"):
		return true
	default:
		return false
	}
}

func digitsOnly(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (w *WorkAreaPanel) showTestMessages(objectID string) {
	dialogs.ShowTestMessagesDialog(w.Window, w.Data, objectID)
}

func (w *WorkAreaPanel) OnThemeChanged(fontSize float32) {
	if w.HeaderStatus != nil {
		w.HeaderStatus.TextSize = fontSize + 3
		w.HeaderStatus.Refresh()
	}
	// Віджети (Labels, Tables) оновляться автоматично через Refresh
	if w.ZonesTable != nil {
		w.ZonesTable.Refresh()
	}
	if w.ContactsList != nil {
		w.ContactsList.Refresh()
	}
	if w.EventsList != nil {
		w.EventsList.Refresh()
	}
	if w.ZonesContent != nil {
		w.ZonesContent.Refresh()
	}
	if w.ContactsContent != nil {
		w.ContactsContent.Refresh()
	}
}
