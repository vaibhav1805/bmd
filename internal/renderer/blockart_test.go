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

// TestQuantizeCell_UniformColorYieldsZeroMask verifies a flat region (no
// color variation across all samples) clusters entirely into one group,
// yielding an all-zero mask — a blank/dot-less glyph showing only the
// solid background color.
func TestQuantizeCell_UniformColorYieldsZeroMask(t *testing.T) {
	c := rgb{100, 150, 200}
	samples := []rgb{c, c, c, c, c, c, c, c}
	mask, bg, fg := quantizeCell(samples)
	if mask != 0 {
		t.Errorf("expected mask=0 for a uniform-color cell, got %#b", mask)
	}
	if bg != c || fg != c {
		t.Errorf("expected both bg and fg to equal the uniform color %v, got bg=%v fg=%v", c, bg, fg)
	}
}

// TestQuantizeCell_TwoDistinctGroupsCluster verifies samples split cleanly
// into two color groups: the first half red, the second half blue. Every
// red sample must land in the background cluster (mask bit unset) and
// every blue sample in the foreground cluster (mask bit set), regardless
// of how many total samples are clustered (4 for quadrants, 8 for
// Braille) — this is the general clustering logic both build on.
func TestQuantizeCell_TwoDistinctGroupsCluster(t *testing.T) {
	red := rgb{255, 0, 0}
	blue := rgb{0, 0, 255}
	samples := []rgb{red, red, red, red, blue, blue, blue, blue}
	mask, bg, fg := quantizeCell(samples)

	var want uint16 = 0b11110000
	if mask != want {
		t.Errorf("expected mask=%#b (last 4 samples foreground), got %#b", want, mask)
	}
	if bg != red {
		t.Errorf("expected background=red, got %v", bg)
	}
	if fg != blue {
		t.Errorf("expected foreground=blue, got %v", fg)
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
