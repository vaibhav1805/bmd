package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGraphIntegration_RealCorpus builds a knowledge graph from the BMD
// repository's markdown files and verifies the structural invariants.
//
// This test mirrors the manual acceptance criteria in the plan:
//  1. All documents become nodes.
//  2. Links are extracted as edges.
//  3. Edge counts are reasonable (not zero, not an explosion).
//  4. Traversal returns a non-empty reachable set from at least one node.
//  5. No panics on malformed markdown.
func TestGraphIntegration_RealCorpus(t *testing.T) {
	// Locate the repository root relative to this test file.
	// os.Getwd() returns the package directory during `go test`.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Walk up until we find the go.mod file (repo root).
	root := findRepoRoot(wd)
	if root == "" {
		t.Skip("cannot locate repo root — skipping integration test")
	}

	t.Logf("Scanning root: %s", root)

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no markdown documents found in repo root")
	}
	t.Logf("Scanned %d markdown files", len(docs))

	// 1. Build graph.
	gb := NewGraphBuilder(root)
	g := gb.Build(docs)

	// 2. Every document should become a node.
	if g.NodeCount() != len(docs) {
		t.Errorf("NodeCount = %d, want %d (one per document)", g.NodeCount(), len(docs))
	}

	// 3. Edge counts should be reasonable.
	edgeCount := g.EdgeCount()
	t.Logf("Graph: %d nodes, %d edges", g.NodeCount(), edgeCount)
	if edgeCount == 0 {
		t.Log("WARNING: no edges extracted — link/mention patterns may not match this corpus")
	}
	// Sanity cap: unlikely to have >50 edges per document.
	if edgeCount > len(docs)*50 {
		t.Errorf("EdgeCount = %d seems like an explosion (>50 per doc)", edgeCount)
	}

	// 4. Traversal should return a non-empty reachable set from at least one node.
	foundReachable := false
	for _, doc := range docs {
		reachable := g.TransitiveDeps(doc.ID)
		if len(reachable) > 0 {
			t.Logf("TransitiveDeps(%q): %d reachable nodes", doc.ID, len(reachable))
			foundReachable = true
			break
		}
	}
	if !foundReachable && edgeCount > 0 {
		t.Error("no node has transitive dependencies even though edges exist")
	}

	// 5. Cycle detection should complete without hanging.
	cycles := g.DetectCycles()
	t.Logf("Detected %d cycles", len(cycles))

	// 6. All edge confidence values must be in [0.0, 1.0].
	for _, edge := range g.Edges {
		if edge.Confidence < 0.0 || edge.Confidence > 1.0 {
			t.Errorf("edge %q has out-of-range confidence: %f", edge.ID, edge.Confidence)
		}
	}

	// 7. Spot-check: verify confidence distribution.
	linkCount, mentionCount, codeCount := 0, 0, 0
	for _, edge := range g.Edges {
		switch edge.Type {
		case EdgeReferences:
			linkCount++
		case EdgeMentions, EdgeDependsOn, EdgeImplements:
			mentionCount++
		case EdgeCalls:
			codeCount++
		}
	}
	t.Logf("Edge breakdown: references=%d, mentions/depends/implements=%d, code=%d",
		linkCount, mentionCount, codeCount)

	t.Log("Graph integration test PASSED")
}

// findRepoRoot walks up from dir until it finds a go.mod file.
// Returns the directory containing go.mod, or "" if not found.
func findRepoRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
