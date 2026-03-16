package nav_test

import (
	"testing"

	"github.com/bmd/bmd/internal/nav"
)

func TestHistory_New(t *testing.T) {
	h := nav.New()
	if h.Current() != "" {
		t.Errorf("New(): Current() = %q, want %q", h.Current(), "")
	}
	if h.CanGoBack() {
		t.Error("New(): CanGoBack() = true, want false")
	}
	if h.CanGoForward() {
		t.Error("New(): CanGoForward() = true, want false")
	}
}

func TestHistory_PushSingle(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	if h.Current() != "a.md" {
		t.Errorf("Push(a.md): Current() = %q, want %q", h.Current(), "a.md")
	}
	if h.CanGoBack() {
		t.Error("Push(a.md): CanGoBack() = true, want false")
	}
	if h.CanGoForward() {
		t.Error("Push(a.md): CanGoForward() = true, want false")
	}
}

func TestHistory_PushTwo_Back(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	h.Push("b.md")
	if h.Current() != "b.md" {
		t.Errorf("Push x2: Current() = %q, want %q", h.Current(), "b.md")
	}
	if !h.CanGoBack() {
		t.Error("Push x2: CanGoBack() = false, want true")
	}
	result := h.Back()
	if result != "a.md" {
		t.Errorf("Back(): got %q, want %q", result, "a.md")
	}
	if h.Current() != "a.md" {
		t.Errorf("After Back(): Current() = %q, want %q", h.Current(), "a.md")
	}
	if !h.CanGoForward() {
		t.Error("After Back(): CanGoForward() = false, want true")
	}
}

func TestHistory_Forward(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	h.Push("b.md")
	h.Back()
	result := h.Forward()
	if result != "b.md" {
		t.Errorf("Forward(): got %q, want %q", result, "b.md")
	}
	if h.Current() != "b.md" {
		t.Errorf("After Forward(): Current() = %q, want %q", h.Current(), "b.md")
	}
	if h.CanGoForward() {
		t.Error("After Forward(): CanGoForward() = true, want false")
	}
}

func TestHistory_PushAfterBack_TruncatesForward(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	h.Push("b.md")
	h.Back() // now at a.md
	h.Push("c.md")
	if h.Current() != "c.md" {
		t.Errorf("Push after Back: Current() = %q, want %q", h.Current(), "c.md")
	}
	if h.CanGoForward() {
		t.Error("Push after Back: CanGoForward() = true, want false (forward history truncated)")
	}
}

func TestHistory_BackNoOp(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	// CanGoBack is false, Back() should be no-op returning current
	result := h.Back()
	if result != "a.md" {
		t.Errorf("Back() when CanGoBack=false: got %q, want %q", result, "a.md")
	}
	if h.Current() != "a.md" {
		t.Errorf("Back() no-op: Current() changed to %q", h.Current())
	}
}

func TestHistory_ForwardNoOp(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	h.Push("b.md")
	// CanGoForward is false, Forward() should be no-op returning current
	result := h.Forward()
	if result != "b.md" {
		t.Errorf("Forward() when CanGoForward=false: got %q, want %q", result, "b.md")
	}
	if h.Current() != "b.md" {
		t.Errorf("Forward() no-op: Current() changed to %q", h.Current())
	}
}

func TestHistory_PushSamePath(t *testing.T) {
	h := nav.New()
	h.Push("a.md")
	h.Push("a.md")
	if !h.CanGoBack() {
		t.Error("Push same path twice: CanGoBack() = false, want true (allows revisiting)")
	}
	if h.Current() != "a.md" {
		t.Errorf("Push same path twice: Current() = %q, want %q", h.Current(), "a.md")
	}
}

func TestHistory_BackOnEmpty(t *testing.T) {
	h := nav.New()
	// Back() on empty history: returns ""
	result := h.Back()
	if result != "" {
		t.Errorf("Back() on empty: got %q, want %q", result, "")
	}
}

func TestHistory_ForwardOnEmpty(t *testing.T) {
	h := nav.New()
	// Forward() on empty history: returns ""
	result := h.Forward()
	if result != "" {
		t.Errorf("Forward() on empty: got %q, want %q", result, "")
	}
}
