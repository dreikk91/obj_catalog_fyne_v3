//go:build qt

package qtui

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/models"
)

type caseHistoryTreeRow struct {
	Time       string
	Event      string
	Context    string
	Zone       string
	TextColor  string
	Background string
}

func alarmMessageHistoryTreeRow(msg models.AlarmMsg, textColor string, background string) caseHistoryTreeRow {
	event := strings.TrimSpace(msg.Details)
	if event == "" {
		event = strings.TrimSpace(firstNonEmpty(msg.Code, msg.ContactID))
	}
	if event == "" {
		event = "Подія"
	}
	if msg.IsAlarm {
		event = "Тривога — " + event
	}

	contextParts := make([]string, 0, 2)
	if code := strings.TrimSpace(msg.Code); code != "" {
		contextParts = append(contextParts, code)
	}
	if contactID := strings.TrimSpace(msg.ContactID); contactID != "" {
		contextParts = append(contextParts, "CID "+contactID)
	}

	return caseHistoryTreeRow{
		Time:       historyTreeTime(msg.Time),
		Event:      event,
		Context:    strings.Join(contextParts, " · "),
		Zone:       historyTreeZone(msg.Number),
		TextColor:  textColor,
		Background: background,
	}
}

func eventHistoryTreeRow(event models.Event, textColor string, background string) caseHistoryTreeRow {
	eventText := strings.TrimSpace(event.GetTypeDisplay())
	if icon := getEventIcon(event.Type); icon != "" {
		eventText = strings.TrimSpace(icon + " " + eventText)
	}
	if details := strings.TrimSpace(event.Details); details != "" {
		eventText += " — " + details
	}

	return caseHistoryTreeRow{
		Time:       historyTreeTime(event.Time),
		Event:      eventText,
		Context:    strings.TrimSpace(event.UserName),
		Zone:       historyTreeZone(event.ZoneNumber),
		TextColor:  textColor,
		Background: background,
	}
}

func historyTreeTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Local().Format("02.01.2006 15:04:05")
}

func historyTreeZone(number int) string {
	if number <= 0 {
		return ""
	}
	return strconv.Itoa(number)
}

func (panel *AlarmPanel) setCaseHistoryTreeStatus(title string, status string) {
	panel.setCaseHistoryTree(title, "", []caseHistoryTreeRow{{Event: strings.TrimSpace(status)}})
}

func (panel *AlarmPanel) setCaseHistoryTreeRows(title string, rows []caseHistoryTreeRow) {
	panel.setCaseHistoryTree(title, historyTreeCountLabel(len(rows)), rows)
}

func (panel *AlarmPanel) setCaseHistoryTree(title string, countLabel string, rows []caseHistoryTreeRow) {
	if panel == nil || panel.historyTree == nil || panel.historyModel == nil {
		return
	}

	panel.historyModel.Clear()
	panel.historyModel.SetHorizontalHeaderLabels([]string{"Час", "Подія", "Код / оператор", "Зона"})

	root := panel.historyModel.InvisibleRootItem()
	titleItem := newReadOnlyItem(strings.TrimSpace(title))
	countItem := newReadOnlyItem(countLabel)
	parentRow := []*qt.QStandardItem{
		titleItem,
		countItem,
		newReadOnlyItem(""),
		newReadOnlyItem(""),
	}
	setHistoryTreeRowColors(parentRow, qtPrimaryColor, qtAltSurfaceColor)
	root.AppendRow(parentRow)

	for _, row := range rows {
		items := []*qt.QStandardItem{
			newReadOnlyItem(row.Time),
			newReadOnlyItem(row.Event),
			newReadOnlyItem(row.Context),
			newReadOnlyItem(row.Zone),
		}
		for _, item := range items {
			item.SetToolTip(strings.TrimSpace(strings.Join([]string{row.Time, row.Event, row.Context, row.Zone}, " | ")))
		}
		setHistoryTreeRowColors(items, row.TextColor, row.Background)
		titleItem.AppendRow(items)
	}

	rootIndex := titleItem.Index()
	panel.historyTree.Expand(rootIndex)
	panel.historyTree.SetColumnWidth(0, 155)
	panel.historyTree.SetColumnWidth(1, 560)
	panel.historyTree.SetColumnWidth(2, 180)
	panel.historyTree.SetColumnWidth(3, 60)
	runtime.KeepAlive(rootIndex)
}

func historyTreeCountLabel(count int) string {
	switch {
	case count%10 == 1 && count%100 != 11:
		return fmt.Sprintf("%d подія", count)
	case count%10 >= 2 && count%10 <= 4 && (count%100 < 12 || count%100 > 14):
		return fmt.Sprintf("%d події", count)
	default:
		return fmt.Sprintf("%d подій", count)
	}
}

func setHistoryTreeRowColors(items []*qt.QStandardItem, textColor string, background string) {
	textColor = strings.TrimSpace(textColor)
	background = strings.TrimSpace(background)
	for _, item := range items {
		if item == nil {
			continue
		}
		if textColor != "" {
			item.SetForeground(qt.NewQBrush3(qt.NewQColor6(textColor)))
		}
		if background != "" {
			item.SetBackground(qt.NewQBrush3(qt.NewQColor6(background)))
		}
	}
}
