package dialogs

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

const subServerChannelPrimary = 1

type subServerObjectsDialogProvider interface {
	ListSubServers() ([]contracts.AdminSubServer, error)
	ListSubServerObjects(filter string) ([]contracts.AdminSubServerObject, error)
	SetObjectSubServer(objn int64, channel int, bind string) error
	ClearObjectSubServer(objn int64, channel int) error
}

type subServerObjectsDialogState struct {
	win        fyne.Window
	provider   subServerObjectsDialogProvider
	onUpdated  func()
	objects    []contracts.AdminSubServerObject
	subservers []contracts.AdminSubServer

	selectedObject  int
	selectedServer  int
	selectedObjNSet map[int64]struct{}
	lastSelectedObj int64
	objRowSelecting bool
	srvRowSelecting bool

	filterEntry      *widget.Entry
	onlyUnboundCheck *widget.Check
	statusLabel      *widget.Label
	hintLabel        *widget.Label
	objectTable      *widget.Table
	subserverTable   *widget.Table
}

func ShowSubServerObjectsDialog(parent fyne.Window, provider subServerObjectsDialogProvider, onUpdated func()) {
	state := newSubServerObjectsDialogState(provider, onUpdated)
	state.win.SetContent(state.buildContent())
	state.bindActions()
	state.win.Show()
	state.reload(0, -1)
}

func newSubServerObjectsDialogState(
	provider subServerObjectsDialogProvider,
	onUpdated func(),
) *subServerObjectsDialogState {
	state := &subServerObjectsDialogState{
		win:             fyne.CurrentApp().NewWindow("Керування об'єктами підсерверів"),
		provider:        provider,
		onUpdated:       onUpdated,
		objects:         []contracts.AdminSubServerObject{},
		subservers:      []contracts.AdminSubServer{},
		selectedObject:  -1,
		selectedServer:  -1,
		selectedObjNSet: map[int64]struct{}{},
	}
	state.win.Resize(fyne.NewSize(1024, 768))

	state.initControls()
	state.subserverTable = state.buildSubserverTable()
	state.objectTable = state.buildObjectTable()

	return state
}

func (s *subServerObjectsDialogState) initControls() {
	s.filterEntry = widget.NewEntry()
	s.filterEntry.SetPlaceHolder("Пошук: № об'єкта, назва, адреса")
	s.onlyUnboundCheck = widget.NewCheck("Очистка без підсерверів", nil)
	s.statusLabel = widget.NewLabel("Готово")
	s.hintLabel = widget.NewLabel("F3 - пошук по номеру об'єкта | Ctrl+A - вибрати всі об'єкти")
}

func (s *subServerObjectsDialogState) buildContent() fyne.CanvasObject {
	mainSplit := container.NewHSplit(
		container.NewBorder(widget.NewLabel(""), nil, nil, nil, s.objectTable),
		container.NewBorder(widget.NewLabel(""), nil, nil, nil, s.subserverTable),
	)
	mainSplit.Offset = 0.68

	top := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			widget.NewLabel("Пошук:"),
			container.NewHBox(
				widget.NewButton("Пошук", func() {
					s.reload(s.lastSelectedObj, s.selectedServer)
				}),
				widget.NewButton("Оновити", func() {
					s.reload(s.lastSelectedObj, s.selectedServer)
				}),
			),
			s.filterEntry,
		),
		widget.NewSeparator(),
	)

	bottom := container.NewHBox(
		s.onlyUnboundCheck,
		layout.NewSpacer(),
		widget.NewButton("Від'єднати перший", s.clearSelectedObjects),
		widget.NewButton("Під'єднати", s.attachSelectedObjects),
		widget.NewButton("Очистити всі (фільтр)", s.clearAllFilteredObjects),
		widget.NewButton("Під'єднати всі (фільтр)", s.attachAllFilteredObjects),
		layout.NewSpacer(),
		widget.NewButton("Закрити", func() {
			s.win.Close()
		}),
	)

	statusBar := container.NewHBox(
		s.hintLabel,
		layout.NewSpacer(),
		s.statusLabel,
	)

	return container.NewBorder(
		top,
		container.NewVBox(bottom, statusBar),
		nil,
		nil,
		mainSplit,
	)
}

func (s *subServerObjectsDialogState) bindActions() {
	s.filterEntry.OnSubmitted = func(string) {
		s.reload(s.lastSelectedObj, s.selectedServer)
	}
	s.onlyUnboundCheck.OnChanged = func(bool) {
		s.pruneSelection()
		s.objectTable.Refresh()
		s.selectObjectByObjN(s.lastSelectedObj)
	}

	s.win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev != nil && ev.Name == fyne.KeyF3 {
			s.win.Canvas().Focus(s.filterEntry)
		}
	})
	s.win.Canvas().AddShortcut(&fyne.ShortcutSelectAll{}, func(fyne.Shortcut) {
		s.selectAllVisibleObjects()
	})
}

func (s *subServerObjectsDialogState) buildSubserverTable() *widget.Table {
	table := widget.NewTable(
		func() (int, int) { return len(s.subservers) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			s.updateSubserverTableCell(id, obj.(*widget.Label))
		},
	)
	table.SetColumnWidth(0, 180)
	table.SetColumnWidth(1, 110)
	table.SetColumnWidth(2, 60)
	table.OnSelected = s.handleSubserverSelection

	return table
}

func (s *subServerObjectsDialogState) buildObjectTable() *widget.Table {
	table := widget.NewTable(
		func() (int, int) { return len(s.currentObjects()) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			s.updateObjectTableCell(id, obj.(*widget.Label))
		},
	)
	table.SetColumnWidth(0, 110)
	table.SetColumnWidth(1, 460)
	table.SetColumnWidth(2, 180)
	table.OnSelected = s.handleObjectSelection

	return table
}

func (s *subServerObjectsDialogState) updateSubserverTableCell(id widget.TableCellID, label *widget.Label) {
	if id.Row == 0 {
		switch id.Col {
		case 0:
			label.SetText("Інфо")
		case 1:
			label.SetText("Хост")
		default:
			label.SetText("Тип")
		}
		return
	}

	index := id.Row - 1
	if index < 0 || index >= len(s.subservers) {
		label.SetText("")
		return
	}

	subserver := s.subservers[index]
	switch id.Col {
	case 0:
		info := strings.TrimSpace(subserver.Info)
		if info == "" {
			info = strings.TrimSpace(subserver.Host)
		}
		label.SetText(info)
	case 1:
		label.SetText(strings.TrimSpace(subserver.Host))
	default:
		label.SetText(serverTypeLabel(subserver))
	}
}

func (s *subServerObjectsDialogState) updateObjectTableCell(id widget.TableCellID, label *widget.Label) {
	if id.Row == 0 {
		switch id.Col {
		case 0:
			label.SetText("№пр.")
		case 1:
			label.SetText("Об'єкт")
		default:
			label.SetText("Підсервер 1")
		}
		return
	}

	rows := s.currentObjects()
	index := id.Row - 1
	if index < 0 || index >= len(rows) {
		label.SetText("")
		return
	}

	row := rows[index]
	switch id.Col {
	case 0:
		prefix := "  "
		if _, ok := s.selectedObjNSet[row.ObjN]; ok {
			prefix = "✓ "
		}
		label.SetText(fmt.Sprintf("%s%d", prefix, row.ObjN))
	case 1:
		label.SetText(strings.TrimSpace(row.Name))
	default:
		label.SetText(s.bindLabel(row.SubServerA))
	}
}

func (s *subServerObjectsDialogState) handleSubserverSelection(id widget.TableCellID) {
	if id.Row <= 0 {
		s.selectedServer = -1
		return
	}
	if id.Col != 0 && !s.srvRowSelecting {
		s.srvRowSelecting = true
		s.subserverTable.Select(widget.TableCellID{Row: id.Row, Col: 0})
		s.srvRowSelecting = false
		return
	}

	index := id.Row - 1
	if index < 0 || index >= len(s.subservers) {
		s.selectedServer = -1
		return
	}

	s.selectedServer = index
	s.statusLabel.SetText(fmt.Sprintf("Вибрано підсервер: %s", s.bindLabel(s.subservers[index].Bind)))
}

func (s *subServerObjectsDialogState) handleObjectSelection(id widget.TableCellID) {
	if id.Row <= 0 {
		s.selectedObject = -1
		return
	}
	if id.Col != 0 && !s.objRowSelecting {
		s.objRowSelecting = true
		s.objectTable.Select(widget.TableCellID{Row: id.Row, Col: 0})
		s.objRowSelecting = false
		return
	}

	rows := s.currentObjects()
	index := id.Row - 1
	if index < 0 || index >= len(rows) {
		s.selectedObject = -1
		return
	}

	s.selectedObject = index
	s.lastSelectedObj = rows[index].ObjN
	s.selectSingleObject(rows[index].ObjN)
	s.objectTable.Refresh()
	s.statusLabel.SetText(fmt.Sprintf("Вибрано об'єкт #%d", rows[index].ObjN))
}

func (s *subServerObjectsDialogState) currentObjects() []contracts.AdminSubServerObject {
	if !s.onlyUnboundCheck.Checked {
		return s.objects
	}

	filtered := make([]contracts.AdminSubServerObject, 0, len(s.objects))
	for _, obj := range s.objects {
		if strings.TrimSpace(obj.SubServerA) == "" && strings.TrimSpace(obj.SubServerB) == "" {
			filtered = append(filtered, obj)
		}
	}

	return filtered
}

func (s *subServerObjectsDialogState) bindLabel(bind string) string {
	bind = strings.TrimSpace(bind)
	if bind == "" {
		return "—"
	}

	for _, subserver := range s.subservers {
		if !strings.EqualFold(strings.TrimSpace(subserver.Bind), bind) {
			continue
		}

		host := strings.TrimSpace(subserver.Host)
		if host == "" {
			host = strings.TrimSpace(subserver.Info)
		}
		if host == "" {
			host = fmt.Sprintf("SBS %d", subserver.ID)
		}

		return fmt.Sprintf("%s\\%s", host, serverTypeLabel(subserver))
	}

	return "—"
}

func serverTypeLabel(subserver contracts.AdminSubServer) string {
	switch subserver.Type {
	case 2:
		return "GPRS"
	case 4:
		return "AVD"
	default:
		if subserver.Type > 0 {
			return fmt.Sprintf("%d", subserver.Type)
		}
		return "—"
	}
}

func (s *subServerObjectsDialogState) getSelectedObjects() []contracts.AdminSubServerObject {
	rows := s.currentObjects()
	selected := make([]contracts.AdminSubServerObject, 0, len(s.selectedObjNSet))
	for _, obj := range rows {
		if _, ok := s.selectedObjNSet[obj.ObjN]; ok {
			selected = append(selected, obj)
		}
	}
	return selected
}

func (s *subServerObjectsDialogState) selectedTargets() []contracts.AdminSubServerObject {
	targets := s.getSelectedObjects()
	if len(targets) > 0 {
		return targets
	}

	rows := s.currentObjects()
	if s.selectedObject >= 0 && s.selectedObject < len(rows) {
		return []contracts.AdminSubServerObject{rows[s.selectedObject]}
	}

	return []contracts.AdminSubServerObject{}
}

func (s *subServerObjectsDialogState) selectSingleObject(objn int64) {
	s.selectedObjNSet = map[int64]struct{}{}
	if objn > 0 {
		s.selectedObjNSet[objn] = struct{}{}
	}
}

func (s *subServerObjectsDialogState) selectAllVisibleObjects() {
	rows := s.currentObjects()
	s.selectedObjNSet = make(map[int64]struct{}, len(rows))
	for _, obj := range rows {
		s.selectedObjNSet[obj.ObjN] = struct{}{}
	}
	s.objectTable.Refresh()
	s.statusLabel.SetText(fmt.Sprintf("Вибрано об'єктів: %d", len(s.selectedObjNSet)))
}

func (s *subServerObjectsDialogState) pruneSelection() {
	rows := s.currentObjects()
	valid := make(map[int64]struct{}, len(rows))
	for _, obj := range rows {
		valid[obj.ObjN] = struct{}{}
	}
	for objn := range s.selectedObjNSet {
		if _, ok := valid[objn]; !ok {
			delete(s.selectedObjNSet, objn)
		}
	}
}

func (s *subServerObjectsDialogState) selectObjectByObjN(objn int64) {
	if objn <= 0 {
		s.selectedObject = -1
		s.objectTable.UnselectAll()
		return
	}

	rows := s.currentObjects()
	for index := range rows {
		if rows[index].ObjN != objn {
			continue
		}
		s.selectedObject = index
		s.selectSingleObject(objn)
		s.objectTable.Select(widget.TableCellID{Row: index + 1, Col: 0})
		return
	}

	s.selectedObject = -1
	s.selectSingleObject(0)
	s.objectTable.UnselectAll()
}

func (s *subServerObjectsDialogState) reload(selectObjN int64, selectServerIndex int) {
	loadedSubservers, err := s.provider.ListSubServers()
	if err != nil {
		dialog.ShowError(err, s.win)
		s.statusLabel.SetText("Не вдалося завантажити підсервери")
		return
	}
	s.subservers = loadedSubservers
	s.subserverTable.Refresh()
	if selectServerIndex >= 0 && selectServerIndex < len(s.subservers) {
		s.selectedServer = selectServerIndex
	}

	loadedObjects, err := s.provider.ListSubServerObjects(strings.TrimSpace(s.filterEntry.Text))
	if err != nil {
		dialog.ShowError(err, s.win)
		s.statusLabel.SetText("Не вдалося завантажити об'єкти")
		return
	}
	s.objects = loadedObjects
	s.pruneSelection()
	s.objectTable.Refresh()
	s.selectObjectByObjN(selectObjN)
	if len(s.currentObjects()) == 0 {
		s.statusLabel.SetText("Об'єкти не знайдено")
	}
}

func (s *subServerObjectsDialogState) selectedSubserver() (contracts.AdminSubServer, bool) {
	if s.selectedServer < 0 || s.selectedServer >= len(s.subservers) {
		s.statusLabel.SetText("Виберіть підсервер")
		return contracts.AdminSubServer{}, false
	}

	subserver := s.subservers[s.selectedServer]
	if strings.TrimSpace(subserver.Bind) == "" {
		s.statusLabel.SetText("У підсервера порожній BIND")
		return contracts.AdminSubServer{}, false
	}

	return subserver, true
}

func (s *subServerObjectsDialogState) runForTargets(
	targets []contracts.AdminSubServerObject,
	apply func(contracts.AdminSubServerObject) error,
) (applied int, failed int, lastObjN int64) {
	for _, obj := range targets {
		if err := apply(obj); err != nil {
			failed++
			continue
		}
		lastObjN = obj.ObjN
		applied++
	}
	return applied, failed, lastObjN
}

func (s *subServerObjectsDialogState) afterBulkUpdate(lastObjN int64) {
	if lastObjN > 0 {
		s.lastSelectedObj = lastObjN
	}
	s.reload(s.lastSelectedObj, s.selectedServer)
	if s.onUpdated != nil {
		s.onUpdated()
	}
}

func (s *subServerObjectsDialogState) attachSelectedObjects() {
	subserver, ok := s.selectedSubserver()
	if !ok {
		return
	}

	targets := s.selectedTargets()
	if len(targets) == 0 {
		s.statusLabel.SetText("Виберіть один або кілька об'єктів")
		return
	}

	apply := func() {
		applied, failed, lastObjN := s.runForTargets(
			targets,
			func(obj contracts.AdminSubServerObject) error {
				return s.provider.SetObjectSubServer(obj.ObjN, subServerChannelPrimary, subserver.Bind)
			},
		)
		s.afterBulkUpdate(lastObjN)
		s.statusLabel.SetText(fmt.Sprintf("Прив'язку виконано: успішно %d, помилки %d", applied, failed))
	}

	if len(targets) > 1 {
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf(
				"Прив'язати %d вибраних об'єкт(ів) до підсервера (%s)?",
				len(targets),
				s.bindLabel(subserver.Bind),
			),
			func(confirm bool) {
				if confirm {
					apply()
				}
			},
			s.win,
		)
		return
	}

	apply()
}

func (s *subServerObjectsDialogState) clearSelectedObjects() {
	targets := s.selectedTargets()
	if len(targets) == 0 {
		s.statusLabel.SetText("Виберіть один або кілька об'єктів")
		return
	}

	applied, failed, lastObjN := s.runForTargets(
		targets,
		func(obj contracts.AdminSubServerObject) error {
			return s.provider.ClearObjectSubServer(obj.ObjN, subServerChannelPrimary)
		},
	)
	s.afterBulkUpdate(lastObjN)
	s.statusLabel.SetText(fmt.Sprintf("Очищення підсервера: успішно %d, помилки %d", applied, failed))
}

func (s *subServerObjectsDialogState) attachAllFilteredObjects() {
	rows := s.currentObjects()
	if len(rows) == 0 {
		s.statusLabel.SetText("Немає об'єктів для масової прив'язки")
		return
	}

	subserver, ok := s.selectedSubserver()
	if !ok {
		return
	}

	dialog.ShowConfirm(
		"Масова прив'язка",
		fmt.Sprintf("Прив'язати %d об'єкт(ів) до підсервера (%s)?", len(rows), s.bindLabel(subserver.Bind)),
		func(confirm bool) {
			if !confirm {
				return
			}

			applied, failed, _ := s.runForTargets(
				rows,
				func(obj contracts.AdminSubServerObject) error {
					return s.provider.SetObjectSubServer(obj.ObjN, subServerChannelPrimary, subserver.Bind)
				},
			)
			s.afterBulkUpdate(0)
			s.statusLabel.SetText(fmt.Sprintf("Масова прив'язка завершена: успішно %d, помилки %d", applied, failed))
		},
		s.win,
	)
}

func (s *subServerObjectsDialogState) clearAllFilteredObjects() {
	rows := s.currentObjects()
	if len(rows) == 0 {
		s.statusLabel.SetText("Немає об'єктів для масового очищення")
		return
	}

	dialog.ShowConfirm(
		"Масове очищення",
		fmt.Sprintf("Очистити підсервер 1 для %d об'єкт(ів)?", len(rows)),
		func(confirm bool) {
			if !confirm {
				return
			}

			applied, failed, _ := s.runForTargets(
				rows,
				func(obj contracts.AdminSubServerObject) error {
					return s.provider.ClearObjectSubServer(obj.ObjN, subServerChannelPrimary)
				},
			)
			s.afterBulkUpdate(0)
			s.statusLabel.SetText(fmt.Sprintf("Масове очищення завершено: успішно %d, помилки %d", applied, failed))
		},
		s.win,
	)
}
