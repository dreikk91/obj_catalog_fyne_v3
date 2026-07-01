package dialogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneDialog "fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
)

// ShowPhoenixLoginDialog asks for a Phoenix identity and saves it for autologin.
func ShowPhoenixLoginDialog(
	parent fyne.Window,
	prefs fyne.Preferences,
	onSaved func(config.DBConfig),
) {
	if parent == nil || prefs == nil {
		return
	}
	cfg := config.LoadDBConfig(prefs)
	host := widget.NewEntry()
	host.SetText(cfg.PhoenixControlHost)
	host.SetPlaceHolder("IP або DNS Phoenix Control Center")

	role := widget.NewSelect([]string{phoenixRoleDutyLabel, phoenixRoleAdminLabel}, nil)
	if config.NormalizePhoenixClientRole(cfg.PhoenixClientRole) == config.PhoenixClientRoleAdministrator {
		role.SetSelected(phoenixRoleAdminLabel)
	} else {
		role.SetSelected(phoenixRoleDutyLabel)
	}
	operator := widget.NewSelect(nil, nil)
	operator.PlaceHolder = "Завантаження користувачів..."
	password := widget.NewPasswordEntry()
	password.SetText(cfg.PhoenixOperatorPassword)
	status := widget.NewLabel("Завантаження користувачів і портів із Phoenix...")
	status.Wrapping = fyne.TextWrapWord
	operators := make(map[string]data.PhoenixOperator)

	loginButton := widget.NewButton("Увійти і запам'ятати", nil)
	loginButton.Disable()
	cancelButton := widget.NewButton("Не зараз", nil)
	content := container.NewVBox(
		widget.NewLabel("Оберіть обліковий запис Phoenix. Наступного разу вхід відбудеться автоматично."),
		widget.NewForm(
			widget.NewFormItem("Центр керування", host),
			widget.NewFormItem("Роль клієнта", role),
			widget.NewFormItem("Користувач", operator),
			widget.NewFormItem("Пароль", password),
		),
		status,
		container.NewHBox(loginButton, cancelButton),
	)
	custom := fyneDialog.NewCustomWithoutButtons("Вхід у Phoenix", content, parent)
	cancelButton.OnTapped = custom.Hide

	loginButton.OnTapped = func() {
		selected := operators[operator.Selected]
		if selected.ID <= 0 || strings.TrimSpace(host.Text) == "" || strings.TrimSpace(password.Text) == "" {
			status.SetText("Вкажіть центр керування, користувача та пароль.")
			return
		}
		loginButton.Disable()
		status.SetText("Перевірка облікового запису Phoenix...")
		candidate := cfg
		candidate.PhoenixControlHost = strings.TrimSpace(host.Text)
		candidate.PhoenixOperatorID = selected.ID
		candidate.PhoenixOperatorName = strings.TrimSpace(selected.Login)
		if candidate.PhoenixOperatorName == "" {
			candidate.PhoenixOperatorName = strings.TrimSpace(selected.Name)
		}
		candidate.PhoenixOperatorPassword = password.Text
		candidate.PhoenixClientRole = config.PhoenixClientRoleDuty
		if role.Selected == phoenixRoleAdminLabel {
			candidate.PhoenixClientRole = config.PhoenixClientRoleAdministrator
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			err := data.ValidatePhoenixOperatorCredentials(ctx, candidate)
			cancel()
			fyne.Do(func() {
				if err != nil {
					status.SetText(err.Error())
					loginButton.Enable()
					return
				}
				config.SaveDBConfig(prefs, candidate)
				custom.Hide()
				if onSaved != nil {
					onSaved(candidate)
				}
			})
		}()
	}

	custom.Show()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		metadata, err := data.LoadPhoenixRuntimeMetadata(ctx, cfg)
		cancel()
		fyne.Do(func() {
			if err != nil {
				status.SetText("Не вдалося завантажити користувачів: " + err.Error())
				return
			}
			options := make([]string, 0, len(metadata.Operators))
			selected := ""
			for _, item := range metadata.Operators {
				label := item.DisplayName()
				operators[label] = item
				options = append(options, label)
				if item.ID == cfg.PhoenixOperatorID {
					selected = label
				}
			}
			operator.Options = options
			operator.Refresh()
			if selected != "" {
				operator.SetSelected(selected)
			} else if len(options) > 0 {
				operator.SetSelected(options[0])
			}
			status.SetText(fmt.Sprintf(
				"Порти з БД: Duty Operator %d, Administrator %d",
				metadata.ClientPort,
				metadata.AdminPort,
			))
			if len(options) > 0 {
				loginButton.Enable()
			}
		})
	}()
}
