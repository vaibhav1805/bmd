package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

func testRenderer() *Renderer {
	return NewRenderer(theme.NewThemeForScheme(theme.Dark), 80)
}

func TestRenderDocument_Empty(t *testing.T) {
	doc := ast.NewDocument()
	r := testRenderer()
	out := r.Render(doc)
	// Empty document produces just a trailing newline
	if out != "\n" {
		t.Errorf("Expected empty document to produce newline, got: %q", out)
	}
}

func TestRenderDocument_WithParagraph(t *testing.T) {
	doc := ast.NewDocument()
	p := ast.NewParagraph()
	p.AddChild(ast.NewText("Hello, world!"))
	doc.AddChild(p)

	r := testRenderer()
	out := r.Render(doc)
	if !strings.Contains(out, "Hello, world!") {
		t.Errorf("Expected paragraph text in output, got: %q", out)
	}
}

func TestRenderDocument_TwoParagraphs_HasNewlines(t *testing.T) {
	doc := ast.NewDocument()
	p1 := ast.NewParagraph()
	p1.AddChild(ast.NewText("First paragraph."))
	p2 := ast.NewParagraph()
	p2.AddChild(ast.NewText("Second paragraph."))
	doc.AddChild(p1)
	doc.AddChild(p2)

	r := testRenderer()
	out := r.Render(doc)
	if !strings.Contains(out, "First paragraph.") {
		t.Errorf("Expected first paragraph in output, got: %q", out)
	}
	if !strings.Contains(out, "Second paragraph.") {
		t.Errorf("Expected second paragraph in output, got: %q", out)
	}
	// Verify there's a newline between paragraphs
	firstIdx := strings.Index(out, "First paragraph.")
	secondIdx := strings.Index(out, "Second paragraph.")
	if firstIdx >= secondIdx {
		t.Errorf("Expected first paragraph before second")
	}
	between := out[firstIdx+len("First paragraph.") : secondIdx]
	if !strings.Contains(between, "\n") {
		t.Errorf("Expected newline between paragraphs, got: %q", between)
	}
}

func TestRenderText_BoldInParagraph(t *testing.T) {
	doc := ast.NewDocument()
	p := ast.NewParagraph()
	bold := ast.NewText("bold text")
	bold.Bold = true
	p.AddChild(bold)
	doc.AddChild(p)

	r := testRenderer()
	out := r.Render(doc)
	if !strings.Contains(out, ansiBold) {
		t.Errorf("Expected bold ANSI code in output, got: %q", out)
	}
	if !strings.Contains(out, "bold text") {
		t.Errorf("Expected bold text content in output, got: %q", out)
	}
}

func TestRenderNode_AllTypesNoParic(t *testing.T) {
	r := testRenderer()

	// All node types should render without panicking
	nodes := []ast.Node{
		func() ast.Node { n := ast.NewText("t"); return n }(),
		func() ast.Node { n := ast.NewCode("code"); return n }(),
		func() ast.Node {
			n := ast.NewHeading(2)
			n.AddChild(ast.NewText("heading"))
			return n
		}(),
		func() ast.Node {
			n := ast.NewCodeBlock("go", "fmt.Println(\"hi\")\n")
			return n
		}(),
		func() ast.Node {
			n := ast.NewBlockQuote()
			p := ast.NewParagraph()
			p.AddChild(ast.NewText("quote"))
			n.AddChild(p)
			return n
		}(),
		func() ast.Node {
			n := ast.NewList(false)
			li := ast.NewListItem()
			li.AddChild(ast.NewText("item"))
			n.AddChild(li)
			return n
		}(),
		func() ast.Node {
			n := ast.NewTable()
			row := ast.NewTableRow(true)
			cell := ast.NewTableCell("")
			cell.AddChild(ast.NewText("header"))
			row.AddChild(cell)
			n.AddChild(row)
			return n
		}(),
		ast.NewHardBreak(),
		ast.NewSoftBreak(),
		ast.NewHorizontalRule(),
	}

	for _, node := range nodes {
		result := r.RenderNode(node)
		_ = result // just verify no panic
	}
}

// TestThemesWithRenderer verifies all themes work correctly with the renderer.
func TestThemesWithRenderer(t *testing.T) {
	doc := ast.NewDocument()
	h := ast.NewHeading(1)
	h.AddChild(ast.NewText("Test Heading"))
	doc.AddChild(h)

	p := ast.NewParagraph()
	p.AddChild(ast.NewText("Body text"))
	doc.AddChild(p)

	code := ast.NewCodeBlock("go", "fmt.Println(\"hello\")\n")
	doc.AddChild(code)

	themeNames := theme.AvailableThemes()
	for _, name := range themeNames {
		th := theme.NewThemeByName(name)
		r := NewRenderer(th, 80)
		output := r.Render(doc)

		// Verify output is not empty
		if output == "" {
			t.Errorf("theme %s produced empty output", name)
		}

		// Verify output contains ANSI color codes (themes should apply colors)
		if !strings.Contains(output, "\x1b[") {
			t.Errorf("theme %s produced output with no ANSI codes", name)
		}
	}
}

// TestRenderImage verifies image rendering with alt text fallback.
func TestRenderImage(t *testing.T) {
	doc := ast.NewDocument()
	h := ast.NewHeading(1)
	h.AddChild(ast.NewText("Image Test"))
	doc.AddChild(h)

	// Create an image node
	img := &ast.Image{
		URL: "example.png",
		Alt: "Alt text",
	}
	doc.AddChild(img)

	r := testRenderer()
	output := r.Render(doc)

	// Verify output is not empty
	if output == "" {
		t.Error("image rendering produced empty output")
	}

	// Verify alt text appears (fallback for missing local file)
	if !strings.Contains(output, "Alt text") {
		t.Error("alt text not found in output")
	}
}

// TestRenderImage_PlainImagesFallback verifies that WithPlainImages() forces
// alt-text rendering even for a loadable local image and an active inline
// image protocol, so callers embedding output in a fixed-width sub-region
// (e.g. a split-pane preview) never receive an unclippable OSC escape blob.
func TestRenderImage_PlainImagesFallback(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if DetectImageProtocol() == ProtocolNone {
		t.Skip("no inline image protocol detected in this environment")
	}

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "sample.png")
	if err := os.WriteFile(imgPath, []byte("not a real png but non-empty"), 0o644); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	doc := ast.NewDocument()
	img := &ast.Image{URL: "sample.png", Alt: "Sample screenshot"}
	doc.AddChild(img)

	r := NewRenderer(theme.NewThemeForScheme(theme.Dark), 62).WithDocDir(dir).WithPlainImages()
	output := r.Render(doc)

	if strings.Contains(output, "\x1b]1337") {
		t.Errorf("WithPlainImages() output still contains inline image escape sequence: %q", output)
	}
	if !strings.Contains(output, "Sample screenshot") {
		t.Errorf("WithPlainImages() output missing alt text fallback: %q", output)
	}
}

// TestRenderImage_UnicodeProtocolUsesWiderRenderThanPixelProtocols verifies
// the block-art (ProtocolUnicode) fallback renders at nearly the full
// terminal width rather than the conservative 60%/100-column cap used for
// native image protocols (Kitty/iTerm2/Sixel, where width is just a
// cell-sizing hint for a real raster image). Block-art's actual resolution
// IS the column count, so a small width guarantees illegible output for
// anything with fine detail — this was the fix for real feedback that a
// dense textbook-page screenshot was unreadable even after the dithering
// fix, confirmed to meaningfully help once rendered at a wider column
// count.
func TestRenderImage_UnicodeProtocolUsesWiderRenderThanPixelProtocols(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("COLORTERM", "")
	if got := DetectImageProtocol(); got != ProtocolUnicode {
		t.Fatalf("expected ProtocolUnicode in this environment, got %v", got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.png"), makeTestPNG(t, 400, 400), 0o644); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	termWidth := 180
	r := NewRenderer(theme.NewThemeForScheme(theme.Dark), termWidth).WithDocDir(dir)
	img := &ast.Image{URL: "sample.png", Alt: "Sample"}
	out := r.renderImage(img)

	rows := strings.Count(out, "\n")
	// A square image renders roughly (targetWidth/2) rows. At the old
	// 60%-of-180-clamped-to-100 logic that's ~50 rows; at nearly the full
	// terminal width (180-4=176) it should be closer to ~88.
	if rows < 60 {
		t.Errorf("expected a much wider/taller render at termWidth=%d (ProtocolUnicode should use ~full width, not the 100-col pixel-protocol cap), got %d rows in output of length %d", termWidth, rows, len(out))
	}
}

// TestImageProtocolDetection verifies image protocol detection works.
func TestImageProtocolDetection(t *testing.T) {
	protocol := DetectImageProtocol()
	// Just verify it returns a valid value
	if protocol < 0 || protocol > 4 {
		t.Errorf("DetectImageProtocol returned invalid value: %d", protocol)
	}
}

// TestDetectImageProtocol_XtermTermWithoutTermProgram_NoProtocolGuessed
// guards against two real-world misdetections found via manual testing,
// both for the same ambiguous case (TERM=xterm-256color, no recognized
// TERM_PROGRAM — Alacritty configured for broad/SSH compatibility hits
// this, since it bypasses the explicit "alacritty" TERM/COLORTERM check):
//
//  1. The original logic assumed any such xterm-family TERM on macOS meant
//     Terminal.app (via os.Stat on Terminal.app's install path — present on
//     virtually every Mac regardless of the running terminal) and reported
//     ProtocolNone, permanently disabling images.
//  2. A first fix instead guessed ProtocolKitty unconditionally — but
//     Alacritty does not support the Kitty graphics protocol, so this
//     printed the raw, unrendered escape sequence as literal garbage text.
//
// With no reliable signal either way, detection must not gamble on a
// specific protocol and should fall through to the safe alt-text default.
func TestDetectImageProtocol_XtermTermWithoutTermProgram_NoProtocolGuessed(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("COLORTERM", "")

	got := DetectImageProtocol()
	if got != ProtocolUnicode {
		t.Errorf("expected the safe ProtocolUnicode fallback (alt text, no protocol-specific escape emitted) for an unrecognized xterm-family terminal, got %v", got)
	}
}

// TestSixelAvailable checks if Sixel availability detection works.
func TestSixelAvailable(t *testing.T) {
	available := SixelAvailable()
	// Should return bool without panicking
	_ = available
}

// TestImageToSixel verifies Sixel conversion handling.
func TestImageToSixel(t *testing.T) {
	// Test with empty data
	result := ImageToSixel([]byte{}, 40, 10)
	if result != "" {
		t.Errorf("ImageToSixel with empty data should return empty string, got: %q", result)
	}

	// Test with dummy image data (will fail if convert not available, but shouldn't panic)
	dummyData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	result = ImageToSixel(dummyData, 40, 10)
	// Result should be either Sixel data or fallback message, never empty with non-empty input
	if result == "" {
		t.Errorf("ImageToSixel should return non-empty result for non-empty input")
	}
}

// TestProtocolCapabilities verifies human-readable capability strings.
func TestProtocolCapabilities(t *testing.T) {
	caps := ProtocolCapabilities()
	if caps == "" {
		t.Errorf("ProtocolCapabilities should return non-empty string")
	}
	if !strings.Contains(caps, "protocol") && !strings.Contains(caps, "support") && !strings.Contains(caps, "fallback") {
		t.Errorf("ProtocolCapabilities should describe a protocol or fallback, got: %q", caps)
	}
}

// TestConvertImageToSixel verifies the conversion helper.
func TestConvertImageToSixel(t *testing.T) {
	// Test with empty data
	result := ConvertImageToSixel([]byte{})
	if result != "" {
		t.Errorf("ConvertImageToSixel with empty data should return empty string, got: %q", result)
	}

	// Test with invalid image data (should fail gracefully)
	result = ConvertImageToSixel([]byte{0xFF, 0xFE, 0xFD})
	// Should return empty string if convert fails or not available
	_ = result
}
