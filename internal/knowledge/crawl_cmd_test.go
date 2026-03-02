package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── argument parsing tests ───────────────────────────────────────────────────

func TestParseCrawlArgs_Defaults(t *testing.T) {
	a, err := ParseCrawlArgs([]string{"--from-multiple", "api.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.FromMultiple) != 1 || a.FromMultiple[0] != "api.md" {
		t.Errorf("FromMultiple: got %v, want [api.md]", a.FromMultiple)
	}
	if a.Direction != "backward" {
		t.Errorf("Direction: got %q, want backward", a.Direction)
	}
	if a.Depth != 3 {
		t.Errorf("Depth: got %d, want 3", a.Depth)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

func TestParseCrawlArgs_MultipleFiles(t *testing.T) {
	a, err := ParseCrawlArgs([]string{"--from-multiple", "api.md,services.md,db.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.FromMultiple) != 3 {
		t.Fatalf("FromMultiple: got %d files, want 3", len(a.FromMultiple))
	}
	if a.FromMultiple[0] != "api.md" || a.FromMultiple[1] != "services.md" || a.FromMultiple[2] != "db.md" {
		t.Errorf("FromMultiple: got %v", a.FromMultiple)
	}
}

func TestParseCrawlArgs_AllFlags(t *testing.T) {
	a, err := ParseCrawlArgs([]string{
		"--from-multiple", "api.md",
		"--dir", "/docs",
		"--direction", "forward",
		"--depth", "5",
		"--format", "tree",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/docs" {
		t.Errorf("Dir: got %q, want /docs", a.Dir)
	}
	if a.Direction != "forward" {
		t.Errorf("Direction: got %q, want forward", a.Direction)
	}
	if a.Depth != 5 {
		t.Errorf("Depth: got %d, want 5", a.Depth)
	}
	if a.Format != "tree" {
		t.Errorf("Format: got %q, want tree", a.Format)
	}
}

func TestParseCrawlArgs_MissingFromMultiple(t *testing.T) {
	_, err := ParseCrawlArgs([]string{})
	if err == nil {
		t.Fatal("expected error for missing --from-multiple")
	}
	if !strings.Contains(err.Error(), "from-multiple") {
		t.Errorf("error should mention from-multiple, got: %v", err)
	}
}

func TestParseCrawlArgs_InvalidDirection(t *testing.T) {
	_, err := ParseCrawlArgs([]string{"--from-multiple", "api.md", "--direction", "sideways"})
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
	if !strings.Contains(err.Error(), "invalid direction") {
		t.Errorf("error should mention invalid direction, got: %v", err)
	}
}

func TestParseCrawlArgs_InvalidFormat(t *testing.T) {
	_, err := ParseCrawlArgs([]string{"--from-multiple", "api.md", "--format", "xml"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error should mention invalid format, got: %v", err)
	}
}

// ─── setupCrawlTestDocs helper ────────────────────────────────────────────────

// setupCrawlTestDocs creates a temp directory with markdown files that form a
// known graph: api-gateway.md -> user-service.md -> auth-service.md
func setupCrawlTestDocs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"api-gateway.md": `# API Gateway

Routes requests to downstream services.
[user-service](./user-service.md)
`,
		"user-service.md": `# User Service

Handles user management.
Depends on [auth-service](./auth-service.md) for token validation.
`,
		"auth-service.md": `# Auth Service

Provides JWT token validation.
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("create %q: %v", path, err)
		}
	}

	// Pre-build the index so CmdCrawl doesn't need to auto-build.
	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir .bmd: %v", err)
	}
	if err := CmdIndex([]string{"--dir", dir, "--db", dbPath}); err != nil {
		t.Fatalf("CmdIndex: %v", err)
	}

	return dir
}

// captureStdoutCrawl runs fn while capturing stdout, returning the output.
// Named differently from captureStdout in context_test.go to avoid redeclaration.
func captureStdoutCrawl(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	buf := make([]byte, 32768)
	n, _ := r.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}

// ─── integration tests ────────────────────────────────────────────────────────

func TestCrawlCLI_MultiStart(t *testing.T) {
	dir := setupCrawlTestDocs(t)

	output := captureStdoutCrawl(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md,auth-service.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "5",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	// Parse JSON envelope.
	var envelope ContractResponse
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope.Status != "ok" {
		t.Errorf("expected status=ok, got %q", envelope.Status)
	}

	// Re-marshal data to inspect it.
	dataBytes, _ := json.Marshal(envelope.Data)
	var data crawlResponseJSON
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("data not valid crawl JSON: %v", err)
	}

	// Should have found multiple start nodes.
	if len(data.StartNodes) < 1 {
		t.Error("expected at least 1 start node")
	}

	// Should have discovered nodes.
	if data.TotalNodes < 2 {
		t.Errorf("expected at least 2 total nodes, got %d", data.TotalNodes)
	}
}

func TestCrawlCLI_JSON(t *testing.T) {
	dir := setupCrawlTestDocs(t)

	output := captureStdoutCrawl(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "5",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	// Verify valid JSON ContractResponse envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", envelope["status"])
	}

	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data field is not an object: %v", envelope["data"])
	}

	// Verify required JSON fields.
	for _, field := range []string{"start_nodes", "nodes", "strategy", "total_nodes", "total_edges"} {
		if _, ok := data[field]; !ok {
			t.Errorf("JSON data missing %q field", field)
		}
	}

	// Verify nodes is a map.
	nodes, ok := data["nodes"].(map[string]interface{})
	if !ok {
		t.Fatalf("nodes is not an object: %v", data["nodes"])
	}

	// Start node should be in the result.
	if _, ok := nodes["api-gateway.md"]; !ok {
		t.Error("expected api-gateway.md in nodes")
	}

	// Verify node structure.
	for nodeID, nodeVal := range nodes {
		node, ok := nodeVal.(map[string]interface{})
		if !ok {
			t.Errorf("node %q is not an object", nodeID)
			continue
		}
		if _, ok := node["depth"]; !ok {
			t.Errorf("node %q missing 'depth' field", nodeID)
		}
		if _, ok := node["edges_out"]; !ok {
			t.Errorf("node %q missing 'edges_out' field", nodeID)
		}
	}
}

func TestCrawlCLI_Tree(t *testing.T) {
	dir := setupCrawlTestDocs(t)

	output := captureStdoutCrawl(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "5",
			"--format", "tree",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	// Tree format should not be JSON.
	if strings.HasPrefix(output, "{") {
		t.Error("tree output should not be JSON")
	}

	// Should contain the start node.
	if !strings.Contains(output, "api-gateway.md") {
		t.Error("tree output should contain api-gateway.md")
	}

	// Should contain child nodes with tree characters.
	if !strings.Contains(output, "user-service.md") {
		t.Error("tree output should contain user-service.md")
	}
}

func TestCrawlCLI_Dot(t *testing.T) {
	dir := setupCrawlTestDocs(t)

	output := captureStdoutCrawl(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "5",
			"--format", "dot",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	// DOT output should start with digraph.
	if !strings.HasPrefix(output, "digraph") {
		t.Errorf("DOT output should start with 'digraph', got: %s", output[:min(50, len(output))])
	}

	// Should contain node declarations.
	if !strings.Contains(output, "api-gateway.md") {
		t.Error("DOT output should contain api-gateway.md")
	}

	// Should contain edge declarations.
	if !strings.Contains(output, "->") {
		t.Error("DOT output should contain edges (->)")
	}

	// Should end with closing brace.
	if !strings.HasSuffix(strings.TrimSpace(output), "}") {
		t.Error("DOT output should end with '}'")
	}
}

func TestCrawlCLI_List(t *testing.T) {
	dir := setupCrawlTestDocs(t)

	output := captureStdoutCrawl(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "5",
			"--format", "list",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	// List format should not be JSON.
	if strings.HasPrefix(output, "{") {
		t.Error("list output should not be JSON")
	}

	// Should contain depth and parents info.
	if !strings.Contains(output, "depth=") {
		t.Error("list output should contain 'depth=' labels")
	}
	if !strings.Contains(output, "parents=") {
		t.Error("list output should contain 'parents=' labels")
	}

	// Start node should be at depth 0.
	if !strings.Contains(output, "depth=0") {
		t.Error("list output should have a node at depth=0")
	}

	// Should contain the start node.
	if !strings.Contains(output, "api-gateway.md") {
		t.Error("list output should contain api-gateway.md")
	}
}

func TestCrawlCLI_Errors(t *testing.T) {
	dir := setupCrawlTestDocs(t)

	t.Run("missing files in graph", func(t *testing.T) {
		output := captureStdoutCrawl(t, func() {
			err := CmdCrawl([]string{
				"--from-multiple", "nonexistent.md",
				"--dir", dir,
				"--format", "json",
			})
			if err != nil {
				t.Fatalf("CmdCrawl should return nil for JSON errors, got: %v", err)
			}
		})

		var envelope map[string]interface{}
		if err := json.Unmarshal([]byte(output), &envelope); err != nil {
			t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
		}
		if envelope["status"] != "error" {
			t.Errorf("expected status=error, got %v", envelope["status"])
		}
		if envelope["code"] != ErrCodeFileNotFound {
			t.Errorf("expected code=%s, got %v", ErrCodeFileNotFound, envelope["code"])
		}
	})

	t.Run("invalid direction", func(t *testing.T) {
		_, err := ParseCrawlArgs([]string{"--from-multiple", "api.md", "--direction", "invalid"})
		if err == nil {
			t.Fatal("expected error for invalid direction")
		}
	})

	t.Run("missing files text format", func(t *testing.T) {
		err := CmdCrawl([]string{
			"--from-multiple", "nonexistent.md",
			"--dir", dir,
			"--format", "text",
		})
		// "text" is not a valid format, should error at parse time
		if err == nil {
			t.Fatal("expected error for invalid format")
		}
	})

	t.Run("missing files list format", func(t *testing.T) {
		err := CmdCrawl([]string{
			"--from-multiple", "nonexistent.md",
			"--dir", dir,
			"--format", "list",
		})
		if err == nil {
			t.Fatal("expected error for missing files in non-JSON format")
		}
	})
}
