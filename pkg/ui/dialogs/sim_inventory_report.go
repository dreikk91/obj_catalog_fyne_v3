package dialogs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type SIMInventoryReportProvider = viewmodels.SIMInventoryReportProvider

func ShowSIMInventoryReportDialog(parent fyne.Window, provider viewmodels.SIMInventoryReportProvider) {
	if provider == nil {
		ShowInfoDialog(parent, "Недоступно", "Провайдер звіту по SIM-картах недоступний.")
		return
	}

	vm := viewmodels.NewSIMInventoryViewModel()

	win := fyne.CurrentApp().NewWindow("Звіт по SIM-картах")
	win.Resize(fyne.NewSize(1460, 860))

	caslLimitEntry := widget.NewEntry()
	caslLimitEntry.SetText(strconv.Itoa(viewmodels.DefaultCASLSIMReportLimit))
	caslLimitEntry.SetPlaceHolder("Ліміт CASL rows")
	if !provider.SupportsCASLReports() {
		caslLimitEntry.Disable()
	}

	statusLabel := makeStatusLabel("Готово")
	summaryLabel := makeStatusLabel("Рядків: 0")
	statusLabel.Wrapping = fyne.TextWrapWord
	summaryLabel.Wrapping = fyne.TextWrapWord

	var (
		lastRows     []viewmodels.SIMInventoryReportRow
		exportTSVBtn *widget.Button
		exportCSVBtn *widget.Button
	)

	parseLimit := func(raw string) int {
		value := strings.TrimSpace(raw)
		if value == "" {
			return viewmodels.DefaultCASLSIMReportLimit
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return viewmodels.DefaultCASLSIMReportLimit
		}
		if parsed > 100000 {
			return 100000
		}
		return parsed
	}

	reload := func() {
		limit := parseLimit(caslLimitEntry.Text)
		lastRows = nil
		statusLabel.SetText("Завантаження звіту...")
		summaryLabel.SetText("Завантаження даних з джерел...")
		exportTSVBtn.Disable()
		exportCSVBtn.Disable()

		go func() {
			result, err := vm.BuildReport(context.Background(), provider, limit, func(stage string) {
				fyne.Do(func() {
					statusLabel.SetText(stage)
				})
			})
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Помилка завантаження")
					exportTSVBtn.Disable()
					exportCSVBtn.Disable()
					dialog.ShowError(err, win)
					return
				}

				lastRows = append([]viewmodels.SIMInventoryReportRow(nil), result.Rows...)
				summaryLabel.SetText(vm.FormatSummary(result))
				exportTSVBtn.Enable()
				exportCSVBtn.Enable()
				statusLabel.SetText(vm.FormatReadyStatus(result))
			})
		}()
	}

	refreshBtn := makePrimaryButton("Оновити", reload)
	exportTSVBtn = makeIconButton("Експорт TSV", iconExport(), widget.MediumImportance, func() {
		if len(lastRows) == 0 {
			ShowInfoDialog(win, "Немає даних", "Спочатку завантажте звіт.")
			return
		}
		dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uc == nil {
				return
			}
			defer uc.Close()

			statusLabel.SetText("Формую TSV для експорту...")
			if _, err := uc.Write([]byte(vm.BuildTSV(lastRows))); err != nil {
				dialog.ShowError(err, win)
				return
			}
			statusLabel.SetText(fmt.Sprintf("TSV експортовано: %s", uriPathToLocalPath(uc.URI().Path())))
		}, win).Show()
	})
	exportTSVBtn.Disable()

	exportCSVBtn = makeIconButton("Експорт CSV", iconExport(), widget.MediumImportance, func() {
		if len(lastRows) == 0 {
			ShowInfoDialog(win, "Немає даних", "Спочатку завантажте звіт.")
			return
		}
		dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uc == nil {
				return
			}
			defer uc.Close()

			statusLabel.SetText("Формую CSV для експорту...")
			if _, err := uc.Write([]byte(vm.BuildCSV(lastRows))); err != nil {
				dialog.ShowError(err, win)
				return
			}
			statusLabel.SetText(fmt.Sprintf("CSV експортовано: %s", uriPathToLocalPath(uc.URI().Path())))
		}, win).Show()
	})
	exportCSVBtn.Disable()

	closeBtn := makeIconButton("Закрити", iconClose(), widget.LowImportance, func() { win.Close() })

	controlsRow := container.NewHBox(
		widget.NewLabel("CASL limit:"),
		container.NewGridWrap(fyne.NewSize(120, 36), caslLimitEntry),
		container.NewGridWrap(fyne.NewSize(120, 36), refreshBtn),
		container.NewGridWrap(fyne.NewSize(120, 36), exportTSVBtn),
		container.NewGridWrap(fyne.NewSize(120, 36), exportCSVBtn),
		layout.NewSpacer(),
	)
	statusRow := container.NewBorder(nil, nil, widget.NewLabel("Стан:"), nil, statusLabel)
	formatNote := "Загальний звіт по БД/МІСТ, Phoenix і CASL. Один рядок = один об'єкт, окремі колонки для SIM 1 та SIM 2."
	if !provider.SupportsCASLReports() {
		formatNote = "Загальний звіт по БД/МІСТ і Phoenix. CASL зараз недоступний, тому CASL-рядки не підтягуються."
	}
	top := container.NewVBox(
		widget.NewCard("Параметри", "", container.NewVBox(controlsRow, statusRow)),
		widget.NewCard("Зведення", "", summaryLabel),
		widget.NewCard("Формат", "", widget.NewLabel(formatNote)),
		widget.NewCard(
			"Експорт",
			"",
			widget.NewLabel("Перегляд у вікні вимкнений. Після завершення збору даних звіт можна зберегти через кнопки «Експорт TSV» або «Експорт CSV»."),
		),
	)

	content := container.NewBorder(
		top,
		container.NewHBox(layout.NewSpacer(), closeBtn),
		nil,
		nil,
		widget.NewLabel(""),
	)
	win.SetContent(content)

	reload()
	win.Show()
}
