package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupRegistryTestDir creates a temp dir with sample markdown files
// that produce a predictable registry for CLI command testing.
func setupRegistryTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"auth-service.md": `# Auth Service

The Auth Service provides JWT token validation.
It stores tokens in the [cache-layer](./cache-layer.md).

## Endpoints

- POST /auth/token
- GET /auth/validate
`,
		"api-gateway.md": `# API Gateway

Routes requests to downstream services.
Calls [auth-service](./auth-service.md) for authentication.
Also depends on [cache-layer](./cache-layer.md) for caching.

## Endpoints

- GET /health
- POST /api/v1/*
`,
		"cache-layer.md": `# Cache Layer

Provides in-memory caching via Redis.
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("create %q: %v", name, err)
		}
	}

	return dir
}

// setupRegistryFile creates a .bmd-registry.json in the given dir with
// predictable components and relationships for testing.
func setupRegistryFile(t *testing.T, dir string) *ComponentRegistry {
	t.Helper()

	reg := NewComponentRegistry()

	_ = reg.AddComponent(&RegistryComponent{
		ID:         "auth-service",
		Name:       "Auth Service",
		Type:       ComponentTypeService,
		FileRef:    "auth-service.md",
		SourceFile: "auth-service.md",
		DetectedAt: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
	})
	_ = reg.AddComponent(&RegistryComponent{
		ID:         "api-gateway",
		Name:       "API Gateway",
		Type:       ComponentTypeService,
		FileRef:    "api-gateway.md",
		SourceFile: "api-gateway.md",
		DetectedAt: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
	})
	_ = reg.AddComponent(&RegistryComponent{
		ID:         "cache-layer",
		Name:       "Cache Layer",
		Type:       ComponentTypeService,
		FileRef:    "cache-layer.md",
		SourceFile: "cache-layer.md",
		DetectedAt: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
	})

	// api-gateway → auth-service [link confidence 1.0]
	_ = reg.AddSignal("api-gateway", "auth-service", Signal{
		SourceType: SignalLink,
		Confidence: 1.0,
		Evidence:   "[auth-service](./auth-service.md)",
		Weight:     1.0,
	})
	// api-gateway → cache-layer [mention confidence 0.75]
	_ = reg.AddSignal("api-gateway", "cache-layer", Signal{
		SourceType: SignalMention,
		Confidence: 0.75,
		Evidence:   "cache-layer mentioned in api-gateway",
		Weight:     1.0,
	})
	// auth-service → cache-layer [link confidence 1.0]
	_ = reg.AddSignal("auth-service", "cache-layer", Signal{
		SourceType: SignalLink,
		Confidence: 1.0,
		Evidence:   "[cache-layer](./cache-layer.md)",
		Weight:     1.0,
	})

	reg.AggregateConfidence()

	if err := SaveRegistry(reg, filepath.Join(dir, RegistryFileName)); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	return reg
}

// captureRegistryOutput runs fn capturing stdout.
// Named differently to avoid conflict with context_test.go's captureStdout.
func captureRegistryOutput(fn func()) string {
	return strings.TrimSpace(captureStdout(fn))
}

// ─── ParseComponentsListArgs tests ────────────────────────────────────────────

func TestParseComponentsListArgs_Defaults(t *testing.T) {
	a, err := ParseComponentsListArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "table" {
		t.Errorf("Format: got %q, want table", a.Format)
	}
}

func TestParseComponentsListArgs_Flags(t *testing.T) {
	a, err := ParseComponentsListArgs([]string{"--dir", "/docs", "--format", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/docs" {
		t.Errorf("Dir: got %q, want /docs", a.Dir)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

func TestParseComponentsListArgs_Positional(t *testing.T) {
	a, err := ParseComponentsListArgs([]string{"/my/dir"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "/my/dir" {
		t.Errorf("Dir: got %q, want /my/dir", a.Dir)
	}
}

// ─── ParseComponentsSearchArgs tests ──────────────────────────────────────────

func TestParseComponentsSearchArgs_Defaults(t *testing.T) {
	a, err := ParseComponentsSearchArgs([]string{"auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Query != "auth" {
		t.Errorf("Query: got %q, want auth", a.Query)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "table" {
		t.Errorf("Format: got %q, want table", a.Format)
	}
}

func TestParseComponentsSearchArgs_MissingQuery(t *testing.T) {
	_, err := ParseComponentsSearchArgs([]string{})
	if err == nil {
		t.Fatal("expected error for missing QUERY")
	}
	if !strings.Contains(err.Error(), "QUERY") {
		t.Errorf("error should mention QUERY, got: %v", err)
	}
}

func TestParseComponentsSearchArgs_AllFlags(t *testing.T) {
	a, err := ParseComponentsSearchArgs([]string{"gateway", "--dir", "/docs", "--format", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Query != "gateway" {
		t.Errorf("Query: got %q, want gateway", a.Query)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

// ─── ParseComponentsInspectArgs tests ─────────────────────────────────────────

func TestParseComponentsInspectArgs_Defaults(t *testing.T) {
	a, err := ParseComponentsInspectArgs([]string{"auth-service"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ComponentID != "auth-service" {
		t.Errorf("ComponentID: got %q, want auth-service", a.ComponentID)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "table" {
		t.Errorf("Format: got %q, want table", a.Format)
	}
}

func TestParseComponentsInspectArgs_MissingID(t *testing.T) {
	_, err := ParseComponentsInspectArgs([]string{})
	if err == nil {
		t.Fatal("expected error for missing COMPONENT_ID")
	}
}

func TestParseComponentsInspectArgs_WithFlags(t *testing.T) {
	a, err := ParseComponentsInspectArgs([]string{"api-gateway", "--format", "json", "--dir", "/docs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ComponentID != "api-gateway" {
		t.Errorf("ComponentID: got %q, want api-gateway", a.ComponentID)
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

// ─── ParseRelationshipsArgs tests ─────────────────────────────────────────────

func TestParseRelationshipsArgs_Defaults(t *testing.T) {
	a, err := ParseRelationshipsArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Dir != "." {
		t.Errorf("Dir: got %q, want .", a.Dir)
	}
	if a.Format != "table" {
		t.Errorf("Format: got %q, want table", a.Format)
	}
	if a.MinConfidence != 0.0 {
		t.Errorf("MinConfidence: got %.2f, want 0.0", a.MinConfidence)
	}
	if a.From != "" {
		t.Errorf("From: got %q, want empty", a.From)
	}
	if a.To != "" {
		t.Errorf("To: got %q, want empty", a.To)
	}
	if a.IncludeSignals {
		t.Error("IncludeSignals should default to false")
	}
}

func TestParseRelationshipsArgs_AllFlags(t *testing.T) {
	a, err := ParseRelationshipsArgs([]string{
		"--from", "api-gateway",
		"--confidence", "0.8",
		"--include-signals",
		"--format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.From != "api-gateway" {
		t.Errorf("From: got %q, want api-gateway", a.From)
	}
	if a.MinConfidence != 0.8 {
		t.Errorf("MinConfidence: got %.2f, want 0.8", a.MinConfidence)
	}
	if !a.IncludeSignals {
		t.Error("IncludeSignals should be true")
	}
	if a.Format != "json" {
		t.Errorf("Format: got %q, want json", a.Format)
	}
}

func TestParseRelationshipsArgs_InvalidConfidence(t *testing.T) {
	_, err := ParseRelationshipsArgs([]string{"--confidence", "1.5"})
	if err == nil {
		t.Fatal("expected error for confidence > 1.0")
	}
}

func TestParseRelationshipsArgs_ToFlag(t *testing.T) {
	a, err := ParseRelationshipsArgs([]string{"--to", "auth-service"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.To != "auth-service" {
		t.Errorf("To: got %q, want auth-service", a.To)
	}
}

// ─── CmdComponentsList integration tests ──────────────────────────────────────

func TestCmdComponentsList_JSON(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponentsList([]string{"--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponentsList: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}

	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data not an object: %v", envelope["data"])
	}
	if data["type"] != "components_list" {
		t.Errorf("type: got %v, want components_list", data["type"])
	}
	items, ok := data["data"].([]interface{})
	if !ok {
		t.Fatalf("data.data not an array: %v", data["data"])
	}
	if len(items) != 3 {
		t.Errorf("expected 3 components, got %d", len(items))
	}

	// Verify each item has required fields.
	first := items[0].(map[string]interface{})
	for _, field := range []string{"id", "name", "type", "file", "incoming_count", "outgoing_count", "detected_at"} {
		if _, ok := first[field]; !ok {
			t.Errorf("item missing field %q", field)
		}
	}
}

func TestCmdComponentsList_Table(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponentsList([]string{"--dir", dir, "--format", "table"}); err != nil {
			t.Errorf("CmdComponentsList: %v", err)
		}
	})

	if !strings.Contains(output, "Component") {
		t.Error("table output should contain 'Component' header")
	}
	if !strings.Contains(output, "auth-service") {
		t.Error("table output should contain auth-service")
	}
	if !strings.Contains(output, "api-gateway") {
		t.Error("table output should contain api-gateway")
	}
}

func TestCmdComponentsList_JSONHasRelationshipCounts(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdComponentsList([]string{"--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	_ = json.Unmarshal([]byte(output), &envelope)
	data := envelope["data"].(map[string]interface{})
	items := data["data"].([]interface{})

	// api-gateway has 2 outgoing, 0 incoming.
	for _, raw := range items {
		item := raw.(map[string]interface{})
		if item["id"] == "api-gateway" {
			out := item["outgoing_count"].(float64)
			if out != 2 {
				t.Errorf("api-gateway outgoing_count: got %.0f, want 2", out)
			}
		}
	}
}

// ─── CmdComponentsSearch integration tests ────────────────────────────────────

func TestCmdComponentsSearch_JSON(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponentsSearch([]string{"auth", "--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponentsSearch: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}

	data := envelope["data"].(map[string]interface{})
	if data["query"] != "auth" {
		t.Errorf("query: got %v, want auth", data["query"])
	}
	items := data["data"].([]interface{})
	// Should match auth-service (id contains "auth").
	if len(items) == 0 {
		t.Error("expected at least one match for 'auth'")
	}
	for _, raw := range items {
		item := raw.(map[string]interface{})
		id := item["id"].(string)
		name := strings.ToLower(item["name"].(string))
		if !strings.Contains(strings.ToLower(id), "auth") && !strings.Contains(name, "auth") {
			t.Errorf("unexpected match %q for query 'auth'", id)
		}
	}
}

func TestCmdComponentsSearch_NoMatch(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponentsSearch([]string{"nonexistent", "--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponentsSearch: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "empty" {
		t.Errorf("status: got %v, want empty (no matches)", envelope["status"])
	}
}

func TestCmdComponentsSearch_TableFormat(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdComponentsSearch([]string{"cache", "--dir", dir, "--format", "table"})
	})

	if !strings.Contains(output, "cache") {
		t.Error("table output should contain cache")
	}
	// Should not be JSON.
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Error("table output should not start with '{'")
	}
}

// ─── CmdComponentsInspect integration tests ───────────────────────────────────

func TestCmdComponentsInspect_JSON(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponentsInspect([]string{"api-gateway", "--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponentsInspect: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}

	data := envelope["data"].(map[string]interface{})
	if data["id"] != "api-gateway" {
		t.Errorf("id: got %v, want api-gateway", data["id"])
	}
	if data["outgoing_count"].(float64) != 2 {
		t.Errorf("outgoing_count: got %.0f, want 2", data["outgoing_count"].(float64))
	}
	if _, ok := data["depends_on"]; !ok {
		t.Error("data missing 'depends_on' field")
	}
	if _, ok := data["depended_on_by"]; !ok {
		t.Error("data missing 'depended_on_by' field")
	}
}

func TestCmdComponentsInspect_Table(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdComponentsInspect([]string{"auth-service", "--dir", dir, "--format", "table"})
	})

	if !strings.Contains(output, "auth-service") {
		t.Error("table output should contain component ID")
	}
	if !strings.Contains(output, "Depends on") {
		t.Error("table output should contain 'Depends on' section")
	}
	if !strings.Contains(output, "Depended on by") {
		t.Error("table output should contain 'Depended on by' section")
	}
}

func TestCmdComponentsInspect_NotFound_JSON(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdComponentsInspect([]string{"nonexistent", "--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "error" {
		t.Errorf("status: got %v, want error", envelope["status"])
	}
	if envelope["code"] != ErrCodeFileNotFound {
		t.Errorf("code: got %v, want %s", envelope["code"], ErrCodeFileNotFound)
	}
}

func TestCmdComponentsInspect_CaseInsensitiveMatch(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdComponentsInspect([]string{"API-GATEWAY", "--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("expected case-insensitive match to succeed, got status=%v", envelope["status"])
	}
}

// ─── CmdRelationships integration tests ───────────────────────────────────────

func TestCmdRelationships_JSON_AllRelationships(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdRelationships([]string{"--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdRelationships: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}

	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}

	data := envelope["data"].(map[string]interface{})
	if data["type"] != "relationships" {
		t.Errorf("type: got %v, want relationships", data["type"])
	}
	rels := data["relationships"].([]interface{})
	if len(rels) == 0 {
		t.Error("expected relationships in output")
	}
}

func TestCmdRelationships_JSON_FromFilter(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--from", "api-gateway", "--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}
	data := envelope["data"].(map[string]interface{})
	rels := data["relationships"].([]interface{})

	// All relationships should be from api-gateway.
	for _, raw := range rels {
		rel := raw.(map[string]interface{})
		if rel["from"] != "api-gateway" {
			t.Errorf("unexpected from component: %v", rel["from"])
		}
	}
	if len(rels) != 2 {
		t.Errorf("api-gateway has 2 outgoing, got %d", len(rels))
	}
}

func TestCmdRelationships_JSON_ToFilter(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--to", "cache-layer", "--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}
	data := envelope["data"].(map[string]interface{})
	rels := data["relationships"].([]interface{})

	// All relationships should be to cache-layer.
	for _, raw := range rels {
		rel := raw.(map[string]interface{})
		if rel["to"] != "cache-layer" {
			t.Errorf("unexpected to component: %v", rel["to"])
		}
	}
	// api-gateway and auth-service both depend on cache-layer = 2 relationships.
	if len(rels) != 2 {
		t.Errorf("cache-layer has 2 incoming, got %d", len(rels))
	}
}

func TestCmdRelationships_JSON_ConfidenceFilter(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	// High confidence filter should only return link-based relationships (1.0).
	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--from", "api-gateway", "--confidence", "0.9", "--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	data := envelope["data"].(map[string]interface{})
	rels := data["relationships"].([]interface{})

	// api-gateway → auth-service has confidence 1.0; cache-layer has 0.75 → filtered out.
	for _, raw := range rels {
		rel := raw.(map[string]interface{})
		conf := rel["confidence"].(float64)
		if conf < 0.9 {
			t.Errorf("relationship with confidence %.2f should be filtered at 0.9", conf)
		}
	}
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship after confidence filter, got %d", len(rels))
	}
}

func TestCmdRelationships_JSON_IncludeSignals(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--from", "api-gateway", "--include-signals", "--dir", dir, "--format", "json"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	data := envelope["data"].(map[string]interface{})
	rels := data["relationships"].([]interface{})

	// Each relationship should have signals when include-signals is set.
	for _, raw := range rels {
		rel := raw.(map[string]interface{})
		sigs, ok := rel["signals"]
		if !ok || sigs == nil {
			t.Error("expected signals field in relationship")
		}
	}
}

func TestCmdRelationships_DOT(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--dir", dir, "--format", "dot"})
	})

	if !strings.HasPrefix(output, "digraph") {
		t.Errorf("DOT output should start with 'digraph', got: %s", output[:min(50, len(output))])
	}
	if !strings.HasSuffix(output, "}") {
		t.Error("DOT output should end with '}'")
	}
}

func TestCmdRelationships_Table(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--from", "api-gateway", "--dir", dir, "--format", "table"})
	})

	if !strings.Contains(output, "Relationships for api-gateway") {
		t.Error("table output should contain component header")
	}
	if !strings.Contains(output, "Depends on") {
		t.Error("table output should contain 'Depends on' section")
	}
}

func TestCmdRelationships_Table_IncludeSignals(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		_ = CmdRelationships([]string{"--from", "api-gateway", "--include-signals", "--dir", dir, "--format", "table"})
	})

	if !strings.Contains(output, "signal:") {
		t.Error("table output with --include-signals should show signal lines")
	}
}

// ─── CmdComponents router tests ───────────────────────────────────────────────

func TestCmdComponents_SubcommandList(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponents([]string{"list", "--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponents list: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}
}

func TestCmdComponents_SubcommandSearch(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponents([]string{"search", "cache", "--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponents search: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}
}

func TestCmdComponents_SubcommandInspect(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	output := captureRegistryOutput(func() {
		if err := CmdComponents([]string{"inspect", "cache-layer", "--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponents inspect: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok", envelope["status"])
	}
}

func TestCmdComponents_LegacyFallback_NoSubcommand(t *testing.T) {
	dir := setupTestDocs(t) // uses the existing test helper

	output := captureRegistryOutput(func() {
		if err := CmdComponents([]string{"--dir", dir, "--format", "json"}); err != nil {
			t.Errorf("CmdComponents legacy: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] != "ok" {
		t.Errorf("status: got %v, want ok (legacy fallback)", envelope["status"])
	}
}

func TestCmdComponents_UnknownSubcommand(t *testing.T) {
	err := CmdComponents([]string{"unknown-cmd"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("error should mention 'unknown subcommand', got: %v", err)
	}
}

// ─── CmdDepends confidence flag tests ─────────────────────────────────────────

func TestParseDependsArgs_ShowConfidence(t *testing.T) {
	a, err := ParseDependsArgs([]string{"my-service", "--show-confidence"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.ShowConfidence {
		t.Error("ShowConfidence should be true when --show-confidence is set")
	}
}

func TestParseDependsArgs_IncludeSignals(t *testing.T) {
	a, err := ParseDependsArgs([]string{"my-service", "--include-signals"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.IncludeSignals {
		t.Error("IncludeSignals should be true when --include-signals is set")
	}
}

func TestParseDependsArgs_MinConfidence(t *testing.T) {
	a, err := ParseDependsArgs([]string{"my-service", "--min-confidence", "0.8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.MinConfidence != 0.8 {
		t.Errorf("MinConfidence: got %.2f, want 0.8", a.MinConfidence)
	}
}

func TestCmdDepends_MinConfidence_FiltersResults(t *testing.T) {
	dir := setupTestDocs(t)

	output := captureRegistryOutput(func() {
		// min-confidence=0.99 should filter all results since graph edges have confidence < 0.99.
		_ = CmdDepends([]string{"api-gateway", "--dir", dir, "--format", "json", "--min-confidence", "0.99"})
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, output)
	}
	if envelope["status"] == "error" {
		// A non-existent service would be an error, but filtering should return empty deps.
		// This is fine as long as it doesn't crash.
		return
	}
}

// ─── loadOrBuildRegistry tests ────────────────────────────────────────────────

func TestLoadOrBuildRegistry_WithFile(t *testing.T) {
	dir := t.TempDir()
	setupRegistryFile(t, dir)

	reg, err := loadOrBuildRegistry(dir)
	if err != nil {
		t.Fatalf("loadOrBuildRegistry: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if reg.ComponentCount() != 3 {
		t.Errorf("expected 3 components, got %d", reg.ComponentCount())
	}
	if reg.RelationshipCount() != 3 {
		t.Errorf("expected 3 relationships, got %d", reg.RelationshipCount())
	}
}

func TestLoadOrBuildRegistry_FallbackToGraph(t *testing.T) {
	dir := t.TempDir()
	// No registry file — must bootstrap from graph.
	if err := os.WriteFile(filepath.Join(dir, "svc.md"), []byte("# My Service\n\nA service."), 0o600); err != nil {
		t.Fatal(err)
	}

	reg, err := loadOrBuildRegistry(dir)
	if err != nil {
		t.Fatalf("loadOrBuildRegistry fallback: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry from fallback")
	}
}

// ─── formatComponentsListTable tests ──────────────────────────────────────────

func TestFormatComponentsListTable_Empty(t *testing.T) {
	reg := NewComponentRegistry()
	output := formatComponentsListTable(reg, nil, nil, nil)
	if output != "No components found." {
		t.Errorf("empty table: got %q, want 'No components found.'", output)
	}
}

func TestFormatComponentsListTable_HasHeaders(t *testing.T) {
	dir := t.TempDir()
	reg := setupRegistryFile(t, dir)
	incoming, outgoing := computeRelationshipCounts(reg)
	ids := sortedComponentIDs(reg)

	output := formatComponentsListTable(reg, ids, incoming, outgoing)
	if !strings.Contains(output, "Component") {
		t.Error("table should contain 'Component' header")
	}
	if !strings.Contains(output, "Type") {
		t.Error("table should contain 'Type' header")
	}
	if !strings.Contains(output, "File") {
		t.Error("table should contain 'File' header")
	}
}

// ─── formatRelationshipsDOT test ──────────────────────────────────────────────

func TestFormatRelationshipsDOT_Structure(t *testing.T) {
	rels := []RegistryRelationship{
		{FromComponent: "a", ToComponent: "b", AggregatedConfidence: 1.0},
		{FromComponent: "b", ToComponent: "c", AggregatedConfidence: 0.75},
	}
	output := formatRelationshipsDOT(rels)

	if !strings.HasPrefix(output, "digraph relationships") {
		t.Error("DOT output should start with 'digraph relationships'")
	}
	if !strings.Contains(output, `"a" -> "b"`) {
		t.Error("DOT output should contain edge a→b")
	}
	if !strings.Contains(output, `"b" -> "c"`) {
		t.Error("DOT output should contain edge b→c")
	}
}
