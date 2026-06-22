//go:build qt

package qtui

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/utils"
)

func runOnMainThread(f func()) {
	timer := qt.NewQTimer()
	timer.SetSingleShot(true)
	timer.OnTimeout(func() {
		f()
		timer.Delete()
	})
	timer.Start(0)
}

func colorToHTML(c color.NRGBA) string {
	return fmt.Sprintf("rgba(%d,%d,%d,%f)", c.R, c.G, c.B, float64(c.A)/255.0)
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func eventColorsForSC1(sc1 int) (string, string) {
	textColor, rowColor := utils.SelectColorNRGBA(sc1)
	return colorToHTML(textColor), colorToHTML(rowColor)
}

func eventRowColorsBySeverity(severity models.VisualSeverity, sc1 int) (textColor, rowColor color.NRGBA) {
	if severity != models.VisualSeverityUnknown {
		sc1 = sc1FromVisualSeverity(severity, sc1)
	}
	return utils.SelectColorNRGBA(sc1)
}

func sc1FromVisualSeverity(severity models.VisualSeverity, fallback int) int {
	switch severity {
	case models.VisualSeverityCritical:
		return 1
	case models.VisualSeverityWarning:
		return 2
	case models.VisualSeverityInfo:
		return 10
	case models.VisualSeverityNormal:
		if fallback != 0 {
			return fallback
		}
		return 10
	default:
		return fallback
	}
}

func alarmSourceDisplayName(objectID int) string {
	switch {
	case ids.IsCASLObjectID(objectID):
		return "CASL"
	case ids.IsPhoenixObjectID(objectID):
		return "Phoenix"
	default:
		return "БД/МІСТ"
	}
}

func formatAlarmSourceMessageText(msg models.AlarmMsg) string {
	text := "—"
	if !msg.Time.IsZero() {
		text = msg.Time.Local().Format("02.01.2006 15:04:05")
	}

	state := "Подія"
	if msg.IsAlarm {
		state = "Тривога"
	}
	text += " | " + state

	if msg.Number > 0 {
		text += " | Зона " + strconv.Itoa(msg.Number)
	}

	details := strings.TrimSpace(msg.Details)
	code := strings.TrimSpace(msg.Code)
	contactID := strings.TrimSpace(msg.ContactID)
	switch {
	case details != "":
		text += " — " + details
	case code != "":
		text += " — " + code
	case contactID != "":
		text += " — " + contactID
	}

	if code != "" && details != "" {
		text += " [code=" + code + "]"
	}
	if contactID != "" && details != "" {
		text += " [cid=" + contactID + "]"
	}
	return text
}

func alarmSourceMessageSC1(msg models.AlarmMsg) int {
	sc1 := msg.SC1
	if sc1 == 0 {
		if msg.IsAlarm {
			sc1 = 1
		} else {
			sc1 = 6
		}
	}
	return sc1
}

func filterAlarmSourceMessagesSince(alarm models.Alarm, sourceMsgs []models.AlarmMsg) []models.AlarmMsg {
	if len(sourceMsgs) == 0 {
		return nil
	}

	msgs := append([]models.AlarmMsg(nil), sourceMsgs...)
	if alarm.Time.IsZero() {
		return msgs
	}

	filtered := make([]models.AlarmMsg, 0, len(msgs))
	for _, msg := range msgs {
		if !msg.Time.IsZero() && msg.Time.Before(alarm.Time) {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func prepareSourceMessagesForDisplay(alarm models.Alarm, sourceMsgs []models.AlarmMsg, bridgeHistoryMode string) []models.AlarmMsg {
	if len(sourceMsgs) == 0 {
		return nil
	}

	msgs := append([]models.AlarmMsg(nil), sourceMsgs...)
	if ids.IsCASLObjectID(alarm.ObjectID) {
		return msgs
	}
	if !ids.IsCASLObjectID(alarm.ObjectID) &&
		!ids.IsPhoenixObjectID(alarm.ObjectID) &&
		config.NormalizeBridgeAlarmHistoryMode(bridgeHistoryMode) == config.BridgeAlarmHistoryModeActiveOnly {
		return msgs
	}

	return filterAlarmSourceMessagesSince(alarm, msgs)
}

func caseHistoryEventText(event models.Event) string {
	parts := []string{event.GetDateTimeDisplay()}
	if icon := getEventIcon(event.Type); icon != "" {
		parts = append(parts, icon)
	}
	parts = append(parts, strings.TrimSpace(event.GetTypeDisplay()))

	line := strings.Join(parts, " ")
	if event.ZoneNumber > 0 {
		line += " | Зона " + strconv.Itoa(event.ZoneNumber)
	}
	if user := strings.TrimSpace(event.UserName); user != "" {
		line += " | " + user
	}
	if details := strings.TrimSpace(event.Details); details != "" {
		line += " — " + details
	}
	return line
}

func getEventIcon(eventType models.EventType) string {
	switch eventType {
	case models.EventFire:
		return "🔴"
	case models.EventBurglary:
		return "🚨"
	case models.EventPanic:
		return "🆘"
	case models.EventMedical:
		return "🩺"
	case models.EventGas:
		return "☣"
	case models.EventTamper:
		return "🔧"
	case models.EventAlarmNotification:
		return "📥"
	case models.EventOperatorAction:
		return "👤"
	case models.EventManagerAssigned:
		return "🚓"
	case models.EventManagerArrived:
		return "✅"
	case models.EventManagerCanceled:
		return "↩"
	case models.EventAlarmFinished:
		return "✔"
	case models.EventDeviceBlocked:
		return "⛔"
	case models.EventDeviceUnblocked:
		return "🔓"
	case models.EventArm, models.EventDisarm:
		return "🔵"
	case models.EventRestore, models.EventOnline, models.EventPowerOK:
		return "🟢"
	default:
		return "⚪"
	}
}

func newPlaceholder(text string) *qt.QWidget {
	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)
	label := qt.NewQLabel3(text)
	label.SetWordWrap(true)
	label.SetAlignment(qt.AlignCenter)
	layout.AddWidget(label.QWidget)
	widget.SetLayout(layout.QLayout)
	return widget
}

func addReadOnlyRow(model *qt.QStandardItemModel, values []string) {
	items := make([]*qt.QStandardItem, 0, len(values))
	for _, value := range values {
		item := qt.NewQStandardItem2(value)
		item.SetEditable(false)
		items = append(items, item)
	}
	model.AppendRow(items)
}

func addColoredReadOnlyRow(model *qt.QStandardItemModel, values []string, objectID int, textColor color.NRGBA, rowColor color.NRGBA) {
	items := make([]*qt.QStandardItem, 0, len(values))
	foreground := qt.NewQColor11(int(textColor.R), int(textColor.G), int(textColor.B), int(textColor.A)).ToQVariant()
	background := qt.NewQColor11(int(rowColor.R), int(rowColor.G), int(rowColor.B), int(rowColor.A)).ToQVariant()
	for idx, value := range values {
		item := qt.NewQStandardItem2(value)
		item.SetEditable(false)
		item.SetData(foreground, int(qt.ForegroundRole))
		item.SetData(background, int(qt.BackgroundRole))
		if idx == 0 {
			item.SetData(qt.NewQVariant4(objectID), int(qt.UserRole))
		}
		items = append(items, item)
	}
	model.AppendRow(items)
}

func newTable(model *qt.QStandardItemModel, headers []string) *qt.QTableView {
	model.SetHorizontalHeaderLabels(headers)
	table := qt.NewQTableView2()
	table.SetModel(model.QAbstractItemModel)
	table.SetSortingEnabled(true)
	table.SetAlternatingRowColors(true)
	table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	table.HorizontalHeader().SetStretchLastSection(true)
	return table
}

func newTree(model *qt.QStandardItemModel, headers []string) *qt.QTreeView {
	model.SetHorizontalHeaderLabels(headers)
	tree := qt.NewQTreeView2()
	tree.SetModel(model.QAbstractItemModel)
	tree.SetSortingEnabled(true)
	tree.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	tree.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	tree.SetRootIsDecorated(true)
	tree.SetUniformRowHeights(true)
	tree.Header().SetStretchLastSection(true)
	return tree
}

func wrapWidget(child *qt.QWidget) *qt.QWidget {
	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)
	layout.AddWidget(child)
	widget.SetLayout(layout.QLayout)
	return widget
}

func workAreaHeaderAddress(object models.Object) string {
	parts := make([]string, 0, 4)
	if address := strings.TrimSpace(object.Address); address != "" {
		parts = append(parts, address)
	}
	if phone := firstNonEmpty(object.Phones1, object.Phone); phone != "" {
		parts = append(parts, "тел. "+phone)
	}
	if contract := strings.TrimSpace(object.ContractNum); contract != "" {
		parts = append(parts, "договір "+contract)
	}
	parts = append(parts, viewmodels.ObjectSourceByID(object.ID))
	return strings.Join(parts, " | ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func setZoneRows(model *qt.QStandardItemModel, zones []models.Zone) {
	model.Clear()
	model.SetHorizontalHeaderLabels(zoneTreeHeaders())
	root := model.InvisibleRootItem()
	for _, group := range groupZones(zones) {
		groupItem := newReadOnlyItem(group.numberText)
		groupItem.SetData(qt.NewQColor11(232, 240, 254, 255).ToQVariant(), int(qt.BackgroundRole))
		parentRow := []*qt.QStandardItem{
			groupItem,
			newReadOnlyItem(group.name),
			newReadOnlyItem(""),
			newReadOnlyItem(fmt.Sprintf("%d зон", len(group.zones))),
			newReadOnlyItem(""),
			newReadOnlyItem(group.state),
		}
		for _, item := range parentRow[1:] {
			item.SetData(qt.NewQColor11(232, 240, 254, 255).ToQVariant(), int(qt.BackgroundRole))
		}
		root.AppendRow(parentRow)
		for _, zone := range group.zones {
			groupItem.AppendRow([]*qt.QStandardItem{
				newReadOnlyItem(group.numberText),
				newReadOnlyItem(group.name),
				newReadOnlyItem(fmt.Sprintf("%d", zone.Number)),
				newReadOnlyItem(strings.TrimSpace(zone.Name)),
				newReadOnlyItem(strings.TrimSpace(zone.SensorType)),
				newReadOnlyItem(zone.GetStatusDisplay()),
			})
		}
	}
}

func zoneTreeHeaders() []string {
	return []string{"Група №", "Назва групи", "Зона №", "Назва зони", "Тип", "Стан"}
}

type zoneGroup struct {
	key        string
	numberText string
	name       string
	state      string
	zones      []models.Zone
}

func groupZones(zones []models.Zone) []zoneGroup {
	sorted := append([]models.Zone(nil), zones...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].GroupNumber != sorted[j].GroupNumber {
			return sorted[i].GroupNumber < sorted[j].GroupNumber
		}
		if sorted[i].GroupName != sorted[j].GroupName {
			return sorted[i].GroupName < sorted[j].GroupName
		}
		return sorted[i].Number < sorted[j].Number
	})

	indexByKey := map[string]int{}
	groups := make([]zoneGroup, 0)
	for _, zone := range sorted {
		key := zoneGroupKey(zone)
		idx, ok := indexByKey[key]
		if !ok {
			idx = len(groups)
			indexByKey[key] = idx
			groups = append(groups, zoneGroup{
				key:        key,
				numberText: zoneGroupNumberText(zone),
				name:       zoneGroupName(zone),
				state:      emptyDash(zone.GroupStateText),
			})
		}
		groups[idx].zones = append(groups[idx].zones, zone)
	}
	return groups
}

func zoneGroupKey(zone models.Zone) string {
	if id := strings.TrimSpace(zone.GroupID); id != "" {
		return id
	}
	return fmt.Sprintf("%d|%s", zone.GroupNumber, strings.TrimSpace(zone.GroupName))
}

func zoneGroupNumberText(zone models.Zone) string {
	if zone.GroupNumber > 0 {
		return fmt.Sprintf("%d", zone.GroupNumber)
	}
	return "-"
}

func zoneGroupName(zone models.Zone) string {
	if name := strings.TrimSpace(zone.GroupName); name != "" {
		return name
	}
	if zone.GroupNumber > 0 {
		return fmt.Sprintf("Група %d", zone.GroupNumber)
	}
	return "Без групи"
}

func newReadOnlyItem(value string) *qt.QStandardItem {
	item := qt.NewQStandardItem2(emptyDash(value))
	item.SetEditable(false)
	return item
}

func setContactRows(model *qt.QStandardItemModel, contacts []models.Contact) {
	model.Clear()
	model.SetHorizontalHeaderLabels([]string{"Ім'я", "Посада", "Телефон", "Група"})
	for _, contact := range contacts {
		group := strings.TrimSpace(contact.GroupName)
		if group == "" && contact.GroupNumber > 0 {
			group = fmt.Sprintf("%d", contact.GroupNumber)
		}
		addReadOnlyRow(model, []string{
			strings.TrimSpace(contact.Name),
			contactPositionText(contact),
			strings.TrimSpace(contact.Phone),
			group,
		})
	}
}

func setEventRows(model *qt.QStandardItemModel, events []models.Event) {
	model.Clear()
	model.SetHorizontalHeaderLabels([]string{"Час", "Подія", "Опис"})
	for _, event := range events {
		textColor, rowColor := eventRowColors(event)
		addColoredReadOnlyRow(model, []string{
			event.GetDateTimeDisplay(),
			event.GetTypeDisplay(),
			strings.TrimSpace(event.Details),
		}, event.ObjectID, textColor, rowColor)
	}
}

func setGlobalEventRows(model *qt.QStandardItemModel, events []models.Event) {
	model.Clear()
	model.SetHorizontalHeaderLabels([]string{"Час", "№", "Подія", "Об'єкт", "Опис", "Джерело"})
	for _, event := range events {
		textColor, rowColor := eventRowColors(event)
		addColoredReadOnlyRow(model, []string{
			event.GetDateTimeDisplay(),
			eventObjectNumber(event),
			event.GetTypeDisplay(),
			strings.TrimSpace(event.ObjectName),
			strings.TrimSpace(event.Details),
			viewmodels.ObjectSourceByID(event.ObjectID),
		}, event.ObjectID, textColor, rowColor)
	}
}

func eventObjectNumber(event models.Event) string {
	if number := strings.TrimSpace(event.ObjectNumber); number != "" {
		return number
	}
	if event.ObjectID > 0 {
		return fmt.Sprintf("%d", event.ObjectID)
	}
	return "-"
}

func eventRowColors(event models.Event) (color.NRGBA, color.NRGBA) {
	if event.IsCritical() {
		return color.NRGBA{R: 120, G: 0, B: 0, A: 255}, color.NRGBA{R: 255, G: 225, B: 225, A: 255}
	}
	if event.IsWarning() {
		return color.NRGBA{R: 80, G: 60, B: 0, A: 255}, color.NRGBA{R: 255, G: 245, B: 205, A: 255}
	}
	return color.NRGBA{R: 0, G: 0, B: 0, A: 255}, color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}

func resizeTableToContents(table *qt.QTableView) {
	if table == nil {
		return
	}
	table.ResizeColumnsToContents()
	table.HorizontalHeader().SetSectionResizeMode(qt.QHeaderView__Interactive)
	table.HorizontalHeader().SetStretchLastSection(true)
}

func resizeTreeToContents(tree *qt.QTreeView) {
	if tree == nil {
		return
	}
	for column := 0; column < tree.Model().ColumnCount(qt.NewQModelIndex()); column++ {
		tree.ResizeColumnToContents(column)
	}
	tree.ExpandAll()
	tree.Header().SetSectionResizeMode(qt.QHeaderView__Interactive)
	tree.Header().SetStretchLastSection(true)
}

func resizeObjectListColumns(table *qt.QTableView) {
	if table == nil {
		return
	}
	table.ResizeColumnsToContents()
	header := table.HorizontalHeader()
	header.SetStretchLastSection(false)
	header.SetSectionResizeMode(qt.QHeaderView__Interactive)
	table.SetColumnWidth(0, maxInt(table.ColumnWidth(0), 72))
	header.SetSectionResizeMode2(0, qt.QHeaderView__ResizeToContents)
	header.SetSectionResizeMode2(1, qt.QHeaderView__Stretch)
	header.SetSectionResizeMode2(2, qt.QHeaderView__Stretch)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func indexForNormalizedStatusFilter(combo *qt.QComboBox, normalized string) int {
	for i := 0; i < combo.Count(); i++ {
		if viewmodels.NormalizeObjectListFilter(combo.ItemText(i)) == normalized {
			return i
		}
	}
	return 0
}

func indexForNormalizedSourceFilter(combo *qt.QComboBox, normalized string) int {
	for i := 0; i < combo.Count(); i++ {
		if viewmodels.NormalizeObjectSourceFilter(combo.ItemText(i)) == normalized {
			return i
		}
	}
	return 0
}

func filterAlarmsBySeverity(alarms []models.Alarm, severity string) []models.Alarm {
	switch severity {
	case "Критичні":
		filtered := make([]models.Alarm, 0, len(alarms))
		for _, alarm := range alarms {
			if alarm.IsCritical() {
				filtered = append(filtered, alarm)
			}
		}
		return filtered
	case "Звичайні":
		filtered := make([]models.Alarm, 0, len(alarms))
		for _, alarm := range alarms {
			if !alarm.IsCritical() {
				filtered = append(filtered, alarm)
			}
		}
		return filtered
	default:
		return alarms
	}
}

func updateComboItems(combo *qt.QComboBox, options []string, current string) {
	if combo == nil || len(options) == 0 {
		return
	}
	target := 0
	for index, option := range options {
		if strings.HasPrefix(option, current+" (") || option == current {
			target = index
			break
		}
	}
	combo.Clear()
	combo.AddItems(options)
	combo.SetCurrentIndex(target)
}
