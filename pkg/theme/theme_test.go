package theme

import (
	"image/color"
	"testing"

	fynetheme "fyne.io/fyne/v2/theme"
)

func TestRecoveredFrontendPaletteTokens(t *testing.T) {
	tests := []struct {
		name string
		got  color.Color
		want color.NRGBA
	}{
		{name: "primary", got: ColorInfo, want: color.NRGBA{R: 69, G: 133, B: 188, A: 255}},
		{name: "light background", got: NewLightTheme(12).Color(fynetheme.ColorNameBackground, fynetheme.VariantLight), want: color.NRGBA{R: 248, G: 249, B: 250, A: 255}},
		{name: "light separator", got: NewLightTheme(12).Color(fynetheme.ColorNameSeparator, fynetheme.VariantLight), want: color.NRGBA{R: 177, G: 191, B: 205, A: 255}},
		{name: "dark background", got: NewDarkTheme(12).Color(fynetheme.ColorNameBackground, fynetheme.VariantDark), want: color.NRGBA{R: 43, G: 49, B: 69, A: 255}},
		{name: "dark surface", got: NewDarkTheme(12).Color(fynetheme.ColorNameButton, fynetheme.VariantDark), want: color.NRGBA{R: 37, G: 43, B: 62, A: 255}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := color.NRGBAModel.Convert(test.got).(color.NRGBA); got != test.want {
				t.Fatalf("color = %+v, want %+v", got, test.want)
			}
		})
	}
}
