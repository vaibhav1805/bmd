// Package tui provides the interactive terminal user interface for bmd.
package tui

import "strings"

// Sentinel delimiters embedded in rendered output to mark link positions.
// These are control characters not used in normal markdown text.
const (
	linkSentinelPrefix = "\x00LINK:"
	linkSentinelSep    = "\x00"
	linkSentinelEnd    = "\x00/LINK\x00"
)

// LinkEntry records a link found in the rendered output.
type LinkEntry struct {
	LineIndex int    // 0-based line in rendered output
	URL       string // the href value
}

// LinkRegistry maps rendered line positions to link URLs and tracks focus state.
type LinkRegistry struct {
	Links   []LinkEntry // ordered by first appearance in output
	focused int         // index into Links (-1 = none focused)
}

// BuildRegistry scans rendered lines for link sentinel patterns and builds a
// registry of all links found. Lines are searched in order; multiple links on
// the same line each get a separate entry (using the same LineIndex).
func BuildRegistry(lines []string) LinkRegistry {
	reg := LinkRegistry{focused: -1}
	for lineIdx, line := range lines {
		rest := line
		for {
			start := strings.Index(rest, linkSentinelPrefix)
			if start == -1 {
				break
			}
			// Find the separator after the prefix (marks end of URL)
			urlStart := start + len(linkSentinelPrefix)
			sepIdx := strings.Index(rest[urlStart:], linkSentinelSep)
			if sepIdx == -1 {
				break
			}
			url := rest[urlStart : urlStart+sepIdx]
			if url != "" {
				reg.Links = append(reg.Links, LinkEntry{
					LineIndex: lineIdx,
					URL:       url,
				})
			}
			// Advance past the end sentinel to find more links on this line
			endIdx := strings.Index(rest[urlStart+sepIdx:], linkSentinelEnd)
			if endIdx == -1 {
				break
			}
			rest = rest[urlStart+sepIdx+endIdx+len(linkSentinelEnd):]
		}
	}
	return reg
}

// StripSentinels removes all link sentinel markers from a rendered line,
// returning only the visible text. Used before displaying lines to the user.
func StripSentinels(line string) string {
	result := line
	for {
		start := strings.Index(result, linkSentinelPrefix)
		if start == -1 {
			break
		}
		urlStart := start + len(linkSentinelPrefix)
		sepIdx := strings.Index(result[urlStart:], linkSentinelSep)
		if sepIdx == -1 {
			break
		}
		// Remove the prefix+url+sep
		result = result[:start] + result[urlStart+sepIdx+len(linkSentinelSep):]

		// Remove the end sentinel if present
		endIdx := strings.Index(result, linkSentinelEnd)
		if endIdx == -1 {
			break
		}
		result = result[:endIdx] + result[endIdx+len(linkSentinelEnd):]
	}
	return result
}

// FocusNext advances focus to the next link (wrapping around) and returns the
// new focused index. Returns -1 if no links are registered.
func (r *LinkRegistry) FocusNext() int {
	if len(r.Links) == 0 {
		return -1
	}
	if r.focused < 0 {
		r.focused = 0
	} else {
		r.focused = (r.focused + 1) % len(r.Links)
	}
	return r.focused
}

// FocusPrev moves focus to the previous link (wrapping around) and returns the
// new focused index. Returns -1 if no links are registered.
func (r *LinkRegistry) FocusPrev() int {
	if len(r.Links) == 0 {
		return -1
	}
	if r.focused < 0 {
		r.focused = len(r.Links) - 1
	} else {
		r.focused = (r.focused - 1 + len(r.Links)) % len(r.Links)
	}
	return r.focused
}

// FocusedURL returns the URL of the currently focused link, or "" if nothing
// is focused.
func (r *LinkRegistry) FocusedURL() string {
	if r.focused < 0 || r.focused >= len(r.Links) {
		return ""
	}
	return r.Links[r.focused].URL
}

// FocusedLine returns the line index (0-based) of the currently focused link,
// or -1 if nothing is focused.
func (r *LinkRegistry) FocusedLine() int {
	if r.focused < 0 || r.focused >= len(r.Links) {
		return -1
	}
	return r.Links[r.focused].LineIndex
}

// Clear removes focus without changing the link list.
func (r *LinkRegistry) Clear() {
	r.focused = -1
}

// Focused returns the current focused index (-1 if none).
func (r *LinkRegistry) Focused() int {
	return r.focused
}
