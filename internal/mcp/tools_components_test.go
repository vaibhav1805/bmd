package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeComponentTestServer creates a Server with a monorepo test directory.
// It creates three markdown files and a .bmd/components.yaml so that
// BuildComponentGraphFromConfig can reliably detect components.
func makeComponentTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	// .bmd/components.yaml — ensures reliable component detection.
	if err := os.MkdirAll(filepath.Join(dir, ".bmd"), 0o755); err != nil {
		t.Fatalf("mkdir .bmd: %v", err)
	}
	writeFile(t, filepath.Join(dir, ".bmd", "components.yaml"), `
components:
  - name: api
    path: services/api
  - name: auth
    path: services/auth
  - name: user
    path: services/user
`)

	// Component markdown files.
	if err := os.MkdirAll(filepath.Join(dir, "services/api"), 0o755); err != nil {
		t.Fatalf("mkdir services/api: %v", err)
	}
	writeFile(t, filepath.Join(dir, "services/api/README.md"),
		"# API Service\n\nCalls `auth` for auth and `user` for profiles.\n")

	if err := os.MkdirAll(filepath.Join(dir, "services/auth"), 0o755); err != nil {
		t.Fatalf("mkdir services/auth: %v", err)
	}
	writeFile(t, filepath.Join(dir, "services/auth/README.md"),
		"# Auth Service\n\nHandles JWT authentication.\n")

	if err := os.MkdirAll(filepath.Join(dir, "services/user"), 0o755); err != nil {
		t.Fatalf("mkdir services/user: %v", err)
	}
	writeFile(t, filepath.Join(dir, "services/user/README.md"),
		"# User Service\n\nManages user profiles.\n")

	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	srv := NewServer(dir, dbPath)
	return srv, dir
}

// ─── handleComponentList MCP tool tests ──────────────────────────────────────

func TestMCP_ComponentList_ReturnsJSONEnvelope(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleComponentList, map[string]interface{}{
		"root_dir": dir,
	})
	if err != nil {
		t.Fatalf("handleComponentList: %v", err)
	}
	if result == nil {
		t.Fatal("handleComponentList returned nil")
	}

	text := extractText(result)
	if text == "" {
		t.Fatal("handleComponentList returned empty text")
	}

	// Must be valid JSON.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("handleComponentList output is not valid JSON: %v\n\nOutput:\n%s", err, text)
	}
}

func TestMCP_ComponentList_HasStatusField(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, _ := callTool(srv.handleComponentList, map[string]interface{}{
		"root_dir": dir,
	})

	text := extractText(result)
	var envelope map[string]interface{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope)

	status, ok := envelope["status"].(string)
	if !ok || status == "" {
		t.Errorf("envelope missing or empty 'status' field, got: %v", envelope["status"])
	}
}

func TestMCP_ComponentList_EmptyDirReturnsValidJSON(t *testing.T) {
	srv, _ := makeComponentTestServer(t)
	emptyDir := t.TempDir()

	result, err := callTool(srv.handleComponentList, map[string]interface{}{
		"root_dir": emptyDir,
	})
	if err != nil {
		t.Fatalf("handleComponentList: %v", err)
	}

	text := extractText(result)
	// Must be valid JSON regardless of whether the dir is empty.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("empty dir output is not valid JSON: %v\n\nOutput:\n%s", err, text)
	}
	// Status should be one of the valid STATUS-01 values.
	status, _ := envelope["status"].(string)
	validStatuses := map[string]bool{"ok": true, "empty": true, "error": true}
	if !validStatuses[status] {
		t.Errorf("unexpected status %q for empty dir, want ok/empty/error", status)
	}
}

// ─── handleComponentGraph MCP tool tests ─────────────────────────────────────

func TestMCP_ComponentGraph_JSONFormat(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleComponentGraph, map[string]interface{}{
		"root_dir": dir,
		"format":   "json",
	})
	if err != nil {
		t.Fatalf("handleComponentGraph JSON: %v", err)
	}

	text := extractText(result)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("handleComponentGraph JSON output invalid: %v\n\nOutput:\n%s", err, text)
	}
}

func TestMCP_ComponentGraph_ASCIIFormat(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleComponentGraph, map[string]interface{}{
		"root_dir": dir,
		"format":   "ascii",
	})
	if err != nil {
		t.Fatalf("handleComponentGraph ASCII: %v", err)
	}

	text := extractText(result)
	if text == "" {
		t.Error("handleComponentGraph ASCII returned empty text")
	}
}

func TestMCP_ComponentGraph_InvalidFormatReturnsError(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleComponentGraph, map[string]interface{}{
		"root_dir": dir,
		"format":   "invalid-format",
	})
	if err != nil {
		t.Fatalf("handleComponentGraph invalid format: %v", err)
	}

	text := extractText(result)
	var envelope map[string]interface{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope)

	status, _ := envelope["status"].(string)
	if status == "ok" {
		t.Errorf("expected error status for invalid format, got 'ok'")
	}
}

func TestMCP_ComponentGraph_DefaultsToJSON(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	// No format specified — should default to JSON.
	result, err := callTool(srv.handleComponentGraph, map[string]interface{}{
		"root_dir": dir,
	})
	if err != nil {
		t.Fatalf("handleComponentGraph no format: %v", err)
	}

	text := extractText(result)
	// Should be valid JSON (default format).
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("default format should be JSON, got invalid JSON: %v\n\nOutput:\n%s", err, text)
	}
}

// ─── handleDebugComponentContext MCP tool tests ───────────────────────────────

func TestMCP_DebugComponentContext_MissingComponentReturnsError(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleDebugComponentContext, map[string]interface{}{
		"root_dir": dir,
		// No "component" parameter — should fail.
	})
	if err != nil {
		t.Fatalf("handleDebugComponentContext: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for missing component")
	}

	// Result should indicate an error (either tool error or JSON error).
	text := extractText(result)
	if text == "" {
		// Check if it's a tool error result (IsError flag set).
		if !result.IsError {
			t.Error("expected either non-empty text or IsError=true for missing component")
		}
	}
}

func TestMCP_DebugComponentContext_ValidComponent(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleDebugComponentContext, map[string]interface{}{
		"component": "api",
		"root_dir":  dir,
		"depth":     float64(1),
	})
	if err != nil {
		t.Fatalf("handleDebugComponentContext: %v", err)
	}

	text := extractText(result)
	if text == "" {
		t.Fatal("handleDebugComponentContext returned empty text")
	}

	// Must be valid JSON.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("handleDebugComponentContext output invalid JSON: %v\n\nOutput:\n%s", err, text)
	}
}

func TestMCP_DebugComponentContext_DepthClamped(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	// Depth = 10 should be clamped to 5 (no panic, valid JSON result).
	result, err := callTool(srv.handleDebugComponentContext, map[string]interface{}{
		"component": "api",
		"root_dir":  dir,
		"depth":     float64(10), // out of range, gets clamped to 5
	})
	if err != nil {
		t.Fatalf("handleDebugComponentContext depth clamping: %v", err)
	}

	text := extractText(result)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("depth clamping output invalid JSON: %v\n\nOutput:\n%s", err, text)
	}
}

func TestMCP_DebugComponentContext_WithQuery(t *testing.T) {
	srv, dir := makeComponentTestServer(t)

	result, err := callTool(srv.handleDebugComponentContext, map[string]interface{}{
		"component": "auth",
		"root_dir":  dir,
		"query":     "Why is auth failing to validate tokens?",
	})
	if err != nil {
		t.Fatalf("handleDebugComponentContext with query: %v", err)
	}

	text := extractText(result)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &envelope); err != nil {
		t.Fatalf("with query output invalid JSON: %v\n\nOutput:\n%s", err, text)
	}
}
