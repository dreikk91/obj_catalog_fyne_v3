package dialogs

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
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

	mapDefaultLvivLat = 49.8397
	mapDefaultLvivLon = 24.0297
	mapDefaultKyivLat = 50.4501
	mapDefaultKyivLon = 30.5234
	mapDefaultZoom    = 12
)

func buildObjectPersonalTab(parent fyne.Window, provider contracts.AdminObjectPersonalTabProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	vm := viewmodels.NewObjectPersonalsTabViewModel()

	table := widget.NewTable(
		func() (int, int) { return vm.Count() + 1, 6 },
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
			it, ok := vm.ItemAt(itemIdx)
			if !ok {
				lbl.SetText("")
				return
			}
			switch id.Col {
			case 0:
				lbl.SetText(strconv.FormatInt(it.Number, 10))
			case 1:
				lbl.SetText(vm.FullName(it))
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
		vm.SelectByTableRow(id.Row)
	}

	reload := func() {
		loaded, err := provider.ListObjectPersonals(objn)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося завантажити В/О")
			return
		}
		vm.SetItems(loaded)
		table.UnselectAll()
		table.Refresh()
		statusLabel.SetText(vm.CountStatusText())
	}

	addBtn := widget.NewButton("Додати", func() {
		showObjectPersonalEditor(parent, provider, "Додати В/О", contracts.AdminObjectPersonal{}, func(item contracts.AdminObjectPersonal) error {
			return provider.AddObjectPersonal(objn, item)
		}, statusLabel, func() {
			reload()
			statusLabel.SetText("В/О додано")
		})
	})

	editBtn := widget.NewButton("Змінити", func() {
		initial, ok := vm.SelectedItem()
		if !ok {
			statusLabel.SetText("Виберіть В/О у таблиці")
			return
		}
		showObjectPersonalEditor(parent, provider, "Редагування В/О", initial, func(item contracts.AdminObjectPersonal) error {
			return provider.UpdateObjectPersonal(objn, vm.PrepareUpdatedItem(initial, item))
		}, statusLabel, func() {
			reload()
			statusLabel.SetText("В/О оновлено")
		})
	})

	deleteBtn := widget.NewButton("Видалити", func() {
		target, ok := vm.SelectedItem()
		if !ok {
			statusLabel.SetText("Виберіть В/О у таблиці")
			return
		}
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити запис \"%s\"?", vm.FullName(target)),
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
	tableScroll := container.NewScroll(table)
	tableScroll.SetMinSize(fyne.NewSize(420, 260))

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addBtn, editBtn, deleteBtn, layout.NewSpacer(), refreshBtn),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		tableScroll,
	)

	reload()
	return content
}

func buildObjectZonesTab(parent fyne.Window, provider contracts.AdminObjectZonesTabProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	vm := viewmodels.NewObjectZonesTabViewModel()

	quickNameEntry := widget.NewEntry()
	quickNameEntry.SetPlaceHolder("Назва зони (Enter -> наступна зона)")
	selectedZoneLabel := widget.NewLabel("Зона: —")

	table := widget.NewTable(
		func() (int, int) { return vm.Count() + 1, 3 },
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
			it, ok := vm.ItemAt(itemIdx)
			if !ok {
				lbl.SetText("")
				return
			}
			switch id.Col {
			case 0:
				zonen := vm.EffectiveZoneNumberAt(itemIdx)
				lbl.SetText(strconv.FormatInt(zonen, 10))
			case 1:
				lbl.SetText("пож.")
			default:
				lbl.SetText(strings.TrimSpace(it.Description))
			}
		},
	)
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

	updateSelectedZoneLabel := func() {
		selectedZoneLabel.SetText(vm.SelectedZoneLabel())
	}

	ensureZoneExists := func(zoneNumber int64, defaultDescription string) error {
		if vm.FindRowByZoneNumber(zoneNumber) >= 0 {
			return nil
		}
		zone, err := vm.BuildZoneForCreate(zoneNumber, defaultDescription)
		if err != nil {
			return err
		}
		return provider.AddObjectZone(objn, zone)
	}

	selectByZoneNumber := func(zoneNumber int64, focusQuickName bool) {
		if !vm.SelectZoneByNumber(zoneNumber) {
			table.UnselectAll()
			quickNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}

		targetRow := vm.SelectedRow()
		table.Select(widget.TableCellID{Row: targetRow + 1, Col: 0})
		quickNameEntry.SetText(vm.SelectedZoneDescription())
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
		vm.SetItems(loaded)
		table.Refresh()
		applyZoneTableLayout()
		statusLabel.SetText(vm.CountStatusText())
		selectByZoneNumber(targetZoneNumber, focusQuickName)
	}

	reload := func() {
		reloadAndSelect(0, false)
	}

	table.OnSelected = func(id widget.TableCellID) {
		if !vm.SelectByTableRow(id.Row) {
			quickNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		quickNameEntry.SetText(vm.SelectedZoneDescription())
		updateSelectedZoneLabel()
		// Даємо змогу одразу вводити назву наступної/поточної зони.
		focusIfOnCanvas(parent, quickNameEntry)
	}

	moveToNextZone := func() {
		if _, ok := vm.SelectedItem(); !ok {
			if vm.Count() == 0 {
				if err := ensureZoneExists(1, strings.TrimSpace(quickNameEntry.Text)); err != nil {
					dialog.ShowError(err, parent)
					statusLabel.SetText("Не вдалося додати першу зону")
					return
				}
				reloadAndSelect(1, true)
				statusLabel.SetText("Додано зону #1")
				return
			}
			selectByZoneNumber(0, true)
		}

		current, currentZoneNumber, ok := vm.PrepareSelectedZoneForSave(quickNameEntry.Text)
		if !ok {
			statusLabel.SetText("Виберіть зону у таблиці")
			return
		}
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
		nextZoneNumber := vm.NextZoneNumberForAdd()
		if err := ensureZoneExists(nextZoneNumber, ""); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося додати зону")
			return
		}
		reloadAndSelect(nextZoneNumber, true)
		statusLabel.SetText(fmt.Sprintf("Готово до введення зони #%d", nextZoneNumber))
	})

	editBtn := widget.NewButton("Змінити", func() {
		if vm.Count() == 0 {
			if err := ensureZoneExists(1, ""); err != nil {
				dialog.ShowError(err, parent)
				statusLabel.SetText("Не вдалося створити першу зону")
				return
			}
			reloadAndSelect(1, true)
			statusLabel.SetText("Створено зону #1, можна вводити назву")
			return
		}
		if _, ok := vm.SelectedItem(); !ok {
			selectByZoneNumber(0, true)
			statusLabel.SetText("Виберіть зону і вводьте назву")
			return
		}
		zoneNumber, ok := vm.SelectedZoneNumber()
		if !ok {
			statusLabel.SetText("Виберіть зону і вводьте назву")
			return
		}
		updateSelectedZoneLabel()
		focusIfOnCanvas(parent, quickNameEntry)
		statusLabel.SetText(fmt.Sprintf("Редагування зони #%d: введіть назву і натисніть Enter", zoneNumber))
	})

	deleteBtn := widget.NewButton("Видалити", func() {
		target, ok := vm.SelectedItem()
		if !ok {
			statusLabel.SetText("Виберіть зону у таблиці")
			return
		}
		targetZoneNumber, ok := vm.SelectedZoneNumber()
		if !ok {
			targetZoneNumber = target.ZoneNumber
		}
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити зону #%d?", targetZoneNumber),
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
		defaultCount := suggestZoneFillCount(provider, objn, vm.Items())
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
	tableScroll := container.NewScroll(table)
	tableScroll.SetMinSize(fyne.NewSize(420, 260))

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
		tableScroll,
	)

	reload()
	return content
}

func buildObjectAdditionalTab(
	parent fyne.Window,
	provider contracts.AdminObjectAdditionalTabProvider,
	objn int64,
	statusLabel *widget.Label,
	getAddressFromObjectTab func() string,
	setRegionInObjectTab func(regionID int64) bool,
) fyne.CanvasObject {
	vm := viewmodels.NewObjectAdditionalTabViewModel()

	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("Адреса для геопошуку")

	latitudeEntry := widget.NewEntry()
	latitudeEntry.SetPlaceHolder("Широта (LATITUDE)")

	longitudeEntry := widget.NewEntry()
	longitudeEntry.SetPlaceHolder("Довгота (LONGITUDE)")

	syncAddressFromObjectTab := func() {
		address, ok := vm.AddressFromObjectTab(getAddressFromObjectTab)
		if !ok {
			return
		}
		addressEntry.SetText(address)
	}

	geoByAddress := func(addressRaw string) (string, string, []string, error) {
		address, err := vm.RequireLookupAddress(addressRaw)
		if err != nil {
			return "", "", nil, err
		}
		lat, lon, districtHints, err := geocodeAddress(address)
		if err != nil {
			return "", "", nil, err
		}
		vm.RememberGeocode(address, districtHints)
		return lat, lon, districtHints, nil
	}

	reload := func() {
		coords, err := provider.GetObjectCoordinates(objn)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося завантажити координати")
			return
		}
		syncAddressFromObjectTab()
		latitudeEntry.SetText(strings.TrimSpace(coords.Latitude))
		longitudeEntry.SetText(strings.TrimSpace(coords.Longitude))
		statusLabel.SetText("Координати завантажено")
	}

	save := func() {
		coords := vm.BuildCoordinates(latitudeEntry.Text, longitudeEntry.Text)
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
	findByAddressBtn := widget.NewButton("Координати з адреси", func() {
		lat, lon, districtHints, err := geoByAddress(addressEntry.Text)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося знайти координати за адресою")
			return
		}
		latitudeEntry.SetText(lat)
		longitudeEntry.SetText(lon)
		if len(districtHints) > 0 {
			statusLabel.SetText(fmt.Sprintf("Знайдено координати за адресою. Можна також заповнити район (%s)", districtHints[0]))
			return
		}
		statusLabel.SetText("Знайдено координати за адресою")
	})
	fillDistrictBtn := widget.NewButton("Район з адреси", func() {
		address, err := vm.RequireLookupAddress(addressEntry.Text)
		if err != nil {
			statusLabel.SetText(err.Error())
			return
		}
		hints, ok := vm.CachedDistrictHintsForAddress(address)
		if !ok {
			_, _, resolvedHints, err := geoByAddress(address)
			if err != nil {
				dialog.ShowError(err, parent)
				statusLabel.SetText("Не вдалося визначити район за адресою")
				return
			}
			hints = resolvedHints
		}
		regionID, regionName, err := resolveRegionByAddressHints(provider, hints)
		if err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося підібрати район за адресою")
			return
		}
		if setRegionInObjectTab != nil && setRegionInObjectTab(regionID) {
			statusLabel.SetText(fmt.Sprintf("Район встановлено: %s (натисніть \"Зберегти\" у картці об'єкта)", regionName))
			return
		}
		statusLabel.SetText(fmt.Sprintf("Знайдено район: %s, але не вдалося застосувати у вкладці \"Об'єкт\"", regionName))
	})
	useObjectAddressBtn := widget.NewButton("Взяти адресу з Об'єкта", func() {
		syncAddressFromObjectTab()
		statusLabel.SetText("Адресу синхронізовано зі вкладки \"Об'єкт\"")
	})
	refreshBtn := widget.NewButton("Оновити", reload)

	form := widget.NewForm(
		widget.NewFormItem("Адреса:", addressEntry),
		widget.NewFormItem("Широта:", latitudeEntry),
		widget.NewFormItem("Довгота:", longitudeEntry),
	)

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(saveBtn, clearBtn, mapPickBtn, findByAddressBtn, fillDistrictBtn),
			container.NewHBox(useObjectAddressBtn, layout.NewSpacer(), refreshBtn),
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

func geocodeAddress(address string) (string, string, []string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", "", nil, fmt.Errorf("адреса порожня")
	}

	target := buildGeocodeTarget(address)
	queries := buildGeocodeQueries(target.Cleaned)
	bestScore := math.Inf(-1)
	var best geocodeCandidate
	bestFound := false

	for _, query := range queries {
		rows, err := geocodeCandidatesForQuery(query)
		if err != nil {
			return "", "", nil, err
		}
		for _, row := range rows {
			score := scoreGeocodeCandidate(target, row)
			if !bestFound || score > bestScore {
				bestScore = score
				best = row
				bestFound = true
			}
		}
		// Достатньо точний збіг (місто+вулиця+будинок) - далі мережеві запити не потрібні.
		if bestFound && bestScore >= 90 {
			break
		}
	}

	if bestFound {
		lat := strings.TrimSpace(best.Lat)
		lon := strings.TrimSpace(best.Lon)
		if lat == "" || lon == "" {
			return "", "", nil, fmt.Errorf("геосервіс не повернув координати")
		}
		return lat, lon, collectDistrictHints(best.Address, best.DisplayName), nil
	}

	return "", "", nil, fmt.Errorf("адресу не знайдено")
}

// GeocodeAddressExact повертає координати з максимально точним підбором.
// Використовується також утилітою масової перевірки адрес.
func GeocodeAddressExact(address string) (lat string, lon string, cleaned string, err error) {
	target := buildGeocodeTarget(address)
	lat, lon, _, err = geocodeAddress(address)
	return lat, lon, target.Cleaned, err
}

// GeocodePreviewQueries повертає всі запити, які будуть використані для геопошуку.
func GeocodePreviewQueries(address string) []string {
	target := buildGeocodeTarget(address)
	return buildGeocodeQueries(target.Cleaned)
}

type geocodeTarget struct {
	Cleaned string
	City    string
	Street  string
	House   string
}

type geocodeCandidate struct {
	Lat         string            `json:"lat"`
	Lon         string            `json:"lon"`
	DisplayName string            `json:"display_name"`
	Address     map[string]string `json:"address"`
	Importance  float64           `json:"importance"`
	Class       string            `json:"class"`
	Type        string            `json:"type"`
}

var (
	geocodeRequestMu    sync.Mutex
	geocodeLastRequest  time.Time
	geocodeMinInterval  = 1100 * time.Millisecond
	geocodeHTTPClient   = &http.Client{Timeout: 14 * time.Second}
	geocodeMaxRetry429  = 3
	geocodeRetryBackoff = 2 * time.Second
)

func buildGeocodeTarget(address string) geocodeTarget {
	cleaned := normalizeAddressForGeocode(address)
	city, street, house, _ := parseAddressComponents(cleaned)
	if city == "" {
		if cityOnly, ok := parseCityOnly(cleaned); ok {
			city = cityOnly
		}
	}
	return geocodeTarget{
		Cleaned: cleaned,
		City:    city,
		Street:  street,
		House:   house,
	}
}

func geocodeCandidatesForQuery(query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "jsonv2")
	params.Set("limit", "8")
	params.Set("addressdetails", "1")
	params.Set("accept-language", "uk")
	params.Set("countrycodes", "ua")
	params.Set("dedupe", "0")

	searchURL := "https://nominatim.openstreetmap.org/search?" + params.Encode()

	var last429Details string
	for attempt := 0; attempt <= geocodeMaxRetry429; attempt++ {
		waitForGeocodeRequestSlot()

		req, err := http.NewRequest(http.MethodGet, searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("не вдалося сформувати запит геопошуку: %w", err)
		}
		req.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")

		resp, err := geocodeHTTPClient.Do(req)
		if err != nil {
			if attempt < geocodeMaxRetry429 {
				time.Sleep(time.Duration(attempt+1) * geocodeRetryBackoff)
				continue
			}
			phRows, phErr := geocodeCandidatesPhoton(query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
			if phErr != nil {
				return nil, fmt.Errorf("помилка запиту геопошуку: %v; fallback photon помилка: %v", err, phErr)
			}
			return nil, fmt.Errorf("помилка запиту геопошуку: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			_ = resp.Body.Close()
			last429Details = strings.TrimSpace(string(body))
			if attempt < geocodeMaxRetry429 {
				time.Sleep(time.Duration(attempt+1) * geocodeRetryBackoff)
				continue
			}
			phRows, phErr := geocodeCandidatesPhoton(query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
			if phErr != nil {
				return nil, fmt.Errorf("геосервіс повернув 429 (%s), fallback photon помилка: %v", last429Details, phErr)
			}
			return nil, fmt.Errorf("геосервіс повернув 429: %s", last429Details)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			_ = resp.Body.Close()
			return nil, fmt.Errorf("геосервіс повернув %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var rows []geocodeCandidate
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rows)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("не вдалося обробити відповідь геосервісу: %w", decodeErr)
		}
		if len(rows) == 0 {
			phRows, phErr := geocodeCandidatesPhoton(query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
		}
		return rows, nil
	}

	phRows, phErr := geocodeCandidatesPhoton(query)
	if phErr == nil && len(phRows) > 0 {
		return phRows, nil
	}
	if phErr != nil {
		return nil, fmt.Errorf("геосервіс недоступний, fallback photon помилка: %v", phErr)
	}
	return nil, fmt.Errorf("геосервіс недоступний")
}

func waitForGeocodeRequestSlot() {
	geocodeRequestMu.Lock()
	defer geocodeRequestMu.Unlock()

	if !geocodeLastRequest.IsZero() {
		wait := geocodeMinInterval - time.Since(geocodeLastRequest)
		if wait > 0 {
			time.Sleep(wait)
		}
	}
	geocodeLastRequest = time.Now()
}

func geocodeCandidatesPhoton(query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("lang", "uk")
	params.Set("limit", "8")
	photonURL := "https://photon.komoot.io/api/?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, photonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("не вдалося сформувати запит photon: %w", err)
	}
	req.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")

	resp, err := geocodeHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("помилка запиту photon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("photon повернув %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Name        string `json:"name"`
				Street      string `json:"street"`
				HouseNumber string `json:"housenumber"`
				City        string `json:"city"`
				District    string `json:"district"`
				State       string `json:"state"`
				Country     string `json:"country"`
				CountryCode string `json:"countrycode"`
				OSMKey      string `json:"osm_key"`
				OSMValue    string `json:"osm_value"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("не вдалося обробити відповідь photon: %w", err)
	}

	rows := make([]geocodeCandidate, 0, len(payload.Features))
	for _, f := range payload.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		lon := strconv.FormatFloat(f.Geometry.Coordinates[0], 'f', 7, 64)
		lat := strconv.FormatFloat(f.Geometry.Coordinates[1], 'f', 7, 64)
		address := map[string]string{
			"road":         strings.TrimSpace(f.Properties.Street),
			"house_number": strings.TrimSpace(f.Properties.HouseNumber),
			"city":         strings.TrimSpace(f.Properties.City),
			"district":     strings.TrimSpace(f.Properties.District),
			"state":        strings.TrimSpace(f.Properties.State),
			"country":      strings.TrimSpace(f.Properties.Country),
			"country_code": strings.TrimSpace(f.Properties.CountryCode),
		}

		displayParts := []string{
			strings.TrimSpace(f.Properties.Name),
			strings.TrimSpace(f.Properties.Street),
			strings.TrimSpace(f.Properties.HouseNumber),
			strings.TrimSpace(f.Properties.City),
			strings.TrimSpace(f.Properties.State),
		}
		displayFiltered := make([]string, 0, len(displayParts))
		for _, p := range displayParts {
			if p != "" {
				displayFiltered = append(displayFiltered, p)
			}
		}

		rows = append(rows, geocodeCandidate{
			Lat:         lat,
			Lon:         lon,
			DisplayName: strings.Join(displayFiltered, ", "),
			Address:     address,
			Importance:  0,
			Class:       strings.TrimSpace(f.Properties.OSMKey),
			Type:        strings.TrimSpace(f.Properties.OSMValue),
		})
	}

	return rows, nil
}

func geocodeAutocompleteCandidates(query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	rows, err := geocodeCandidatesPhoton(query)
	if err == nil && len(rows) > 0 {
		return rows, nil
	}
	if err == nil {
		return geocodeCandidatesForQuery(query)
	}

	fallbackRows, fallbackErr := geocodeCandidatesForQuery(query)
	if fallbackErr != nil {
		return nil, err
	}
	return fallbackRows, nil
}

func geocodeSuggestionOptions(rows []geocodeCandidate) ([]string, map[string]geocodeCandidate) {
	options := make([]string, 0, len(rows))
	items := make(map[string]geocodeCandidate, len(rows))
	seen := make(map[string]int, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row.DisplayName)
		if label == "" {
			label = strings.TrimSpace(strings.Join([]string{
				firstAddressValue(row.Address, "road", "street", "pedestrian", "residential"),
				firstAddressValue(row.Address, "house_number"),
				firstAddressValue(row.Address, "city", "town", "village"),
			}, ", "))
		}
		label = strings.Trim(label, " ,")
		if label == "" {
			continue
		}
		if count := seen[label]; count > 0 {
			label = fmt.Sprintf("%s [%s, %s]", label, strings.TrimSpace(row.Lat), strings.TrimSpace(row.Lon))
		}
		seen[label]++
		options = append(options, label)
		items[label] = row
	}
	return options, items
}

func scoreGeocodeCandidate(target geocodeTarget, row geocodeCandidate) float64 {
	score := row.Importance * 10

	candidateCity := firstAddressValue(row.Address, "city", "town", "village", "hamlet", "municipality")
	candidateStreet := firstAddressValue(row.Address, "road", "pedestrian", "residential", "street")
	candidateHouse := firstAddressValue(row.Address, "house_number")

	if target.City != "" {
		score += similarityScore(target.City, candidateCity, 38, 18, -7)
	}
	if target.Street != "" {
		// Для вулиці перевіряємо також display_name, бо інколи road порожній.
		streetScore := similarityScore(target.Street, candidateStreet, 35, 16, -6)
		if streetScore < 0 {
			streetScore = similarityScore(target.Street, row.DisplayName, 22, 10, -3)
		}
		score += streetScore
	}
	if target.House != "" {
		score += houseMatchScore(target.House, candidateHouse, row.DisplayName)
	}

	if strings.EqualFold(strings.TrimSpace(row.Address["country_code"]), "ua") {
		score += 2
	}

	poiType := strings.ToLower(strings.TrimSpace(row.Type))
	poiClass := strings.ToLower(strings.TrimSpace(row.Class))
	if poiType == "house" || poiType == "building" || poiClass == "building" {
		score += 6
	}
	if poiClass == "boundary" {
		score -= 12
	}

	return score
}

func similarityScore(target string, candidate string, exact float64, partial float64, mismatch float64) float64 {
	t := normalizeGeoToken(target)
	c := normalizeGeoToken(candidate)
	if t == "" || c == "" {
		return 0
	}
	if t == c {
		return exact
	}
	if strings.Contains(c, t) || strings.Contains(t, c) {
		return partial
	}
	return mismatch
}

func houseMatchScore(targetHouse string, candidateHouse string, displayName string) float64 {
	t := normalizeHouseToken(targetHouse)
	if t == "" {
		return 0
	}
	c := normalizeHouseToken(candidateHouse)
	if c != "" {
		if c == t {
			return 36
		}
		if strings.HasPrefix(c, t) || strings.HasPrefix(t, c) {
			return 18
		}
		return -10
	}

	if strings.Contains(normalizeGeoToken(displayName), normalizeGeoToken(targetHouse)) {
		return 10
	}
	return -4
}

func normalizeGeoToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	v = strings.NewReplacer("’", "'", "`", "'", "ʼ", "'", "ё", "е", "ї", "і").Replace(v)

	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func normalizeHouseToken(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)\d+[0-9\p{L}/-]*`)
	token := strings.ToLower(strings.TrimSpace(re.FindString(v)))
	token = strings.ReplaceAll(token, " ", "")
	token = strings.ReplaceAll(token, "/", "")
	token = strings.ReplaceAll(token, "-", "")
	return token
}

func firstAddressValue(address map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(address[key]); v != "" {
			return v
		}
	}
	return ""
}

func buildGeocodeQueries(address string) []string {
	queries := make([]string, 0, 16)
	addQuery := func(v string) {
		v = normalizeAddressSpaces(v)
		if v == "" {
			return
		}
		for _, existing := range queries {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		queries = append(queries, v)
	}

	raw := normalizeAddressSpaces(address)
	cleaned := normalizeAddressForGeocode(raw)
	addQuery(raw)
	addQuery(cleaned)

	expanded := expandAddressAbbreviations(raw)
	expandedClean := expandAddressAbbreviations(cleaned)
	addQuery(expanded)
	addQuery(expandedClean)
	addQuery(ensureCountrySuffix(raw))
	addQuery(ensureCountrySuffix(cleaned))
	addQuery(ensureCountrySuffix(expanded))
	addQuery(ensureCountrySuffix(expandedClean))

	city, street, house, ok := parseAddressComponents(cleaned)
	if ok {
		if house != "" {
			addQuery(fmt.Sprintf("вулиця %s %s, %s, Україна", street, house, city))
			addQuery(fmt.Sprintf("%s %s, %s, Україна", street, house, city))
			addQuery(fmt.Sprintf("%s, вулиця %s, %s, Україна", city, street, house))
		}
		addQuery(fmt.Sprintf("вулиця %s, %s, Україна", street, city))
		addQuery(fmt.Sprintf("%s, %s, Україна", street, city))
		addQuery(fmt.Sprintf("%s, вулиця %s, Україна", city, street))
	}

	if cityOnly, ok := parseCityOnly(cleaned); ok {
		addQuery(fmt.Sprintf("%s, Україна", cityOnly))
	} else if streetOnly, houseOnly, ok := parseStreetAndHouseOnly(cleaned); ok {
		const defaultCity = "Львів"
		if houseOnly != "" {
			addQuery(fmt.Sprintf("вулиця %s %s, %s, Україна", streetOnly, houseOnly, defaultCity))
			addQuery(fmt.Sprintf("%s %s, %s, Україна", streetOnly, houseOnly, defaultCity))
		}
		addQuery(fmt.Sprintf("вулиця %s, %s, Україна", streetOnly, defaultCity))
		addQuery(fmt.Sprintf("%s, %s, Україна", streetOnly, defaultCity))
	}

	return queries
}

func parseAddressComponents(address string) (string, string, string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	parts := strings.Split(raw, ",")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = normalizeAddressSpaces(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return "", "", "", false
	}

	city := ""
	street := ""
	house := ""

	for _, p := range clean {
		if city == "" || street == "" {
			combinedCity, combinedStreet, combinedHouse, ok := splitCombinedLocalityStreetPart(p)
			if ok {
				if city == "" {
					city = combinedCity
				}
				if street == "" {
					street = combinedStreet
				}
				if house == "" && combinedHouse != "" {
					house = combinedHouse
				}
			}
		}

		if city == "" {
			if v, ok := extractCity(p); ok {
				city = v
				continue
			}
		}
		if street == "" {
			if v, ok := extractStreet(p); ok {
				street = v
			}
		}
		if house == "" {
			house = extractHouseNumber(p)
		}
	}

	if city == "" {
		for _, p := range clean {
			if extractHouseNumber(p) != "" {
				continue
			}
			if _, ok := extractStreet(p); ok {
				continue
			}
			if !isAdministrativePart(p) {
				city = normalizeAddressSpaces(p)
				break
			}
		}
	}
	if street == "" {
		for _, p := range clean {
			if extractHouseNumber(p) != "" {
				continue
			}
			if isAdministrativePart(p) {
				continue
			}
			street = normalizeStreetName(p)
			if street != "" {
				break
			}
		}
	}
	if house == "" {
		if h := extractHouseNumber(raw); h != "" {
			house = h
		}
	}

	if city == "" || street == "" {
		return "", "", "", false
	}
	return city, street, house, true
}

func parseCityOnly(address string) (string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		p = normalizeAddressSpaces(p)
		if p == "" {
			continue
		}
		if city, ok := extractCity(p); ok {
			return city, true
		}
	}
	return "", false
}

func parseStreetAndHouseOnly(address string) (string, string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	house := extractHouseNumber(raw)
	if house == "" {
		return "", "", false
	}
	street := normalizeStreetName(raw)
	if street == "" || isAdministrativePart(street) {
		return "", "", false
	}
	return street, house, true
}

func normalizeAddressSpaces(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func normalizeAddressForGeocode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "\"'`")
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}

	v = strings.NewReplacer(
		"Львіська", "Львівська",
		"Червоноі", "Червоної",
		"Мечнікова", "Мечникова",
		"буд.", " ",
		"буд ", " ",
	).Replace(v)

	// Склеюємо випадки типу "Незалежност, і".
	letterComma := regexp.MustCompile(`([\p{L}]{2,})\s*,\s*([\p{L}])\b`)
	for i := 0; i < 3; i++ {
		nv := letterComma.ReplaceAllString(v, "$1$2")
		if nv == v {
			break
		}
		v = nv
	}

	// Вирізаємо дужки та службові примітки.
	v = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(v, " ")

	// Прибираємо телефони та поштові індекси.
	v = regexp.MustCompile(`\b\d{2,4}[- ]\d{2}[- ]\d{2}(?:[- ]\d{2})?\b`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\b\d{5}\b`).ReplaceAllString(v, " ")

	// Зайві службові хвости.
	if idx := indexOfAddressNoise(v); idx > 0 {
		v = v[:idx]
	}

	if idx := strings.Index(v, "+"); idx > 0 {
		v = v[:idx]
	}

	// Нормалізуємо розділювачі.
	v = strings.ReplaceAll(v, ";", ", ")
	v = strings.ReplaceAll(v, "|", ", ")
	v = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(v, ", ")
	v = regexp.MustCompile(`\s*\.\s*`).ReplaceAllString(v, ". ")
	v = strings.Trim(v, " ,.-")
	return normalizeAddressSpaces(v)
}

func indexOfAddressNoise(v string) int {
	lower := strings.ToLower(v)
	keywords := []string{
		" ю/а ",
		" фактична",
		" завгосп",
		" централь",
		" охор",
		" режим роботи",
		" пожеж",
		" вхід ",
		" у дворі",
		" напроти ",
		" біля ",
		" на територ",
		" терітор",
		" корпус",
	}
	best := -1
	for _, kw := range keywords {
		idx := strings.Index(lower, kw)
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
}

func expandAddressAbbreviations(v string) string {
	v = normalizeAddressSpaces(v)
	abbrRules := []struct {
		pattern string
		repl    string
	}{
		{pattern: `(?i)(^|[\s,])м\.\s*`, repl: "${1}місто "},
		{pattern: `(?i)(^|[\s,])с\.\s*`, repl: "${1}село "},
		{pattern: `(?i)(^|[\s,])в\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])вуп\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])смт\.\s*`, repl: "${1}смт "},
		{pattern: `(?i)(^|[\s,])обл\.\s*`, repl: "${1}область "},
		{pattern: `(?i)(^|[\s,])вул\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])пр\.\s*`, repl: "${1}проспект "},
		{pattern: `(?i)(^|[\s,])просп\.\s*`, repl: "${1}проспект "},
		{pattern: `(?i)(^|[\s,])пл\.\s*`, repl: "${1}площа "},
		{pattern: `(?i)(^|[\s,])бул\.\s*`, repl: "${1}бульвар "},
		{pattern: `(?i)(^|[\s,])пров\.\s*`, repl: "${1}провулок "},
	}
	for _, rule := range abbrRules {
		re := regexp.MustCompile(rule.pattern)
		v = re.ReplaceAllString(v, rule.repl)
	}

	repl := strings.NewReplacer(
		"м.", "місто ",
		"м ", "місто ",
		"с.", "село ",
		"с ", "село ",
		"в.", "вулиця ",
		"обл.", "область ",
		"обл ", "область ",
		"смт.", "смт ",
		"вул.", "вулиця ",
		"вул ", "вулиця ",
		"пр-т.", "проспект ",
		"пр-т", "проспект ",
		"просп.", "проспект ",
		"пл.", "площа ",
		"бул.", "бульвар ",
		"пров.", "провулок ",
	)
	v = repl.Replace(v)
	return normalizeAddressSpaces(v)
}

func ensureCountrySuffix(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "україн") || strings.Contains(lower, "ukraine") {
		return v
	}
	return v + ", Україна"
}

func extractCity(v string) (string, bool) {
	v = strings.NewReplacer(
		"смт.", "смт ",
		"м.", "місто ",
		"місто.", "місто ",
		"село.", "село ",
		"с.", "село ",
	).Replace(v)
	v = normalizeAddressSpaces(v)
	lower := strings.ToLower(v)
	switch {
	case strings.HasPrefix(lower, "місто "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("місто "):])), true
	case strings.HasPrefix(lower, "смт "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("смт "):])), true
	case strings.HasPrefix(lower, "селище "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("селище "):])), true
	case strings.HasPrefix(lower, "село "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("село "):])), true
	}

	// Підтримка рядків типу:
	// "Львівська область місто Львів" / "Львівська область село Зимна Вода".
	for _, marker := range []string{" місто ", " смт ", " селище ", " село "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			candidate := normalizeAddressSpaces(strings.TrimSpace(v[idx+len(marker):]))
			if candidate != "" {
				return candidate, true
			}
			before := normalizeAddressSpaces(strings.TrimSpace(v[:idx]))
			if before != "" && !isAdministrativePart(before) {
				return before, true
			}
		}
	}

	// Підтримка "Яворів м." / "Яворів місто".
	for _, marker := range []string{" місто", " м"} {
		if strings.HasSuffix(lower, marker) {
			before := normalizeAddressSpaces(strings.TrimSpace(v[:len(v)-len(marker)]))
			if before != "" && !isAdministrativePart(before) {
				return before, true
			}
		}
	}
	return "", false
}

func extractStreet(v string) (string, bool) {
	v = normalizeAddressSpaces(v)
	lower := strings.ToLower(v)
	prefixes := []string{
		"вулиця ",
		"проспект ",
		"бульвар ",
		"площа ",
		"провулок ",
		"шосе ",
		"узвіз ",
	}
	for _, pref := range prefixes {
		if strings.HasPrefix(lower, pref) {
			street := normalizeStreetName(strings.TrimSpace(v[len(pref):]))
			if street != "" {
				return street, true
			}
		}
	}

	// Підтримка "Маковея вулиця" / "Червоної Калини проспект".
	for _, pref := range prefixes {
		suffix := strings.TrimSpace(pref)
		if strings.HasSuffix(lower, " "+suffix) {
			street := normalizeStreetName(strings.TrimSpace(v[:len(v)-len(suffix)]))
			if street != "" {
				return street, true
			}
		}
	}
	return "", false
}

func splitCombinedLocalityStreetPart(part string) (string, string, string, bool) {
	part = normalizeAddressSpaces(part)
	if part == "" {
		return "", "", "", false
	}

	lower := strings.ToLower(part)
	streetPrefixes := []string{
		"вулиця ",
		"проспект ",
		"бульвар ",
		"площа ",
		"провулок ",
		"шосе ",
		"узвіз ",
	}

	streetIdx := -1
	for _, pref := range streetPrefixes {
		if strings.HasPrefix(lower, pref) {
			streetIdx = 0
			break
		}
		if idx := strings.Index(lower, " "+pref); idx >= 0 {
			idx++
			if streetIdx < 0 || idx < streetIdx {
				streetIdx = idx
			}
		}
	}
	if streetIdx <= 0 {
		return "", "", "", false
	}

	localityPart := normalizeAddressSpaces(part[:streetIdx])
	streetPart := normalizeAddressSpaces(part[streetIdx:])
	if localityPart == "" || streetPart == "" {
		return "", "", "", false
	}

	city, ok := extractCity(localityPart)
	if !ok {
		return "", "", "", false
	}

	street, ok := extractStreet(streetPart)
	if !ok {
		return "", "", "", false
	}
	house := extractHouseNumber(streetPart)
	return city, street, house, true
}

func normalizeStreetName(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	// Якщо номер будинку написали разом зі вулицею, відкидаємо номер.
	house := extractHouseNumber(v)
	if house != "" {
		v = strings.TrimSpace(strings.Replace(v, house, "", 1))
	}
	return normalizeAddressSpaces(v)
}

func extractHouseNumber(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)\d+[0-9\p{L}/-]*`)
	return normalizeAddressSpaces(re.FindString(v))
}

func isAdministrativePart(v string) bool {
	l := strings.ToLower(normalizeAddressSpaces(v))
	if l == "" {
		return true
	}
	adminWords := []string{
		"район",
		"область",
		"громада",
		"україна",
		"украина",
		"ukraine",
	}
	for _, w := range adminWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

func collectDistrictHints(address map[string]string, displayName string) []string {
	hints := make([]string, 0, 8)
	addHint := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existing := range hints {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		hints = append(hints, v)
	}

	keys := []string{
		"city_district",
		"district",
		"suburb",
		"borough",
		"county",
		"state_district",
		"municipality",
	}
	for _, key := range keys {
		addHint(address[key])
	}

	parts := strings.Split(displayName, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(strings.ToLower(part), "район") {
			addHint(part)
		}
	}

	return hints
}

func resolveRegionByAddressHints(provider contracts.DistrictReferenceService, hints []string) (int64, string, error) {
	if len(hints) == 0 {
		return 0, "", fmt.Errorf("геосервіс не повернув район")
	}
	regions, err := provider.ListObjectDistricts()
	if err != nil {
		return 0, "", fmt.Errorf("не вдалося завантажити райони: %w", err)
	}
	if len(regions) == 0 {
		return 0, "", fmt.Errorf("довідник районів порожній")
	}

	type regionCandidate struct {
		ID   int64
		Name string
		Norm string
	}
	candidates := make([]regionCandidate, 0, len(regions))
	for _, region := range regions {
		name := strings.TrimSpace(region.Name)
		if name == "" || region.ID <= 0 {
			continue
		}
		candidates = append(candidates, regionCandidate{
			ID:   region.ID,
			Name: name,
			Norm: normalizeDistrictName(name),
		})
	}
	if len(candidates) == 0 {
		return 0, "", fmt.Errorf("не знайдено валідних районів у довіднику")
	}

	hintNorms := make([]string, 0, len(hints))
	for _, hint := range hints {
		if norm := normalizeDistrictName(hint); norm != "" {
			hintNorms = append(hintNorms, norm)
		}
	}
	if len(hintNorms) == 0 {
		return 0, "", fmt.Errorf("не вдалося витягнути назву району з адреси")
	}

	for _, hintNorm := range hintNorms {
		for _, c := range candidates {
			if c.Norm != "" && c.Norm == hintNorm {
				return c.ID, c.Name, nil
			}
		}
	}
	for _, hintNorm := range hintNorms {
		for _, c := range candidates {
			if c.Norm == "" {
				continue
			}
			if strings.Contains(hintNorm, c.Norm) || strings.Contains(c.Norm, hintNorm) {
				return c.ID, c.Name, nil
			}
		}
	}

	return 0, "", fmt.Errorf("район не зіставлено з довідником: %s", strings.Join(hints, ", "))
}

func normalizeDistrictName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"’", "'",
		"`", "'",
		"ʼ", "'",
		".", " ",
		",", " ",
		"(", " ",
		")", " ",
		"-", " ",
		"/", " ",
	)
	s = replacer.Replace(s)
	s = strings.ReplaceAll(s, "р-н", "район")

	tokens := strings.Fields(s)
	filtered := make([]string, 0, len(tokens))
	stopWords := map[string]struct{}{
		"район":   {},
		"місто":   {},
		"м":       {},
		"область": {},
		"обл":     {},
		"city":    {},
	}
	for _, t := range tokens {
		if _, skip := stopWords[t]; skip {
			continue
		}
		filtered = append(filtered, t)
	}
	return strings.Join(filtered, " ")
}

func showObjectPersonalEditor(
	parent fyne.Window,
	provider contracts.AdminObjectPersonalService,
	title string,
	initial contracts.AdminObjectPersonal,
	onSave func(item contracts.AdminObjectPersonal) error,
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

	applyPersonalLookup := func(found *contracts.AdminObjectPersonal) {
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

		item := contracts.AdminObjectPersonal{
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
	initial contracts.AdminObjectZone,
	onSave func(zone contracts.AdminObjectZone) error,
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

		zone := contracts.AdminObjectZone{
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

func suggestZoneFillCount(provider contracts.AdminObjectZonesTabProvider, objn int64, current []contracts.AdminObjectZone) int64 {
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

type mapInteractionSurface struct {
	widget.BaseWidget

	onTapped          func(*fyne.PointEvent)
	onTappedSecondary func(*fyne.PointEvent)
	onDragged         func(*fyne.DragEvent)
	onDragEnd         func()
	onScrolled        func(*fyne.ScrollEvent)
}

func newMapInteractionSurface() *mapInteractionSurface {
	surface := &mapInteractionSurface{}
	surface.ExtendBaseWidget(surface)
	return surface
}

func (s *mapInteractionSurface) Tapped(ev *fyne.PointEvent) {
	if s.onTapped != nil {
		s.onTapped(ev)
	}
}

func (s *mapInteractionSurface) TappedSecondary(ev *fyne.PointEvent) {
	if s.onTappedSecondary != nil {
		s.onTappedSecondary(ev)
	}
}

func (s *mapInteractionSurface) Dragged(ev *fyne.DragEvent) {
	if s.onDragged != nil {
		s.onDragged(ev)
	}
}

func (s *mapInteractionSurface) DragEnd() {
	if s.onDragEnd != nil {
		s.onDragEnd()
	}
}

func (s *mapInteractionSurface) Scrolled(ev *fyne.ScrollEvent) {
	if s.onScrolled != nil {
		s.onScrolled(ev)
	}
}

func (s *mapInteractionSurface) CreateRenderer() fyne.WidgetRenderer {
	// Мінімальна прозорість, щоб поверхня гарантовано брала pointer-події.
	hitBox := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 1})
	return widget.NewSimpleRenderer(hitBox)
}

type coordinatesMapPickerOptions struct {
	Title           string
	InitialAddress  string
	ForceLvivCenter bool
}

func showCoordinatesMapPicker(parent fyne.Window, initialLatRaw string, initialLonRaw string, onPick func(lat, lon string)) {
	showCoordinatesMapPickerWithOptions(parent, initialLatRaw, initialLonRaw, coordinatesMapPickerOptions{}, onPick)
}

func showCoordinatesMapPickerWithOptions(parent fyne.Window, initialLatRaw string, initialLonRaw string, opts coordinatesMapPickerOptions, onPick func(lat, lon string)) {
	centerLat, centerLon, zoom, hasObjectMarker := resolveInitialMapCenterWithOptions(initialLatRaw, initialLonRaw, opts.ForceLvivCenter)

	mapView := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.AtZoomLevel(zoom),
		xwidget.AtLatLon(centerLat, centerLon),
	)

	previousMarker := canvas.NewCircle(color.NRGBA{R: 255, G: 40, B: 40, A: 210})
	previousMarker.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 230}
	previousMarker.StrokeWidth = 2
	previousMarker.Resize(fyne.NewSize(12, 12))
	previousMarker.Hide()

	selectedMarker := canvas.NewCircle(color.NRGBA{R: 25, G: 122, B: 255, A: 210})
	selectedMarker.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 230}
	selectedMarker.StrokeWidth = 2
	selectedMarker.Resize(fyne.NewSize(16, 16))
	selectedMarker.Hide()

	selectedHalo := canvas.NewCircle(color.NRGBA{R: 25, G: 122, B: 255, A: 70})
	selectedHalo.Resize(fyne.NewSize(28, 28))
	selectedHalo.Hide()

	markerLayer := container.NewWithoutLayout(previousMarker, selectedHalo, selectedMarker)
	interaction := newMapInteractionSurface()

	mapStack := container.NewStack(
		mapView,
		markerLayer,
		interaction,
	)

	selectionLat := centerLat
	selectionLon := centerLon
	if lat, lon, ok := parseLatLon(initialLatRaw, initialLonRaw); ok {
		selectionLat = lat
		selectionLon = lon
	}

	centerLabel := widget.NewLabel("Центр: —")
	selectedLabel := widget.NewLabel("")
	selectedLabel.TextStyle = fyne.TextStyle{Bold: true}

	updateCenterLabel := func() {
		lat, lon, err := mapCenterLatLon(mapView)
		if err != nil {
			centerLabel.SetText("Центр: невизначено")
			return
		}
		zoomText := "?"
		if state, stateErr := readMapInternalState(mapView); stateErr == nil {
			zoomText = strconv.Itoa(state.zoom)
		}
		centerLabel.SetText(fmt.Sprintf("Центр: %s, %s | Z=%s", formatCoordinate(lat), formatCoordinate(lon), zoomText))
	}
	updateSelectedLabel := func() {
		selectedLabel.SetText(fmt.Sprintf("Вибрана точка: %s, %s", formatCoordinate(selectionLat), formatCoordinate(selectionLon)))
	}

	var updateMarkers func()
	lastMarkerUpdate := time.Time{}
	lastCenterUpdate := time.Time{}
	forceMapOverlayRefresh := func() {
		updateCenterLabel()
		if updateMarkers != nil {
			updateMarkers()
		}
		now := time.Now()
		lastMarkerUpdate = now
		lastCenterUpdate = now
	}
	updateMapOverlayDuringDrag := func() {
		now := time.Now()
		if updateMarkers != nil && now.Sub(lastMarkerUpdate) >= 80*time.Millisecond {
			updateMarkers()
			lastMarkerUpdate = now
		}
		if now.Sub(lastCenterUpdate) >= 220*time.Millisecond {
			updateCenterLabel()
			lastCenterUpdate = now
		}
	}

	objectMarkerLat := centerLat
	objectMarkerLon := centerLon
	updateMarkers = func() {
		if !hasObjectMarker {
			previousMarker.Hide()
		} else {
			x, y, ok := mapLatLonToCanvasPoint(mapView, objectMarkerLat, objectMarkerLon)
			if !ok {
				previousMarker.Hide()
			} else {
				size := mapView.Size()
				if x < -20 || y < -20 || x > size.Width+20 || y > size.Height+20 {
					previousMarker.Hide()
				} else {
					prevSize := previousMarker.Size()
					previousMarker.Move(fyne.NewPos(x-prevSize.Width/2, y-prevSize.Height/2))
					previousMarker.Show()
					previousMarker.Refresh()
				}
			}
		}

		sx, sy, ok := mapLatLonToCanvasPoint(mapView, selectionLat, selectionLon)
		if !ok {
			selectedMarker.Hide()
			selectedHalo.Hide()
			return
		}
		size := mapView.Size()
		if sx < -30 || sy < -30 || sx > size.Width+30 || sy > size.Height+30 {
			selectedMarker.Hide()
			selectedHalo.Hide()
			return
		}

		haloSize := selectedHalo.Size()
		selSize := selectedMarker.Size()
		selectedHalo.Move(fyne.NewPos(sx-haloSize.Width/2, sy-haloSize.Height/2))
		selectedMarker.Move(fyne.NewPos(sx-selSize.Width/2, sy-selSize.Height/2))
		selectedHalo.Show()
		selectedMarker.Show()
		selectedHalo.Refresh()
		selectedMarker.Refresh()
	}

	setSelectionAt := func(lat, lon float64) {
		selectionLat = lat
		selectionLon = lon
		updateSelectedLabel()
		updateMarkers()
	}
	updateSelectedLabel()
	forceMapOverlayRefresh()

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Вибір координат на карті"
	}
	pickerWin := fyne.CurrentApp().NewWindow(title)
	pickerWin.Resize(fyne.NewSize(980, 680))

	searchEntry := widget.NewSelectEntry(nil)
	searchEntry.SetPlaceHolder("Пошук адреси")
	searchEntry.SetText(strings.TrimSpace(opts.InitialAddress))
	searchStatusLabel := widget.NewLabel("")
	suggestionOptions := map[string]geocodeCandidate{}
	var suggestionMu sync.Mutex
	suggestionReqID := 0

	setSuggestionState := func(options []string, items map[string]geocodeCandidate) {
		suggestionMu.Lock()
		defer suggestionMu.Unlock()
		suggestionOptions = items
		searchEntry.SetOptions(options)
	}

	nextSuggestionRequestID := func() int {
		suggestionMu.Lock()
		defer suggestionMu.Unlock()
		suggestionReqID++
		return suggestionReqID
	}

	isCurrentSuggestionRequest := func(id int) bool {
		suggestionMu.Lock()
		defer suggestionMu.Unlock()
		return id == suggestionReqID
	}

	runAddressSearch := func() {
		address := strings.TrimSpace(searchEntry.Text)
		if address == "" {
			searchStatusLabel.SetText("Вкажіть адресу для пошуку")
			return
		}

		searchStatusLabel.SetText("Пошук адреси...")
		go func() {
			latRaw, lonRaw, _, err := geocodeAddress(address)
			fyne.Do(func() {
				if err != nil {
					searchStatusLabel.SetText("Адресу не знайдено")
					dialog.ShowError(err, pickerWin)
					return
				}
				lat, latErr := parseCoordinate(latRaw)
				lon, lonErr := parseCoordinate(lonRaw)
				if latErr != nil || lonErr != nil {
					searchStatusLabel.SetText("Сервіс повернув некоректні координати")
					dialog.ShowError(fmt.Errorf("не вдалося розпізнати координати адреси"), pickerWin)
					return
				}
				setSelectionAt(lat, lon)
				mapView.PanToLatLon(lat, lon)
				forceMapOverlayRefresh()
				searchStatusLabel.SetText(fmt.Sprintf("Знайдено: %s, %s", formatCoordinate(lat), formatCoordinate(lon)))
			})
		}()
	}

	applySuggestion := func(candidate geocodeCandidate) {
		lat, latErr := parseCoordinate(candidate.Lat)
		lon, lonErr := parseCoordinate(candidate.Lon)
		if latErr != nil || lonErr != nil {
			searchStatusLabel.SetText("Підказка містить некоректні координати")
			return
		}
		setSelectionAt(lat, lon)
		mapView.PanToLatLon(lat, lon)
		forceMapOverlayRefresh()
		searchStatusLabel.SetText(fmt.Sprintf("Підказка: %s", firstNonEmpty(candidate.DisplayName, searchEntry.Text)))
	}

	searchEntry.OnSubmitted = func(string) {
		runAddressSearch()
	}
	searchEntry.OnChanged = func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			setSuggestionState(nil, map[string]geocodeCandidate{})
			searchStatusLabel.SetText("")
			return
		}

		suggestionMu.Lock()
		candidate, ok := suggestionOptions[value]
		suggestionMu.Unlock()
		if ok {
			applySuggestion(candidate)
			return
		}

		if len([]rune(value)) < 3 {
			setSuggestionState(nil, map[string]geocodeCandidate{})
			searchStatusLabel.SetText("Введіть щонайменше 3 символи для підказок")
			return
		}

		reqID := nextSuggestionRequestID()
		searchStatusLabel.SetText("Пошук підказок...")
		go func(query string, expectedReqID int) {
			time.Sleep(350 * time.Millisecond)
			if !isCurrentSuggestionRequest(expectedReqID) {
				return
			}

			rows, err := geocodeAutocompleteCandidates(query)
			fyne.Do(func() {
				if !isCurrentSuggestionRequest(expectedReqID) {
					return
				}
				if err != nil {
					setSuggestionState(nil, map[string]geocodeCandidate{})
					searchStatusLabel.SetText("Не вдалося завантажити підказки")
					return
				}

				options, items := geocodeSuggestionOptions(rows)
				setSuggestionState(options, items)
				if len(options) == 0 {
					searchStatusLabel.SetText("Підказки не знайдено")
					return
				}
				searchStatusLabel.SetText(fmt.Sprintf("Знайдено підказок: %d", len(options)))
			})
		}(value, reqID)
	}

	interaction.onTapped = func(ev *fyne.PointEvent) {
		lat, lon, err := mapCanvasPointToLatLon(mapView, ev.Position.X, ev.Position.Y)
		if err != nil {
			return
		}
		setSelectionAt(lat, lon)
	}
	interaction.onTappedSecondary = func(ev *fyne.PointEvent) {
		lat, lon, err := mapCanvasPointToLatLon(mapView, ev.Position.X, ev.Position.Y)
		if err != nil {
			return
		}
		setSelectionAt(lat, lon)
		mapView.PanToLatLon(lat, lon)
		forceMapOverlayRefresh()
	}
	interaction.onDragged = func(ev *fyne.DragEvent) {
		mapView.Dragged(ev)
		updateMapOverlayDuringDrag()
	}
	interaction.onDragEnd = func() {
		mapView.DragEnd()
		forceMapOverlayRefresh()
	}
	interaction.onScrolled = func(ev *fyne.ScrollEvent) {
		delta := ev.Scrolled.DY
		if math.Abs(float64(ev.Scrolled.DX)) > math.Abs(float64(delta)) {
			delta = ev.Scrolled.DX
		}
		steps := mapScrollStepCount(delta)
		if steps == 0 {
			return
		}

		centerLat, centerLon, centerErr := mapCenterLatLon(mapView)

		// Для desktop/Fyne: позитивний wheel delta = прокрутка вгору.
		// Вгору -> наближення, вниз -> віддалення.
		if delta > 0 {
			for i := 0; i < steps; i++ {
				mapView.ZoomIn()
			}
		} else {
			for i := 0; i < steps; i++ {
				mapView.ZoomOut()
			}
		}
		if centerErr == nil {
			// Тримаємо центр стабільним, щоб карта не "пливла" при zoom.
			mapView.PanToLatLon(centerLat, centerLon)
		}
		forceMapOverlayRefresh()
	}

	useSelectionBtn := widget.NewButton("Підтвердити вибір", func() {
		centerLat, centerLon, err := mapCenterLatLon(mapView)
		if err == nil {
			saveLastMapCenter(centerLat, centerLon)
		}
		if onPick != nil {
			onPick(formatCoordinate(selectionLat), formatCoordinate(selectionLon))
		}
		pickerWin.Close()
	})
	setFromCenterBtn := widget.NewButton("Точка = центр", func() {
		lat, lon, err := mapCenterLatLon(mapView)
		if err != nil {
			dialog.ShowError(err, pickerWin)
			return
		}
		setSelectionAt(lat, lon)
	})
	centerOnSelectionBtn := widget.NewButton("До вибраної точки", func() {
		mapView.PanToLatLon(selectionLat, selectionLon)
		forceMapOverlayRefresh()
	})
	zoomInBtn := widget.NewButton("＋", func() {
		mapView.ZoomIn()
		forceMapOverlayRefresh()
	})
	zoomOutBtn := widget.NewButton("－", func() {
		mapView.ZoomOut()
		forceMapOverlayRefresh()
	})
	refreshBtn := widget.NewButton("Оновити", func() {
		forceMapOverlayRefresh()
	})
	centerLvivBtn := widget.NewButton("Львів", func() {
		mapView.PanToLatLon(mapDefaultLvivLat, mapDefaultLvivLon)
		forceMapOverlayRefresh()
	})
	mapSettingsBtn := widget.NewButton("Налаштування карти", func() {
		showMapCenterSettingsDialog(pickerWin, func(lat, lon float64, zoom int) {
			mapView.Zoom(zoom)
			mapView.PanToLatLon(lat, lon)
			forceMapOverlayRefresh()
		})
	})
	cancelBtn := widget.NewButton("Скасувати", func() { pickerWin.Close() })

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("ЛКМ: вибір точки | ПКМ: вибір + центрування | Колесо: зум | Перетягування: панорама."),
			widget.NewLabel("Червоний маркер: поточна точка об'єкта. Синій маркер: точка, яку ви обрали."),
			container.NewBorder(nil, nil, nil, container.NewHBox(widget.NewButton("Знайти адресу", runAddressSearch), centerLvivBtn), searchEntry),
			searchStatusLabel,
			widget.NewSeparator(),
		),
		container.NewVBox(
			container.NewHBox(centerLabel, layout.NewSpacer(), selectedLabel),
			container.NewHBox(
				widget.NewLabel("Зум:"),
				zoomOutBtn,
				zoomInBtn,
				layout.NewSpacer(),
				mapSettingsBtn,
				refreshBtn,
				setFromCenterBtn,
				centerOnSelectionBtn,
				useSelectionBtn,
				cancelBtn,
			),
		),
		nil,
		nil,
		mapStack,
	)
	pickerWin.SetContent(content)

	forceMapOverlayRefresh()
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

func mapCanvasPointToLatLon(m *xwidget.Map, x float32, y float32) (float64, float64, error) {
	state, err := readMapInternalState(m)
	if err != nil {
		return 0, 0, err
	}
	px := float64(x) * state.scale
	py := float64(y) * state.scale
	xTile := state.mx + (px-state.midTileX-state.offsetX*state.scale)/state.tilePx
	yTile := state.my + (py-state.midTileY-state.offsetY*state.scale)/state.tilePx
	return tileXYToLatLon(xTile, yTile, state.n)
}

func mapScrollStepCount(deltaY float32) int {
	abs := math.Abs(float64(deltaY))
	if abs < 0.05 {
		return 0
	}
	// Робимо зум плавним: один рівень за одну подію прокрутки.
	return 1
}

type mapInternalState struct {
	mx, my             float64
	zoom               int
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
		zoom:     zoom,
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
	return resolveInitialMapCenterWithOptions(initialLatRaw, initialLonRaw, false)
}

func resolveInitialMapCenterWithOptions(initialLatRaw string, initialLonRaw string, forceLvivCenter bool) (float64, float64, int, bool) {
	if lat, lon, ok := parseLatLon(initialLatRaw, initialLonRaw); ok {
		return lat, lon, mapDefaultZoom, true
	}
	if forceLvivCenter {
		return mapDefaultLvivLat, mapDefaultLvivLon, mapDefaultZoom, false
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
		return mapDefaultKyivLat, mapDefaultKyivLon, mapDefaultZoom, false
	case mapCenterModeCustom:
		if prefs != nil {
			lat, latErr := parseCoordinate(prefs.String(mapCenterCustomLatKey))
			lon, lonErr := parseCoordinate(prefs.String(mapCenterCustomLonKey))
			if latErr == nil && lonErr == nil && lat >= -85 && lat <= 85 && lon >= -180 && lon <= 180 {
				return lat, lon, mapDefaultZoom, false
			}
		}
	case mapCenterModeLast:
		if prefs != nil {
			lat, latErr := parseCoordinate(prefs.String(mapCenterLastLatPrefKey))
			lon, lonErr := parseCoordinate(prefs.String(mapCenterLastLonPrefKey))
			if latErr == nil && lonErr == nil && lat >= -85 && lat <= 85 && lon >= -180 && lon <= 180 {
				return lat, lon, mapDefaultZoom, false
			}
		}
	}

	return mapDefaultLvivLat, mapDefaultLvivLon, mapDefaultZoom, false
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
