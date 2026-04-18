package casleditor

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type RoomsView struct {
	vm *EditorViewModel

	Container *fyne.Container

	roomsBox   *fyne.Container
	stateLabel *widget.Label
}

func NewRoomsView(vm *EditorViewModel) *RoomsView {
	v := &RoomsView{
		vm: vm,

		roomsBox:   container.NewVBox(),
		stateLabel: widget.NewLabel("Прив'яжіть до приміщень відповідальних і зони."),
	}
	v.setupLayout()
	v.bind()
	v.refreshRooms()

	return v
}

func (v *RoomsView) setupLayout() {
	roomsScroll := container.NewVScroll(v.roomsBox)
	roomsScroll.SetMinSize(fyne.NewSize(980, 460))
	roomsCard := newWizardPanel("Зв'язка приміщень", container.NewBorder(v.stateLabel, nil, nil, nil, roomsScroll))
	v.Container = container.NewBorder(nil, nil, nil, nil, roomsCard)
}

func (v *RoomsView) bind() {
	v.vm.AddDataChangedListener(func() {
		v.refreshRooms()
	})
}

func (v *RoomsView) Title() string { return "Крок 3. Приміщення та зв'язки" }

func (v *RoomsView) ProgressLabel() string {
	return "Зв'язка приміщення зі шлейфами"
}

func (v *RoomsView) Content() fyne.CanvasObject { return v.Container }

func (v *RoomsView) CommitDraft() error {
	return v.vm.ValidateDraftRooms()
}

func (v *RoomsView) refreshRooms() {
	if v.autoBindSingleRoomLines() {
		return
	}

	rooms := v.vm.Snapshot.Object.Rooms
	items := make([]fyne.CanvasObject, 0, max(len(rooms), 1))
	if len(rooms) == 0 {
		items = append(items, widget.NewLabel("Ще немає приміщень. Додайте хоча б одне приміщення на кроці створення об'єкта."))
	} else {
		for idx, room := range rooms {
			items = append(items, v.buildRoomBindingRow(idx, room))
		}
	}
	v.roomsBox.Objects = items
	v.roomsBox.Refresh()
}

func (v *RoomsView) autoBindSingleRoomLines() bool {
	if len(v.vm.Snapshot.Object.Rooms) != 1 {
		return false
	}

	room := &v.vm.Snapshot.Object.Rooms[0]
	if len(v.vm.Snapshot.Object.Device.Lines) == 0 {
		return false
	}

	existing := make(map[int]struct{}, len(room.Lines))
	for _, link := range room.Lines {
		existing[link.LineNumber] = struct{}{}
	}

	changed := false
	for idx, line := range v.vm.Snapshot.Object.Device.Lines {
		if strings.TrimSpace(line.RoomID) == room.RoomID {
			if _, ok := existing[line.LineNumber]; !ok {
				room.Lines = append(room.Lines, contracts.CASLRoomLineLink{
					LineNumber:    line.LineNumber,
					AdapterType:   line.AdapterType,
					GroupNumber:   line.GroupNumber,
					AdapterNumber: line.AdapterNumber,
				})
				existing[line.LineNumber] = struct{}{}
				changed = true
			}
			continue
		}
		if strings.TrimSpace(line.RoomID) != "" {
			continue
		}
		v.vm.Snapshot.Object.Device.Lines[idx].RoomID = room.RoomID
		room.Lines = append(room.Lines, contracts.CASLRoomLineLink{
			LineNumber:    line.LineNumber,
			AdapterType:   line.AdapterType,
			GroupNumber:   line.GroupNumber,
			AdapterNumber: line.AdapterNumber,
		})
		existing[line.LineNumber] = struct{}{}
		changed = true
	}

	if changed {
		v.vm.initDictionaries()
		v.vm.emitDataChanged()
	}
	return changed
}

func (v *RoomsView) buildRoomBindingRow(index int, room contracts.CASLRoomDetails) fyne.CanvasObject {
	roomTitle := widget.NewLabelWithStyle(FirstNonEmpty(room.Name, "Нове приміщення"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	roomTitle.Wrapping = fyne.TextWrapWord

	allUserOptions := v.userOptions("")
	userSearch := widget.NewSelectEntry(allUserOptions)
	userSearch.SetPlaceHolder("Виберіть відповідального")
	bindUserPicker(userSearch, allUserOptions, func(selected string) {
		userID := v.vm.UserOptionToID[selected]
		if userID == "" || v.roomHasUser(v.vm.Snapshot.Object.Rooms[index], userID) {
			return
		}
		v.vm.SelectRoom(index)
		v.vm.AddUserToRoom(userID)
		userSearch.SetText("")
	})

	addUserBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		v.vm.SelectRoom(index)
		v.vm.CreateUserAndAddToRoom()
	})
	addUserBtn.Importance = widget.LowImportance

	lineOptions, lineOptionToNumber := v.roomAvailableLineOptions(room)
	var lineSelect *widget.Select
	lineSelect = widget.NewSelect(lineOptions, func(selected string) {
		lineNumber, ok := lineOptionToNumber[selected]
		if !ok {
			return
		}
		v.vm.SelectRoom(index)
		v.vm.BindLineToRoom(lineNumber, room.RoomID)
		lineSelect.ClearSelected()
	})
	if len(lineOptions) > 0 {
		lineSelect.PlaceHolder = "Оберіть зону"
	} else {
		lineSelect.PlaceHolder = "Немає доступних зон"
		lineSelect.Disable()
	}

	selectedUsers := v.buildSelectedUsersBox(index, room)
	selectedLines := v.buildSelectedLinesBox(room)

	row := container.NewGridWithColumns(
		4,
		newWizardField("Приміщення", fixedWidthField(roomTitle, 220)),
		newWizardField("Відповідальні", userSearch),
		container.NewVBox(widget.NewLabel(""), addUserBtn),
		newWizardField("Зони", lineSelect),
	)

	content := container.NewVBox(
		row,
		container.NewGridWithColumns(
			2,
			newWizardPanel("Вибрані відповідальні", selectedUsers),
			newWizardPanel("Вибрані зони", selectedLines),
		),
	)

	header := container.NewHBox(
		widget.NewLabelWithStyle(FirstNonEmpty(room.Name, "Нове приміщення"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabel(fmt.Sprintf("Відповідальних: %d | Зон: %d", len(room.Users), len(room.Lines))),
	)
	return newWizardPanelWithHeader(header, content)
}

func (v *RoomsView) buildSelectedUsersBox(roomIndex int, room contracts.CASLRoomDetails) fyne.CanvasObject {
	if len(room.Users) == 0 {
		return widget.NewLabel("Ще немає відповідальних.")
	}
	items := make([]fyne.CanvasObject, 0, len(room.Users))
	for userIndex, user := range room.Users {
		removeBtn := widget.NewButton("Видалити", func() {
			v.vm.SelectRoom(roomIndex)
			v.vm.RemoveUserFromRoom(userIndex)
		})
		removeBtn.Importance = widget.LowImportance

		details := v.vm.RoomUserDetailsText(user)
		if details == "" {
			details = "Без додаткових реквізитів"
		}
		items = append(items, container.NewBorder(
			nil,
			nil,
			nil,
			removeBtn,
			container.NewVBox(
				widget.NewLabel(v.vm.UserLabelByID(user.UserID)),
				widget.NewLabel(details),
			),
		))
	}
	return container.NewVBox(items...)
}

func (v *RoomsView) buildSelectedLinesBox(room contracts.CASLRoomDetails) fyne.CanvasObject {
	if len(room.Lines) == 0 {
		return widget.NewLabel("Ще немає прив'язаних зон.")
	}
	items := make([]fyne.CanvasObject, 0, len(room.Lines))
	for _, lineLink := range sortedRoomLines(room.Lines) {
		items = append(items, widget.NewLabel(v.roomLineLabel(lineLink.LineNumber)))
	}
	return container.NewVBox(items...)
}

func (v *RoomsView) roomAvailableLineOptions(room contracts.CASLRoomDetails) ([]string, map[string]int) {
	options := make([]string, 0, len(v.vm.Snapshot.Object.Device.Lines))
	mapping := make(map[string]int, len(v.vm.Snapshot.Object.Device.Lines))
	for _, line := range v.vm.Snapshot.Object.Device.Lines {
		if line.RoomID != "" && line.RoomID != room.RoomID {
			continue
		}
		label := fmt.Sprintf("#%d | Гр.%d | %s", line.LineNumber, line.GroupNumber, FirstNonEmpty(line.Description, "Без назви"))
		options = append(options, label)
		mapping[label] = line.LineNumber
	}
	sort.Strings(options)
	return options, mapping
}

func (v *RoomsView) roomLineLabel(lineNumber int) string {
	line, ok := v.vm.LineDetailsByNumber(lineNumber)
	if !ok {
		return fmt.Sprintf("#%d", lineNumber)
	}
	return fmt.Sprintf("Гр.%d | #%d | %s", line.GroupNumber, line.LineNumber, FirstNonEmpty(line.Description, "Без назви"))
}

func (v *RoomsView) roomHasUser(room contracts.CASLRoomDetails, userID string) bool {
	for _, link := range room.Users {
		if link.UserID == userID {
			return true
		}
	}
	return false
}

func (v *RoomsView) userOptions(filter string) []string {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return slices.Clone(v.vm.AllUserOptions)
	}
	options := make([]string, 0, len(v.vm.AllUserOptions))
	for _, option := range v.vm.AllUserOptions {
		if strings.Contains(strings.ToLower(option), filter) {
			options = append(options, option)
		}
	}
	return options
}

func bindUserPicker(entry *widget.SelectEntry, allOptions []string, onPicked func(string)) {
	applying := false
	entry.OnChanged = func(value string) {
		if applying {
			return
		}
		options := filterSelectEntryOptions(allOptions, value)
		entry.SetOptions(options)
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || !stringSliceContains(allOptions, trimmed) {
			return
		}
		onPicked(trimmed)
		applying = true
		entry.SetText("")
		entry.SetOptions(slices.Clone(allOptions))
		applying = false
	}
	entry.OnSubmitted = func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		onPicked(trimmed)
		entry.SetText("")
	}
}

func filterSelectEntryOptions(options []string, filter string) []string {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return slices.Clone(options)
	}
	filtered := make([]string, 0, len(options))
	for _, option := range options {
		if strings.Contains(strings.ToLower(option), filter) {
			filtered = append(filtered, option)
		}
	}
	return filtered
}

func sortedRoomLines(lines []contracts.CASLRoomLineLink) []contracts.CASLRoomLineLink {
	cloned := slices.Clone(lines)
	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].GroupNumber != cloned[j].GroupNumber {
			return cloned[i].GroupNumber < cloned[j].GroupNumber
		}
		return cloned[i].LineNumber < cloned[j].LineNumber
	})
	return cloned
}
