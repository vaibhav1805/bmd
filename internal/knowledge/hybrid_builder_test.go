package knowledge

import (
	"fmt"
	"testing"
)

// ─── AggregateSignals tests ───────────────────────────────────────────────────

func TestAggregateSignals_Empty(t *testing.T) {
	h := NewHybridBuilder()
	got := h.AggregateSignals(nil)
	if got != 0.0 {
		t.Errorf("AggregateSignals(nil) = %.4f; want 0.0", got)
	}

	got = h.AggregateSignals([]Signal{})
	if got != 0.0 {
		t.Errorf("AggregateSignals([]) = %.4f; want 0.0", got)
	}
}

func TestAggregateSignals_SingleSignal(t *testing.T) {
	h := NewHybridBuilder()
	sig := Signal{SourceType: SignalLink, Confidence: 0.9, Weight: 1.0}
	got := h.AggregateSignals([]Signal{sig})
	if got != 0.9 {
		t.Errorf("AggregateSignals single = %.4f; want 0.9", got)
	}
}

func TestAggregateSignals_MaxStrategy(t *testing.T) {
	h := NewHybridBuilder() // default: AggregationMax
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Weight: 1.0},
		{SourceType: SignalMention, Confidence: 0.75, Weight: 1.0},
		{SourceType: SignalLLM, Confidence: 0.65, Weight: 1.0},
	}
	got := h.AggregateSignals(signals)
	if got != 1.0 {
		t.Errorf("AggregateSignals max = %.4f; want 1.0", got)
	}
}

func TestAggregateSignals_MaxStrategy_PicksHighest(t *testing.T) {
	h := NewHybridBuilder()
	signals := []Signal{
		{SourceType: SignalMention, Confidence: 0.65, Weight: 1.0},
		{SourceType: SignalLLM, Confidence: 0.80, Weight: 1.0},
	}
	got := h.AggregateSignals(signals)
	if got != 0.80 {
		t.Errorf("AggregateSignals max = %.4f; want 0.80", got)
	}
}

func TestAggregateSignals_WeightedAverage(t *testing.T) {
	h := NewHybridBuilder()
	h.Strategy = AggregationWeightedAverage
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Weight: 2.0},
		{SourceType: SignalMention, Confidence: 0.5, Weight: 1.0},
	}
	// weighted avg = (1.0*2.0 + 0.5*1.0) / (2.0 + 1.0) = 2.5/3 ≈ 0.8333
	got := h.AggregateSignals(signals)
	want := 2.5 / 3.0
	if diff := got - want; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("AggregateSignals weighted = %.6f; want %.6f", got, want)
	}
}

func TestAggregateSignals_ThresholdFiltering(t *testing.T) {
	h := NewHybridBuilder()
	h.MinConfidence = 0.7
	signals := []Signal{
		{SourceType: SignalMention, Confidence: 0.4, Weight: 1.0}, // below threshold
		{SourceType: SignalLLM, Confidence: 0.6, Weight: 1.0},     // below threshold
	}
	got := h.AggregateSignals(signals)
	if got != 0.0 {
		t.Errorf("AggregateSignals threshold filtered = %.4f; want 0.0", got)
	}
}

func TestAggregateSignals_ThresholdFiltering_SomePass(t *testing.T) {
	h := NewHybridBuilder()
	h.MinConfidence = 0.7
	signals := []Signal{
		{SourceType: SignalMention, Confidence: 0.4, Weight: 1.0}, // filtered
		{SourceType: SignalLink, Confidence: 0.9, Weight: 1.0},    // passes
	}
	got := h.AggregateSignals(signals)
	if got != 0.9 {
		t.Errorf("AggregateSignals partial threshold = %.4f; want 0.9", got)
	}
}

func TestAggregateSignals_CapAt1(t *testing.T) {
	h := NewHybridBuilder()
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Weight: 2.0}, // weight pushes score over 1.0
	}
	got := h.AggregateSignals(signals)
	if got > 1.0 {
		t.Errorf("AggregateSignals cap = %.4f; want <= 1.0", got)
	}
}

func TestAggregateSignals_DefaultWeight(t *testing.T) {
	h := NewHybridBuilder()
	// Weight=0 should be treated as 1.0.
	signals := []Signal{
		{SourceType: SignalMention, Confidence: 0.75, Weight: 0},
	}
	got := h.AggregateSignals(signals)
	if got != 0.75 {
		t.Errorf("AggregateSignals zero weight = %.4f; want 0.75", got)
	}
}

// ─── MergeEdgeConfidences tests ───────────────────────────────────────────────

func TestMergeEdgeConfidences_HigherSignalWins(t *testing.T) {
	h := NewHybridBuilder()
	edge := &Edge{
		ID: "a\x00b\x00references", Source: "a", Target: "b",
		Type: EdgeReferences, Confidence: 0.5,
	}
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 0.9, Weight: 1.0},
	}
	got := h.MergeEdgeConfidences(edge, signals)
	if got < 0.5 {
		t.Errorf("MergeEdgeConfidences = %.4f; want >= 0.5 (edge confidence preserved)", got)
	}
}

func TestMergeEdgeConfidences_ExistingHigherPreserved(t *testing.T) {
	h := NewHybridBuilder()
	edge := &Edge{
		ID: "a\x00b\x00references", Source: "a", Target: "b",
		Type: EdgeReferences, Confidence: 0.95,
	}
	signals := []Signal{
		{SourceType: SignalMention, Confidence: 0.6, Weight: 1.0},
	}
	// Max of [0.95 existing, 0.6 mention] = 0.95
	got := h.MergeEdgeConfidences(edge, signals)
	if got != 0.95 {
		t.Errorf("MergeEdgeConfidences = %.4f; want 0.95 (existing confidence preserved)", got)
	}
}

// ─── UpdateEdgeConfidence tests ───────────────────────────────────────────────

func TestUpdateEdgeConfidence_Success(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Title: "A", Type: "document"})
	_ = g.AddNode(&Node{ID: "b", Title: "B", Type: "document"})
	edge, _ := NewEdge("a", "b", EdgeReferences, 0.5, "test")
	_ = g.AddEdge(edge)

	err := g.UpdateEdgeConfidence("a", "b", 0.9)
	if err != nil {
		t.Fatalf("UpdateEdgeConfidence: %v", err)
	}
	// Verify the edge was updated.
	for _, e := range g.BySource["a"] {
		if e.Target == "b" && e.Confidence != 0.9 {
			t.Errorf("UpdateEdgeConfidence: edge confidence = %.4f; want 0.9", e.Confidence)
		}
	}
}

func TestUpdateEdgeConfidence_NotFound(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Title: "A", Type: "document"})
	_ = g.AddNode(&Node{ID: "b", Title: "B", Type: "document"})

	err := g.UpdateEdgeConfidence("a", "b", 0.9)
	if err == nil {
		t.Error("UpdateEdgeConfidence: expected error for non-existent edge")
	}
}

func TestUpdateEdgeConfidence_InvalidRange(t *testing.T) {
	g := NewGraph()
	err := g.UpdateEdgeConfidence("a", "b", 1.5)
	if err == nil {
		t.Error("UpdateEdgeConfidence: expected error for out-of-range confidence")
	}
}

// ─── BuildHybridGraph tests ───────────────────────────────────────────────────

func makeSampleGraph() *Graph {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "services/auth.md", Title: "Auth Service", Type: "document"})
	_ = g.AddNode(&Node{ID: "services/api-gateway.md", Title: "API Gateway", Type: "document"})
	_ = g.AddNode(&Node{ID: "services/user.md", Title: "User Service", Type: "document"})
	edge, _ := NewEdge("services/api-gateway.md", "services/auth.md", EdgeReferences, 0.5, "link")
	_ = g.AddEdge(edge)
	return g
}

func makeSampleRegistry() *ComponentRegistry {
	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:      "auth",
		Name:    "Auth Service",
		FileRef: "services/auth.md",
		Type:    ComponentTypeService,
	})
	_ = r.AddComponent(&RegistryComponent{
		ID:      "api-gateway",
		Name:    "API Gateway",
		FileRef: "services/api-gateway.md",
		Type:    ComponentTypeService,
	})
	_ = r.AddComponent(&RegistryComponent{
		ID:      "user",
		Name:    "User Service",
		FileRef: "services/user.md",
		Type:    ComponentTypeService,
	})
	return r
}

func TestBuildHybridGraph_NilRegistry(t *testing.T) {
	g := makeSampleGraph()
	origEdgeCount := g.EdgeCount()
	h := NewHybridBuilder()
	result := h.BuildHybridGraph(nil, g)
	if result != g {
		t.Error("BuildHybridGraph(nil) should return the same graph pointer")
	}
	if result.EdgeCount() != origEdgeCount {
		t.Errorf("BuildHybridGraph(nil): edge count changed from %d to %d", origEdgeCount, result.EdgeCount())
	}
}

func TestBuildHybridGraph_UpdatesExistingEdgeConfidence(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()

	// Add a high-confidence link signal for the existing edge (api-gateway → auth).
	_ = r.AddSignal("api-gateway", "auth", Signal{
		SourceType: SignalLink,
		Confidence: 1.0,
		Weight:     1.0,
	})
	r.AggregateConfidence()

	h := NewHybridBuilder()
	result := h.BuildHybridGraph(r, g)

	// Find the edge from api-gateway to auth.
	var updated *Edge
	for _, e := range result.BySource["services/api-gateway.md"] {
		if e.Target == "services/auth.md" {
			updated = e
			break
		}
	}
	if updated == nil {
		t.Fatal("BuildHybridGraph: existing edge not found after merge")
	}
	if updated.Confidence < 0.9 {
		t.Errorf("BuildHybridGraph: edge confidence = %.4f; want >= 0.9 after high-confidence signal", updated.Confidence)
	}
}

func TestBuildHybridGraph_AddsNewEdge(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()

	// Add a signal for a relationship NOT in the base graph.
	_ = r.AddSignal("user", "auth", Signal{
		SourceType: SignalMention,
		Confidence: 0.75,
		Weight:     1.0,
	})
	r.AggregateConfidence()

	h := NewHybridBuilder()
	result := h.BuildHybridGraph(r, g)

	// Expect a new edge from user → auth.
	var newEdge *Edge
	for _, e := range result.BySource["services/user.md"] {
		if e.Target == "services/auth.md" {
			newEdge = e
			break
		}
	}
	if newEdge == nil {
		t.Fatal("BuildHybridGraph: expected new edge user→auth not found")
	}
	if newEdge.Confidence < 0.5 {
		t.Errorf("BuildHybridGraph: new edge confidence = %.4f; want >= 0.5", newEdge.Confidence)
	}
}

func TestBuildHybridGraph_SkipsUnresolvableComponents(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()

	// Add a component with a FileRef not in the graph.
	_ = r.AddComponent(&RegistryComponent{
		ID:      "ghost",
		Name:    "Ghost Service",
		FileRef: "services/ghost.md", // not in graph
		Type:    ComponentTypeService,
	})
	_ = r.AddSignal("ghost", "auth", Signal{
		SourceType: SignalLLM,
		Confidence: 0.65,
		Weight:     1.0,
	})
	r.AggregateConfidence()

	origEdgeCount := g.EdgeCount()
	h := NewHybridBuilder()
	result := h.BuildHybridGraph(r, g)

	// No new edge should have been added (ghost not in graph).
	if result.EdgeCount() != origEdgeCount {
		t.Errorf("BuildHybridGraph: edge count changed unexpectedly from %d to %d",
			origEdgeCount, result.EdgeCount())
	}
}

func TestBuildHybridGraph_SkipsSelfLoops(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()

	// Self-loop signal — should be skipped.
	_ = r.AddSignal("auth", "auth", Signal{
		SourceType: SignalMention,
		Confidence: 0.7,
		Weight:     1.0,
	})
	// Note: AddSignal returns an error for self-loops, so the signal won't be added.
	// Test that this doesn't panic or corrupt graph.
	origEdgeCount := g.EdgeCount()
	h := NewHybridBuilder()
	result := h.BuildHybridGraph(r, g)
	if result.EdgeCount() != origEdgeCount {
		t.Errorf("BuildHybridGraph self-loop: edge count changed from %d to %d",
			origEdgeCount, result.EdgeCount())
	}
}

func TestBuildHybridGraph_MultipleSignals_AggregatesCorrectly(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()

	// Add multiple signals for the same relationship.
	_ = r.AddSignal("user", "auth", Signal{
		SourceType: SignalMention, Confidence: 0.6, Weight: 1.0,
	})
	_ = r.AddSignal("user", "auth", Signal{
		SourceType: SignalLLM, Confidence: 0.8, Weight: 1.0,
	})
	r.AggregateConfidence()

	h := NewHybridBuilder()
	result := h.BuildHybridGraph(r, g)

	// Should create a new edge with max(0.6, 0.8) = 0.8 confidence.
	var newEdge *Edge
	for _, e := range result.BySource["services/user.md"] {
		if e.Target == "services/auth.md" {
			newEdge = e
			break
		}
	}
	if newEdge == nil {
		t.Fatal("BuildHybridGraph multi-signal: edge not found")
	}
	// max of 0.6 and 0.8 = 0.8
	if newEdge.Confidence < 0.79 {
		t.Errorf("BuildHybridGraph multi-signal: confidence = %.4f; want >= 0.79", newEdge.Confidence)
	}
}

// ─── MergeRegistry convenience method tests ───────────────────────────────────

func TestMergeRegistry_NilNoOp(t *testing.T) {
	g := makeSampleGraph()
	origEdges := g.EdgeCount()
	err := g.MergeRegistry(nil)
	if err != nil {
		t.Errorf("MergeRegistry(nil): unexpected error: %v", err)
	}
	if g.EdgeCount() != origEdges {
		t.Errorf("MergeRegistry(nil): edge count changed from %d to %d", origEdges, g.EdgeCount())
	}
}

func TestMergeRegistry_AddsRegistryEdges(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()
	_ = r.AddSignal("user", "api-gateway", Signal{
		SourceType: SignalMention, Confidence: 0.7, Weight: 1.0,
	})
	r.AggregateConfidence()

	err := g.MergeRegistry(r)
	if err != nil {
		t.Fatalf("MergeRegistry: %v", err)
	}

	// Expect a new edge user → api-gateway.
	found := false
	for _, e := range g.BySource["services/user.md"] {
		if e.Target == "services/api-gateway.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("MergeRegistry: expected new edge user→api-gateway not found")
	}
}

// ─── buildComponentToNodeMap tests ───────────────────────────────────────────

func TestBuildComponentToNodeMap_PrimaryFileRef(t *testing.T) {
	g := makeSampleGraph()
	r := makeSampleRegistry()

	m := buildComponentToNodeMap(r, g)
	if m["auth"] != "services/auth.md" {
		t.Errorf("buildComponentToNodeMap auth = %q; want %q", m["auth"], "services/auth.md")
	}
}

func TestBuildComponentToNodeMap_FallbackByStem(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "docs/payments.md", Title: "Payments", Type: "document"})

	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:      "payments",
		Name:    "Payments Service",
		FileRef: "nonexistent.md", // won't match
		Type:    ComponentTypeService,
	})

	m := buildComponentToNodeMap(r, g)
	if m["payments"] != "docs/payments.md" {
		t.Errorf("buildComponentToNodeMap stem fallback = %q; want %q", m["payments"], "docs/payments.md")
	}
}

func TestBuildComponentToNodeMap_UnresolvableReturnsNoEntry(t *testing.T) {
	g := makeSampleGraph()
	r := NewComponentRegistry()
	_ = r.AddComponent(&RegistryComponent{
		ID:      "nonexistent",
		Name:    "Nonexistent",
		FileRef: "nonexistent.md",
		Type:    ComponentTypeUnknown,
	})

	m := buildComponentToNodeMap(r, g)
	if _, ok := m["nonexistent"]; ok {
		t.Error("buildComponentToNodeMap: unresolvable component should not appear in map")
	}
}

// ─── buildEvidenceSummary tests ───────────────────────────────────────────────

func TestBuildEvidenceSummary_Empty(t *testing.T) {
	got := buildEvidenceSummary(nil)
	if got == "" {
		t.Error("buildEvidenceSummary(nil): want non-empty string")
	}
}

func TestBuildEvidenceSummary_SingleWithEvidence(t *testing.T) {
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Evidence: "see link in docs"},
	}
	got := buildEvidenceSummary(signals)
	if got == "" {
		t.Error("buildEvidenceSummary: want non-empty string")
	}
}

func TestBuildEvidenceSummary_MultipleDedupeTypes(t *testing.T) {
	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0},
		{SourceType: SignalLink, Confidence: 0.9},    // duplicate type
		{SourceType: SignalMention, Confidence: 0.7},
	}
	got := buildEvidenceSummary(signals)
	if got == "" {
		t.Error("buildEvidenceSummary multi: want non-empty string")
	}
}

// ─── backward compatibility tests ────────────────────────────────────────────

func TestGraph_ExistingOperationsUnaffected(t *testing.T) {
	// Verify that hybrid builder additions don't break existing graph operations.
	g := makeSampleGraph()
	r := makeSampleRegistry()
	_ = r.AddSignal("user", "auth", Signal{
		SourceType: SignalMention, Confidence: 0.7, Weight: 1.0,
	})
	r.AggregateConfidence()

	h := NewHybridBuilder()
	h.BuildHybridGraph(r, g)

	// TraverseBFS should still work.
	nodes := g.TraverseBFS("services/api-gateway.md", 2)
	if nodes == nil {
		t.Error("TraverseBFS after hybrid merge returned nil")
	}

	// DetectCycles should still work.
	_ = g.DetectCycles()

	// NodeCount/EdgeCount should be sane.
	if g.NodeCount() < 3 {
		t.Errorf("NodeCount after merge = %d; want >= 3", g.NodeCount())
	}
	if g.EdgeCount() < 1 {
		t.Errorf("EdgeCount after merge = %d; want >= 1", g.EdgeCount())
	}
}

// ─── performance sanity test ──────────────────────────────────────────────────

func TestBuildHybridGraph_LargeGraphPerformance(t *testing.T) {
	// Build a graph with 100 nodes and 100 edges.
	g := NewGraph()
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("services/svc-%02d.md", i)
		_ = g.AddNode(&Node{ID: id, Title: id, Type: "document"})
	}
	for i := 0; i < 100; i++ {
		src := fmt.Sprintf("services/svc-%02d.md", i)
		dst := fmt.Sprintf("services/svc-%02d.md", (i+1)%100)
		if src == dst {
			continue
		}
		e, _ := NewEdge(src, dst, EdgeReferences, 0.7, "perf test")
		if e != nil {
			_ = g.AddEdge(e)
		}
	}

	// Build a registry with signals for all relationships.
	r := NewComponentRegistry()
	for i := 0; i < 100; i++ {
		compID := fmt.Sprintf("svc-%02d", i)
		fileRef := fmt.Sprintf("services/svc-%02d.md", i)
		_ = r.AddComponent(&RegistryComponent{
			ID:      compID,
			Name:    compID,
			FileRef: fileRef,
			Type:    ComponentTypeService,
		})
	}
	for i := 0; i < 100; i++ {
		fromID := fmt.Sprintf("svc-%02d", i)
		toID := fmt.Sprintf("svc-%02d", (i+1)%100)
		if fromID == toID {
			continue
		}
		_ = r.AddSignal(fromID, toID, Signal{
			SourceType: SignalMention, Confidence: 0.8, Weight: 1.0,
		})
	}
	r.AggregateConfidence()

	h := NewHybridBuilder()
	// If this completes without timeout, performance is acceptable.
	result := h.BuildHybridGraph(r, g)
	if result == nil {
		t.Error("BuildHybridGraph large graph returned nil")
	}
}
