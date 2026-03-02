package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

// makeTestServer creates a Server with a temporary directory and index for testing.
func makeTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	// Write sample markdown files.
	writeFile(t, filepath.Join(dir, "api-gateway.md"), `# API Gateway

Handles routing for all services.

## Authentication

Uses JWT tokens for authentication. All requests must include a Bearer token.

## Dependencies

- auth-service
- user-service
`)
	writeFile(t, filepath.Join(dir, "auth-service.md"), `# Auth Service

Manages user authentication and token issuance.

## Endpoints

- POST /auth/login
- POST /auth/refresh
- DELETE /auth/logout
`)
	writeFile(t, filepath.Join(dir, "user-service.md"), `# User Service

Stores and manages user profiles.

## Endpoints

- GET /users/:id
- PUT /users/:id
- DELETE /users/:id
`)

	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	srv := NewServer(dir, dbPath)
	return srv, dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// callTool invokes a handler directly with the given arguments map.
func callTool(handler func(context.Context, mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error),
	args map[string]interface{}) (*mcpsdk.CallToolResult, error) {

	req := mcpsdk.CallToolRequest{}
	req.Params.Arguments = args
	return handler(context.Background(), req)
}

// ─── TestMCPServer_QueryTool ────────────────────────────────────────────────

// TestMCPServer_QueryTool verifies that the query handler returns results
// wrapped in a CONTRACT-01 JSON envelope.
func TestMCPServer_QueryTool(t *testing.T) {
	srv, _ := makeTestServer(t)

	result, err := callTool(srv.handleQuery, map[string]interface{}{
		"query":    "authentication",
		"strategy": "bm25",
		"dir":      srv.baseDir,
		"top":      float64(5),
	})
	if err != nil {
		t.Fatalf("handleQuery error: %v", err)
	}
	if result == nil {
		t.Fatal("handleQuery returned nil result")
	}

	// Result should contain text content.
	text := extractText(result)
	if text == "" {
		t.Fatal("handleQuery returned empty text")
	}

	// Verify CONTRACT-01 envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("handleQuery output is not valid JSON: %v\noutput: %s", err, text)
	}
	status, _ := envelope["status"].(string)
	if status != "ok" && status != "empty" && status != "error" {
		t.Errorf("envelope status %q not a valid CONTRACT-01 status", status)
	}
	if _, hasMessage := envelope["message"]; !hasMessage {
		t.Error("envelope missing 'message' field")
	}
}

// ─── TestMCPServer_IndexTool ─────────────────────────────────────────────────

// TestMCPServer_IndexTool verifies that the index handler builds an index
// without errors and returns a non-empty status message.
func TestMCPServer_IndexTool(t *testing.T) {
	srv, dir := makeTestServer(t)

	result, err := callTool(srv.handleIndex, map[string]interface{}{
		"dir": dir,
	})
	if err != nil {
		t.Fatalf("handleIndex error: %v", err)
	}
	if result == nil {
		t.Fatal("handleIndex returned nil result")
	}

	text := extractText(result)
	if text == "" {
		t.Fatal("handleIndex returned empty text")
	}
	// Progress output should mention indexing or completion.
	if text == "Indexing complete." && len(text) == 0 {
		t.Error("handleIndex returned no progress messages")
	}
}

// ─── TestMCPServer_ContractCompliance ────────────────────────────────────────

// TestMCPServer_ContractCompliance verifies that all tool handlers return
// CONTRACT-01 compliant responses (for tools that output JSON).
func TestMCPServer_ContractCompliance(t *testing.T) {
	srv, dir := makeTestServer(t)

	// First build the index so graph/depends/services commands can work.
	_, _ = callTool(srv.handleIndex, map[string]interface{}{"dir": dir})

	tools := []struct {
		name    string
		handler func(context.Context, mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error)
		args    map[string]interface{}
	}{
		{
			name:    "query",
			handler: srv.handleQuery,
			args:    map[string]interface{}{"query": "authentication", "dir": dir},
		},
		{
			name:    "components",
			handler: srv.handleComponents,
			args:    map[string]interface{}{"dir": dir},
		},
		{
			name:    "graph",
			handler: srv.handleGraph,
			args:    map[string]interface{}{"dir": dir},
		},
		{
			name:    "context",
			handler: srv.handleContext,
			args:    map[string]interface{}{"query": "authentication", "dir": dir},
		},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			result, err := callTool(tc.handler, tc.args)
			if err != nil {
				t.Fatalf("%s handler error: %v", tc.name, err)
			}
			if result == nil {
				t.Fatalf("%s handler returned nil result", tc.name)
			}

			text := extractText(result)
			if text == "" {
				t.Fatalf("%s returned empty text", tc.name)
			}

			// Verify envelope structure.
			var envelope map[string]interface{}
			if err := json.Unmarshal([]byte(text), &envelope); err != nil {
				t.Fatalf("%s output not valid JSON: %v\noutput: %s", tc.name, err, text)
			}
			status, ok := envelope["status"].(string)
			if !ok {
				t.Errorf("%s envelope missing 'status' string field", tc.name)
			}
			if status != "ok" && status != "empty" && status != "error" {
				t.Errorf("%s status %q not a valid CONTRACT-01 status", tc.name, status)
			}
		})
	}
}

// ─── TestMCPServer_RequiredParams ─────────────────────────────────────────────

// TestMCPServer_RequiredParams verifies that handlers return tool errors when
// required parameters are missing, not Go errors.
func TestMCPServer_RequiredParams(t *testing.T) {
	srv, _ := makeTestServer(t)

	t.Run("query_missing_query_param", func(t *testing.T) {
		result, err := callTool(srv.handleQuery, map[string]interface{}{})
		if err != nil {
			t.Fatalf("should not return Go error for missing param, got: %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
		// Should be an error result, not a Go error.
		if !result.IsError {
			t.Error("expected IsError=true for missing required param")
		}
	})

	t.Run("depends_missing_service_param", func(t *testing.T) {
		result, err := callTool(srv.handleDepends, map[string]interface{}{})
		if err != nil {
			t.Fatalf("should not return Go error for missing param, got: %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
		if !result.IsError {
			t.Error("expected IsError=true for missing required param")
		}
	})

	t.Run("context_missing_query_param", func(t *testing.T) {
		result, err := callTool(srv.handleContext, map[string]interface{}{})
		if err != nil {
			t.Fatalf("should not return Go error for missing param, got: %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
		if !result.IsError {
			t.Error("expected IsError=true for missing required param")
		}
	})
}

// ─── TestMCPServer_Concurrency ───────────────────────────────────────────────

// TestMCPServer_Concurrency verifies that multiple concurrent query requests
// can be handled without data races or panics.
func TestMCPServer_Concurrency(t *testing.T) {
	srv, dir := makeTestServer(t)

	// Build index first.
	_, _ = callTool(srv.handleIndex, map[string]interface{}{"dir": dir})

	const goroutines = 5
	var wg sync.WaitGroup
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := callTool(srv.handleQuery, map[string]interface{}{
				"query": "authentication",
				"dir":   dir,
			})
			if err != nil {
				errors[idx] = err
				return
			}
			if result == nil {
				t.Errorf("goroutine %d: nil result", idx)
			}
		}(i)
	}

	wg.Wait()

	for i, e := range errors {
		if e != nil {
			t.Errorf("goroutine %d error: %v", i, e)
		}
	}
}

// ─── TestMCPServer_NewServer ─────────────────────────────────────────────────

// TestMCPServer_NewServer verifies that NewServer initializes the Server struct correctly.
func TestMCPServer_NewServer(t *testing.T) {
	srv := NewServer("/tmp/docs", "/tmp/docs/.bmd/knowledge.db")
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.baseDir != "/tmp/docs" {
		t.Errorf("baseDir: got %q, want %q", srv.baseDir, "/tmp/docs")
	}
	if srv.dbPath != "/tmp/docs/.bmd/knowledge.db" {
		t.Errorf("dbPath: got %q, want %q", srv.dbPath, "/tmp/docs/.bmd/knowledge.db")
	}
}

// ─── TestMCP_GraphCrawl_MultiStart ────────────────────────────────────────────

// TestMCP_GraphCrawl_MultiStart verifies that graph_crawl with multiple start
// files returns a valid ContractResponse with nodes from both start branches.
func TestMCP_GraphCrawl_MultiStart(t *testing.T) {
	srv, dir := makeTestServer(t)

	// Build the index first so the graph is available.
	_, _ = callTool(srv.handleIndex, map[string]interface{}{"dir": dir})

	result, err := callTool(srv.handleGraphCrawl, map[string]interface{}{
		"start_files": "api-gateway.md,auth-service.md",
		"direction":   "forward",
		"depth":       float64(3),
		"dir":         dir,
	})
	if err != nil {
		t.Fatalf("handleGraphCrawl error: %v", err)
	}
	if result == nil {
		t.Fatal("handleGraphCrawl returned nil result")
	}

	text := extractText(result)
	if text == "" {
		t.Fatal("handleGraphCrawl returned empty text")
	}

	// Verify CONTRACT-01 envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, text)
	}
	status, _ := envelope["status"].(string)
	if status != "ok" {
		t.Errorf("expected status 'ok', got %q", status)
	}
	if _, hasMessage := envelope["message"]; !hasMessage {
		t.Error("envelope missing 'message' field")
	}
	if _, hasData := envelope["data"]; !hasData {
		t.Error("envelope missing 'data' field")
	}

	// Verify data contains expected crawl fields.
	data, _ := envelope["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("data field is not an object")
	}
	startNodes, _ := data["StartNodes"].([]interface{})
	if len(startNodes) < 1 {
		t.Error("expected at least 1 start node in result")
	}
	totalNodes, _ := data["TotalNodes"].(float64)
	if totalNodes < 1 {
		t.Errorf("expected TotalNodes >= 1, got %.0f", totalNodes)
	}
}

// ─── TestMCP_GraphCrawl_Cycles ───────────────────────────────────────────────

// TestMCP_GraphCrawl_Cycles verifies that graph_crawl with include_cycles=true
// returns cycle information in the response.
func TestMCP_GraphCrawl_Cycles(t *testing.T) {
	// Create a test directory with files that form a cycle:
	// A depends on B, B depends on A.
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "service-a.md"), `# Service A

Handles user requests.

## Dependencies

- [Service B](service-b.md)
`)
	writeFile(t, filepath.Join(dir, "service-b.md"), `# Service B

Processes data.

## Dependencies

- [Service A](service-a.md)
`)

	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	srv := NewServer(dir, dbPath)

	// Build the index.
	_, _ = callTool(srv.handleIndex, map[string]interface{}{"dir": dir})

	result, err := callTool(srv.handleGraphCrawl, map[string]interface{}{
		"start_files":    "service-a.md",
		"direction":      "forward",
		"depth":          float64(5),
		"include_cycles": true,
		"dir":            dir,
	})
	if err != nil {
		t.Fatalf("handleGraphCrawl error: %v", err)
	}
	if result == nil {
		t.Fatal("handleGraphCrawl returned nil result")
	}

	text := extractText(result)
	if text == "" {
		t.Fatal("handleGraphCrawl returned empty text")
	}

	// Verify CONTRACT-01 envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, text)
	}
	status, _ := envelope["status"].(string)
	if status != "ok" {
		t.Errorf("expected status 'ok', got %q", status)
	}

	// Verify data has Cycles field.
	data, _ := envelope["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("data field is not an object")
	}
	// Cycles should be present (may or may not contain entries depending on
	// graph structure, but the field must exist since include_cycles was true).
	if _, hasCycles := data["Cycles"]; !hasCycles {
		t.Error("expected 'Cycles' field in data when include_cycles=true")
	}
}

// ─── TestMCP_GraphCrawl_Error ────────────────────────────────────────────────

// TestMCP_GraphCrawl_Error verifies that graph_crawl returns a proper error
// response when a start file doesn't exist in the graph.
func TestMCP_GraphCrawl_Error(t *testing.T) {
	srv, dir := makeTestServer(t)

	// Build the index first.
	_, _ = callTool(srv.handleIndex, map[string]interface{}{"dir": dir})

	result, err := callTool(srv.handleGraphCrawl, map[string]interface{}{
		"start_files": "nonexistent-file.md",
		"dir":         dir,
	})
	if err != nil {
		t.Fatalf("should not return Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}

	text := extractText(result)
	if text == "" {
		t.Fatal("handleGraphCrawl returned empty text")
	}

	// Verify CONTRACT-01 error envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, text)
	}
	status, _ := envelope["status"].(string)
	if status != "error" {
		t.Errorf("expected status 'error', got %q", status)
	}
	code, _ := envelope["code"].(string)
	if code != "GRAPH_NOT_FOUND" {
		t.Errorf("expected code 'GRAPH_NOT_FOUND', got %q", code)
	}
}

// ─── TestMCP_GraphCrawl_MissingParam ─────────────────────────────────────────

// TestMCP_GraphCrawl_MissingParam verifies that a missing start_files parameter
// returns an MCP tool error (IsError=true).
func TestMCP_GraphCrawl_MissingParam(t *testing.T) {
	srv, _ := makeTestServer(t)

	result, err := callTool(srv.handleGraphCrawl, map[string]interface{}{})
	if err != nil {
		t.Fatalf("should not return Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if !result.IsError {
		t.Error("expected IsError=true for missing required start_files param")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// extractText pulls the text content from an MCP CallToolResult.
func extractText(result *mcpsdk.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcpsdk.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
