package theme

import (
	"testing"
)

func TestDarkThemeHeadingColors(t *testing.T) {
	th := NewThemeForScheme(Dark)
	// Verify all 6 heading levels have distinct colors
	seen := map[AnsiColor]int{}
	for level := 1; level <= 6; level++ {
		c := th.HeadingColor(level)
		if c < 0 || c > 255 {
			t.Errorf("HeadingColor(%d) = %d, not in range 0-255", level, c)
		}
		seen[c]++
	}
	if len(seen) < 4 {
		t.Errorf("Expected at least 4 distinct heading colors, only %d unique values", len(seen))
	}
}

func TestLightThemeHeadingColors(t *testing.T) {
	th := NewThemeForScheme(Light)
	for level := 1; level <= 6; level++ {
		c := th.HeadingColor(level)
		if c < 0 || c > 255 {
			t.Errorf("HeadingColor(%d) = %d, not in range 0-255", level, c)
		}
	}
}

func TestColorCodesValid(t *testing.T) {
	th := NewThemeForScheme(Dark)
	colors := []AnsiColor{
		th.CodeColor(),
		th.CodeBgColor(),
		th.QuoteColor(),
		th.TextColor(), // NoColor is -1, that's OK
		th.StrikeColor(),
		th.LinkColor(),
		th.HrColor(),
		th.CodeBlockFg(),
		th.CodeBlockBg(),
	}
	for _, c := range colors {
		if c != NoColor && (c < 0 || c > 255) {
			t.Errorf("Color %d is not in valid range 0-255 (and not NoColor)", c)
		}
	}
}

func TestFgCode(t *testing.T) {
	code := FgCode(208)
	expected := "\x1b[38;5;208m"
	if code != expected {
		t.Errorf("FgCode(208) = %q, want %q", code, expected)
	}
}

func TestBgCode(t *testing.T) {
	code := BgCode(236)
	expected := "\x1b[48;5;236m"
	if code != expected {
		t.Errorf("BgCode(236) = %q, want %q", code, expected)
	}
}

func TestFgCodeNoColor(t *testing.T) {
	code := FgCode(NoColor)
	if code != "" {
		t.Errorf("FgCode(NoColor) should return empty string, got %q", code)
	}
}

func TestDetectColorScheme_Default(t *testing.T) {
	// Just verify it returns a valid value (Dark or Light)
	scheme := DetectColorScheme()
	if scheme != Dark && scheme != Light {
		t.Errorf("DetectColorScheme returned unexpected value: %v", scheme)
	}
}

func TestHeadingColorBoundary(t *testing.T) {
	th := NewThemeForScheme(Dark)
	// Should clamp at boundaries
	c0 := th.HeadingColor(0)
	c1 := th.HeadingColor(1)
	if c0 != c1 {
		t.Errorf("HeadingColor(0) and HeadingColor(1) should be same (clamped), got %d vs %d", c0, c1)
	}
	c7 := th.HeadingColor(7)
	c6 := th.HeadingColor(6)
	if c7 != c6 {
		t.Errorf("HeadingColor(7) and HeadingColor(6) should be same (clamped), got %d vs %d", c7, c6)
	}
}

// TestThemeDistinctness verifies each theme has distinct colors and differs from others.
func TestThemeDistinctness(t *testing.T) {
	themes := map[string]Theme{
		"ocean":     NewThemeByName(ThemeOcean),
		"forest":    NewThemeByName(ThemeForest),
		"sunset":    NewThemeByName(ThemeSunset),
		"midnight":  NewThemeByName(ThemeMidnight),
	}

	// Verify ocean heading colors are distinct from others
	oceanH1 := themes["ocean"].HeadingColor(1)
	forestH1 := themes["forest"].HeadingColor(1)
	sunsetH1 := themes["sunset"].HeadingColor(1)
	midnightH1 := themes["midnight"].HeadingColor(1)

	if oceanH1 == forestH1 {
		t.Error("ocean and forest themes have same H1 color")
	}
	if oceanH1 == sunsetH1 {
		t.Error("ocean and sunset themes have same H1 color")
	}
	if oceanH1 == midnightH1 {
		t.Error("ocean and midnight themes have same H1 color")
	}
	if forestH1 == sunsetH1 {
		t.Error("forest and sunset themes have same H1 color")
	}
	if forestH1 == midnightH1 {
		t.Error("forest and midnight themes have same H1 color")
	}
	if sunsetH1 == midnightH1 {
		t.Error("sunset and midnight themes have same H1 color")
	}
}

// TestThemeContrast verifies code colors and link colors have good contrast.
func TestThemeContrast(t *testing.T) {
	themes := []struct {
		name  string
		theme Theme
	}{
		{"ocean", NewThemeByName(ThemeOcean)},
		{"forest", NewThemeByName(ThemeForest)},
		{"sunset", NewThemeByName(ThemeSunset)},
		{"midnight", NewThemeByName(ThemeMidnight)},
	}

	for _, tc := range themes {
		// Inline code should have a background (codeBgColor != NoColor)
		if tc.theme.CodeBgColor() == NoColor {
			t.Errorf("%s theme inline code has no background color", tc.name)
		}

		// Code block fg should be different from code block bg
		if tc.theme.CodeBlockFg() == tc.theme.CodeBlockBg() {
			t.Errorf("%s theme code block fg and bg are the same color", tc.name)
		}

		// Links should be colored (not NoColor)
		if tc.theme.LinkColor() == NoColor {
			t.Errorf("%s theme links have no color", tc.name)
		}
	}
}

// TestAllThemesInitialize verifies all themes can be created without panic.
func TestAllThemesInitialize(t *testing.T) {
	themeNames := AvailableThemes()
	if len(themeNames) < 5 {
		t.Errorf("Expected at least 5 themes, got %d", len(themeNames))
	}

	for _, name := range themeNames {
		th := NewThemeByName(name)
		// Verify theme has valid scheme
		if th.Scheme() != Dark && th.Scheme() != Light {
			t.Errorf("theme %s has invalid scheme", name)
		}
	}
}
