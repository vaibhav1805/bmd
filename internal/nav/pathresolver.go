package nav

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// IsExternalURL checks if the href is a web URL.
func IsExternalURL(href string) bool {
	return strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")
}

// OpenURL opens the given URL in the default web browser.
// Supports macOS, Linux (xdg-open, gnome-open), and Windows.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// macOS: use 'open'
		cmd = exec.Command("open", url)
	case "linux":
		// Linux: try xdg-open first, then gnome-open as fallback
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", url)
		} else if _, err := exec.LookPath("gnome-open"); err == nil {
			cmd = exec.Command("gnome-open", url)
		} else {
			return fmt.Errorf("no browser launcher found (try installing xdg-open or gnome-open)")
		}
	case "windows":
		// Windows: use 'start' (cmd.exe /c start)
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// ResolveLink resolves a relative markdown link href against the directory of
// currentFile, enforcing the following security and correctness constraints:
//
//   - Accepts http:// and https:// links (returns special marker)
//   - Rejects "#..." anchor links
//   - Requires local links to end in ".md"
//   - The resolved path must stay within startDir (no traversal above it)
//   - The resolved path must exist as a regular file (not a symlink)
//
// For external URLs, returns the URL prefixed with "external://" marker.
// For local files, returns the absolute, clean resolved path.
// Returns descriptive error on validation failure.
func ResolveLink(currentFile, href, startDir string) (string, error) {
	// 1. Allow external links - mark them for special handling
	if IsExternalURL(href) {
		return "external://" + href, nil
	}

	// 2. Reject anchor links.
	if strings.HasPrefix(href, "#") {
		return "", fmt.Errorf("anchor links not supported: %s", href)
	}

	// 3. Require .md extension.
	if !strings.HasSuffix(href, ".md") {
		return "", fmt.Errorf("link must point to a .md file, got: %s", href)
	}

	// 4. Resolve the href relative to the directory of currentFile.
	base := filepath.Dir(currentFile)
	resolved := filepath.Clean(filepath.Join(base, href))

	// 5. Prevent path traversal above startDir.
	cleanStart := filepath.Clean(startDir)
	// Add a trailing separator to prevent a prefix match on a sibling directory
	// that happens to share a prefix string (e.g., /docs matches /docs-extra).
	if !strings.HasPrefix(resolved, cleanStart+string(filepath.Separator)) && resolved != cleanStart {
		return "", fmt.Errorf("path traversal not allowed: %s escapes start directory", href)
	}

	// 6. Check that the file exists (Lstat does not follow symlinks).
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", resolved)
	}

	// 7. Reject symlinks.
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlinks not allowed: %s", resolved)
	}

	return resolved, nil
}
