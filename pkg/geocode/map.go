package geocode

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"sync"
)

const tileSize = 256

var mapTileCache = struct {
	sync.RWMutex
	tiles map[string]image.Image
}{tiles: make(map[string]image.Image)}

// MapSnapshot is a fixed OpenStreetMap view that can translate clicks to coordinates.
type MapSnapshot struct {
	base       image.Image
	width      int
	height     int
	zoom       int
	leftWorldX float64
	topWorldY  float64
}

// LoadMapSnapshot downloads and composes OpenStreetMap tiles around a coordinate.
func LoadMapSnapshot(ctx context.Context, latitude, longitude float64, width, height, zoom int) (*MapSnapshot, error) {
	if width <= 0 || height <= 0 || zoom < 1 || zoom > 19 {
		return nil, fmt.Errorf("некоректні параметри карти")
	}
	latitude = math.Max(-85.05112878, math.Min(85.05112878, latitude))
	worldSize := math.Ldexp(float64(tileSize), zoom)
	centerX := (longitude + 180) / 360 * worldSize
	sinLatitude := math.Sin(latitude * math.Pi / 180)
	centerY := (0.5 - math.Log((1+sinLatitude)/(1-sinLatitude))/(4*math.Pi)) * worldSize
	left := centerX - float64(width)/2
	top := centerY - float64(height)/2
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	firstTileX := int(math.Floor(left / tileSize))
	lastTileX := int(math.Floor((left + float64(width-1)) / tileSize))
	firstTileY := int(math.Floor(top / tileSize))
	lastTileY := int(math.Floor((top + float64(height-1)) / tileSize))
	tileCount := 1 << zoom

	for tileY := firstTileY; tileY <= lastTileY; tileY++ {
		if tileY < 0 || tileY >= tileCount {
			continue
		}
		for tileX := firstTileX; tileX <= lastTileX; tileX++ {
			wrappedX := ((tileX % tileCount) + tileCount) % tileCount
			tile, err := loadMapTile(ctx, zoom, wrappedX, tileY)
			if err != nil {
				return nil, err
			}
			offsetX := int(math.Round(float64(tileX*tileSize) - left))
			offsetY := int(math.Round(float64(tileY*tileSize) - top))
			draw.Draw(canvas, image.Rect(offsetX, offsetY, offsetX+tileSize, offsetY+tileSize), tile, image.Point{}, draw.Src)
		}
	}
	return &MapSnapshot{
		base: canvas, width: width, height: height, zoom: zoom,
		leftWorldX: left, topWorldY: top,
	}, nil
}

func loadMapTile(ctx context.Context, zoom, x, y int) (image.Image, error) {
	key := fmt.Sprintf("%d/%d/%d", zoom, x, y)
	mapTileCache.RLock()
	cached := mapTileCache.tiles[key]
	mapTileCache.RUnlock()
	if cached != nil {
		return cached, nil
	}
	url := "https://tile.openstreetmap.org/" + key + ".png"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("завантаження карти: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("сервер карти повернув %d", response.StatusCode)
	}
	tile, err := png.Decode(response.Body)
	if err != nil {
		return nil, fmt.Errorf("декодування карти: %w", err)
	}
	mapTileCache.Lock()
	if len(mapTileCache.tiles) >= 256 {
		clear(mapTileCache.tiles)
	}
	mapTileCache.tiles[key] = tile
	mapTileCache.Unlock()
	return tile, nil
}

// CoordinateAt translates a pixel in the snapshot to latitude and longitude.
func (snapshot *MapSnapshot) CoordinateAt(x, y int) (float64, float64) {
	worldSize := math.Ldexp(float64(tileSize), snapshot.zoom)
	worldX := snapshot.leftWorldX + float64(x)
	worldY := snapshot.topWorldY + float64(y)
	longitude := worldX/worldSize*360 - 180
	latitude := math.Atan(math.Sinh(math.Pi*(1-2*worldY/worldSize))) * 180 / math.Pi
	return latitude, longitude
}

// PNGWithMarker returns the snapshot PNG with a marker at the selected pixel.
func (snapshot *MapSnapshot) PNGWithMarker(x, y int) []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, snapshot.width, snapshot.height))
	draw.Draw(canvas, canvas.Bounds(), snapshot.base, image.Point{}, draw.Src)
	drawMarker(canvas, x, y)
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil
	}
	return output.Bytes()
}

func drawMarker(target *image.RGBA, centerX, centerY int) {
	red := color.RGBA{R: 220, G: 35, B: 35, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	rings := []struct {
		radius int
		color  color.RGBA
	}{{radius: 8, color: white}, {radius: 6, color: red}}
	for _, ring := range rings {
		radius, markerColor := ring.radius, ring.color
		for y := -radius; y <= radius; y++ {
			for x := -radius; x <= radius; x++ {
				distance := x*x + y*y
				if distance <= radius*radius && distance >= (radius-2)*(radius-2) {
					pointX, pointY := centerX+x, centerY+y
					if image.Pt(pointX, pointY).In(target.Bounds()) {
						target.SetRGBA(pointX, pointY, markerColor)
					}
				}
			}
		}
	}
	for y := 6; y <= 16; y++ {
		if image.Pt(centerX, centerY+y).In(target.Bounds()) {
			target.SetRGBA(centerX, centerY+y, red)
		}
	}
}
