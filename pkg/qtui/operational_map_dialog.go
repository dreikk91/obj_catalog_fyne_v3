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
	status.SetStyleSheet("color: #555;")
	toolbar.AddWidget(showObjects.QWidget)
	toolbar.AddWidget(showAlarms.QWidget)
	toolbar.AddWidget(showGroups.QWidget)
	toolbar.AddStretch()
	toolbar.AddWidget(status.QWidget)
	layout.AddLayout(toolbar.QLayout)

	list := qt.NewQListWidget2()
	list.SetMinimumWidth(320)
	mapLabel := qt.NewQLabel3("Підготовка карти...")
	mapLabel.SetAlignment(qt.AlignCenter)
	mapLabel.SetMinimumSize2(760, 560)
	mapLabel.SetStyleSheet("background: #f5f5f5; border: 1px solid #cfcfcf;")

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
		renderSeq atomic.Int64
		closed    atomic.Bool
		items     []operationalMapItem
		selected  int
	)
	dialog.OnFinished(func(int) {
		closed.Store(true)
		renderSeq.Add(1)
	})
	render := func() {
		seq := renderSeq.Add(1)
		items = buildOperationalMapItems(objects, alarms, groups, showObjects.IsChecked(), showAlarms.IsChecked(), showGroups.IsChecked())
		list.Clear()
		for _, item := range items {
			text := item.Title
			if strings.TrimSpace(item.Details) != "" {
				text += " | " + strings.TrimSpace(item.Details)
			}
			list.AddItem(text)
		}
		if len(items) == 0 {
			status.SetText("Немає точок із координатами")
			mapLabel.SetPixmap(qt.NewQPixmap())
			mapLabel.SetText("Для вибраних категорій немає координат")
			return
		}
		status.SetText(fmt.Sprintf("Точок: %d | завантаження карти...", len(items)))
		points := make([]geocode.MapPoint, 0, len(items))
		markers := make([]geocode.MapMarker, 0, len(items))
		for _, item := range items {
			point := geocode.MapPoint{Latitude: item.Latitude, Longitude: item.Longitude}
			points = append(points, point)
			markers = append(markers, geocode.MapMarker{MapPoint: point, Color: item.MarkerColor})
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			snapshot, err := geocode.LoadMapSnapshotForPoints(ctx, points, 820, 620)
			var body []byte
			if err == nil {
				body = snapshot.PNGWithMarkers(markers)
			}
			runOnMainThread(func() {
				if closed.Load() || seq != renderSeq.Load() {
					return
				}
				if err != nil {
					status.SetText("Помилка карти")
					mapLabel.SetText(strings.TrimSpace(err.Error()))
					return
				}
				pixmap := qt.NewQPixmap()
				if len(body) == 0 || !pixmap.LoadFromData(&body[0], uint(len(body))) {
					mapLabel.SetText("Не вдалося сформувати карту")
					return
				}
				mapLabel.SetText("")
				mapLabel.SetPixmap(pixmap)
				status.SetText(fmt.Sprintf("Точок: %d | червоні: тривоги | зелені: МГР", len(items)))
			})
		}()
	}

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
		for _, alarm := range alarms {
			object, ok := objectsByID[alarm.ObjectID]
			if !ok {
				continue
			}
			lat, lon, ok := parseOperationalCoordinates(object.Latitude, object.Longitude)
			if !ok {
				continue
			}
			alarmObjects[object.ID] = struct{}{}
			result = append(result, operationalMapItem{
				Kind: operationalMapAlarm, ObjectID: object.ID,
				Title:   "Тривога №" + alarm.GetObjectNumberDisplay() + " " + strings.TrimSpace(object.Name),
				Details: alarm.GetTypeDisplay(), Latitude: lat, Longitude: lon,
				MarkerColor: color.RGBA{R: 205, G: 35, B: 35, A: 255},
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
				MarkerColor: color.RGBA{R: 35, G: 95, B: 180, A: 255},
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
				MarkerColor: color.RGBA{R: 25, G: 135, B: 70, A: 255},
			})
		}
	}
	return result
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
