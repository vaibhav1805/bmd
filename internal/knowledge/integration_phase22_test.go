package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// buildIntegrationMonorepo creates a fully wired test monorepo directory with
// markdown files that reference each other and package markers so that the
// full Phase 22 pipeline (DiscoverComponents → BuildComponentGraph → BFS) works
// end-to-end without external dependencies.
//
// Layout:
//
//	root/
//	  services/payment/
//	    go.mod
//	    README.md        (calls auth and user)
//	  services/auth/
//	    go.mod
//	    README.md        (standalone)
//	  services/user/
//	    go.mod
//	    README.md        (standalone)
func buildIntegrationMonorepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	mustMkdir(t, root, "services/payment")
	mustWriteFile(t, root, "services/payment/go.mod", "module payment")
	mustWriteFile(t, root, "services/payment/README.md",
		"# Payment Service\n\nCalls `auth` service for authentication.\nAlso depends on the user service.\n")

	mustMkdir(t, root, "services/auth")
	mustWriteFile(t, root, "services/auth/go.mod", "module auth")
	mustWriteFile(t, root, "services/auth/README.md",
		"# Auth Service\n\nHandles authentication tokens.\n")

	mustMkdir(t, root, "services/user")
	mustWriteFile(t, root, "services/user/go.mod", "module user")
	mustWriteFile(t, root, "services/user/README.md",
		"# User Service\n\nManages user profiles.\n")

	return root
}

// ─── Integration Test 1: Discovery → ComponentGraph pipeline ──────────────────

func TestIntegration_DiscoverAndBuildGraph(t *testing.T) {
	root := buildIntegrationMonorepo(t)

	// Step 1: discover components.
	discovered, err := DiscoverComponents("", root, false)
	if err != nil {
		t.Fatalf("DiscoverComponents: %v", err)
	}
	if len(discovered) < 3 {
		t.Fatalf("expected >= 3 components, got %d: %v", len(discovered), discovered)
	}

	// Step 2: build file-level graph (empty — just for structure).
	fileGraph := NewGraph()
	for _, dc := range discovered {
		docPath := filepath.Join(dc.Path, "README.md")
		relPath, _ := filepath.Rel(root, docPath)
		_ = fileGraph.AddNode(&Node{ID: relPath, Title: dc.Name})
	}

	// Step 3: convert discovered → Component slice.
	comps := make([]Component, len(discovered))
	for i, dc := range discovered {
		relPath, _ := filepath.Rel(root, filepath.Join(dc.Path, "README.md"))
		comps[i] = Component{
			ID:         dc.Name,
			Name:       dc.Name,
			File:       filepath.ToSlash(relPath),
			Confidence: ConfidenceComponentFilename,
		}
	}

	// Step 4: build component graph.
	cg, err := BuildComponentGraph(comps, fileGraph, nil)
	if err != nil {
		t.Fatalf("BuildComponentGraph: %v", err)
	}
	if cg.NodeCount() < 3 {
		t.Errorf("NodeCount = %d, want >= 3", cg.NodeCount())
	}
}

// ─── Integration Test 2: BFS traversal produces DebugContext ──────────────────

func TestIntegration_BFSProducesDebugContext(t *testing.T) {
	tmpDir := t.TempDir()
	cg := buildDebugTestGraph(t, tmpDir)

	bfs, err := NewBFS(cg, "payment", tmpDir)
	if err != nil {
		t.Fatalf("NewBFS: %v", err)
	}

	if err := bfs.Traverse(3, 1024*1024); err != nil {
		t.Fatalf("Traverse: %v", err)
	}

	dc := bfs.BuildDebugContext("payment", "integration test query")

	// DebugContext should have 3 components (payment + auth + user).
	if len(dc.Components) < 3 {
		t.Errorf("Components count = %d, want >= 3", len(dc.Components))
	}

	// Documentation should be loaded from actual files.
	if dc.Stats.DocumentationSize == 0 {
		t.Error("Stats.DocumentationSize = 0, expected documentation from real files")
	}
}

// ─── Integration Test 3: ToJSON round-trips to valid STATUS-01 envelope ───────

func TestIntegration_ToJSONRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cg := buildDebugTestGraph(t, tmpDir)

	bfs, _ := NewBFS(cg, "payment", tmpDir)
	_ = bfs.Traverse(3, 1024*1024)
	dc := bfs.BuildDebugContext("payment", "round-trip test")

	raw, err := dc.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	// Unmarshal into a generic map to verify shape without strict coupling.
	var envelope map[string]interface{}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("ToJSON output is not valid JSON: %v\n\nOutput:\n%s", err, raw)
	}

	for _, field := range []string{"status", "code", "message"} {
		if _, ok := envelope[field]; !ok {
			t.Errorf("envelope missing field %q", field)
		}
	}
}

// ─── Integration Test 4: ComponentGraph + CmdDebug integration ────────────────

func TestIntegration_ParseDebugArgs_Required(t *testing.T) {
	// No --component should return an error.
	_, err := ParseDebugArgs([]string{"--dir", "."})
	if err == nil {
		t.Fatal("expected error when --component is missing, got nil")
	}
}

func TestIntegration_ParseDebugArgs_Valid(t *testing.T) {
	a, err := ParseDebugArgs([]string{"--component", "payment", "--depth", "3", "--format", "json"})
	if err != nil {
		t.Fatalf("ParseDebugArgs: %v", err)
	}
	if a.Component != "payment" {
		t.Errorf("Component = %q, want 'payment'", a.Component)
	}
	if a.Depth != 3 {
		t.Errorf("Depth = %d, want 3", a.Depth)
	}
}

func TestIntegration_ParseDebugArgs_PositionalComponent(t *testing.T) {
	a, err := ParseDebugArgs([]string{"payment"})
	if err != nil {
		t.Fatalf("ParseDebugArgs positional: %v", err)
	}
	if a.Component != "payment" {
		t.Errorf("Component = %q, want 'payment' from positional arg", a.Component)
	}
}

// ─── Integration Test 5: ParseComponentsGraphArgs ─────────────────────────────

func TestIntegration_ParseComponentsGraphArgs_Defaults(t *testing.T) {
	a, err := ParseComponentsGraphArgs([]string{})
	if err != nil {
		t.Fatalf("ParseComponentsGraphArgs: %v", err)
	}
	if a.Dir != "." {
		t.Errorf("Dir default = %q, want '.'", a.Dir)
	}
	if a.Format != "ascii" {
		t.Errorf("Format default = %q, want 'ascii'", a.Format)
	}
}

func TestIntegration_ParseComponentsGraphArgs_Flags(t *testing.T) {
	a, err := ParseComponentsGraphArgs([]string{"--dir", "/tmp/myrepo", "--format", "json"})
	if err != nil {
		t.Fatalf("ParseComponentsGraphArgs: %v", err)
	}
	if a.Dir != "/tmp/myrepo" {
		t.Errorf("Dir = %q, want '/tmp/myrepo'", a.Dir)
	}
	if a.Format != "json" {
		t.Errorf("Format = %q, want 'json'", a.Format)
	}
}

// ─── Integration Test 6: formatComponentGraphASCII ────────────────────────────

func TestIntegration_FormatComponentGraphASCII_NoEdges(t *testing.T) {
	comps := []Component{
		{ID: "svc", Name: "Svc", File: "svc.md"},
	}
	cg := NewComponentGraph(comps)
	out := formatComponentGraphASCII(cg)
	if out == "" {
		t.Error("expected non-empty ASCII output even for graph with no edges")
	}
}

func TestIntegration_FormatComponentGraphASCII_WithEdges(t *testing.T) {
	comps := []Component{
		{ID: "a", Name: "A", File: "a.md"},
		{ID: "b", Name: "B", File: "b.md"},
	}
	cg := NewComponentGraph(comps)
	_ = cg.AddEdge("a", "b", 0.85, "depends_on", nil)

	out := formatComponentGraphASCII(cg)
	if out == "" {
		t.Error("expected non-empty ASCII output for graph with edges")
	}
	// Should contain the arrow character.
	if !containsRune(out, '→') {
		t.Errorf("ASCII graph output should contain '→', got: %s", out)
	}
}

// ─── Integration Test 7: full pipeline with real temp dir ─────────────────────

func TestIntegration_BuildComponentGraphFromConfig_NoDocError(t *testing.T) {
	root := t.TempDir()
	// Empty directory — no markdown files, so scan returns 0 docs.
	_, err := BuildComponentGraphFromConfig(root, nil)
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
}

func TestIntegration_NewGraph_EmptyIsValid(t *testing.T) {
	g := NewGraph()
	if g.NodeCount() != 0 {
		t.Errorf("empty graph NodeCount = %d, want 0", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("empty graph EdgeCount = %d, want 0", g.EdgeCount())
	}
}

// ─── Integration Test 8: FileToComponent populated after MapFilesToComponents ──

func TestIntegration_FileToComponent_Populated(t *testing.T) {
	comps := []Component{
		{ID: "api", Name: "API", File: "services/api/README.md", Confidence: ConfidenceConfigured},
		{ID: "worker", Name: "Worker", File: "services/worker/README.md", Confidence: ConfidenceConfigured},
	}
	fileGraph := NewGraph()
	allFiles := []string{
		"services/api/README.md",
		"services/api/docs.md",
		"services/worker/README.md",
	}
	for _, f := range allFiles {
		_ = fileGraph.AddNode(&Node{ID: f})
	}

	cg, err := BuildComponentGraph(comps, fileGraph, nil)
	if err != nil {
		t.Fatalf("BuildComponentGraph: %v", err)
	}

	if comp, ok := cg.FileToComponent["services/api/README.md"]; !ok || comp != "api" {
		t.Errorf("FileToComponent['services/api/README.md'] = %q, want 'api'", comp)
	}
	if comp, ok := cg.FileToComponent["services/worker/README.md"]; !ok || comp != "worker" {
		t.Errorf("FileToComponent['services/worker/README.md'] = %q, want 'worker'", comp)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// buildTestMarkdownDir creates a temp directory with the given filename→content
// map and returns the root.
func buildTestMarkdownDir(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}
