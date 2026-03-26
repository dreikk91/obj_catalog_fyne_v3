// Package ui - робоча область з деталями об'єкта
package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/models"
	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
	"obj_catalog_fyne_v3/pkg/utils"
)

// WorkAreaPanel - структура робочої області
type WorkAreaPanel struct {
	Container     *fyne.Container
	Data          contracts.DataProvider
	CurrentObject *models.Object
	Window        fyne.Window

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
	GSMLabel             *widget.Label
	PowerLabel           *widget.Label
	SIMLabel             *widget.Label
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
	CopyPhonesBtn   *widget.Button
	CopyNotesBtn    *widget.Button
	CopyLocationBtn *widget.Button

	// Таблиці
	ZonesTable   *widget.Table
	ContactsList *widget.List
	EventsList   *widget.List
}

// NewWorkAreaPanel створює робочу область
func NewWorkAreaPanel(provider contracts.DataProvider, window fyne.Window) *WorkAreaPanel {
	panel := &WorkAreaPanel{
		Data:   provider,
		Window: window,
	}

	// Шапка
	themeSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)

	// Назва об'єкта: використовуємо Label з перенесенням рядків,
	// щоб довгі назви (до ~200 символів) коректно відображались у межах правої вкладки.
	panel.HeaderName = widget.NewLabel("← Оберіть об'єкт зі списку")
	panel.HeaderName.TextStyle = fyne.TextStyle{Bold: true}
	panel.HeaderName.Wrapping = fyne.TextWrapWord

	// Адреса об'єкта також може бути довгою — вмикаємо перенесення.
	panel.HeaderAddress = widget.NewLabel("")
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

		row := panel.buildExcelRowTSV(*panel.CurrentObject)
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
	w.DeviceTypeLabel = widget.NewLabel("🔧 Тип: —")
	w.PanelMarkLabel = widget.NewLabel("🏷️ Марка: —") // Initialized PanelMarkLabel
	// w.GSMLabel = widget.NewLabel("📶 GSM: —")
	w.PowerLabel = widget.NewLabel("🔌 Живлення: —")
	w.SIMLabel = widget.NewLabel("📱 SIM: —")
	w.AutoTestLabel = widget.NewLabel("⏱️ Автотест: —")
	w.GuardLabel = widget.NewLabel("🔒 Стан: —")
	w.GuardLabel.TextStyle = fyne.TextStyle{Bold: true}
	w.CopySimBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	w.ChanLabel = widget.NewLabel("📡 Канал: —")
	w.PhoneLabel = widget.NewLabel("☎️ Тел. об'єкта: —")
	w.CopyPhonesBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	w.AkbLabel = widget.NewLabel("🔋 АКБ: —")
	w.TestControlLabel = widget.NewLabel("⏲️ Контроль тесту: —")
	w.SignalLabel = widget.NewLabel("📶 Рівень: —")
	w.LastTestLabel = widget.NewLabel("📝 Тест: —")
	w.LastTestTimeLabel = widget.NewLabel("📅 Ост. тест: —")
	w.LastMessageTimeLabel = widget.NewLabel("📅 Ост. подія: —")
	w.TestLogsBtn = widget.NewButtonWithIcon("Тестові повідомлення", fyneTheme.HistoryIcon(), nil)

	w.Notes1Label = widget.NewLabel("")
	w.Notes1Label.Wrapping = fyne.TextWrapWord
	w.CopyNotesBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)

	w.Location1Label = widget.NewLabel("")
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
				container.NewBorder(nil, nil, nil, w.CopySimBtn, w.SIMLabel),
				container.NewBorder(nil, nil, nil, w.CopyPhonesBtn, w.PhoneLabel),
				w.AkbLabel,
				w.AutoTestLabel,
				w.TestControlLabel,
				w.LastTestLabel,
				w.LastTestTimeLabel,
				w.LastMessageTimeLabel,
				w.GuardLabel,
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
			return len(w.Zones), 5
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

			if id.Row >= len(w.Zones) {
				label.SetText("")
				btn.Hide()
				return
			}
			zone := w.Zones[id.Row]

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

	return container.NewBorder(nil, nil, nil, nil, container.New(&zonesTableLayout{table: w.ZonesTable}, w.ZonesTable))
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

func (w *WorkAreaPanel) createContactsTab() fyne.CanvasObject {
	w.ContactsList = widget.NewList(
		func() int {
			return len(w.Contacts)
		},
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
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(w.Contacts) {
				return
			}
			contact := w.Contacts[id]
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
	return w.ContactsList
}

func (w *WorkAreaPanel) createEventsTab() fyne.CanvasObject {
	eventsList := widget.NewList(
		func() int {
			return len(w.Events)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Подія", color.White)
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(w.Events) {
				return
			}
			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtContainer := stack.Objects[1].(*fyne.Container)
			txt := txtContainer.Objects[0].(*canvas.Text)

			event := w.Events[id]

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
	if w.HeaderName != nil {
		w.HeaderName.SetText(fmt.Sprintf("%s (№%d)", object.Name, object.ID))
	}
	if w.HeaderAddress != nil {
		w.HeaderAddress.SetText(fmt.Sprintf("📌 %s | 📄 %s", object.Address, object.ContractNum))
	}
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
		exportData := w.buildObjectExportData(obj, zones, contacts, events)
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

func (w *WorkAreaPanel) buildObjectExportData(
	obj models.Object,
	zones []models.Zone,
	contacts []models.Contact,
	events []models.Event,
) objexport.ObjectExportData {
	lastEventText := "Немає"
	if len(events) > 0 {
		latest := events[0]
		eventTime := "Немає дати"
		if !latest.Time.IsZero() {
			eventTime = latest.Time.Format("02.01.2006 15:04:05")
		}
		lastEventText = fmt.Sprintf("%s | %s", eventTime, latest.GetTypeDisplay())
		if latest.ZoneNumber > 0 {
			lastEventText += fmt.Sprintf(" | Зона %d", latest.ZoneNumber)
		}
		if strings.TrimSpace(latest.Details) != "" {
			lastEventText += " | " + strings.TrimSpace(latest.Details)
		}
	}

	signal, testMsg, lastTestTime, _ := w.Data.GetExternalData(itoa(obj.ID))
	lastTestText := "Немає"
	if !lastTestTime.IsZero() {
		lastTestText = lastTestTime.Format("02.01.2006 15:04:05")
	}
	if strings.TrimSpace(testMsg) != "" && strings.TrimSpace(testMsg) != "—" {
		if lastTestText == "Немає" {
			lastTestText = strings.TrimSpace(testMsg)
		} else {
			lastTestText += " | " + strings.TrimSpace(testMsg)
		}
	}

	_ = signal // currently not requested in export file body

	zoneRows := make([]objexport.ZoneExportRow, 0, len(zones))
	for _, z := range zones {
		zoneRows = append(zoneRows, objexport.ZoneExportRow{
			Number: fmt.Sprintf("%d", z.Number),
			Name:   emptyFallback(z.Name),
			Type:   emptyFallback(z.SensorType),
			Status: emptyFallback(z.GetStatusDisplay()),
		})
	}

	responsibleRows := make([]objexport.ResponsibleExportRow, 0, len(contacts))
	for _, c := range contacts {
		responsibleRows = append(responsibleRows, objexport.ResponsibleExportRow{
			Name:  emptyFallback(c.Name),
			Phone: emptyFallback(c.Phone),
			Note:  emptyFallback(c.Position),
		})
	}

	return objexport.ObjectExportData{
		Number:         obj.ID,
		Name:           emptyFallback(obj.Name),
		Address:        emptyFallback(obj.Address),
		ContractNumber: emptyFallback(obj.ContractNum),
		LaunchDate:     emptyFallback(obj.LaunchDate),
		SimCard:        buildSimValue(obj),
		DeviceType:     emptyFallback(obj.DeviceType),
		TestPeriod:     buildTestPeriod(obj),
		LastEvent:      lastEventText,
		LastTest:       lastTestText,
		Channel:        channelText(obj.ObjChan),
		ObjectPhone:    emptyFallback(obj.Phones1),
		Location:       emptyFallback(obj.Location1),
		AdditionalInfo: emptyFallback(obj.Notes1),
		Zones:          zoneRows,
		Responsibles:   responsibleRows,
	}
}

func (w *WorkAreaPanel) buildExcelRowTSV(obj models.Object) string {
	managerName := ""
	managerPhone := ""
	if len(w.Contacts) > 0 {
		managerName = strings.TrimSpace(w.Contacts[0].Name)
		managerPhone = strings.TrimSpace(w.Contacts[0].Phone)
	} else if len(obj.Contacts) > 0 {
		managerName = strings.TrimSpace(obj.Contacts[0].Name)
		managerPhone = strings.TrimSpace(obj.Contacts[0].Phone)
	}

	fields := []string{
		itoa(obj.ID),                          // собсс
		cleanTSV(obj.LaunchDate),              // Дата підключен. до ПЦС
		cleanTSV(obj.ContractNum),             // Дата угоди (за поточними даними: номер/ідентифікатор угоди)
		"",                                    // Юридична назва, згідно угоди
		"",                                    // Юридична адреса, згідно угоди
		cleanTSV(obj.Name),                    // Фізична назва об’єкту по вивісці
		cleanTSV(obj.Address),                 // Фізична адреса об’єкту
		cleanTSV(obj.DeviceType),              // ПКП
		cleanTSV(obj.PanelMark),               // СЦС
		cleanTSV(strings.TrimSpace(obj.SIM1)), // Основний канал зв’язку / телефон підключення
		cleanTSV(strings.TrimSpace(obj.SIM2)), // Резервний канал зв’язку / телефон підключення
		"",                                    // Місячна оплата
		"",                                    // Електронна пошта об’єкту
		cleanTSV(managerName),                 // Керівник об’єкту
		cleanTSV(managerPhone),                // Контакт керівника
		cleanTSV(obj.Notes1),                  // Примітки
	}

	return strings.Join(fields, "\t")
}

func cleanTSV(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func buildSimValue(obj models.Object) string {
	sim1 := strings.TrimSpace(obj.SIM1)
	sim2 := strings.TrimSpace(obj.SIM2)
	if sim1 == "" && sim2 == "" {
		return "Немає"
	}
	if sim2 == "" {
		return sim1
	}
	if sim1 == "" {
		return sim2
	}
	return sim1 + " / " + sim2
}

func buildTestPeriod(obj models.Object) string {
	if obj.AutoTestHours > 0 {
		return fmt.Sprintf("Кожні %d год", obj.AutoTestHours)
	}
	if obj.TestTime > 0 {
		return fmt.Sprintf("Кожні %d хв", obj.TestTime)
	}
	return "Немає"
}

func channelText(chanID int) string {
	switch chanID {
	case 1:
		return "Автододзвон"
	case 5:
		return "GPRS"
	default:
		return "Інший канал"
	}
}

func emptyFallback(v string) string {
	if strings.TrimSpace(v) == "" {
		return "Немає"
	}
	return strings.TrimSpace(v)
}

func (w *WorkAreaPanel) loadObjectDetails(id int) {
	idStr := itoa(id)

	// Отримуємо повні дані (якщо вони були не всі в списку)
	fullObj := w.Data.GetObjectByID(idStr)

	// Зони
	zones := w.Data.GetZones(idStr)

	// Контакти
	contacts := w.Data.GetEmployees(idStr)

	// Події
	events := w.Data.GetObjectEvents(idStr)
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	if uiCfg.ObjectLogLimit > 0 && len(events) > uiCfg.ObjectLogLimit {
		events = events[:uiCfg.ObjectLogLimit]
	}

	fyne.Do(func() {
		// Перевіряємо, чи користувач досі на цьому ж об'єкті
		if w.CurrentObject == nil || w.CurrentObject.ID != id {
			return
		}

		if fullObj != nil {
			w.CurrentObject = fullObj
			w.updateDeviceInfo()
		}

		w.Zones = zones
		w.Contacts = contacts
		w.Events = events

		w.refreshTabs()
	})
}

func (w *WorkAreaPanel) refreshTabs() {
	if w.ZonesTable != nil {
		w.ZonesTable.Refresh()
	}
	if w.ContactsList != nil {
		w.ContactsList.Refresh()
	}
	if w.EventsList != nil {
		w.EventsList.Refresh()
	}
}

func (w *WorkAreaPanel) updateDeviceInfo() {
	if w.CurrentObject == nil {
		return
	}
	obj := w.CurrentObject

	w.DeviceTypeLabel.SetText("🔧 Тип: " + obj.DeviceType)
	w.PanelMarkLabel.SetText("🏷️ Марка: " + obj.PanelMark) // Updated PanelMarkLabel
	// gsmIcon := "📶"
	// if obj.GSMLevel < 30 {
	// 	gsmIcon = "📵"
	// }
	// w.GSMLabel.SetText(fmt.Sprintf("%s GSM: %d%%", gsmIcon, obj.GSMLevel))

	powerText := "220В (мережа)"
	if obj.PowerSource == models.PowerBattery {
		powerText = "🔋 АКБ (резерв)"
	}
	w.PowerLabel.SetText("🔌 " + powerText)

	simText := "SIM1: " + obj.SIM1
	copyText := obj.SIM1
	if obj.SIM2 != "" {
		simText += " | SIM2: " + obj.SIM2
		copyText += " / " + obj.SIM2
	}
	w.SIMLabel.SetText("📱 " + simText)
	w.CopySimBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(copyText)
		ShowToast(w.Window, "Скопійовано SIM")
	}

	w.AutoTestLabel.SetText(fmt.Sprintf("⏱️ Автотест: кожні %d год", obj.AutoTestHours))

	chanText := "Інший канал"
	switch obj.ObjChan {
	case 1:
		chanText = "Автододзвон"
	case 5:
		chanText = "GPRS"
	}
	w.ChanLabel.SetText("📡 Канал: " + chanText)

	// АКБ
	akbText := "Норма"
	if obj.AkbState > 0 {
		akbText = "ТРИВОГА (Розряд/Відсутній)"
	}
	w.AkbLabel.SetText("🔋 АКБ: " + akbText)

	// Тестування
	testCtrl := "Виключено"
	if obj.TestControl > 0 {
		testCtrl = fmt.Sprintf("Активно (кожні %d хв)", obj.TestTime)
	}
	w.TestControlLabel.SetText("⏲️ Контроль тесту: " + testCtrl)

	// Скидаємо динамічні дані перед завантаженням нових
	w.SignalLabel.SetText("📶 Рівень: ...")
	w.LastTestLabel.SetText("📝 Тест: ...")
	w.LastTestTimeLabel.SetText("📅 Ост. тест: ...")
	w.LastMessageTimeLabel.SetText("📅 Ост. подія: ...")

	// Рівень сигналу та останній тест
	go func() {
		signal, lastMsg, lTest, lMsg := w.Data.GetExternalData(itoa(obj.ID))
		fyne.Do(func() {
			w.SignalLabel.SetText("📶 Рівень: " + signal)
			w.LastTestLabel.SetText("📝 Тест: " + lastMsg)

			timeFormat := "02.01.2006 15:04:05"
			if !lTest.IsZero() {
				w.LastTestTimeLabel.SetText("📅 Ост. тест: " + lTest.Format(timeFormat))
			} else {
				w.LastTestTimeLabel.SetText("📅 Ост. тест: —")
			}

			if !lMsg.IsZero() {
				w.LastMessageTimeLabel.SetText("📅 Ост. подія: " + lMsg.Format(timeFormat))
			} else {
				w.LastMessageTimeLabel.SetText("📅 Ост. подія: —")
			}
		})
	}()

	w.TestLogsBtn.OnTapped = func() {
		w.showTestMessages(itoa(obj.ID))
	}

	w.PhoneLabel.SetText("☎️ Тел. об'єкта: " + obj.Phones1)
	w.CopyPhonesBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(obj.Phones1)
		ShowToast(w.Window, "Скопійовано телефон(и)")
	}

	w.Notes1Label.SetText(obj.Notes1)
	w.CopyNotesBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(obj.Notes1)
		ShowToast(w.Window, "Скопійовано примітку")
	}

	w.Location1Label.SetText(obj.Location1)
	w.CopyLocationBtn.OnTapped = func() {
		w.Window.Clipboard().SetContent(obj.Location1)
		ShowToast(w.Window, "Скопійовано розташування")
	}

	guardText := "🔒 ПІД ОХОРОНОЮ"
	if !obj.IsUnderGuard {
		guardText = "🔓 ЗНЯТО З ОХОРОНИ"
	}
	w.GuardLabel.SetText(guardText)
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
	w.ZonesTable.Refresh()
	w.ContactsList.Refresh()
	w.EventsList.Refresh()
}
