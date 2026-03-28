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
	PowerText        string
	SIMText          string
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
	if obj.PowerSource == models.PowerBattery {
		powerText = "🔋 АКБ (резерв)"
	}

	simText := "SIM1: " + strings.TrimSpace(obj.SIM1)
	copySimText := strings.TrimSpace(obj.SIM1)
	if strings.TrimSpace(obj.SIM2) != "" {
		simText += " | SIM2: " + strings.TrimSpace(obj.SIM2)
		if copySimText != "" {
			copySimText += " / " + strings.TrimSpace(obj.SIM2)
		} else {
			copySimText = strings.TrimSpace(obj.SIM2)
		}
	}

	channelText := "Інший канал"
	switch obj.ObjChan {
	case 1:
		channelText = "Автододзвон"
	case 5:
		channelText = "GPRS"
	}

	akbText := "Норма"
	if obj.AkbState > 0 {
		akbText = "ТРИВОГА (Розряд/Відсутній)"
	}

	testCtrlText := "Виключено"
	if obj.TestControl > 0 {
		testCtrlText = fmt.Sprintf("Активно (кожні %d хв)", obj.TestTime)
	}

	guardText := "🔒 ПІД ОХОРОНОЮ"
	if !obj.IsUnderGuard {
		guardText = "🔓 ЗНЯТО З ОХОРОНИ"
	}

	return WorkAreaDevicePresentation{
		DeviceTypeText:   "🔧 Тип: " + obj.DeviceType,
		PanelMarkText:    "🏷️ Марка: " + obj.PanelMark,
		PowerText:        "🔌 " + powerText,
		SIMText:          "📱 " + simText,
		SIMCopyText:      copySimText,
		AutoTestText:     fmt.Sprintf("⏱️ Автотест: кожні %d год", obj.AutoTestHours),
		GuardText:        guardText,
		ChannelText:      "📡 Канал: " + channelText,
		PhoneText:        "☎️ Тел. об'єкта: " + obj.Phones1,
		PhoneCopyText:    obj.Phones1,
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
