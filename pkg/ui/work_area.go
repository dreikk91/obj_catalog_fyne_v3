// Package ui - робоча область з деталями об'єкта
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/models"
	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
)

// WorkAreaPanel - структура робочої області
type WorkAreaPanel struct {
	Container     *fyne.Container
	Data          data.DataProvider
	CurrentObject *models.Object
	Window        fyne.Window

	// Стан завантаження
	Zones     []models.Zone
	Contacts  []models.Contact
	Events    []models.Event
	IsLoading bool

	// UI елементи
	HeaderName    *canvas.Text
	HeaderAddress *widget.Label
	HeaderStatus  *canvas.Text
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
	EventsTable  *widget.Table
}

// NewWorkAreaPanel створює робочу область
func NewWorkAreaPanel(provider data.DataProvider, window fyne.Window) *WorkAreaPanel {
	panel := &WorkAreaPanel{
		Data:   provider,
		Window: window,
	}

	// Шапка
	themeSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	panel.HeaderName = canvas.NewText("← Оберіть об'єкт зі списку", nil)
	panel.HeaderName.TextSize = themeSize + 5
	panel.HeaderName.TextStyle = fyne.TextStyle{Bold: true}

	panel.HeaderAddress = widget.NewLabel("")
	panel.HeaderStatus = canvas.NewText("", appTheme.ColorNormal)
	panel.HeaderStatus.TextSize = themeSize + 1
	panel.HeaderStatus.TextStyle = fyne.TextStyle{Bold: true}

	panel.CopyNameBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)
	panel.CopyAddressBtn = widget.NewButtonWithIcon("", fyneTheme.ContentCopyIcon(), nil)

	header := container.NewVBox(
		container.NewBorder(nil, nil, nil, panel.CopyNameBtn, panel.HeaderName),
		container.NewBorder(nil, nil, nil, panel.CopyAddressBtn, panel.HeaderAddress),
		panel.HeaderStatus,
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
		widget.NewLabel("📍 РОЗТАШУВАННЯ:"),
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
				}
				return
			}

			label.SetText(text)
			label.Show()
			if id.Col == 1 {
				btn.Show()
				btn.OnTapped = func() { w.Window.Clipboard().SetContent(zone.Name) }
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

	return container.NewBorder(nil, nil, nil, nil, w.ZonesTable)
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
			nameBtn.OnTapped = func() { w.Window.Clipboard().SetContent(contact.Name) }

			phoneRow := vbox.Objects[1].(*fyne.Container)
			phoneLabel := phoneRow.Objects[0].(*widget.Label)
			phoneBtn := phoneRow.Objects[1].(*widget.Button)
			phoneLabel.SetText("📞 " + contact.Phone)
			phoneBtn.OnTapped = func() { w.Window.Clipboard().SetContent(contact.Phone) }
		},
	)
	return w.ContactsList
}

func (w *WorkAreaPanel) createEventsTab() fyne.CanvasObject {
	w.EventsTable = widget.NewTable(
		func() (int, int) {
			return len(w.Events), 3
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("—")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row >= len(w.Events) {
				return
			}
			event := w.Events[id.Row]
			switch id.Col {
			case 0:
				label.SetText(event.GetDateTimeDisplay())
			case 1:
				label.SetText(event.GetTypeDisplay())
			case 2:
				label.SetText(event.Details)
			}
		},
	)
	w.EventsTable.SetColumnWidth(0, 140)
	w.EventsTable.SetColumnWidth(1, 120)
	w.EventsTable.SetColumnWidth(2, 200)

	return w.EventsTable
}

// SetObject встановлює об'єкт та запускає фонове завантаження деталей
func (w *WorkAreaPanel) SetObject(object models.Object) {
	w.CurrentObject = &object

	// Оновлюємо базову інфу
	w.HeaderName.Text = fmt.Sprintf("%s (№%d)", object.Name, object.ID)
	w.HeaderName.Refresh()
	w.HeaderAddress.SetText(fmt.Sprintf("📍 %s | 📄 %s", object.Address, object.ContractNum))
	w.HeaderStatus.Text = object.GetStatusDisplay()
	w.HeaderStatus.Color = GetStatusColor(object.Status)
	w.HeaderStatus.Refresh()

	// Налаштовуємо дії копіювання
	w.CopyNameBtn.OnTapped = func() { w.Window.Clipboard().SetContent(object.Name) }
	w.CopyAddressBtn.OnTapped = func() { w.Window.Clipboard().SetContent(object.Address) }

	// Очищуємо старі деталі та показуємо завантаження
	w.Zones = nil
	w.Contacts = nil
	w.Events = nil

	w.updateDeviceInfo()
	w.refreshTabs()

	// Запускаємо асинхронне завантаження
	go w.loadObjectDetails(object.ID)
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
	if w.EventsTable != nil {
		w.EventsTable.Refresh()
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
	w.CopySimBtn.OnTapped = func() { w.Window.Clipboard().SetContent(copyText) }

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
	w.CopyPhonesBtn.OnTapped = func() { w.Window.Clipboard().SetContent(obj.Phones1) }

	w.Notes1Label.SetText(obj.Notes1)
	w.CopyNotesBtn.OnTapped = func() { w.Window.Clipboard().SetContent(obj.Notes1) }

	w.Location1Label.SetText(obj.Location1)
	w.CopyLocationBtn.OnTapped = func() { w.Window.Clipboard().SetContent(obj.Location1) }

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
	if w.HeaderName != nil {
		w.HeaderName.TextSize = fontSize + 5
		w.HeaderName.Refresh()
	}
	if w.HeaderStatus != nil {
		w.HeaderStatus.TextSize = fontSize + 1
		w.HeaderStatus.Refresh()
	}
	// Віджети (Labels, Tables) оновляться автоматично через Refresh
	w.ZonesTable.Refresh()
	w.ContactsList.Refresh()
	w.EventsTable.Refresh()
}
