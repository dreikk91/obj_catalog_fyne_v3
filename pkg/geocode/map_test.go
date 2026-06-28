package geocode

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"testing"
)

func TestMapSnapshotCoordinateAtCenter(t *testing.T) {
	snapshot := &MapSnapshot{
		base:  image.NewRGBA(image.Rect(0, 0, 100, 100)),
		width: 100, height: 100, zoom: 1,
		leftWorldX: 206, topWorldY: 206,
	}
	latitude, longitude := snapshot.CoordinateAt(50, 50)
	if math.Abs(latitude) > 1e-9 || math.Abs(longitude) > 1e-9 {
		t.Fatalf("center = %f, %f", latitude, longitude)
	}
	body := snapshot.PNGWithMarker(50, 50)
	if len(body) == 0 {
		t.Fatal("PNGWithMarker() returned no data")
	}
	if _, err := png.Decode(bytes.NewReader(body)); err != nil {
		t.Fatalf("PNGWithMarker() invalid PNG: %v", err)
	}
}
