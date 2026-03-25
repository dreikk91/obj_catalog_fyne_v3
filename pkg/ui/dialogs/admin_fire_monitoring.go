package dialogs

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/data"
	uiwidgets "obj_catalog_fyne_v3/pkg/ui/widgets"
)

func ShowFireMonitoringSettingsDialog(parent fyne.Window, provider data.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Налаштування пожежного моніторингу")
	win.Resize(fyne.NewSize(1060, 680))

	var (
		servers       []data.FireMonitoringServer
		serverStatus  []string
		selectedRow   = -1
		serversMu     sync.RWMutex
		pollingCancel context.CancelFunc
	)

	statusLabel := widget.NewLabel("Готово")

	enabledCheck := widget.NewCheck("Підключено", nil)
	objectIDEntry := widget.NewEntry()
	objectIDEntry.SetPlaceHolder("Ідентифікатор об'єктів")
	ackWaitEntry := widget.NewEntry()
	ackWaitEntry.SetPlaceHolder("сек.")

	dateFmtRadio := widget.NewRadioGroup(
		[]string{
			"Стандартний формат дати",
			"Нестандартний формат дати",
		},
		nil,
	)
	dateFmtRadio.Horizontal = true
	dateFmtRadio.SetSelected("Стандартний формат дати")

	serverHostEntry := widget.NewEntry()
	serverHostEntry.SetPlaceHolder("IP або DNS-ім'я")
	serverPortEntry := widget.NewEntry()
	serverPortEntry.SetPlaceHolder("Порт")
	serverInfoEntry := widget.NewEntry()
	serverInfoEntry.SetPlaceHolder("Опис (напр. ДСНС)")
	serverEnabledCheck := widget.NewCheck("Активний сервер", nil)

	serverTableView := uiwidgets.NewQTableViewWithHeaders(
		[]string{"№", "Сервер", "Порт", "Інфо", "Статус"},
		func() int { return len(servers) },
		func(row, col int) string {
			if row < 0 || row >= len(servers) {
				return ""
			}
			serversMu.RLock()
			s := servers[row]
			status := ""
			if row >= 0 && row < len(serverStatus) {
				status = serverStatus[row]
			}
			serversMu.RUnlock()
			switch col {
			case 0:
				return strconv.Itoa(row + 1)
			case 1:
				return strings.TrimSpace(s.Host)
			case 2:
				if s.Port > 0 {
					return strconv.FormatInt(s.Port, 10)
				}
				return ""
			case 3:
				return strings.TrimSpace(s.Info)
			default:
				return status
			}
		},
	)
	serverTableView.SetSelectionBehavior(uiwidgets.SelectRows)
	serverTable := serverTableView.Widget()
	serverTableView.SetColumnWidth(0, 60)
	serverTableView.SetColumnWidth(1, 320)
	serverTableView.SetColumnWidth(2, 110)
	serverTableView.SetColumnWidth(3, 330)
	serverTableView.SetColumnWidth(4, 90)

	updateServerEditor := func() {
		serversMu.RLock()
		defer serversMu.RUnlock()
		if selectedRow < 0 || selectedRow >= len(servers) {
			serverHostEntry.SetText("")
			serverPortEntry.SetText("")
			serverInfoEntry.SetText("")
			serverEnabledCheck.SetChecked(false)
			return
		}
		s := servers[selectedRow]
		serverHostEntry.SetText(s.Host)
		if s.Port > 0 {
			serverPortEntry.SetText(strconv.FormatInt(s.Port, 10))
		} else {
			serverPortEntry.SetText("")
		}
		serverInfoEntry.SetText(s.Info)
		serverEnabledCheck.SetChecked(s.Enabled)
	}

	serverTableView.OnSelected = func(index uiwidgets.ModelIndex) {
		if index.Row < 0 || index.Row >= len(servers) {
			return
		}
		selectedRow = index.Row
		updateServerEditor()
		statusLabel.SetText(fmt.Sprintf("Вибрано сервер №%d", selectedRow+1))
	}

	applyServerEdits := func() error {
		if selectedRow < 0 || selectedRow >= len(servers) {
			return fmt.Errorf("спочатку виберіть сервер у таблиці")
		}

		host := strings.TrimSpace(serverHostEntry.Text)
		info := strings.TrimSpace(serverInfoEntry.Text)
		portRaw := strings.TrimSpace(serverPortEntry.Text)
		port := int64(0)
		if portRaw != "" {
			n, err := strconv.ParseInt(portRaw, 10, 64)
			if err != nil || n < 0 || n > 65535 {
				return fmt.Errorf("некоректний порт (0..65535)")
			}
			port = n
		}

		serversMu.Lock()
		servers[selectedRow] = data.FireMonitoringServer{
			Host:    host,
			Port:    port,
			Info:    info,
			Enabled: serverEnabledCheck.Checked,
		}
		serversMu.Unlock()
		serverTable.Refresh()
		return nil
	}

	statusForServer := func(s data.FireMonitoringServer) string {
		if !s.Enabled {
			return "вимкн"
		}
		if strings.TrimSpace(s.Host) == "" || s.Port <= 0 {
			return "не задано"
		}
		addr := net.JoinHostPort(strings.TrimSpace(s.Host), strconv.FormatInt(s.Port, 10))
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err != nil {
			return "нема зв'язку"
		}
		_ = conn.Close()
		return "ok"
	}

	runStatusCheck := func() {
		serversMu.RLock()
		snapshot := append([]data.FireMonitoringServer(nil), servers...)
		serversMu.RUnlock()
		if len(snapshot) == 0 {
			return
		}

		newStatus := make([]string, len(snapshot))
		for i, s := range snapshot {
			newStatus[i] = statusForServer(s)
		}

		fyne.Do(func() {
			serversMu.Lock()
			serverStatus = newStatus
			serversMu.Unlock()
			serverTable.Refresh()
		})
	}

	startPolling := func(intervalSec int64) {
		if pollingCancel != nil {
			pollingCancel()
			pollingCancel = nil
		}
		if intervalSec <= 0 {
			intervalSec = 5
		}
		if intervalSec > 3600 {
			intervalSec = 3600
		}

		ctx, cancel := context.WithCancel(context.Background())
		pollingCancel = cancel

		go func() {
			runStatusCheck()
			ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					runStatusCheck()
				}
			}
		}()
	}

	load := func() {
		s, err := provider.GetFireMonitoringSettings()
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося завантажити налаштування")
			return
		}

		enabledCheck.SetChecked(s.Enabled)
		objectIDEntry.SetText(strings.TrimSpace(s.ObjectID))
		ackWaitEntry.SetText(strconv.FormatInt(s.AckWaitSec, 10))
		if s.UseStdDateFmt {
			dateFmtRadio.SetSelected("Стандартний формат дати")
		} else {
			dateFmtRadio.SetSelected("Нестандартний формат дати")
		}

		serversMu.Lock()
		servers = append([]data.FireMonitoringServer(nil), s.Servers...)
		if len(servers) == 0 {
			servers = []data.FireMonitoringServer{{Enabled: true}}
		}
		serverStatus = make([]string, len(servers))
		serversMu.Unlock()

		selectedRow = -1
		serverTable.UnselectAll()
		serverTable.Refresh()
		if len(servers) > 0 {
			serverTable.Select(widget.TableCellID{Row: 0, Col: 0})
		}
		statusLabel.SetText("Налаштування завантажено")
		startPolling(s.AckWaitSec)
	}

	addServerBtn := widget.NewButton("Додати", func() {
		serversMu.RLock()
		if len(servers) >= 8 {
			serversMu.RUnlock()
			statusLabel.SetText("Максимум 8 серверів")
			return
		}
		serversMu.RUnlock()
		serversMu.Lock()
		servers = append(servers, data.FireMonitoringServer{Enabled: true})
		serverStatus = append(serverStatus, "не задано")
		serversMu.Unlock()
		serverTable.Refresh()
		selectedRow = len(servers) - 1
		serverTable.Select(widget.TableCellID{Row: selectedRow, Col: 0})
		statusLabel.SetText("Додано новий сервер")
	})

	deleteServerBtn := widget.NewButton("Видалити", func() {
		if selectedRow < 0 || selectedRow >= len(servers) {
			statusLabel.SetText("Виберіть сервер для видалення")
			return
		}
		serversMu.Lock()
		servers = append(servers[:selectedRow], servers[selectedRow+1:]...)
		if selectedRow >= 0 && selectedRow < len(serverStatus) {
			serverStatus = append(serverStatus[:selectedRow], serverStatus[selectedRow+1:]...)
		}
		if len(servers) == 0 {
			servers = []data.FireMonitoringServer{{Enabled: true}}
			serverStatus = []string{"не задано"}
			selectedRow = 0
		} else if selectedRow >= len(servers) {
			selectedRow = len(servers) - 1
		}
		serversMu.Unlock()
		serverTable.Refresh()
		serverTable.Select(widget.TableCellID{Row: selectedRow, Col: 0})
		statusLabel.SetText("Сервер видалено")
	})

	applyServerBtn := widget.NewButton("Застосувати до рядка", func() {
		if err := applyServerEdits(); err != nil {
			statusLabel.SetText(err.Error())
			return
		}
		statusLabel.SetText("Зміни сервера застосовано")
	})

	saveBtn := widget.NewButton("Застосувати", func() {
		if err := applyServerEdits(); err != nil {
			statusLabel.SetText(err.Error())
			return
		}

		ackWaitRaw := strings.TrimSpace(ackWaitEntry.Text)
		ackWait := int64(5)
		if ackWaitRaw != "" {
			n, err := strconv.ParseInt(ackWaitRaw, 10, 64)
			if err != nil || n <= 0 {
				statusLabel.SetText("Некоректний час очікування підтвердження")
				return
			}
			ackWait = n
		}

		settings := data.FireMonitoringSettings{
			Enabled:       enabledCheck.Checked,
			ObjectID:      strings.TrimSpace(objectIDEntry.Text),
			AckWaitSec:    ackWait,
			UseStdDateFmt: dateFmtRadio.Selected == "Стандартний формат дати",
			Servers: func() []data.FireMonitoringServer {
				serversMu.RLock()
				defer serversMu.RUnlock()
				return append([]data.FireMonitoringServer(nil), servers...)
			}(),
		}

		if err := provider.SaveFireMonitoringSettings(settings); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося зберегти налаштування")
			return
		}

		statusLabel.SetText("Налаштування пожежного моніторингу збережено")
		load()
	})

	checkNowBtn := widget.NewButton("Перевірити сервери", func() {
		go runStatusCheck()
		statusLabel.SetText("Виконую перевірку серверів...")
	})

	refreshBtn := widget.NewButton("Оновити", func() { load() })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	top := container.NewVBox(
		container.NewHBox(
			enabledCheck,
			layout.NewSpacer(),
			widget.NewLabel("Ідентифікатор об'єктів:"),
			container.NewGridWrap(fyne.NewSize(180, 36), objectIDEntry),
			widget.NewLabel("Час очікування підтвердження, сек.:"),
			container.NewGridWrap(fyne.NewSize(80, 36), ackWaitEntry),
		),
		dateFmtRadio,
		widget.NewSeparator(),
	)

	serverEditor := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Сервер:"),
			container.NewGridWrap(fyne.NewSize(250, 36), serverHostEntry),
			widget.NewLabel("Порт:"),
			container.NewGridWrap(fyne.NewSize(100, 36), serverPortEntry),
			widget.NewLabel("Інфо:"),
			container.NewGridWrap(fyne.NewSize(220, 36), serverInfoEntry),
			serverEnabledCheck,
			applyServerBtn,
		),
		container.NewHBox(addServerBtn, deleteServerBtn),
	)

	content := container.NewBorder(
		top,
		container.NewHBox(statusLabel, layout.NewSpacer(), checkNowBtn, refreshBtn, saveBtn, closeBtn),
		nil, nil,
		container.NewBorder(
			nil,
			serverEditor,
			nil, nil,
			serverTable,
		),
	)

	win.SetContent(content)
	win.SetCloseIntercept(func() {
		if pollingCancel != nil {
			pollingCancel()
			pollingCancel = nil
		}
		win.Close()
	})
	load()
	win.Show()
}
