package viewmodels

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
	"time"
)

type VodafoneSIMViewModel struct{}

const vodafoneManualBlockingReason = "Інша причина"

func NewVodafoneSIMViewModel() *VodafoneSIMViewModel {
	return &VodafoneSIMViewModel{}
}

func (vm *VodafoneSIMViewModel) BlockingReasonOptions() []string {
	return []string{
		"Нема угоди",
		"Заміна сім карти",
		"Розірвали угоду",
		vodafoneManualBlockingReason,
	}
}

func (vm *VodafoneSIMViewModel) IsManualBlockingReason(reason string) bool {
	return strings.TrimSpace(reason) == vodafoneManualBlockingReason
}

func (vm *VodafoneSIMViewModel) BuildStatusText(status contracts.VodafoneSIMStatus) string {
	msisdn := strings.TrimSpace(status.MSISDN)
	if msisdn == "" {
		return "Vodafone: SIM не вказана"
	}
	if !status.Available {
		return fmt.Sprintf("Vodafone: %s відсутній у списку доступних IoT SIM", msisdn)
	}

	parts := []string{
		fmt.Sprintf("Vodafone: %s", msisdn),
	}
	if strings.TrimSpace(status.Connectivity.SIMStatus) != "" {
		parts = append(parts, "SIM="+strings.TrimSpace(status.Connectivity.SIMStatus))
	}
	if strings.TrimSpace(status.Connectivity.ConnectionTimeRaw) != "" {
		parts = append(parts, "ост. зв'язок "+strings.TrimSpace(status.Connectivity.ConnectionTimeRaw))
	}
	if blockingText := humanizeVodafoneBlockingStatus(status.Blocking.Status); blockingText != "" {
		parts = append(parts, "блокування "+blockingText)
	}
	if strings.TrimSpace(status.Blocking.BlockingDateRaw) != "" {
		parts = append(parts, "дата блок. "+strings.TrimSpace(status.Blocking.BlockingDateRaw))
	}
	if strings.TrimSpace(status.Blocking.BlockingRequestDateRaw) != "" {
		parts = append(parts, "заявка "+strings.TrimSpace(status.Blocking.BlockingRequestDateRaw))
	}
	if strings.TrimSpace(status.LastEvent.CallType) != "" || strings.TrimSpace(status.LastEvent.EventTimeRaw) != "" {
		eventText := strings.TrimSpace(status.LastEvent.CallType)
		if strings.TrimSpace(status.LastEvent.EventTimeRaw) != "" {
			if eventText != "" {
				eventText += " "
			}
			eventText += strings.TrimSpace(status.LastEvent.EventTimeRaw)
		}
		parts = append(parts, "подія "+eventText)
	}
	if strings.TrimSpace(status.SubscriberName) != "" {
		parts = append(parts, "назва "+strings.TrimSpace(status.SubscriberName))
	}
	return strings.Join(parts, " | ")
}

func (vm *VodafoneSIMViewModel) BuildOverviewText(status contracts.VodafoneSIMStatus) string {
	msisdn := strings.TrimSpace(status.MSISDN)
	if msisdn == "" {
		return "SIM не вказана"
	}
	if !status.Available {
		return fmt.Sprintf("%s відсутній у списку доступних IoT SIM", msisdn)
	}

	parts := []string{msisdn}
	if simStatus := strings.TrimSpace(status.Connectivity.SIMStatus); simStatus != "" {
		parts = append(parts, "SIM "+simStatus)
	}
	if eventText := vm.buildLastEventInline(status); eventText != "" {
		parts = append(parts, "подія "+eventText)
	}
	return strings.Join(parts, " | ")
}

func (vm *VodafoneSIMViewModel) BuildConnectivityText(status contracts.VodafoneSIMStatus) string {
	if !status.Available {
		return "Номер не знайдено в IoT-кабінеті Vodafone."
	}

	parts := make([]string, 0, 5)
	if value := strings.TrimSpace(status.Connectivity.SIMStatus); value != "" {
		parts = append(parts, "SIM статус: "+value)
	}
	if value := strings.TrimSpace(status.Connectivity.OperationStatus); value != "" {
		parts = append(parts, "Статус операції: "+value)
	}
	if value := strings.TrimSpace(status.Connectivity.BaseStationStatus); value != "" {
		parts = append(parts, "Базова станція: "+value)
	}
	if value := strings.TrimSpace(status.Connectivity.LBSStatusKey); value != "" {
		parts = append(parts, "LBS: "+value)
	}
	if value := strings.TrimSpace(status.Connectivity.ConnectionTimeRaw); value != "" {
		parts = append(parts, "Останній зв'язок: "+value)
	}
	if len(parts) == 0 {
		return "Дані підключення ще не завантажені."
	}
	return strings.Join(parts, "\n")
}

func (vm *VodafoneSIMViewModel) BuildBlockingText(status contracts.VodafoneSIMStatus) string {
	if !status.Available {
		return "Інформація про блокування недоступна."
	}

	parts := make([]string, 0, 4)
	if value := humanizeVodafoneBlockingStatus(status.Blocking.Status); value != "" {
		parts = append(parts, "Стан блокування: "+value)
	}
	if value := strings.TrimSpace(status.Blocking.BlockingDateRaw); value != "" {
		parts = append(parts, "Дата блокування: "+value)
	}
	if value := strings.TrimSpace(status.Blocking.BlockingRequestDateRaw); value != "" {
		parts = append(parts, "Дата заявки: "+value)
	}
	if value := strings.TrimSpace(status.Blocking.UpdateDateRaw); value != "" {
		parts = append(parts, "Оновлено: "+value)
	}
	if len(parts) == 0 {
		return "Блокування не активне."
	}
	return strings.Join(parts, "\n")
}

func (vm *VodafoneSIMViewModel) BuildEventText(status contracts.VodafoneSIMStatus) string {
	if !status.Available {
		return "Остання подія недоступна."
	}
	if eventText := vm.buildLastEventInline(status); eventText != "" {
		return eventText
	}
	return "Подій немає."
}

func (vm *VodafoneSIMViewModel) BuildIdentityText(status contracts.VodafoneSIMStatus) string {
	if !status.Available {
		return "Назва абонента недоступна."
	}
	if value := strings.TrimSpace(status.SubscriberName); value != "" {
		return "Назва абонента: " + value
	}
	return "Назва абонента не задана."
}

func (vm *VodafoneSIMViewModel) BuildMetadata(msisdn string, objn string, shortName string, fullName string) (string, string, error) {
	msisdn = strings.TrimSpace(msisdn)
	if msisdn == "" {
		return "", "", fmt.Errorf("SIM не вказана")
	}
	objn = strings.TrimSpace(objn)
	if objn == "" {
		return "", "", fmt.Errorf("не вказано № об'єкта")
	}

	comment := strings.TrimSpace(shortName)
	if comment == "" {
		comment = strings.TrimSpace(fullName)
	}
	if comment == "" {
		return "", "", fmt.Errorf("не вказано назву об'єкта")
	}
	return objn, comment, nil
}

func (vm *VodafoneSIMViewModel) BuildBlockingMetadata(objectNumber string, reason string, manualReason string, now time.Time) (string, string, error) {
	objectNumber = strings.TrimSpace(objectNumber)
	if objectNumber == "" {
		return "", "", fmt.Errorf("не вказано № об'єкта")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", "", fmt.Errorf("не вказано причину блокування")
	}
	if vm.IsManualBlockingReason(reason) {
		reason = strings.TrimSpace(manualReason)
		if reason == "" {
			return "", "", fmt.Errorf("вкажіть власну причину блокування")
		}
	}

	comment := reason
	if !now.IsZero() {
		comment = fmt.Sprintf("%s (%s)", reason, now.Format("02.01.2006"))
	}
	return objectNumber, comment, nil
}

func (vm *VodafoneSIMViewModel) BuildBarringResultText(result contracts.VodafoneSIMBarringResult) string {
	operation := "зміну стану номера"
	switch strings.TrimSpace(result.Operation) {
	case "block":
		operation = "блокування"
	case "unblock":
		operation = "розблокування"
	}

	if strings.TrimSpace(result.OrderID) == "" {
		return "Vodafone: заявку на " + operation + " відправлено"
	}
	if strings.TrimSpace(result.State) == "" {
		return "Vodafone: заявку на " + operation + " відправлено, ID " + result.OrderID
	}
	return "Vodafone: заявку на " + operation + " відправлено, ID " + result.OrderID + ", стан " + result.State
}

func (vm *VodafoneSIMViewModel) BuildRebootResultText(result contracts.VodafoneSIMRebootResult) string {
	if strings.TrimSpace(result.OrderID) == "" {
		return "Vodafone: заявку на перезавантаження створено"
	}
	if strings.TrimSpace(result.State) == "" {
		return "Vodafone: заявку створено, ID " + result.OrderID
	}
	return "Vodafone: заявку створено, ID " + result.OrderID + ", стан " + result.State
}

func (vm *VodafoneSIMViewModel) buildLastEventInline(status contracts.VodafoneSIMStatus) string {
	eventText := strings.TrimSpace(status.LastEvent.CallType)
	if value := strings.TrimSpace(status.LastEvent.EventTimeRaw); value != "" {
		if eventText != "" {
			eventText += " "
		}
		eventText += value
	}
	return strings.TrimSpace(eventText)
}

func humanizeVodafoneBlockingStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "":
		return ""
	case "NotBlocked":
		return "не активне"
	case "PartBlocked":
		return "часткове"
	case "FullBlocked":
		return "повне"
	case "FinalBlocked":
		return "фінальне"
	default:
		return strings.TrimSpace(status)
	}
}
