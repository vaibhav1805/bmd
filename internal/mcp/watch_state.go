package mcp

import (
	"fmt"
	"sync"
	"time"

	"github.com/bmd/bmd/internal/knowledge"
)

// WatchNotification is a single change event delivered to an agent on poll.
type WatchNotification struct {
	Kind    string `json:"kind"`     // "created" | "modified" | "deleted"
	Path    string `json:"path"`     // absolute path
	RelPath string `json:"rel_path"` // directory-relative path
}

// WatchSession tracks an active filesystem watch for one MCP client.
type WatchSession struct {
	ID        string    `json:"session_id"`
	Dir       string    `json:"dir"`
	StartedAt time.Time `json:"started_at"`

	updater *knowledge.IncrementalUpdater
	mu      sync.Mutex
	pending []WatchNotification // buffered notifications since last poll
}

// append adds a WatchEvent to the pending notification queue.
// It is safe to call from the onChange callback goroutine.
func (s *WatchSession) append(evt knowledge.WatchEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, WatchNotification{
		Kind:    watchEventKindToString(evt.Kind),
		Path:    evt.Path,
		RelPath: evt.RelPath,
	})
}

// DrainPending atomically replaces the pending queue with an empty slice and
// returns all notifications that were waiting. Returns an empty (non-nil)
// slice when no notifications are pending.
func (s *WatchSession) DrainPending() []WatchNotification {
	s.mu.Lock()
	defer s.mu.Unlock()
	drained := s.pending
	s.pending = []WatchNotification{}
	if drained == nil {
		drained = []WatchNotification{}
	}
	return drained
}

// watchEventKindToString converts a WatchEventKind to its string representation.
func watchEventKindToString(k knowledge.WatchEventKind) string {
	switch k {
	case knowledge.WatchCreated:
		return "created"
	case knowledge.WatchModified:
		return "modified"
	case knowledge.WatchDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// WatchSessionManager is a concurrent-safe store of active WatchSessions.
// Each MCP client gets its own isolated session so that multiple agents
// can watch the same directory independently.
type WatchSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*WatchSession
}

// NewWatchSessionManager returns an empty, ready-to-use WatchSessionManager.
func NewWatchSessionManager() *WatchSessionManager {
	return &WatchSessionManager{
		sessions: make(map[string]*WatchSession),
	}
}

// Create registers a new WatchSession for the given directory and incremental
// updater. The onChange callback on updater is wired to append notifications
// to the session's pending queue. Returns the created session.
func (m *WatchSessionManager) Create(dir string, updater *knowledge.IncrementalUpdater) *WatchSession {
	sessionID := fmt.Sprintf("ws-%d", time.Now().UnixNano())
	session := &WatchSession{
		ID:        sessionID,
		Dir:       dir,
		StartedAt: time.Now(),
		updater:   updater,
		pending:   []WatchNotification{},
	}

	// Wire the updater's onChange callback to populate the session's queue.
	updater.SetOnChange(func(evt knowledge.WatchEvent) {
		session.append(evt)
	})

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session
}

// Get returns the WatchSession with the given id, or (nil, false) if not found.
func (m *WatchSessionManager) Get(id string) (*WatchSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Delete terminates the session's updater (and watcher) and removes the
// session from the manager. It is a no-op if the id does not exist.
func (m *WatchSessionManager) Delete(id string) {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if ok && session.updater != nil {
		session.updater.Stop()
	}
}
