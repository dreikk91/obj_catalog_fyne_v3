//go:build qt

package qtui

import (
	"context"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/geocode"
	"obj_catalog_fyne_v3/pkg/models"
)

type operationalMapKind string

const (
	operationalMapObject operationalMapKind = "object"
	operationalMapAlarm  operationalMapKind = "alarm"
	operationalMapGroup  operationalMapKind = "group"
)

type operationalMapItem struct {
	Kind        operationalMapKind
	ObjectID    int
	Title       string
	Details     string
	Latitude    float64
	Longitude   float64
	MarkerColor color.RGBA
}

type operationalMapRenderResult struct {
	seq           int64
	pointCount    int
	body          []byte
	loadErr       error
	usingFallback bool
}

func ShowOperationalMapDialog(
	parent *qt.QWidget,
	objects []models.Object,
	alarms []models.Alarm,
	groups []contracts.FrontendResponseGroup,
) (int, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Оперативна карта")
	dialog.Resize(1200, 760)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	toolbar := qt.NewQHBoxLayout2()
	showObjects := qt.NewQCheckBox3("Об'єкти")
	showAlarms := qt.NewQCheckBox3("Тривоги")
	showGroups := qt.NewQCheckBox3("МГР")
	showAlarms.SetChecked(true)
	showGroups.SetChecked(true)
	status := qt.NewQLabel3("")
	status.SetStyleSheet("color: " + qtMutedTextColor + ";")
	retryButton := qt.NewQPushButton3("Повторити")
	retryButton.SetToolTip("Повторити завантаження тайлів OpenStreetMap")
	retryButton.QWidget.SetVisible(false)
	toolbar.AddWidget(showObjects.QWidget)
	toolbar.AddWidget(showAlarms.QWidget)
	toolbar.AddWidget(showGroups.QWidget)
	toolbar.AddStretch()
	toolbar.AddWidget(status.QWidget)
	toolbar.AddWidget(retryButton.QWidget)
	layout.AddLayout(toolbar.QLayout)

	list := qt.NewQListWidget2()
	list.SetMinimumWidth(320)
	mapLabel := qt.NewQLabel3("Підготовка карти...")
	mapLabel.SetAlignment(qt.AlignCenter)
	mapLabel.SetMinimumSize2(760, 560)
	mapLabel.SetWordWrap(true)
	mapLabel.SetStyleSheet("background: " + qtAltSurfaceColor + "; border: 1px solid " + qtBorderColor + "; color: " + qtMutedTextColor + ";")

	splitter := qt.NewQSplitter3(qt.Horizontal)
	splitter.AddWidget(list.QWidget)
	splitter.AddWidget(mapLabel.QWidget)
	splitter.SetSizes([]int{340, 820})
	layout.AddWidget(splitter.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	buttons.OnRejected(dialog.Reject)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	var (
		renderSeq    atomic.Int64
		closed       atomic.Bool
		items        []operationalMapItem
		selected     int
		renderCancel context.CancelFunc
	)
	renderResults := make(chan operationalMapRenderResult, 4)
	resultTimer := qt.NewQTimer()
	resultTimer.SetInterval(50)
	resultTimer.OnTimeout(func() {
		for {
			select {
			case result := <-renderResults:
				if closed.Load() || result.seq != renderSeq.Load() {
					continue
				}
				pixmap := qt.NewQPixmap()
				if len(result.body) == 0 || !pixmap.LoadFromData(&result.body[0], uint(len(result.body))) {
					status.SetText("Помилка формування карти")
					mapLabel.SetText("Не вдалося сформувати карту")
					if result.loadErr != nil {
						mapLabel.SetToolTip(strings.TrimSpace(result.loadErr.Error()))
					}
					retryButton.QWidget.SetVisible(true)
					continue
				}
				mapLabel.SetText("")
				mapLabel.SetPixmap(pixmap)
				if result.usingFallback {
					status.SetText(fmt.Sprintf("Точок: %d | схематична карта — OpenStreetMap недоступна", result.pointCount))
					mapLabel.SetToolTip(strings.TrimSpace(result.loadErr.Error()))
					retryButton.QWidget.SetVisible(true)
					continue
				}
				status.SetText(fmt.Sprintf("Точок: %d | червоні: тривоги | зелені: МГР | сині: об'єкти", result.pointCount))
			default:
				return
			}
		}
	})
	resultTimer.Start2()
	dialog.OnFinished(func(int) {
		closed.Store(true)
		renderSeq.Add(1)
		if renderCancel != nil {
			renderCancel()
		}
		resultTimer.Stop()
	})
	var render func()
	render = func() {
		if renderCancel != nil {
			renderCancel()
		}
		seq := renderSeq.Add(1)
		items = buildOperationalMapItems(objects, alarms, groups, showObjects.IsChecked(), showAlarms.IsChecked(), showGroups.IsChecked())
		list.Clear()
		retryButton.QWidget.SetVisible(false)
		mapLabel.SetToolTip("")
		mapLabel.SetPixmap(qt.NewQPixmap())
		for _, item := range items {
			text := item.Title
			if strings.TrimSpace(item.Details) != "" {
				text += " | " + strings.TrimSpace(item.Details)
			}
			list.AddItem(text)
		}
		if len(items) == 0 {
			status.SetText("Немає точок із координатами")
			mapLabel.SetText("Для вибраних категорій немає координат")
			return
		}
		status.SetText(fmt.Sprintf("Точок: %d | завантаження OpenStreetMap...", len(items)))
		mapLabel.SetText("Завантаження карти…")
		points := make([]geocode.MapPoint, 0, len(items))
		markers := make([]geocode.MapMarker, 0, len(items))
		for _, item := range items {
			point := geocode.MapPoint{Latitude: item.Latitude, Longitude: item.Longitude}
			points = append(points, point)
			markers = append(markers, geocode.MapMarker{MapPoint: point, Color: item.MarkerColor})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		renderCancel = cancel
		go func(ctx context.Context, cancel context.CancelFunc) {
			defer cancel()
			snapshot, loadErr := geocode.LoadMapSnapshotForPoints(ctx, points, 820, 620)
			usingFallback := false
			if loadErr != nil {
				fallback, fallbackErr := geocode.BuildFallbackMapSnapshotForPoints(points, 820, 620)
				if fallbackErr == nil {
					snapshot = fallback
					usingFallback = true
				} else {
					loadErr = fmt.Errorf("%v; резервна карта: %w", loadErr, fallbackErr)
				}
			}
			var body []byte
			if snapshot != nil {
				body = snapshot.PNGWithMarkers(markers)
			}
			result := operationalMapRenderResult{
				seq:           seq,
				pointCount:    len(points),
				body:          body,
				loadErr:       loadErr,
				usingFallback: usingFallback,
			}
			select {
			case renderResults <- result:
			default:
			}
		}(ctx, cancel)
	}

	retryButton.OnClicked(func() { render() })
	showObjects.OnToggled(func(bool) { render() })
	showAlarms.OnToggled(func(bool) { render() })
	showGroups.OnToggled(func(bool) { render() })
	list.OnItemDoubleClicked(func(item *qt.QListWidgetItem) {
		row := list.Row(item)
		if row < 0 || row >= len(items) || items[row].ObjectID <= 0 {
			return
		}
		selected = items[row].ObjectID
		dialog.Accept()
	})

	render()
	accepted := dialog.Exec() == int(qt.QDialog__Accepted) && selected > 0
	resultTimer.Delete()
	return selected, accepted
}

func buildOperationalMapItems(
	objects []models.Object,
	alarms []models.Alarm,
	groups []contracts.FrontendResponseGroup,
	includeObjects bool,
	includeAlarms bool,
	includeGroups bool,
) []operationalMapItem {
	objectsByID := make(map[int]models.Object, len(objects))
	objectsByNumber := make(map[string]models.Object, len(objects))
	for _, object := range objects {
		objectsByID[object.ID] = object
		objectsByNumber[objectDisplayNumberForMap(object)] = object
	}
	result := make([]operationalMapItem, 0, len(objects)+len(alarms)+len(groups))
	alarmObjects := make(map[int]struct{}, len(alarms))

	if includeAlarms {
		type alarmSummary struct {
			primary models.Alarm
			count   int
		}
		summaries := make(map[int]alarmSummary, len(alarms))
		order := make([]int, 0, len(alarms))
		for _, alarm := range alarms {
			object, ok := objectsByID[alarm.ObjectID]
			if !ok {
				continue
			}
			if _, _, valid := parseOperationalCoordinates(object.Latitude, object.Longitude); !valid {
				continue
			}
			summary, exists := summaries[object.ID]
			if !exists {
				order = append(order, object.ID)
				summary.primary = alarm
			} else if preferOperationalMapAlarm(alarm, summary.primary) {
				summary.primary = alarm
			}
			summary.count++
			summaries[object.ID] = summary
		}
		for _, objectID := range order {
			object := objectsByID[objectID]
			summary := summaries[objectID]
			lat, lon, _ := parseOperationalCoordinates(object.Latitude, object.Longitude)
			alarmObjects[object.ID] = struct{}{}
			details := summary.primary.GetTypeDisplay()
			if summary.count > 1 {
				details += fmt.Sprintf(" · %d активних подій", summary.count)
			}
			result = append(result, operationalMapItem{
				Kind: operationalMapAlarm, ObjectID: object.ID,
				Title:   "Тривога №" + summary.primary.GetObjectNumberDisplay() + " " + strings.TrimSpace(object.Name),
				Details: details, Latitude: lat, Longitude: lon,
				MarkerColor: color.RGBA{R: 198, G: 40, B: 40, A: 255},
			})
		}
	}
	if includeObjects {
		for _, object := range objects {
			if _, hasAlarm := alarmObjects[object.ID]; hasAlarm {
				continue
			}
			lat, lon, ok := parseOperationalCoordinates(object.Latitude, object.Longitude)
			if !ok {
				continue
			}
			result = append(result, operationalMapItem{
				Kind: operationalMapObject, ObjectID: object.ID,
				Title: "Об'єкт №" + objectDisplayNumberForMap(object), Details: strings.TrimSpace(object.Name),
				Latitude: lat, Longitude: lon,
				MarkerColor: color.RGBA{R: 69, G: 133, B: 188, A: 255},
			})
		}
	}
	if includeGroups {
		for _, group := range groups {
			lat, lon, ok := parseOperationalCoordinates(group.Latitude, group.Longitude)
			if !ok {
				continue
			}
			objectID := 0
			if object, found := objectsByNumber[strings.TrimSpace(group.ObjectNumber)]; found {
				objectID = object.ID
			}
			result = append(result, operationalMapItem{
				Kind:     operationalMapGroup,
				ObjectID: objectID,
				Title:    "МГР " + responseGroupDisplayName(group),
				Details:  responseGroupDisplayStatus(group),
				Latitude: lat, Longitude: lon,
				MarkerColor: color.RGBA{R: 61, G: 156, B: 59, A: 255},
			})
		}
	}
	return result
}

func preferOperationalMapAlarm(candidate models.Alarm, current models.Alarm) bool {
	candidatePriority := operationalMapAlarmPriority(candidate.VisualSeverityValue())
	currentPriority := operationalMapAlarmPriority(current.VisualSeverityValue())
	if candidatePriority != currentPriority {
		return candidatePriority > currentPriority
	}
	return candidate.Time.After(current.Time)
}

func operationalMapAlarmPriority(severity models.VisualSeverity) int {
	switch severity {
	case models.VisualSeverityCritical:
		return 4
	case models.VisualSeverityWarning:
		return 3
	case models.VisualSeverityInfo:
		return 2
	case models.VisualSeverityNormal:
		return 1
	default:
		return 0
	}
}

func parseOperationalCoordinates(latitude string, longitude string) (float64, float64, bool) {
	lat, errLat := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(latitude), ",", "."), 64)
	lon, errLon := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(longitude), ",", "."), 64)
	if errLat != nil || errLon != nil || lat < -85 || lat > 85 || lon < -180 || lon > 180 {
		return 0, 0, false
	}
	return lat, lon, true
}

func objectDisplayNumberForMap(object models.Object) string {
	if number := strings.TrimSpace(object.DisplayNumber); number != "" {
		return number
	}
	return strconv.Itoa(object.ID)
}
