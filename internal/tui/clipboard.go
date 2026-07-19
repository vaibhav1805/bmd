package tui

import (
	"os"

	"github.com/atotto/clipboard"
	osc52 "github.com/aymanbagabas/go-osc52/v2"
)

// writeAllFn is a package-level seam over clipboard.WriteAll so tests can
// intercept the native clipboard call without depending on OS tools being
// installed on the machine running the tests.
var writeAllFn = clipboard.WriteAll

// copyWithFallback writes text to the clipboard via OSC52 first (D-01); only
// if that write itself errors does it fall back to the native OS clipboard
// tool via atotto/clipboard (D-02).
//
// Known limitation (D-01, accepted): osc52.New(text).WriteTo(os.Stderr) writes
// bytes to stderr and will almost always return a nil error even on terminals
// that silently ignore/don't understand the OSC52 escape sequence — a nil
// error here does not prove the terminal actually received a usable clipboard
// write. This means the native fallback triggers less often in practice than
// a heuristic terminal-capability check would. This tradeoff was explicitly
// chosen by the user; do not "fix" it with heuristic capability detection
// without checking back with the user first.
func copyWithFallback(text string) (usedFallback bool, err error) {
	if _, werr := osc52.New(text).WriteTo(os.Stderr); werr == nil {
		return false, nil
	}
	if werr := writeAllFn(text); werr != nil {
		// writeAllFn (clipboard.WriteAll) returns a normal Go error
		// (e.g. clipboard's own "missingCommands" error when no native
		// clipboard tool is found on the platform) rather than panicking.
		return true, werr
	}
	return true, nil
}
