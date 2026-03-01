package knowledge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ─── argument parsing tests ───────────────────────────────────────────────────

func TestParseIndexArgs_Defaults(t *testing.T) {
	a, err := ParseIndexArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want %q", a.Dir, ".")
	}
	if a.DB != "knowledge.db" {
		t.Errorf("DB: got %q, want %q", a.DB, "knowledge.db")
	}
	if a.Watch {
		t.Error("Watch should default to false")
	}
	if a.PollInterval != 5 {
		t.Errorf("PollInterval: got %d, want 5", a.PollInterval)
	}
	if a.Strategy != "bm25" {
		t.Errorf("Strategy: got %q, want %q (resolved from env var or default)", a.Strategy, "bm25")
	}
	if a.Model != "claude-sonnet-4-5" {
		t.Errorf("Model: got %q, want %q", a.Model, "claude-sonnet-4-5")
	}
}

func TestParseIndexArgs_StrategyFlag(t *testing.T) {
	a, err := ParseIndexArgs([]string{"--strategy", "pageindex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Strategy != "pageindex" {
		t.Errorf("Strategy: got %q, want %q", a.Strategy, "pageindex")
	}
}

func TestParseIndexArgs_ModelFlag(t *testing.T) {
	a, err := ParseIndexArgs([]string{"--strategy", "pageindex", "--model", "claude-opus-4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Model != "claude-opus-4" {
		t.Errorf("Model: got %q, want %q", a.Model, "claude-opus-4")
	}
}

func TestParseIndexArgs_Positional(t *testing.T) {
	a, err := ParseIndexArgs([]string{"/tmp/docs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/tmp/docs" {
		t.Errorf("Dir: got %q, want %q", a.Dir, "/tmp/docs")
	}
}

func TestParseIndexArgs_Flags(t *testing.T) {
	a, err := ParseIndexArgs([]string{"--dir", "/docs", "--db", "custom.db", "--watch", "--poll-interval", "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/docs" {
		t.Errorf("Dir: got %q, want /docs", a.Dir)
	}
	if a.DB != "custom.db" {
		t.Errorf("DB: got %q, want custom.db", a.DB)
	}
	if !a.Watch {
		t.Error("Watch should be true")
	}
	if a.PollInterval != 10 {
		t.Errorf("PollInterval: got %d, want 10", a.PollInterval)
	}
}

func TestParseQueryArgs_Defaults(t *testing.T) {
	a, err := ParseQueryArgs([]string{"authentication"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Query != "authentication" {
		t.Errorf("Query: got %q, want %q", a.Query, "authentication")
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
	if a.Top != 10 {
		t.Errorf("Top: got %d, want 10", a.Top)
	}
}

func TestParseQueryArgs_MissingTerm(t *testing.T) {
	_, err := ParseQueryArgs([]string{})
	if err == nil {
		t.Fatal("expected error for missing TERM")
	}
	if !strings.Contains(err.Error(), "TERM") {
		t.Errorf("error should mention TERM, got: %v", err)
	}
}

func TestParseQueryArgs_AllFlags(t *testing.T) {
	a, err := ParseQueryArgs([]string{"service", "--dir", "/docs", "--format", "text", "--top", "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Query != "service" {
		t.Errorf("Query: got %q, want service", a.Query)
	}
	if a.Dir != "/docs" {
		t.Errorf("Dir: got %q, want /docs", a.Dir)
	}
	if a.Format != "text" {
		t.Errorf("Format: got %q, want text", a.Format)
	}
	if a.Top != 5 {
		t.Errorf("Top: got %d, want 5", a.Top)
	}
}

func TestParseQueryArgs_InvalidTop(t *testing.T) {
	_, err := ParseQueryArgs([]string{"term", "--top", "0"})
	if err == nil {
		t.Fatal("expected error for top=0")
	}
}

func TestParseQueryArgs_PositionalDir(t *testing.T) {
	a, err := ParseQueryArgs([]string{"term", "/my/docs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/my/docs" {
		t.Errorf("Dir: got %q, want /my/docs", a.Dir)
	}
}

// ─── strategy / model flag tests (Plan 11-03) ─────────────────────────────────

func TestParseQueryArgs_StrategyDefault(t *testing.T) {
	a, err := ParseQueryArgs([]string{"term"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Strategy != "bm25" {
		t.Errorf("Strategy: got %q, want %q (resolved from env var or default)", a.Strategy, "bm25")
	}
}

func TestParseQueryArgs_StrategyBM25(t *testing.T) {
	a, err := ParseQueryArgs([]string{"term", "--strategy", "bm25"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Strategy != "bm25" {
		t.Errorf("Strategy: got %q, want bm25", a.Strategy)
	}
}

func TestParseQueryArgs_StrategyPageindex(t *testing.T) {
	a, err := ParseQueryArgs([]string{"term", "--strategy", "pageindex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Strategy != "pageindex" {
		t.Errorf("Strategy: got %q, want pageindex", a.Strategy)
	}
}

func TestParseQueryArgs_ModelFlag(t *testing.T) {
	a, err := ParseQueryArgs([]string{"term", "--strategy", "pageindex", "--model", "claude-opus-4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Model != "claude-opus-4" {
		t.Errorf("Model: got %q, want claude-opus-4", a.Model)
	}
}

func TestParseQueryArgs_ModelDefault(t *testing.T) {
	a, err := ParseQueryArgs([]string{"term"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Model != "claude-sonnet-4-5" {
		t.Errorf("Model: got %q, want claude-sonnet-4-5", a.Model)
	}
}

// TestCmdQuery_DefaultStrategyIsBM25 verifies that CmdQuery with no --strategy
// flag uses BM25 (the response data does not contain a top-level "strategy" field,
// which differentiates it from the pageindex path).
func TestCmdQuery_DefaultStrategyIsBM25(t *testing.T) {
	dir := setupTestDocs(t)
	dbPath := filepath.Join(dir, "test.db")
	if err := CmdIndex([]string{"--dir", dir, "--db", dbPath}); err != nil {
		t.Fatalf("CmdIndex: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdQuery([]string{"authentication", "--dir", dir, "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdQuery error: %v", err)
	}

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	// BM25 default path response data must not have a "strategy" key.
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data field is not an object: %v", envelope["data"])
	}
	if _, hasStrategy := data["strategy"]; hasStrategy {
		t.Error("BM25 default path should not include 'strategy' field in data payload")
	}
}

// TestCmdQuery_PageindexStrategy_NoTrees verifies that --strategy pageindex
// with no .bmd-tree.json files returns an INDEX_NOT_FOUND error envelope.
func TestCmdQuery_PageindexStrategy_NoTrees(t *testing.T) {
	// Empty temp dir: no markdown files, no tree files.
	emptyDir := t.TempDir()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdQuery([]string{"searchterm", "--dir", emptyDir, "--strategy", "pageindex", "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdQuery should not return an error (handled internally): %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "error" {
		t.Errorf("expected status=error, got %v", envelope["status"])
	}
	if envelope["code"] != ErrCodeIndexNotFound {
		t.Errorf("expected code=%s, got %v", ErrCodeIndexNotFound, envelope["code"])
	}
}

// TestCmdQuery_PageindexStrategy_NoBinary verifies that --strategy pageindex
// with tree files present but no pageindex binary returns PAGEINDEX_NOT_AVAILABLE.
// Note: This test is skipped if pageindex is actually installed in the environment.
func TestCmdQuery_PageindexStrategy_NoBinary(t *testing.T) {
	// Skip test if pageindex is actually available
	if isPageIndexAvailable() {
		t.Skip("pageindex binary is installed; skipping binary-missing test")
	}

	tmpDir := t.TempDir()

	// Write a minimal valid .bmd-tree.json so LoadTreeFiles returns a result.
	ft := FileTree{
		File: "test.md",
		Root: &TreeNode{Heading: "", Summary: "test", LineStart: 1, LineEnd: 10},
	}
	if err := SaveTreeFile(tmpDir, ft); err != nil {
		t.Fatalf("SaveTreeFile: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdQuery([]string{"searchterm", "--dir", tmpDir, "--strategy", "pageindex", "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdQuery should not return an error (handled internally): %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "error" {
		t.Errorf("expected status=error, got %v", envelope["status"])
	}
	if envelope["code"] != ErrCodePageIndexNotAvailable {
		t.Errorf("expected code=%s, got %v", ErrCodePageIndexNotAvailable, envelope["code"])
	}
}

// isPageIndexAvailable checks if the pageindex binary is available in PATH.
func isPageIndexAvailable() bool {
	_, err := exec.LookPath("pageindex")
	return err == nil
}

func TestParseDependsArgs_Defaults(t *testing.T) {
	a, err := ParseDependsArgs([]string{"api-gateway"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Service != "api-gateway" {
		t.Errorf("Service: got %q, want api-gateway", a.Service)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Transitive {
		t.Error("Transitive should default to false")
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

func TestParseDependsArgs_MissingService(t *testing.T) {
	_, err := ParseDependsArgs([]string{})
	if err == nil {
		t.Fatal("expected error for missing SERVICE")
	}
	if !strings.Contains(err.Error(), "SERVICE") {
		t.Errorf("error should mention SERVICE, got: %v", err)
	}
}

func TestParseDependsArgs_AllFlags(t *testing.T) {
	a, err := ParseDependsArgs([]string{"auth-service", "--dir", "/docs", "--transitive", "--format", "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Transitive {
		t.Error("Transitive should be true")
	}
	if a.Format != "text" {
		t.Errorf("Format: got %q, want text", a.Format)
	}
}

func TestParseServicesArgs_Defaults(t *testing.T) {
	a, err := ParseServicesArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

func TestParseServicesArgs_Flags(t *testing.T) {
	a, err := ParseServicesArgs([]string{"--dir", "/docs", "--format", "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/docs" {
		t.Errorf("Dir: got %q, want /docs", a.Dir)
	}
	if a.Format != "text" {
		t.Errorf("Format: got %q, want text", a.Format)
	}
}

func TestParseGraphArgs_Defaults(t *testing.T) {
	a, err := ParseGraphArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "dot" {
		t.Errorf("Format: got %q, want dot", a.Format)
	}
	if a.Service != "" {
		t.Errorf("Service: got %q, want empty", a.Service)
	}
}

func TestParseGraphArgs_ServiceFlag(t *testing.T) {
	a, err := ParseGraphArgs([]string{"--service", "api-gateway", "--format", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Service != "api-gateway" {
		t.Errorf("Service: got %q, want api-gateway", a.Service)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

func TestParseGraphArgs_PositionalService(t *testing.T) {
	a, err := ParseGraphArgs([]string{"auth-service"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Service != "auth-service" {
		t.Errorf("Service: got %q, want auth-service", a.Service)
	}
}

// ─── command integration tests ─────────────────────────────────────────────────

// setupTestDocs creates a temporary directory with sample markdown files for
// integration testing.  Returns the directory path and a cleanup function.
func setupTestDocs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Service files.
	files := map[string]string{
		"api-gateway.md": `# API Gateway

The API Gateway routes requests to downstream services.

It calls the user-service for authentication:
[user-service](./user-service.md)

## Endpoints

- GET /health
- POST /api/v1/users
`,
		"user-service.md": `# User Service

The User Service handles user management.

It depends on [auth-service](./auth-service.md) for token validation.

## Endpoints

- GET /users
- POST /users
`,
		"auth-service.md": `# Auth Service

The Auth Service provides JWT token validation.

## Endpoints

- POST /auth/token
- GET /auth/validate
`,
		"README.md": `# Documentation

Welcome to the documentation.

See [api-gateway](./api-gateway.md) for the main entry point.
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("create %q: %v", path, err)
		}
	}

	return dir
}

func TestCmdIndex_Basic(t *testing.T) {
	dir := setupTestDocs(t)
	dbPath := filepath.Join(dir, "test.db")

	err := CmdIndex([]string{"--dir", dir, "--db", dbPath})
	if err != nil {
		t.Fatalf("CmdIndex error: %v", err)
	}

	// Verify database was created.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}

	// Verify database has content.
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close() //nolint:errcheck

	idx := NewIndex()
	if err := db.LoadIndex(idx); err != nil {
		t.Fatalf("load index: %v", err)
	}

	if idx.DocCount() == 0 {
		t.Error("index should have documents")
	}
}

func TestCmdQuery_JSON(t *testing.T) {
	dir := setupTestDocs(t)
	dbPath := filepath.Join(dir, "test.db")

	// Pre-build index.
	if err := CmdIndex([]string{"--dir", dir, "--db", dbPath}); err != nil {
		t.Fatalf("CmdIndex: %v", err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdQuery([]string{"authentication", "--dir", dir, "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdQuery error: %v", err)
	}

	// Read captured output.
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	// Verify JSON is valid and is a ContractResponse envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify top-level envelope fields.
	if _, ok := envelope["status"]; !ok {
		t.Error("JSON missing top-level 'status' field")
	}
	if _, ok := envelope["data"]; !ok {
		t.Error("JSON missing top-level 'data' field")
	}

	// Verify data payload contains expected search fields.
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data field is not an object: %v", envelope["data"])
	}
	if _, ok := data["query"]; !ok {
		t.Error("JSON data missing 'query' field")
	}
	if _, ok := data["results"]; !ok {
		t.Error("JSON data missing 'results' field")
	}
	if _, ok := data["count"]; !ok {
		t.Error("JSON data missing 'count' field")
	}
}

func TestCmdQuery_TextFormat(t *testing.T) {
	dir := setupTestDocs(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdQuery([]string{"service", "--dir", dir, "--format", "text"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdQuery error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Text format should not be JSON (no leading '{').
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "{") {
		t.Error("text output should not be JSON")
	}
}

func TestCmdServices_JSON(t *testing.T) {
	dir := setupTestDocs(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdServices([]string{"--dir", dir, "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdServices error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	// Verify JSON is valid and is a ContractResponse envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify top-level envelope fields.
	if _, ok := envelope["status"]; !ok {
		t.Error("JSON missing top-level 'status' field")
	}
	if _, ok := envelope["data"]; !ok {
		t.Error("JSON missing top-level 'data' field")
	}

	// Verify data payload contains services.
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data field is not an object: %v", envelope["data"])
	}
	if _, ok := data["services"]; !ok {
		t.Error("JSON data missing 'services' field")
	}
}

func TestCmdServices_TextFormat(t *testing.T) {
	dir := setupTestDocs(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdServices([]string{"--dir", dir, "--format", "text"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdServices error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "{") {
		t.Error("text output should not be JSON")
	}
}

func TestCmdGraph_DOT(t *testing.T) {
	dir := setupTestDocs(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdGraph([]string{"--dir", dir, "--format", "dot"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdGraph error: %v", err)
	}

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	if !strings.HasPrefix(output, "digraph") {
		t.Errorf("DOT output should start with 'digraph', got: %s", output[:min(50, len(output))])
	}
	if !strings.HasSuffix(output, "}") {
		t.Error("DOT output should end with '}'")
	}
}

func TestCmdGraph_JSON(t *testing.T) {
	dir := setupTestDocs(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdGraph([]string{"--dir", dir, "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdGraph error: %v", err)
	}

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	// Verify JSON is valid and is a ContractResponse envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify top-level envelope fields.
	if _, ok := envelope["status"]; !ok {
		t.Error("JSON missing top-level 'status' field")
	}
	if _, ok := envelope["data"]; !ok {
		t.Error("JSON missing top-level 'data' field")
	}

	// Verify data payload contains graph fields.
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data field is not an object: %v", envelope["data"])
	}
	if _, ok := data["nodes"]; !ok {
		t.Error("JSON data missing 'nodes' field")
	}
	if _, ok := data["edges"]; !ok {
		t.Error("JSON data missing 'edges' field")
	}
}

func TestCmdDepends_MissingService(t *testing.T) {
	dir := setupTestDocs(t)

	// JSON format (default): missing service emits FILE_NOT_FOUND envelope on stdout, returns nil.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdDepends([]string{"nonexistent-service", "--dir", dir, "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdDepends with JSON format should return nil, got: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

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

	// Text format: missing service returns an error.
	err = CmdDepends([]string{"nonexistent-service", "--dir", dir, "--format", "text"})
	if err == nil {
		t.Fatal("expected error for unknown service with text format")
	}
}

func TestCmdIndex_InvalidDir(t *testing.T) {
	err := CmdIndex([]string{"--dir", "/nonexistent/path/12345"})
	if err == nil {
		t.Fatal("expected error for invalid directory")
	}
}

func TestCmdQuery_MissingTerm(t *testing.T) {
	err := CmdQuery([]string{"--dir", "."})
	if err == nil {
		t.Fatal("expected error for missing search term")
	}
}

// ─── output formatter tests ───────────────────────────────────────────────────

func TestFormatSearchResultsJSON(t *testing.T) {
	results := []SearchResult{
		{
			DocID:   "docs/auth.md",
			RelPath: "docs/auth.md",
			Title:   "Auth Service",
			Score:   0.87,
			Snippet: "The Auth Service provides JWT token validation",
		},
	}
	output := FormatSearchResults(results, "authentication", "json", 12)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	if result["query"] != "authentication" {
		t.Errorf("query: got %v, want authentication", result["query"])
	}
	if result["count"].(float64) != 1 {
		t.Errorf("count: got %v, want 1", result["count"])
	}
}

func TestFormatSearchResultsText(t *testing.T) {
	results := []SearchResult{
		{
			RelPath: "docs/auth.md",
			Title:   "Auth Service",
			Score:   0.87,
			Snippet: "JWT token validation",
		},
	}
	output := FormatSearchResults(results, "auth", "text", 0)

	if !strings.Contains(output, "docs/auth.md") {
		t.Error("text output should contain file path")
	}
	if !strings.Contains(output, "0.8700") {
		t.Error("text output should contain score")
	}
}

func TestFormatSearchResultsCSV(t *testing.T) {
	results := []SearchResult{
		{
			RelPath: "docs/auth.md",
			Title:   "Auth Service",
			Score:   0.87,
			Snippet: "JWT token validation",
		},
	}
	output := FormatSearchResults(results, "auth", "csv", 0)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("CSV should have header + data row, got %d lines", len(lines))
	}
	// Header row.
	if !strings.HasPrefix(lines[0], "rank|") {
		t.Errorf("CSV header should start with 'rank|', got: %s", lines[0])
	}
}

func TestFormatSearchResultsEmpty(t *testing.T) {
	output := FormatSearchResults(nil, "nothing", "text", 0)
	if output != "No results found." {
		t.Errorf("empty text result: got %q, want %q", output, "No results found.")
	}
}

func TestFormatServicesJSON(t *testing.T) {
	services := []Service{
		{ID: "auth-service", Name: "Auth Service", File: "auth-service.md", Confidence: 0.9},
	}
	depCounts := map[string]int{"auth-service": 2}
	output := FormatServices(services, depCounts, "json")

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	svcs, ok := result["services"].([]interface{})
	if !ok || len(svcs) != 1 {
		t.Fatalf("expected 1 service, got: %v", result["services"])
	}
	svc := svcs[0].(map[string]interface{})
	if svc["id"] != "auth-service" {
		t.Errorf("id: got %v, want auth-service", svc["id"])
	}
	if svc["dependency_count"].(float64) != 2 {
		t.Errorf("dependency_count: got %v, want 2", svc["dependency_count"])
	}
}

func TestFormatServicesText(t *testing.T) {
	services := []Service{
		{ID: "api-gateway", Name: "API Gateway", File: "api-gateway.md", Confidence: 1.0},
	}
	output := FormatServices(services, nil, "text")
	if !strings.Contains(output, "api-gateway") {
		t.Error("text output should contain service ID")
	}
	if !strings.Contains(output, "1.00") {
		t.Error("text output should contain confidence score")
	}
}

func TestFormatServicesEmpty(t *testing.T) {
	output := FormatServices(nil, nil, "text")
	if output != "No services detected." {
		t.Errorf("empty text: got %q, want %q", output, "No services detected.")
	}
}

func TestFormatDependenciesJSON_Direct(t *testing.T) {
	refs := []ServiceRef{
		{ServiceID: "user-service", Type: "direct-call", Confidence: 0.95},
	}
	output := FormatDependencies("api-gateway", refs, false, nil, nil, "json")

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if result["service"] != "api-gateway" {
		t.Errorf("service: got %v, want api-gateway", result["service"])
	}
}

func TestFormatDependenciesJSON_Transitive(t *testing.T) {
	paths := [][]string{{"api-gateway", "user-service", "postgres"}}
	output := FormatDependencies("api-gateway", nil, true, paths, nil, "json")

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if _, ok := result["transitive_dependencies"]; !ok {
		t.Error("JSON missing 'transitive_dependencies' field")
	}
}

func TestFormatDependenciesDOT(t *testing.T) {
	refs := []ServiceRef{
		{ServiceID: "user-service", Type: "direct-call", Confidence: 0.95},
	}
	output := FormatDependencies("api-gateway", refs, false, nil, nil, "dot")
	if !strings.HasPrefix(strings.TrimSpace(output), "digraph") {
		t.Errorf("DOT output should start with digraph, got: %s", output[:min(40, len(output))])
	}
}

func TestFormatDependenciesText_WithCycles(t *testing.T) {
	refs := []ServiceRef{
		{ServiceID: "user-service", Type: "direct-call", Confidence: 0.9},
	}
	cycles := [][]string{{"api-gateway", "user-service", "api-gateway"}}
	output := FormatDependencies("api-gateway", refs, false, nil, cycles, "text")
	if !strings.Contains(output, "Cycles detected") {
		t.Error("text output should mention cycles")
	}
}

func TestFormatGraphDOT(t *testing.T) {
	graph := NewGraph()
	_ = graph.AddNode(&Node{ID: "api-gateway", Title: "API Gateway", Type: "document"})
	_ = graph.AddNode(&Node{ID: "user-service", Title: "User Service", Type: "document"})
	edge, _ := NewEdge("api-gateway", "user-service", EdgeCalls, 0.95, "")
	_ = graph.AddEdge(edge)

	output := FormatGraph(graph, "dot")
	if !strings.Contains(output, "digraph knowledge_graph") {
		t.Error("DOT output should contain 'digraph knowledge_graph'")
	}
	if !strings.Contains(output, "api-gateway") {
		t.Error("DOT output should contain node 'api-gateway'")
	}
	if !strings.Contains(output, "user-service") {
		t.Error("DOT output should contain node 'user-service'")
	}
	if !strings.HasSuffix(strings.TrimSpace(output), "}") {
		t.Error("DOT output should end with '}'")
	}
}

func TestFormatGraphJSON(t *testing.T) {
	graph := NewGraph()
	_ = graph.AddNode(&Node{ID: "api-gateway", Title: "API Gateway", Type: "document"})
	_ = graph.AddNode(&Node{ID: "user-service", Title: "User Service", Type: "document"})
	edge, _ := NewEdge("api-gateway", "user-service", EdgeReferences, 1.0, "link")
	_ = graph.AddEdge(edge)

	output := FormatGraph(graph, "json")

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	nodes, ok := result["nodes"].([]interface{})
	if !ok {
		t.Fatal("JSON missing or invalid 'nodes'")
	}
	if len(nodes) != 2 {
		t.Errorf("nodes: got %d, want 2", len(nodes))
	}

	edges, ok := result["edges"].([]interface{})
	if !ok {
		t.Fatal("JSON missing or invalid 'edges'")
	}
	if len(edges) != 1 {
		t.Errorf("edges: got %d, want 1", len(edges))
	}
}

func TestFormatGraphEmpty(t *testing.T) {
	graph := NewGraph()
	output := FormatGraph(graph, "json")

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	nodes := result["nodes"].([]interface{})
	if len(nodes) != 0 {
		t.Errorf("empty graph should have 0 nodes, got %d", len(nodes))
	}
}

// ─── helper function tests ────────────────────────────────────────────────────

func TestRoundFloat(t *testing.T) {
	cases := []struct {
		input    float64
		decimals int
		want     float64
	}{
		{0.87654, 4, 0.8765},
		{0.87656, 4, 0.8766},
		{1.5, 0, 2.0},
		{0.12345, 2, 0.12},
	}
	for _, tc := range cases {
		got := roundFloat(tc.input, tc.decimals)
		if got != tc.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tc.input, tc.decimals, got, tc.want)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{512, "512B"},
		{1024, "1.0KB"},
		{1024 * 1024, "1.0MB"},
		{12 * 1024 * 1024, "12.0MB"},
	}
	for _, tc := range cases {
		got := humanBytes(tc.input)
		if got != tc.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDefaultDBPath(t *testing.T) {
	got := defaultDBPath("/some/dir")
	want := filepath.Join("/some/dir", "knowledge.db")
	if got != want {
		t.Errorf("defaultDBPath: got %q, want %q", got, want)
	}
}

func TestFindNodeForService(t *testing.T) {
	graph := NewGraph()
	_ = graph.AddNode(&Node{ID: "services/auth-service.md", Title: "Auth Service", Type: "document"})
	_ = graph.AddNode(&Node{ID: "api-gateway.md", Title: "API Gateway", Type: "document"})

	// Exact match.
	got := findNodeForService(graph, "services/auth-service.md")
	if got != "services/auth-service.md" {
		t.Errorf("exact match: got %q, want services/auth-service.md", got)
	}

	// Stem match.
	got = findNodeForService(graph, "auth-service")
	if got != "services/auth-service.md" {
		t.Errorf("stem match: got %q, want services/auth-service.md", got)
	}

	// Not found.
	got = findNodeForService(graph, "nonexistent")
	if got != "" {
		t.Errorf("not-found: got %q, want empty", got)
	}
}

// ─── ContractResponse unit tests ──────────────────────────────────────────────

func TestContractResponsePaths(t *testing.T) {
	t.Run("ok response has nil code", func(t *testing.T) {
		resp := NewOKResponse("done", map[string]int{"count": 1})
		if resp.Status != "ok" {
			t.Errorf("expected status=ok, got %q", resp.Status)
		}
		if resp.Code != nil {
			t.Errorf("expected nil code, got %q", *resp.Code)
		}
		if resp.Data == nil {
			t.Error("expected non-nil data")
		}
	})

	t.Run("empty response has nil code", func(t *testing.T) {
		resp := NewEmptyResponse("no results", nil)
		if resp.Status != "empty" {
			t.Errorf("expected status=empty, got %q", resp.Status)
		}
		if resp.Code != nil {
			t.Errorf("expected nil code, got %q", *resp.Code)
		}
	})

	t.Run("error response INDEX_NOT_FOUND", func(t *testing.T) {
		resp := NewErrorResponse(ErrCodeIndexNotFound, "no index")
		if resp.Status != "error" {
			t.Errorf("expected status=error, got %q", resp.Status)
		}
		if resp.Code == nil || *resp.Code != ErrCodeIndexNotFound {
			t.Errorf("expected code=INDEX_NOT_FOUND, got %v", resp.Code)
		}
		if resp.Data != nil {
			t.Error("expected nil data for error response")
		}
	})

	t.Run("error response FILE_NOT_FOUND", func(t *testing.T) {
		resp := NewErrorResponse(ErrCodeFileNotFound, "not found")
		if resp.Status != "error" {
			t.Errorf("expected status=error, got %q", resp.Status)
		}
		if resp.Code == nil || *resp.Code != ErrCodeFileNotFound {
			t.Errorf("expected code=FILE_NOT_FOUND, got %v", resp.Code)
		}
	})

	t.Run("error response INVALID_QUERY", func(t *testing.T) {
		resp := NewErrorResponse(ErrCodeInvalidQuery, "bad query")
		if resp.Status != "error" {
			t.Errorf("expected status=error, got %q", resp.Status)
		}
		if resp.Code == nil || *resp.Code != ErrCodeInvalidQuery {
			t.Errorf("expected code=INVALID_QUERY, got %v", resp.Code)
		}
	})

	t.Run("error response INTERNAL_ERROR", func(t *testing.T) {
		resp := NewErrorResponse(ErrCodeInternalError, "unexpected failure")
		if resp.Status != "error" {
			t.Errorf("expected status=error, got %q", resp.Status)
		}
		if resp.Code == nil || *resp.Code != ErrCodeInternalError {
			t.Errorf("expected code=INTERNAL_ERROR, got %v", resp.Code)
		}
	})

	t.Run("marshalContract produces valid JSON", func(t *testing.T) {
		resp := NewOKResponse("test", nil)
		out := marshalContract(resp)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Errorf("marshalContract output is not valid JSON: %v", err)
		}
		if parsed["status"] != "ok" {
			t.Errorf("expected status=ok in marshaled output, got %v", parsed["status"])
		}
	})

	t.Run("empty response serializes code as null", func(t *testing.T) {
		resp := NewEmptyResponse("no results", nil)
		out := marshalContract(resp)
		if !strings.Contains(out, `"code": null`) {
			t.Errorf("expected code:null in empty response JSON, got: %s", out)
		}
	})
}
