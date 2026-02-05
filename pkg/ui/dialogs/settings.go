package dialogs

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/config"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowSettingsDialog(win fyne.Window, pref fyne.Preferences, onSave func(config.DBConfig, config.UIConfig)) {
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

				newUiCfg := config.UIConfig{
					FontSize:        float32(fSize),
					FontSizeObjects: float32(fObjSize),
					FontSizeEvents:  float32(fEvSize),
					FontSizeAlarms:  float32(fAlmSize),
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

	d.Resize(fyne.NewSize(500, 450))
	d.Show()
}
