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
	resultLabel := widget.NewLabel("Vodafone: перевірка за запитом")
	resultLabel.Wrapping = fyne.TextWrapWord

	titleLabel := widget.NewLabel(fmt.Sprintf("SIM: %s", msisdn))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	objectLabel := widget.NewLabel(fmt.Sprintf("Об'єкт: #%s %s", objectNumber, strings.TrimSpace(objectName)))
	objectLabel.Wrapping = fyne.TextWrapWord

	overviewLabel := widget.NewLabel("Стан ще не завантажено.")
	overviewLabel.TextStyle = fyne.TextStyle{Bold: true}
	overviewLabel.Wrapping = fyne.TextWrapWord

	connectivityLabel := widget.NewLabel("Натисніть \"Статус\", щоб отримати актуальні дані.")
	connectivityLabel.Wrapping = fyne.TextWrapWord

	identityLabel := widget.NewLabel("Назва та коментар з'являться тут.")
	identityLabel.Wrapping = fyne.TextWrapWord

	eventLabel := widget.NewLabel("Остання подія ще не завантажена.")
	eventLabel.Wrapping = fyne.TextWrapWord

	blockingLabel := widget.NewLabel("Стан блокування ще не завантажено.")
	blockingLabel.Wrapping = fyne.TextWrapWord

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

	applyStatus := func(status contracts.VodafoneSIMStatus) {
		overviewLabel.SetText(vm.BuildOverviewText(status))
		connectivityLabel.SetText(vm.BuildConnectivityText(status))
		identityLabel.SetText(vm.BuildIdentityText(status))
		eventLabel.SetText(vm.BuildEventText(status))
		blockingLabel.SetText(vm.BuildBlockingText(status))
	}

	runAction := func(startedText string, action func() (string, error)) {
		resultLabel.SetText(startedText)
		setBusy(true, refreshBtn, rebootBtn, syncBtn, blockBtn, unblockBtn, reasonSelect, customReasonEntry)
		go func() {
			text, err := action()
			fyne.Do(func() {
				setBusy(false, refreshBtn, rebootBtn, syncBtn, blockBtn, unblockBtn, reasonSelect, customReasonEntry)
				setManualReasonState(reasonSelect.Selected)
				if err != nil {
					resultLabel.SetText(err.Error())
					return
				}
				resultLabel.SetText(text)
			})
		}()
	}

	refreshBtn = makeIconButton("Статус", iconRefresh(), widget.MediumImportance, func() {
		runAction("Vodafone: перевірка стану...", func() (string, error) {
			status, err := provider.GetVodafoneSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			fyne.DoAndWait(func() {
				applyStatus(status)
			})
			return vm.BuildStatusText(status), nil
		})
	})

	rebootBtn = makeIconButton("Reset SIM", iconRefresh(), widget.MediumImportance, func() {
		runAction("Vodafone: створення заявки...", func() (string, error) {
			result, err := provider.RebootVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildRebootResultText(result), nil
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

	syncBtn = makeIconButton("Записати дані", iconEdit(), widget.LowImportance, func() {
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

	blockBtn = makeIconButton("Блокувати номер", iconClose(), widget.DangerImportance, func() {
		showSIMNumberActionConfirm(
			parent,
			"Підтвердити блокування",
			fmt.Sprintf("Заблокувати номер %s для об'єкта #%s?", msisdn, objectNumber),
			func() {
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
			},
		)
	})

	unblockBtn = makeIconButton("Розблокувати номер", iconAdd(), widget.HighImportance, func() {
		showSIMNumberActionConfirm(
			parent,
			"Підтвердити розблокування",
			fmt.Sprintf("Розблокувати номер %s для об'єкта #%s?", msisdn, objectNumber),
			func() {
				runAction("Vodafone: розблокування номера...", func() (string, error) {
					result, err := provider.UnblockVodafoneSIM(msisdn)
					if err != nil {
						return "", err
					}
					return vm.BuildBarringResultText(result), nil
				})
			},
		)
	})

	reasonCard := widget.NewCard(
		"Параметри блокування",
		"",
		widget.NewForm(
			widget.NewFormItem("Причина блокування:", reasonSelect),
			widget.NewFormItem("Своя причина:", customReasonEntry),
		),
	)

	headerCard := widget.NewCard(
		"Номер та об'єкт",
		"",
		container.NewVBox(titleLabel, objectLabel),
	)

	stateCard := widget.NewCard(
		"Стан",
		"",
		container.NewVBox(overviewLabel, connectivityLabel),
	)

	identityCard := widget.NewCard("Назва та коментар", "", identityLabel)
	eventCard := widget.NewCard("Остання подія", "", eventLabel)
	blockingCard := widget.NewCard("Блокування", "", blockingLabel)

	actionsCard := widget.NewCard(
		"Дії",
		"",
		container.NewVBox(
			container.NewGridWithColumns(3, refreshBtn, rebootBtn, syncBtn),
			container.NewGridWithColumns(2, blockBtn, unblockBtn),
		),
	)

	resultCard := widget.NewCard("Результат", "", resultLabel)

	content := container.NewVBox(
		headerCard,
		container.NewGridWithColumns(3, stateCard, identityCard, eventCard),
		blockingCard,
		actionsCard,
		reasonCard,
		resultCard,
	)

	scrollContent := container.NewVScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(780, 560))

	dlg := dialog.NewCustom("Vodafone запити", "Закрити", scrollContent, parent)
	dlg.Resize(fyne.NewSize(840, 620))
	dlg.Show()
}
