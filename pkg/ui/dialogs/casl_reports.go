package dialogs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// CASLReportsProvider описує мінімальний API для CASL-звітів.
type CASLReportsProvider interface {
	GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error)
}

func ShowCASLReportsDialog(parent fyne.Window, provider CASLReportsProvider) {
	if provider == nil {
		ShowInfoDialog(parent, "Недоступно", "Провайдер CASL-звітів недоступний.")
		return
	}

	win := fyne.CurrentApp().NewWindow("CASL Cloud: Звіти")
	win.Resize(fyne.NewSize(1280, 820))

	reportNameEntry := widget.NewEntry()
	reportNameEntry.SetText("stats_events")
	reportNameEntry.SetPlaceHolder("Назва звіту, наприклад: stats_events")
	avalaibleCommands := widget.NewLabel("Доступні команди: stats_events, stats_objects, stats_activity, stats_mgr_status, stats_device_blocked, stats_users, stats_devices_v2, stats_devices_offline, stats_connection")


	limitEntry := widget.NewEntry()
	limitEntry.SetText("500")
	limitEntry.SetPlaceHolder("Ліміт рядків")

	statusLabel := widget.NewLabel("Готово")
	summaryLabel := widget.NewLabel("Рядків: 0")
	summaryLabel.Wrapping = fyne.TextWrapWord

	output := widget.NewMultiLineEntry()
	output.SetText("[]")
	output.Wrapping = fyne.TextWrapOff
	output.MultiLine = true
	// output.Disable()

	parseLimit := func(raw string) int {
		value := strings.TrimSpace(raw)
		if value == "" {
			return 500
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return 500
		}
		if parsed > 100000 {
			return 100000
		}
		return parsed
	}

	renderRows := func(rows []map[string]any) {
		pretty, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			output.SetText(fmt.Sprintf("Не вдалося відобразити JSON: %v", err))
			return
		}
		output.SetText(string(pretty))

		keySet := make(map[string]struct{})
		for _, row := range rows {
			for key := range row {
				keySet[key] = struct{}{}
			}
		}
		keys := make([]string, 0, len(keySet))
		for key := range keySet {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			summaryLabel.SetText(fmt.Sprintf("Рядків: %d", len(rows)))
			return
		}
		summaryLabel.SetText(fmt.Sprintf("Рядків: %d | Поля: %s", len(rows), strings.Join(keys, ", ")))
	}

	loadReport := func() {
		name := strings.TrimSpace(reportNameEntry.Text)
		if name == "" {
			ShowInfoDialog(win, "Некоректні параметри", "Вкажіть назву звіту (name).")
			return
		}
		limit := parseLimit(limitEntry.Text)

		statusLabel.SetText("Завантаження...")
		go func() {
			rows, err := provider.GetStatisticReport(context.Background(), name, limit)
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Помилка завантаження")
					dialog.ShowError(err, win)
					return
				}
				statusLabel.SetText("Готово")
				renderRows(rows)
			})
		}()
	}

	loadBtn := makePrimaryButton("Завантажити", loadReport)
	closeBtn := makeIconButton("Закрити", iconClose(), widget.LowImportance, func() { win.Close() })

	form := widget.NewForm(
		widget.NewFormItem("name", reportNameEntry),
		widget.NewFormItem("limit", limitEntry),
	)

	top := container.NewVBox(
		widget.NewCard("Параметри звіту", "", form),
		widget.NewCard("Доступні команди", "", avalaibleCommands),
		container.NewHBox(loadBtn, layout.NewSpacer(), statusLabel),
		widget.NewCard("Зведення", "", summaryLabel),
	)

	content := container.NewBorder(
		top,
		container.NewHBox(layout.NewSpacer(), closeBtn),
		nil, nil,
		container.NewScroll(output),
	)
	win.SetContent(content)

	loadReport()
	win.Show()
}
