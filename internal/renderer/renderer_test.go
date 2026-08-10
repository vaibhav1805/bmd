package renderer

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
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

// TestRenderImage_UnicodeProtocolIsThumbnailSized verifies the block-art
// (ProtocolUnicode) fallback renders as a compact thumbnail — smaller than
// native image protocols get — rather than stretched toward the full
// terminal width or matching the native-protocol sizing. Per user
// testing, a dithered dot pattern reads more cleanly at a smaller size
// (like halftone print), and since block-art has a hard resolution
// ceiling no width increase can fix for fine detail anyway (see
// renderImage's comment), a large render just adds scroll-heavy document
// real estate without adding legibility.
func TestRenderImage_UnicodeProtocolIsThumbnailSized(t *testing.T) {
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
	// A square image renders roughly (targetWidth/2) rows. At the new
	// 40%-of-180-clamped-to-50 thumbnail sizing, that's ~25 rows — well
	// short of the ~50 rows the native-protocol 60%/100-col sizing (or
	// the ~88 rows a near-full-terminal-width render) would produce.
	if rows > 30 {
		t.Errorf("expected a compact thumbnail render (~25 rows) at termWidth=%d, got %d rows — image is larger than intended", termWidth, rows)
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

// TestImageToKitty_UsesSemicolonSeparator is a regression test for a real
// bug found via manual testing: opening an image doc in an actual Kitty
// terminal produced no image at all (not even a text fallback) because
// ImageToKitty separated its control data from the base64 payload with a
// colon (':') instead of the protocol's actual semicolon (';') separator
// — per the Kitty graphics protocol spec, "All graphics escape codes are
// of the form: <ESC>_G<control data>;<payload><ESC>\". With the wrong
// separator, the terminal never finds the control-data boundary it
// expects and the whole command silently fails to parse.
func TestImageToKitty_UsesSemicolonSeparator(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 1, 2, 3}
	seq := ImageToKitty(data, 20)

	const wantPrefix = "\x1b_Ga=T,f=100,r=20,m=0;"
	if !strings.HasPrefix(seq, wantPrefix) {
		t.Fatalf("expected control data terminated by ';', got: %q", seq)
	}
	if !strings.HasSuffix(seq, "\x1b\\") {
		t.Errorf("expected the sequence to end with the ST terminator (\\x1b\\\\), got: %q", seq)
	}
	// The colon must not appear as a stand-in separator anywhere before
	// the payload — assert the exact control-data prefix above catches
	// this, but also guard against a colon sneaking in right at the
	// control-data/payload boundary.
	if strings.Contains(seq, "m=0:") {
		t.Errorf("found the old, wrong colon separator in the Kitty escape sequence: %q", seq)
	}
}

// TestImageToKitty_ChunksLargePayloads is a regression test for a second
// real bug found via manual testing (after fixing the ':' vs ';'
// separator): a large image (base64-encoded well over 4096 bytes, which
// any real photo/screenshot always is) still produced no image at all in
// a real Kitty terminal. Root cause: the Kitty graphics protocol has a
// hard 4096-byte limit on base64 payload PER CHUNK — any larger transfer
// must be split across multiple escape sequences (m=1 on every chunk but
// the last, m=0 on the final one), with only the first chunk carrying the
// full control-data set. ImageToKitty previously sent the entire payload,
// however large, as a single m=0 ("final, no more chunks") chunk,
// violating that limit — the terminal never finds a validly-sized single
// command and drops it.
func TestImageToKitty_ChunksLargePayloads(t *testing.T) {
	// 10000 raw bytes -> ~13336 base64 chars -> should split into 4 chunks
	// of kittyChunkSize (4096) plus a final shorter one.
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	seq := ImageToKitty(data, 20)

	encoded := base64.StdEncoding.EncodeToString(data)
	wantChunks := (len(encoded) + kittyChunkSize - 1) / kittyChunkSize
	if wantChunks < 2 {
		t.Fatalf("test setup error: expected the encoded payload to need multiple chunks, got %d", wantChunks)
	}

	gotChunks := strings.Count(seq, "\x1b_G")
	if gotChunks != wantChunks {
		t.Errorf("expected %d chunked escape sequences for a %d-byte encoded payload, got %d", wantChunks, len(encoded), gotChunks)
	}

	// Every chunk but the last must be marked m=1 (more chunks follow);
	// only the last must be m=0.
	if strings.Count(seq, "m=1;") != wantChunks-1 {
		t.Errorf("expected exactly %d chunks marked m=1 (all but the last), got %d in: %.200q...", wantChunks-1, strings.Count(seq, "m=1;"), seq)
	}
	if strings.Count(seq, "m=0;") != 1 {
		t.Errorf("expected exactly 1 chunk marked m=0 (the final one), got %d", strings.Count(seq, "m=0;"))
	}

	// Only the first chunk should carry the full control-data set.
	if strings.Count(seq, "a=T,f=100,") != 1 {
		t.Errorf("expected the full control-data set (a=T,f=100,...) exactly once (first chunk only), got %d occurrences", strings.Count(seq, "a=T,f=100,"))
	}

	// Reassembling the payload from all chunks (in order) must exactly
	// reproduce the original base64 string.
	var reassembled strings.Builder
	rest := seq
	for {
		idx := strings.Index(rest, "\x1b_G")
		if idx == -1 {
			break
		}
		rest = rest[idx+len("\x1b_G"):]
		semi := strings.Index(rest, ";")
		if semi == -1 {
			t.Fatalf("malformed chunk: no ';' separator found")
		}
		rest = rest[semi+1:]
		term := strings.Index(rest, "\x1b\\")
		if term == -1 {
			t.Fatalf("malformed chunk: no ST terminator found")
		}
		reassembled.WriteString(rest[:term])
		rest = rest[term+len("\x1b\\"):]
	}
	if reassembled.String() != encoded {
		t.Errorf("reassembled chunk payloads don't match the original base64 data (lengths: got %d, want %d)", reassembled.Len(), len(encoded))
	}
}

// TestKittyDeleteAllImages is a regression test for bmd-xqh's ghost-image
// symptom: Kitty renders images as an out-of-band pixel overlay that
// persists on screen even after bmd redraws different text in the same
// cells (e.g. navigating to a different file), so bmd must be able to emit
// an explicit delete-all-images command (a=d action, d=A: all placements +
// cached data) at navigation choke points.
func TestKittyDeleteAllImages(t *testing.T) {
	got := KittyDeleteAllImages()
	want := "\x1b_Ga=d,d=A\x1b\\"
	if got != want {
		t.Errorf("KittyDeleteAllImages() = %q, want %q", got, want)
	}
}

// TestImageToKitty_ConstrainsDisplaySizeWithRowsOnly is a regression test
// for bmd-xqh: ImageToKitty previously omitted Kitty's r= placement key
// entirely, so the terminal rendered the image at its natural pixel size
// (intrinsic PNG dimensions / cell size) — almost always many more terminal
// rows than the single v.Lines entry bmd's document model accounts for per
// image, causing the image to overlap subsequent content and breaking
// click-to-line math for everything below it. r= must be set to the exact
// height (in terminal cells) the caller requested, on the first chunk only
// (matching a=/f=/m=).
//
// A second regression (also bmd-xqh): an earlier attempt set BOTH c= and
// r=, which per the Kitty spec forces the image into that exact box
// regardless of its real aspect ratio — visibly stretching/squashing a
// non-square image in real-terminal testing. c= must NOT be set, so Kitty
// computes columns from the PNG's own aspect ratio given the fixed r=.
func TestImageToKitty_ConstrainsDisplaySizeWithRowsOnly(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 1, 2, 3}
	seq := ImageToKitty(data, 17)

	const wantPrefix = "\x1b_Ga=T,f=100,r=17,m=0;"
	if !strings.HasPrefix(seq, wantPrefix) {
		t.Fatalf("expected control data to constrain display size via r= only, got: %q", seq)
	}
	if strings.Contains(seq, "c=") {
		t.Errorf("expected no c= key (would force distortion instead of preserving aspect ratio), got: %q", seq)
	}
}

// TestRenderImage_KittyHeightCappedAndReservesMatchingRows is a regression
// test for bmd-xqh: a first attempt let imageHeight scale proportionally
// with imageWidth (up to 66 rows on a wide terminal) with no relation to
// the terminal's actual visible row count, and requesting a Kitty image
// taller than the real window corrupted rendering in practice. renderImage
// must cap native-protocol (Kitty/iTerm2) height to maxNativeImageHeight
// regardless of how wide the terminal is, and the blank-line padding after
// the image (so v.Lines row-accounting matches what's on screen) must
// track that same capped value, not the uncapped width-derived one.
func TestRenderImage_KittyHeightCappedAndReservesMatchingRows(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	if got := DetectImageProtocol(); got != ProtocolKitty {
		t.Fatalf("expected ProtocolKitty in this environment, got %v", got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.png"), makeTestPNG(t, 400, 400), 0o644); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	// A wide terminal: uncapped, imageWidth would clamp to 100 and
	// imageHeight to 66 — well over maxNativeImageHeight.
	termWidth := 200
	r := NewRenderer(theme.NewThemeForScheme(theme.Dark), termWidth).WithDocDir(dir)
	img := &ast.Image{URL: "sample.png", Alt: "Sample"}
	out := r.renderImage(img)

	wantHeightMarker := "r=" + strconv.Itoa(maxNativeImageHeight)
	if !strings.Contains(out, wantHeightMarker) {
		t.Errorf("expected the Kitty escape height capped at maxNativeImageHeight=%d, got: %.200q...", maxNativeImageHeight, out)
	}

	rows := strings.Count(out, "\n")
	if rows != maxNativeImageHeight {
		t.Errorf("expected renderImage to produce %d newlines (matching the capped r=%d row constraint) so v.Lines row-accounting matches the image's real footprint, got %d", maxNativeImageHeight, maxNativeImageHeight, rows)
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
