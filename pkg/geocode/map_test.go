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

func TestMapSnapshotPixelAtRoundTrip(t *testing.T) {
	snapshot := &MapSnapshot{
		base:  image.NewRGBA(image.Rect(0, 0, 400, 300)),
		width: 400, height: 300, zoom: 12,
	}
	centerX, centerY := mapWorldPoint(49.8397, 24.0297, snapshot.zoom)
	snapshot.leftWorldX = centerX - 200
	snapshot.topWorldY = centerY - 150
	x, y := snapshot.PixelAt(49.8397, 24.0297)
	if x != 200 || y != 150 {
		t.Fatalf("center pixel = %d,%d", x, y)
	}
	body := snapshot.PNGWithMarkers([]MapMarker{{
		MapPoint: MapPoint{Latitude: 49.8397, Longitude: 24.0297},
	}})
	if len(body) == 0 {
		t.Fatal("PNGWithMarkers() returned no data")
	}
}

func TestMapViewportFitsNearbyPointsAtUsefulZoom(t *testing.T) {
	_, _, zoom := mapViewport([]MapPoint{
		{Latitude: 49.8397, Longitude: 24.0297},
		{Latitude: 49.85, Longitude: 24.05},
	}, 800, 600)
	if zoom < 10 || zoom > 16 {
		t.Fatalf("zoom = %d", zoom)
	}
}
