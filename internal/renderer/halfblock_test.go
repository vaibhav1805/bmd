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
// half is solid red and the bottom half is solid blue — chosen so a
// half-block render's top/bottom color split is easy to assert on.
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

func TestRenderHalfBlockImage_DecodesAndRenders(t *testing.T) {
	data := makeTestPNG(t, 8, 8)

	art, err := renderHalfBlockImage(data, 4)
	if err != nil {
		t.Fatalf("renderHalfBlockImage: %v", err)
	}
	if !strings.Contains(art, "▄") {
		t.Errorf("expected half-block glyph U+2584 in output, got: %q", art)
	}
	if !strings.Contains(art, "\x1b[38;2;") || !strings.Contains(art, "\x1b[48;2;") {
		t.Errorf("expected truecolor foreground and background SGR codes, got: %q", art)
	}
	// Top rows should be background-red (approx 255;0;0), reflecting the
	// red top half of the source image.
	if !strings.Contains(art, "\x1b[48;2;255;0;0m") {
		t.Errorf("expected a red background cell for the image's red top half, got: %q", art)
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
		t.Errorf("expected real half-block art, got the text placeholder: %q", out)
	}
	if !strings.Contains(out, "▄") {
		t.Errorf("expected half-block glyph in output, got: %q", out)
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
	if !strings.Contains(out, "▄") {
		t.Errorf("expected half-block art from ImageToTerminal's ProtocolUnicode path, got: %q", out)
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
