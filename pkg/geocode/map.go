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

	"golang.org/x/sync/errgroup"
)

const (
	tileSize               = 256
	mapTileDownloadLimit   = 4
	fallbackMapGridSpacing = 64
)

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

// MapPoint is a geographic point used to fit an operational map.
type MapPoint struct {
	Latitude  float64
	Longitude float64
}

// MapMarker describes a colored marker rendered over a snapshot.
type MapMarker struct {
	MapPoint
	Color color.RGBA
}

// LoadMapSnapshotForPoints fits all valid points into one OpenStreetMap view.
func LoadMapSnapshotForPoints(ctx context.Context, points []MapPoint, width, height int) (*MapSnapshot, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("немає координат для карти")
	}
	centerLat, centerLon, zoom := mapViewport(points, width, height)
	return LoadMapSnapshot(ctx, centerLat, centerLon, width, height, zoom)
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

	type tilePlacement struct {
		requestX int
		requestY int
		tile     image.Image
		offsetX  int
		offsetY  int
	}
	placements := make([]tilePlacement, 0, (lastTileX-firstTileX+1)*(lastTileY-firstTileY+1))
	for tileY := firstTileY; tileY <= lastTileY; tileY++ {
		if tileY < 0 || tileY >= tileCount {
			continue
		}
		for tileX := firstTileX; tileX <= lastTileX; tileX++ {
			wrappedX := ((tileX % tileCount) + tileCount) % tileCount
			placements = append(placements, tilePlacement{
				requestX: wrappedX,
				requestY: tileY,
				offsetX:  int(math.Round(float64(tileX*tileSize) - left)),
				offsetY:  int(math.Round(float64(tileY*tileSize) - top)),
			})
		}
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(mapTileDownloadLimit)
	for index := range placements {
		placementIndex := index
		group.Go(func() error {
			placement := &placements[placementIndex]
			tile, err := loadMapTile(groupCtx, zoom, placement.requestX, placement.requestY)
			if err != nil {
				return err
			}
			placement.tile = tile
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	for _, placement := range placements {
		draw.Draw(
			canvas,
			image.Rect(placement.offsetX, placement.offsetY, placement.offsetX+tileSize, placement.offsetY+tileSize),
			placement.tile,
			image.Point{},
			draw.Src,
		)
	}
	return &MapSnapshot{
		base: canvas, width: width, height: height, zoom: zoom,
		leftWorldX: left, topWorldY: top,
	}, nil
}

// BuildFallbackMapSnapshotForPoints builds an offline schematic map preserving
// the relative geographic positions of all markers.
func BuildFallbackMapSnapshotForPoints(points []MapPoint, width, height int) (*MapSnapshot, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("немає координат для карти")
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("некоректні розміри карти")
	}
	centerLat, centerLon, zoom := mapViewport(points, width, height)
	worldSize := math.Ldexp(float64(tileSize), zoom)
	centerX := (centerLon + 180) / 360 * worldSize
	sinLatitude := math.Sin(centerLat * math.Pi / 180)
	centerY := (0.5 - math.Log((1+sinLatitude)/(1-sinLatitude))/(4*math.Pi)) * worldSize
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{R: 239, G: 243, B: 247, A: 255}), image.Point{}, draw.Src)

	gridColor := image.NewUniform(color.RGBA{R: 209, G: 220, B: 230, A: 255})
	for x := 0; x < width; x += fallbackMapGridSpacing {
		draw.Draw(canvas, image.Rect(x, 0, x+1, height), gridColor, image.Point{}, draw.Src)
	}
	for y := 0; y < height; y += fallbackMapGridSpacing {
		draw.Draw(canvas, image.Rect(0, y, width, y+1), gridColor, image.Point{}, draw.Src)
	}

	return &MapSnapshot{
		base:       canvas,
		width:      width,
		height:     height,
		zoom:       zoom,
		leftWorldX: centerX - float64(width)/2,
		topWorldY:  centerY - float64(height)/2,
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

// PixelAt translates coordinates to a pixel in the snapshot.
func (snapshot *MapSnapshot) PixelAt(latitude, longitude float64) (int, int) {
	worldSize := math.Ldexp(float64(tileSize), snapshot.zoom)
	latitude = math.Max(-85.05112878, math.Min(85.05112878, latitude))
	worldX := (longitude + 180) / 360 * worldSize
	sinLatitude := math.Sin(latitude * math.Pi / 180)
	worldY := (0.5 - math.Log((1+sinLatitude)/(1-sinLatitude))/(4*math.Pi)) * worldSize
	return int(math.Round(worldX - snapshot.leftWorldX)), int(math.Round(worldY - snapshot.topWorldY))
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

// PNGWithMarkers returns the snapshot with multiple colored markers.
func (snapshot *MapSnapshot) PNGWithMarkers(markers []MapMarker) []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, snapshot.width, snapshot.height))
	draw.Draw(canvas, canvas.Bounds(), snapshot.base, image.Point{}, draw.Src)
	for _, marker := range markers {
		x, y := snapshot.PixelAt(marker.Latitude, marker.Longitude)
		drawColoredMarker(canvas, x, y, marker.Color)
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil
	}
	return output.Bytes()
}

func drawMarker(target *image.RGBA, centerX, centerY int) {
	drawColoredMarker(target, centerX, centerY, color.RGBA{R: 220, G: 35, B: 35, A: 255})
}

func drawColoredMarker(target *image.RGBA, centerX, centerY int, markerColor color.RGBA) {
	if markerColor.A == 0 {
		markerColor.A = 255
	}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	rings := []struct {
		radius int
		color  color.RGBA
	}{{radius: 8, color: white}, {radius: 6, color: markerColor}}
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
			target.SetRGBA(centerX, centerY+y, markerColor)
		}
	}
}

func mapViewport(points []MapPoint, width, height int) (float64, float64, int) {
	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64
	for _, point := range points {
		x, y := mapWorldPoint(point.Latitude, point.Longitude, 0)
		minX = math.Min(minX, x)
		maxX = math.Max(maxX, x)
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	zoom := 16
	availableWidth := math.Max(1, float64(width-100))
	availableHeight := math.Max(1, float64(height-100))
	for zoom > 2 {
		scale := math.Ldexp(1, zoom)
		if (maxX-minX)*scale <= availableWidth && (maxY-minY)*scale <= availableHeight {
			break
		}
		zoom--
	}
	longitude := centerX/float64(tileSize)*360 - 180
	latitude := math.Atan(math.Sinh(math.Pi*(1-2*centerY/float64(tileSize)))) * 180 / math.Pi
	return latitude, longitude, zoom
}

func mapWorldPoint(latitude, longitude float64, zoom int) (float64, float64) {
	latitude = math.Max(-85.05112878, math.Min(85.05112878, latitude))
	worldSize := math.Ldexp(float64(tileSize), zoom)
	x := (longitude + 180) / 360 * worldSize
	sinLatitude := math.Sin(latitude * math.Pi / 180)
	y := (0.5 - math.Log((1+sinLatitude)/(1-sinLatitude))/(4*math.Pi)) * worldSize
	return x, y
}
