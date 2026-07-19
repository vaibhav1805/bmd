package tui

import (
	"errors"
	"os"
	"testing"
)

// breakStderr temporarily replaces os.Stderr with the write end of a pipe
// whose read end has already been closed, forcing any subsequent write to
// os.Stderr to return a "broken pipe" error. It returns a restore function
// that must be deferred by the caller. This lets tests deterministically
// force the OSC52 branch of copyWithFallback to fail without depending on a
// real terminal.
func breakStderr(t *testing.T) (restore func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close pipe read end: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	return func() {
		os.Stderr = orig
		_ = w.Close()
	}
}

// TestCopyWithFallback_OSC52Success verifies that when the OSC52 write
// succeeds (returns a nil error), copyWithFallback returns
// (usedFallback=false, err=nil) and does not invoke the native fallback at
// all.
func TestCopyWithFallback_OSC52Success(t *testing.T) {
	orig := writeAllFn
	defer func() { writeAllFn = orig }()

	called := false
	writeAllFn = func(text string) error {
		called = true
		return nil
	}

	// os.Stderr is left untouched here, so the OSC52 write succeeds.
	usedFallback, err := copyWithFallback("hello world")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if usedFallback {
		t.Fatalf("expected usedFallback=false, got true")
	}
	if called {
		t.Fatalf("expected writeAllFn NOT to be called on OSC52 success")
	}
}

// TestCopyWithFallback_OSC52Failure verifies that when the OSC52 write fails,
// copyWithFallback calls the (stubbed) native fallback and returns its
// (successful) result.
func TestCopyWithFallback_OSC52Failure(t *testing.T) {
	restore := breakStderr(t)
	defer restore()

	orig := writeAllFn
	defer func() { writeAllFn = orig }()

	called := false
	writeAllFn = func(text string) error {
		called = true
		return nil
	}

	usedFallback, err := copyWithFallback("hello world")
	if err != nil {
		t.Fatalf("expected nil error when fallback succeeds, got %v", err)
	}
	if !usedFallback {
		t.Fatalf("expected usedFallback=true when OSC52 write fails")
	}
	if !called {
		t.Fatalf("expected writeAllFn to be called when OSC52 write fails")
	}
}

// TestCopyWithFallback_BothFail verifies that when both OSC52 and the native
// fallback fail, copyWithFallback returns a non-nil error and does not
// panic.
func TestCopyWithFallback_BothFail(t *testing.T) {
	restore := breakStderr(t)
	defer restore()

	orig := writeAllFn
	defer func() { writeAllFn = orig }()

	stubErr := errors.New("no clipboard tool found")
	writeAllFn = func(text string) error {
		return stubErr
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("copyWithFallback panicked: %v", r)
		}
	}()

	usedFallback, err := copyWithFallback("hello world")
	if err == nil {
		t.Fatalf("expected non-nil error when both OSC52 and fallback fail")
	}
	if !usedFallback {
		t.Fatalf("expected usedFallback=true even when the fallback itself fails")
	}
}
