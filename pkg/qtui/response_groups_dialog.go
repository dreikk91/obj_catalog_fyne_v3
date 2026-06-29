//go:build qt

package qtui

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
)

const (
	responseGroupsAllSources = "Всі джерела"
	responseGroupsAllStates  = "Всі стани"
)

type ResponseGroupsReload func(done func([]contracts.FrontendResponseGroup, error))

func ShowResponseGroupsDialog(
	parent *qt.QWidget,
	initialGroups []contracts.FrontendResponseGroup,
	reload ResponseGroupsReload,
) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Групи реагування")
	dialog.Resize(1050, 650)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	toolbar := qt.NewQHBoxLayout2()

	search := qt.NewQLineEdit2()
	search.SetPlaceholderText("Пошук за назвою, позивним, телефоном або об'єктом")
	search.SetClearButtonEnabled(true)

	sourceFilter := qt.NewQComboBox2()
	sourceFilter.AddItems([]string{responseGroupsAllSources, contracts.FrontendSourceCASL.DisplayName(), contracts.FrontendSourcePhoenix.DisplayName()})

	statusFilter := qt.NewQComboBox2()
	statusFilter.AddItems([]string{responseGroupsAllStates, "Вільні", "Направлені", "Прибули", "Невідомий стан"})

	summary := qt.NewQLabel3("")
	summary.SetStyleSheet("font-weight: 700;")
	autoRefresh := qt.NewQCheckBox3("Автооновлення 10 с")
	autoRefresh.SetChecked(true)
	refreshState := qt.NewQLabel3("")
	refreshState.SetStyleSheet("color: #666;")

	toolbar.AddWidget(search.QWidget)
	toolbar.AddWidget(sourceFilter.QWidget)
	toolbar.AddWidget(statusFilter.QWidget)
	toolbar.AddWidget(summary.QWidget)
	toolbar.AddWidget(autoRefresh.QWidget)
	layout.AddLayout(toolbar.QLayout)

	table := qt.NewQTableWidget3(0, 7)
	table.SetHorizontalHeaderLabels([]string{"Джерело", "МГР", "Позивний", "Стан", "Поточний об'єкт", "Телефон", "Зміна стану"})
	table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	table.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	table.SetAlternatingRowColors(true)
	table.SetSortingEnabled(true)
	table.VerticalHeader().SetVisible(false)
	table.HorizontalHeader().SetStretchLastSection(false)
	layout.AddWidget(table.QWidget)

	groups := append([]contracts.FrontendResponseGroup(nil), initialGroups...)
	fill := func() {
		filtered := filterResponseGroups(groups, search.Text(), sourceFilter.CurrentText(), statusFilter.CurrentText())
		summary.SetText(responseGroupsSummary(filtered, len(groups)))
		table.SetSortingEnabled(false)
		table.SetRowCount(len(filtered))
		for row, group := range filtered {
			values := []string{
				group.Source.DisplayName(),
				responseGroupDisplayName(group),
				strings.TrimSpace(group.Callsign),
				responseGroupDisplayStatus(group),
				responseGroupObjectText(group),
				strings.TrimSpace(group.Phone),
				responseGroupChangedAtText(group),
			}
			for column, value := range values {
				item := qt.NewQTableWidgetItem2(value)
				item.SetData(int(qt.BackgroundRole), responseGroupBackground(group.Status))
				table.SetItem(row, column, item)
			}
		}
		table.SetSortingEnabled(true)
		table.ResizeColumnsToContents()
		table.HorizontalHeader().SetSectionResizeMode(qt.QHeaderView__Interactive)
		table.HorizontalHeader().SetSectionResizeMode2(1, qt.QHeaderView__Stretch)
		table.HorizontalHeader().SetSectionResizeMode2(4, qt.QHeaderView__Stretch)
	}

	search.OnTextChanged(func(string) { fill() })
	sourceFilter.OnCurrentTextChanged(func(string) { fill() })
	statusFilter.OnCurrentTextChanged(func(string) { fill() })

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	refreshButton := buttons.AddButton2("Оновити", qt.QDialogButtonBox__ActionRole)
	buttonRow := qt.NewQHBoxLayout2()
	buttonRow.AddWidget(refreshState.QWidget)
	buttonRow.AddStretch()
	buttonRow.AddWidget(buttons.QWidget)
	layout.AddLayout(buttonRow.QLayout)

	refreshing := false
	var closed atomic.Bool
	requestRefresh := func() {
		if closed.Load() || refreshing || reload == nil {
			return
		}
		refreshing = true
		refreshButton.SetEnabled(false)
		refreshState.SetText("Оновлення...")
		reload(func(updated []contracts.FrontendResponseGroup, err error) {
			if closed.Load() {
				return
			}
			refreshing = false
			refreshButton.SetEnabled(true)
			if err != nil {
				refreshState.SetText("Помилка оновлення: " + strings.TrimSpace(err.Error()))
				return
			}
			groups = append(groups[:0], updated...)
			refreshState.SetText("Оновлено: " + responseGroupsRefreshTime())
			fill()
		})
	}
	refreshButton.OnClicked(requestRefresh)
	buttons.OnRejected(dialog.Reject)

	timer := qt.NewQTimer2(dialog.QObject)
	timer.SetInterval(10_000)
	timer.OnTimeout(func() {
		if autoRefresh.IsChecked() {
			requestRefresh()
		}
	})
	timer.Start2()
	dialog.OnFinished(func(int) {
		closed.Store(true)
		timer.Stop()
	})
	dialog.SetLayout(layout.QLayout)

	fill()
	refreshState.SetText("Оновлено: " + responseGroupsRefreshTime())
	dialog.Exec()
}

func responseGroupsRefreshTime() string {
	return time.Now().Format("15:04:05")
}

func filterResponseGroups(
	groups []contracts.FrontendResponseGroup,
	query string,
	sourceText string,
	statusText string,
) []contracts.FrontendResponseGroup {
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]contracts.FrontendResponseGroup, 0, len(groups))
	for _, group := range groups {
		if sourceText != "" && sourceText != responseGroupsAllSources && group.Source.DisplayName() != sourceText {
			continue
		}
		if !responseGroupStatusMatches(group.Status, statusText) {
			continue
		}
		searchText := strings.ToLower(strings.Join([]string{
			group.ID,
			group.Name,
			group.Callsign,
			group.Phone,
			group.ObjectNumber,
			group.ObjectName,
		}, " "))
		if query != "" && !strings.Contains(searchText, query) {
			continue
		}
		result = append(result, group)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := responseGroupStatusOrder(result[i].Status)
		right := responseGroupStatusOrder(result[j].Status)
		if left != right {
			return left < right
		}
		return strings.ToLower(responseGroupDisplayName(result[i])) < strings.ToLower(responseGroupDisplayName(result[j]))
	})
	return result
}

func responseGroupStatusMatches(status contracts.ResponseGroupStatus, filter string) bool {
	switch filter {
	case "", responseGroupsAllStates:
		return true
	case "Вільні":
		return status == contracts.ResponseGroupStatusFree
	case "Направлені":
		return status == contracts.ResponseGroupStatusDispatched
	case "Прибули":
		return status == contracts.ResponseGroupStatusArrived
	case "Невідомий стан":
		return status == contracts.ResponseGroupStatusUnknown
	default:
		return true
	}
}

func responseGroupStatusOrder(status contracts.ResponseGroupStatus) int {
	switch status {
	case contracts.ResponseGroupStatusDispatched:
		return 0
	case contracts.ResponseGroupStatusArrived:
		return 1
	case contracts.ResponseGroupStatusFree:
		return 2
	default:
		return 3
	}
}

func responseGroupDisplayName(group contracts.FrontendResponseGroup) string {
	if name := strings.TrimSpace(group.Name); name != "" {
		return name
	}
	return "МГР " + strings.TrimSpace(group.ID)
}

func responseGroupDisplayStatus(group contracts.FrontendResponseGroup) string {
	if text := strings.TrimSpace(group.StatusText); text != "" {
		return text
	}
	switch group.Status {
	case contracts.ResponseGroupStatusFree:
		return "Вільна"
	case contracts.ResponseGroupStatusDispatched:
		return "Направлена"
	case contracts.ResponseGroupStatusArrived:
		return "Прибула"
	default:
		return "Невідомо"
	}
}

func responseGroupObjectText(group contracts.FrontendResponseGroup) string {
	number := strings.TrimSpace(group.ObjectNumber)
	name := strings.TrimSpace(group.ObjectName)
	switch {
	case number != "" && name != "":
		return "№" + number + " " + name
	case number != "":
		return "№" + number
	default:
		return name
	}
}

func responseGroupChangedAtText(group contracts.FrontendResponseGroup) string {
	if group.StatusChangedAt.IsZero() {
		return ""
	}
	return group.StatusChangedAt.Local().Format("02.01.2006 15:04:05")
}

func responseGroupsSummary(filtered []contracts.FrontendResponseGroup, total int) string {
	active := 0
	for _, group := range filtered {
		if group.Status == contracts.ResponseGroupStatusDispatched || group.Status == contracts.ResponseGroupStatusArrived {
			active++
		}
	}
	return fmt.Sprintf("Показано: %d/%d | активні: %d", len(filtered), total, active)
}

func responseGroupBackground(status contracts.ResponseGroupStatus) *qt.QVariant {
	switch status {
	case contracts.ResponseGroupStatusDispatched:
		return qt.NewQColor11(255, 242, 204, 255).ToQVariant()
	case contracts.ResponseGroupStatusArrived:
		return qt.NewQColor11(230, 244, 234, 255).ToQVariant()
	case contracts.ResponseGroupStatusFree:
		return qt.NewQColor11(255, 255, 255, 255).ToQVariant()
	default:
		return qt.NewQColor11(242, 242, 242, 255).ToQVariant()
	}
}
