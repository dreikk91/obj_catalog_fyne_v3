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

func ShowSubServerObjectsDialog(parent fyne.Window, provider contracts.AdminProvider, onUpdated func()) {
	win := fyne.CurrentApp().NewWindow("Керування об'єктами підсерверів")
	win.Resize(fyne.NewSize(1024, 768))

	var (
		objects         []contracts.AdminSubServerObject
		subservers      []contracts.AdminSubServer
		selectedObject  = -1
		selectedServer  = -1
		selectedObjNSet = map[int64]struct{}{}
		lastSelectedObj int64
		objRowSelecting bool
		srvRowSelecting bool
	)

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Пошук: № об'єкта, назва, адреса")
	onlyUnboundCheck := widget.NewCheck("Очистка без підсерверів", nil)
	statusLabel := widget.NewLabel("Готово")
	hintLabel := widget.NewLabel("F3 - пошук по номеру об'єкта | Ctrl+A - вибрати всі об'єкти")

	bindLabel := func(bind string) string {
		bind = strings.TrimSpace(bind)
		if bind == "" {
			return "—"
		}
		for _, s := range subservers {
			if strings.EqualFold(strings.TrimSpace(s.Bind), bind) {
				host := strings.TrimSpace(s.Host)
				if host == "" {
					host = strings.TrimSpace(s.Info)
				}
				if host == "" {
					host = fmt.Sprintf("SBS %d", s.ID)
				}
				typeLabel := "—"
				switch s.Type {
				case 2:
					typeLabel = "GPRS"
				case 4:
					typeLabel = "AVD"
				default:
					if s.Type > 0 {
						typeLabel = fmt.Sprintf("%d", s.Type)
					}
				}
				return fmt.Sprintf("%s\\%s", host, typeLabel)
			}
		}
		return "—"
	}

	serverTypeLabel := func(s contracts.AdminSubServer) string {
		switch s.Type {
		case 2:
			return "GPRS"
		case 4:
			return "AVD"
		default:
			if s.Type > 0 {
				return fmt.Sprintf("%d", s.Type)
			}
			return "—"
		}
	}

	currentObjects := func() []contracts.AdminSubServerObject {
		if !onlyUnboundCheck.Checked {
			return objects
		}
		filtered := make([]contracts.AdminSubServerObject, 0, len(objects))
		for _, obj := range objects {
			if strings.TrimSpace(obj.SubServerA) == "" && strings.TrimSpace(obj.SubServerB) == "" {
				filtered = append(filtered, obj)
			}
		}
		return filtered
	}
	var objectTable *widget.Table

	getSelectedObjects := func() []contracts.AdminSubServerObject {
		rows := currentObjects()
		selected := make([]contracts.AdminSubServerObject, 0, len(selectedObjNSet))
		for _, obj := range rows {
			if _, ok := selectedObjNSet[obj.ObjN]; ok {
				selected = append(selected, obj)
			}
		}
		return selected
	}

	selectSingleObject := func(objn int64) {
		selectedObjNSet = map[int64]struct{}{}
		if objn > 0 {
			selectedObjNSet[objn] = struct{}{}
		}
	}

	selectAllVisibleObjects := func() {
		rows := currentObjects()
		selectedObjNSet = map[int64]struct{}{}
		for _, obj := range rows {
			selectedObjNSet[obj.ObjN] = struct{}{}
		}
		objectTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Вибрано об'єктів: %d", len(selectedObjNSet)))
	}

	pruneSelection := func() {
		rows := currentObjects()
		valid := make(map[int64]struct{}, len(rows))
		for _, obj := range rows {
			valid[obj.ObjN] = struct{}{}
		}
		for objn := range selectedObjNSet {
			if _, ok := valid[objn]; !ok {
				delete(selectedObjNSet, objn)
			}
		}
	}

	subserverTable := widget.NewTable(
		func() (int, int) { return len(subservers) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("Інфо")
				case 1:
					lbl.SetText("Хост")
				default:
					lbl.SetText("Тип")
				}
				return
			}
			idx := id.Row - 1
			if idx < 0 || idx >= len(subservers) {
				lbl.SetText("")
				return
			}
			sb := subservers[idx]
			switch id.Col {
			case 0:
				info := strings.TrimSpace(sb.Info)
				if info == "" {
					info = strings.TrimSpace(sb.Host)
				}
				lbl.SetText(info)
			case 1:
				lbl.SetText(strings.TrimSpace(sb.Host))
			default:
				lbl.SetText(serverTypeLabel(sb))
			}
		},
	)
	subserverTable.SetColumnWidth(0, 180)
	subserverTable.SetColumnWidth(1, 110)
	subserverTable.SetColumnWidth(2, 60)
	subserverTable.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			selectedServer = -1
			return
		}
		if id.Col != 0 && !srvRowSelecting {
			srvRowSelecting = true
			subserverTable.Select(widget.TableCellID{Row: id.Row, Col: 0})
			srvRowSelecting = false
			return
		}
		idx := id.Row - 1
		if idx < 0 || idx >= len(subservers) {
			selectedServer = -1
			return
		}
		selectedServer = idx
		statusLabel.SetText(fmt.Sprintf("Вибрано підсервер: %s", bindLabel(subservers[idx].Bind)))
	}

	objectTable = widget.NewTable(
		func() (int, int) { return len(currentObjects()) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("№пр.")
				case 1:
					lbl.SetText("Об'єкт")
				default:
					lbl.SetText("Підсервер 1")
				}
				return
			}

			dataRows := currentObjects()
			idx := id.Row - 1
			if idx < 0 || idx >= len(dataRows) {
				lbl.SetText("")
				return
			}
			row := dataRows[idx]
			switch id.Col {
			case 0:
				prefix := "  "
				if _, ok := selectedObjNSet[row.ObjN]; ok {
					prefix = "✓ "
				}
				lbl.SetText(fmt.Sprintf("%s%d", prefix, row.ObjN))
			case 1:
				lbl.SetText(strings.TrimSpace(row.Name))
			default:
				lbl.SetText(bindLabel(row.SubServerA))
			}
		},
	)
	objectTable.SetColumnWidth(0, 110)
	objectTable.SetColumnWidth(1, 460)
	objectTable.SetColumnWidth(2, 180)
	objectTable.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			selectedObject = -1
			return
		}
		if id.Col != 0 && !objRowSelecting {
			objRowSelecting = true
			objectTable.Select(widget.TableCellID{Row: id.Row, Col: 0})
			objRowSelecting = false
			return
		}
		rows := currentObjects()
		idx := id.Row - 1
		if idx < 0 || idx >= len(rows) {
			selectedObject = -1
			return
		}
		selectedObject = idx
		lastSelectedObj = rows[idx].ObjN
		selectSingleObject(rows[idx].ObjN)
		objectTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Вибрано об'єкт #%d", rows[idx].ObjN))
	}

	selectObjectByObjN := func(objn int64) {
		if objn <= 0 {
			selectedObject = -1
			objectTable.UnselectAll()
			return
		}
		rows := currentObjects()
		for i := range rows {
			if rows[i].ObjN == objn {
				selectedObject = i
				selectSingleObject(objn)
				objectTable.Select(widget.TableCellID{Row: i + 1, Col: 0})
				return
			}
		}
		selectedObject = -1
		selectSingleObject(0)
		objectTable.UnselectAll()
	}

	reload := func(selectObjN int64, selectServerIndex int) {
		loadedSub, err := provider.ListSubServers()
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося завантажити підсервери")
			return
		}
		subservers = loadedSub
		subserverTable.Refresh()
		if selectServerIndex >= 0 && selectServerIndex < len(subservers) {
			selectedServer = selectServerIndex
		}

		loadedObj, err := provider.ListSubServerObjects(strings.TrimSpace(filterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося завантажити об'єкти")
			return
		}
		objects = loadedObj
		pruneSelection()
		objectTable.Refresh()
		selectObjectByObjN(selectObjN)
		if len(currentObjects()) == 0 {
			statusLabel.SetText("Об'єкти не знайдено")
		}
	}

	attach := func() {
		if selectedServer < 0 || selectedServer >= len(subservers) {
			statusLabel.SetText("Виберіть підсервер")
			return
		}
		sb := subservers[selectedServer]
		if strings.TrimSpace(sb.Bind) == "" {
			statusLabel.SetText("У підсервера порожній BIND")
			return
		}

		targets := getSelectedObjects()
		if len(targets) == 0 {
			rows := currentObjects()
			if selectedObject >= 0 && selectedObject < len(rows) {
				targets = append(targets, rows[selectedObject])
			}
		}
		if len(targets) == 0 {
			statusLabel.SetText("Виберіть один або кілька об'єктів")
			return
		}

		apply := func() {
			applied := 0
			failed := 0
			lastObjN := int64(0)
			for _, obj := range targets {
				if err := provider.SetObjectSubServer(obj.ObjN, 1, sb.Bind); err != nil {
					failed++
					continue
				}
				lastObjN = obj.ObjN
				applied++
			}
			if lastObjN > 0 {
				lastSelectedObj = lastObjN
			}
			reload(lastSelectedObj, selectedServer)
			statusLabel.SetText(fmt.Sprintf("Прив'язку виконано: успішно %d, помилки %d", applied, failed))
			if onUpdated != nil {
				onUpdated()
			}
		}

		if len(targets) > 1 {
			dialog.ShowConfirm(
				"Підтвердження",
				fmt.Sprintf("Прив'язати %d вибраних об'єкт(ів) до підсервера (%s)?", len(targets), bindLabel(sb.Bind)),
				func(ok bool) {
					if !ok {
						return
					}
					apply()
				},
				win,
			)
			return
		}

		apply()
	}

	clearFirst := func() {
		targets := getSelectedObjects()
		if len(targets) == 0 {
			rows := currentObjects()
			if selectedObject >= 0 && selectedObject < len(rows) {
				targets = append(targets, rows[selectedObject])
			}
		}
		if len(targets) == 0 {
			statusLabel.SetText("Виберіть один або кілька об'єктів")
			return
		}

		applied := 0
		failed := 0
		lastObjN := int64(0)
		for _, obj := range targets {
			if err := provider.ClearObjectSubServer(obj.ObjN, 1); err != nil {
				failed++
				continue
			}
			lastObjN = obj.ObjN
			applied++
		}
		if lastObjN > 0 {
			lastSelectedObj = lastObjN
		}
		reload(lastSelectedObj, selectedServer)
		statusLabel.SetText(fmt.Sprintf("Очищення підсервера: успішно %d, помилки %d", applied, failed))
		if onUpdated != nil {
			onUpdated()
		}
	}

	attachAllFiltered := func() {
		rows := currentObjects()
		if len(rows) == 0 {
			statusLabel.SetText("Немає об'єктів для масової прив'язки")
			return
		}
		if selectedServer < 0 || selectedServer >= len(subservers) {
			statusLabel.SetText("Виберіть підсервер")
			return
		}
		sb := subservers[selectedServer]
		if strings.TrimSpace(sb.Bind) == "" {
			statusLabel.SetText("У підсервера порожній BIND")
			return
		}

		dialog.ShowConfirm(
			"Масова прив'язка",
			fmt.Sprintf("Прив'язати %d об'єкт(ів) до підсервера (%s)?", len(rows), bindLabel(sb.Bind)),
			func(ok bool) {
				if !ok {
					return
				}
				applied := 0
				failed := 0
				for _, obj := range rows {
					if err := provider.SetObjectSubServer(obj.ObjN, 1, sb.Bind); err != nil {
						failed++
						continue
					}
					applied++
				}
				reload(lastSelectedObj, selectedServer)
				statusLabel.SetText(fmt.Sprintf("Масова прив'язка завершена: успішно %d, помилки %d", applied, failed))
				if onUpdated != nil {
					onUpdated()
				}
			},
			win,
		)
	}

	clearAllFiltered := func() {
		rows := currentObjects()
		if len(rows) == 0 {
			statusLabel.SetText("Немає об'єктів для масового очищення")
			return
		}
		dialog.ShowConfirm(
			"Масове очищення",
			fmt.Sprintf("Очистити підсервер 1 для %d об'єкт(ів)?", len(rows)),
			func(ok bool) {
				if !ok {
					return
				}
				applied := 0
				failed := 0
				for _, obj := range rows {
					if err := provider.ClearObjectSubServer(obj.ObjN, 1); err != nil {
						failed++
						continue
					}
					applied++
				}
				reload(lastSelectedObj, selectedServer)
				statusLabel.SetText(fmt.Sprintf("Масове очищення завершено: успішно %d, помилки %d", applied, failed))
				if onUpdated != nil {
					onUpdated()
				}
			},
			win,
		)
	}

	detachBtn := widget.NewButton("Від'єднати перший", clearFirst)
	attachBtn := widget.NewButton("Під'єднати", attach)
	attachAllBtn := widget.NewButton("Під'єднати всі (фільтр)", attachAllFiltered)
	detachAllBtn := widget.NewButton("Очистити всі (фільтр)", clearAllFiltered)
	searchBtn := widget.NewButton("Пошук", func() { reload(lastSelectedObj, selectedServer) })
	refreshBtn := widget.NewButton("Оновити", func() { reload(lastSelectedObj, selectedServer) })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	filterEntry.OnSubmitted = func(string) { reload(lastSelectedObj, selectedServer) }
	onlyUnboundCheck.OnChanged = func(bool) {
		pruneSelection()
		objectTable.Refresh()
		selectObjectByObjN(lastSelectedObj)
	}

	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev == nil {
			return
		}
		if ev.Name == fyne.KeyF3 {
			win.Canvas().Focus(filterEntry)
		}
	})
	win.Canvas().AddShortcut(&fyne.ShortcutSelectAll{}, func(fyne.Shortcut) {
		selectAllVisibleObjects()
	})

	mainSplit := container.NewHSplit(
		container.NewBorder(widget.NewLabel(""), nil, nil, nil, objectTable),
		container.NewBorder(widget.NewLabel(""), nil, nil, nil, subserverTable),
	)
	mainSplit.Offset = 0.68

	top := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			widget.NewLabel("Пошук:"),
			container.NewHBox(searchBtn, refreshBtn),
			filterEntry,
		),
		widget.NewSeparator(),
	)

	bottom := container.NewHBox(
		onlyUnboundCheck,
		layout.NewSpacer(),
		detachBtn,
		attachBtn,
		detachAllBtn,
		attachAllBtn,
		layout.NewSpacer(),
		closeBtn,
	)

	statusBar := container.NewHBox(
		hintLabel,
		layout.NewSpacer(),
		statusLabel,
	)

	win.SetContent(container.NewBorder(top, container.NewVBox(bottom, statusBar), nil, nil, mainSplit))
	win.Show()
	reload(0, -1)
}
