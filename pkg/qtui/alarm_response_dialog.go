//go:build qt

package qtui

import (
	"fmt"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

type AlarmResponseAction int

const (
	AlarmResponseNone AlarmResponseAction = iota
	AlarmResponseTake
	AlarmResponseProcess
	AlarmResponseAssign
	AlarmResponseArrived
	AlarmResponseCancel
)

type AlarmResponseInput struct {
	Action  AlarmResponseAction
	GroupID string
}

func ShowAlarmResponseDialog(
	parent *qt.QWidget,
	alarm models.Alarm,
	groups []contracts.FrontendResponseGroup,
	history []models.AlarmMsg,
) (AlarmResponseInput, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Картка реагування")
	dialog.Resize(760, 620)

	layout := qt.NewQVBoxLayout(dialog.QWidget)

	title := qt.NewQLabel3(fmt.Sprintf("№%s  %s", alarm.GetObjectNumberDisplay(), strings.TrimSpace(alarm.ObjectName)))
	title.SetStyleSheet("font-size: 16px; font-weight: 700;")
	layout.AddWidget(title.QWidget)

	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Адреса", responseValueLabel(alarm.Address).QWidget)
	form.AddRow3("Подія", responseValueLabel(alarm.GetTypeDisplay()).QWidget)
	form.AddRow3("Час", responseValueLabel(alarm.GetDateTimeDisplay()).QWidget)
	form.AddRow3("Зона", responseValueLabel(alarmZoneSummary(alarm)).QWidget)
	form.AddRow3("Опис", responseValueLabel(alarm.Details).QWidget)
	form.AddRow3("Оператор", responseValueLabel(alarmOperatorState(alarm)).QWidget)

	state := qt.NewQLabel3(alarmResponseState(alarm))
	state.SetStyleSheet(alarmResponseStateStyle(alarm))
	form.AddRow3("Реагування", state.QWidget)

	groupSelect := qt.NewQComboBox2()
	groupSelect.SetMinimumContentsLength(36)
	availableGroups := selectableResponseGroups(alarm, groups)
	canRespond := alarmResponseActionsAllowed(alarm)
	selectedIndex := -1
	for _, group := range availableGroups {
		label := responseGroupLabel(group)
		groupSelect.AddItem3(label, qt.NewQVariant14(strings.TrimSpace(group.ID)))
		index := groupSelect.Count() - 1
		if strings.TrimSpace(group.ID) == strings.TrimSpace(alarm.ResponseGroupID) {
			selectedIndex = index
		}
	}
	if selectedIndex >= 0 {
		groupSelect.SetCurrentIndex(selectedIndex)
	}
	if len(availableGroups) == 0 {
		groupSelect.AddItem("Немає доступних груп")
		groupSelect.SetEnabled(false)
	}
	form.AddRow3("Група", groupSelect.QWidget)
	layout.AddLayout(form.QLayout)

	historyTitle := qt.NewQLabel3("Хронологія кейсу")
	historyTitle.SetStyleSheet("font-weight: 700; margin-top: 6px;")
	layout.AddWidget(historyTitle.QWidget)

	historyBrowser := qt.NewQTextBrowser(nil)
	historyBrowser.SetMinimumHeight(180)
	historyBrowser.SetHtml(alarmResponseHistoryHTML(alarm, history))
	layout.AddWidget(historyBrowser.QWidget)

	hint := qt.NewQLabel3("Дії виконуються для вибраної тривоги та фіксуються у джерельній системі.")
	hint.SetWordWrap(true)
	hint.SetStyleSheet("color: #555;")
	layout.AddWidget(hint.QWidget)
	layout.AddStretch()

	actions := qt.NewQHBoxLayout2()
	takeButton := qt.NewQPushButton3(alarmPickActionVerb([]models.Alarm{alarm}))
	takeButton.SetEnabled(!alarm.IsInProgress || alarm.CanTakeOver)
	processButton := qt.NewQPushButton3("Причина / завершити")
	processButton.SetEnabled(alarm.CanProcess)
	assignButton := qt.NewQPushButton3("Призначити МГР")
	assignButton.SetEnabled(canRespond && len(availableGroups) > 0 && !alarm.IsResponseGroupDispatched)
	arrivedButton := qt.NewQPushButton3("МГР прибула")
	arrivedButton.SetEnabled(canRespond && alarm.IsResponseGroupDispatched && !alarm.IsResponseGroupArrived)
	cancelButton := qt.NewQPushButton3("Зняти МГР")
	cancelButton.SetEnabled(canRespond && alarm.IsResponseGroupDispatched)
	closeButton := qt.NewQPushButton3("Закрити")

	result := AlarmResponseInput{}
	takeButton.OnClicked(func() {
		result.Action = AlarmResponseTake
		dialog.Accept()
	})
	processButton.OnClicked(func() {
		result.Action = AlarmResponseProcess
		dialog.Accept()
	})
	assignButton.OnClicked(func() {
		result.Action = AlarmResponseAssign
		result.GroupID = strings.TrimSpace(groupSelect.CurrentData().ToString())
		if result.GroupID != "" {
			dialog.Accept()
		}
	})
	arrivedButton.OnClicked(func() {
		result.Action = AlarmResponseArrived
		dialog.Accept()
	})
	cancelButton.OnClicked(func() {
		result.Action = AlarmResponseCancel
		dialog.Accept()
	})
	closeButton.OnClicked(dialog.Reject)

	actions.AddWidget(takeButton.QWidget)
	actions.AddWidget(processButton.QWidget)
	actions.AddWidget(assignButton.QWidget)
	actions.AddWidget(arrivedButton.QWidget)
	actions.AddWidget(cancelButton.QWidget)
	actions.AddStretch()
	actions.AddWidget(closeButton.QWidget)
	layout.AddLayout(actions.QLayout)
	dialog.SetLayout(layout.QLayout)

	if dialog.Exec() != int(qt.QDialog__Accepted) || result.Action == AlarmResponseNone {
		return AlarmResponseInput{}, false
	}
	return result, true
}

func alarmResponseHistoryHTML(alarm models.Alarm, history []models.AlarmMsg) string {
	msgs := prepareSourceMessagesForDisplay(alarm, history, "")
	if len(msgs) == 0 {
		return "<div style='color:#666; padding:8px;'>Додаткових подій кейсу немає.</div>"
	}

	var b strings.Builder
	b.WriteString("<table width='100%' cellpadding='5' cellspacing='0' style='border-collapse:collapse;'>")
	for _, msg := range msgs {
		textColor, rowColor := eventColorsForSC1(alarmSourceMessageSC1(msg))
		weight := "normal"
		if msg.IsAlarm {
			weight = "bold"
		}
		fmt.Fprintf(
			&b,
			"<tr style='background:%s; color:%s; font-weight:%s;'><td style='border-bottom:1px solid #ddd;'>%s</td></tr>",
			rowColor,
			textColor,
			weight,
			htmlEscape(formatAlarmSourceMessageText(msg)),
		)
	}
	b.WriteString("</table>")
	return b.String()
}

func responseValueLabel(value string) *qt.QLabel {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	label := qt.NewQLabel3(value)
	label.SetWordWrap(true)
	label.SetTextInteractionFlags(qt.TextSelectableByMouse)
	return label
}

func alarmOperatorState(alarm models.Alarm) string {
	if !alarm.IsInProgress {
		return "Не взята в роботу"
	}
	if operator := strings.TrimSpace(alarm.InProgressBy); operator != "" {
		return "У роботі: " + operator
	}
	return "У роботі"
}

func alarmResponseState(alarm models.Alarm) string {
	switch {
	case alarm.IsResponseGroupArrived:
		return "МГР прибула" + responseGroupSuffix(alarm.ResponseGroupID)
	case alarm.IsResponseGroupDispatched:
		return "МГР направлена" + responseGroupSuffix(alarm.ResponseGroupID)
	default:
		return "МГР не направлена"
	}
}

func alarmResponseActionsAllowed(alarm models.Alarm) bool {
	source := contracts.DetectFrontendSourceByObjectID(alarm.ObjectID)
	if source == contracts.FrontendSourceCASL || source == contracts.FrontendSourcePhoenix {
		return alarm.IsOwnedByMe
	}
	return true
}

func responseGroupSuffix(groupID string) string {
	if groupID = strings.TrimSpace(groupID); groupID != "" {
		return " (" + groupID + ")"
	}
	return ""
}

func alarmResponseStateStyle(alarm models.Alarm) string {
	switch {
	case alarm.IsResponseGroupArrived:
		return "font-weight: 700; color: #155724; background: #e6f4ea; padding: 5px;"
	case alarm.IsResponseGroupDispatched:
		return "font-weight: 700; color: #8a5a00; background: #fff2cc; padding: 5px;"
	default:
		return "font-weight: 700; color: #555; background: #f2f2f2; padding: 5px;"
	}
}

func responseGroupLabel(group contracts.FrontendResponseGroup) string {
	name := strings.TrimSpace(group.Name)
	if name == "" {
		name = "МГР " + strings.TrimSpace(group.ID)
	}
	parts := []string{name}
	if callsign := strings.TrimSpace(group.Callsign); callsign != "" {
		parts = append(parts, "позивний "+callsign)
	}
	if phone := strings.TrimSpace(group.Phone); phone != "" {
		parts = append(parts, phone)
	}
	if group.Status != "" && group.Status != contracts.ResponseGroupStatusUnknown {
		parts = append(parts, responseGroupDisplayStatus(group))
	}
	return strings.Join(parts, " | ")
}

func selectableResponseGroups(alarm models.Alarm, groups []contracts.FrontendResponseGroup) []contracts.FrontendResponseGroup {
	result := make([]contracts.FrontendResponseGroup, 0, len(groups))
	currentID := strings.TrimSpace(alarm.ResponseGroupID)
	for _, group := range groups {
		switch {
		case strings.TrimSpace(group.ID) == currentID && currentID != "":
			result = append(result, group)
		case group.Status == "", group.Status == contracts.ResponseGroupStatusUnknown, group.Status == contracts.ResponseGroupStatusFree:
			result = append(result, group)
		}
	}
	return result
}
