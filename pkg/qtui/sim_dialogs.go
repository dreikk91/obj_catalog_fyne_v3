//go:build qt

package qtui

import (
	"fmt"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

func ShowVodafoneSIMDialog(
	parent *qt.QWidget,
	provider contracts.AdminObjectVodafoneService,
	msisdn string,
	objectNumber string,
	objectName string,
) {
	msisdn = strings.TrimSpace(msisdn)
	objectNumber = strings.TrimSpace(objectNumber)
	if msisdn == "" {
		qt.QMessageBox_Information(parent, "Vodafone", "SIM номер не вказаний.")
		return
	}
	if provider == nil {
		qt.QMessageBox_Information(parent, "Vodafone", "Vodafone сервіс недоступний.")
		return
	}

	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Vodafone IoT Управління")
	dialog.Resize(680, 520)

	mainLayout := qt.NewQVBoxLayout(dialog.QWidget)

	vm := viewmodels.NewVodafoneSIMViewModel()

	// 1. Header Info
	headerGroup := qt.NewQGroupBox4("Номер та об'єкт", dialog.QWidget)
	headerLayout := qt.NewQVBoxLayout(headerGroup.QWidget)
	titleLabel := qt.NewQLabel3(fmt.Sprintf("<b>SIM:</b> %s", msisdn))
	objectLabel := qt.NewQLabel3(fmt.Sprintf("<b>Об'єкт:</b> #%s %s", objectNumber, strings.TrimSpace(objectName)))
	headerLayout.AddWidget(titleLabel.QWidget)
	headerLayout.AddWidget(objectLabel.QWidget)
	mainLayout.AddWidget(headerGroup.QWidget)

	// 2. Status Info
	statusGroup := qt.NewQGroupBox4("Стан SIM-картки", dialog.QWidget)
	statusLayout := qt.NewQFormLayout(statusGroup.QWidget)
	overviewLabel := qt.NewQLabel3("Стан ще не завантажено.")
	connectivityLabel := qt.NewQLabel3("Натисніть \"Статус\", щоб отримати актуальні дані.")
	identityLabel := qt.NewQLabel3("Назва та коментар з'являться тут.")
	eventLabel := qt.NewQLabel3("Остання подія ще не завантажена.")
	blockingLabel := qt.NewQLabel3("Стан блокування ще не завантажено.")

	statusLayout.AddRow3("Огляд:", overviewLabel.QWidget)
	statusLayout.AddRow3("Зв'язок:", connectivityLabel.QWidget)
	statusLayout.AddRow3("Ідентифікатори:", identityLabel.QWidget)
	statusLayout.AddRow3("Остання подія:", eventLabel.QWidget)
	statusLayout.AddRow3("Блокування:", blockingLabel.QWidget)
	mainLayout.AddWidget(statusGroup.QWidget)

	// 3. Blocking Options
	blockGroup := qt.NewQGroupBox4("Параметри блокування", dialog.QWidget)
	blockLayout := qt.NewQFormLayout(blockGroup.QWidget)

	reasonSelect := qt.NewQComboBox2()
	reasons := vm.BlockingReasonOptions()
	reasonSelect.AddItems(reasons)

	customReasonEntry := qt.NewQLineEdit2()
	customReasonEntry.SetPlaceholderText("Вкажіть причину вручную")
	customReasonEntry.SetEnabled(false)

	reasonSelect.OnCurrentTextChanged(func(selected string) {
		customReasonEntry.SetEnabled(vm.IsManualBlockingReason(selected))
	})

	blockLayout.AddRow3("Причина:", reasonSelect.QWidget)
	blockLayout.AddRow3("Своя причина:", customReasonEntry.QWidget)
	mainLayout.AddWidget(blockGroup.QWidget)

	// 4. Action Buttons
	actionsLayout := qt.NewQHBoxLayout2()
	refreshBtn := qt.NewQPushButton3("Оновити статус")
	rebootBtn := qt.NewQPushButton3("Reset SIM")
	syncBtn := qt.NewQPushButton3("Записати дані")
	blockBtn := qt.NewQPushButton3("Блокувати")
	blockBtn.SetStyleSheet("background-color: #f44336; color: white; font-weight: bold;")
	unblockBtn := qt.NewQPushButton3("Розблокувати")
	unblockBtn.SetStyleSheet("background-color: #4CAF50; color: white; font-weight: bold;")

	actionsLayout.AddWidget(refreshBtn.QWidget)
	actionsLayout.AddWidget(rebootBtn.QWidget)
	actionsLayout.AddWidget(syncBtn.QWidget)
	actionsLayout.AddWidget(blockBtn.QWidget)
	actionsLayout.AddWidget(unblockBtn.QWidget)
	mainLayout.AddLayout(actionsLayout.QLayout)

	// 5. Result Display
	resultLabel := qt.NewQLabel3("Vodafone: перевірка за запитом")
	resultLabel.SetFrameStyle(int(qt.QFrame__StyledPanel) | int(qt.QFrame__Sunken))
	resultLabel.SetStyleSheet("padding: 8px; background-color: #f5f5f5; font-family: monospace;")
	resultLabel.SetWordWrap(true)
	mainLayout.AddWidget(resultLabel.QWidget)

	// Buttons Box
	buttonBox := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	buttonBox.OnRejected(dialog.Reject)
	mainLayout.AddWidget(buttonBox.QWidget)

	dialog.SetLayout(mainLayout.QLayout)

	setBusy := func(busy bool) {
		refreshBtn.SetEnabled(!busy)
		rebootBtn.SetEnabled(!busy)
		syncBtn.SetEnabled(!busy)
		blockBtn.SetEnabled(!busy)
		unblockBtn.SetEnabled(!busy)
		reasonSelect.SetEnabled(!busy)
		if !busy {
			customReasonEntry.SetEnabled(vm.IsManualBlockingReason(reasonSelect.CurrentText()))
		} else {
			customReasonEntry.SetEnabled(false)
		}
	}

	runAction := func(startedText string, action func() (string, error)) {
		resultLabel.SetText(startedText)
		setBusy(true)
		go func() {
			text, err := action()
			runOnMainThread(func() {
				setBusy(false)
				if err != nil {
					resultLabel.SetText("<font color='red'>Помилка: " + err.Error() + "</font>")
					return
				}
				resultLabel.SetText(text)
			})
		}()
	}

	refreshBtn.OnClicked(func() {
		runAction("Vodafone: перевірка стану...", func() (string, error) {
			status, err := provider.GetVodafoneSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			runOnMainThread(func() {
				overviewLabel.SetText(vm.BuildOverviewText(status))
				connectivityLabel.SetText(vm.BuildConnectivityText(status))
				identityLabel.SetText(vm.BuildIdentityText(status))
				eventLabel.SetText(vm.BuildEventText(status))
				blockingLabel.SetText(vm.BuildBlockingText(status))
			})
			return vm.BuildStatusText(status), nil
		})
	})

	rebootBtn.OnClicked(func() {
		runAction("Vodafone: створення заявки на скидання...", func() (string, error) {
			result, err := provider.RebootVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildRebootResultText(result), nil
		})
	})

	syncBtn.OnClicked(func() {
		runAction("Vodafone: запис назви та коментаря...", func() (string, error) {
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
			return "Vodafone: назву та коментар оновлено", nil
		})
	})

	blockBtn.OnClicked(func() {
		reply := qt.QMessageBox_Question(dialog.QWidget, "Підтвердження", fmt.Sprintf("Заблокувати номер %s для об'єкта #%s?", msisdn, objectNumber))
		if reply != qt.QMessageBox__Yes {
			return
		}
		reason := reasonSelect.CurrentText()
		customText := customReasonEntry.Text()

		runAction("Vodafone: блокування номера...", func() (string, error) {
			name, comment, err := vm.BuildBlockingMetadata(objectNumber, reason, customText, time.Now())
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

	unblockBtn.OnClicked(func() {
		reply := qt.QMessageBox_Question(dialog.QWidget, "Підтвердження", fmt.Sprintf("Розблокувати номер %s для об'єкта #%s?", msisdn, objectNumber))
		if reply != qt.QMessageBox__Yes {
			return
		}
		runAction("Vodafone: розблокування номера...", func() (string, error) {
			result, err := provider.UnblockVodafoneSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildBarringResultText(result), nil
		})
	})

	dialog.Exec()
}

func ShowKyivstarSIMDialog(
	parent *qt.QWidget,
	provider contracts.AdminObjectKyivstarService,
	msisdn string,
	objectNumber string,
	objectName string,
) {
	msisdn = strings.TrimSpace(msisdn)
	objectNumber = strings.TrimSpace(objectNumber)
	if msisdn == "" {
		qt.QMessageBox_Information(parent, "Kyivstar", "SIM номер не вказаний.")
		return
	}
	if provider == nil {
		qt.QMessageBox_Information(parent, "Kyivstar", "Kyivstar сервіс недоступний.")
		return
	}

	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Kyivstar M2M Управління")
	dialog.Resize(720, 580)

	mainLayout := qt.NewQVBoxLayout(dialog.QWidget)

	vm := viewmodels.NewKyivstarSIMViewModel()

	// 1. Header Info
	headerGroup := qt.NewQGroupBox4("Номер та об'єкт", dialog.QWidget)
	headerLayout := qt.NewQVBoxLayout(headerGroup.QWidget)
	titleLabel := qt.NewQLabel3(fmt.Sprintf("<b>SIM:</b> %s", msisdn))
	objectLabel := qt.NewQLabel3(fmt.Sprintf("<b>Об'єкт:</b> #%s %s", objectNumber, strings.TrimSpace(objectName)))
	headerLayout.AddWidget(titleLabel.QWidget)
	headerLayout.AddWidget(objectLabel.QWidget)
	mainLayout.AddWidget(headerGroup.QWidget)

	// 2. Status Info
	statusGroup := qt.NewQGroupBox4("Стан SIM-картки", dialog.QWidget)
	statusLayout := qt.NewQFormLayout(statusGroup.QWidget)
	overviewLabel := qt.NewQLabel3("Стан ще не завантажено.")
	stateLabel := qt.NewQLabel3("Натисніть \"Статус\", щоб отримати актуальні дані.")
	identityLabel := qt.NewQLabel3("Ідентифікатори та назва пристрою з'являться тут.")
	usageLabel := qt.NewQLabel3("Статистика використання ще не завантажена.")

	statusLayout.AddRow3("Огляд:", overviewLabel.QWidget)
	statusLayout.AddRow3("Стан номера:", stateLabel.QWidget)
	statusLayout.AddRow3("Ідентифікатори:", identityLabel.QWidget)
	statusLayout.AddRow3("Використання:", usageLabel.QWidget)
	mainLayout.AddWidget(statusGroup.QWidget)

	// 3. Kyivstar Services Checkboxes
	servicesGroup := qt.NewQGroupBox4("Сервісні блокування", dialog.QWidget)
	servicesLayout := qt.NewQVBoxLayout(servicesGroup.QWidget)
	servicesScroll := qt.NewQScrollArea2()
	servicesScroll.SetWidgetResizable(true)
	servicesScroll.SetMinimumHeight(140)

	servicesListContainer := qt.NewQWidget2()
	servicesListLayout := qt.NewQVBoxLayout(servicesListContainer)
	servicesListLayout.AddWidget(qt.NewQLabel3("Натисніть 'Статус', щоб завантажити сервіси.").QWidget)
	servicesScroll.SetWidget(servicesListContainer)
	servicesLayout.AddWidget(servicesScroll.QWidget)

	servicesButtonsLayout := qt.NewQHBoxLayout2()
	blockServicesBtn := qt.NewQPushButton3("Блокувати сервіси")
	blockServicesBtn.SetStyleSheet("background-color: #f44336; color: white;")
	activateServicesBtn := qt.NewQPushButton3("Розблокувати сервіси")
	activateServicesBtn.SetStyleSheet("background-color: #4CAF50; color: white;")
	servicesButtonsLayout.AddWidget(blockServicesBtn.QWidget)
	servicesButtonsLayout.AddWidget(activateServicesBtn.QWidget)
	servicesLayout.AddLayout(servicesButtonsLayout.QLayout)
	mainLayout.AddWidget(servicesGroup.QWidget)

	// 4. Action Buttons
	actionsLayout := qt.NewQHBoxLayout2()
	refreshBtn := qt.NewQPushButton3("Оновити статус")
	rebootBtn := qt.NewQPushButton3("Reset SIM")
	syncBtn := qt.NewQPushButton3("Записати дані")
	pauseBtn := qt.NewQPushButton3("Блокувати")
	pauseBtn.SetStyleSheet("background-color: #f44336; color: white; font-weight: bold;")
	activateBtn := qt.NewQPushButton3("Розблокувати")
	activateBtn.SetStyleSheet("background-color: #4CAF50; color: white; font-weight: bold;")

	actionsLayout.AddWidget(refreshBtn.QWidget)
	actionsLayout.AddWidget(rebootBtn.QWidget)
	actionsLayout.AddWidget(syncBtn.QWidget)
	actionsLayout.AddWidget(pauseBtn.QWidget)
	actionsLayout.AddWidget(activateBtn.QWidget)
	mainLayout.AddLayout(actionsLayout.QLayout)

	// 5. Result Display
	resultLabel := qt.NewQLabel3("Kyivstar: перевірка за запитом")
	resultLabel.SetFrameStyle(int(qt.QFrame__StyledPanel) | int(qt.QFrame__Sunken))
	resultLabel.SetStyleSheet("padding: 8px; background-color: #f5f5f5; font-family: monospace;")
	resultLabel.SetWordWrap(true)
	mainLayout.AddWidget(resultLabel.QWidget)

	// Close Box
	buttonBox := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	buttonBox.OnRejected(dialog.Reject)
	mainLayout.AddWidget(buttonBox.QWidget)

	dialog.SetLayout(mainLayout.QLayout)

	var (
		serviceChecks = make(map[string]*qt.QCheckBox)
	)

	applyStatus := func(status contracts.KyivstarSIMStatus) {
		overviewLabel.SetText(vm.BuildOverviewText(status))
		stateLabel.SetText(vm.BuildStateText(status))
		identityLabel.SetText(vm.BuildIdentityText(status))
		usageLabel.SetText(vm.BuildUsageText(status))
	}

	setBusy := func(busy bool) {
		refreshBtn.SetEnabled(!busy)
		rebootBtn.SetEnabled(!busy)
		syncBtn.SetEnabled(!busy)
		pauseBtn.SetEnabled(!busy)
		activateBtn.SetEnabled(!busy)
		blockServicesBtn.SetEnabled(!busy)
		activateServicesBtn.SetEnabled(!busy)
		for _, check := range serviceChecks {
			check.SetEnabled(!busy)
		}
	}

	gatherSelectedServiceIDs := func() []string {
		selected := make([]string, 0)
		for serviceID, check := range serviceChecks {
			if check != nil && check.IsChecked() {
				selected = append(selected, serviceID)
			}
		}
		return selected
	}

	renderServices := func(services []contracts.KyivstarSIMServiceStatus) {
		serviceChecks = make(map[string]*qt.QCheckBox, len(services))

		// Clear previous widgets in scroll container
		for {
			item := servicesListLayout.TakeAt(0)
			if item == nil {
				break
			}
			if widget := item.Widget(); widget != nil {
				widget.Hide()
				widget.Delete()
			}
		}

		if len(services) == 0 {
			servicesListLayout.AddWidget(qt.NewQLabel3("Kyivstar: сервісних блокувань не знайдено.").QWidget)
			return
		}

		for _, service := range services {
			serviceID := strings.TrimSpace(service.ServiceID)
			if serviceID == "" {
				continue
			}

			itemWidget := qt.NewQWidget2()
			itemLayout := qt.NewQHBoxLayout(itemWidget)
			itemLayout.SetContentsMargins(4, 4, 4, 4)

			check := qt.NewQCheckBox2()
			check.SetChecked(false)
			serviceChecks[serviceID] = check

			descLabel := qt.NewQLabel3(fmt.Sprintf("<b>%s</b><br/><font color='#666'>%s</font>", 
				htmlEscape(vm.BuildServiceTitle(service)), 
				htmlEscape(vm.BuildServiceDetails(service))))
			descLabel.SetWordWrap(true)

			itemLayout.AddWidget(check.QWidget)
			itemLayout.AddWidget(descLabel.QWidget)
			itemLayout.AddStretch()
			itemWidget.SetLayout(itemLayout.QLayout)

			servicesListLayout.AddWidget(itemWidget)
		}
	}

	runAction := func(startedText string, action func() (string, error)) {
		resultLabel.SetText(startedText)
		setBusy(true)
		go func() {
			text, err := action()
			runOnMainThread(func() {
				setBusy(false)
				if err != nil {
					resultLabel.SetText("<font color='red'>Помилка: " + err.Error() + "</font>")
					return
				}
				resultLabel.SetText(text)
			})
		}()
	}

	refreshBtn.OnClicked(func() {
		runAction("Kyivstar: перевірка стану...", func() (string, error) {
			status, err := provider.GetKyivstarSIMStatus(msisdn)
			if err != nil {
				return "", err
			}
			runOnMainThread(func() {
				applyStatus(status)
				renderServices(status.Services)
			})
			return vm.BuildStatusText(status), nil
		})
	})

	rebootBtn.OnClicked(func() {
		runAction("Kyivstar: запит на скидання...", func() (string, error) {
			result, err := provider.RebootKyivstarSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildResetResultText(result), nil
		})
	})

	syncBtn.OnClicked(func() {
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

	pauseBtn.OnClicked(func() {
		reply := qt.QMessageBox_Question(dialog.QWidget, "Підтвердження", fmt.Sprintf("Заблокувати номер %s для об'єкта #%s?", msisdn, objectNumber))
		if reply != qt.QMessageBox__Yes {
			return
		}
		runAction("Kyivstar: блокування номера...", func() (string, error) {
			result, err := provider.PauseKyivstarSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildOperationText(result), nil
		})
	})

	activateBtn.OnClicked(func() {
		reply := qt.QMessageBox_Question(dialog.QWidget, "Підтвердження", fmt.Sprintf("Розблокувати номер %s для об'єкта #%s?", msisdn, objectNumber))
		if reply != qt.QMessageBox__Yes {
			return
		}
		runAction("Kyivstar: розблокування номера...", func() (string, error) {
			result, err := provider.ActivateKyivstarSIM(msisdn)
			if err != nil {
				return "", err
			}
			return vm.BuildOperationText(result), nil
		})
	})

	blockServicesBtn.OnClicked(func() {
		selected := gatherSelectedServiceIDs()
		if len(selected) == 0 {
			qt.QMessageBox_Information(dialog.QWidget, "Kyivstar", "Будь ласка, виберіть хоча б один сервіс.")
			return
		}
		runAction("Kyivstar: блокування сервісів...", func() (string, error) {
			result, err := provider.PauseKyivstarSIMServices(msisdn, selected)
			if err != nil {
				return "", err
			}
			return "Kyivstar: змінено блокування сервісів для " + result.MSISDN, nil
		})
	})

	activateServicesBtn.OnClicked(func() {
		selected := gatherSelectedServiceIDs()
		if len(selected) == 0 {
			qt.QMessageBox_Information(dialog.QWidget, "Kyivstar", "Будь ласка, виберіть хоча б один сервіс.")
			return
		}
		runAction("Kyivstar: розблокування сервісів...", func() (string, error) {
			result, err := provider.ActivateKyivstarSIMServices(msisdn, selected)
			if err != nil {
				return "", err
			}
			return "Kyivstar: змінено стани сервісів для " + result.MSISDN, nil
		})
	})

	dialog.Exec()
}
