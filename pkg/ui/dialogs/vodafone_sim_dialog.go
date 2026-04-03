package dialogs

import (
	"fmt"
	"strings"
	"time"

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
	objectNumber string,
	objectName string,
) {
	msisdn = strings.TrimSpace(msisdn)
	objectNumber = strings.TrimSpace(objectNumber)
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
	objectLabel := widget.NewLabel(fmt.Sprintf("Об'єкт: #%s %s", objectNumber, strings.TrimSpace(objectName)))
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
		refreshBtn           *widget.Button
		rebootBtn            *widget.Button
		syncBtn              *widget.Button
		blockBtn             *widget.Button
		unblockBtn           *widget.Button
		reasonSelect         *widget.Select
		customReasonEntry    *widget.Entry
		setManualReasonState func(string)
	)

	runAction := func(startedText string, action func() (string, error)) {
		statusLabel.SetText(startedText)
		setBusy(true, refreshBtn, rebootBtn, syncBtn, blockBtn, unblockBtn, reasonSelect, customReasonEntry)
		go func() {
			text, err := action()
			fyne.Do(func() {
				setBusy(false, refreshBtn, rebootBtn, syncBtn, blockBtn, unblockBtn, reasonSelect, customReasonEntry)
				setManualReasonState(reasonSelect.Selected)
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

	reasons := vm.BlockingReasonOptions()
	reasonSelect = widget.NewSelect(reasons, nil)
	customReasonEntry = widget.NewEntry()
	customReasonEntry.SetPlaceHolder("Вкажіть причину вручну")
	setManualReasonState = func(selected string) {
		if vm.IsManualBlockingReason(selected) {
			customReasonEntry.Enable()
			return
		}
		customReasonEntry.Disable()
	}
	reasonSelect.OnChanged = func(selected string) {
		setManualReasonState(selected)
	}
	if len(reasons) > 0 {
		reasonSelect.SetSelected(reasons[0])
	}
	setManualReasonState(reasonSelect.Selected)

	syncBtn = widget.NewButton("Записати №/назву", func() {
		runAction("Vodafone: запис name/comment...", func() (string, error) {
			name, comment, err := vm.BuildMetadata(
				msisdn,
				objectNumber,
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

	blockBtn = widget.NewButton("Блокувати номер", func() {
		runAction("Vodafone: блокування номера...", func() (string, error) {
			name, comment, err := vm.BuildBlockingMetadata(objectNumber, reasonSelect.Selected, customReasonEntry.Text, time.Now())
			if err != nil {
				return "", err
			}
			if err := provider.UpdateVodafoneSIMMetadata(msisdn, name, comment); err != nil {
				return "", err
			}
			result, err := provider.BlockVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildBarringResultText(result), nil
		})
	})

	unblockBtn = widget.NewButton("Розблокувати номер", func() {
		runAction("Vodafone: розблокування номера...", func() (string, error) {
			result, err := provider.UnblockVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildBarringResultText(result), nil
		})
	})

	content := container.NewVBox(
		titleLabel,
		objectLabel,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Причина блокування:", reasonSelect),
			widget.NewFormItem("Своя причина:", customReasonEntry),
		),
		container.NewHBox(refreshBtn, rebootBtn, syncBtn),
		container.NewHBox(blockBtn, unblockBtn),
		statusLabel,
	)

	dlg := dialog.NewCustom("Vodafone запити", "Закрити", content, parent)
	dlg.Resize(fyne.NewSize(640, 280))
	dlg.Show()
}
