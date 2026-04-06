package dialogs

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

func ShowKyivstarSIMDialog(
	parent fyne.Window,
	provider contracts.AdminObjectKyivstarService,
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
		ShowInfoDialog(parent, "Kyivstar", "SIM номер не вказаний.")
		return
	}
	if provider == nil {
		ShowInfoDialog(parent, "Kyivstar", "Kyivstar сервіс недоступний.")
		return
	}

	vm := viewmodels.NewKyivstarSIMViewModel()
	resultLabel := widget.NewLabel("Kyivstar: перевірка за запитом")
	resultLabel.Wrapping = fyne.TextWrapWord

	titleLabel := widget.NewLabel(fmt.Sprintf("SIM: %s", msisdn))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	objectLabel := widget.NewLabel(fmt.Sprintf("Об'єкт: #%s %s", objectNumber, strings.TrimSpace(objectName)))
	objectLabel.Wrapping = fyne.TextWrapWord

	overviewLabel := widget.NewLabel("Стан ще не завантажено.")
	overviewLabel.TextStyle = fyne.TextStyle{Bold: true}
	overviewLabel.Wrapping = fyne.TextWrapWord

	stateLabel := widget.NewLabel("Натисніть \"Статус і сервіси\", щоб отримати актуальні дані.")
	stateLabel.Wrapping = fyne.TextWrapWord

	identityLabel := widget.NewLabel("Ідентифікатори та назва пристрою з'являться тут.")
	identityLabel.Wrapping = fyne.TextWrapWord

	usageLabel := widget.NewLabel("Статистика використання ще не завантажена.")
	usageLabel.Wrapping = fyne.TextWrapWord

	var (
		refreshBtn          *widget.Button
		rebootBtn           *widget.Button
		syncBtn             *widget.Button
		pauseBtn            *widget.Button
		activateBtn         *widget.Button
		pauseServicesBtn    *widget.Button
		activateServicesBtn *widget.Button
	)

	serviceChecks := make(map[string]*widget.Check)
	serviceBox := container.NewVBox(widget.NewLabel("Після перевірки стану тут з'являться сервіси номера."))
	serviceScroll := container.NewVScroll(serviceBox)
	serviceScroll.SetMinSize(fyne.NewSize(0, 140))

	applyStatus := func(status contracts.KyivstarSIMStatus) {
		overviewLabel.SetText(vm.BuildOverviewText(status))
		stateLabel.SetText(vm.BuildStateText(status))
		identityLabel.SetText(vm.BuildIdentityText(status))
		usageLabel.SetText(vm.BuildUsageText(status))
	}

	gatherSelectedServiceIDs := func() []string {
		selected := make([]string, 0, len(serviceChecks))
		for serviceID, check := range serviceChecks {
			if check != nil && check.Checked {
				selected = append(selected, serviceID)
			}
		}
		return selected
	}

	renderServices := func(services []contracts.KyivstarSIMServiceStatus) {
		serviceChecks = make(map[string]*widget.Check, len(services))
		if len(services) == 0 {
			serviceBox.Objects = []fyne.CanvasObject{widget.NewLabel("Kyivstar: сервісних блокувань не знайдено.")}
			serviceBox.Refresh()
			return
		}

		items := make([]fyne.CanvasObject, 0, len(services))
		for _, service := range services {
			serviceID := strings.TrimSpace(service.ServiceID)
			if serviceID == "" {
				continue
			}
			check := widget.NewCheck("", nil)
			title := widget.NewLabel(vm.BuildServiceTitle(service))
			title.TextStyle = fyne.TextStyle{Bold: true}
			title.Wrapping = fyne.TextWrapWord
			details := widget.NewLabel(vm.BuildServiceDetails(service))
			details.Wrapping = fyne.TextWrapWord
			serviceChecks[serviceID] = check
			items = append(items, widget.NewCard(
				"",
				"",
				container.NewBorder(
					nil,
					nil,
					check,
					nil,
					container.NewVBox(title, details),
				),
			))
		}
		if len(items) == 0 {
			items = append(items, widget.NewLabel("Kyivstar: сервіси недоступні."))
		}
		serviceBox.Objects = items
		serviceBox.Refresh()
	}

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
		for _, check := range serviceChecks {
			if check == nil {
				continue
			}
			if busy {
				check.Disable()
			} else {
				check.Enable()
			}
		}
	}

	runAction := func(startedText string, action func() (string, error)) {
		resultLabel.SetText(startedText)
		setBusy(true, refreshBtn, rebootBtn, syncBtn, pauseBtn, activateBtn, pauseServicesBtn, activateServicesBtn)
		go func() {
			text, err := action()
			fyne.Do(func() {
				setBusy(false, refreshBtn, rebootBtn, syncBtn, pauseBtn, activateBtn, pauseServicesBtn, activateServicesBtn)
				if err != nil {
					resultLabel.SetText(err.Error())
					return
				}
				resultLabel.SetText(text)
			})
		}()
	}

	refreshBtn = makeIconButton("Статус", iconRefresh(), widget.MediumImportance, func() {
		runAction("Kyivstar: перевірка стану...", func() (string, error) {
			status, err := provider.GetKyivstarSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			fyne.DoAndWait(func() {
				applyStatus(status)
				renderServices(status.Services)
			})
			return vm.BuildStatusText(status), nil
		})
	})

	rebootBtn = makeIconButton("Reset SIM", iconRefresh(), widget.MediumImportance, func() {
		runAction("Kyivstar: запит на reset...", func() (string, error) {
			result, err := provider.RebootKyivstarSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildResetResultText(result), nil
		})
	})

	syncBtn = makeIconButton("Записати дані", iconEdit(), widget.LowImportance, func() {
		runAction("Kyivstar: запис deviceName/deviceId...", func() (string, error) {
			deviceName, deviceID, err := vm.BuildMetadata(msisdn, objectNumber, objectName, objectName)
			if err != nil {
				return "", err
			}
			if err := provider.UpdateKyivstarSIMMetadata(msisdn, deviceName, deviceID); err != nil {
				return "", err
			}
			return "Kyivstar: deviceName/deviceId оновлено", nil
		})
	})

	pauseBtn = makeIconButton("Блокувати номер", iconClose(), widget.DangerImportance, func() {
		showSIMNumberActionConfirm(
			parent,
			"Підтвердити блокування",
			fmt.Sprintf("Заблокувати номер %s для об'єкта #%s?", msisdn, objectNumber),
			func() {
				runAction("Kyivstar: блокування номера...", func() (string, error) {
					result, err := provider.PauseKyivstarSIM(msisdn)
					if err != nil {
						return "", err
					}
					return vm.BuildOperationText(result), nil
				})
			},
		)
	})

	activateBtn = makeIconButton("Розблокувати номер", iconAdd(), widget.HighImportance, func() {
		showSIMNumberActionConfirm(
			parent,
			"Підтвердити розблокування",
			fmt.Sprintf("Розблокувати номер %s для об'єкта #%s?", msisdn, objectNumber),
			func() {
				runAction("Kyivstar: розблокування номера...", func() (string, error) {
					result, err := provider.ActivateKyivstarSIM(msisdn)
					if err != nil {
						return "", err
					}
					return vm.BuildOperationText(result), nil
				})
			},
		)
	})

	pauseServicesBtn = makeIconButton("Блокувати сервіси", iconClose(), widget.DangerImportance, func() {
		selectedServiceIDs := gatherSelectedServiceIDs()
		runAction("Kyivstar: блокування сервісів...", func() (string, error) {
			result, err := provider.PauseKyivstarSIMServices(msisdn, selectedServiceIDs)
			if err != nil {
				return "", err
			}
			return "Kyivstar: змінено блокування сервісів для " + result.MSISDN, nil
		})
	})

	activateServicesBtn = makeIconButton("Розблокувати сервіси", iconAdd(), widget.HighImportance, func() {
		selectedServiceIDs := gatherSelectedServiceIDs()
		runAction("Kyivstar: розблокування сервісів...", func() (string, error) {
			result, err := provider.ActivateKyivstarSIMServices(msisdn, selectedServiceIDs)
			if err != nil {
				return "", err
			}
			return "Kyivstar: змінено стани сервісів для " + result.MSISDN, nil
		})
	})

	headerCard := widget.NewCard(
		"Номер та об'єкт",
		"",
		container.NewVBox(titleLabel, objectLabel),
	)

	statusCard := widget.NewCard(
		"Стан",
		"",
		container.NewVBox(overviewLabel, stateLabel),
	)

	identityCard := widget.NewCard("Ідентифікатори", "", identityLabel)
	usageCard := widget.NewCard("Використання", "", usageLabel)

	actionsCard := widget.NewCard(
		"Дії",
		"",
		container.NewVBox(
			container.NewGridWithColumns(3, refreshBtn, rebootBtn, syncBtn),
			container.NewGridWithColumns(2, pauseBtn, activateBtn),
		),
	)

	servicesCard := widget.NewCard(
		"Сервісні блокування",
		"Позначте сервіси, для яких треба змінити стан.",
		container.NewVBox(
			serviceScroll,
			container.NewGridWithColumns(2, pauseServicesBtn, activateServicesBtn),
		),
	)

	resultCard := widget.NewCard("Результат", "", resultLabel)

	content := container.NewVBox(
		headerCard,
		container.NewGridWithColumns(3, statusCard, identityCard, usageCard),
		actionsCard,
		servicesCard,
		resultCard,
	)

	scrollContent := container.NewVScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(780, 560))

	dlg := dialog.NewCustom("Kyivstar запити", "Закрити", scrollContent, parent)
	dlg.Resize(fyne.NewSize(840, 620))
	dlg.Show()
}
