package viewmodels

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/utils"
	"strings"
)

type KyivstarSIMViewModel struct{}

func NewKyivstarSIMViewModel() *KyivstarSIMViewModel {
	return &KyivstarSIMViewModel{}
}

func (vm *KyivstarSIMViewModel) BuildStatusText(status contracts.KyivstarSIMStatus) string {
	msisdn := strings.TrimSpace(status.MSISDN)
	if msisdn == "" {
		return "Kyivstar: SIM не вказана"
	}
	if !status.Available {
		return fmt.Sprintf("Kyivstar: %s відсутній у списку доступних IoT номерів", msisdn)
	}

	parts := []string{fmt.Sprintf("Kyivstar: %s", msisdn)}
	if strings.TrimSpace(status.NumberStatus) != "" {
		parts = append(parts, "статус "+strings.TrimSpace(status.NumberStatus))
	}
	if status.IsOnline {
		parts = append(parts, "online")
	} else {
		parts = append(parts, "offline")
	}
	if strings.TrimSpace(status.DeviceName) != "" {
		parts = append(parts, "назва "+strings.TrimSpace(status.DeviceName))
	}
	if strings.TrimSpace(status.DeviceID) != "" {
		parts = append(parts, "ID "+strings.TrimSpace(status.DeviceID))
	}
	if strings.TrimSpace(status.ICCID) != "" {
		parts = append(parts, "ICCID "+strings.TrimSpace(status.ICCID))
	}
	if strings.TrimSpace(status.TariffPlan) != "" {
		parts = append(parts, "тариф "+strings.TrimSpace(status.TariffPlan))
	}
	if strings.TrimSpace(status.Account) != "" {
		parts = append(parts, "рахунок "+strings.TrimSpace(status.Account))
	}
	serviceParts := make([]string, 0, len(status.Services))
	for _, service := range status.Services {
		name := strings.TrimSpace(service.Name)
		if name == "" {
			name = strings.TrimSpace(service.ServiceID)
		}
		if name == "" {
			continue
		}
		state := strings.TrimSpace(service.Status)
		if state != "" {
			name += "=" + state
		}
		serviceParts = append(serviceParts, name)
	}
	if len(serviceParts) > 0 {
		parts = append(parts, "сервіси "+strings.Join(serviceParts, ", "))
	}
	return strings.Join(parts, " | ")
}

func (vm *KyivstarSIMViewModel) BuildOverviewText(status contracts.KyivstarSIMStatus) string {
	msisdn := strings.TrimSpace(status.MSISDN)
	if msisdn == "" {
		return "SIM не вказана"
	}
	if !status.Available {
		return fmt.Sprintf("%s відсутній у списку доступних IoT номерів", msisdn)
	}

	onlineText := "offline"
	if status.IsOnline {
		onlineText = "online"
	}
	if strings.TrimSpace(status.NumberStatus) == "" {
		return fmt.Sprintf("%s | %s", msisdn, onlineText)
	}
	return fmt.Sprintf("%s | %s | статус %s", msisdn, onlineText, strings.TrimSpace(status.NumberStatus))
}

func (vm *KyivstarSIMViewModel) BuildStateText(status contracts.KyivstarSIMStatus) string {
	if !status.Available {
		return "Номер не знайдено в IoT-кабінеті Kyivstar."
	}

	parts := []string{
		"Доступний у кабінеті: так",
	}
	if status.IsOnline {
		parts = append(parts, "Онлайн: так")
	} else {
		parts = append(parts, "Онлайн: ні")
	}
	if strings.TrimSpace(status.NumberStatus) != "" {
		parts = append(parts, "Статус номера: "+strings.TrimSpace(status.NumberStatus))
	}
	if len(status.AvailableActions) > 0 {
		parts = append(parts, "Дії: "+strings.Join(utils.TrimmedNonEmptyStrings(status.AvailableActions), ", "))
	}
	if status.IsTestPeriod {
		parts = append(parts, "Тестовий період: так")
	}
	return strings.Join(parts, "\n")
}

func (vm *KyivstarSIMViewModel) BuildIdentityText(status contracts.KyivstarSIMStatus) string {
	parts := make([]string, 0, 6)
	if strings.TrimSpace(status.DeviceName) != "" {
		parts = append(parts, "Назва пристрою: "+strings.TrimSpace(status.DeviceName))
	}
	if strings.TrimSpace(status.DeviceID) != "" {
		parts = append(parts, "Ідентифікатор: "+strings.TrimSpace(status.DeviceID))
	}
	if strings.TrimSpace(status.ICCID) != "" {
		parts = append(parts, "ICCID: "+strings.TrimSpace(status.ICCID))
	}
	if strings.TrimSpace(status.IMEI) != "" {
		parts = append(parts, "IMEI: "+strings.TrimSpace(status.IMEI))
	}
	if strings.TrimSpace(status.TariffPlan) != "" {
		parts = append(parts, "Тариф: "+strings.TrimSpace(status.TariffPlan))
	}
	if strings.TrimSpace(status.Account) != "" {
		parts = append(parts, "Рахунок: "+strings.TrimSpace(status.Account))
	}
	if len(parts) == 0 {
		return "Довідкові дані ще не завантажені."
	}
	return strings.Join(parts, "\n")
}

func (vm *KyivstarSIMViewModel) BuildUsageText(status contracts.KyivstarSIMStatus) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(status.DataUsage) != "" {
		parts = append(parts, "Трафік: "+strings.TrimSpace(status.DataUsage))
	}
	if strings.TrimSpace(status.SMSUsage) != "" {
		parts = append(parts, "SMS: "+strings.TrimSpace(status.SMSUsage))
	}
	if strings.TrimSpace(status.VoiceUsage) != "" {
		parts = append(parts, "Голос: "+strings.TrimSpace(status.VoiceUsage))
	}
	if len(parts) == 0 {
		return "Статистика використання відсутня."
	}
	return strings.Join(parts, "\n")
}

func (vm *KyivstarSIMViewModel) BuildServiceTitle(service contracts.KyivstarSIMServiceStatus) string {
	name := strings.TrimSpace(service.Name)
	if name == "" {
		name = strings.TrimSpace(service.ServiceID)
	}
	if name == "" {
		name = "Невідомий сервіс"
	}
	return name
}

func (vm *KyivstarSIMViewModel) BuildServiceDetails(service contracts.KyivstarSIMServiceStatus) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(service.ServiceID) != "" {
		parts = append(parts, "ID: "+strings.TrimSpace(service.ServiceID))
	}
	if strings.TrimSpace(service.Status) != "" {
		parts = append(parts, "Статус: "+strings.TrimSpace(service.Status))
	}
	if len(service.AvailableActions) > 0 {
		parts = append(parts, "Доступні дії: "+strings.Join(utils.TrimmedNonEmptyStrings(service.AvailableActions), ", "))
	}
	return strings.Join(parts, "\n")
}

func (vm *KyivstarSIMViewModel) BuildMetadata(msisdn string, objn string, shortName string, fullName string) (string, string, error) {
	msisdn = strings.TrimSpace(msisdn)
	if msisdn == "" {
		return "", "", fmt.Errorf("SIM не вказана")
	}
	objn = strings.TrimSpace(objn)
	if objn == "" {
		return "", "", fmt.Errorf("не вказано № об'єкта")
	}

	deviceName := strings.TrimSpace(shortName)
	if deviceName == "" {
		deviceName = strings.TrimSpace(fullName)
	}
	if deviceName == "" {
		return "", "", fmt.Errorf("не вказано назву об'єкта")
	}
	return deviceName, objn, nil
}

func (vm *KyivstarSIMViewModel) BuildOperationText(result contracts.KyivstarSIMOperationResult) string {
	operation := "операцію"
	switch strings.TrimSpace(result.Operation) {
	case "pause":
		operation = "блокування"
	case "activate":
		operation = "розблокування"
	}
	return "Kyivstar: виконано " + operation + " для " + strings.TrimSpace(result.MSISDN)
}

func (vm *KyivstarSIMViewModel) BuildResetResultText(result contracts.KyivstarSIMResetResult) string {
	if strings.TrimSpace(result.Email) == "" {
		return "Kyivstar: запит на reset відправлено для " + strings.TrimSpace(result.MSISDN)
	}
	return "Kyivstar: запит на reset відправлено для " + strings.TrimSpace(result.MSISDN) + "\nEmail: " + strings.TrimSpace(result.Email)
}
