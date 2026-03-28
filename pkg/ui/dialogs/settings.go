package dialogs

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/config"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowSettingsDialog(
	win fyne.Window,
	pref fyne.Preferences,
	isDarkTheme bool,
	onSave func(config.DBConfig, config.UIConfig),
	onColorsChanged func(),
) {
	dbCfg := config.LoadDBConfig(pref)
	uiCfg := config.LoadUIConfig(pref)

	// Database fields
	userEntry := widget.NewEntry()
	userEntry.SetText(dbCfg.User)
	passEntry := widget.NewPasswordEntry()
	passEntry.SetText(dbCfg.Password)
	hostEntry := widget.NewEntry()
	hostEntry.SetText(dbCfg.Host)
	portEntry := widget.NewEntry()
	portEntry.SetText(dbCfg.Port)
	pathEntry := widget.NewEntry()
	pathEntry.SetText(dbCfg.Path)
	paramsEntry := widget.NewEntry()
	paramsEntry.SetText(dbCfg.Params)

	// UI fields
	fontEntry := widget.NewEntry()
	fontEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSize))
	fontObjEntry := widget.NewEntry()
	fontObjEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSizeObjects))
	fontEvEntry := widget.NewEntry()
	fontEvEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSizeEvents))
	fontAlmEntry := widget.NewEntry()
	fontAlmEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSizeAlarms))

	eventLimitEntry := widget.NewEntry()
	eventLimitEntry.SetText(strconv.Itoa(uiCfg.EventLogLimit))
	eventLimitEntry.SetPlaceHolder("2000")

	objectLimitEntry := widget.NewEntry()
	objectLimitEntry.SetText(strconv.Itoa(uiCfg.ObjectLogLimit))
	objectLimitEntry.SetPlaceHolder("0 = без обмеження")

	exportDirEntry := widget.NewEntry()
	exportDirEntry.SetText(uiCfg.ExportDir)
	exportDirEntry.SetPlaceHolder("Папка запуску програми")

	browseExportDirBtn := makeIconButton("Обрати...", iconFolder(), widget.MediumImportance, func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uri == nil {
				return
			}
			exportDirEntry.SetText(uriPathToLocalPath(uri.Path()))
		}, win).Show()
	})

	clearExportDirBtn := makeIconButton("Очистити", iconClear(), widget.LowImportance, func() {
		exportDirEntry.SetText("")
	})

	exportDirRow := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(browseExportDirBtn, clearExportDirBtn),
		exportDirEntry,
	)

	colorsBtn := makeIconButton("Налаштувати кольори...", iconSearch(), widget.LowImportance, func() {
		ShowColorPaletteDialog(win, isDarkTheme, onColorsChanged)
	})

	tabs := container.NewAppTabs(
		container.NewTabItem("База даних", widget.NewForm(
			widget.NewFormItem("Користувач", userEntry),
			widget.NewFormItem("Пароль", passEntry),
			widget.NewFormItem("Хост", hostEntry),
			widget.NewFormItem("Порт", portEntry),
			widget.NewFormItem("Шлях до БД", pathEntry),
			widget.NewFormItem("Параметри", paramsEntry),
		)),
		container.NewTabItem("Інтерфейс", widget.NewForm(
			widget.NewFormItem("Загальний шрифт", fontEntry),
			widget.NewFormItem("Шрифт об'єктів", fontObjEntry),
			widget.NewFormItem("Шрифт подій", fontEvEntry),
			widget.NewFormItem("Шрифт тривог", fontAlmEntry),
			widget.NewFormItem("Ліміт загального журналу", eventLimitEntry),
			widget.NewFormItem("Ліміт журналу об'єкта", objectLimitEntry),
			widget.NewFormItem("Папка експорту", exportDirRow),
			widget.NewFormItem("Кольори подій/об'єктів", colorsBtn),
		)),
	)

	d := dialog.NewCustomConfirm(
		"Налаштування системи",
		"Зберегти",
		"Скасувати",
		tabs,
		func(save bool) {
			if save {
				newDbCfg := config.DBConfig{
					User:     userEntry.Text,
					Password: passEntry.Text,
					Host:     hostEntry.Text,
					Port:     portEntry.Text,
					Path:     pathEntry.Text,
					Params:   paramsEntry.Text,
				}

				fSize, _ := strconv.ParseFloat(fontEntry.Text, 32)
				fObjSize, _ := strconv.ParseFloat(fontObjEntry.Text, 32)
				fEvSize, _ := strconv.ParseFloat(fontEvEntry.Text, 32)
				fAlmSize, _ := strconv.ParseFloat(fontAlmEntry.Text, 32)
				evLimit, _ := strconv.Atoi(strings.TrimSpace(eventLimitEntry.Text))
				objLimit, _ := strconv.Atoi(strings.TrimSpace(objectLimitEntry.Text))

				newUiCfg := config.UIConfig{
					FontSize:        float32(fSize),
					FontSizeObjects: float32(fObjSize),
					FontSizeEvents:  float32(fEvSize),
					FontSizeAlarms:  float32(fAlmSize),
					ExportDir:       strings.TrimSpace(exportDirEntry.Text),
					EventLogLimit:   evLimit,
					ObjectLogLimit:  objLimit,
				}

				config.SaveDBConfig(pref, newDbCfg)
				config.SaveUIConfig(pref, newUiCfg)

				if onSave != nil {
					onSave(newDbCfg, newUiCfg)
				}
			}
		},
		win,
	)

	d.Resize(fyne.NewSize(560, 520))
	d.Show()
}

func uriPathToLocalPath(path string) string {
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		return path[1:]
	}
	return path
}
