package tui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Wian47/GitSketch/internal/config"
)

func TestApplyThemeOverridesColors(t *testing.T) {
	t.Cleanup(func() { ApplyTheme(config.DefaultTheme()) })

	custom := config.DefaultTheme()
	custom.Hash = "#123456"
	ApplyTheme(custom)

	got := HashStyle.GetForeground()
	want := lipglossColor(t, "#123456")
	assertSameColor(t, got, want)
}

func TestApplyThemeDefaultMatchesOriginalPalette(t *testing.T) {
	ApplyTheme(config.DefaultTheme())

	got := DimStyle.GetForeground()
	want := lipglossColor(t, "#546E7A")
	assertSameColor(t, got, want)
}

func lipglossColor(t *testing.T, hex string) color.Color {
	t.Helper()
	return lipgloss.Color(hex)
}

func assertSameColor(t *testing.T, got, want color.Color) {
	t.Helper()
	gr, gg, gb, ga := got.RGBA()
	wr, wg, wb, wa := want.RGBA()
	if gr != wr || gg != wg || gb != wb || ga != wa {
		t.Fatalf("color mismatch: got RGBA(%d,%d,%d,%d), want RGBA(%d,%d,%d,%d)", gr, gg, gb, ga, wr, wg, wb, wa)
	}
}
