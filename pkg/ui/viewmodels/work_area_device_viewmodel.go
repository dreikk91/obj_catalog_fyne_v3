package viewmodels

import (
	"fmt"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

const workAreaDateTimeLayout = "02.01.2006 15:04:05"

// WorkAreaDevicePresentation містить підготовлений текст для лейблів вкладки "Стан".
type WorkAreaDevicePresentation struct {
	DeviceTypeText   string
	PanelMarkText    string
	GroupsText       string
	PowerText        string
	SIMText          string
	SIM1Text         string
	SIM2Text         string
	SIM1Value        string
	SIM2Value        string
	SIMCopyText      string
	AutoTestText     string
	GuardText        string
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
}

// WorkAreaDeviceViewModel інкапсулює форматування даних вкладки "Стан".
type WorkAreaDeviceViewModel struct{}

func NewWorkAreaDeviceViewModel() *WorkAreaDeviceViewModel {
	return &WorkAreaDeviceViewModel{}
}

func (vm *WorkAreaDeviceViewModel) BuildObjectPresentation(obj models.Object) WorkAreaDevicePresentation {
	powerText := "220В (мережа)"
	if obj.PowerSource == models.PowerBattery || obj.PowerFault != 0 {
		powerText = "🔋 АКБ (резерв)"
	}

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

	akbText := "Норма"
	if obj.AkbState != 0 {
		akbText = "ТРИВОГА (Розряд/Відсутній)"
	}

	testCtrlText := "Виключено"
	if obj.TestControl > 0 {
		testCtrlText = fmt.Sprintf("Активно (кожні %d хв)", obj.TestTime)
	}

	guardText := "🔒 ПІД ОХОРОНОЮ"
	if IsPhoenixObjectID(obj.ID) && obj.BlockedArmedOnOff == 1 {
		guardText = "⛔ ЗАБЛОКОВАНО"
	} else if IsPhoenixObjectID(obj.ID) && obj.BlockedArmedOnOff == 2 {
		guardText = "🧪 СТЕНДИ"
	} else if !obj.IsUnderGuard {
		if IsPhoenixObjectID(obj.ID) {
			guardText = "🔓 БЕЗ ОХОРОНИ"
		} else {
			guardText = "🔓 ЗНЯТО З ОХОРОНИ"
		}
	}

	deviceType := strings.TrimSpace(obj.DeviceType)
	if deviceType == "" {
		deviceType = "—"
	}
	panelMark := strings.TrimSpace(obj.PanelMark)
	if panelMark == "" {
		panelMark = "—"
	}
	phone := strings.TrimSpace(obj.Phones1)
	if IsCASLObjectID(obj.ID) || phone == "" {
		phone = "—"
	}

	return WorkAreaDevicePresentation{
		DeviceTypeText:   "🔧 Тип: " + deviceType,
		PanelMarkText:    "🏷️ Марка: " + panelMark,
		GroupsText:       groupsText,
		PowerText:        "🔌 " + powerText,
		SIMText:          "📱 " + simText,
		SIM1Text:         sim1Text,
		SIM2Text:         sim2Text,
		SIM1Value:        sim1,
		SIM2Value:        sim2,
		SIMCopyText:      copySimText,
		AutoTestText:     fmt.Sprintf("⏱️ Автотест: кожні %d год", obj.AutoTestHours),
		GuardText:        guardText,
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

func (vm *WorkAreaDeviceViewModel) BuildLoadingExternalPresentation() WorkAreaExternalPresentation {
	return WorkAreaExternalPresentation{
		SignalText:          "📶 Рівень: ...",
		LastTestText:        "📝 Тест: ...",
		LastTestTimeText:    "📅 Ост. тест: ...",
		LastMessageTimeText: "📅 Ост. подія: ...",
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
	}
}
