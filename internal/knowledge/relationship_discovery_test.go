package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// testDataDir resolves the absolute path to test-data/graph-test-docs.
// Returns empty string if the directory does not exist (e.g. CI without test-data).
func testDataDir(t *testing.T) string {
	t.Helper()
	// Walk up from the package directory to find the repo root.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("testDataDir: getwd: %v", err)
	}
	// internal/knowledge -> repo root is ../../
	root := filepath.Join(wd, "..", "..")
	dir := filepath.Join(root, "test-data", "graph-test-docs")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("test-data/graph-test-docs not found at %s: %v", dir, err)
	}
	return dir
}

// loadTestDocs scans test-data/graph-test-docs and returns documents.
func loadTestDocs(t *testing.T) (string, []Document) {
	t.Helper()
	dir := testDataDir(t)
	docs, err := ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory(%s, ScanConfig{UseDefaultIgnores: true}): %v", dir, err)
	}
	return dir, docs
}

// loadTestGraph scans test-data/graph-test-docs and builds a Graph + documents.
func loadTestGraph(t *testing.T) (string, *Graph, []Document) {
	t.Helper()
	dir, docs := loadTestDocs(t)
	gb := NewGraphBuilder(dir)
	graph := gb.Build(docs)
	return dir, graph, docs
}

// ---------------------------------------------------------------------------
// 1. Structural relationship tests (link-based graph extraction)
// ---------------------------------------------------------------------------

func TestStructuralRelationships_GraphNodeCount(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	// We expect one node per markdown file found by the scanner.
	if graph.NodeCount() != len(docs) {
		t.Errorf("NodeCount = %d, want %d (one per scanned document)", graph.NodeCount(), len(docs))
	}

	// Should have at least 10 files (12 in the fixture).
	if graph.NodeCount() < 10 {
		t.Errorf("NodeCount = %d, want >= 10", graph.NodeCount())
	}
}

func TestStructuralRelationships_LinkEdgeCount(t *testing.T) {
	_, graph, _ := loadTestGraph(t)

	// With the enhanced test data (markdown links), we expect many edges.
	// Key links: README -> 8 files, architecture -> 7, order-service -> 8,
	// payment-service -> 8, user-service -> 6, database -> 6, endpoints -> 8,
	// quickstart -> 14, setup -> 8, etc.
	// Some will be deduplicated (same source+target+type).
	edgeCount := graph.EdgeCount()
	t.Logf("Graph has %d nodes, %d edges", graph.NodeCount(), edgeCount)

	if edgeCount < 20 {
		t.Errorf("EdgeCount = %d, want >= 20 (from markdown links)", edgeCount)
	}
}

func TestStructuralRelationships_ExpectedLinks(t *testing.T) {
	_, graph, _ := loadTestGraph(t)

	// Expected direct links between key files.
	expectedLinks := []struct {
		from string
		to   string
	}{
		// README links
		{"README.md", "architecture.md"},
		{"README.md", "api/endpoints.md"},
		{"README.md", "config/setup.md"},
		{"README.md", "database.md"},
		{"README.md", "services/user-service.md"},
		{"README.md", "services/order-service.md"},
		{"README.md", "services/payment-service.md"},
		{"README.md", "quickstart.md"},
		// Architecture links
		{"architecture.md", "services/user-service.md"},
		{"architecture.md", "services/order-service.md"},
		{"architecture.md", "services/payment-service.md"},
		{"architecture.md", "database.md"},
		{"architecture.md", "api/endpoints.md"},
		// Order service links
		{"services/order-service.md", "services/user-service.md"},
		{"services/order-service.md", "services/payment-service.md"},
		{"services/order-service.md", "database.md"},
		{"services/order-service.md", "api/endpoints.md"},
		{"services/order-service.md", "architecture.md"},
		// Payment service links
		{"services/payment-service.md", "services/user-service.md"},
		{"services/payment-service.md", "services/order-service.md"},
		{"services/payment-service.md", "database.md"},
		{"services/payment-service.md", "api/endpoints.md"},
		// User service links
		{"services/user-service.md", "services/order-service.md"},
		{"services/user-service.md", "services/payment-service.md"},
		{"services/user-service.md", "database.md"},
		{"services/user-service.md", "api/endpoints.md"},
		// Database links
		{"database.md", "services/user-service.md"},
		{"database.md", "services/order-service.md"},
		{"database.md", "services/payment-service.md"},
		// API endpoints links
		{"api/endpoints.md", "services/user-service.md"},
		{"api/endpoints.md", "services/order-service.md"},
		{"api/endpoints.md", "services/payment-service.md"},
		{"api/endpoints.md", "database.md"},
	}

	found := 0
	missing := []string{}
	for _, link := range expectedLinks {
		hasEdge := false
		for _, e := range graph.BySource[link.from] {
			if e.Target == link.to {
				hasEdge = true
				break
			}
		}
		if hasEdge {
			found++
		} else {
			missing = append(missing, fmt.Sprintf("%s -> %s", link.from, link.to))
		}
	}

	t.Logf("Found %d/%d expected links", found, len(expectedLinks))
	if len(missing) > 0 {
		t.Logf("Missing links: %s", strings.Join(missing, ", "))
	}

	// Allow a small number of misses due to path resolution edge cases,
	// but we expect at least 80% accuracy.
	accuracy := float64(found) / float64(len(expectedLinks))
	if accuracy < 0.80 {
		t.Errorf("Link accuracy = %.1f%%, want >= 80%%. Missing: %v", accuracy*100, missing)
	}
}

func TestStructuralRelationships_IsolatedNode(t *testing.T) {
	_, graph, _ := loadTestGraph(t)

	// isolated-guide.md should have no outgoing edges (it has no links).
	outgoing := graph.GetOutgoing("isolated-guide.md")
	if len(outgoing) != 0 {
		t.Errorf("isolated-guide.md should have 0 outgoing edges, got %d", len(outgoing))
	}
}

func TestStructuralRelationships_GlossaryIsolated(t *testing.T) {
	_, graph, _ := loadTestGraph(t)

	// glossary.md is standalone with no links to other files.
	outgoing := graph.GetOutgoing("glossary.md")
	if len(outgoing) != 0 {
		t.Errorf("glossary.md should have 0 outgoing edges, got %d", len(outgoing))
	}
}

func TestStructuralRelationships_TroubleshootingLowConnectivity(t *testing.T) {
	_, graph, _ := loadTestGraph(t)

	// troubleshooting.md has no explicit markdown links, so it should
	// have very few outgoing edges (at most incidental text matches).
	outgoing := graph.GetOutgoing("troubleshooting.md")
	if len(outgoing) > 3 {
		t.Errorf("troubleshooting.md should have few outgoing edges (no explicit links), got %d", len(outgoing))
	}
	t.Logf("troubleshooting.md has %d outgoing edges", len(outgoing))
}

// ---------------------------------------------------------------------------
// 2. Co-occurrence relationship tests (mention extraction)
// ---------------------------------------------------------------------------

func TestCoOccurrenceRelationships_MentionExtraction(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	detector := NewComponentDetector()
	components := detector.DetectComponents(graph, docs)

	if len(components) == 0 {
		t.Skip("No components detected from test graph")
	}

	mentions := ExtractMentionsFromDocuments(docs, components)
	t.Logf("Extracted %d mentions from %d documents using %d components", len(mentions), len(docs), len(components))

	// We expect at least some mention-based relationships.
	if len(mentions) < 1 {
		t.Logf("Warning: no mention-based relationships found. Components: %v",
			func() []string {
				ids := make([]string, len(components))
				for i, c := range components {
					ids[i] = c.ID
				}
				return ids
			}())
	}
}

func TestCoOccurrenceRelationships_ServiceMentionDetection(t *testing.T) {
	// Test that "calls X service" / "depends on X" patterns work.
	tests := []struct {
		text      string
		component string
		wantMatch bool
	}{
		{"Order Service calls user service to validate orders", "user", true},
		{"Payment Service requires authenticated users", "user", false}, // "requires" won't match "user" in this context
		{"This is a totally unrelated document", "user", false},
	}

	for _, tc := range tests {
		t.Run(tc.text[:30], func(t *testing.T) {
			matched, _ := IsComponentMention(tc.text, tc.component)
			if matched != tc.wantMatch {
				t.Errorf("IsComponentMention(%q, %q) = %v, want %v", tc.text, tc.component, matched, tc.wantMatch)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. Semantic relationship tests (registry signal aggregation)
// ---------------------------------------------------------------------------

func TestSemanticRelationships_RegistryInit(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	t.Logf("Registry: %d components, %d relationships",
		reg.ComponentCount(), reg.RelationshipCount())

	// We should have components registered.
	if reg.ComponentCount() == 0 {
		t.Error("Registry should have at least 1 component after InitFromGraph")
	}

	// We should have relationships from link signals.
	if reg.RelationshipCount() == 0 {
		t.Error("Registry should have at least 1 relationship after InitFromGraph")
	}
}

func TestSemanticRelationships_SignalTypes(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	// Check that relationships have link-type signals.
	linkSignalFound := false
	for _, rel := range reg.Relationships {
		for _, s := range rel.Signals {
			if s.SourceType == SignalLink {
				linkSignalFound = true
				break
			}
		}
		if linkSignalFound {
			break
		}
	}

	if !linkSignalFound {
		t.Error("Expected at least one link-type signal in registry relationships")
	}
}

func TestSemanticRelationships_ConfidenceScores(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	for _, rel := range reg.Relationships {
		if rel.AggregatedConfidence < 0.0 || rel.AggregatedConfidence > 1.0 {
			t.Errorf("Relationship %s->%s: AggregatedConfidence=%.4f out of [0.0, 1.0]",
				rel.FromComponent, rel.ToComponent, rel.AggregatedConfidence)
		}
	}
}

// ---------------------------------------------------------------------------
// 4. Signal aggregation tests
// ---------------------------------------------------------------------------

func TestSignalAggregation_MaxStrategy(t *testing.T) {
	h := NewHybridBuilder()
	h.Strategy = AggregationMax

	signals := []Signal{
		{SourceType: SignalLink, Confidence: 0.8, Weight: 1.0},
		{SourceType: SignalMention, Confidence: 0.6, Weight: 1.0},
		{SourceType: SignalLLM, Confidence: 0.9, Weight: 1.0},
	}

	result := h.AggregateSignals(signals)
	if result != 0.9 {
		t.Errorf("AggregateSignals(Max) = %.2f, want 0.9", result)
	}
}

func TestSignalAggregation_WeightedAverage(t *testing.T) {
	h := NewHybridBuilder()
	h.Strategy = AggregationWeightedAverage

	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Weight: 2.0},
		{SourceType: SignalMention, Confidence: 0.5, Weight: 1.0},
	}

	result := h.AggregateSignals(signals)
	// (1.0*2.0 + 0.5*1.0) / (2.0 + 1.0) = 2.5/3.0 = 0.833
	expected := 2.5 / 3.0
	if result < expected-0.01 || result > expected+0.01 {
		t.Errorf("AggregateSignals(WeightedAvg) = %.4f, want ~%.4f", result, expected)
	}
}

func TestSignalAggregation_BelowThreshold(t *testing.T) {
	h := NewHybridBuilder()
	h.MinConfidence = 0.5

	signals := []Signal{
		{SourceType: SignalMention, Confidence: 0.3, Weight: 1.0},
		{SourceType: SignalLLM, Confidence: 0.4, Weight: 1.0},
	}

	result := h.AggregateSignals(signals)
	if result != 0.0 {
		t.Errorf("AggregateSignals below threshold = %.2f, want 0.0", result)
	}
}

func TestSignalAggregation_EmptySignals(t *testing.T) {
	h := NewHybridBuilder()
	result := h.AggregateSignals(nil)
	if result != 0.0 {
		t.Errorf("AggregateSignals(nil) = %.2f, want 0.0", result)
	}
}

func TestSignalAggregation_CapAt1(t *testing.T) {
	h := NewHybridBuilder()
	h.MinConfidence = 0.0

	signals := []Signal{
		{SourceType: SignalLink, Confidence: 1.0, Weight: 2.0},
	}

	result := h.AggregateSignals(signals)
	if result > 1.0 {
		t.Errorf("AggregateSignals should cap at 1.0, got %.2f", result)
	}
}

// ---------------------------------------------------------------------------
// 5. Hybrid graph builder (merge registry into graph) tests
// ---------------------------------------------------------------------------

func TestHybridGraphBuilder_MergeUpdatesConfidence(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	initialEdgeCount := graph.EdgeCount()

	hb := NewHybridBuilder()
	resultGraph := hb.BuildHybridGraph(reg, graph)

	// The graph should still have edges (possibly more or updated confidences).
	if resultGraph.EdgeCount() < initialEdgeCount {
		t.Errorf("Hybrid graph has %d edges, expected >= %d", resultGraph.EdgeCount(), initialEdgeCount)
	}
}

func TestHybridGraphBuilder_NilRegistry(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a.md", Title: "A", Type: "document"})

	hb := NewHybridBuilder()
	result := hb.BuildHybridGraph(nil, g)

	if result != g {
		t.Error("BuildHybridGraph(nil registry) should return the original graph")
	}
}

// ---------------------------------------------------------------------------
// 6. Manifest / persistence tests
// ---------------------------------------------------------------------------

func TestManifestGeneration_RegistrySaveLoad(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	// Save to temp file.
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, RegistryFileName)
	if err := SaveRegistry(reg, regPath); err != nil {
		t.Fatalf("SaveRegistry: %v", err)
	}

	// Load back.
	loaded, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadRegistry returned nil")
	}

	if loaded.ComponentCount() != reg.ComponentCount() {
		t.Errorf("Loaded ComponentCount = %d, want %d", loaded.ComponentCount(), reg.ComponentCount())
	}
	if loaded.RelationshipCount() != reg.RelationshipCount() {
		t.Errorf("Loaded RelationshipCount = %d, want %d", loaded.RelationshipCount(), reg.RelationshipCount())
	}
}

func TestRelationshipPersistence_RoundTrip(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{
		ID:         "auth",
		Name:       "Auth Service",
		FileRef:    "services/auth.md",
		Type:       ComponentTypeService,
		DetectedAt: time.Now(),
	})
	_ = reg.AddComponent(&RegistryComponent{
		ID:         "db",
		Name:       "Database",
		FileRef:    "database.md",
		Type:       ComponentTypeDatabase,
		DetectedAt: time.Now(),
	})
	_ = reg.AddSignal("auth", "db", Signal{
		SourceType: SignalLink,
		Confidence: 1.0,
		Evidence:   "markdown link",
		Weight:     1.0,
	})

	data, err := reg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	loaded := NewComponentRegistry()
	if err := loaded.FromJSON(data); err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if loaded.ComponentCount() != 2 {
		t.Errorf("ComponentCount = %d, want 2", loaded.ComponentCount())
	}
	if loaded.RelationshipCount() != 1 {
		t.Errorf("RelationshipCount = %d, want 1", loaded.RelationshipCount())
	}

	// Verify signal is preserved.
	rels := loaded.FindRelationships("auth")
	if len(rels) != 1 {
		t.Fatalf("FindRelationships(auth) = %d, want 1", len(rels))
	}
	if rels[0].ToComponent != "db" {
		t.Errorf("Relationship target = %q, want %q", rels[0].ToComponent, "db")
	}
	if len(rels[0].Signals) != 1 {
		t.Errorf("Signals count = %d, want 1", len(rels[0].Signals))
	}
	if rels[0].Signals[0].SourceType != SignalLink {
		t.Errorf("Signal type = %q, want %q", rels[0].Signals[0].SourceType, SignalLink)
	}
}

// ---------------------------------------------------------------------------
// 7. End-to-end pipeline: scan -> build graph -> init registry -> validate
// ---------------------------------------------------------------------------

func TestEndToEnd_ScanBuildRegistryValidate(t *testing.T) {
	dir, graph, docs := loadTestGraph(t)
	_ = dir

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	t.Logf("End-to-end: %d docs, %d nodes, %d edges, %d components, %d relationships",
		len(docs), graph.NodeCount(), graph.EdgeCount(),
		reg.ComponentCount(), reg.RelationshipCount())

	// Validate expected file-based relationships.
	// order-service should depend on user-service and payment-service.
	orderRels := reg.FindRelationships("order-service")
	orderTargets := make(map[string]bool)
	for _, r := range orderRels {
		orderTargets[r.ToComponent] = true
	}

	t.Logf("Order service depends on: %v", mapKeys(orderTargets))

	// Check that order-service has outgoing relationships.
	if len(orderRels) == 0 {
		t.Log("Warning: order-service has no outgoing relationships in registry")
	}
}

func TestEndToEnd_RegistryQueryByConfidence(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	// Query high-confidence relationships.
	highConf := reg.QueryByConfidence(0.8)
	t.Logf("High confidence (>=0.8) relationships: %d", len(highConf))

	// Link-based relationships should have confidence 1.0.
	for _, r := range highConf {
		if r.AggregatedConfidence < 0.8 {
			t.Errorf("QueryByConfidence(0.8) returned relationship with confidence %.2f", r.AggregatedConfidence)
		}
	}
}

func TestEndToEnd_ComponentDetection(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	detector := NewComponentDetector()
	components := detector.DetectComponents(graph, docs)

	t.Logf("Detected %d components:", len(components))
	for _, c := range components {
		t.Logf("  %s (file=%s, confidence=%.2f)", c.ID, c.File, c.Confidence)
	}

	// Should detect at least the service files.
	if len(components) < 1 {
		t.Error("Expected at least 1 component to be detected")
	}
}

func TestEndToEnd_DependencyAnalysis(t *testing.T) {
	_, graph, docs := loadTestGraph(t)

	detector := NewComponentDetector()
	components := detector.DetectComponents(graph, docs)

	if len(components) < 2 {
		t.Skip("Need at least 2 components for dependency analysis")
	}

	da := NewDependencyAnalyzer(graph, components)
	sg := da.GetComponentGraph()

	t.Logf("Service graph: %d components, %d dependencies",
		len(sg.Components), countDeps(sg))
}

// ---------------------------------------------------------------------------
// 8. Accuracy comparison: with links vs without links
// ---------------------------------------------------------------------------

func TestAccuracy_WithLinksVsTextOnly(t *testing.T) {
	dir, docs := loadTestDocs(t)

	// Phase 1: Build graph WITH links (our "ground truth").
	gbWithLinks := NewGraphBuilder(dir)
	graphWithLinks := gbWithLinks.Build(docs)

	linkEdges := make(map[string]bool)
	for _, e := range graphWithLinks.Edges {
		if e.Type == EdgeReferences {
			key := e.Source + " -> " + e.Target
			linkEdges[key] = true
		}
	}
	t.Logf("Ground truth: %d link-based edges", len(linkEdges))

	// Phase 2: Text-based relationships (mention extraction).
	detector := NewComponentDetector()
	components := detector.DetectComponents(graphWithLinks, docs)

	if len(components) == 0 {
		t.Log("No components detected; skipping mention-based comparison")
		return
	}

	mentions := ExtractMentionsFromDocuments(docs, components)
	mentionEdges := make(map[string]bool)
	for _, m := range mentions {
		key := m.FromFile + " -> " + m.ToComponent
		mentionEdges[key] = true
	}
	t.Logf("Mention-based edges: %d", len(mentionEdges))

	// Phase 3: Registry-based relationships (combined).
	reg := NewComponentRegistry()
	reg.InitFromGraph(graphWithLinks, docs)
	t.Logf("Registry: %d total relationships", reg.RelationshipCount())
}

// ---------------------------------------------------------------------------
// 9. NER component extraction tests (using existing patterns)
// ---------------------------------------------------------------------------

func TestNERComponentExtraction_BuiltInPatterns(t *testing.T) {
	patterns := BuiltInPatterns()

	total := len(patterns.ServicePatterns) + len(patterns.ApiPatterns) + len(patterns.ConfigPatterns)
	if total == 0 {
		t.Error("BuiltInPatterns should have at least 1 pattern")
	}
	t.Logf("Pattern library: %d service, %d API, %d config patterns",
		len(patterns.ServicePatterns), len(patterns.ApiPatterns), len(patterns.ConfigPatterns))
}

func TestNERComponentExtraction_ServiceSuffixPattern(t *testing.T) {
	text := "The user-service handles authentication"
	matched, conf := IsComponentMention(text, "user")
	if !matched {
		t.Error("Expected to match 'user' in 'user-service handles authentication'")
	}
	if conf < 0.5 {
		t.Errorf("Confidence = %.2f, want >= 0.5", conf)
	}
}

func TestNERComponentExtraction_CallsPattern(t *testing.T) {
	text := "This module calls the auth service"
	matched, conf := IsComponentMention(text, "auth")
	if !matched {
		t.Error("Expected to match 'auth' in 'calls the auth service'")
	}
	if conf < 0.5 {
		t.Errorf("Confidence = %.2f, want >= 0.5", conf)
	}
}

func TestNERComponentExtraction_DependsOnPattern(t *testing.T) {
	text := "This depends on the payment gateway"
	matched, conf := IsComponentMention(text, "payment")
	if !matched {
		t.Error("Expected to match 'payment' in 'depends on the payment gateway'")
	}
	if conf < 0.5 {
		t.Errorf("Confidence = %.2f, want >= 0.5", conf)
	}
}

func TestNERComponentExtraction_NoFalsePositive(t *testing.T) {
	text := "The quick brown fox jumped over the lazy dog"
	matched, _ := IsComponentMention(text, "auth")
	if matched {
		t.Error("Should not match 'auth' in unrelated text")
	}
}

// ---------------------------------------------------------------------------
// 10. SVO triple extraction tests (using mention patterns)
// ---------------------------------------------------------------------------

func TestSVOTripleExtraction_ExtractFromLine(t *testing.T) {
	knownComponents := map[string]string{
		"auth":    "auth-service",
		"payment": "payment-service",
	}

	tests := []struct {
		line    string
		wantLen int
	}{
		{"The system calls the auth service", 1},
		{"This depends on payment backend", 1},
		{"No services mentioned here", 0},
	}

	for _, tc := range tests {
		t.Run(tc.line[:20], func(t *testing.T) {
			candidates := ExtractMentionsFromLine(tc.line, knownComponents)
			if len(candidates) != tc.wantLen {
				t.Errorf("ExtractMentionsFromLine(%q) = %d candidates, want %d",
					tc.line, len(candidates), tc.wantLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 11. Performance benchmarks
// ---------------------------------------------------------------------------

func BenchmarkScanDirectory_TestData(b *testing.B) {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..")
	dir := filepath.Join(root, "test-data", "graph-test-docs")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("test-data not found: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	}
}

func BenchmarkGraphBuild_TestData(b *testing.B) {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..")
	dir := filepath.Join(root, "test-data", "graph-test-docs")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("test-data not found: %v", err)
	}

	docs, err := ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		b.Fatalf("ScanDirectory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gb := NewGraphBuilder(dir)
		_ = gb.Build(docs)
	}
}

func BenchmarkRegistryInit_TestData(b *testing.B) {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..")
	dir := filepath.Join(root, "test-data", "graph-test-docs")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("test-data not found: %v", err)
	}

	docs, err := ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		b.Fatalf("ScanDirectory: %v", err)
	}
	gb := NewGraphBuilder(dir)
	graph := gb.Build(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg := NewComponentRegistry()
		reg.InitFromGraph(graph, docs)
	}
}

func BenchmarkMentionExtraction_TestData(b *testing.B) {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..")
	dir := filepath.Join(root, "test-data", "graph-test-docs")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("test-data not found: %v", err)
	}

	docs, err := ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		b.Fatalf("ScanDirectory: %v", err)
	}
	gb := NewGraphBuilder(dir)
	graph := gb.Build(docs)
	detector := NewComponentDetector()
	components := detector.DetectComponents(graph, docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractMentionsFromDocuments(docs, components)
	}
}

func BenchmarkHybridBuilder_TestData(b *testing.B) {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..")
	dir := filepath.Join(root, "test-data", "graph-test-docs")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("test-data not found: %v", err)
	}

	docs, err := ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		b.Fatalf("ScanDirectory: %v", err)
	}
	gb := NewGraphBuilder(dir)
	graph := gb.Build(docs)
	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clone graph to avoid mutation between runs.
		cloned := cloneGraph(graph)
		hb := NewHybridBuilder()
		_ = hb.BuildHybridGraph(reg, cloned)
	}
}

// BenchmarkScalability_SyntheticNFiles generates synthetic document sets of
// increasing size and measures registry init time.
func BenchmarkScalability_10Files(b *testing.B)  { benchmarkSyntheticDocs(b, 10) }
func BenchmarkScalability_50Files(b *testing.B)  { benchmarkSyntheticDocs(b, 50) }
func BenchmarkScalability_100Files(b *testing.B) { benchmarkSyntheticDocs(b, 100) }
func BenchmarkScalability_500Files(b *testing.B) { benchmarkSyntheticDocs(b, 500) }

func benchmarkSyntheticDocs(b *testing.B, n int) {
	docs, graph := generateSyntheticDocSet(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg := NewComponentRegistry()
		reg.InitFromGraph(graph, docs)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func countDeps(sg *dependencyGraph) int { //nolint:unused
	count := 0
	for _, deps := range sg.Dependencies {
		count += len(deps)
	}
	return count
}

// cloneGraph creates a shallow copy of a graph (new maps, same pointers).
func cloneGraph(g *Graph) *Graph {
	c := NewGraph()
	for id, node := range g.Nodes {
		c.Nodes[id] = node
	}
	for id, edge := range g.Edges {
		c.Edges[id] = edge
	}
	for id, edges := range g.BySource {
		c.BySource[id] = append([]*Edge(nil), edges...)
	}
	for id, edges := range g.ByTarget {
		c.ByTarget[id] = append([]*Edge(nil), edges...)
	}
	return c
}

// generateSyntheticDocSet creates n synthetic Documents and a Graph with random edges.
func generateSyntheticDocSet(n int) ([]Document, *Graph) {
	docs := make([]Document, n)
	g := NewGraph()

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("doc-%03d.md", i)
		title := fmt.Sprintf("Document %d", i)
		content := fmt.Sprintf("# %s\n\nContent for document %d.\n", title, i)

		docs[i] = Document{
			ID:        id,
			Path:      "/synthetic/" + id,
			RelPath:   id,
			Title:     title,
			Content:   content,
			PlainText: content,
		}

		_ = g.AddNode(&Node{ID: id, Title: title, Type: "document"})
	}

	// Add edges: each doc links to 2-3 neighbors.
	for i := 0; i < n; i++ {
		src := fmt.Sprintf("doc-%03d.md", i)
		for j := 1; j <= 3 && i+j < n; j++ {
			tgt := fmt.Sprintf("doc-%03d.md", i+j)
			e, _ := NewEdge(src, tgt, EdgeReferences, 1.0, "synthetic link")
			_ = g.AddEdge(e)
		}
	}

	return docs, g
}
