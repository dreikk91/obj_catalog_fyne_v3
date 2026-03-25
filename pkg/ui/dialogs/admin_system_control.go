package dialogs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/data"
)

func ShowAdminSystemControlDialog(parent fyne.Window, provider data.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Контроль системи")
	win.Resize(fyne.NewSize(1120, 720))

	var (
		issues       []data.AdminDataCheckIssue
		accessStatus data.AdminAccessStatus
	)

	statusLabel := widget.NewLabel("Готово")
	accessLabel := widget.NewLabel("Доступ: перевірка...")
	accessLabel.Wrapping = fyne.TextWrapWord

	issueFilterEntry := widget.NewEntry()
	issueFilterEntry.SetPlaceHolder("Фільтр перевірок: код / № об'єкта / текст")

	filteredIssues := func() []data.AdminDataCheckIssue {
		filter := strings.ToLower(strings.TrimSpace(issueFilterEntry.Text))
		if filter == "" {
			return issues
		}
		out := make([]data.AdminDataCheckIssue, 0, len(issues))
		for _, it := range issues {
			objnText := ""
			if it.ObjN > 0 {
				objnText = strconv.FormatInt(it.ObjN, 10)
			}
			full := strings.ToLower(strings.Join([]string{
				it.Severity,
				it.Code,
				objnText,
				it.Details,
			}, " "))
			if strings.Contains(full, filter) {
				out = append(out, it)
			}
		}
		return out
	}

	issueTable := widget.NewTable(
		func() (int, int) { return len(filteredIssues()) + 1, 4 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("Рівень")
				case 1:
					lbl.SetText("Код")
				case 2:
					lbl.SetText("№пр.")
				default:
					lbl.SetText("Опис")
				}
				return
			}
			rows := filteredIssues()
			idx := id.Row - 1
			if idx < 0 || idx >= len(rows) {
				lbl.SetText("")
				return
			}
			it := rows[idx]
			switch id.Col {
			case 0:
				switch strings.ToLower(strings.TrimSpace(it.Severity)) {
				case "error":
					lbl.SetText("Помилка")
				case "warn":
					lbl.SetText("Попередження")
				default:
					lbl.SetText(strings.TrimSpace(it.Severity))
				}
			case 1:
				lbl.SetText(strings.TrimSpace(it.Code))
			case 2:
				if it.ObjN > 0 {
					lbl.SetText(strconv.FormatInt(it.ObjN, 10))
				} else {
					lbl.SetText("—")
				}
			default:
				lbl.SetText(strings.TrimSpace(it.Details))
			}
		},
	)
	issueTable.SetColumnWidth(0, 130)
	issueTable.SetColumnWidth(1, 170)
	issueTable.SetColumnWidth(2, 90)
	issueTable.SetColumnWidth(3, 680)

	logSourceSelect := widget.NewSelect([]string{
		"log/app.log",
		"log/error.log",
	}, nil)
	logSourceSelect.SetSelected("log/app.log")

	logFilterEntry := widget.NewEntry()
	logFilterEntry.SetPlaceHolder("Фільтр логу (текст)")
	logTailCountEntry := widget.NewEntry()
	logTailCountEntry.SetText("300")
	logTailCountEntry.SetPlaceHolder("К-сть рядків")

	logText := widget.NewTextGrid()
	logText.SetText("")
	logScroll := container.NewScroll(logText)
	logScroll.SetMinSize(fyne.NewSize(0, 420))
	currentLogContent := ""

	readLogContent := func(path string, tail int) (string, error) {
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		lines := make([]string, 0, tail)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)
			if len(lines) > tail {
				lines = lines[1:]
			}
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}

		filter := strings.ToLower(strings.TrimSpace(logFilterEntry.Text))
		if filter == "" {
			return strings.Join(lines, "\n"), nil
		}
		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), filter) {
				filtered = append(filtered, line)
			}
		}
		return strings.Join(filtered, "\n"), nil
	}

	resolveLogPath := func(selected string) string {
		selected = strings.TrimSpace(selected)
		if selected == "" {
			selected = "log/app.log"
		}
		return filepath.Clean(selected)
	}

	reloadLogs := func() {
		selected := resolveLogPath(logSourceSelect.Selected)
		tail := int64(300)
		if raw := strings.TrimSpace(logTailCountEntry.Text); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				if parsed > 0 && parsed <= 5000 {
					tail = parsed
				}
			}
		}
		content, err := readLogContent(selected, int(tail))
		if err != nil {
			currentLogContent = ""
			logText.SetText("")
			statusLabel.SetText(fmt.Sprintf("Не вдалося прочитати %s", selected))
			return
		}
		currentLogContent = content
		logText.SetText(content)
		statusLabel.SetText(fmt.Sprintf("Лог завантажено: %s", selected))
	}

	reloadChecks := func() {
		st, err := provider.GetAdminAccessStatus()
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка перевірки прав доступу")
			return
		}
		accessStatus = st
		accessText := fmt.Sprintf("Користувач: %s | Адмін у PERSONAL: %d | Доступ: %s",
			blankFallback(strings.TrimSpace(accessStatus.CurrentUser), "невизначено"),
			accessStatus.AdminUsersCount,
			boolToAccessLabel(accessStatus.HasFullAccess),
		)
		if strings.TrimSpace(accessStatus.MatchedPersonal) != "" {
			accessText += fmt.Sprintf(" | Match: %s", accessStatus.MatchedPersonal)
		}
		if strings.TrimSpace(accessStatus.MatchDescription) != "" {
			accessText += fmt.Sprintf(" (%s)", accessStatus.MatchDescription)
		}
		accessLabel.SetText(accessText)

		loaded, err := provider.RunDataIntegrityChecks(800)
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка перевірки цілісності БД")
			return
		}
		issues = loaded
		issueTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Перевірки виконано: %d проблем(а/и)", len(issues)))
	}

	exportCurrentTab := func(activeTab string) {
		dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uc == nil {
				return
			}
			defer uc.Close()

			path := uriPathToLocalPath(uc.URI().Path())
			var content string
			switch activeTab {
			case "logs":
				content = currentLogContent
				if strings.TrimSpace(content) == "" {
					content = fmt.Sprintf("Немає даних у %s\n", resolveLogPath(logSourceSelect.Selected))
				}
			default:
				lines := []string{"severity;code;objn;details"}
				for _, it := range filteredIssues() {
					objn := ""
					if it.ObjN > 0 {
						objn = strconv.FormatInt(it.ObjN, 10)
					}
					lines = append(lines, fmt.Sprintf("%s;%s;%s;%s",
						escapeCSVCell(it.Severity),
						escapeCSVCell(it.Code),
						escapeCSVCell(objn),
						escapeCSVCell(it.Details),
					))
				}
				if len(lines) == 1 {
					lines = append(lines, "info;NO_ISSUES;;Проблем не виявлено")
				}
				content = strings.Join(lines, "\n")
			}

			if _, err := uc.Write([]byte(content)); err != nil {
				dialog.ShowError(err, win)
				return
			}
			statusLabel.SetText(fmt.Sprintf("Експортовано: %s", path))
		}, win).Show()
	}

	issueFilterEntry.OnChanged = func(string) {
		issueTable.Refresh()
	}

	logSourceSelect.OnChanged = func(string) {
		reloadLogs()
	}
	logFilterEntry.OnSubmitted = func(string) {
		reloadLogs()
	}
	logTailCountEntry.OnSubmitted = func(string) {
		reloadLogs()
	}

	checksTab := container.NewBorder(
		container.NewVBox(
			accessLabel,
			container.NewBorder(nil, nil, widget.NewLabel("Фільтр:"), nil, issueFilterEntry),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		issueTable,
	)

	logsTab := container.NewBorder(
		container.NewVBox(
			container.NewHBox(
				widget.NewLabel("Файл:"),
				container.NewGridWrap(fyne.NewSize(170, 36), logSourceSelect),
				widget.NewLabel("Рядків:"),
				container.NewGridWrap(fyne.NewSize(90, 36), logTailCountEntry),
				layout.NewSpacer(),
				widget.NewLabel("Фільтр:"),
				container.NewGridWrap(fyne.NewSize(300, 36), logFilterEntry),
			),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		logScroll,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Перевірки БД", checksTab),
		container.NewTabItem("Локальні логи", logsTab),
	)

	refreshBtn := widget.NewButton("Оновити", func() {
		if tabs.SelectedIndex() == 1 {
			reloadLogs()
			return
		}
		reloadChecks()
	})
	exportBtn := widget.NewButton("Експорт", func() {
		active := "checks"
		if tabs.SelectedIndex() == 1 {
			active = "logs"
		}
		exportCurrentTab(active)
	})
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	content := container.NewBorder(
		container.NewHBox(exportBtn, refreshBtn, layout.NewSpacer(), widget.NewLabel(time.Now().Format("02.01.2006"))),
		container.NewHBox(statusLabel, layout.NewSpacer(), closeBtn),
		nil, nil,
		tabs,
	)
	win.SetContent(content)

	reloadChecks()
	reloadLogs()
	win.Show()
}

func boolToAccessLabel(v bool) string {
	if v {
		return "повний"
	}
	return "обмежений"
}

func blankFallback(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func escapeCSVCell(v string) string {
	v = strings.ReplaceAll(v, "\"", "\"\"")
	return fmt.Sprintf("\"%s\"", v)
}
