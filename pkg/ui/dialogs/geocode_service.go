package dialogs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

func geocodeAddress(address string) (string, string, []string, error) {
	return geocodeAddressContext(context.Background(), address)
}

func geocodeAddressContext(ctx context.Context, address string) (string, string, []string, error) {
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
		rows, err := geocodeCandidatesForQuery(ctx, query)
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

func geocodeCandidatesForQuery(ctx context.Context, query string) ([]geocodeCandidate, error) {
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
		if err := waitForGeocodeRequestSlot(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("не вдалося сформувати запит геопошуку: %w", err)
		}
		req.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")

		resp, err := geocodeHTTPClient.Do(req)
		if err != nil {
			if attempt < geocodeMaxRetry429 {
				if err := sleepWithContext(ctx, time.Duration(attempt+1)*geocodeRetryBackoff); err != nil {
					return nil, err
				}
				continue
			}
			phRows, phErr := geocodeCandidatesPhoton(ctx, query)
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
				if err := sleepWithContext(ctx, time.Duration(attempt+1)*geocodeRetryBackoff); err != nil {
					return nil, err
				}
				continue
			}
			phRows, phErr := geocodeCandidatesPhoton(ctx, query)
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
			phRows, phErr := geocodeCandidatesPhoton(ctx, query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
		}
		return rows, nil
	}

	phRows, phErr := geocodeCandidatesPhoton(ctx, query)
	if phErr == nil && len(phRows) > 0 {
		return phRows, nil
	}
	if phErr != nil {
		return nil, fmt.Errorf("геосервіс недоступний, fallback photon помилка: %v", phErr)
	}
	return nil, fmt.Errorf("геосервіс недоступний")
}

func waitForGeocodeRequestSlot(ctx context.Context) error {
	geocodeRequestMu.Lock()
	defer geocodeRequestMu.Unlock()

	if !geocodeLastRequest.IsZero() {
		wait := geocodeMinInterval - time.Since(geocodeLastRequest)
		if wait > 0 {
			if err := sleepWithContext(ctx, wait); err != nil {
				return err
			}
		}
	}
	geocodeLastRequest = time.Now()
	return nil
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func geocodeCandidatesPhoton(ctx context.Context, query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("lang", "uk")
	params.Set("limit", "8")
	photonURL := "https://photon.komoot.io/api/?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, photonURL, nil)
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
	return geocodeAutocompleteCandidatesContext(context.Background(), query)
}

func geocodeAutocompleteCandidatesContext(ctx context.Context, query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	rows, err := geocodeCandidatesPhoton(ctx, query)
	if err == nil && len(rows) > 0 {
		return rows, nil
	}
	if err == nil {
		return geocodeCandidatesForQuery(ctx, query)
	}

	fallbackRows, fallbackErr := geocodeCandidatesForQuery(ctx, query)
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
