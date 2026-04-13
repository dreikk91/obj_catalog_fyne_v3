package viewmodels

import (
	"fmt"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

const workAreaDateTimeLayout = "02.01.2006 15:04:05"

// WorkAreaDevicePresentation містить підготовлений текст для лейблів вкладки "Стан".
type WorkAreaDevicePresentation struct {
	DeviceTypeText   string
	PanelMarkText    string
	GroupsText       string
	PowerText        string
	SummaryPowerText string
	SIMText          string
	SIM1Text         string
	SIM2Text         string
	SIM1Value        string
	SIM2Value        string
	SIMCopyText      string
	AutoTestText     string
	GuardText        string
	SummaryModeText  string
	ConnectionText   string
	ChannelText      string
	PhoneText        string
	PhoneCopyText    string
	AkbText          string
	TestControlText  string
	NotesText        string
	NotesCopyText    string
	LocationText     string
	LocationCopyText string
}

// WorkAreaExternalPresentation містить динамічні дані від зовнішніх джерел.
type WorkAreaExternalPresentation struct {
	SignalText          string
	LastTestText        string
	LastTestTimeText    string
	LastMessageTimeText string
	SummarySignalText   string
	SummaryActivityText string
}

// WorkAreaDeviceViewModel інкапсулює форматування даних вкладки "Стан".
type WorkAreaDeviceViewModel struct{}

func NewWorkAreaDeviceViewModel() *WorkAreaDeviceViewModel {
	return &WorkAreaDeviceViewModel{}
}

func (vm *WorkAreaDeviceViewModel) BuildObjectPresentation(obj models.Object) WorkAreaDevicePresentation {
	powerText := buildWorkAreaPowerSummary(obj)

	sim1 := strings.TrimSpace(obj.SIM1)
	sim2 := strings.TrimSpace(obj.SIM2)
	simText := "SIM1: —"
	sim1Text := "📱 SIM1: —"
	sim2Text := "📱 SIM2: —"
	copySimText := ""
	if sim1 != "" {
		simText = "SIM1: " + sim1
		sim1Text = "📱 SIM1: " + sim1
		copySimText = sim1
	}
	if sim2 != "" {
		simText += " | SIM2: " + sim2
		sim2Text = "📱 SIM2: " + sim2
		if copySimText != "" {
			copySimText += " / " + sim2
		} else {
			copySimText = sim2
		}
	}
	if copySimText == "" {
		copySimText = "—"
	}

	groupsText := "🔐 Групи: —"
	if len(obj.Groups) > 0 {
		lines := make([]string, 0, len(obj.Groups))
		for _, group := range obj.Groups {
			labelParts := []string{fmt.Sprintf("Група %d", group.Number)}
			if groupName := strings.TrimSpace(group.Name); groupName != "" {
				labelParts = append(labelParts, groupName)
			} else if roomName := strings.TrimSpace(group.RoomName); roomName != "" {
				labelParts = append(labelParts, roomName)
			}

			stateText := strings.TrimSpace(group.StateText)
			if stateText == "" {
				if group.Armed {
					stateText = "ПІД ОХОРОНОЮ"
				} else {
					stateText = "ЗНЯТО"
				}
			}
			labelParts = append(labelParts, stateText)

			lines = append(lines, strings.Join(labelParts, " | "))
		}
		groupsText = "🔐 Групи:\n" + strings.Join(lines, "\n")
	}

	channelText := "Інший канал"
	switch obj.ObjChan {
	case 1:
		channelText = "Автододзвон"
	case 5:
		channelText = "GPRS"
	}

	akbText := buildWorkAreaBatterySummary(obj)

	autoTestText := "⏱️ Автотест: —"
	if obj.AutoTestHours > 0 {
		autoTestText = fmt.Sprintf("⏱️ Автотест: кожні %d год", obj.AutoTestHours)
	} else if obj.TestControl > 0 && obj.TestTime > 0 {
		autoTestText = "⏱️ Автотест: " + formatWorkAreaTestInterval(obj.TestTime)
	}

	testCtrlText := "Виключено"
	if obj.TestControl > 0 {
		testCtrlText = "Активно"
		if obj.TestTime > 0 {
			testCtrlText += " (" + formatWorkAreaTestInterval(obj.TestTime) + ")"
		}
	}

	guardText := buildWorkAreaGuardSummary(obj)
	connectionText := buildWorkAreaConnectionSummary(obj)

	deviceType := strings.TrimSpace(obj.DeviceType)
	if deviceType == "" {
		deviceType = "—"
	}
	panelMark := strings.TrimSpace(obj.PanelMark)
	if panelMark == "" {
		panelMark = "—"
	}
	phone := strings.TrimSpace(obj.Phones1)
	if phone == "" {
		phone = strings.TrimSpace(obj.Phone)
	}
	if ids.IsCASLObjectID(obj.ID) || phone == "" {
		phone = "—"
	}

	return WorkAreaDevicePresentation{
		DeviceTypeText:   "🔧 Тип: " + deviceType,
		PanelMarkText:    "🏷️ Марка: " + panelMark,
		GroupsText:       groupsText,
		PowerText:        "🔌 " + powerText,
		SummaryPowerText: powerText,
		SIMText:          "📱 " + simText,
		SIM1Text:         sim1Text,
		SIM2Text:         sim2Text,
		SIM1Value:        sim1,
		SIM2Value:        sim2,
		SIMCopyText:      copySimText,
		AutoTestText:     autoTestText,
		GuardText:        guardText,
		SummaryModeText:  guardText,
		ConnectionText:   connectionText,
		ChannelText:      "📡 Канал: " + channelText,
		PhoneText:        "☎️ Тел. об'єкта: " + phone,
		PhoneCopyText:    phone,
		AkbText:          "🔋 АКБ: " + akbText,
		TestControlText:  "⏲️ Контроль тесту: " + testCtrlText,
		NotesText:        obj.Notes1,
		NotesCopyText:    obj.Notes1,
		LocationText:     obj.Location1,
		LocationCopyText: obj.Location1,
	}
}

func buildWorkAreaPowerSummary(obj models.Object) string {
	if ids.IsCASLObjectID(obj.ID) {
		powerText, powerKnown, powerAlarm := buildWorkAreaCASLPowerState(obj.PowerFault)
		batteryText, batteryKnown, batteryAlarm := buildWorkAreaCASLBatteryState(obj.AkbState)

		switch {
		case powerKnown && batteryKnown:
			return powerText + ", " + batteryText
		case powerKnown && !batteryKnown:
			return powerText + ", АКБ невідомо"
		case !powerKnown && batteryKnown:
			return "220В невідоме, " + batteryText
		case powerAlarm:
			return "220В відсутнє, АКБ невідомо"
		case batteryAlarm:
			return "220В невідоме, АКБ тривога"
		default:
			return "Стан живлення невідомий"
		}
	}

	hasBatteryIssue := obj.AkbState != 0
	hasMainsIssue := obj.PowerFault != 0

	switch {
	case hasMainsIssue && hasBatteryIssue:
		return "220В відсутнє, АКБ тривога"
	case hasMainsIssue:
		return "220В відсутнє, резерв АКБ"
	case obj.PowerSource == models.PowerBattery:
		return "Резерв АКБ"
	case hasBatteryIssue:
		return "220В в нормі, АКБ тривога"
	default:
		return "220В в нормі"
	}
}

func buildWorkAreaBatterySummary(obj models.Object) string {
	if ids.IsCASLObjectID(obj.ID) {
		switch obj.AkbState {
		case 0:
			return "Тривога"
		case 1:
			return "Норма"
		default:
			return "Невідомо"
		}
	}

	if obj.AkbState != 0 {
		return "ТРИВОГА (Розряд/Відсутній)"
	}
	return "Норма"
}

func buildWorkAreaCASLPowerState(raw int64) (text string, known bool, alarm bool) {
	switch raw {
	case 0:
		return "220В відсутнє", true, true
	case 1:
		return "220В в нормі", true, false
	default:
		return "220В невідоме", false, false
	}
}

func buildWorkAreaCASLBatteryState(raw int64) (text string, known bool, alarm bool) {
	switch raw {
	case 0:
		return "АКБ тривога", true, true
	case 1:
		return "АКБ в нормі", true, false
	default:
		return "АКБ невідомо", false, false
	}
}

func buildWorkAreaConnectionSummary(obj models.Object) string {
	if obj.Status == models.StatusOffline {
		return "Немає зв'язку"
	}
	if obj.IsConnState > 0 || obj.IsConnOK {
		return "На зв'язку"
	}
	return "На зв'язку"
}

func buildWorkAreaGuardSummary(obj models.Object) string {
	if strings.Contains(strings.ToUpper(strings.TrimSpace(obj.StatusText)), "ЧАСТКОВО") {
		return "Частково без охорони"
	}

	if len(obj.Groups) > 1 {
		hasArmed := false
		hasDisarmed := false
		for _, group := range obj.Groups {
			stateText := strings.ToUpper(strings.TrimSpace(group.StateText))
			switch {
			case strings.Contains(stateText, "ЧАСТКОВО"):
				return "Частково без охорони"
			case strings.Contains(stateText, "БЕЗ ОХОРОНИ"), strings.Contains(stateText, "ЗНЯТО"):
				hasDisarmed = true
			default:
				if group.Armed {
					hasArmed = true
				}
			}
		}
		if hasArmed && hasDisarmed {
			return "Частково без охорони"
		}
	}

	switch {
	case ids.IsPhoenixObjectID(obj.ID) && obj.BlockedArmedOnOff == 1:
		return "Заблоковано"
	case ids.IsPhoenixObjectID(obj.ID) && obj.BlockedArmedOnOff == 2:
		return "Стенди"
	case obj.BlockedArmedOnOff == 1:
		return "Знято зі спостереження"
	case obj.BlockedArmedOnOff == 2:
		return "Режим налагодження"
	case obj.GuardState == 0 || !obj.IsUnderGuard:
		if ids.IsPhoenixObjectID(obj.ID) {
			return "Без охорони"
		}
		return "Знято з охорони"
	default:
		return "Під охороною"
	}
}

func formatWorkAreaTestInterval(minutes int64) string {
	if minutes <= 0 {
		return "—"
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("кожні %d год", minutes/60)
	}
	return fmt.Sprintf("кожні %d хв", minutes)
}

func (vm *WorkAreaDeviceViewModel) BuildLoadingExternalPresentation() WorkAreaExternalPresentation {
	return WorkAreaExternalPresentation{
		SignalText:          "📶 Рівень: ...",
		LastTestText:        "📝 Тест: ...",
		LastTestTimeText:    "📅 Ост. тест: ...",
		LastMessageTimeText: "📅 Ост. подія: ...",
		SummarySignalText:   "—",
		SummaryActivityText: "—",
	}
}

func (vm *WorkAreaDeviceViewModel) BuildExternalPresentation(signal, testMsg string, lastTest, lastMessage time.Time) WorkAreaExternalPresentation {
	lastTestTimeText := "📅 Ост. тест: —"
	if !lastTest.IsZero() {
		lastTestTimeText = "📅 Ост. тест: " + lastTest.Format(workAreaDateTimeLayout)
	}

	lastMessageTimeText := "📅 Ост. подія: —"
	if !lastMessage.IsZero() {
		lastMessageTimeText = "📅 Ост. подія: " + lastMessage.Format(workAreaDateTimeLayout)
	}

	return WorkAreaExternalPresentation{
		SignalText:          "📶 Рівень: " + signal,
		LastTestText:        "📝 Тест: " + testMsg,
		LastTestTimeText:    lastTestTimeText,
		LastMessageTimeText: lastMessageTimeText,
		SummarySignalText:   emptyWorkAreaValue(signal),
		SummaryActivityText: formatWorkAreaTimestamp(lastMessage),
	}
}

func formatWorkAreaTimestamp(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Format(workAreaDateTimeLayout)
}

func emptyWorkAreaValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "..." {
		return "—"
	}
	return value
}
