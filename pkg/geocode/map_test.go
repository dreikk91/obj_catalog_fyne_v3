package geocode

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

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

func TestBuildFallbackMapSnapshotForPoints(t *testing.T) {
	points := []MapPoint{
		{Latitude: 49.8397, Longitude: 24.0297},
		{Latitude: 49.85, Longitude: 24.05},
	}
	snapshot, err := BuildFallbackMapSnapshotForPoints(points, 800, 600)
	if err != nil {
		t.Fatalf("BuildFallbackMapSnapshotForPoints() error = %v", err)
	}
	body := snapshot.PNGWithMarkers([]MapMarker{
		{MapPoint: points[0], Color: color.RGBA{R: 198, G: 40, B: 40, A: 255}},
		{MapPoint: points[1], Color: color.RGBA{R: 61, G: 156, B: 59, A: 255}},
	})
	if len(body) == 0 {
		t.Fatal("fallback snapshot returned no PNG data")
	}
	if _, err := png.Decode(bytes.NewReader(body)); err != nil {
		t.Fatalf("fallback snapshot returned invalid PNG: %v", err)
	}
}

func TestLoadMapSnapshotDownloadsTilesWithBoundedConcurrency(t *testing.T) {
	var tile bytes.Buffer
	if err := png.Encode(&tile, image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))); err != nil {
		t.Fatalf("encode tile: %v", err)
	}
	tileBody := tile.Bytes()

	originalClient := httpClient
	var active atomic.Int32
	var maximum atomic.Int32
	httpClient = &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(tileBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	mapTileCache.Lock()
	clear(mapTileCache.tiles)
	mapTileCache.Unlock()
	t.Cleanup(func() {
		httpClient = originalClient
		mapTileCache.Lock()
		clear(mapTileCache.tiles)
		mapTileCache.Unlock()
	})

	if _, err := LoadMapSnapshot(context.Background(), 49.8397, 24.0297, 820, 620, 13); err != nil {
		t.Fatalf("LoadMapSnapshot() error = %v", err)
	}
	if got := maximum.Load(); got < 2 || got > mapTileDownloadLimit {
		t.Fatalf("maximum concurrent downloads = %d, want 2..%d", got, mapTileDownloadLimit)
	}
}
