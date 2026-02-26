package nav

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveLink resolves a relative markdown link href against the directory of
// currentFile, enforcing the following security and correctness constraints:
//
//   - Rejects http:// and https:// links (external links not supported)
//   - Rejects "#..." anchor links
//   - Requires the href to end in ".md"
//   - The resolved path must stay within startDir (no traversal above it)
//   - The resolved path must exist as a regular file (not a symlink)
//
// Returns the absolute, clean resolved path on success, or a descriptive error.
func ResolveLink(currentFile, href, startDir string) (string, error) {
	// 1. Reject external links.
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return "", fmt.Errorf("external link not supported: %s", href)
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
