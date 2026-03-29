package viewmodels

import (
	"fmt"
	"strings"

	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/models"
)

// WorkAreaExportViewModel інкапсулює підготовку даних для експорту об'єкта.
type WorkAreaExportViewModel struct{}

func NewWorkAreaExportViewModel() *WorkAreaExportViewModel {
	return &WorkAreaExportViewModel{}
}

func (vm *WorkAreaExportViewModel) BuildObjectExportData(
	obj models.Object,
	zones []models.Zone,
	contacts []models.Contact,
	events []models.Event,
	external WorkAreaExternalData,
) objexport.ObjectExportData {
	lastEventText := "Немає"
	if len(events) > 0 {
		latest := events[0]
		eventTime := "Немає дати"
		if !latest.Time.IsZero() {
			eventTime = latest.Time.Format("02.01.2006 15:04:05")
		}
		lastEventText = fmt.Sprintf("%s | %s", eventTime, latest.GetTypeDisplay())
		if latest.ZoneNumber > 0 {
			lastEventText += fmt.Sprintf(" | Зона %d", latest.ZoneNumber)
		}
		if strings.TrimSpace(latest.Details) != "" {
			lastEventText += " | " + strings.TrimSpace(latest.Details)
		}
	}

	lastTestText := "Немає"
	if !external.LastTest.IsZero() {
		lastTestText = external.LastTest.Format("02.01.2006 15:04:05")
	}
	trimmedTestMsg := strings.TrimSpace(external.TestMessage)
	if trimmedTestMsg != "" && trimmedTestMsg != "—" {
		if lastTestText == "Немає" {
			lastTestText = trimmedTestMsg
		} else {
			lastTestText += " | " + trimmedTestMsg
		}
	}

	zoneRows := make([]objexport.ZoneExportRow, 0, len(zones))
	for _, z := range zones {
		zoneRows = append(zoneRows, objexport.ZoneExportRow{
			Number: fmt.Sprintf("%d", z.Number),
			Name:   emptyFallback(z.Name),
			Type:   emptyFallback(z.SensorType),
			Status: emptyFallback(z.GetStatusDisplay()),
		})
	}

	responsibleRows := make([]objexport.ResponsibleExportRow, 0, len(contacts))
	for _, c := range contacts {
		responsibleRows = append(responsibleRows, objexport.ResponsibleExportRow{
			Name:  emptyFallback(c.Name),
			Phone: emptyFallback(c.Phone),
			Note:  emptyFallback(c.Position),
		})
	}

	return objexport.ObjectExportData{
		Number:         obj.ID,
		Name:           emptyFallback(obj.Name),
		Address:        emptyFallback(obj.Address),
		ContractNumber: emptyFallback(obj.ContractNum),
		LaunchDate:     emptyFallback(obj.LaunchDate),
		SimCard:        buildSimValue(obj),
		DeviceType:     emptyFallback(obj.DeviceType),
		TestPeriod:     buildTestPeriod(obj),
		LastEvent:      lastEventText,
		LastTest:       lastTestText,
		Channel:        channelText(obj.ObjChan),
		ObjectPhone:    emptyFallback(obj.Phones1),
		Location:       emptyFallback(obj.Location1),
		AdditionalInfo: emptyFallback(obj.Notes1),
		Zones:          zoneRows,
		Responsibles:   responsibleRows,
	}
}

func (vm *WorkAreaExportViewModel) BuildExcelRowTSV(obj models.Object, contacts []models.Contact) string {
	managerName := ""
	managerPhone := ""
	if len(contacts) > 0 {
		managerName = strings.TrimSpace(contacts[0].Name)
		managerPhone = strings.TrimSpace(contacts[0].Phone)
	} else if len(obj.Contacts) > 0 {
		managerName = strings.TrimSpace(obj.Contacts[0].Name)
		managerPhone = strings.TrimSpace(obj.Contacts[0].Phone)
	}

	fields := []string{
		fmt.Sprintf("%d", obj.ID),             // собсс
		cleanTSV(obj.LaunchDate),              // Дата підключен. до ПЦС
		cleanTSV(obj.ContractNum),             // Дата угоди (за поточними даними: номер/ідентифікатор угоди)
		"",                                    // Юридична назва, згідно угоди
		"",                                    // Юридична адреса, згідно угоди
		cleanTSV(obj.Name),                    // Фізична назва об’єкту по вивісці
		cleanTSV(obj.Address),                 // Фізична адреса об’єкту
		cleanTSV(obj.DeviceType),              // ПКП
		cleanTSV(obj.PanelMark),               // СЦС
		cleanTSV(strings.TrimSpace(obj.SIM1)), // Основний канал зв’язку / телефон підключення
		cleanTSV(strings.TrimSpace(obj.SIM2)), // Резервний канал зв’язку / телефон підключення
		"",                                    // Місячна оплата
		"",                                    // Електронна пошта об’єкту
		cleanTSV(managerName),                 // Керівник об’єкту
		cleanTSV(managerPhone),                // Контакт керівника
		cleanTSV(obj.Notes1),                  // Примітки
	}

	return strings.Join(fields, "\t")
}

func cleanTSV(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func buildSimValue(obj models.Object) string {
	sim1 := strings.TrimSpace(obj.SIM1)
	sim2 := strings.TrimSpace(obj.SIM2)
	if sim1 == "" && sim2 == "" {
		return "Немає"
	}
	if sim2 == "" {
		return sim1
	}
	if sim1 == "" {
		return sim2
	}
	return sim1 + " / " + sim2
}

func buildTestPeriod(obj models.Object) string {
	if obj.AutoTestHours > 0 {
		return fmt.Sprintf("Кожні %d год", obj.AutoTestHours)
	}
	if obj.TestTime > 0 {
		return fmt.Sprintf("Кожні %d хв", obj.TestTime)
	}
	return "Немає"
}

func channelText(chanID int) string {
	switch chanID {
	case 1:
		return "Автододзвон"
	case 5:
		return "GPRS"
	default:
		return "Інший канал"
	}
}

func emptyFallback(v string) string {
	if strings.TrimSpace(v) == "" {
		return "Немає"
	}
	return strings.TrimSpace(v)
}
