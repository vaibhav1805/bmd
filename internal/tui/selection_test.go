package tui

import "testing"

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
