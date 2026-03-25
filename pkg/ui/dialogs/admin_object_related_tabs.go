package dialogs

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/data"
	uiwidgets "obj_catalog_fyne_v3/pkg/ui/widgets"
)

const (
	mapCenterModePrefKey    = "admin.map.center.mode"
	mapCenterCustomLatKey   = "admin.map.center.custom.lat"
	mapCenterCustomLonKey   = "admin.map.center.custom.lon"
	mapCenterLastLatPrefKey = "admin.map.center.last.lat"
	mapCenterLastLonPrefKey = "admin.map.center.last.lon"

	mapCenterModeLviv   = "lviv"
	mapCenterModeKyiv   = "kyiv"
	mapCenterModeCustom = "custom"
	mapCenterModeLast   = "last"
)

func buildObjectPersonalTab(parent fyne.Window, provider data.AdminProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	var (
		items       []data.AdminObjectPersonal
		selectedRow = -1
	)

	fullName := func(item data.AdminObjectPersonal) string {
		parts := []string{
			strings.TrimSpace(item.Surname),
			strings.TrimSpace(item.Name),
			strings.TrimSpace(item.SecName),
		}
		filtered := make([]string, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return "(без ПІБ)"
		}
		return strings.Join(filtered, " ")
	}

	tableView := uiwidgets.NewQTableViewWithCallbacks(
		func() (int, int) { return len(items) + 1, 6 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("№")
				case 1:
					lbl.SetText("ПІБ")
				case 2:
					lbl.SetText("Телефон")
				case 3:
					lbl.SetText("Посада")
				case 4:
					lbl.SetText("Доступ")
				default:
					lbl.SetText("Примітка")
				}
				return
			}
			itemIdx := id.Row - 1
			if itemIdx < 0 || itemIdx >= len(items) {
				lbl.SetText("")
				return
			}
			it := items[itemIdx]
			switch id.Col {
			case 0:
				lbl.SetText(strconv.FormatInt(it.Number, 10))
			case 1:
				lbl.SetText(fullName(it))
			case 2:
				lbl.SetText(strings.TrimSpace(it.Phones))
			case 3:
				lbl.SetText(strings.TrimSpace(it.Position))
			case 4:
				if it.Access1 > 0 {
					lbl.SetText("Адмін")
				} else {
					lbl.SetText("Оператор")
				}
			case 5:
				lbl.SetText(strings.TrimSpace(it.Notes))
			}
		},
	)
	table := tableView.Widget()
	const (
		personalColWNum   = float32(60)
		personalColWName  = float32(280)
		personalColWPhone = float32(200)
		personalColWPos   = float32(180)
		personalColWRole  = float32(110)
		personalColWNote  = float32(220)
	)
	table.SetColumnWidth(0, personalColWNum)
	table.SetColumnWidth(1, personalColWName)
	table.SetColumnWidth(2, personalColWPhone)
	table.SetColumnWidth(3, personalColWPos)
	table.SetColumnWidth(4, personalColWRole)
	table.SetColumnWidth(5, personalColWNote)
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			selectedRow = -1
			return
		}
		itemIdx := id.Row - 1
		if itemIdx < 0 || itemIdx >= len(items) {
			selectedRow = -1
			return
		}
		selectedRow = itemIdx
	}

	reload := func() {
		loaded, err := provider.ListObjectPersonals(objn)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося завантажити В/О")
			return
		}
		items = loaded
		selectedRow = -1
		table.UnselectAll()
		table.Refresh()
		statusLabel.SetText(fmt.Sprintf("В/О: %d запис(ів)", len(items)))
	}

	addBtn := widget.NewButton("Додати", func() {
		showObjectPersonalEditor(parent, provider, "Додати В/О", data.AdminObjectPersonal{}, func(item data.AdminObjectPersonal) error {
			return provider.AddObjectPersonal(objn, item)
		}, statusLabel, func() {
			reload()
			statusLabel.SetText("В/О додано")
		})
	})

	editBtn := widget.NewButton("Змінити", func() {
		if selectedRow < 0 || selectedRow >= len(items) {
			statusLabel.SetText("Виберіть В/О у таблиці")
			return
		}
		initial := items[selectedRow]
		showObjectPersonalEditor(parent, provider, "Редагування В/О", initial, func(item data.AdminObjectPersonal) error {
			item.ID = initial.ID
			if strings.TrimSpace(item.CreatedAt) == "" {
				item.CreatedAt = initial.CreatedAt
			}
			return provider.UpdateObjectPersonal(objn, item)
		}, statusLabel, func() {
			reload()
			statusLabel.SetText("В/О оновлено")
		})
	})

	deleteBtn := widget.NewButton("Видалити", func() {
		if selectedRow < 0 || selectedRow >= len(items) {
			statusLabel.SetText("Виберіть В/О у таблиці")
			return
		}
		target := items[selectedRow]
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити запис \"%s\"?", fullName(target)),
			func(ok bool) {
				if !ok {
					return
				}
				if err := provider.DeleteObjectPersonal(objn, target.ID); err != nil {
					dialog.ShowError(err, parent)
					statusLabel.SetText("Не вдалося видалити В/О")
					return
				}
				reload()
				statusLabel.SetText("В/О видалено")
			},
			parent,
		)
	})

	refreshBtn := widget.NewButton("Оновити", reload)

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addBtn, editBtn, deleteBtn, layout.NewSpacer(), refreshBtn),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		table,
	)

	reload()
	return content
}

func buildObjectZonesTab(parent fyne.Window, provider data.AdminProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	var (
		items       []data.AdminObjectZone
		selectedRow = -1
	)

	quickNameEntry := widget.NewEntry()
	quickNameEntry.SetPlaceHolder("Назва зони (Enter -> наступна зона)")
	selectedZoneLabel := widget.NewLabel("Зона: —")

	effectiveZoneNumberAt := func(idx int) int64 {
		if idx < 0 || idx >= len(items) {
			return 0
		}
		if items[idx].ZoneNumber > 0 {
			return items[idx].ZoneNumber
		}
		return int64(idx) + 1
	}

	tableView := uiwidgets.NewQTableViewWithCallbacks(
		func() (int, int) { return len(items) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("ZONEN")
				case 1:
					lbl.SetText("Тип")
				default:
					lbl.SetText("Опис")
				}
				return
			}
			itemIdx := id.Row - 1
			if itemIdx < 0 || itemIdx >= len(items) {
				lbl.SetText("")
				return
			}
			it := items[itemIdx]
			switch id.Col {
			case 0:
				zonen := effectiveZoneNumberAt(itemIdx)
				lbl.SetText(strconv.FormatInt(zonen, 10))
			case 1:
				lbl.SetText("пож.")
			default:
				lbl.SetText(strings.TrimSpace(it.Description))
			}
		},
	)
	table := tableView.Widget()
	const (
		zoneColWNum  = float32(120)
		zoneColWType = float32(120)
		zoneColWDesc = float32(520)
	)
	table.StickyRowCount = 1
	table.StickyColumnCount = 1
	applyZoneTableLayout := func() {
		table.SetColumnWidth(0, zoneColWNum)
		table.SetColumnWidth(1, zoneColWType)
		table.SetColumnWidth(2, zoneColWDesc)
	}
	applyZoneTableLayout()

	findRowByZoneNumber := func(zoneNumber int64) int {
		if zoneNumber <= 0 {
			return -1
		}
		for i := range items {
			if effectiveZoneNumberAt(i) == zoneNumber {
				return i
			}
		}
		return -1
	}

	updateSelectedZoneLabel := func() {
		if selectedRow < 0 || selectedRow >= len(items) {
			selectedZoneLabel.SetText("Зона: —")
			return
		}
		selectedZoneLabel.SetText(fmt.Sprintf("Зона: #%d", effectiveZoneNumberAt(selectedRow)))
	}

	ensureZoneExists := func(zoneNumber int64, defaultDescription string) error {
		if zoneNumber <= 0 {
			return fmt.Errorf("invalid zone number")
		}
		if findRowByZoneNumber(zoneNumber) >= 0 {
			return nil
		}
		desc := strings.TrimSpace(defaultDescription)
		if desc == "" {
			desc = fmt.Sprintf("Шлейф %d", zoneNumber)
		}
		return provider.AddObjectZone(objn, data.AdminObjectZone{
			ZoneNumber:    zoneNumber,
			ZoneType:      1,
			Description:   desc,
			EntryDelaySec: 0,
		})
	}

	selectByZoneNumber := func(zoneNumber int64, focusQuickName bool) {
		if len(items) == 0 {
			selectedRow = -1
			table.UnselectAll()
			quickNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}

		targetRow := 0
		if zoneNumber > 0 {
			if row := findRowByZoneNumber(zoneNumber); row >= 0 {
				targetRow = row
			}
		}

		selectedRow = targetRow
		table.Select(widget.TableCellID{Row: targetRow + 1, Col: 0})
		quickNameEntry.SetText(strings.TrimSpace(items[targetRow].Description))
		updateSelectedZoneLabel()
		if focusQuickName {
			focusIfOnCanvas(parent, quickNameEntry)
		}
	}

	reloadAndSelect := func(targetZoneNumber int64, focusQuickName bool) {
		loaded, err := provider.ListObjectZones(objn)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося завантажити зони")
			return
		}
		items = loaded
		table.Refresh()
		applyZoneTableLayout()
		statusLabel.SetText(fmt.Sprintf("Зони: %d запис(ів)", len(items)))
		selectByZoneNumber(targetZoneNumber, focusQuickName)
	}

	reload := func() {
		reloadAndSelect(0, false)
	}

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			selectedRow = -1
			quickNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		itemIdx := id.Row - 1
		if itemIdx < 0 || itemIdx >= len(items) {
			selectedRow = -1
			quickNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		selectedRow = itemIdx
		quickNameEntry.SetText(strings.TrimSpace(items[itemIdx].Description))
		updateSelectedZoneLabel()
		// Даємо змогу одразу вводити назву наступної/поточної зони.
		focusIfOnCanvas(parent, quickNameEntry)
	}

	moveToNextZone := func() {
		if selectedRow < 0 || selectedRow >= len(items) {
			if len(items) == 0 {
				if err := ensureZoneExists(1, strings.TrimSpace(quickNameEntry.Text)); err != nil {
					dialog.ShowError(err, parent)
					statusLabel.SetText("Не вдалося додати першу зону")
					return
				}
				reloadAndSelect(1, true)
				statusLabel.SetText("Додано зону #1")
				return
			}
			selectByZoneNumber(effectiveZoneNumberAt(0), true)
		}
		if selectedRow < 0 || selectedRow >= len(items) {
			statusLabel.SetText("Виберіть зону у таблиці")
			return
		}

		current := items[selectedRow]
		currentZoneNumber := effectiveZoneNumberAt(selectedRow)
		if current.ZoneNumber <= 0 {
			current.ZoneNumber = currentZoneNumber
		}
		current.Description = strings.TrimSpace(quickNameEntry.Text)
		if err := provider.UpdateObjectZone(objn, current); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося зберегти назву зони")
			return
		}

		nextZoneNumber := currentZoneNumber + 1
		if err := ensureZoneExists(nextZoneNumber, ""); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося додати наступну зону")
			return
		}

		reloadAndSelect(nextZoneNumber, true)
		statusLabel.SetText(fmt.Sprintf("Збережено зону #%d, перехід на #%d", currentZoneNumber, nextZoneNumber))
	}
	quickNameEntry.OnSubmitted = func(string) {
		moveToNextZone()
	}

	addBtn := widget.NewButton("Додати", func() {
		nextZoneNumber := int64(1)
		if selectedRow >= 0 && selectedRow < len(items) {
			nextZoneNumber = effectiveZoneNumberAt(selectedRow) + 1
		} else if len(items) > 0 {
			lastZone := effectiveZoneNumberAt(len(items) - 1)
			if lastZone > 0 {
				nextZoneNumber = lastZone + 1
			}
		}
		if err := ensureZoneExists(nextZoneNumber, ""); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося додати зону")
			return
		}
		reloadAndSelect(nextZoneNumber, true)
		statusLabel.SetText(fmt.Sprintf("Готово до введення зони #%d", nextZoneNumber))
	})

	editBtn := widget.NewButton("Змінити", func() {
		if len(items) == 0 {
			if err := ensureZoneExists(1, ""); err != nil {
				dialog.ShowError(err, parent)
				statusLabel.SetText("Не вдалося створити першу зону")
				return
			}
			reloadAndSelect(1, true)
			statusLabel.SetText("Створено зону #1, можна вводити назву")
			return
		}
		if selectedRow < 0 || selectedRow >= len(items) {
			selectByZoneNumber(effectiveZoneNumberAt(0), true)
			statusLabel.SetText("Виберіть зону і вводьте назву")
			return
		}
		updateSelectedZoneLabel()
		focusIfOnCanvas(parent, quickNameEntry)
		statusLabel.SetText(fmt.Sprintf("Редагування зони #%d: введіть назву і натисніть Enter", effectiveZoneNumberAt(selectedRow)))
	})

	deleteBtn := widget.NewButton("Видалити", func() {
		if selectedRow < 0 || selectedRow >= len(items) {
			statusLabel.SetText("Виберіть зону у таблиці")
			return
		}
		target := items[selectedRow]
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити зону #%d?", target.ZoneNumber),
			func(ok bool) {
				if !ok {
					return
				}
				if err := provider.DeleteObjectZone(objn, target.ID); err != nil {
					dialog.ShowError(err, parent)
					statusLabel.SetText("Не вдалося видалити зону")
					return
				}
				reload()
				statusLabel.SetText("Зону видалено")
			},
			parent,
		)
	})

	fillBtn := widget.NewButton("Заповнити", func() {
		defaultCount := suggestZoneFillCount(provider, objn, items)
		showZoneFillDialog(parent, defaultCount, func(count int64) {
			if err := provider.FillObjectZones(objn, count); err != nil {
				dialog.ShowError(err, parent)
				statusLabel.SetText("Не вдалося заповнити зони")
				return
			}
			reload()
			statusLabel.SetText("Зони заповнено")
		}, statusLabel)
	})

	clearBtn := widget.NewButton("Очистити", func() {
		dialog.ShowConfirm(
			"Підтвердження",
			"Видалити всі зони об'єкта?",
			func(ok bool) {
				if !ok {
					return
				}
				if err := provider.ClearObjectZones(objn); err != nil {
					dialog.ShowError(err, parent)
					statusLabel.SetText("Не вдалося очистити зони")
					return
				}
				reload()
				statusLabel.SetText("Зони очищено")
			},
			parent,
		)
	})

	refreshBtn := widget.NewButton("Оновити", reload)
	nextBtn := widget.NewButton("Enter -> Наступна", moveToNextZone)

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addBtn, editBtn, deleteBtn, fillBtn, clearBtn, layout.NewSpacer(), refreshBtn),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(widget.NewLabel("Швидке введення:"), layout.NewSpacer(), selectedZoneLabel),
				nextBtn,
				quickNameEntry,
			),
		),
		nil,
		nil,
		table,
	)

	reload()
	return content
}

func buildObjectAdditionalTab(parent fyne.Window, provider data.AdminProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	latitudeEntry := widget.NewEntry()
	latitudeEntry.SetPlaceHolder("Широта (LATITUDE)")

	longitudeEntry := widget.NewEntry()
	longitudeEntry.SetPlaceHolder("Довгота (LONGITUDE)")

	reload := func() {
		coords, err := provider.GetObjectCoordinates(objn)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося завантажити координати")
			return
		}
		latitudeEntry.SetText(strings.TrimSpace(coords.Latitude))
		longitudeEntry.SetText(strings.TrimSpace(coords.Longitude))
		statusLabel.SetText("Координати завантажено")
	}

	save := func() {
		coords := data.AdminObjectCoordinates{
			Latitude:  strings.TrimSpace(latitudeEntry.Text),
			Longitude: strings.TrimSpace(longitudeEntry.Text),
		}
		if err := provider.SaveObjectCoordinates(objn, coords); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося зберегти координати")
			return
		}
		statusLabel.SetText("Координати збережено")
	}

	saveBtn := widget.NewButton("Зберегти координати", save)
	clearBtn := widget.NewButton("Очистити", func() {
		latitudeEntry.SetText("")
		longitudeEntry.SetText("")
		save()
	})
	mapPickBtn := widget.NewButton("Вибрати на карті", func() {
		showCoordinatesMapPicker(
			parent,
			strings.TrimSpace(latitudeEntry.Text),
			strings.TrimSpace(longitudeEntry.Text),
			func(lat, lon string) {
				latitudeEntry.SetText(lat)
				longitudeEntry.SetText(lon)
				statusLabel.SetText("Координати вибрано на карті")
			},
		)
	})
	refreshBtn := widget.NewButton("Оновити", reload)

	form := widget.NewForm(
		widget.NewFormItem("Широта:", latitudeEntry),
		widget.NewFormItem("Довгота:", longitudeEntry),
	)

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(saveBtn, clearBtn, mapPickBtn, layout.NewSpacer(), refreshBtn),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewPadded(form),
	)

	reload()
	return content
}

func showObjectPersonalEditor(
	parent fyne.Window,
	provider data.AdminProvider,
	title string,
	initial data.AdminObjectPersonal,
	onSave func(item data.AdminObjectPersonal) error,
	statusLabel *widget.Label,
	onDone func(),
) {
	numberEntry := widget.NewEntry()
	if initial.Number > 0 {
		numberEntry.SetText(strconv.FormatInt(initial.Number, 10))
	}
	numberEntry.SetPlaceHolder("1..999")
	surnameEntry := widget.NewEntry()
	surnameEntry.SetText(initial.Surname)
	nameEntry := widget.NewEntry()
	nameEntry.SetText(initial.Name)
	secNameEntry := widget.NewEntry()
	secNameEntry.SetText(initial.SecName)
	addressEntry := widget.NewEntry()
	addressEntry.SetText(initial.Address)
	phonesEntry := widget.NewEntry()
	phonesEntry.SetText(initial.Phones)
	phoneLookupLabel := widget.NewLabel("")
	phoneLookupLabel.Wrapping = fyne.TextWrapWord
	positionEntry := widget.NewEntry()
	positionEntry.SetText(initial.Position)
	notesEntry := widget.NewEntry()
	notesEntry.SetText(initial.Notes)
	isRangCheck := widget.NewCheck("ISRANG (старший/ранг)", nil)
	isRangCheck.SetChecked(initial.IsRang)
	if initial.ID == 0 {
		isRangCheck.SetChecked(true)
	}
	accessCheck := widget.NewCheck("Повний доступ до адмін-функцій (ACCESS1=1)", nil)
	accessCheck.SetChecked(initial.Access1 > 0)
	viberIDEntry := widget.NewEntry()
	viberIDEntry.SetText(initial.ViberID)
	viberIDEntry.SetPlaceHolder("Viber ID (необов'язково)")
	telegramIDEntry := widget.NewEntry()
	telegramIDEntry.SetText(initial.TelegramID)
	telegramIDEntry.SetPlaceHolder("Telegram ID (необов'язково)")
	createdAtLabel := widget.NewLabel(initial.CreatedAt)
	if strings.TrimSpace(initial.CreatedAt) == "" {
		createdAtLabel.SetText("буде встановлено автоматично")
	}
	trkCheck := widget.NewCheck("Перевіряючий ТРК", nil)
	trkCheck.SetChecked(initial.IsTRKTester)

	digitsCount := func(s string) int {
		cnt := 0
		for _, r := range s {
			if r >= '0' && r <= '9' {
				cnt++
			}
		}
		return cnt
	}

	applyPersonalLookup := func(found *data.AdminObjectPersonal) {
		if found == nil {
			return
		}
		if strings.TrimSpace(numberEntry.Text) == "" && found.Number > 0 {
			numberEntry.SetText(strconv.FormatInt(found.Number, 10))
		}
		surnameEntry.SetText(strings.TrimSpace(found.Surname))
		nameEntry.SetText(strings.TrimSpace(found.Name))
		secNameEntry.SetText(strings.TrimSpace(found.SecName))
		addressEntry.SetText(strings.TrimSpace(found.Address))
		positionEntry.SetText(strings.TrimSpace(found.Position))
		notesEntry.SetText(strings.TrimSpace(found.Notes))
		isRangCheck.SetChecked(found.IsRang)
		accessCheck.SetChecked(found.Access1 > 0)
		viberIDEntry.SetText(strings.TrimSpace(found.ViberID))
		telegramIDEntry.SetText(strings.TrimSpace(found.TelegramID))
		trkCheck.SetChecked(found.IsTRKTester)
		if strings.TrimSpace(createdAtLabel.Text) == "" || createdAtLabel.Text == "буде встановлено автоматично" {
			if strings.TrimSpace(found.CreatedAt) != "" {
				createdAtLabel.SetText(found.CreatedAt)
			}
		}

		source := "Знайдено контакт у базі, дані підтягнуто автоматично"
		if found.SourceObjN > 0 {
			source = fmt.Sprintf("Знайдено контакт у базі (об'єкт #%d), дані підтягнуто автоматично", found.SourceObjN)
		}
		phoneLookupLabel.SetText(source)
	}

	lastPhoneLookupRaw := ""
	tryLookupByPhone := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			lastPhoneLookupRaw = ""
			phoneLookupLabel.SetText("")
			return
		}
		if digitsCount(raw) < 10 {
			phoneLookupLabel.SetText("")
			return
		}
		if raw == lastPhoneLookupRaw {
			return
		}
		lastPhoneLookupRaw = raw

		found, err := provider.FindPersonalByPhone(raw)
		if err != nil {
			phoneLookupLabel.SetText("Не вдалося перевірити телефон у базі")
			return
		}
		if found == nil {
			phoneLookupLabel.SetText("")
			return
		}
		applyPersonalLookup(found)
	}
	phonesEntry.OnChanged = func(text string) {
		tryLookupByPhone(text)
	}
	phonesEntry.OnSubmitted = func(text string) {
		tryLookupByPhone(text)
	}

	form := widget.NewForm(
		widget.NewFormItem("№:", numberEntry),
		widget.NewFormItem("Створено:", createdAtLabel),
		widget.NewFormItem("Прізвище:", surnameEntry),
		widget.NewFormItem("Ім'я:", nameEntry),
		widget.NewFormItem("По батькові:", secNameEntry),
		widget.NewFormItem("Адреса:", addressEntry),
		widget.NewFormItem("Телефон:", container.NewVBox(phonesEntry, phoneLookupLabel)),
		widget.NewFormItem("Посада:", positionEntry),
		widget.NewFormItem("Примітка:", notesEntry),
		widget.NewFormItem("", isRangCheck),
		widget.NewFormItem("", accessCheck),
		widget.NewFormItem("Viber ID:", viberIDEntry),
		widget.NewFormItem("Telegram ID:", telegramIDEntry),
		widget.NewFormItem("", trkCheck),
	)

	dlg := dialog.NewCustomConfirm(title, "Зберегти", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}

		numRaw := strings.TrimSpace(numberEntry.Text)
		number := int64(0)
		if numRaw != "" {
			n, err := strconv.ParseInt(numRaw, 10, 64)
			if err != nil {
				statusLabel.SetText("Некоректний номер В/О")
				return
			}
			number = n
		}

		item := data.AdminObjectPersonal{
			Number:      number,
			Surname:     strings.TrimSpace(surnameEntry.Text),
			Name:        strings.TrimSpace(nameEntry.Text),
			SecName:     strings.TrimSpace(secNameEntry.Text),
			Address:     strings.TrimSpace(addressEntry.Text),
			Phones:      strings.TrimSpace(phonesEntry.Text),
			Position:    strings.TrimSpace(positionEntry.Text),
			Notes:       strings.TrimSpace(notesEntry.Text),
			IsRang:      isRangCheck.Checked,
			Access1:     boolToInt64(accessCheck.Checked),
			ViberID:     strings.TrimSpace(viberIDEntry.Text),
			TelegramID:  strings.TrimSpace(telegramIDEntry.Text),
			CreatedAt:   strings.TrimSpace(createdAtLabel.Text),
			IsTRKTester: trkCheck.Checked,
		}
		if err := onSave(item); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося зберегти В/О")
			return
		}
		onDone()
	}, parent)
	dlg.Show()
}

func showObjectZoneEditor(
	parent fyne.Window,
	title string,
	initial data.AdminObjectZone,
	onSave func(zone data.AdminObjectZone) error,
	statusLabel *widget.Label,
	onDone func(),
) {
	numberEntry := widget.NewEntry()
	if initial.ZoneNumber > 0 {
		numberEntry.SetText(strconv.FormatInt(initial.ZoneNumber, 10))
	}
	numberEntry.SetPlaceHolder("1..9999")

	descriptionEntry := widget.NewEntry()
	descriptionEntry.SetText(initial.Description)

	form := widget.NewForm(
		widget.NewFormItem("Номер:", numberEntry),
		widget.NewFormItem("Тип:", widget.NewLabel("пож.")),
		widget.NewFormItem("Опис:", descriptionEntry),
	)

	dlg := dialog.NewCustomConfirm(title, "Зберегти", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}

		zoneNumber, err := strconv.ParseInt(strings.TrimSpace(numberEntry.Text), 10, 64)
		if err != nil {
			statusLabel.SetText("Некоректний номер зони")
			return
		}

		zone := data.AdminObjectZone{
			ZoneNumber:    zoneNumber,
			ZoneType:      1,
			Description:   strings.TrimSpace(descriptionEntry.Text),
			EntryDelaySec: 0,
		}
		if err := onSave(zone); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося зберегти зону")
			return
		}
		onDone()
	}, parent)
	dlg.Show()
}

func showZoneFillDialog(parent fyne.Window, defaultCount int64, onApply func(count int64), statusLabel *widget.Label) {
	entry := widget.NewEntry()
	if defaultCount <= 0 {
		defaultCount = 24
	}
	entry.SetText(strconv.FormatInt(defaultCount, 10))
	entry.SetPlaceHolder("Кількість зон")

	form := widget.NewForm(
		widget.NewFormItem("Кількість зон:", entry),
	)

	dlg := dialog.NewCustomConfirm("Заповнення зон", "Застосувати", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}
		count, err := strconv.ParseInt(strings.TrimSpace(entry.Text), 10, 64)
		if err != nil {
			statusLabel.SetText("Некоректна кількість зон")
			return
		}
		onApply(count)
	}, parent)
	dlg.Show()
}

func suggestZoneFillCount(provider data.AdminProvider, objn int64, current []data.AdminObjectZone) int64 {
	maxZone := int64(0)
	for _, z := range current {
		if z.ZoneNumber > maxZone {
			maxZone = z.ZoneNumber
		}
	}

	card, err := provider.GetObjectCard(objn)
	if err == nil && card.PPKID > 0 {
		ppkItems, ppkErr := provider.ListPPKConstructor()
		if ppkErr == nil {
			for _, it := range ppkItems {
				if it.ID == card.PPKID && it.ZoneCount > 0 {
					return it.ZoneCount
				}
			}
		}
	}

	if maxZone > 0 {
		return maxZone
	}
	return 24
}

func focusIfOnCanvas(parent fyne.Window, target fyne.Focusable) {
	if parent == nil || target == nil {
		return
	}
	canvas := parent.Canvas()
	if canvas == nil {
		return
	}
	root := canvas.Content()
	if root == nil {
		return
	}
	targetObj, ok := target.(fyne.CanvasObject)
	if !ok {
		return
	}
	if !containsCanvasObject(root, targetObj) {
		return
	}
	canvas.Focus(target)
}

func containsCanvasObject(root fyne.CanvasObject, target fyne.CanvasObject) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	cont, ok := root.(*fyne.Container)
	if !ok {
		return false
	}
	for _, child := range cont.Objects {
		if containsCanvasObject(child, target) {
			return true
		}
	}
	return false
}

func showCoordinatesMapPicker(parent fyne.Window, initialLatRaw string, initialLonRaw string, onPick func(lat, lon string)) {
	centerLat, centerLon, zoom, hasObjectMarker := resolveInitialMapCenter(initialLatRaw, initialLonRaw)

	mapView := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(true),
		xwidget.WithScrollButtons(true),
		xwidget.AtZoomLevel(zoom),
		xwidget.AtLatLon(centerLat, centerLon),
	)

	crosshair := canvas.NewText("+", color.NRGBA{R: 220, G: 20, B: 20, A: 255})
	crosshair.TextSize = 36
	crosshair.TextStyle.Bold = true

	objectMarker := canvas.NewCircle(color.NRGBA{R: 255, G: 30, B: 30, A: 210})
	objectMarker.Resize(fyne.NewSize(14, 14))
	objectMarker.Hide()
	markerLayer := container.NewWithoutLayout(objectMarker)

	mapStack := container.NewStack(
		mapView,
		markerLayer,
		container.NewCenter(crosshair),
	)

	centerLabel := widget.NewLabel("")
	updateCenterLabel := func() {
		lat, lon, err := mapCenterLatLon(mapView)
		if err != nil {
			centerLabel.SetText("Центр: невизначено")
			return
		}
		centerLabel.SetText(fmt.Sprintf("Центр: %s, %s", formatCoordinate(lat), formatCoordinate(lon)))
	}
	updateCenterLabel()

	objectMarkerLat := centerLat
	objectMarkerLon := centerLon
	updateObjectMarker := func() {
		if !hasObjectMarker {
			objectMarker.Hide()
			return
		}
		x, y, ok := mapLatLonToCanvasPoint(mapView, objectMarkerLat, objectMarkerLon)
		if !ok {
			objectMarker.Hide()
			return
		}
		size := mapView.Size()
		if x < -20 || y < -20 || x > size.Width+20 || y > size.Height+20 {
			objectMarker.Hide()
			return
		}
		objectMarker.Move(fyne.NewPos(x-7, y-7))
		objectMarker.Show()
		objectMarker.Refresh()
	}

	pickerWin := fyne.CurrentApp().NewWindow("Вибір координат на карті")
	pickerWin.Resize(fyne.NewSize(980, 680))

	useCenterBtn := widget.NewButton("Вибрати центр карти", func() {
		lat, lon, err := mapCenterLatLon(mapView)
		if err != nil {
			dialog.ShowError(err, pickerWin)
			return
		}
		saveLastMapCenter(lat, lon)
		if onPick != nil {
			onPick(formatCoordinate(lat), formatCoordinate(lon))
		}
		pickerWin.Close()
	})
	refreshCenterBtn := widget.NewButton("Оновити центр", func() {
		updateCenterLabel()
		updateObjectMarker()
	})
	mapSettingsBtn := widget.NewButton("Налаштування карти", func() {
		showMapCenterSettingsDialog(pickerWin, func(lat, lon float64, zoom int) {
			mapView.Zoom(zoom)
			mapView.PanToLatLon(lat, lon)
			updateCenterLabel()
			updateObjectMarker()
		})
	})
	cancelBtn := widget.NewButton("Скасувати", func() { pickerWin.Close() })

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Перетягніть/масштабуйте карту, '+' у центрі = точка вибору."),
			widget.NewLabel("Якщо в об'єкта вже є координати, червоний маркер показує поточну позицію об'єкта."),
			widget.NewSeparator(),
		),
		container.NewHBox(centerLabel, layout.NewSpacer(), mapSettingsBtn, refreshCenterBtn, useCenterBtn, cancelBtn),
		nil,
		nil,
		mapStack,
	)
	pickerWin.SetContent(content)

	// Періодично оновлюємо marker-позицію під час pan/zoom.
	done := make(chan struct{})
	ticker := time.NewTicker(160 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				fyne.Do(func() {
					updateObjectMarker()
				})
			case <-done:
				return
			}
		}
	}()
	pickerWin.SetOnClosed(func() {
		close(done)
		ticker.Stop()
	})

	updateObjectMarker()
	pickerWin.Show()
}

func parseLatLon(latRaw string, lonRaw string) (float64, float64, bool) {
	lat, err := parseCoordinate(latRaw)
	if err != nil {
		return 0, 0, false
	}
	lon, err := parseCoordinate(lonRaw)
	if err != nil {
		return 0, 0, false
	}
	if lat < -85 || lat > 85 {
		return 0, 0, false
	}
	if lon < -180 || lon > 180 {
		return 0, 0, false
	}
	return lat, lon, true
}

func parseCoordinate(raw string) (float64, error) {
	clean := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if clean == "" {
		return 0, fmt.Errorf("empty coordinate")
	}
	return strconv.ParseFloat(clean, 64)
}

func formatCoordinate(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func mapCenterLatLon(m *xwidget.Map) (float64, float64, error) {
	state, err := readMapInternalState(m)
	if err != nil {
		return 0, 0, err
	}
	xTile := state.mx + (state.centerX-state.midTileX-state.offsetX*state.scale)/state.tilePx
	yTile := state.my + (state.centerY-state.midTileY-state.offsetY*state.scale)/state.tilePx
	return tileXYToLatLon(xTile, yTile, state.n)
}

func mapLatLonToCanvasPoint(m *xwidget.Map, lat float64, lon float64) (float32, float32, bool) {
	state, err := readMapInternalState(m)
	if err != nil {
		return 0, 0, false
	}
	xTile, yTile := latLonToTileXY(lat, lon, state.n)
	px := state.midTileX + (xTile-state.mx)*state.tilePx + state.offsetX*state.scale
	py := state.midTileY + (yTile-state.my)*state.tilePx + state.offsetY*state.scale
	if math.IsNaN(px) || math.IsNaN(py) || math.IsInf(px, 0) || math.IsInf(py, 0) {
		return 0, 0, false
	}
	return float32(px / state.scale), float32(py / state.scale), true
}

type mapInternalState struct {
	mx, my             float64
	n                  float64
	offsetX, offsetY   float64
	scale              float64
	centerX, centerY   float64
	midTileX, midTileY float64
	tilePx             float64
}

func readMapInternalState(m *xwidget.Map) (mapInternalState, error) {
	if m == nil {
		return mapInternalState{}, fmt.Errorf("map is nil")
	}

	mv := reflect.ValueOf(m)
	if mv.Kind() != reflect.Pointer || mv.IsNil() {
		return mapInternalState{}, fmt.Errorf("invalid map value")
	}
	me := mv.Elem()

	getIntField := func(name string) (int, error) {
		f := me.FieldByName(name)
		if !f.IsValid() || f.Kind() != reflect.Int {
			return 0, fmt.Errorf("map field %s is unavailable", name)
		}
		return int(f.Int()), nil
	}
	getFloatField := func(name string) (float64, error) {
		f := me.FieldByName(name)
		if !f.IsValid() {
			return 0, fmt.Errorf("map field %s is unavailable", name)
		}
		switch f.Kind() {
		case reflect.Float32, reflect.Float64:
			return f.Float(), nil
		default:
			return 0, fmt.Errorf("map field %s has unsupported type", name)
		}
	}

	x, err := getIntField("x")
	if err != nil {
		return mapInternalState{}, err
	}
	y, err := getIntField("y")
	if err != nil {
		return mapInternalState{}, err
	}
	zoom, err := getIntField("zoom")
	if err != nil {
		return mapInternalState{}, err
	}
	offsetX, err := getFloatField("offsetX")
	if err != nil {
		return mapInternalState{}, err
	}
	offsetY, err := getFloatField("offsetY")
	if err != nil {
		return mapInternalState{}, err
	}

	if zoom < 0 || zoom > 19 {
		return mapInternalState{}, fmt.Errorf("invalid zoom level")
	}
	count := 1 << zoom
	n := float64(count)
	half := int(float32(count)/2 - 0.5)
	mx := x + half
	my := y + half

	scale := float64(1)
	if c := fyne.CurrentApp().Driver().CanvasForObject(m); c != nil {
		scale = float64(c.Scale())
		if scale <= 0 {
			scale = 1
		}
	}

	size := m.Size()
	wPx := int(math.Round(float64(size.Width) * scale))
	hPx := int(math.Round(float64(size.Height) * scale))
	if wPx <= 0 || hPx <= 0 {
		return mapInternalState{}, fmt.Errorf("map is not sized yet")
	}

	tilePx := int(math.Round(256 * scale))
	if tilePx <= 0 {
		return mapInternalState{}, fmt.Errorf("invalid tile size")
	}

	midTileX := (wPx - tilePx*2) / 2
	midTileY := (hPx - tilePx*2) / 2
	if zoom == 0 {
		midTileX += tilePx / 2
		midTileY += tilePx / 2
	}

	return mapInternalState{
		mx:       float64(mx),
		my:       float64(my),
		n:        n,
		offsetX:  offsetX,
		offsetY:  offsetY,
		scale:    scale,
		centerX:  float64(wPx) / 2,
		centerY:  float64(hPx) / 2,
		midTileX: float64(midTileX),
		midTileY: float64(midTileY),
		tilePx:   float64(tilePx),
	}, nil
}

func latLonToTileXY(lat float64, lon float64, n float64) (float64, float64) {
	xTile := (lon + 180.0) / 360.0 * n
	latRad := lat * math.Pi / 180.0
	yTile := (1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n
	return xTile, yTile
}

func tileXYToLatLon(xTile float64, yTile float64, n float64) (float64, float64, error) {
	lon := xTile/n*360.0 - 180.0
	latRad := math.Atan(math.Sinh(math.Pi * (1 - 2*yTile/n)))
	lat := latRad * 180.0 / math.Pi
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return 0, 0, fmt.Errorf("failed to resolve coordinates")
	}
	return lat, lon, nil
}

func resolveInitialMapCenter(initialLatRaw string, initialLonRaw string) (float64, float64, int, bool) {
	const (
		lvivLat = 49.8397
		lvivLon = 24.0297
		kyivLat = 50.4501
		kyivLon = 30.5234
		mapZoom = 12
	)
	if lat, lon, ok := parseLatLon(initialLatRaw, initialLonRaw); ok {
		return lat, lon, mapZoom, true
	}

	mode := mapCenterModeLviv
	prefs := fyne.CurrentApp().Preferences()
	if prefs != nil {
		if m := strings.TrimSpace(prefs.String(mapCenterModePrefKey)); m != "" {
			mode = m
		}
	}

	switch mode {
	case mapCenterModeKyiv:
		return kyivLat, kyivLon, mapZoom, false
	case mapCenterModeCustom:
		if prefs != nil {
			lat, latErr := parseCoordinate(prefs.String(mapCenterCustomLatKey))
			lon, lonErr := parseCoordinate(prefs.String(mapCenterCustomLonKey))
			if latErr == nil && lonErr == nil && lat >= -85 && lat <= 85 && lon >= -180 && lon <= 180 {
				return lat, lon, mapZoom, false
			}
		}
	case mapCenterModeLast:
		if prefs != nil {
			lat, latErr := parseCoordinate(prefs.String(mapCenterLastLatPrefKey))
			lon, lonErr := parseCoordinate(prefs.String(mapCenterLastLonPrefKey))
			if latErr == nil && lonErr == nil && lat >= -85 && lat <= 85 && lon >= -180 && lon <= 180 {
				return lat, lon, mapZoom, false
			}
		}
	}

	return lvivLat, lvivLon, mapZoom, false
}

func saveLastMapCenter(lat float64, lon float64) {
	prefs := fyne.CurrentApp().Preferences()
	if prefs == nil {
		return
	}
	prefs.SetString(mapCenterLastLatPrefKey, formatCoordinate(lat))
	prefs.SetString(mapCenterLastLonPrefKey, formatCoordinate(lon))
}

func showMapCenterSettingsDialog(parent fyne.Window, onApply func(lat, lon float64, zoom int)) {
	prefs := fyne.CurrentApp().Preferences()
	mode := mapCenterModeLviv
	customLat := "49.8397"
	customLon := "24.0297"
	if prefs != nil {
		if m := strings.TrimSpace(prefs.String(mapCenterModePrefKey)); m != "" {
			mode = m
		}
		if v := strings.TrimSpace(prefs.String(mapCenterCustomLatKey)); v != "" {
			customLat = v
		}
		if v := strings.TrimSpace(prefs.String(mapCenterCustomLonKey)); v != "" {
			customLon = v
		}
	}

	modeSelect := widget.NewSelect([]string{
		"Львів",
		"Київ",
		"Власні координати",
		"Остання вибрана точка",
	}, nil)
	switch mode {
	case mapCenterModeKyiv:
		modeSelect.SetSelected("Київ")
	case mapCenterModeCustom:
		modeSelect.SetSelected("Власні координати")
	case mapCenterModeLast:
		modeSelect.SetSelected("Остання вибрана точка")
	default:
		modeSelect.SetSelected("Львів")
	}

	customLatEntry := widget.NewEntry()
	customLonEntry := widget.NewEntry()
	customLatEntry.SetText(customLat)
	customLonEntry.SetText(customLon)
	customLatEntry.SetPlaceHolder("49.8397")
	customLonEntry.SetPlaceHolder("24.0297")

	updateCustomState := func() {
		enabled := modeSelect.Selected == "Власні координати"
		if enabled {
			customLatEntry.Enable()
			customLonEntry.Enable()
			return
		}
		customLatEntry.Disable()
		customLonEntry.Disable()
	}
	modeSelect.OnChanged = func(string) { updateCustomState() }
	updateCustomState()

	form := widget.NewForm(
		widget.NewFormItem("Центр мапи при відкритті:", modeSelect),
		widget.NewFormItem("Широта (власна):", customLatEntry),
		widget.NewFormItem("Довгота (власна):", customLonEntry),
	)

	dialog.ShowCustomConfirm(
		"Налаштування карти",
		"Зберегти",
		"Скасувати",
		container.NewPadded(form),
		func(ok bool) {
			if !ok {
				return
			}

			selectedMode := mapCenterModeLviv
			switch modeSelect.Selected {
			case "Київ":
				selectedMode = mapCenterModeKyiv
			case "Власні координати":
				selectedMode = mapCenterModeCustom
			case "Остання вибрана точка":
				selectedMode = mapCenterModeLast
			}

			customLatVal := strings.TrimSpace(customLatEntry.Text)
			customLonVal := strings.TrimSpace(customLonEntry.Text)
			if selectedMode == mapCenterModeCustom {
				lat, lon, ok := parseLatLon(customLatVal, customLonVal)
				if !ok {
					dialog.ShowError(fmt.Errorf("некоректні власні координати"), parent)
					return
				}
				customLatVal = formatCoordinate(lat)
				customLonVal = formatCoordinate(lon)
			}

			if prefs != nil {
				prefs.SetString(mapCenterModePrefKey, selectedMode)
				prefs.SetString(mapCenterCustomLatKey, customLatVal)
				prefs.SetString(mapCenterCustomLonKey, customLonVal)
			}

			if onApply != nil {
				lat, lon, zoom, _ := resolveInitialMapCenter("", "")
				onApply(lat, lon, zoom)
			}
		},
		parent,
	)
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
