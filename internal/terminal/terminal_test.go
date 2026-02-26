package terminal

import (
	"strings"
	"testing"
)

func TestDetectTerminalWidth_ReturnsPositive(t *testing.T) {
	width := DetectTerminalWidth()
	if width < 40 {
		t.Errorf("DetectTerminalWidth returned %d, expected >= 40", width)
	}
	// Also verify it's a reasonable value (not absurdly large)
	if width > 10000 {
		t.Errorf("DetectTerminalWidth returned unreasonably large value: %d", width)
	}
}

func TestWrapText_ShortLine_Unchanged(t *testing.T) {
	text := "Short line"
	result := WrapText(text, 80)
	if result != text {
		t.Errorf("Short line should be unchanged: got %q", result)
	}
}

func TestWrapText_ExceedsWidth(t *testing.T) {
	text := "This is a long line that should definitely be wrapped at the specified width boundary"
	result := WrapText(text, 40)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if len(line) > 40 {
			t.Errorf("Line exceeds width 40: %q (len=%d)", line, len(line))
		}
	}
	if len(lines) < 2 {
		t.Errorf("Expected wrapping to create multiple lines, got: %q", result)
	}
}

func TestWrapText_PreservesExistingNewlines(t *testing.T) {
	text := "Line one\nLine two\nLine three"
	result := WrapText(text, 80)
	if !strings.Contains(result, "Line one") {
		t.Errorf("Expected Line one in result: %q", result)
	}
	if !strings.Contains(result, "Line two") {
		t.Errorf("Expected Line two in result: %q", result)
	}
	if !strings.Contains(result, "Line three") {
		t.Errorf("Expected Line three in result: %q", result)
	}
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines, got %d", len(lines))
	}
}

func TestWrapText_DoesNotBreakMidWord(t *testing.T) {
	// If all words fit without breaking, they shouldn't be split
	text := "word1 word2 word3 word4 word5"
	result := WrapText(text, 20)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		words := strings.Fields(line)
		for _, word := range words {
			// Each word should be intact (same as original words)
			found := false
			for _, original := range []string{"word1", "word2", "word3", "word4", "word5"} {
				if word == original {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Word %q appears broken/modified in output", word)
			}
		}
	}
}

func TestWrapText_ZeroWidth_Unchanged(t *testing.T) {
	text := "This should not be wrapped"
	result := WrapText(text, 0)
	if result != text {
		t.Errorf("Width 0 should return text unchanged, got: %q", result)
	}
}

func TestWrapText_EmptyString(t *testing.T) {
	result := WrapText("", 80)
	if result != "" {
		t.Errorf("Expected empty result for empty input, got: %q", result)
	}
}

func TestWrapLine_ExactWidth(t *testing.T) {
	// A line of exactly width characters should not be wrapped
	text := strings.Repeat("a", 80)
	result := wrapLine(text, 80)
	lines := strings.Split(result, "\n")
	if len(lines) != 1 {
		t.Errorf("Line of exactly width=%d should not wrap, got %d lines", 80, len(lines))
	}
}
