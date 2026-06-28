package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// SearchAddress returns the best OpenStreetMap coordinate match for a Ukrainian address.
func SearchAddress(ctx context.Context, address string) (string, string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", "", fmt.Errorf("вкажіть адресу")
	}
	params := url.Values{}
	params.Set("q", address)
	params.Set("format", "jsonv2")
	params.Set("limit", "1")
	params.Set("countrycodes", "ua")
	params.Set("accept-language", "uk")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://nominatim.openstreetmap.org/search?"+params.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")
	response, err := httpClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("геопошук: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return "", "", fmt.Errorf("геосервіс повернув %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var rows []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&rows); err != nil {
		return "", "", fmt.Errorf("відповідь геосервісу: %w", err)
	}
	if len(rows) == 0 || strings.TrimSpace(rows[0].Lat) == "" || strings.TrimSpace(rows[0].Lon) == "" {
		return "", "", fmt.Errorf("адресу не знайдено")
	}
	return strings.TrimSpace(rows[0].Lat), strings.TrimSpace(rows[0].Lon), nil
}
