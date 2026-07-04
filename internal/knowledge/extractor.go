package knowledge

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Extractor extracts relationship edges from a single Document.
//
// The graph is link-based only: an edge exists between two documents when one
// actually links to the other via a markdown link (EdgeReferences, confidence
// 1.0). Prose-pattern-matched "dependency" guessing (depends-on/calls/mentions/
// implements from text like "calls X" or "depends on X") and code-import
// detection were removed — that speculative, confidence-0.7-guessed edge
// extraction belonged to bmd's earlier service-dependency-analysis direction,
// which moved to the separate graphmd project.
type Extractor struct {
	// root is the absolute path of the scanned directory.  It is used to
	// validate whether link targets exist on disk.
	root string

	// md is the goldmark instance used to parse markdown into an AST.
	md goldmark.Markdown
}

// NewExtractor creates an Extractor for documents rooted at root.
// root should be the same root directory used by ScanDirectory so that link
// targets can be resolved correctly.
func NewExtractor(root string) *Extractor {
	return &Extractor{
		root: root,
		md:   goldmark.New(),
	}
}

// Extract analyses doc and returns all discovered relationship edges.
//
// Malformed links are skipped silently (no panics).
func (ex *Extractor) Extract(doc *Document) []*Edge {
	return ex.extractLinks(doc)
}

// --- LinkExtractor -----------------------------------------------------------

// extractLinks walks the goldmark AST for doc.Content and produces a
// EdgeReferences edge for every markdown link whose destination resolves to
// another markdown file.
func (ex *Extractor) extractLinks(doc *Document) []*Edge {
	src := []byte(doc.Content)
	reader := text.NewReader(src)
	parsed := ex.md.Parser().Parse(reader)

	var edges []*Edge

	ast.Walk(parsed, func(n ast.Node, entering bool) (ast.WalkStatus, error) { //nolint:errcheck
		if !entering {
			return ast.WalkContinue, nil
		}

		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}

		dest := string(link.Destination)
		if dest == "" {
			return ast.WalkContinue, nil
		}

		// Skip non-file links (http, https, mailto, anchors, …).
		lower := strings.ToLower(dest)
		if strings.HasPrefix(lower, "http://") ||
			strings.HasPrefix(lower, "https://") ||
			strings.HasPrefix(lower, "mailto:") ||
			strings.HasPrefix(lower, "#") ||
			strings.HasPrefix(lower, "ftp://") {
			return ast.WalkContinue, nil
		}

		// Strip inline anchor fragment (#section) before resolving.
		if idx := strings.Index(dest, "#"); idx >= 0 {
			dest = dest[:idx]
		}
		if dest == "" {
			return ast.WalkContinue, nil
		}

		// Resolve to canonical relative path.
		canonical, confidence := ResolveLink(doc.RelPath, dest, ex.root)
		if canonical == "" {
			return ast.WalkContinue, nil
		}

		// Extract link text as evidence.
		var linkText strings.Builder
		for c := link.FirstChild(); c != nil; c = c.NextSibling() {
			if textNode, ok := c.(*ast.Text); ok {
				linkText.Write(textNode.Segment.Value(src))
			}
		}
		evidence := linkText.String()
		if evidence == "" {
			evidence = dest
		}

		edge, err := NewEdge(doc.ID, canonical, EdgeReferences, confidence, evidence)
		if err != nil {
			// Skip self-loops and invalid edges silently.
			return ast.WalkContinue, nil
		}

		edges = append(edges, edge)
		return ast.WalkContinue, nil
	}) //nolint:errcheck

	return edges
}

// --- ResolveLink -------------------------------------------------------------

// ResolveLink resolves a link destination found in sourcePath to a canonical
// relative path (relative to the index root).
//
// Parameters:
//   - sourcePath: relative path of the document containing the link (e.g. "services/auth.md")
//   - linkDest: the raw link destination from the markdown (e.g. "../api/gateway.md")
//   - root: absolute filesystem path of the index root directory
//
// Returns:
//   - canonical: the resolved relative path (forward slashes), or "" to signal
//     that the link should be skipped (self-reference, etc.)
//   - confidence: ConfidenceLink (1.0) when the target exists on disk, or
//     ConfidenceUnresolved (0.5) when it cannot be found
//
// Path resolution rules:
//  1. Strip query strings and anchor fragments (#...) before processing.
//  2. If linkDest is an absolute path starting with "/" it is treated as
//     relative to root.
//  3. Relative paths are resolved from the directory containing sourcePath.
//  4. The result is normalised to forward slashes.
//  5. A link that resolves to sourcePath itself (self-reference) returns "".
//  6. Circular symlinks are not followed (os.Lstat is used for existence checks).
func ResolveLink(sourcePath, linkDest, root string) (canonical string, confidence float64) {
	if linkDest == "" {
		return "", 0
	}

	// Strip anchor fragment.
	if idx := strings.Index(linkDest, "#"); idx >= 0 {
		linkDest = linkDest[:idx]
	}

	// Strip query string.
	if idx := strings.Index(linkDest, "?"); idx >= 0 {
		linkDest = linkDest[:idx]
	}

	linkDest = strings.TrimSpace(linkDest)
	if linkDest == "" {
		return "", 0
	}

	// Normalise path separators (Windows compatibility).
	linkDest = filepath.ToSlash(linkDest)
	sourcePath = filepath.ToSlash(sourcePath)

	var resolved string

	if strings.HasPrefix(linkDest, "/") {
		// Absolute-from-root: treat as relative to the root.
		resolved = path.Clean(strings.TrimPrefix(linkDest, "/"))
	} else {
		// Relative: resolve from the directory containing sourcePath.
		sourceDir := path.Dir(sourcePath)
		resolved = path.Clean(path.Join(sourceDir, linkDest))
	}

	// Reject self-references.
	if resolved == path.Clean(sourcePath) {
		return "", 0
	}

	// Check whether the resolved target exists on disk to set confidence.
	confidence = ConfidenceLink
	if root != "" {
		absTarget := filepath.Join(filepath.FromSlash(root), filepath.FromSlash(resolved))
		// Use Lstat to avoid following symlinks (circular link guard).
		if _, err := os.Lstat(absTarget); err != nil {
			confidence = ConfidenceUnresolved
		}
	}

	return resolved, confidence
}
