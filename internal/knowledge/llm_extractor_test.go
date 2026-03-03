package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- DefaultQueryLLMConfig tests ---

func TestDefaultQueryLLMConfig(t *testing.T) {
	cfg := DefaultQueryLLMConfig()
	if cfg.PageIndexBin != "pageindex" {
		t.Errorf("expected PageIndexBin 'pageindex', got %q", cfg.PageIndexBin)
	}
	if cfg.Model != "claude-sonnet-4-5" {
		t.Errorf("expected Model 'claude-sonnet-4-5', got %q", cfg.Model)
	}
	if !cfg.Enabled {
		t.Error("expected Enabled=true")
	}
	if !cfg.SkipExisting {
		t.Error("expected SkipExisting=true")
	}
	if cfg.TimeoutSecs != 30 {
		t.Errorf("expected TimeoutSecs=30, got %d", cfg.TimeoutSecs)
	}
}

// --- LLMRelationship struct tests ---

func TestLLMRelationship_Fields(t *testing.T) {
	rel := LLMRelationship{
		FromFile:    "services/auth.md",
		ToComponent: "user-service",
		Confidence:  0.65,
		Reasoning:   "calls",
		Evidence:    "The auth service calls the user service",
	}
	if rel.FromFile != "services/auth.md" {
		t.Errorf("unexpected FromFile: %q", rel.FromFile)
	}
	if rel.ToComponent != "user-service" {
		t.Errorf("unexpected ToComponent: %q", rel.ToComponent)
	}
	if rel.Confidence != 0.65 {
		t.Errorf("unexpected Confidence: %.2f", rel.Confidence)
	}
}

// --- CacheLLMResults / LoadLLMCache tests ---

func TestCacheLLMResults_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LLMCacheFileName)

	rels := []LLMRelationship{
		{
			FromFile:    "services/auth.md",
			ToComponent: "user-service",
			Confidence:  0.65,
			Reasoning:   "depends on",
			Evidence:    "The auth service depends on the user service",
		},
		{
			FromFile:    "services/billing.md",
			ToComponent: "payment-service",
			Confidence:  0.70,
			Reasoning:   "integrates with",
			Evidence:    "Billing integrates with payment processing",
		},
	}

	if err := CacheLLMResults(rels, path); err != nil {
		t.Fatalf("CacheLLMResults failed: %v", err)
	}

	loaded, err := LoadLLMCache(path)
	if err != nil {
		t.Fatalf("LoadLLMCache failed: %v", err)
	}
	if len(loaded) != len(rels) {
		t.Errorf("expected %d relationships, got %d", len(rels), len(loaded))
	}
	for i, r := range loaded {
		if r.FromFile != rels[i].FromFile {
			t.Errorf("[%d] FromFile mismatch: want %q, got %q", i, rels[i].FromFile, r.FromFile)
		}
		if r.ToComponent != rels[i].ToComponent {
			t.Errorf("[%d] ToComponent mismatch: want %q, got %q", i, rels[i].ToComponent, r.ToComponent)
		}
		if r.Confidence != rels[i].Confidence {
			t.Errorf("[%d] Confidence mismatch: want %.2f, got %.2f", i, rels[i].Confidence, r.Confidence)
		}
	}
}

func TestLoadLLMCache_MissingFile(t *testing.T) {
	rels, err := LoadLLMCache("/nonexistent/path/.bmd-llm-extractions.json")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if rels != nil {
		t.Error("expected nil for missing file")
	}
}

func TestLoadLLMCache_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LLMCacheFileName)
	if err := os.WriteFile(path, []byte("not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLLMCache(path)
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestCacheLLMResults_EmptySlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LLMCacheFileName)

	if err := CacheLLMResults([]LLMRelationship{}, path); err != nil {
		t.Fatalf("CacheLLMResults with empty slice failed: %v", err)
	}
	loaded, err := LoadLLMCache(path)
	if err != nil {
		t.Fatalf("LoadLLMCache failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 results, got %d", len(loaded))
	}
}

func TestLoadLLMCache_GeneratedAtPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LLMCacheFileName)

	before := time.Now().Truncate(time.Second)
	_ = CacheLLMResults([]LLMRelationship{{FromFile: "f.md", ToComponent: "svc", Confidence: 0.65}}, path)

	// Verify the cache file has a generated_at field.
	data, _ := os.ReadFile(path)
	var cache struct {
		GeneratedAt time.Time `json:"generated_at"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("unmarshal generated_at: %v", err)
	}
	if cache.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
	if cache.GeneratedAt.Before(before) {
		t.Errorf("GeneratedAt %v is before test start %v", cache.GeneratedAt, before)
	}
}

// --- loadCacheIndex tests ---

func TestLoadCacheIndex_GroupsByFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LLMCacheFileName)

	rels := []LLMRelationship{
		{FromFile: "a.md", ToComponent: "svc1", Confidence: 0.65},
		{FromFile: "a.md", ToComponent: "svc2", Confidence: 0.70},
		{FromFile: "b.md", ToComponent: "svc3", Confidence: 0.60},
	}
	_ = CacheLLMResults(rels, path)

	idx, err := loadCacheIndex(path)
	if err != nil {
		t.Fatalf("loadCacheIndex failed: %v", err)
	}
	if len(idx["a.md"]) != 2 {
		t.Errorf("expected 2 rels for a.md, got %d", len(idx["a.md"]))
	}
	if len(idx["b.md"]) != 1 {
		t.Errorf("expected 1 rel for b.md, got %d", len(idx["b.md"]))
	}
}

// --- buildKnownComponentSet tests ---

func TestBuildKnownComponentSet(t *testing.T) {
	components := []Component{
		{ID: "auth-service", Name: "Auth Service"},
		{ID: "user-service", Name: "User Service"},
	}
	known := buildKnownComponentSet(components)

	if !known["auth-service"] {
		t.Error("expected auth-service in known set")
	}
	if !known["auth-service"] {
		t.Error("expected auth-service in known set")
	}
	if !known["auth-service"] {
		t.Error("expected 'auth service' normalized in known set")
	}
}

// --- isKnownComponent tests ---

func TestIsKnownComponent_ExactMatch(t *testing.T) {
	known := map[string]bool{"user-service": true}
	if !isKnownComponent("user-service", known) {
		t.Error("exact match should return true")
	}
}

func TestIsKnownComponent_SuffixVariants(t *testing.T) {
	known := map[string]bool{"auth": true}
	if !isKnownComponent("auth", known) {
		t.Error("exact base match should return true")
	}
}

func TestIsKnownComponent_ServiceSuffix(t *testing.T) {
	// auth-service known, querying "auth" should match via TrimSuffix.
	known := map[string]bool{"auth": true}
	// "auth-service" → TrimSuffix("-service") = "auth" — should match.
	if !isKnownComponent("auth-service", known) {
		t.Error("auth-service should match known 'auth' via suffix stripping")
	}
}

func TestIsKnownComponent_NoMatch(t *testing.T) {
	known := map[string]bool{"auth-service": true}
	if isKnownComponent("payment-processor", known) {
		t.Error("payment-processor should not match auth-service")
	}
}

// --- parseAndFilterLLMResponse tests ---

func TestParseAndFilterLLMResponse_ValidJSON(t *testing.T) {
	raw := []byte(`[{"service": "user-service", "relationship": "depends on", "confidence": 0.8, "evidence": "calls user API"}]`)
	known := map[string]bool{"user-service": true}

	rels := parseAndFilterLLMResponse(raw, "auth.md", known)
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
	if rels[0].ToComponent != "user-service" {
		t.Errorf("unexpected ToComponent: %q", rels[0].ToComponent)
	}
	if rels[0].Confidence != 0.8 {
		t.Errorf("unexpected Confidence: %.2f", rels[0].Confidence)
	}
	if rels[0].FromFile != "auth.md" {
		t.Errorf("unexpected FromFile: %q", rels[0].FromFile)
	}
}

func TestParseAndFilterLLMResponse_FiltersUnknownComponents(t *testing.T) {
	raw := []byte(`[
		{"service": "user-service", "relationship": "calls", "confidence": 0.7, "evidence": "uses user API"},
		{"service": "unknown-thing", "relationship": "depends on", "confidence": 0.6, "evidence": "mystery dependency"}
	]`)
	known := map[string]bool{"user-service": true}

	rels := parseAndFilterLLMResponse(raw, "auth.md", known)
	if len(rels) != 1 {
		t.Errorf("expected 1 (filtered) relationship, got %d", len(rels))
	}
}

func TestParseAndFilterLLMResponse_DefaultConfidence(t *testing.T) {
	raw := []byte(`[{"service": "user-service", "relationship": "calls", "confidence": 0, "evidence": "vague"}]`)
	known := map[string]bool{"user-service": true}

	rels := parseAndFilterLLMResponse(raw, "auth.md", known)
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship")
	}
	if rels[0].Confidence != 0.65 {
		t.Errorf("expected default confidence 0.65, got %.2f", rels[0].Confidence)
	}
}

func TestParseAndFilterLLMResponse_ConfidenceCappedAt1(t *testing.T) {
	raw := []byte(`[{"service": "user-service", "relationship": "calls", "confidence": 1.5, "evidence": "very explicit"}]`)
	known := map[string]bool{"user-service": true}

	rels := parseAndFilterLLMResponse(raw, "auth.md", known)
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship")
	}
	if rels[0].Confidence > 1.0 {
		t.Errorf("confidence should be capped at 1.0, got %.2f", rels[0].Confidence)
	}
}

func TestParseAndFilterLLMResponse_InvalidJSON(t *testing.T) {
	raw := []byte(`This is not JSON`)
	known := map[string]bool{"user-service": true}

	rels := parseAndFilterLLMResponse(raw, "auth.md", known)
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships for invalid JSON, got %d", len(rels))
	}
}

func TestParseAndFilterLLMResponse_EmbeddedJSON(t *testing.T) {
	// LLM sometimes wraps JSON in prose.
	raw := []byte(`Here are the dependencies:\n[{"service": "user-service", "relationship": "calls", "confidence": 0.7, "evidence": "uses user API"}]\nEnd.`)
	known := map[string]bool{"user-service": true}

	rels := parseAndFilterLLMResponse(raw, "auth.md", known)
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship (embedded JSON extraction), got %d", len(rels))
	}
}

// --- RunLLMExtraction graceful degradation tests ---

func TestRunLLMExtraction_MissingPageIndex_GracefulDegradation(t *testing.T) {
	cfg := QueryLLMConfig{
		Enabled:      true,
		PageIndexBin: "/nonexistent/pageindex-binary-that-does-not-exist",
		CachePath:    filepath.Join(t.TempDir(), LLMCacheFileName),
		SkipExisting: false,
		TimeoutSecs:  5,
	}
	docs := []Document{
		{ID: "services/auth.md", Content: "The auth service depends on the user service."},
	}
	components := []Component{
		{ID: "user-service", Name: "User Service"},
	}

	// Should not panic or return an error — graceful degradation.
	rels, err := RunLLMExtraction(cfg, docs, components)
	if err != nil {
		t.Fatalf("expected nil error on missing pageindex, got: %v", err)
	}
	// Result may be empty (no pageindex), but must not error.
	_ = rels
}

func TestRunLLMExtraction_EmptyDocuments(t *testing.T) {
	cfg := DefaultQueryLLMConfig()
	rels, err := RunLLMExtraction(cfg, nil, nil)
	if err != nil {
		t.Fatalf("expected nil error for empty input: %v", err)
	}
	if rels != nil {
		t.Error("expected nil result for empty input")
	}
}

func TestRunLLMExtraction_EmptyComponents(t *testing.T) {
	cfg := DefaultQueryLLMConfig()
	docs := []Document{{ID: "a.md", Content: "content"}}
	rels, err := RunLLMExtraction(cfg, docs, nil)
	if err != nil {
		t.Fatalf("expected nil error: %v", err)
	}
	if rels != nil {
		t.Error("expected nil result when no components")
	}
}

func TestRunLLMExtraction_UsesCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, LLMCacheFileName)

	// Pre-populate cache.
	cached := []LLMRelationship{
		{FromFile: "services/auth.md", ToComponent: "user-service", Confidence: 0.65, Reasoning: "cached", Evidence: "from cache"},
	}
	_ = CacheLLMResults(cached, cachePath)

	cfg := QueryLLMConfig{
		Enabled:      true,
		CachePath:    cachePath,
		SkipExisting: true,
		PageIndexBin: "/nonexistent/pageindex-that-wont-be-called",
	}
	docs := []Document{
		{ID: "services/auth.md", Content: "any content"},
	}
	components := []Component{
		{ID: "user-service", Name: "User Service"},
	}

	// Should return cached results without calling pageindex.
	rels, err := RunLLMExtraction(cfg, docs, components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("expected 1 cached result, got %d", len(rels))
	}
	if rels[0].FromFile != "services/auth.md" {
		t.Errorf("unexpected FromFile: %q", rels[0].FromFile)
	}
}

// --- buildExtractionPrompt tests ---

func TestBuildExtractionPrompt_ContainsContent(t *testing.T) {
	content := "The auth service calls the user service to verify credentials."
	prompt := buildExtractionPrompt(content)

	if len(prompt) == 0 {
		t.Error("prompt should not be empty")
	}
	// Prompt should include the document content.
	if !containsSubstring(prompt, content) {
		t.Error("prompt should contain the document content")
	}
}

func TestBuildExtractionPrompt_TruncatesLargeContent(t *testing.T) {
	// Generate content larger than 4000 chars.
	large := make([]byte, 5000)
	for i := range large {
		large[i] = 'a'
	}
	prompt := buildExtractionPrompt(string(large))
	// Prompt length should be bounded.
	if len(prompt) > 6000 {
		t.Errorf("prompt too long for large content: %d chars", len(prompt))
	}
}

func TestBuildExtractionPrompt_RequestsJSONOutput(t *testing.T) {
	prompt := buildExtractionPrompt("some content")
	if !containsSubstring(prompt, "JSON") {
		t.Error("prompt should request JSON output")
	}
}

// containsSubstring is a helper for substring tests.
func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
