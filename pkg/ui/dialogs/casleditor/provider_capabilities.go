package casleditor

import (
	"context"

	"fyne.io/fyne/v2"
)

type CASLGeoZoneOption struct {
	ID   int64
	Name string
}

type caslGeoZoneAccessProvider interface {
	ReadManagers(ctx context.Context, skip int, limit int) ([]map[string]any, error)
}

type CoordinatesPickerFunc func(parent fyne.Window, initialLatRaw string, initialLonRaw string, title string, initialAddress string, onPick func(lat, lon string))

var ShowObjectCoordinatesPicker CoordinatesPickerFunc
