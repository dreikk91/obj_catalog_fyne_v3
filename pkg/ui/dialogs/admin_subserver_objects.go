package dialogs

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	data "obj_catalog_fyne_v3/pkg/contracts"
	uiwidgets "obj_catalog_fyne_v3/pkg/ui/widgets"
)

func ShowSubServerObjectsDialog(parent fyne.Window, provider data.AdminProvider, onUpdated func()) {
	win := fyne.CurrentApp().NewWindow("Керування об'єктами підсерверів")
	win.Resize(fyne.NewSize(1040, 640))

	var (
		objects         []data.AdminSubServerObject
		subservers      []data.AdminSubServer
		selectedObject  = -1
		selectedServer  = -1
		selectedObjNSet = map[int64]struct{}{}
		lastSelectedObj int64
		ctrlMultiNext   bool
	)

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Пошук: № об'єкта, назва, адреса")
	onlyUnboundCheck := widget.NewCheck("Очистка без підсерверів", nil)
	statusLabel := widget.NewLabel("Готово")
	hintLabel := widget.NewLabel("F3 - пошук | Ctrl+клік - мультивибір | Ctrl+A - вибрати всі")

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

	serverTypeLabel := func(s data.AdminSubServer) string {
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

	currentObjects := func() []data.AdminSubServerObject {
		if !onlyUnboundCheck.Checked {
			return objects
		}
		filtered := make([]data.AdminSubServerObject, 0, len(objects))
		for _, obj := range objects {
			if strings.TrimSpace(obj.SubServerA) == "" && strings.TrimSpace(obj.SubServerB) == "" {
				filtered = append(filtered, obj)
			}
		}
		return filtered
	}
	var objectTable *widget.Table

	getSelectedObjects := func() []data.AdminSubServerObject {
		rows := currentObjects()
		selected := make([]data.AdminSubServerObject, 0, len(selectedObjNSet))
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

	subserverTableView := uiwidgets.NewQTableViewWithHeaders(
		[]string{"Інфо", "Хост", "Тип"},
		func() int { return len(subservers) },
		func(row, col int) string {
			if row < 0 || row >= len(subservers) {
				return ""
			}
			sb := subservers[row]
			switch col {
			case 0:
				info := strings.TrimSpace(sb.Info)
				if info == "" {
					info = strings.TrimSpace(sb.Host)
				}
				return info
			case 1:
				return strings.TrimSpace(sb.Host)
			default:
				return serverTypeLabel(sb)
			}
		},
	)
	subserverTableView.SetSelectionBehavior(uiwidgets.SelectRows)
	subserverTable := subserverTableView.Widget()
	subserverTableView.SetColumnWidth(0, 180)
	subserverTableView.SetColumnWidth(1, 110)
	subserverTableView.SetColumnWidth(2, 60)
	subserverTableView.OnSelected = func(index uiwidgets.ModelIndex) {
		if index.Row < 0 || index.Row >= len(subservers) {
			selectedServer = -1
			return
		}
		selectedServer = index.Row
		statusLabel.SetText(fmt.Sprintf("Вибрано підсервер: %s", bindLabel(subservers[index.Row].Bind)))
	}

	objectTableView := uiwidgets.NewQTableViewWithHeaders(
		[]string{"№пр.", "Об'єкт", "Підсервер 1"},
		func() int { return len(currentObjects()) },
		func(row, col int) string {
			dataRows := currentObjects()
			if row < 0 || row >= len(dataRows) {
				return ""
			}
			item := dataRows[row]
			switch col {
			case 0:
				prefix := "  "
				if _, ok := selectedObjNSet[item.ObjN]; ok {
					prefix = "✓ "
				}
				return fmt.Sprintf("%s%d", prefix, item.ObjN)
			case 1:
				return strings.TrimSpace(item.Name)
			default:
				return bindLabel(item.SubServerA)
			}
		},
	)
	objectTableView.SetSelectionBehavior(uiwidgets.SelectRows)
	objectTable = objectTableView.Widget()
	objectTableView.SetColumnWidth(0, 110)
	objectTableView.SetColumnWidth(1, 460)
	objectTableView.SetColumnWidth(2, 180)
	objectTableView.OnSelected = func(index uiwidgets.ModelIndex) {
		rows := currentObjects()
		idx := index.Row
		if idx < 0 || idx >= len(rows) {
			selectedObject = -1
			return
		}

		objn := rows[idx].ObjN
		if ctrlMultiNext {
			ctrlMultiNext = false
			if _, ok := selectedObjNSet[objn]; ok {
				delete(selectedObjNSet, objn)
			} else {
				selectedObjNSet[objn] = struct{}{}
			}
			lastSelectedObj = objn
			if len(selectedObjNSet) == 0 {
				selectedObject = -1
				statusLabel.SetText("Виділення очищено")
			} else {
				selectedObject = idx
				statusLabel.SetText(fmt.Sprintf("Вибрано об'єктів: %d", len(selectedObjNSet)))
			}
			objectTable.Refresh()
			return
		}

		selectedObject = idx
		lastSelectedObj = objn
		selectSingleObject(objn)
		objectTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Вибрано об'єкт #%d", rows[idx].ObjN))
	}

	selectObjectByObjN := func(objn int64) {
		ctrlMultiNext = false
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
				objectTable.Select(widget.TableCellID{Row: i, Col: 0})
				return
			}
		}
		selectedObject = -1
		selectSingleObject(0)
		objectTable.UnselectAll()
	}

	reload := func(selectObjN int64, selectServerIndex int) {
		ctrlMultiNext = false
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
		if ev.Name == desktop.KeyControlLeft || ev.Name == desktop.KeyControlRight {
			ctrlMultiNext = true
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
		container.NewHBox(
			widget.NewLabel("Пошук:"),
			filterEntry,
			layout.NewSpacer(),
			refreshBtn,
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
