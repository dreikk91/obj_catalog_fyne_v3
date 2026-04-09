package data

import (
	"context"
	"testing"
	"time"
)

func TestCASLCloudProvider_ResolveCASLDeviceLineTypeLabel(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	provider.cachedDictionary = map[string]any{
		"dictionary": map[string]any{
			"line_types": []any{"NORMAL"},
			"translate": map[string]any{
				"uk": map[string]any{
					"NORMAL": "Охоронна зона",
				},
			},
		},
	}
	provider.cachedDictionaryAt = time.Now()
	provider.mu.Unlock()

	got := provider.resolveCASLDeviceLineTypeLabel(context.Background(), caslDeviceLine{
		LineType: caslText("NORMAL"),
	})
	if got != "Охоронна зона" {
		t.Fatalf("resolveCASLDeviceLineTypeLabel() = %q, want %q", got, "Охоронна зона")
	}
}

func TestCASLCloudProvider_ResolveCASLDeviceLineTypeLabel_UsesDictionaryTranslateMap(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	provider.cachedDictionary = map[string]any{
		"dictionary": map[string]any{
			"line_types": []any{"ALM_BTN"},
			"translate": map[string]any{
				"uk": map[string]any{
					"ALM_BTN": "Тривожний на Обрив",
				},
			},
		},
	}
	provider.cachedDictionaryAt = time.Now()
	provider.mu.Unlock()

	got := provider.resolveCASLDeviceLineTypeLabel(context.Background(), caslDeviceLine{
		LineType: caslText("ALM_BTN"),
	})
	if got != "Тривожний на Обрив" {
		t.Fatalf("resolveCASLDeviceLineTypeLabel() = %q, want %q", got, "Тривожний на Обрив")
	}
}

func TestCASLCloudProvider_ResolveCASLDeviceLineTypeLabel_Fallbacks(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)

	tests := []struct {
		name string
		line caslDeviceLine
		want string
	}{
		{
			name: "line_type fallback",
			line: caslDeviceLine{LineType: caslText("ZONE_FIRE")},
			want: "Пожежний шлейф",
		},
		{
			name: "preserves human device type",
			line: caslDeviceLine{Type: caslText("PIR")},
			want: "PIR",
		},
		{
			name: "raw type fallback when line type missing",
			line: caslDeviceLine{Type: caslText("ALM_BTN")},
			want: "Тривожна кнопка",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := provider.resolveCASLDeviceLineTypeLabel(context.Background(), tt.line)
			if got != tt.want {
				t.Fatalf("resolveCASLDeviceLineTypeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
