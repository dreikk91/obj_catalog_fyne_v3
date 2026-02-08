package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"obj_catalog_fyne_v3/pkg/models"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#3C3C3C")).
			MarginBottom(1)

	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4"))

	normalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3C3C3C"))

	tabStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#3C3C3C")).
			Padding(0, 1)

	activeTabStyle = tabStyle.Copy().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Bold(true)
)

func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing..."
	}

	if m.Mode == ModeProcessAlarm {
		return m.renderProcessAlarmDialog()
	}
	if m.Mode == ModeSettings {
		return m.renderSettingsDialog()
	}
	if m.Mode == ModeTestMessages {
		return m.renderTestMessagesDialog()
	}

	header := m.renderHeader()

	// Main content split
	leftWidth := m.Width / 3
	rightWidth := m.Width - leftWidth - 4 // minus borders/padding

	leftContent := m.renderLeftPanel(leftWidth)
	rightContent := m.renderRightPanel(rightWidth)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftContent, rightContent)

	// Bottom panel
	bottomContent := m.renderBottomPanel()

	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		mainContent,
		bottomContent,
		footer,
	)
}

func (m Model) renderHeader() string {
	title := titleStyle.Render("АРМ Пожежної Безпеки v1.0")
	status := fmt.Sprintf(" Останнє оновлення: %s", m.LastUpdate.Format("15:04:05"))
	return headerStyle.Width(m.Width).Render(lipgloss.JoinHorizontal(lipgloss.Center, title, status))
}

func (m Model) renderLeftPanel(width int) string {
	style := normalStyle
	if m.Focus == FocusObjectList {
		style = focusedStyle
	}

	return style.Width(width).Height(m.Height * 2 / 3).Render(m.ObjectList.View())
}

func (m Model) renderRightPanel(width int) string {
	style := normalStyle
	if m.Focus == FocusWorkArea {
		style = focusedStyle
	}

	var content string
	if m.SelectedObject == nil {
		content = "\n\n   ← Оберіть об'єкт зі списку"
	} else {
		content = m.renderWorkArea(width)
	}

	return style.Width(width).Height(m.Height * 2 / 3).Render(content)
}

func (m Model) renderWorkArea(width int) string {
	obj := m.SelectedObject

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s (№%d)", obj.Name, obj.ID))
	address := lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("📍 %s | 📄 %s", obj.Address, obj.ContractNum))

	statusColor := "#34C759" // theme.ColorNormal
	switch obj.Status {
	case models.StatusFire:
		statusColor = "#FF3B30"
	case models.StatusFault:
		statusColor = "#FFCC00"
	case models.StatusOffline:
		statusColor = "#888888"
	}
	status := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true).Render(obj.GetStatusDisplay())

	// Tabs
	tabs := []string{"📊 Стан", "🔌 Зони", "👥 Відповідальні", "📜 Журнал"}
	var renderedTabs []string
	for i, t := range tabs {
		if i == m.WorkAreaTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(t))
		} else {
			renderedTabs = append(renderedTabs, tabStyle.Render(t))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	var tabContent string
	switch m.WorkAreaTab {
	case 0:
		tabContent = m.renderSummaryTab()
	case 1:
		tabContent = m.renderZonesTab()
	case 2:
		tabContent = m.renderContactsTab()
	case 3:
		tabContent = m.renderObjectEventsTab()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		address,
		status,
		"",
		row,
		"",
		tabContent,
	)
}

func (m Model) renderSummaryTab() string {
	obj := m.SelectedObject
	if obj == nil { return "" }

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("🔧 Тип: %s\n", obj.DeviceType))
	sb.WriteString(fmt.Sprintf("🏷️ Марка: %s\n", obj.PanelMark))

	powerText := "220В (мережа)"
	if obj.PowerSource == models.PowerBattery {
		powerText = "🔋 АКБ (резерв)"
	}
	sb.WriteString(fmt.Sprintf("🔌 Живлення: %s\n", powerText))
	sb.WriteString(fmt.Sprintf("📱 SIM: %s | %s\n", obj.SIM1, obj.SIM2))
	sb.WriteString(fmt.Sprintf("☎️ Тел: %s\n", obj.Phones1))

	guardText := "🔒 ПІД ОХОРОНОЮ"
	if !obj.IsUnderGuard {
		guardText = "🔓 ЗНЯТО З ОХОРОНИ"
	}
	sb.WriteString(fmt.Sprintf("🛡️ Стан: %s\n", guardText))

	return sb.String()
}

func (m Model) renderZonesTab() string {
	if len(m.Zones) == 0 {
		return "Немає даних про зони"
	}
	sb := strings.Builder{}
	for _, z := range m.Zones {
		sb.WriteString(fmt.Sprintf("№%d: %s (%s) - %s\n", z.Number, z.Name, z.SensorType, z.GetStatusDisplay()))
	}
	return sb.String()
}

func (m Model) renderContactsTab() string {
	if len(m.Contacts) == 0 {
		return "Немає даних про відповідальних осіб"
	}
	sb := strings.Builder{}
	for _, c := range m.Contacts {
		sb.WriteString(fmt.Sprintf("👤 %s (%s) - 📞 %s\n", c.Name, c.Position, c.Phone))
	}
	return sb.String()
}

func (m Model) renderObjectEventsTab() string {
	if len(m.ObjectEvents) == 0 {
		return "Немає подій"
	}
	sb := strings.Builder{}
	for i, e := range m.ObjectEvents {
		if i > 10 { break } // Limit display
		sb.WriteString(fmt.Sprintf("%s | %s | %s\n", e.GetDateTimeDisplay(), e.GetTypeDisplay(), e.Details))
	}
	return sb.String()
}

func (m Model) renderBottomPanel() string {
	style := normalStyle
	if m.Focus == FocusBottomPanel {
		style = focusedStyle
	}

	// Tabs
	tabs := []string{"📜 Журнал подій", "🚨 Активні тривоги"}
	var renderedTabs []string
	for i, t := range tabs {
		if i == m.BottomTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(t))
		} else {
			renderedTabs = append(renderedTabs, tabStyle.Render(t))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	var content string
	if m.BottomTab == 0 {
		content = m.EventLog.View()
	} else {
		content = m.AlarmList.View()
	}

	return style.Width(m.Width - 2).Height(m.Height / 3).Render(
		lipgloss.JoinVertical(lipgloss.Left, row, content),
	)
}

func (m Model) renderFooter() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render(" [Tab] Focus [←/→] Tabs [Enter] Process [m] TestMsg [c] Copy [s] Settings [q] Exit")
}

func (m Model) renderProcessAlarmDialog() string {
	if m.ActiveAlarm == nil {
		return "No active alarm selected"
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Width(60)

	title := lipgloss.NewStyle().Bold(true).Render("ОБРОБКА ТРИВОГИ")
	info := fmt.Sprintf("Об'єкт: %s\nТип: %s\nЧас: %s",
		m.ActiveAlarm.ObjectName, m.ActiveAlarm.GetTypeDisplay(), m.ActiveAlarm.GetDateTimeDisplay())

	var actions []string
	for i, a := range m.AlarmActions {
		if i == m.AlarmActionIndex {
			actions = append(actions, lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("> "+a))
		} else {
			actions = append(actions, "  "+a)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		info,
		"",
		"Результат обробки:",
		lipgloss.JoinVertical(lipgloss.Left, actions...),
		"",
		"Примітка:",
		m.AlarmNoteInput.View(),
		"",
		" [Enter] Підтвердити  [Esc] Скасувати",
	)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, style.Render(content))
}

func (m Model) renderTestMessagesDialog() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Width(70).
		Height(20)

	title := lipgloss.NewStyle().Bold(true).Render("ТЕСТОВІ ПОВІДОМЛЕННЯ")

	var rows []string
	for i, msg := range m.TestMessages {
		if i > 15 { break }
		rows = append(rows, fmt.Sprintf("%s | %s", msg.Time.Format("02.01 15:04"), msg.Info))
	}

	if len(rows) == 0 {
		rows = append(rows, "Немає повідомлень")
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		"",
		" [Esc] Назад",
	)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, style.Render(content))
}

func (m Model) renderSettingsDialog() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Width(60)

	title := lipgloss.NewStyle().Bold(true).Render("НАЛАШТУВАННЯ")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		"Налаштування бази даних (settings.json):",
		" (В даній версії TUI редагування через інтерфейс обмежене)",
		"",
		" [Esc] Назад",
	)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, style.Render(content))
}
