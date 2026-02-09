package tui

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/models"
)

type focusArea int

const (
	FocusObjectList focusArea = iota
	FocusWorkArea
	FocusBottomPanel
)

type appMode int

const (
	ModeNormal appMode = iota
	ModeProcessAlarm
	ModeSettings
	ModeTestMessages
)

type Model struct {
	DataProvider data.DataProvider

	// Data
	Objects      []models.Object
	Alarms       []models.Alarm
	Events       []models.Event

	// UI Components
	ObjectList   list.Model
	AlarmList    list.Model
	EventLog     list.Model
	WorkAreaViewport viewport.Model

	// Selection State
	SelectedObject *models.Object
	Zones          []models.Zone
	Contacts       []models.Contact
	ObjectEvents   []models.Event

	// Navigation State
	Focus        focusArea
	WorkAreaTab  int // 0: Summary, 1: Zones, 2: Contacts, 3: Events
	BottomTab    int // 0: Event Log, 1: Active Alarms
	Mode         appMode

	// Terminal
	Width        int
	Height       int

	// Internal
	LastUpdate   time.Time

	// Process Alarm state
	ActiveAlarm  *models.Alarm
	AlarmActions []string
	AlarmActionIndex int
	AlarmNoteInput   textinput.Model

	// Test Messages state
	TestMessages []models.TestMessage
}

func NewModel(provider data.DataProvider) Model {
	// Initialize lists with empty items for now
	m := Model{
		DataProvider: provider,
		Focus:        FocusObjectList,
		WorkAreaTab:  0,
		BottomTab:    0,
		Mode:         ModeNormal,
		LastUpdate:   time.Now(),
		AlarmActions: []string{
			"Виклик пожежників",
			"Виклик ГШР",
			"Помилкова тривога",
			"Технічна несправність",
			"Контрольна перевірка",
			"Інше",
		},
	}

	m.AlarmNoteInput = textinput.New()
	m.AlarmNoteInput.Placeholder = "Примітка..."

	// We will initialize lists properly in Init or when data arrives
	m.ObjectList = list.New([]list.Item{}, objectDelegate{}, 0, 0)
	m.ObjectList.Title = "Об'єкти"

	m.AlarmList = list.New([]list.Item{}, alarmDelegate{}, 0, 0)
	m.AlarmList.Title = "Активні тривоги"

	m.EventLog = list.New([]list.Item{}, eventDelegate{}, 0, 0)
	m.EventLog.Title = "Журнал подій"

	m.WorkAreaViewport = viewport.New(0, 0)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchObjects,
		m.fetchAlarms,
		m.fetchEvents,
		tick(),
	)
}

// Commands
type msgFetchObjects []models.Object
type msgFetchAlarms []models.Alarm
type msgFetchEvents []models.Event
type msgTick time.Time

func (m Model) fetchObjects() tea.Msg {
	return msgFetchObjects(m.DataProvider.GetObjects())
}

func (m Model) fetchAlarms() tea.Msg {
	return msgFetchAlarms(m.DataProvider.GetAlarms())
}

func (m Model) fetchEvents() tea.Msg {
	return msgFetchEvents(m.DataProvider.GetEvents())
}

func tick() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
		return msgTick(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Mode == ModeProcessAlarm {
			switch msg.String() {
			case "esc":
				m.Mode = ModeNormal
				m.AlarmNoteInput.Blur()
				return m, nil
			case "up":
				m.AlarmActionIndex = (m.AlarmActionIndex - 1 + len(m.AlarmActions)) % len(m.AlarmActions)
			case "down":
				m.AlarmActionIndex = (m.AlarmActionIndex + 1) % len(m.AlarmActions)
			case "tab":
				if m.AlarmNoteInput.Focused() {
					m.AlarmNoteInput.Blur()
				} else {
					m.AlarmNoteInput.Focus()
				}
			case "enter":
				var cmd tea.Cmd
				if m.ActiveAlarm != nil {
					cmd = m.processAlarmCmd(fmt.Sprintf("%d", m.ActiveAlarm.ID), "Диспетчер (TUI)", m.AlarmNoteInput.Value())
				}
				m.Mode = ModeNormal
				m.AlarmNoteInput.Blur()
				return m, tea.Batch(cmd, m.fetchAlarms)
			}

			if m.AlarmNoteInput.Focused() {
				var cmd tea.Cmd
				m.AlarmNoteInput, cmd = m.AlarmNoteInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		if m.Mode == ModeSettings || m.Mode == ModeTestMessages {
			switch msg.String() {
			case "esc":
				m.Mode = ModeNormal
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			m.Mode = ModeSettings
			return m, nil
		case "m":
			if m.SelectedObject != nil {
				m.Mode = ModeTestMessages
				return m, m.fetchTestMessages(m.SelectedObject.ID)
			}
		case "c":
			if m.SelectedObject != nil {
				clipboard.WriteAll(fmt.Sprintf("%s (%s)", m.SelectedObject.Name, m.SelectedObject.Address))
			}
		case "tab":
			m.Focus = (m.Focus + 1) % 3
		case "shift+tab":
			m.Focus = (m.Focus - 1 + 3) % 3
		case "right":
			if m.Focus == FocusWorkArea {
				m.WorkAreaTab = (m.WorkAreaTab + 1) % 4
				m.WorkAreaViewport.GotoTop()
			} else if m.Focus == FocusBottomPanel {
				m.BottomTab = (m.BottomTab + 1) % 2
			}
		case "left":
			if m.Focus == FocusWorkArea {
				m.WorkAreaTab = (m.WorkAreaTab - 1 + 4) % 4
				m.WorkAreaViewport.GotoTop()
			} else if m.Focus == FocusBottomPanel {
				m.BottomTab = (m.BottomTab - 1 + 2) % 2
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.updateLayout()

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			headerHeight := 2
			mainHeight := m.Height - headerHeight - 1
			bottomHeight := mainHeight / 3
			topHeight := mainHeight - bottomHeight

			if msg.Y >= headerHeight {
				if msg.Y < headerHeight+topHeight {
					if msg.X < m.Width/3 {
						m.Focus = FocusObjectList
					} else {
						m.Focus = FocusWorkArea
					}
				} else if msg.Y < m.Height-1 {
					m.Focus = FocusBottomPanel
				}
			}
		}

	case msgFetchObjects:
		m.Objects = msg
		items := make([]list.Item, len(msg))
		for i, obj := range msg {
			items[i] = objectItem{obj: obj}
		}
		m.ObjectList.SetItems(items)

	case msgFetchAlarms:
		m.Alarms = msg
		items := make([]list.Item, len(msg))
		for i, alarm := range msg {
			items[i] = alarmItem{alarm: alarm}
		}
		m.AlarmList.SetItems(items)

	case msgFetchEvents:
		m.Events = msg
		items := make([]list.Item, len(msg))
		for i, event := range msg {
			items[i] = eventItem{event: event}
		}
		m.EventLog.SetItems(items)

	case msgObjectDetails:
		if msg.Object != nil {
			m.SelectedObject = msg.Object
		}
		m.Zones = msg.Zones
		m.Contacts = msg.Contacts
		m.ObjectEvents = msg.Events

	case []models.TestMessage:
		m.TestMessages = msg

	case msgTick:
		m.LastUpdate = time.Time(msg)
		return m, tea.Batch(
			m.fetchObjects,
			m.fetchAlarms,
			m.fetchEvents,
			tick(),
		)
	}

	// Mouse handling and focus-specific updates
	headerHeight := 2
	mainHeight := m.Height - headerHeight - 1
	bottomHeight := mainHeight / 3
	topHeight := mainHeight - bottomHeight

	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		// Pass translated mouse messages to non-focused components
		if mouseMsg.Y >= headerHeight && mouseMsg.Y < headerHeight+topHeight && mouseMsg.X < m.Width/3 {
			if m.Focus != FocusObjectList {
				translatedMsg := mouseMsg
				translatedMsg.Y -= headerHeight
				var cmd tea.Cmd
				m.ObjectList, cmd = m.ObjectList.Update(translatedMsg)
				cmds = append(cmds, cmd)
			}
		}
		if mouseMsg.Y >= headerHeight+topHeight && mouseMsg.Y < m.Height-1 {
			if m.Focus != FocusBottomPanel {
				translatedMsg := mouseMsg
				translatedMsg.Y -= (headerHeight + topHeight)
				var cmd tea.Cmd
				if m.BottomTab == 0 {
					m.EventLog, cmd = m.EventLog.Update(translatedMsg)
				} else {
					m.AlarmList, cmd = m.AlarmList.Update(translatedMsg)
				}
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update focused component
	var cmd tea.Cmd
	switch m.Focus {
	case FocusObjectList:
		if mouseMsg, ok := msg.(tea.MouseMsg); ok {
			translatedMsg := mouseMsg
			translatedMsg.Y -= headerHeight
			m.ObjectList, cmd = m.ObjectList.Update(translatedMsg)
		} else {
			m.ObjectList, cmd = m.ObjectList.Update(msg)
		}
		cmds = append(cmds, cmd)

		// Additional key handling for object list
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter", " ":
				m.Focus = FocusWorkArea
			}
		}
	case FocusWorkArea:
		var cmd tea.Cmd
		m.WorkAreaViewport, cmd = m.WorkAreaViewport.Update(msg)
		cmds = append(cmds, cmd)
	case FocusBottomPanel:
		if m.BottomTab == 0 {
			if mouseMsg, ok := msg.(tea.MouseMsg); ok {
				translatedMsg := mouseMsg
				translatedMsg.Y -= (headerHeight + topHeight)
				m.EventLog, cmd = m.EventLog.Update(translatedMsg)
			} else {
				m.EventLog, cmd = m.EventLog.Update(msg)
			}
			cmds = append(cmds, cmd)
		} else {
			if mouseMsg, ok := msg.(tea.MouseMsg); ok {
				translatedMsg := mouseMsg
				translatedMsg.Y -= (headerHeight + topHeight)
				m.AlarmList, cmd = m.AlarmList.Update(translatedMsg)
			} else {
				m.AlarmList, cmd = m.AlarmList.Update(msg)
			}
			cmds = append(cmds, cmd)

			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
				if item := m.AlarmList.SelectedItem(); item != nil {
					alarm := item.(alarmItem).alarm
					m.ActiveAlarm = &alarm
					m.Mode = ModeProcessAlarm
					m.AlarmNoteInput.SetValue("")
					m.AlarmNoteInput.Focus()
				}
			}
		}
	}

	// Always sync selection from lists
	if m.Focus == FocusObjectList {
		if item := m.ObjectList.SelectedItem(); item != nil {
			obj := item.(objectItem).obj
			if m.SelectedObject == nil || m.SelectedObject.ID != obj.ID {
				m.SelectedObject = &obj
				cmds = append(cmds, m.fetchObjectDetails(obj.ID))
			}
		}
	} else if m.Focus == FocusBottomPanel {
		if m.BottomTab == 0 {
			if item := m.EventLog.SelectedItem(); item != nil {
				event := item.(eventItem).event
				if m.SelectedObject == nil || m.SelectedObject.ID != event.ObjectID {
					obj := m.DataProvider.GetObjectByID(fmt.Sprintf("%d", event.ObjectID))
					if obj != nil {
						m.SelectedObject = obj
						cmds = append(cmds, m.fetchObjectDetails(obj.ID))
					}
				}
			}
		} else {
			if item := m.AlarmList.SelectedItem(); item != nil {
				alarm := item.(alarmItem).alarm
				if m.SelectedObject == nil || m.SelectedObject.ID != alarm.ObjectID {
					obj := m.DataProvider.GetObjectByID(fmt.Sprintf("%d", alarm.ObjectID))
					if obj != nil {
						m.SelectedObject = obj
						cmds = append(cmds, m.fetchObjectDetails(obj.ID))
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateLayout() {
	headerHeight := 2
	footerHeight := 1
	mainHeight := m.Height - headerHeight - footerHeight
	bottomHeight := mainHeight / 3
	topHeight := mainHeight - bottomHeight

	listWidth := m.Width / 3

	m.ObjectList.SetSize(listWidth-2, topHeight-2)

	m.EventLog.SetSize(m.Width-2, bottomHeight-4)
	m.AlarmList.SetSize(m.Width-2, bottomHeight-4)

	m.WorkAreaViewport.Width = m.Width - listWidth - 4
	m.WorkAreaViewport.Height = topHeight - 8
}

type msgObjectDetails struct {
	Object   *models.Object
	Zones    []models.Zone
	Contacts []models.Contact
	Events   []models.Event
}

func (m Model) fetchObjectDetails(id int) tea.Cmd {
	return func() tea.Msg {
		idStr := fmt.Sprintf("%d", id)
		return msgObjectDetails{
			Object:   m.DataProvider.GetObjectByID(idStr),
			Zones:    m.DataProvider.GetZones(idStr),
			Contacts: m.DataProvider.GetEmployees(idStr),
			Events:   m.DataProvider.GetObjectEvents(idStr),
		}
	}
}

func (m Model) fetchTestMessages(id int) tea.Cmd {
	return func() tea.Msg {
		idStr := fmt.Sprintf("%d", id)
		return m.DataProvider.GetTestMessages(idStr)
	}
}

func (m Model) processAlarmCmd(id, user, note string) tea.Cmd {
	return func() tea.Msg {
		m.DataProvider.ProcessAlarm(id, user, note)
		return nil
	}
}

// Item wrappers for list.Model
type objectItem struct {
	obj models.Object
}
func (i objectItem) Title() string       { return fmt.Sprintf("%s (№%d)", i.obj.Name, i.obj.ID) }
func (i objectItem) Description() string { return i.obj.Address }
func (i objectItem) FilterValue() string { return i.obj.Name + " " + i.obj.Address }

type alarmItem struct {
	alarm models.Alarm
}
func (i alarmItem) Title() string       { return fmt.Sprintf("%s - %s", i.alarm.ObjectName, i.alarm.GetTypeDisplay()) }
func (i alarmItem) Description() string { return i.alarm.GetDateTimeDisplay() }
func (i alarmItem) FilterValue() string { return i.alarm.ObjectName }

type eventItem struct {
	event models.Event
}
func (i eventItem) Title() string       { return i.event.GetDateTimeDisplay() + " | " + i.event.GetTypeDisplay() }
func (i eventItem) Description() string { return i.event.Details }
func (i eventItem) FilterValue() string { return i.event.Details }
