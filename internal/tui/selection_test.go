package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestStripANSI_HandlesOSCAndAPCSequences guards against a real bug found
// via manual testing: stripANSI (and everything built on the same
// escape-scanning logic — wrapLineToWidth, ansiPadOrTruncate,
// insertCursorAtVisual) only recognized CSI escapes (\x1b[...m). Inline
// image protocols use OSC (iTerm2, \x1b]...ST) and APC (Kitty,
// \x1b_...ST) instead, and their embedded base64 payload can incidentally
// contain a literal 'm' character near the start — which made the old
// CSI-only scanner stop there and treat the (huge) remainder as literal
// text, corrupting the sequence wherever it got wrapped or truncated.
func TestStripANSI_HandlesOSCAndAPCSequences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "CSI SGR code",
			in:   "\x1b[38;5;39mhello\x1b[0m world",
			want: "hello world",
		},
		{
			name: "OSC iTerm2 image sequence with incidental 'm' in payload",
			in:   "before\x1b]1337;File=name=x.png;size=4;width=1;height=1;inline=1;preserveAspectRatio=1:bWFtbQ==\x1b\\after",
			want: "beforeafter",
		},
		{
			name: "OSC terminated by BEL",
			in:   "before\x1b]1337;File=a:YQ==\x07after",
			want: "beforeafter",
		},
		{
			name: "APC Kitty graphics sequence with incidental 'm' in payload",
			in:   "before\x1b_Ga=T,f=100,m=0:bWFtbQ==\x1b\\after",
			want: "beforeafter",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripANSI(c.in)
			if got != c.want {
				t.Errorf("stripANSI(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestAnsiEscapeRuneLen_ExactBoundaries hand-verifies the shared escape
// length helper against precise expected spans.
func TestAnsiEscapeRuneLen_ExactBoundaries(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{name: "CSI SGR", in: "\x1b[38;5;39mXYZ", want: len("\x1b[38;5;39m")},
		{name: "OSC with ST terminator", in: "\x1b]1337;m=0:AA==\x1b\\XYZ", want: len("\x1b]1337;m=0:AA==\x1b\\")},
		{name: "OSC with BEL terminator", in: "\x1b]1337;m=0:AA==\x07XYZ", want: len("\x1b]1337;m=0:AA==\x07")},
		{name: "APC with ST terminator", in: "\x1b_Ga=T,m=0:AA==\x1b\\XYZ", want: len("\x1b_Ga=T,m=0:AA==\x1b\\")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runes := []rune(c.in)
			got := ansiEscapeRuneLen(runes, 0)
			if got != c.want {
				t.Errorf("ansiEscapeRuneLen(%q, 0) = %d, want %d (segment: %q)", c.in, got, c.want, string(runes[:got]))
			}
		})
	}
}

// TestAnsiPadOrTruncate_EmojiWidthAlignment is a regression test for a real
// visual bug (reported: split-pane browse mode's file-list column drifted
// horizontally, row by row, instead of forming a clean aligned column).
// The old implementation counted one rune as one visible cell, which is
// wrong for wide characters (emoji, CJK -- two cells) and zero-width
// joiners/variation selectors (zero cells) alike, so two lines with the
// same *character* count but a different mix of such runes padded to
// different actual screen widths. Confirmed via this project's own
// README.md, whose feature bullets are emoji-heavy, in the split-pane
// browser (viewer.go) and the directory-browser preview (directory.go) --
// both callers of this function.
func TestAnsiPadOrTruncate_EmojiWidthAlignment(t *testing.T) {
	const width = 20
	lines := []string{
		"plain ascii line",
		"✏️ Syntax-highlighted editing",  // pencil + variation selector (2 runes, 1 visual glyph)
		"🎨 Beautiful rendering here",     // single-codepoint emoji
		"🖱️ Mouse support (click links)", // mouse + variation selector
		"🔍 Full-text search within documents",
	}
	for _, in := range lines {
		out := ansiPadOrTruncate(in, width)
		if got := lipgloss.Width(out); got != width {
			t.Errorf("ansiPadOrTruncate(%q, %d): rendered width = %d, want %d (output %q)", in, width, got, width, out)
		}
	}
}

// TestAnsiPadOrTruncate_PadsShortPlainString covers the ordinary case: a
// short plain string pads to exactly width with trailing spaces, no
// truncation, no reset code (nothing to reset).
func TestAnsiPadOrTruncate_PadsShortPlainString(t *testing.T) {
	got := ansiPadOrTruncate("hi", 5)
	want := "hi   "
	if got != want {
		t.Errorf("ansiPadOrTruncate(%q, 5) = %q, want %q", "hi", got, want)
	}
}

// TestAnsiPadOrTruncate_NoOpWhenExactWidth covers a string already exactly
// width cells wide: returned unchanged, no padding, no truncation.
func TestAnsiPadOrTruncate_NoOpWhenExactWidth(t *testing.T) {
	in := "12345"
	got := ansiPadOrTruncate(in, 5)
	if got != in {
		t.Errorf("ansiPadOrTruncate(%q, 5) = %q, want unchanged %q", in, got, in)
	}
}

// TestAnsiPadOrTruncate_TruncatesAndResets covers a string longer than
// width: truncated to exactly width cells, with a style-reset appended so
// an unclosed color code can't bleed into whatever renders next (the
// file-list divider and column, in both real callers).
func TestAnsiPadOrTruncate_TruncatesAndResets(t *testing.T) {
	in := "\x1b[38;5;39mhello world this is long\x1b[0m"
	out := ansiPadOrTruncate(in, 10)
	if got := lipgloss.Width(out); got != 10 {
		t.Errorf("ansiPadOrTruncate truncated width = %d, want 10 (output %q)", got, out)
	}
	if !strings.HasSuffix(out, "\x1b[0m") {
		t.Errorf("expected a trailing reset code after truncation, got %q", out)
	}
}

// TestAnsiPadOrTruncate_PreservesImageEscapeIntact guards the same
// image-escape-safety property bmd-t0o fixed for the previous
// implementation: an embedded Kitty APC image payload must survive
// verbatim, never truncated mid-sequence, regardless of what visible text
// follows it.
func TestAnsiPadOrTruncate_PreservesImageEscapeIntact(t *testing.T) {
	kitty := "\x1b_Gf=100,a=T;QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=\x1b\\"
	in := kitty + "hello world, more text after the image escape"
	out := ansiPadOrTruncate(in, 5)
	if !strings.HasPrefix(out, kitty) {
		t.Errorf("expected the Kitty APC escape to survive intact at the start of the output, got %q", out)
	}
}
