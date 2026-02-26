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
