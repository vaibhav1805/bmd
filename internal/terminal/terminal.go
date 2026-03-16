// Package terminal provides utilities for terminal detection and text wrapping.
package terminal

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// winsize is the struct used by the ioctl TIOCGWINSZ syscall.
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// DetectTerminalWidth returns the current terminal width in columns.
//
// Detection priority:
//  1. ioctl TIOCGWINSZ syscall on stdout
//  2. COLUMNS environment variable
//  3. Default of 80 columns
func DetectTerminalWidth() int {
	// Try ioctl on stdout (fd 1)
	ws := winsize{}
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	); errno == 0 && ws.Col > 0 {
		return int(ws.Col)
	}

	// Fallback: COLUMNS env var
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}

	// Default: 80
	return 80
}

// WrapText wraps text to the specified column width, preserving word boundaries.
//
// Rules:
//   - Existing newlines in text are preserved (each line is independently wrapped)
//   - Words are not broken mid-word
//   - If a single word is longer than width, it appears on its own line
//   - Width <= 0 returns text unchanged
func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	// Process each existing line separately
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, wrapLine(line, width))
	}
	return strings.Join(result, "\n")
}

// wrapLine wraps a single line (no existing newlines) to the given width.
func wrapLine(line string, width int) string {
	if len(line) <= width {
		return line
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}

	var sb strings.Builder
	currentLen := 0

	for i, word := range words {
		wordLen := len(word)
		if i == 0 {
			sb.WriteString(word)
			currentLen = wordLen
		} else {
			// Would adding this word (with a space) exceed the width?
			if currentLen+1+wordLen > width {
				// Start a new line
				sb.WriteString("\n")
				sb.WriteString(word)
				currentLen = wordLen
			} else {
				sb.WriteString(" ")
				sb.WriteString(word)
				currentLen += 1 + wordLen
			}
		}
	}

	return sb.String()
}
