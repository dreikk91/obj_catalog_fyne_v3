package dialogs

import (
	"obj_catalog_fyne_v3/pkg/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowDatabaseSettingsDialog(win fyne.Window, pref fyne.Preferences, onSave func(config.DBConfig)) {
	cfg := config.LoadDBConfig(pref)

	userEntry := widget.NewEntry()
	userEntry.SetText(cfg.User)
	userEntry.SetPlaceHolder("SYSDBA")

	passEntry := widget.NewPasswordEntry()
	passEntry.SetText(cfg.Password)
	passEntry.SetPlaceHolder("masterkey")

	hostEntry := widget.NewEntry()
	hostEntry.SetText(cfg.Host)
	hostEntry.SetPlaceHolder("127.0.0.1")

	portEntry := widget.NewEntry()
	portEntry.SetText(cfg.Port)
	portEntry.SetPlaceHolder("3050")

	pathEntry := widget.NewEntry()
	pathEntry.SetText(cfg.Path)
	pathEntry.SetPlaceHolder("C:/DB/DATA.FDB")

	paramsEntry := widget.NewEntry()
	paramsEntry.SetText(cfg.Params)
	paramsEntry.SetPlaceHolder("charset=WIN1251")

	form := widget.NewForm(
		widget.NewFormItem("Користувач", userEntry),
		widget.NewFormItem("Пароль", passEntry),
		widget.NewFormItem("Хост", hostEntry),
		widget.NewFormItem("Порт", portEntry),
		widget.NewFormItem("Шлях до БД", pathEntry),
		widget.NewFormItem("Параметри", paramsEntry),
	)

	d := dialog.NewCustomConfirm(
		"Налаштування підключення",
		"Зберегти",
		"Скасувати",
		container.NewVBox(
			widget.NewLabel("Параметри підключення до Firebird"),
			form,
		),
		func(save bool) {
			if save {
				newCfg := config.DBConfig{
					User:     userEntry.Text,
					Password: passEntry.Text,
					Host:     hostEntry.Text,
					Port:     portEntry.Text,
					Path:     pathEntry.Text,
					Params:   paramsEntry.Text,
				}
				config.SaveDBConfig(pref, newCfg)
				if onSave != nil {
					onSave(newCfg)
				}
			}
		},
		win,
	)

	d.Resize(fyne.NewSize(450, 400))
	d.Show()
}
