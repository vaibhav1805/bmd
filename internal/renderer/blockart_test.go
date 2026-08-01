package renderer

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// makeTestPNG returns PNG-encoded bytes for a w x h image where the top
// half is solid red and the bottom half is solid blue.
func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if y < h/2 {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{B: 255, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return buf.Bytes()
}

// TestDitherFloydSteinberg_UniformRegionProducesNoInk is the regression
// test for the "seriously broken... static noise" bug: a flat, uniform
// region (no real detail at all) must dither to zero ink dots, rendering
// as a clean solid background fill. The earlier per-cell RGB-clustering
// approach instead always found *some* two "most different" samples
// (ordinary sampling noise is never perfectly zero-variance) and forced a
// high-contrast dot pattern there anyway, which is what produced the
// speckled/illegible look on real photos and screenshots.
func TestDitherFloydSteinberg_UniformRegionProducesNoInk(t *testing.T) {
	gray := [][]float64{{200, 200, 200, 200}}
	ink := ditherFloydSteinberg(gray, 4, 1)
	for x := 0; x < 4; x++ {
		if ink[0][x] {
			t.Errorf("expected no ink dots in a uniform flat region, got ink at x=%d (row: %v)", x, ink[0])
		}
	}
}

// TestDitherFloydSteinberg_HighContrastSplitProducesInkOnDarkSide verifies
// a genuine light/dark edge is captured correctly: the dark (minority)
// side should dither to ink, the light (majority) side to paper. Values
// chosen and hand-traced against the exact Floyd-Steinberg diffusion
// (7/16, 3/16, 5/16, 1/16 to right/below-left/below/below-right) for a
// single row, so the expected ink pattern is exact, not approximate.
func TestDitherFloydSteinberg_HighContrastSplitProducesInkOnDarkSide(t *testing.T) {
	gray := [][]float64{{240, 240, 15, 15}}
	ink := ditherFloydSteinberg(gray, 4, 1)
	want := []bool{false, false, true, true}
	for x, w := range want {
		if ink[0][x] != w {
			t.Errorf("at x=%d: expected ink=%v, got %v (full row: %v)", x, w, ink[0][x], ink[0])
		}
	}
}

// TestRenderHalfBlockImage_UsesBrailleCodepoints verifies every glyph
// emitted falls in the Unicode Braille Patterns block (U+2800-U+28FF),
// confirming Braille rendering is actually in effect (not a leftover
// block-element glyph from the prior quadrant technique).
func TestRenderHalfBlockImage_UsesBrailleCodepoints(t *testing.T) {
	data := makeTestPNG(t, 8, 8)
	art, err := renderHalfBlockImage(data, 4)
	if err != nil {
		t.Fatalf("renderHalfBlockImage: %v", err)
	}
	plain := stripANSIForTest(art)
	for _, r := range plain {
		if r == '\n' {
			continue
		}
		if r < 0x2800 || r > 0x28FF {
			t.Fatalf("expected only Braille Pattern glyphs (U+2800-U+28FF), found %q (%U) in: %q", r, r, art)
		}
	}
}

// stripANSIForTest strips ANSI SGR escape sequences for glyph inspection
// (a local minimal copy — this package doesn't import the tui package's
// stripANSI helper). ESC and '[' are single-byte ASCII, so byte-level
// scanning can't corrupt multi-byte UTF-8 runes elsewhere in the string.
func stripANSIForTest(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func TestRenderHalfBlockImage_DecodesAndRenders(t *testing.T) {
	data := makeTestPNG(t, 8, 8)

	art, err := renderHalfBlockImage(data, 4)
	if err != nil {
		t.Fatalf("renderHalfBlockImage: %v", err)
	}
	if !strings.Contains(art, "\x1b[38;2;") || !strings.Contains(art, "\x1b[48;2;") {
		t.Errorf("expected truecolor foreground and background SGR codes, got: %q", art)
	}
	// The top rows should render with a red background (approx 255;0;0),
	// reflecting the red top half of the source image.
	if !strings.Contains(art, "\x1b[48;2;255;0;0m") {
		t.Errorf("expected a red background cell for the image's red top half, got: %q", art)
	}
	// And the bottom rows should render with a blue background.
	if !strings.Contains(art, "\x1b[48;2;0;0;255m") {
		t.Errorf("expected a blue background cell for the image's blue bottom half, got: %q", art)
	}
}

func TestRenderHalfBlockImage_InvalidDataReturnsError(t *testing.T) {
	if _, err := renderHalfBlockImage([]byte("not an image"), 10); err == nil {
		t.Error("expected an error for undecodable image data")
	}
}

func TestRenderHalfBlockImage_RowCountMatchesAspectRatio(t *testing.T) {
	// A square (1:1) source image at cols=10, with a 2:1 cell aspect ratio,
	// should render to roughly cols/2 rows.
	data := makeTestPNG(t, 100, 100)
	art, err := renderHalfBlockImage(data, 10)
	if err != nil {
		t.Fatalf("renderHalfBlockImage: %v", err)
	}
	rows := strings.Count(art, "\n") + 1
	if rows < 3 || rows > 7 {
		t.Errorf("expected roughly 5 rows for a square image at 10 cols (2:1 cell aspect), got %d rows in: %q", rows, art)
	}
}

func TestImageToUnicode_UsesRealRenderingWhenDecodable(t *testing.T) {
	data := makeTestPNG(t, 8, 8)
	out := ImageToUnicode(data, "alt text", 4)
	if strings.HasPrefix(out, "[Image") {
		t.Errorf("expected real block-art rendering, got the text placeholder: %q", out)
	}
	if !strings.Contains(out, "\x1b[38;2;") {
		t.Errorf("expected truecolor SGR codes in output, got: %q", out)
	}
}

func TestImageToUnicode_FallsBackToAltTextWhenUndecodable(t *testing.T) {
	out := ImageToUnicode([]byte("garbage"), "alt text", 4)
	if out != "[Image: alt text]" {
		t.Errorf("expected alt-text fallback for undecodable data, got: %q", out)
	}
}

func TestImageToTerminal_UnicodeProtocolRendersRealArt(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("COLORTERM", "")

	data := makeTestPNG(t, 8, 8)
	out := ImageToTerminal(data, "/tmp/whatever.png", "alt text", 4, 4)
	if !strings.Contains(out, "\x1b[38;2;") {
		t.Errorf("expected block-art rendering from ImageToTerminal's ProtocolUnicode path, got: %q", out)
	}
}

func TestDetectImageProtocol_Alacritty_UsesHalfBlockNotKitty(t *testing.T) {
	t.Setenv("TERM", "alacritty")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")

	got := DetectImageProtocol()
	if got != ProtocolUnicode {
		t.Errorf("expected ProtocolUnicode for Alacritty (no native image protocol support), got %v", got)
	}
}

func TestDetectImageProtocol_AppleTerminal_UsesHalfBlockNotITerm2(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")

	got := DetectImageProtocol()
	if got != ProtocolUnicode {
		t.Errorf("expected ProtocolUnicode for Apple_Terminal (no native image protocol support), got %v", got)
	}
}

func TestDetectImageProtocol_RealITerm2StillGetsITerm2Protocol(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")

	got := DetectImageProtocol()
	if got != ProtocolITerm2 {
		t.Errorf("expected ProtocolITerm2 for real iTerm2, got %v", got)
	}
}
