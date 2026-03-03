package knowledge

import (
	"path/filepath"
	"testing"
)

// --- BuildFromLLMExtraction tests ---

func TestBuildFromLLMExtraction_Basic(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth Service", Type: ComponentTypeService})
	_ = r.AddComponent(&RegistryComponent{ID: "user", Name: "User Service", Type: ComponentTypeService})

	rels := []LLMRelationship{
		{
			FromFile:    "services/auth.md",
			ToComponent: "user",
			Confidence:  0.65,
			Reasoning:   "depends on",
			Evidence:    "The auth service depends on the user service",
		},
	}

	r.BuildFromLLMExtraction(rels)

	if r.RelationshipCount() != 1 {
		t.Errorf("expected 1 relationship, got %d", r.RelationshipCount())
	}
	found := r.FindRelationships("services/auth.md")
	if len(found) != 1 {
		t.Fatalf("expected 1 relationship from services/auth.md, got %d", len(found))
	}
	if found[0].AggregatedConfidence != 0.65 {
		t.Errorf("expected confidence 0.65, got %.2f", found[0].AggregatedConfidence)
	}
}

func TestBuildFromLLMExtraction_EmptyInput(t *testing.T) {
	r := NewComponentRegistry()
	r.BuildFromLLMExtraction(nil)
	if r.RelationshipCount() != 0 {
		t.Errorf("expected 0 relationships for nil input, got %d", r.RelationshipCount())
	}
	r.BuildFromLLMExtraction([]LLMRelationship{})
	if r.RelationshipCount() != 0 {
		t.Errorf("expected 0 relationships for empty input, got %d", r.RelationshipCount())
	}
}

func TestBuildFromLLMExtraction_SignalType(t *testing.T) {
	r := NewComponentRegistry()
	rels := []LLMRelationship{
		{FromFile: "a.md", ToComponent: "b", Confidence: 0.65, Evidence: "LLM evidence"},
	}
	r.BuildFromLLMExtraction(rels)

	found := r.FindRelationships("a.md")
	if len(found) == 0 {
		t.Fatal("expected relationships after BuildFromLLMExtraction")
	}

	var hasLLMSignal bool
	for _, sig := range found[0].Signals {
		if sig.SourceType == SignalLLM {
			hasLLMSignal = true
			break
		}
	}
	if !hasLLMSignal {
		t.Error("expected at least one SignalLLM signal")
	}
}

func TestBuildFromLLMExtraction_MultipleRelationships(t *testing.T) {
	r := NewComponentRegistry()
	rels := []LLMRelationship{
		{FromFile: "a.md", ToComponent: "b", Confidence: 0.65},
		{FromFile: "a.md", ToComponent: "c", Confidence: 0.70},
		{FromFile: "x.md", ToComponent: "y", Confidence: 0.60},
	}
	r.BuildFromLLMExtraction(rels)

	if r.RelationshipCount() != 3 {
		t.Errorf("expected 3 relationships, got %d", r.RelationshipCount())
	}
}

// --- GetLLMRelationships tests ---

func TestGetLLMRelationships_Basic(t *testing.T) {
	r := NewComponentRegistry()
	r.BuildFromLLMExtraction([]LLMRelationship{
		{FromFile: "a.md", ToComponent: "target", Confidence: 0.65, Evidence: "LLM"},
	})
	_ = r.AddSignal("b.md", "target", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link"})
	r.AggregateConfidence()

	llmRels := r.GetLLMRelationships("target")
	if len(llmRels) != 1 {
		t.Errorf("expected 1 LLM relationship targeting 'target', got %d", len(llmRels))
	}
	if llmRels[0].FromComponent != "a.md" {
		t.Errorf("unexpected FromComponent: %q", llmRels[0].FromComponent)
	}
}

func TestGetLLMRelationships_ExcludesNonLLM(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddSignal("a.md", "target", Signal{SourceType: SignalLink, Confidence: 1.0, Evidence: "link"})
	_ = r.AddSignal("b.md", "target", Signal{SourceType: SignalMention, Confidence: 0.7, Evidence: "mention"})
	r.AggregateConfidence()

	llmRels := r.GetLLMRelationships("target")
	if len(llmRels) != 0 {
		t.Errorf("expected 0 LLM relationships (no LLM signals), got %d", len(llmRels))
	}
}

func TestGetLLMRelationships_SortedByConfidence(t *testing.T) {
	r := NewComponentRegistry()
	r.BuildFromLLMExtraction([]LLMRelationship{
		{FromFile: "a.md", ToComponent: "target", Confidence: 0.60},
		{FromFile: "b.md", ToComponent: "target", Confidence: 0.75},
		{FromFile: "c.md", ToComponent: "target", Confidence: 0.65},
	})

	llmRels := r.GetLLMRelationships("target")
	if len(llmRels) != 3 {
		t.Fatalf("expected 3 LLM relationships, got %d", len(llmRels))
	}
	for i := 1; i < len(llmRels); i++ {
		if llmRels[i].AggregatedConfidence > llmRels[i-1].AggregatedConfidence {
			t.Errorf("results not sorted descending at index %d", i)
		}
	}
}

func TestGetLLMRelationships_MissingComponent(t *testing.T) {
	r := NewComponentRegistry()
	rels := r.GetLLMRelationships("nonexistent")
	if len(rels) != 0 {
		t.Errorf("expected 0 for missing component, got %d", len(rels))
	}
}

// --- Signal aggregation with LLM tests ---

func TestLLMSignalAggregation_LinkWinsOverLLM(t *testing.T) {
	r := NewComponentRegistry()
	// Link (1.0) should win over LLM (0.65) via max-confidence aggregation.
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLink, Confidence: 1.0, Weight: 1.0})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLLM, Confidence: 0.65, Weight: 1.0})
	r.AggregateConfidence()

	rels := r.FindRelationships("a")
	if len(rels) == 0 {
		t.Fatal("expected relationships")
	}
	if rels[0].AggregatedConfidence != 1.0 {
		t.Errorf("expected 1.0 (link wins), got %.2f", rels[0].AggregatedConfidence)
	}
}

func TestLLMSignalAggregation_MentionVsLLM(t *testing.T) {
	r := NewComponentRegistry()
	// Mention (0.7) should win over LLM (0.65) via max-confidence.
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalMention, Confidence: 0.7, Weight: 1.0})
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLLM, Confidence: 0.65, Weight: 1.0})
	r.AggregateConfidence()

	rels := r.FindRelationships("a")
	if rels[0].AggregatedConfidence != 0.7 {
		t.Errorf("expected 0.7 (mention wins over LLM), got %.2f", rels[0].AggregatedConfidence)
	}
}

func TestLLMSignalAggregation_LLMOnlyRelationship(t *testing.T) {
	r := NewComponentRegistry()
	// LLM-only relationship: confidence should be exactly the LLM confidence.
	_ = r.AddSignal("a", "b", Signal{SourceType: SignalLLM, Confidence: 0.65, Weight: 1.0})
	r.AggregateConfidence()

	rels := r.FindRelationships("a")
	if rels[0].AggregatedConfidence != 0.65 {
		t.Errorf("expected 0.65 (LLM only), got %.2f", rels[0].AggregatedConfidence)
	}
}

// --- InitFromGraphWithLLM tests ---

func TestInitFromGraphWithLLM_LLMDisabled(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "services/auth.md", Title: "Auth Service"})
	_ = g.AddNode(&Node{ID: "services/user.md", Title: "User Service"})
	edge, _ := NewEdge("services/auth.md", "services/user.md", EdgeReferences, 1.0, "link")
	_ = g.AddEdge(edge)

	r := NewComponentRegistry()
	// LLM disabled: behaves identically to InitFromGraph.
	r.InitFromGraphWithLLM(g, nil, QueryLLMConfig{Enabled: false})

	if r.ComponentCount() != 2 {
		t.Errorf("expected 2 components, got %d", r.ComponentCount())
	}
	if r.RelationshipCount() != 1 {
		t.Errorf("expected 1 relationship, got %d", r.RelationshipCount())
	}
}

func TestInitFromGraphWithLLM_MissingPageIndex_DoesNotBlock(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "services/auth.md", Title: "Auth Service"})
	_ = g.AddNode(&Node{ID: "services/user.md", Title: "User Service"})
	edge, _ := NewEdge("services/auth.md", "services/user.md", EdgeReferences, 1.0, "link")
	_ = g.AddEdge(edge)

	docs := []Document{
		{ID: "services/auth.md", Content: "The auth service calls the user service."},
	}

	r := NewComponentRegistry()
	llmCfg := QueryLLMConfig{
		Enabled:      true,
		PageIndexBin: "/nonexistent/pageindex-binary",
		CachePath:    filepath.Join(t.TempDir(), LLMCacheFileName),
		SkipExisting: false,
	}
	// Should not panic or block; graph+mention signals still work.
	r.InitFromGraphWithLLM(g, docs, llmCfg)

	// Components and graph relationships should still exist.
	if r.ComponentCount() != 2 {
		t.Errorf("expected 2 components after graceful LLM fallback, got %d", r.ComponentCount())
	}
}

// --- ParseRegistryArgs --with-llm flag tests ---
// TODO: Uncomment when ParseRegistryArgs is fully implemented

/*
func TestParseRegistryArgs_WithLLMFlag(t *testing.T) {
	a, err := ParseRegistryArgs([]string{"--with-llm"})
	if err != nil {
		t.Fatalf("ParseRegistryArgs failed: %v", err)
	}
	if !a.WithLLM {
		t.Error("expected WithLLM=true when --with-llm is passed")
	}
}

func TestParseRegistryArgs_WithLLMDefaults(t *testing.T) {
	a, err := ParseRegistryArgs([]string{})
	if err != nil {
		t.Fatalf("ParseRegistryArgs failed: %v", err)
	}
	if a.WithLLM {
		t.Error("expected WithLLM=false by default")
	}
	if a.LLMBin != "pageindex" {
		t.Errorf("expected default LLMBin 'pageindex', got %q", a.LLMBin)
	}
	if a.LLMModel != "claude-sonnet-4-5" {
		t.Errorf("expected default LLMModel 'claude-sonnet-4-5', got %q", a.LLMModel)
	}
}

func TestParseRegistryArgs_LLMBinFlag(t *testing.T) {
	a, err := ParseRegistryArgs([]string{"--with-llm", "--llm-bin", "/usr/local/bin/pageindex"})
	if err != nil {
		t.Fatalf("ParseRegistryArgs failed: %v", err)
	}
	if a.LLMBin != "/usr/local/bin/pageindex" {
		t.Errorf("expected LLMBin '/usr/local/bin/pageindex', got %q", a.LLMBin)
	}
}

func TestParseRegistryArgs_LLMModelFlag(t *testing.T) {
	a, err := ParseRegistryArgs([]string{"--with-llm", "--llm-model", "claude-opus-4-6"})
	if err != nil {
		t.Fatalf("ParseRegistryArgs failed: %v", err)
	}
	if a.LLMModel != "claude-opus-4-6" {
		t.Errorf("expected LLMModel 'claude-opus-4-6', got %q", a.LLMModel)
	}
}
*/

// --- registryComponentsToComponents helper tests ---

func TestRegistryComponentsToComponents(t *testing.T) {
	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:      "auth-service",
		Name:    "Auth Service",
		FileRef: "services/auth.md",
		Type:    ComponentTypeService,
	})
	_ = r.AddComponent(&RegistryComponent{
		ID:      "user-service",
		Name:    "User Service",
		FileRef: "services/user.md",
		Type:    ComponentTypeService,
	})

	components := registryComponentsToComponents(r)
	if len(components) != 2 {
		t.Errorf("expected 2 components, got %d", len(components))
	}

	idSet := make(map[string]bool)
	for _, c := range components {
		idSet[c.ID] = true
	}
	if !idSet["auth-service"] {
		t.Error("expected 'auth-service' in converted components")
	}
	if !idSet["user-service"] {
		t.Error("expected 'user-service' in converted components")
	}
}

func TestRegistryComponentsToComponents_Empty(t *testing.T) {
	r := NewComponentRegistry()
	components := registryComponentsToComponents(r)
	if len(components) != 0 {
		t.Errorf("expected 0 components for empty registry, got %d", len(components))
	}
}
