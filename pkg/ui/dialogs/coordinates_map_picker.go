package dialogs

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
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

type coordinatesMapPickerState struct {
	opts   coordinatesMapPickerOptions
	onPick func(lat, lon string)

	win             fyne.Window
	mapView         *xwidget.Map
	previousMarker  *canvas.Circle
	selectedMarker  *canvas.Circle
	selectedHalo    *canvas.Circle
	interaction     *mapInteractionSurface
	mapStack        fyne.CanvasObject
	searchEntry     *widget.SelectEntry
	searchStatus    *widget.Label
	centerLabel     *widget.Label
	selectedLabel   *widget.Label
	selectionLat    float64
	selectionLon    float64
	objectMarkerLat float64
	objectMarkerLon float64
	hasObjectMarker bool

	suggestionOptions map[string]geocodeCandidate
	suggestionMu      sync.Mutex
	suggestionReqID   int
	lastMarkerUpdate  time.Time
	lastCenterUpdate  time.Time
}

func showCoordinatesMapPickerWithOptions(parent fyne.Window, initialLatRaw string, initialLonRaw string, opts coordinatesMapPickerOptions, onPick func(lat, lon string)) {
	state := newCoordinatesMapPickerState(initialLatRaw, initialLonRaw, opts, onPick)
	state.win.SetContent(state.buildContent())
	state.bindSearchHandlers()
	state.bindInteractionHandlers()
	state.forceMapOverlayRefresh()
	state.win.Show()
}

func newCoordinatesMapPickerState(
	initialLatRaw string,
	initialLonRaw string,
	opts coordinatesMapPickerOptions,
	onPick func(lat, lon string),
) *coordinatesMapPickerState {
	centerLat, centerLon, zoom, hasObjectMarker := resolveInitialMapCenterWithOptions(initialLatRaw, initialLonRaw, opts.ForceLvivCenter)

	mapView := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.AtZoomLevel(zoom),
		xwidget.AtLatLon(centerLat, centerLon),
	)

	state := &coordinatesMapPickerState{
		opts:              opts,
		onPick:            onPick,
		mapView:           mapView,
		previousMarker:    newMapPickerMarker(color.NRGBA{R: 255, G: 40, B: 40, A: 210}, 12),
		selectedMarker:    newMapPickerMarker(color.NRGBA{R: 25, G: 122, B: 255, A: 210}, 16),
		selectedHalo:      newMapPickerHalo(),
		interaction:       newMapInteractionSurface(),
		centerLabel:       widget.NewLabel("Центр: —"),
		selectedLabel:     widget.NewLabel(""),
		searchEntry:       widget.NewSelectEntry(nil),
		searchStatus:      widget.NewLabel(""),
		suggestionOptions: map[string]geocodeCandidate{},
		objectMarkerLat:   centerLat,
		objectMarkerLon:   centerLon,
		hasObjectMarker:   hasObjectMarker,
	}

	state.selectedLabel.TextStyle = fyne.TextStyle{Bold: true}
	state.searchEntry.SetPlaceHolder("Пошук адреси")
	state.searchEntry.SetText(strings.TrimSpace(opts.InitialAddress))

	state.selectionLat = centerLat
	state.selectionLon = centerLon
	if lat, lon, ok := parseLatLon(initialLatRaw, initialLonRaw); ok {
		state.selectionLat = lat
		state.selectionLon = lon
	}

	state.mapStack = container.NewStack(
		state.mapView,
		container.NewWithoutLayout(state.previousMarker, state.selectedHalo, state.selectedMarker),
		state.interaction,
	)

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Вибір координат на карті"
	}
	state.win = fyne.CurrentApp().NewWindow(title)
	state.win.Resize(fyne.NewSize(980, 680))
	state.updateSelectedLabel()

	return state
}

func newMapPickerMarker(fill color.NRGBA, size float32) *canvas.Circle {
	marker := canvas.NewCircle(fill)
	marker.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 230}
	marker.StrokeWidth = 2
	marker.Resize(fyne.NewSize(size, size))
	marker.Hide()
	return marker
}

func newMapPickerHalo() *canvas.Circle {
	halo := canvas.NewCircle(color.NRGBA{R: 25, G: 122, B: 255, A: 70})
	halo.Resize(fyne.NewSize(28, 28))
	halo.Hide()
	return halo
}

func (s *coordinatesMapPickerState) buildContent() fyne.CanvasObject {
	centerLvivBtn := widget.NewButton("Львів", func() {
		s.mapView.PanToLatLon(mapDefaultLvivLat, mapDefaultLvivLon)
		s.forceMapOverlayRefresh()
	})
	useSelectionBtn := widget.NewButton("Підтвердити вибір", s.confirmSelection)
	setFromCenterBtn := widget.NewButton("Точка = центр", s.setSelectionFromCenter)
	centerOnSelectionBtn := widget.NewButton("До вибраної точки", func() {
		s.mapView.PanToLatLon(s.selectionLat, s.selectionLon)
		s.forceMapOverlayRefresh()
	})
	zoomInBtn := widget.NewButton("＋", func() {
		s.mapView.ZoomIn()
		s.forceMapOverlayRefresh()
	})
	zoomOutBtn := widget.NewButton("－", func() {
		s.mapView.ZoomOut()
		s.forceMapOverlayRefresh()
	})
	refreshBtn := widget.NewButton("Оновити", s.forceMapOverlayRefresh)
	mapSettingsBtn := widget.NewButton("Налаштування карти", s.openMapSettings)
	cancelBtn := widget.NewButton("Скасувати", func() { s.win.Close() })

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabel("ЛКМ: вибір точки | ПКМ: вибір + центрування | Колесо: зум | Перетягування: панорама."),
			widget.NewLabel("Червоний маркер: поточна точка об'єкта. Синій маркер: точка, яку ви обрали."),
			container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(widget.NewButton("Знайти адресу", s.runAddressSearch), centerLvivBtn),
				s.searchEntry,
			),
			s.searchStatus,
			widget.NewSeparator(),
		),
		container.NewVBox(
			container.NewHBox(s.centerLabel, layout.NewSpacer(), s.selectedLabel),
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
		s.mapStack,
	)
}

func (s *coordinatesMapPickerState) bindSearchHandlers() {
	s.searchEntry.OnSubmitted = func(string) {
		s.runAddressSearch()
	}
	s.searchEntry.OnChanged = s.handleSearchChange
}

func (s *coordinatesMapPickerState) bindInteractionHandlers() {
	s.interaction.onTapped = func(ev *fyne.PointEvent) {
		lat, lon, err := mapCanvasPointToLatLon(s.mapView, ev.Position.X, ev.Position.Y)
		if err == nil {
			s.setSelectionAt(lat, lon)
		}
	}
	s.interaction.onTappedSecondary = func(ev *fyne.PointEvent) {
		lat, lon, err := mapCanvasPointToLatLon(s.mapView, ev.Position.X, ev.Position.Y)
		if err != nil {
			return
		}
		s.setSelectionAt(lat, lon)
		s.mapView.PanToLatLon(lat, lon)
		s.forceMapOverlayRefresh()
	}
	s.interaction.onDragged = func(ev *fyne.DragEvent) {
		s.mapView.Dragged(ev)
		s.updateMapOverlayDuringDrag()
	}
	s.interaction.onDragEnd = func() {
		s.mapView.DragEnd()
		s.forceMapOverlayRefresh()
	}
	s.interaction.onScrolled = func(ev *fyne.ScrollEvent) {
		s.handleScroll(ev)
	}
}

func (s *coordinatesMapPickerState) updateCenterLabel() {
	lat, lon, err := mapCenterLatLon(s.mapView)
	if err != nil {
		s.centerLabel.SetText("Центр: невизначено")
		return
	}

	zoomText := "?"
	if state, stateErr := readMapInternalState(s.mapView); stateErr == nil {
		zoomText = strconv.Itoa(state.zoom)
	}
	s.centerLabel.SetText(fmt.Sprintf("Центр: %s, %s | Z=%s", formatCoordinate(lat), formatCoordinate(lon), zoomText))
}

func (s *coordinatesMapPickerState) updateSelectedLabel() {
	s.selectedLabel.SetText(fmt.Sprintf("Вибрана точка: %s, %s", formatCoordinate(s.selectionLat), formatCoordinate(s.selectionLon)))
}

func (s *coordinatesMapPickerState) updateMarkers() {
	s.updateObjectMarker()
	s.updateSelectionMarker()
}

func (s *coordinatesMapPickerState) updateObjectMarker() {
	if !s.hasObjectMarker {
		s.previousMarker.Hide()
		return
	}

	x, y, ok := mapLatLonToCanvasPoint(s.mapView, s.objectMarkerLat, s.objectMarkerLon)
	if !ok || !pointWithinMapBounds(s.mapView, x, y, 20) {
		s.previousMarker.Hide()
		return
	}

	size := s.previousMarker.Size()
	s.previousMarker.Move(fyne.NewPos(x-size.Width/2, y-size.Height/2))
	s.previousMarker.Show()
	s.previousMarker.Refresh()
}

func (s *coordinatesMapPickerState) updateSelectionMarker() {
	x, y, ok := mapLatLonToCanvasPoint(s.mapView, s.selectionLat, s.selectionLon)
	if !ok || !pointWithinMapBounds(s.mapView, x, y, 30) {
		s.selectedMarker.Hide()
		s.selectedHalo.Hide()
		return
	}

	haloSize := s.selectedHalo.Size()
	markerSize := s.selectedMarker.Size()
	s.selectedHalo.Move(fyne.NewPos(x-haloSize.Width/2, y-haloSize.Height/2))
	s.selectedMarker.Move(fyne.NewPos(x-markerSize.Width/2, y-markerSize.Height/2))
	s.selectedHalo.Show()
	s.selectedMarker.Show()
	s.selectedHalo.Refresh()
	s.selectedMarker.Refresh()
}

func pointWithinMapBounds(mapView *xwidget.Map, x float32, y float32, padding float32) bool {
	size := mapView.Size()
	return x >= -padding && y >= -padding && x <= size.Width+padding && y <= size.Height+padding
}

func (s *coordinatesMapPickerState) setSelectionAt(lat, lon float64) {
	s.selectionLat = lat
	s.selectionLon = lon
	s.updateSelectedLabel()
	s.updateSelectionMarker()
}

func (s *coordinatesMapPickerState) forceMapOverlayRefresh() {
	s.updateCenterLabel()
	s.updateMarkers()
	now := time.Now()
	s.lastMarkerUpdate = now
	s.lastCenterUpdate = now
}

func (s *coordinatesMapPickerState) updateMapOverlayDuringDrag() {
	now := time.Now()
	if now.Sub(s.lastMarkerUpdate) >= 80*time.Millisecond {
		s.updateMarkers()
		s.lastMarkerUpdate = now
	}
	if now.Sub(s.lastCenterUpdate) >= 220*time.Millisecond {
		s.updateCenterLabel()
		s.lastCenterUpdate = now
	}
}

func (s *coordinatesMapPickerState) setSuggestionState(options []string, items map[string]geocodeCandidate) {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	s.suggestionOptions = items
	s.searchEntry.SetOptions(options)
}

func (s *coordinatesMapPickerState) nextSuggestionRequestID() int {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	s.suggestionReqID++
	return s.suggestionReqID
}

func (s *coordinatesMapPickerState) isCurrentSuggestionRequest(id int) bool {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	return id == s.suggestionReqID
}

func (s *coordinatesMapPickerState) suggestionForValue(value string) (geocodeCandidate, bool) {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	candidate, ok := s.suggestionOptions[value]
	return candidate, ok
}

func (s *coordinatesMapPickerState) runAddressSearch() {
	address := strings.TrimSpace(s.searchEntry.Text)
	if address == "" {
		s.searchStatus.SetText("Вкажіть адресу для пошуку")
		return
	}

	s.searchStatus.SetText("Пошук адреси...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		latRaw, lonRaw, _, err := geocodeAddressContext(ctx, address)
		fyne.Do(func() {
			if err != nil {
				s.searchStatus.SetText("Адресу не знайдено")
				dialog.ShowError(err, s.win)
				return
			}

			lat, latErr := parseCoordinate(latRaw)
			lon, lonErr := parseCoordinate(lonRaw)
			if latErr != nil || lonErr != nil {
				s.searchStatus.SetText("Сервіс повернув некоректні координати")
				dialog.ShowError(fmt.Errorf("не вдалося розпізнати координати адреси"), s.win)
				return
			}

			s.setSelectionAt(lat, lon)
			s.mapView.PanToLatLon(lat, lon)
			s.forceMapOverlayRefresh()
			s.searchStatus.SetText(fmt.Sprintf("Знайдено: %s, %s", formatCoordinate(lat), formatCoordinate(lon)))
		})
	}()
}

func (s *coordinatesMapPickerState) applySuggestion(candidate geocodeCandidate) {
	lat, latErr := parseCoordinate(candidate.Lat)
	lon, lonErr := parseCoordinate(candidate.Lon)
	if latErr != nil || lonErr != nil {
		s.searchStatus.SetText("Підказка містить некоректні координати")
		return
	}

	s.setSelectionAt(lat, lon)
	s.mapView.PanToLatLon(lat, lon)
	s.forceMapOverlayRefresh()
	s.searchStatus.SetText(fmt.Sprintf("Підказка: %s", firstNonEmpty(candidate.DisplayName, s.searchEntry.Text)))
}

func (s *coordinatesMapPickerState) handleSearchChange(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		s.setSuggestionState(nil, map[string]geocodeCandidate{})
		s.searchStatus.SetText("")
		return
	}

	if candidate, ok := s.suggestionForValue(value); ok {
		s.applySuggestion(candidate)
		return
	}

	if len([]rune(value)) < 3 {
		s.setSuggestionState(nil, map[string]geocodeCandidate{})
		s.searchStatus.SetText("Введіть щонайменше 3 символи для підказок")
		return
	}

	reqID := s.nextSuggestionRequestID()
	s.searchStatus.SetText("Пошук підказок...")
	go func(query string, expectedReqID int) {
		time.Sleep(350 * time.Millisecond)
		if !s.isCurrentSuggestionRequest(expectedReqID) {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		rows, err := geocodeAutocompleteCandidatesContext(ctx, query)
		fyne.Do(func() {
			if !s.isCurrentSuggestionRequest(expectedReqID) {
				return
			}
			if err != nil {
				s.setSuggestionState(nil, map[string]geocodeCandidate{})
				s.searchStatus.SetText("Не вдалося завантажити підказки")
				return
			}

			options, items := geocodeSuggestionOptions(rows)
			s.setSuggestionState(options, items)
			if len(options) == 0 {
				s.searchStatus.SetText("Підказки не знайдено")
				return
			}
			s.searchStatus.SetText(fmt.Sprintf("Знайдено підказок: %d", len(options)))
		})
	}(value, reqID)
}

func (s *coordinatesMapPickerState) handleScroll(ev *fyne.ScrollEvent) {
	delta := ev.Scrolled.DY
	if math.Abs(float64(ev.Scrolled.DX)) > math.Abs(float64(delta)) {
		delta = ev.Scrolled.DX
	}

	steps := mapScrollStepCount(delta)
	if steps == 0 {
		return
	}

	centerLat, centerLon, centerErr := mapCenterLatLon(s.mapView)
	if delta > 0 {
		for range steps {
			s.mapView.ZoomIn()
		}
	} else {
		for range steps {
			s.mapView.ZoomOut()
		}
	}
	if centerErr == nil {
		s.mapView.PanToLatLon(centerLat, centerLon)
	}
	s.forceMapOverlayRefresh()
}

func (s *coordinatesMapPickerState) confirmSelection() {
	centerLat, centerLon, err := mapCenterLatLon(s.mapView)
	if err == nil {
		saveLastMapCenter(centerLat, centerLon)
	}
	if s.onPick != nil {
		s.onPick(formatCoordinate(s.selectionLat), formatCoordinate(s.selectionLon))
	}
	s.win.Close()
}

func (s *coordinatesMapPickerState) setSelectionFromCenter() {
	lat, lon, err := mapCenterLatLon(s.mapView)
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.setSelectionAt(lat, lon)
}

func (s *coordinatesMapPickerState) openMapSettings() {
	showMapCenterSettingsDialog(s.win, func(lat, lon float64, zoom int) {
		s.mapView.Zoom(zoom)
		s.mapView.PanToLatLon(lat, lon)
		s.forceMapOverlayRefresh()
	})
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
