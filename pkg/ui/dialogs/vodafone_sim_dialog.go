package dialogs

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

func ShowVodafoneSIMDialog(
	parent fyne.Window,
	provider contracts.AdminObjectVodafoneService,
	msisdn string,
	objectNumber int,
	objectName string,
) {
	msisdn = strings.TrimSpace(msisdn)
	if parent == nil {
		return
	}
	if msisdn == "" {
		ShowInfoDialog(parent, "Vodafone", "SIM номер не вказаний.")
		return
	}
	if provider == nil {
		ShowInfoDialog(parent, "Vodafone", "Vodafone сервіс недоступний.")
		return
	}

	vm := viewmodels.NewVodafoneSIMViewModel()
	statusLabel := widget.NewLabel("Vodafone: перевірка за запитом")
	statusLabel.Wrapping = fyne.TextWrapWord

	titleLabel := widget.NewLabel(fmt.Sprintf("SIM: %s", msisdn))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	objectLabel := widget.NewLabel(fmt.Sprintf("Об'єкт: #%d %s", objectNumber, strings.TrimSpace(objectName)))
	objectLabel.Wrapping = fyne.TextWrapWord

	setBusy := func(busy bool, controls ...fyne.Disableable) {
		for _, control := range controls {
			if control == nil {
				continue
			}
			if busy {
				control.Disable()
			} else {
				control.Enable()
			}
		}
	}

	var (
		refreshBtn *widget.Button
		rebootBtn  *widget.Button
		syncBtn    *widget.Button
	)

	runAction := func(startedText string, action func() (string, error)) {
		statusLabel.SetText(startedText)
		setBusy(true, refreshBtn, rebootBtn, syncBtn)
		go func() {
			text, err := action()
			fyne.Do(func() {
				setBusy(false, refreshBtn, rebootBtn, syncBtn)
				if err != nil {
					statusLabel.SetText(err.Error())
					return
				}
				statusLabel.SetText(text)
			})
		}()
	}

	refreshBtn = widget.NewButton("Статус і подія", func() {
		runAction("Vodafone: перевірка стану...", func() (string, error) {
			status, err := provider.GetVodafoneSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildStatusText(status), nil
		})
	})

	rebootBtn = widget.NewButton("Перезавантажити SIM", func() {
		runAction("Vodafone: створення заявки...", func() (string, error) {
			result, err := provider.RebootVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(result.OrderID) == "" {
				return "Vodafone: заявку на перезавантаження створено", nil
			}
			if strings.TrimSpace(result.State) == "" {
				return "Vodafone: заявку створено, ID " + result.OrderID, nil
			}
			return "Vodafone: заявку створено, ID " + result.OrderID + ", стан " + result.State, nil
		})
	})

	syncBtn = widget.NewButton("Записати №/назву", func() {
		runAction("Vodafone: запис name/comment...", func() (string, error) {
			name, comment, err := vm.BuildMetadata(
				msisdn,
				strconv.Itoa(objectNumber),
				objectName,
				objectName,
			)
			if err != nil {
				return "", err
			}
			if err := provider.UpdateVodafoneSIMMetadata(msisdn, name, comment); err != nil {
				return "", err
			}
			return "Vodafone: name/comment оновлено", nil
		})
	})

	content := container.NewVBox(
		titleLabel,
		objectLabel,
		widget.NewSeparator(),
		container.NewHBox(refreshBtn, rebootBtn, syncBtn),
		statusLabel,
	)

	dlg := dialog.NewCustom("Vodafone запити", "Закрити", content, parent)
	dlg.Resize(fyne.NewSize(560, 220))
	dlg.Show()
}
