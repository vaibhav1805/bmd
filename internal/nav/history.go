// Package nav provides navigation history and path resolution for the markdown viewer.
package nav

// History maintains an ordered list of visited file paths with a current position
// pointer, enabling back/forward navigation similar to a web browser.
type History struct {
	entries []string
	pos     int
}

// New returns a new empty History. Current() returns "" and CanGoBack/CanGoForward
// both return false until the first Push.
func New() *History {
	return &History{pos: -1}
}

// Push adds path to the history at the current position, truncating any forward
// history that existed. The path becomes the current entry.
func (h *History) Push(path string) {
	// Truncate forward history from pos+1 onward.
	h.entries = h.entries[:h.pos+1]
	h.entries = append(h.entries, path)
	h.pos++
}

// Back moves the position back one step and returns the new current path.
// If already at the beginning (CanGoBack is false), it is a no-op and
// returns the current path (or "" if empty).
func (h *History) Back() string {
	if h.pos > 0 {
		h.pos--
	}
	return h.Current()
}

// Forward moves the position forward one step and returns the new current path.
// If already at the end (CanGoForward is false), it is a no-op and
// returns the current path (or "" if empty).
func (h *History) Forward() string {
	if h.pos < len(h.entries)-1 {
		h.pos++
	}
	return h.Current()
}

// Current returns the file path at the current position in the history,
// or "" if the history is empty.
func (h *History) Current() string {
	if h.pos < 0 {
		return ""
	}
	return h.entries[h.pos]
}

// CanGoBack returns true if there is a previous entry in the history.
func (h *History) CanGoBack() bool {
	return h.pos > 0
}

// CanGoForward returns true if there is a next entry in the history
// (i.e., the user previously went back).
func (h *History) CanGoForward() bool {
	return h.pos < len(h.entries)-1
}
