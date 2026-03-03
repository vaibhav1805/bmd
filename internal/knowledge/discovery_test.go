package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func makeTestDocWithTitle(id, title, content string) Document {
	return Document{
		ID:        id,
		RelPath:   id,
		Title:     title,
		Content:   content,
		PlainText: content,
	}
}

func TestDiscoverRelationships_ProducesEdges(t *testing.T) {
	docs := []Document{
		makeTestDocWithTitle("services/order-service.md", "Order Service",
			"# Order Service\n\n## Dependencies\n\n- User Service: validates ownership\n- Payment Service: processes payments\n\n## Overview\n\nThe Order Service coordinates with User Service and Payment Service."),
		makeTestDocWithTitle("services/user-service.md", "User Service",
			"# User Service\n\n## Related Services\n\n- Order Service: requires authentication\n- Payment Service: validates users\n"),
		makeTestDocWithTitle("services/payment-service.md", "Payment Service",
			"# Payment Service\n\n## Integration Points\n\n- User Service: validates customer info\n- Order Service: processes payments\n"),
	}

	edges := DiscoverRelationships(docs, nil)

	if len(edges) == 0 {
		t.Fatal("expected discovered relationships, got none")
	}

	t.Logf("Discovered %d relationships", len(edges))
	for _, e := range edges {
		t.Logf("  %s --[%s:%.2f]--> %s (signals: %d)",
			e.Source, e.Type, e.Confidence, e.Target, len(e.Signals))
	}
}

func TestDiscoverRelationships_NilInputs(t *testing.T) {
	edges := DiscoverRelationships(nil, nil)
	if len(edges) != 0 {
		t.Errorf("expected no edges for nil inputs, got %d", len(edges))
	}
}

func TestBuildComponentNameMap(t *testing.T) {
	docs := []Document{
		makeTestDocWithTitle("services/user-service.md", "User Service", ""),
		makeTestDocWithTitle("api/endpoints.md", "REST API Endpoints", ""),
		makeTestDocWithTitle("README.md", "README", ""),
	}

	names := BuildComponentNameMap(docs)

	// Should map title → ID.
	if names["User Service"] != "services/user-service.md" {
		t.Errorf("expected User Service → services/user-service.md, got %q", names["User Service"])
	}

	// Should map stem → ID.
	if names["user-service"] != "services/user-service.md" {
		t.Errorf("expected user-service → services/user-service.md, got %q", names["user-service"])
	}

	// Should map spaced stem → ID.
	if names["user service"] != "services/user-service.md" {
		t.Errorf("expected 'user service' → services/user-service.md, got %q", names["user service"])
	}

	// Should map title for endpoints.
	if names["REST API Endpoints"] != "api/endpoints.md" {
		t.Errorf("expected REST API Endpoints → api/endpoints.md, got %q", names["REST API Endpoints"])
	}
}

func TestMergeDiscoveredEdges_Deduplication(t *testing.T) {
	edge1, _ := NewEdge("a.md", "b.md", EdgeDependsOn, 0.60, "evidence 1")
	edge2, _ := NewEdge("a.md", "b.md", EdgeDependsOn, 0.80, "evidence 2")

	set1 := []*DiscoveredEdge{
		{
			Edge: edge1,
			Signals: []Signal{{
				SourceType: SignalCoOccurrence,
				Confidence: 0.60,
				Evidence:   "evidence 1",
				Weight:     1.0,
			}},
		},
	}

	set2 := []*DiscoveredEdge{
		{
			Edge: edge2,
			Signals: []Signal{{
				SourceType: SignalStructural,
				Confidence: 0.80,
				Evidence:   "evidence 2",
				Weight:     1.0,
			}},
		},
	}

	merged := MergeDiscoveredEdges(set1, set2)

	if len(merged) != 1 {
		t.Fatalf("expected 1 merged edge, got %d", len(merged))
	}

	// Should have both signals.
	if len(merged[0].Signals) != 2 {
		t.Errorf("expected 2 aggregated signals, got %d", len(merged[0].Signals))
	}

	// Should keep the highest confidence.
	if merged[0].Confidence != 0.80 {
		t.Errorf("expected merged confidence 0.80, got %.2f", merged[0].Confidence)
	}
}

func TestMergeDiscoveredEdges_DifferentTypes(t *testing.T) {
	edge1, _ := NewEdge("a.md", "b.md", EdgeDependsOn, 0.80, "depends")
	edge2, _ := NewEdge("a.md", "b.md", EdgeMentions, 0.60, "mentions")

	set1 := []*DiscoveredEdge{
		{Edge: edge1, Signals: []Signal{{SourceType: SignalStructural, Confidence: 0.80}}},
	}
	set2 := []*DiscoveredEdge{
		{Edge: edge2, Signals: []Signal{{SourceType: SignalCoOccurrence, Confidence: 0.60}}},
	}

	merged := MergeDiscoveredEdges(set1, set2)

	if len(merged) != 2 {
		t.Errorf("edges with different types should not be merged: expected 2, got %d", len(merged))
	}
}

func TestAddDiscoveredEdgesToRegistry(t *testing.T) {
	registry := NewComponentRegistry()
	_ = registry.AddComponent(&RegistryComponent{ID: "order-service", Name: "Order Service", FileRef: "services/order-service.md"})
	_ = registry.AddComponent(&RegistryComponent{ID: "user-service", Name: "User Service", FileRef: "services/user-service.md"})

	edge, _ := NewEdge("services/order-service.md", "services/user-service.md", EdgeDependsOn, 0.85, "depends on")
	edges := []*DiscoveredEdge{
		{
			Edge: edge,
			Signals: []Signal{{
				SourceType: SignalStructural,
				Confidence: 0.85,
				Evidence:   "Dependencies section",
				Weight:     1.0,
			}},
		},
	}

	AddDiscoveredEdgesToRegistry(registry, edges)

	rels := registry.FindRelationships("order-service")
	if len(rels) == 0 {
		t.Fatal("expected relationship in registry after adding discovered edges")
	}

	if rels[0].ToComponent != "user-service" {
		t.Errorf("expected relationship to user-service, got %s", rels[0].ToComponent)
	}
}

func TestDiscoverRelationships_IntegrationWithTestData(t *testing.T) {
	testDataDir := filepath.Join("..", "..", "test-data", "graph-test-docs")

	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skip("test-data/graph-test-docs not found, skipping integration test")
	}

	docs, err := ScanDirectory(testDataDir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	if len(docs) < 5 {
		t.Fatalf("expected at least 5 documents in test data, got %d", len(docs))
	}

	edges := DiscoverRelationships(docs, nil)

	t.Logf("Documents scanned: %d", len(docs))
	t.Logf("Relationships discovered: %d", len(edges))

	// Log all edges for visibility.
	for _, e := range edges {
		t.Logf("  %s --[%s:%.2f]--> %s (signals: %d, evidence: %q)",
			e.Source, e.Type, e.Confidence, e.Target, len(e.Signals), truncateEvidence(e.Evidence, 60))
	}

	// The test data has 12 files with rich cross-references.
	// We expect at least 20 discovered relationships.
	if len(edges) < 20 {
		t.Errorf("expected at least 20 discovered edges from test data, got %d", len(edges))
	}

	// Verify edges have signals.
	for _, e := range edges {
		if len(e.Signals) == 0 {
			t.Errorf("edge %s → %s has no signals", e.Source, e.Target)
		}
	}

	// Verify no self-loops.
	for _, e := range edges {
		if e.Source == e.Target {
			t.Errorf("self-loop: %s → %s", e.Source, e.Target)
		}
	}
}

func TestDiscoverRelationships_SignalAggregation(t *testing.T) {
	docs := []Document{
		{
			ID:      "services/order-service.md",
			RelPath: "services/order-service.md",
			Title:   "Order Service",
			Content: "# Order Service\n\n## Dependencies\n\n- User Service: auth\n\n## Overview\n\nThe Order Service uses User Service for authentication.",
			PlainText: "Order Service\n\nDependencies\n\nUser Service: auth\n\nOverview\n\nThe Order Service uses User Service for authentication.",
		},
	}

	componentNames := map[string]string{
		"User Service":  "services/user-service.md",
		"Order Service": "services/order-service.md",
	}

	edges := DiscoverRelationships(docs, componentNames)

	// Should find the relationship from both structural and co-occurrence.
	foundOrderToUser := false
	for _, e := range edges {
		if e.Source == "services/order-service.md" && e.Target == "services/user-service.md" {
			foundOrderToUser = true
			t.Logf("Order→User edge: type=%s confidence=%.2f signals=%d",
				e.Type, e.Confidence, len(e.Signals))
		}
	}

	if !foundOrderToUser {
		t.Error("expected Order Service → User Service relationship")
	}
}
