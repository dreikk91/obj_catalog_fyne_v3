package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"obj_catalog_fyne_v3/pkg/models"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	statusNormalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759"))
	statusFireStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
	statusFaultStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCC00"))
	statusOfflineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

type objectDelegate struct{}

func (d objectDelegate) Height() int                               { return 2 }
func (d objectDelegate) Spacing() int                              { return 0 }
func (d objectDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d objectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(objectItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.obj.Name)
	desc := i.obj.Address

	statusIcon := "● "
	var statusStyle lipgloss.Style
	switch i.obj.Status {
	case models.StatusFire:
		statusStyle = statusFireStyle
	case models.StatusFault:
		statusStyle = statusFaultStyle
	case models.StatusOffline:
		statusStyle = statusOfflineStyle
	default:
		statusStyle = statusNormalStyle
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(s...)
		}
	}

	fmt.Fprintf(w, "%s%s\n%s", statusStyle.Render(statusIcon), fn(str), itemStyle.Faint(true).Render(desc))
}

type eventDelegate struct{}

func (d eventDelegate) Height() int                               { return 1 }
func (d eventDelegate) Spacing() int                              { return 0 }
func (d eventDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d eventDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(eventItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s | %s | %s", i.event.GetDateTimeDisplay(), i.event.GetTypeDisplay(), i.event.Details)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(s...)
		}
	}

	fmt.Fprintf(w, "%s", fn(str))
}

type alarmDelegate struct{}

func (d alarmDelegate) Height() int                               { return 1 }
func (d alarmDelegate) Spacing() int                              { return 0 }
func (d alarmDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d alarmDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(alarmItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("🚨 %s | %s | %s", i.alarm.GetDateTimeDisplay(), i.alarm.ObjectName, i.alarm.GetTypeDisplay())

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(s...)
		}
	}

	fmt.Fprintf(w, "%s", statusFireStyle.Render(fn(str)))
}
