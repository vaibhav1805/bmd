package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// buildTestGraph builds a minimal graph with the provided node IDs and edges.
// Each node gets a title derived from the filename stem.
func buildTestGraph(t *testing.T, nodes []string, edges [][2]string) *Graph {
	t.Helper()
	g := NewGraph()
	for _, id := range nodes {
		title := filenameStem(id)
		_ = g.AddNode(&Node{ID: id, Title: title, Type: "document"})
	}
	for _, e := range edges {
		edge, err := NewEdge(e[0], e[1], EdgeReferences, 1.0, "link")
		if err != nil {
			t.Fatalf("NewEdge(%q, %q) failed: %v", e[0], e[1], err)
		}
		if err := g.AddEdge(edge); err != nil {
			t.Fatalf("AddEdge failed: %v", err)
		}
	}
	return g
}

// buildTestDocs creates Document values from a map of ID → content.
func buildTestDocs(idToContent map[string]string) []Document {
	docs := make([]Document, 0, len(idToContent))
	for id, content := range idToContent {
		docs = append(docs, Document{
			ID:      id,
			RelPath: id,
			Title:   filenameStem(id),
			Content: content,
		})
	}
	return docs
}

// ─── Task 6.1: Registry creation and operations ───────────────────────────────

func TestRegistryAddComponent_Idempotent(t *testing.T) {
	r := NewComponentRegistry()
	comp := &RegistryComponent{
		ID:      "auth-service",
		Name:    "Auth Service",
		FileRef: "services/auth.md",
		Type:    ComponentTypeService,
	}
	if err := r.AddComponent(comp); err != nil {
		t.Fatalf("first AddComponent failed: %v", err)
	}
	// Add same component again — should replace, not duplicate.
	if err := r.AddComponent(comp); err != nil {
		t.Fatalf("second AddComponent failed: %v", err)
	}
	if r.ComponentCount() != 1 {
		t.Errorf("expected 1 component after duplicate add, got %d", r.ComponentCount())
	}
}

func TestRegistryAddSignal_SelfRelationship(t *testing.T) {
	r := NewComponentRegistry()
	err := r.AddSignal("auth", "auth", Signal{SourceType: SignalLink, Confidence: 1.0})
	if err == nil {
		t.Error("expected error for self-relationship, got nil")
	}
}

func TestRegistryAddSignal_MultipleSignals(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link", Weight: 1.0})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalMention, Confidence: 0.75, Evidence: "mention", Weight: 1.0})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLLM, Confidence: 0.65, Evidence: "llm", Weight: 1.0})

	rels := r.FindRelationships("a")
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
	if len(rels[0].Signals) != 3 {
		t.Errorf("expected 3 signals, got %d", len(rels[0].Signals))
	}
}

func TestRegistryGetComponent_Missing(t *testing.T) {
	r := NewComponentRegistry()
	comp := r.GetComponent("nonexistent")
	if comp != nil {
		t.Errorf("expected nil for missing component, got %+v", comp)
	}
}

func TestRegistryAggregateConfidence_MaxWins(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLLM, Confidence: 0.65, Weight: 1.0})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalMention, Confidence: 0.75, Weight: 1.0})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLink, Confidence: 1.0, Weight: 1.0})

	r.AggregateConfidence()

	rels := r.FindRelationships("a")
	if rels[0].AggregatedConfidence != 1.0 {
		t.Errorf("expected max confidence 1.0, got %.2f", rels[0].AggregatedConfidence)
	}
}

// ─── Task 6.1: JSON serialization round-trip ─────────────────────────────────

func TestRegistryJSONRoundTrip_EmptyRegistry(t *testing.T) {
	r := NewComponentRegistry()
	data, err := r.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	r2 := NewComponentRegistry()
	if err := r2.FromJSON(data); err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}
	if r2.ComponentCount() != 0 {
		t.Errorf("expected 0 components, got %d", r2.ComponentCount())
	}
	if r2.RelationshipCount() != 0 {
		t.Errorf("expected 0 relationships, got %d", r2.RelationshipCount())
	}
}

func TestRegistryJSONRoundTrip_WithData(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:      "auth",
		Name:    "Auth Service",
		FileRef: "services/auth.md",
		Type:    ComponentTypeService,
	})
	_ = r.AddComponent(&RegistryComponent{
		ID:      "cache",
		Name:    "Cache Service",
		FileRef: "services/cache.md",
		Type:    ComponentTypeDatabase,
	})
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link", Weight: 1.0})
	r.AggregateConfidence()

	data, err := r.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	r2 := NewComponentRegistry()
	if err := r2.FromJSON(data); err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if r2.ComponentCount() != 2 {
		t.Errorf("expected 2 components after round-trip, got %d", r2.ComponentCount())
	}
	if r2.RelationshipCount() != 1 {
		t.Errorf("expected 1 relationship after round-trip, got %d", r2.RelationshipCount())
	}

	// Verify index is rebuilt after FromJSON.
	rels := r2.FindRelationships("auth")
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship from auth, got %d — index may not have been rebuilt", len(rels))
	}
}

func TestRegistryJSONRoundTrip_PreservesConfidence(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalMention, Confidence: 0.75, Weight: 1.0})
	r.AggregateConfidence()

	data, err := r.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	r2 := NewComponentRegistry()
	if err := r2.FromJSON(data); err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	rels := r2.FindRelationships("a")
	if len(rels) == 0 {
		t.Fatal("expected relationships after round-trip")
	}
	if rels[0].AggregatedConfidence != 0.75 {
		t.Errorf("confidence not preserved: got %.2f, want 0.75", rels[0].AggregatedConfidence)
	}
}

func TestRegistrySaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, RegistryFileName)

	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:         "auth",
		Name:       "Auth Service",
		FileRef:    "auth.md",
		Type:       ComponentTypeService,
		DetectedAt: time.Now(),
	})
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link"})
	r.AggregateConfidence()

	if err := SaveRegistry(r, path); err != nil {
		t.Fatalf("SaveRegistry failed: %v", err)
	}

	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadRegistry returned nil")
	}
	if loaded.ComponentCount() != 1 {
		t.Errorf("expected 1 component after load, got %d", loaded.ComponentCount())
	}
	if loaded.RelationshipCount() != 1 {
		t.Errorf("expected 1 relationship after load, got %d", loaded.RelationshipCount())
	}
}

func TestLoadRegistry_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	r, err := LoadRegistry(path)
	if err != nil {
		t.Errorf("LoadRegistry should return nil, nil for missing file; got err: %v", err)
	}
	if r != nil {
		t.Errorf("LoadRegistry should return nil for missing file, got non-nil")
	}
}

// ─── Task 6.1: Mention extraction ────────────────────────────────────────────

func TestExtractMentionsFromDocument_Basic(t *testing.T) {
	doc := Document{
		ID:      "services/gateway.md",
		Content: "The API Gateway calls the auth service to validate tokens.",
	}
	components := []Component{
		{ID: "auth", Name: "Auth Service", File: "services/auth.md"},
		{ID: "cache", Name: "Cache Service", File: "services/cache.md"},
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) == 0 {
		t.Fatal("expected at least one mention")
	}

	found := false
	for _, m := range mentions {
		if m.ToComponent == "auth" {
			found = true
			if m.Confidence < 0.5 {
				t.Errorf("mention confidence too low: %.2f", m.Confidence)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected mention of 'auth', mentions: %+v", mentions)
	}
}

func TestExtractMentionsFromDocument_ExcludesSelf(t *testing.T) {
	doc := Document{
		ID:      "services/auth.md",
		Content: "The auth service handles login.",
	}
	components := []Component{
		{ID: "auth", Name: "Auth Service", File: "services/auth.md"},
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	for _, m := range mentions {
		if m.ToComponent == "auth" {
			t.Error("should not mention self-component")
		}
	}
}

func TestExtractMentionsFromDocuments_MultipleFiles(t *testing.T) {
	docs := buildTestDocs(map[string]string{
		"services/gateway.md": "Gateway calls auth to check tokens.\nAlso uses cache for sessions.",
		"services/api.md":     "API depends on auth-service for security.",
		"services/auth.md":    "Auth service handles tokens.",
	})
	components := []Component{
		{ID: "auth", Name: "Auth Service", File: "services/auth.md"},
		{ID: "cache", Name: "Cache Service", File: "services/cache.md"},
	}

	mentions := ExtractMentionsFromDocuments(docs, components)
	if len(mentions) == 0 {
		t.Fatal("expected mentions from multiple files")
	}
}

func TestMentionDeduplication_SameEvidenceSkipped(t *testing.T) {
	r := NewComponentRegistry()
	mentions := []Mention{
		{FromFile: "a", ToComponent: "b", Confidence: 0.7, ExampleEvidence: "calls b"},
		{FromFile: "a", ToComponent: "b", Confidence: 0.7, ExampleEvidence: "calls b"},
	}
	r.BuildFromMentions(mentions)

	rels := r.FindRelationships("a")
	if len(rels) == 0 {
		t.Fatal("expected at least one relationship")
	}
	for _, rel := range rels {
		if rel.ToComponent == "b" {
			mentionCount := 0
			for _, sig := range rel.Signals {
				if sig.SourceType == SignalMention {
					mentionCount++
				}
			}
			if mentionCount > 1 {
				t.Errorf("expected 1 deduplicated mention signal, got %d", mentionCount)
			}
		}
	}
}

// ─── Task 6.1: LLM extraction ─────────────────────────────────────────────────

func TestLLMExtractionWithMockPageIndex_NotFound(t *testing.T) {
	cfg := QueryLLMConfig{
		Enabled:      true,
		PageIndexBin: "/nonexistent/pageindex-does-not-exist",
		CachePath:    filepath.Join(t.TempDir(), LLMCacheFileName),
		SkipExisting: false,
		TimeoutSecs:  5,
	}
	docs := buildTestDocs(map[string]string{
		"auth.md": "The auth service depends on the database.",
	})
	components := []Component{
		{ID: "database", Name: "Database", File: "db.md"},
	}

	// With missing pageindex binary, should return empty and no error.
	rels, err := RunLLMExtraction(cfg, docs, components)
	if err != nil {
		t.Errorf("expected no error for missing pageindex binary, got: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships for missing binary, got %d", len(rels))
	}
}

func TestLLMCaching_WriteAndReadBack(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, LLMCacheFileName)

	rels := []LLMRelationship{
		{FromFile: "auth.md", ToComponent: "cache", Confidence: 0.65, Reasoning: "depends on", Evidence: "uses cache"},
		{FromFile: "gateway.md", ToComponent: "auth", Confidence: 0.70, Reasoning: "calls", Evidence: "auth required"},
	}

	if err := CacheLLMResults(rels, cachePath); err != nil {
		t.Fatalf("CacheLLMResults failed: %v", err)
	}

	loaded, err := LoadLLMCache(cachePath)
	if err != nil {
		t.Fatalf("LoadLLMCache failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 cached relationships, got %d", len(loaded))
	}
}

func TestLLMCaching_MissingCacheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent-cache.json")

	rels, err := LoadLLMCache(path)
	if err != nil {
		t.Errorf("expected nil error for missing cache, got: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships for missing cache, got %d", len(rels))
	}
}

func TestLLMFallback_EmptyDocuments(t *testing.T) {
	cfg := QueryLLMConfig{Enabled: true, CachePath: filepath.Join(t.TempDir(), LLMCacheFileName)}
	rels, err := RunLLMExtraction(cfg, nil, nil)
	if err != nil {
		t.Errorf("expected no error for empty docs, got: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships for empty docs, got %d", len(rels))
	}
}

func TestLLMSignalIntegration_AddedToRegistry(t *testing.T) {
	r := NewComponentRegistry()
	llmRels := []LLMRelationship{
		{FromFile: "gateway.md", ToComponent: "auth", Confidence: 0.65, Evidence: "LLM evidence"},
		{FromFile: "gateway.md", ToComponent: "cache", Confidence: 0.70, Evidence: "LLM evidence 2"},
	}
	r.BuildFromLLMExtraction(llmRels)

	if r.RelationshipCount() != 2 {
		t.Errorf("expected 2 relationships from LLM, got %d", r.RelationshipCount())
	}

	rels := r.FindRelationships("gateway.md")
	if len(rels) != 2 {
		t.Errorf("expected 2 outgoing relationships from gateway.md, got %d", len(rels))
	}
}

// ─── Task 6.1: Hybrid Builder ─────────────────────────────────────────────────

func TestBuildHybridGraph_SignalAggregation_AllSources(t *testing.T) {
	g := buildTestGraph(t,
		[]string{"services/auth.md", "services/cache.md"},
		[][2]string{{"services/auth.md", "services/cache.md"}},
	)

	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{ID: "auth", Name: "auth", FileRef: "services/auth.md"})
	_ = r.AddComponent(&RegistryComponent{ID: "cache", Name: "cache", FileRef: "services/cache.md"})
	// Add multiple signal sources.
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalLink, Confidence: 1.0, Weight: 1.0})
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalMention, Confidence: 0.75, Weight: 1.0})
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalLLM, Confidence: 0.65, Weight: 1.0})
	r.AggregateConfidence()

	builder := NewHybridBuilder()
	result := builder.BuildHybridGraph(r, g)
	if result == nil {
		t.Fatal("BuildHybridGraph returned nil graph")
	}

	// The edge should exist with max confidence (1.0 from link).
	edges := result.BySource["services/auth.md"]
	if len(edges) == 0 {
		t.Fatal("expected edges from auth node")
	}
	found := false
	for _, e := range edges {
		if e.Target == "services/cache.md" {
			found = true
			if e.Confidence < 1.0 {
				t.Errorf("expected confidence >= 1.0, got %.2f", e.Confidence)
			}
			break
		}
	}
	if !found {
		t.Error("expected edge from auth to cache")
	}
}

func TestEdgeMerging_HigherConfidenceWins(t *testing.T) {
	g := buildTestGraph(t,
		[]string{"a.md", "b.md"},
		[][2]string{{"a.md", "b.md"}},
	)
	// Original edge has confidence 0.5.
	for _, edges := range g.BySource {
		for _, e := range edges {
			e.Confidence = 0.5
		}
	}

	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{ID: "a", Name: "a", FileRef: "a.md"})
	_ = r.AddComponent(&RegistryComponent{ID: "b", Name: "b", FileRef: "b.md"})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalMention, Confidence: 0.9, Weight: 1.0})
	r.AggregateConfidence()

	builder := NewHybridBuilder()
	result := builder.BuildHybridGraph(r, g)

	edges := result.BySource["a.md"]
	for _, e := range edges {
		if e.Target == "b.md" {
			if e.Confidence < 0.9 {
				t.Errorf("expected confidence >= 0.9 (higher signal wins), got %.2f", e.Confidence)
			}
			return
		}
	}
	t.Error("expected edge from a.md to b.md")
}

func TestConfidenceAggregation_WeightedAverage(t *testing.T) {
	builder := &HybridBuilder{
		Strategy:      AggregationWeightedAverage,
		MinConfidence: 0.0,
	}
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Weight: 2.0},
		{SourceType: SignalMention, Confidence: 0.5, Weight: 1.0},
	}
	// Weighted average: (1.0*2.0 + 0.5*1.0) / (2.0+1.0) = 2.5/3.0 ≈ 0.833
	got := builder.AggregateSignals(signals)
	expected := 2.5 / 3.0
	if got < expected-0.01 || got > expected+0.01 {
		t.Errorf("expected weighted average ~%.3f, got %.3f", expected, got)
	}
}

func TestBackwardCompatibility_NilRegistry(t *testing.T) {
	g := buildTestGraph(t,
		[]string{"a.md", "b.md"},
		[][2]string{{"a.md", "b.md"}},
	)
	originalEdgeCount := 0
	for _, edges := range g.BySource {
		originalEdgeCount += len(edges)
	}

	builder := NewHybridBuilder()
	result := builder.BuildHybridGraph(nil, g)

	if result != g {
		t.Error("expected same graph pointer when registry is nil")
	}
	newEdgeCount := 0
	for _, edges := range result.BySource {
		newEdgeCount += len(edges)
	}
	if newEdgeCount != originalEdgeCount {
		t.Errorf("graph was modified with nil registry: was %d edges, now %d", originalEdgeCount, newEdgeCount)
	}
}

// ─── Task 6.1: Full pipeline end-to-end ──────────────────────────────────────

func TestFullPipeline_ExtractAggregateQueryBuild(t *testing.T) {
	// Build a test graph with three services.
	g := buildTestGraph(t,
		[]string{"services/auth.md", "services/cache.md", "services/database.md"},
		[][2]string{
			{"services/auth.md", "services/cache.md"},
			{"services/auth.md", "services/database.md"},
		},
	)

	docs := buildTestDocs(map[string]string{
		"services/auth.md":     "# Auth Service\n\nThe auth service uses the cache for sessions and the database for user data.",
		"services/cache.md":    "# Cache Service\n\nProvides fast in-memory caching.",
		"services/database.md": "# Database Service\n\nStores persistent data.",
	})

	// Full pipeline: init from graph (includes mention extraction).
	r := NewComponentRegistry()
	r.InitFromGraph(g, docs)

	// Should have discovered all three components.
	if r.ComponentCount() == 0 {
		t.Fatal("expected components after full pipeline")
	}

	// Should have relationships from graph edges.
	if r.RelationshipCount() == 0 {
		t.Fatal("expected relationships after full pipeline")
	}

	// Build hybrid graph and verify result.
	builder := NewHybridBuilder()
	result := builder.BuildHybridGraph(r, g)
	if result == nil {
		t.Fatal("BuildHybridGraph returned nil")
	}

	// Verify auth→cache and auth→database edges.
	edges := result.BySource["services/auth.md"]
	if len(edges) < 2 {
		t.Errorf("expected at least 2 edges from auth, got %d", len(edges))
	}
}

func TestMultiFileScenarios_FiveFiles(t *testing.T) {
	nodes := []string{
		"services/api.md",
		"services/auth.md",
		"services/cache.md",
		"services/database.md",
		"services/gateway.md",
	}
	edges := [][2]string{
		{"services/api.md", "services/auth.md"},
		{"services/api.md", "services/cache.md"},
		{"services/gateway.md", "services/api.md"},
		{"services/gateway.md", "services/auth.md"},
		{"services/auth.md", "services/database.md"},
	}
	g := buildTestGraph(t, nodes, edges)

	docs := buildTestDocs(map[string]string{
		"services/api.md":      "# API Service\n\nCalls auth and cache.",
		"services/auth.md":     "# Auth Service\n\nReads from database.",
		"services/cache.md":    "# Cache Service",
		"services/database.md": "# Database Service",
		"services/gateway.md":  "# Gateway Service\n\nRoutes to api and auth.",
	})

	r := NewComponentRegistry()
	r.InitFromGraph(g, docs)

	if r.ComponentCount() == 0 {
		t.Error("expected components for 5-file scenario")
	}
	if r.RelationshipCount() == 0 {
		t.Error("expected relationships for 5-file scenario")
	}
}

func TestMultiFileScenarios_TenFiles(t *testing.T) {
	nodes := make([]string, 10)
	for i := range nodes {
		nodes[i] = filepath.Join("services", fmt.Sprintf("service-%d.md", i))
	}
	// Build a chain: 0→1→2→...→9
	edges := make([][2]string, 9)
	for i := range edges {
		edges[i] = [2]string{nodes[i], nodes[i+1]}
	}

	g := buildTestGraph(t, nodes, edges)

	docContent := make(map[string]string, 10)
	for _, n := range nodes {
		docContent[n] = "# " + filenameStem(n) + "\n\nA service in the chain."
	}
	docs := buildTestDocs(docContent)

	r := NewComponentRegistry()
	r.InitFromGraph(g, docs)

	if r.ComponentCount() == 0 {
		t.Error("expected components for 10-file chain")
	}
}

func TestSignalDeduplication_SameRelationshipMultipleSources(t *testing.T) {
	r := NewComponentRegistry()
	// Same relationship from 3 sources — should be 1 relationship with 3 distinct signals.
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link", Weight: 1.0})
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalMention, Confidence: 0.75, Evidence: "mention text", Weight: 1.0})
	_ = r.AddSignal("auth", "cache", Signal{SourceType: SignalLLM, Confidence: 0.65, Evidence: "llm analysis", Weight: 1.0})
	r.AggregateConfidence()

	if r.RelationshipCount() != 1 {
		t.Errorf("expected 1 relationship for same pair (3 sources), got %d", r.RelationshipCount())
	}
	rels := r.FindRelationships("auth")
	if len(rels[0].Signals) != 3 {
		t.Errorf("expected 3 distinct signals, got %d", len(rels[0].Signals))
	}
}

func TestQueryByConfidence_FilteredCorrectly(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link"})
	_ = r.AddSignal("x", "y", Signal{SourceType: SignalLLM, Confidence: 0.65, Evidence: "llm"})
	_ = r.AddSignal("m", "n", Signal{SourceType: SignalMention, Confidence: 0.40, Evidence: "weak"})
	r.AggregateConfidence()

	highConf := r.QueryByConfidence(0.80)
	if len(highConf) != 1 {
		t.Errorf("expected 1 result with confidence >= 0.80, got %d", len(highConf))
	}

	midConf := r.QueryByConfidence(0.60)
	if len(midConf) != 2 {
		t.Errorf("expected 2 results with confidence >= 0.60, got %d", len(midConf))
	}

	allConf := r.QueryByConfidence(0.0)
	if len(allConf) != 3 {
		t.Errorf("expected all 3 results with confidence >= 0.0, got %d", len(allConf))
	}
}

// ─── Task 6.1: Performance benchmarks ────────────────────────────────────────

func TestRegistryInitPerformance_50Files(t *testing.T) {
	nodes := make([]string, 50)
	for i := range nodes {
		nodes[i] = filepath.Join("services", fmt.Sprintf("service-%d.md", i))
	}
	// Each node connects to the next two (fan-out structure).
	var edges [][2]string
	for i := 0; i < len(nodes)-1; i++ {
		edges = append(edges, [2]string{nodes[i], nodes[i+1]})
	}
	g := buildTestGraph(t, nodes, edges)

	docContent := make(map[string]string, 50)
	for _, n := range nodes {
		docContent[n] = "# " + filenameStem(n) + "\n\nService documentation."
	}
	docs := buildTestDocs(docContent)

	start := time.Now()
	r := NewComponentRegistry()
	r.InitFromGraph(g, docs)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("registry init for 50 files took %v; want <500ms", elapsed)
	}
	_ = r // avoid unused variable
}

func TestHybridGraphMergePerformance_100Edges(t *testing.T) {
	const nodeCount = 20
	nodes := make([]string, nodeCount)
	for i := range nodes {
		nodes[i] = filepath.Join("services", fmt.Sprintf("node-%d.md", i))
	}

	// Build ~100 edges: each node connects to ~5 others.
	var edges [][2]string
	for i := 0; i < nodeCount; i++ {
		for j := 1; j <= 5 && i+j < nodeCount; j++ {
			edges = append(edges, [2]string{nodes[i], nodes[i+j]})
		}
	}
	g := buildTestGraph(t, nodes, edges)

	// Build registry with same edges.
	r := NewComponentRegistry()
	for i, n := range nodes {
		_ = r.AddComponent(&RegistryComponent{
			ID:      fmt.Sprintf("node-%d", i),
			Name:    fmt.Sprintf("Node %d", i),
			FileRef: n,
		})
	}
	for _, e := range edges {
		fromComp := filenameStem(e[0])
		toComp := filenameStem(e[1])
		_ = r.AddSignal(fromComp, toComp, Signal{SourceType: SignalMention, Confidence: 0.75, Weight: 1.0})
	}
	r.AggregateConfidence()

	start := time.Now()
	builder := NewHybridBuilder()
	builder.BuildHybridGraph(r, g)
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Errorf("hybrid graph merge for ~%d edges took %v; want <1s", len(edges), elapsed)
	}
}

// ─── Task 6.1: Cache persistence ─────────────────────────────────────────────

func TestCachePersistenceAndReuse(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, LLMCacheFileName)

	// Write initial results.
	initialRels := []LLMRelationship{
		{FromFile: "a.md", ToComponent: "b", Confidence: 0.65, Evidence: "initial"},
	}
	if err := CacheLLMResults(initialRels, cachePath); err != nil {
		t.Fatalf("CacheLLMResults failed: %v", err)
	}

	// Write new results (should append / overwrite as new cache).
	newRels := []LLMRelationship{
		{FromFile: "a.md", ToComponent: "b", Confidence: 0.65, Evidence: "initial"},
		{FromFile: "c.md", ToComponent: "d", Confidence: 0.70, Evidence: "new"},
	}
	if err := CacheLLMResults(newRels, cachePath); err != nil {
		t.Fatalf("CacheLLMResults overwrite failed: %v", err)
	}

	loaded, err := LoadLLMCache(cachePath)
	if err != nil {
		t.Fatalf("LoadLLMCache failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 cached relationships, got %d", len(loaded))
	}
}

// ─── Task 6.3: Backward compatibility ────────────────────────────────────────

func TestBackwardCompatibility_GraphWorksWithoutRegistry(t *testing.T) {
	// Graph operations should work independently of the registry.
	g := buildTestGraph(t,
		[]string{"a.md", "b.md", "c.md"},
		[][2]string{{"a.md", "b.md"}, {"b.md", "c.md"}},
	)

	if g.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 2 {
		t.Errorf("expected 2 edges, got %d", g.EdgeCount())
	}

	// Merge nil registry — should be a no-op.
	if err := g.MergeRegistry(nil); err != nil {
		t.Errorf("MergeRegistry(nil) returned error: %v", err)
	}
	if g.EdgeCount() != 2 {
		t.Errorf("expected edge count unchanged, got %d", g.EdgeCount())
	}
}

func TestBackwardCompatibility_RegistryJSONIsValid(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:      "auth",
		Name:    "Auth Service",
		FileRef: "auth.md",
		Type:    ComponentTypeService,
	})
	_ = r.AddSignal("auth", "db", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link"})
	r.AggregateConfidence()

	data, err := r.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON.
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("registry JSON is not valid: %v\nJSON: %s", err, data)
	}

	// Verify top-level keys.
	if _, ok := raw["components"]; !ok {
		t.Error("registry JSON missing 'components' key")
	}
	if _, ok := raw["relationships"]; !ok {
		t.Error("registry JSON missing 'relationships' key")
	}
}

func TestBackwardCompatibility_LoadRegistryGracefulFallback(t *testing.T) {
	dir := t.TempDir()

	// Write invalid JSON to the registry file.
	path := filepath.Join(dir, RegistryFileName)
	if err := os.WriteFile(path, []byte("invalid json {{{"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// LoadRegistry should return an error (not crash).
	_, err := LoadRegistry(path)
	if err == nil {
		t.Error("expected error loading invalid registry JSON")
	}
}

// ─── Task 6.1: Commands integration ─────────────────────────────────────────

func TestComponentsList_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	content := "# Auth Component\n\nHandles authentication."
	filePath := filepath.Join(dir, "auth-component.md")
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdComponents([]string{"--dir", dir, "--format", "json"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdComponents failed: %v", err)
	}

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	if output == "" {
		t.Error("expected non-empty output from CmdComponents")
	}
}

// TODO: Uncomment when CmdRegistryCmd is fully implemented
/*
func TestRelationshipsQuery_NoRegistryFile(t *testing.T) {
	dir := t.TempDir()
	content := "# Auth Component\n\nHandles authentication."
	filePath := filepath.Join(dir, "auth-component.md")
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdRegistryCmd([]string{"--dir", dir, "--format", "json", "--from", "auth"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdRegistryCmd failed: %v", err)
	}

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	if output == "" {
		t.Error("expected non-empty output")
	}
}
*/

func TestDependsWithConfidence_MinConfidenceFilter(t *testing.T) {
	dir := t.TempDir()

	// Write two markdown files with a link between them.
	authContent := "# Auth Component\n\nSee [Database](database-component.md) for storage."
	dbContent := "# Database Component\n\nStores data."
	if err := os.WriteFile(filepath.Join(dir, "auth-component.md"), []byte(authContent), 0o600); err != nil {
		t.Fatalf("WriteFile auth failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "database-component.md"), []byte(dbContent), 0o600); err != nil {
		t.Fatalf("WriteFile database failed: %v", err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CmdDepends([]string{"auth", "--dir", dir, "--format", "json", "--min-confidence", "0.0"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CmdDepends failed: %v", err)
	}

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	if output == "" {
		t.Error("expected non-empty output from CmdDepends")
	}
}

// ─── Gitignore check ─────────────────────────────────────────────────────────

func TestGitignoreIncludesRegistryFiles(t *testing.T) {
	gitignorePath := "/Users/flurryhead/Developer/Opensource/bmd/.gitignore"
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Skipf("could not read .gitignore: %v", err)
	}

	content := string(data)
	patterns := []string{
		".bmd-registry.json",
		".bmd-llm-extractions.json",
	}
	for _, p := range patterns {
		found := false
		for _, line := range splitLines(content) {
			if line == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(".gitignore does not include pattern %q", p)
		}
	}
}

// splitLines splits content on newlines, trimming each line.
func splitLines(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
