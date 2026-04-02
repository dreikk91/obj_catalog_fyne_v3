package viewmodels

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
)

type VodafoneSIMViewModel struct{}

func NewVodafoneSIMViewModel() *VodafoneSIMViewModel {
	return &VodafoneSIMViewModel{}
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
