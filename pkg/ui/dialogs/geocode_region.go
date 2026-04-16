package dialogs

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func resolveRegionByAddressHints(provider contracts.DistrictReferenceService, hints []string) (int64, string, error) {
	if len(hints) == 0 {
		return 0, "", fmt.Errorf("геосервіс не повернув район")
	}
	regions, err := provider.ListObjectDistricts()
	if err != nil {
		return 0, "", fmt.Errorf("не вдалося завантажити райони: %w", err)
	}
	if len(regions) == 0 {
		return 0, "", fmt.Errorf("довідник районів порожній")
	}

	type regionCandidate struct {
		ID   int64
		Name string
		Norm string
	}
	candidates := make([]regionCandidate, 0, len(regions))
	for _, region := range regions {
		name := strings.TrimSpace(region.Name)
		if name == "" || region.ID <= 0 {
			continue
		}
		candidates = append(candidates, regionCandidate{
			ID:   region.ID,
			Name: name,
			Norm: normalizeDistrictName(name),
		})
	}
	if len(candidates) == 0 {
		return 0, "", fmt.Errorf("не знайдено валідних районів у довіднику")
	}

	hintNorms := make([]string, 0, len(hints))
	for _, hint := range hints {
		if norm := normalizeDistrictName(hint); norm != "" {
			hintNorms = append(hintNorms, norm)
		}
	}
	if len(hintNorms) == 0 {
		return 0, "", fmt.Errorf("не вдалося витягнути назву району з адреси")
	}

	for _, hintNorm := range hintNorms {
		for _, c := range candidates {
			if c.Norm != "" && c.Norm == hintNorm {
				return c.ID, c.Name, nil
			}
		}
	}
	for _, hintNorm := range hintNorms {
		for _, c := range candidates {
			if c.Norm == "" {
				continue
			}
			if strings.Contains(hintNorm, c.Norm) || strings.Contains(c.Norm, hintNorm) {
				return c.ID, c.Name, nil
			}
		}
	}

	return 0, "", fmt.Errorf("район не зіставлено з довідником: %s", strings.Join(hints, ", "))
}

func normalizeDistrictName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"’", "'",
		"`", "'",
		"ʼ", "'",
		".", " ",
		",", " ",
		"(", " ",
		")", " ",
		"-", " ",
		"/", " ",
	)
	s = replacer.Replace(s)
	s = strings.ReplaceAll(s, "р-н", "район")

	tokens := strings.Fields(s)
	filtered := make([]string, 0, len(tokens))
	stopWords := map[string]struct{}{
		"район":   {},
		"місто":   {},
		"м":       {},
		"область": {},
		"обл":     {},
		"city":    {},
	}
	for _, t := range tokens {
		if _, skip := stopWords[t]; skip {
			continue
		}
		filtered = append(filtered, t)
	}
	return strings.Join(filtered, " ")
}
