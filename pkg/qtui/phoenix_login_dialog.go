//go:build qt

package qtui

import (
	"context"
	"strconv"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
)

// ShowPhoenixLoginDialog asks for the Phoenix workstation identity and saves it.
func ShowPhoenixLoginDialog(
	parent *qt.QWidget,
	prefs config.Preferences,
	onSaved func(config.DBConfig),
) {
	if prefs == nil {
		return
	}
	cfg := config.LoadDBConfig(prefs)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	metadata, err := data.LoadPhoenixRuntimeMetadata(ctx, cfg)
	cancel()
	if err != nil {
		qt.QMessageBox_Critical(parent, "Вхід Phoenix", "Не вдалося завантажити користувачів: "+err.Error())
		return
	}

	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Вхід у Phoenix")
	dialog.SetMinimumWidth(460)
	layout := qt.NewQVBoxLayout(dialog.QWidget)

	intro := qt.NewQLabel3("Оберіть обліковий запис Phoenix. Дані буде збережено для автоматичного входу.")
	intro.SetWordWrap(true)
	layout.AddWidget(intro.QWidget)

	form := qt.NewQFormLayout2()
	host := lineEdit()
	host.SetText(cfg.PhoenixControlHost)
	host.SetPlaceholderText("IP або DNS Phoenix Control Center")
	form.AddRow3("Центр керування", host.QWidget)

	role := qt.NewQComboBox2()
	role.AddItems([]string{qtPhoenixRoleDutyLabel, qtPhoenixRoleAdminLabel})
	if config.NormalizePhoenixClientRole(cfg.PhoenixClientRole) == config.PhoenixClientRoleAdministrator {
		role.SetCurrentText(qtPhoenixRoleAdminLabel)
	}
	form.AddRow3("Роль клієнта", role.QWidget)

	operator := qt.NewQComboBox2()
	operatorsByLabel := make(map[string]data.PhoenixOperator, len(metadata.Operators))
	selectedIndex := -1
	for _, item := range metadata.Operators {
		label := item.DisplayName()
		operatorsByLabel[label] = item
		operator.AddItem(label)
		if item.ID == cfg.PhoenixOperatorID {
			selectedIndex = operator.Count() - 1
		}
	}
	if selectedIndex >= 0 {
		operator.SetCurrentIndex(selectedIndex)
	}
	form.AddRow3("Користувач", operator.QWidget)

	password := passwordEdit()
	password.SetText(cfg.PhoenixOperatorPassword)
	form.AddRow3("Пароль", password.QWidget)
	layout.AddLayout(form.QLayout)

	ports := qt.NewQLabel3(
		"Порти з БД: Duty Operator " + strconv.Itoa(metadata.ClientPort) +
			", Administrator " + strconv.Itoa(metadata.AdminPort),
	)
	ports.SetWordWrap(true)
	layout.AddWidget(ports.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(func() { dialog.Accept() })
	buttons.OnRejected(func() { dialog.Reject() })
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	for dialog.Exec() == int(qt.QDialog__Accepted) {
		selected := operatorsByLabel[operator.CurrentText()]
		if selected.ID <= 0 || strings.TrimSpace(password.Text()) == "" || strings.TrimSpace(host.Text()) == "" {
			qt.QMessageBox_Information(parent, "Вхід Phoenix", "Вкажіть центр керування, користувача та пароль.")
			continue
		}
		cfg.PhoenixControlHost = strings.TrimSpace(host.Text())
		cfg.PhoenixOperatorID = selected.ID
		cfg.PhoenixOperatorName = strings.TrimSpace(selected.Login)
		if cfg.PhoenixOperatorName == "" {
			cfg.PhoenixOperatorName = strings.TrimSpace(selected.Name)
		}
		cfg.PhoenixOperatorPassword = password.Text()
		cfg.PhoenixClientRole = config.PhoenixClientRoleDuty
		if role.CurrentText() == qtPhoenixRoleAdminLabel {
			cfg.PhoenixClientRole = config.PhoenixClientRoleAdministrator
		}

		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		err := data.ValidatePhoenixOperatorCredentials(ctx, cfg)
		cancel()
		if err != nil {
			qt.QMessageBox_Critical(parent, "Вхід Phoenix", err.Error())
			continue
		}
		config.SaveDBConfig(prefs, cfg)
		if onSaved != nil {
			onSaved(cfg)
		}
		return
	}
}
