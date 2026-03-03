package knowledge

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// --- NER Tests ---------------------------------------------------------------

func TestNormalizeNERName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User Service", "user-service"},
		{"API Gateway", "api-gateway"},
		{"user-service.md", "user-service"},
		{"Order Svc", "order-svc"},
		{"payment_service", "payment-service"},
		{"  Spaces  Everywhere  ", "spaces-everywhere"},
		{"", ""},
		{"SingleWord", "singleword"},
		{"Database Design", "database-design"},
	}
	for _, tc := range tests {
		got := NormalizeNERName(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeNERName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestExtractComponentNames_FilenameSignal(t *testing.T) {
	docs := []Document{
		{ID: "services/user-service.md", Title: "User Service", Content: "# User Service\nHandles users."},
		{ID: "services/order-service.md", Title: "Order Service", Content: "# Order Service\nHandles orders."},
		{ID: "api/endpoints.md", Title: "REST API Endpoints", Content: "# REST API Endpoints\nAPI docs."},
		{ID: "README.md", Title: "Graph Test Documentation", Content: "# Graph Test Documentation\nMain entry."},
	}

	registry := ExtractComponentNames(docs)

	// user-service and order-service should be detected from filenames.
	if _, ok := registry["user-service"]; !ok {
		t.Error("expected user-service in registry from filename signal")
	}
	if _, ok := registry["order-service"]; !ok {
		t.Error("expected order-service in registry from filename signal")
	}
	// endpoints contains "api" keyword
	found := false
	for id := range registry {
		if strings.Contains(id, "endpoint") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected endpoints component in registry from filename containing 'api'")
	}
}

func TestExtractComponentNames_HeadingSignal(t *testing.T) {
	docs := []Document{
		{
			ID:      "auth.md",
			Title:   "Auth Service",
			Content: "# Auth Service\nHandles authentication.\n## Config\nSome config.",
		},
	}

	registry := ExtractComponentNames(docs)

	if _, ok := registry["auth-service"]; !ok {
		t.Errorf("expected auth-service in registry from H1 heading; got keys: %v", registryKeys(registry))
	}
}

func TestExtractComponentNames_ServiceListSignal(t *testing.T) {
	docs := []Document{
		{
			ID:    "architecture.md",
			Title: "Architecture Overview",
			Content: `# Architecture Overview
## Backend Services
- User Service - Handles authentication
- Order Service - Manages orders
- Payment Service - Processes payments
`,
		},
	}

	registry := ExtractComponentNames(docs)

	for _, expected := range []string{"user-service", "order-service", "payment-service"} {
		if _, ok := registry[expected]; !ok {
			t.Errorf("expected %q in registry from service list; got keys: %v", expected, registryKeys(registry))
		}
	}
}

func TestBuildComponentRegistry_ResolvesFiles(t *testing.T) {
	docs := []Document{
		{ID: "services/user-service.md", Title: "User Service", Content: "# User Service\nHandles users."},
		{
			ID:    "architecture.md",
			Title: "Architecture Overview",
			Content: `# Architecture Overview
- User Service - authentication
`,
		},
	}

	registry := BuildComponentRegistry(docs)

	comp, ok := registry["user-service"]
	if !ok {
		t.Fatal("expected user-service in registry")
	}
	if comp.File != "services/user-service.md" {
		t.Errorf("expected file to be services/user-service.md, got %q", comp.File)
	}
}

func TestFuzzyComponentMatch_ExactMatch(t *testing.T) {
	registry := map[string]*NERComponent{
		"user-service":    {ID: "user-service", Name: "User Service"},
		"order-service":   {ID: "order-service", Name: "Order Service"},
		"payment-service": {ID: "payment-service", Name: "Payment Service"},
	}

	comp := FuzzyComponentMatch("user-service", registry)
	if comp == nil || comp.ID != "user-service" {
		t.Errorf("expected exact match for user-service, got %v", comp)
	}
}

func TestFuzzyComponentMatch_SuffixVariation(t *testing.T) {
	registry := map[string]*NERComponent{
		"user-service": {ID: "user-service", Name: "User Service"},
	}

	// "user" should match "user-service" via suffix stripping.
	comp := FuzzyComponentMatch("user", registry)
	if comp == nil {
		t.Error("expected fuzzy match for 'user' -> 'user-service'")
	} else if comp.ID != "user-service" {
		t.Errorf("expected user-service, got %q", comp.ID)
	}
}

func TestFuzzyComponentMatch_AliasMatch(t *testing.T) {
	registry := map[string]*NERComponent{
		"user-service": {
			ID:      "user-service",
			Name:    "User Service",
			Aliases: []string{"user service", "user-service", "user"},
		},
	}

	comp := FuzzyComponentMatch("User Service", registry)
	if comp == nil || comp.ID != "user-service" {
		t.Errorf("expected alias match for 'User Service', got %v", comp)
	}
}

func TestFuzzyComponentMatch_NoMatch(t *testing.T) {
	registry := map[string]*NERComponent{
		"user-service": {ID: "user-service", Name: "User Service"},
	}

	comp := FuzzyComponentMatch("nonexistent-thing", registry)
	if comp != nil {
		t.Errorf("expected no match for 'nonexistent-thing', got %v", comp)
	}
}

func TestFuzzyComponentMatch_Empty(t *testing.T) {
	comp := FuzzyComponentMatch("", nil)
	if comp != nil {
		t.Error("expected nil for empty input")
	}
}

func TestFindComponentsInLine(t *testing.T) {
	registry := map[string]*NERComponent{
		"user-service":    {ID: "user-service", Name: "User Service", Aliases: []string{"user service"}},
		"payment-service": {ID: "payment-service", Name: "Payment Service", Aliases: []string{"payment service"}},
		"order-service":   {ID: "order-service", Name: "Order Service", Aliases: []string{"order service"}},
	}

	comps := FindComponentsInLine("Order Service calls User Service to validate orders", registry)
	if len(comps) < 2 {
		t.Errorf("expected at least 2 components in line, got %d", len(comps))
	}

	ids := make(map[string]bool)
	for _, c := range comps {
		ids[c.ID] = true
	}
	if !ids["order-service"] {
		t.Error("expected order-service in line components")
	}
	if !ids["user-service"] {
		t.Error("expected user-service in line components")
	}
}

func TestResolveComponentToFile(t *testing.T) {
	comp := &NERComponent{ID: "user-service", Name: "User Service", File: ""}
	docs := []Document{
		{ID: "services/user-service.md", Title: "User Service"},
		{ID: "services/order-service.md", Title: "Order Service"},
	}

	file := ResolveComponentToFile(comp, docs)
	if file != "services/user-service.md" {
		t.Errorf("expected services/user-service.md, got %q", file)
	}
}

func TestResolveComponentToFile_WithFileSet(t *testing.T) {
	comp := &NERComponent{ID: "user-service", File: "services/user-service.md"}
	file := ResolveComponentToFile(comp, nil)
	if file != "services/user-service.md" {
		t.Errorf("expected services/user-service.md, got %q", file)
	}
}

// --- SVO Tests ---------------------------------------------------------------

func TestExtractSVOTriples_DependsOn(t *testing.T) {
	triples := ExtractSVOTriples("Order Service depends on User Service.")
	if len(triples) == 0 {
		t.Fatal("expected at least one triple for 'depends on' pattern")
	}

	found := false
	for _, tr := range triples {
		if tr.Verb == "depends on" &&
			strings.Contains(strings.ToLower(tr.Subject), "order") &&
			strings.Contains(strings.ToLower(tr.Object), "user") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected triple with 'Order depends on User', got %v", triples)
	}
}

func TestExtractSVOTriples_Calls(t *testing.T) {
	triples := ExtractSVOTriples("Order Service calls user service to validate orders")
	if len(triples) == 0 {
		t.Fatal("expected at least one triple for 'calls' pattern")
	}

	found := false
	for _, tr := range triples {
		if tr.Verb == "calls" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected triple with verb 'calls', got %v", triples)
	}
}

func TestExtractSVOTriples_Requires(t *testing.T) {
	triples := ExtractSVOTriples("Payment Service requires authenticated users.")
	if len(triples) == 0 {
		t.Fatal("expected at least one triple for 'requires' pattern")
	}

	found := false
	for _, tr := range triples {
		if tr.Verb == "requires" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected triple with verb 'requires', got %v", triples)
	}
}

func TestExtractSVOTriples_IntegratesWith(t *testing.T) {
	triples := ExtractSVOTriples("Auth integrates with the Payment Service.")
	if len(triples) == 0 {
		t.Fatal("expected at least one triple for 'integrates with' pattern")
	}
	if triples[0].Verb != "integrates with" {
		t.Errorf("expected verb 'integrates with', got %q", triples[0].Verb)
	}
}

func TestExtractSVOTriples_EmptyInput(t *testing.T) {
	triples := ExtractSVOTriples("")
	if triples != nil {
		t.Error("expected nil for empty input")
	}
}

func TestExtractSVOTriples_NoMatch(t *testing.T) {
	triples := ExtractSVOTriples("The quick brown fox jumps over the lazy dog.")
	if len(triples) > 0 {
		t.Errorf("expected no triples for unrelated sentence, got %v", triples)
	}
}

func TestExtractSVOTriples_SelfReference(t *testing.T) {
	triples := ExtractSVOTriples("Service calls Service.")
	for _, tr := range triples {
		if strings.EqualFold(tr.Subject, tr.Object) {
			t.Errorf("unexpected self-referential triple: %v", tr)
		}
	}
}

func TestClassifyVerb(t *testing.T) {
	tests := []struct {
		verb       string
		wantType   EdgeType
		wantMinConf float64
	}{
		{"depends on", EdgeDependsOn, 0.80},
		{"requires", EdgeDependsOn, 0.80},
		{"calls", EdgeCalls, 0.75},
		{"uses", EdgeMentions, 0.70},
		{"integrates with", EdgeDependsOn, 0.65},
		{"connects to", EdgeDependsOn, 0.65},
		{"provides", EdgeImplements, 0.70},
		{"communicates with", EdgeDependsOn, 0.65},
		{"unknown-verb", EdgeMentions, 0.60},
	}

	for _, tc := range tests {
		gotType, gotConf := ClassifyVerb(tc.verb)
		if gotType != tc.wantType {
			t.Errorf("ClassifyVerb(%q) type = %v, want %v", tc.verb, gotType, tc.wantType)
		}
		if gotConf < tc.wantMinConf {
			t.Errorf("ClassifyVerb(%q) confidence = %.2f, want >= %.2f", tc.verb, gotConf, tc.wantMinConf)
		}
	}
}

// --- Integration Tests (NERRelationships) ------------------------------------

func TestNERRelationships_BasicExtraction(t *testing.T) {
	docs := []Document{
		{
			ID:      "services/user-service.md",
			Title:   "User Service",
			Content: "# User Service\nHandles user management.\n\n## Integration\n\nOrder Service calls user service to validate orders.\nPayment Service requires authenticated users.",
		},
		{
			ID:      "services/order-service.md",
			Title:   "Order Service",
			Content: "# Order Service\nManages orders.\n\n## Dependencies\n\nOrder Service depends on User Service.\nOrder Service calls Payment Service.",
		},
		{
			ID:      "services/payment-service.md",
			Title:   "Payment Service",
			Content: "# Payment Service\nHandles payments.\n\n## Integration Points\n\nPayment Service integrates with User Service.",
		},
	}

	edges := NERRelationships(docs)

	if len(edges) == 0 {
		t.Fatal("expected at least one edge from NERRelationships")
	}

	// Check that edges are directional.
	for _, edge := range edges {
		if edge.Source == edge.Target {
			t.Errorf("unexpected self-loop: %s -> %s", edge.Source, edge.Target)
		}
		if edge.Confidence < 0.60 || edge.Confidence > 1.0 {
			t.Errorf("edge confidence %.2f outside expected range [0.60, 1.0]: %s", edge.Confidence, edge)
		}
	}

	t.Logf("NERRelationships produced %d edges", len(edges))
	for _, e := range edges {
		t.Logf("  %s", e)
	}
}

func TestNERRelationships_Directionality(t *testing.T) {
	docs := []Document{
		{
			ID:      "services/order-service.md",
			Title:   "Order Service",
			Content: "# Order Service\n\nOrder Service depends on User Service.\n",
		},
		{
			ID:      "services/user-service.md",
			Title:   "User Service",
			Content: "# User Service\nHandles users.",
		},
	}

	edges := NERRelationships(docs)

	// Expect: order-service -> user-service (not the reverse).
	foundCorrect := false
	for _, edge := range edges {
		if edge.Source == "services/order-service.md" && edge.Target == "services/user-service.md" {
			foundCorrect = true
		}
		if edge.Source == "services/user-service.md" && edge.Target == "services/order-service.md" {
			t.Error("found reverse edge: user-service -> order-service (wrong directionality)")
		}
	}

	if !foundCorrect {
		t.Error("expected directional edge order-service -> user-service")
		for _, e := range edges {
			t.Logf("  found: %s", e)
		}
	}
}

func TestNERRelationships_EmptyInput(t *testing.T) {
	edges := NERRelationships(nil)
	if edges != nil {
		t.Error("expected nil for nil input")
	}

	edges = NERRelationships([]Document{})
	if edges != nil {
		t.Error("expected nil for empty input")
	}
}

func TestNERRelationships_NoServiceDocs(t *testing.T) {
	docs := []Document{
		{
			ID:      "glossary.md",
			Title:   "Glossary",
			Content: "# Glossary\n\nTerms and definitions.",
		},
	}

	edges := NERRelationships(docs)
	// Should produce zero or very few edges from a standalone glossary.
	if len(edges) > 0 {
		t.Logf("glossary produced %d edges (expected 0)", len(edges))
	}
}

func TestNERRelationshipsToRegistry(t *testing.T) {
	docs := []Document{
		{
			ID:      "services/user-service.md",
			Title:   "User Service",
			Content: "# User Service\nHandles users.",
		},
		{
			ID:      "services/order-service.md",
			Title:   "Order Service",
			Content: "# Order Service\n\nOrder Service depends on User Service.",
		},
	}

	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "user-service", Name: "User Service", FileRef: "services/user-service.md"})
	_ = reg.AddComponent(&RegistryComponent{ID: "order-service", Name: "Order Service", FileRef: "services/order-service.md"})

	NERRelationshipsToRegistry(docs, reg)

	// Should have at least one relationship.
	if reg.RelationshipCount() == 0 {
		t.Error("expected at least one relationship in registry after NER+SVO extraction")
	}
}

// --- Test with actual test-data files ----------------------------------------

func TestNERRelationships_WithTestData(t *testing.T) {
	testDir := filepath.Join(".", "..", "..", "test-data", "graph-test-docs")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("test-data directory not found")
	}

	docs, err := ScanDirectory(testDir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}
	if len(docs) == 0 {
		t.Skip("no documents found in test-data")
	}

	// First test the component registry extraction.
	registry := BuildComponentRegistry(docs)
	t.Logf("Component registry has %d entries:", len(registry))
	for id, comp := range registry {
		t.Logf("  %s (type=%s, file=%s)", id, comp.Type, comp.File)
	}

	// Now test the full NER relationship extraction.
	edges := NERRelationships(docs)
	t.Logf("NERRelationships produced %d edges from test-data:", len(edges))

	// Sort for stable output.
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})

	for _, e := range edges {
		t.Logf("  %s", e)
	}

	// Verify: all edges have confidence >= 0.60.
	for _, e := range edges {
		if e.Confidence < 0.60 {
			t.Errorf("edge confidence %.2f < 0.60: %s", e.Confidence, e)
		}
	}

	// Verify: no self-loops.
	for _, e := range edges {
		if e.Source == e.Target {
			t.Errorf("self-loop detected: %s", e)
		}
	}

	// Verify: edges are directional (have both source and target).
	for _, e := range edges {
		if e.Source == "" || e.Target == "" {
			t.Errorf("edge with empty source or target: %s", e)
		}
	}
}

// --- Sentence extraction tests -----------------------------------------------

func TestExtractSentences(t *testing.T) {
	content := `# Title

This is a paragraph. It has two sentences.

## Section

- List item one.
- List item two.

` + "```\ncode block\n```" + `

Another paragraph after code.
`

	sentences := extractSentences(content)
	if len(sentences) == 0 {
		t.Fatal("expected at least one sentence")
	}

	// Should not contain the code block.
	for _, s := range sentences {
		if strings.Contains(s, "code block") {
			t.Errorf("sentence should not contain code block content: %q", s)
		}
	}

	t.Logf("Extracted %d sentences:", len(sentences))
	for _, s := range sentences {
		t.Logf("  %q", s)
	}
}

func TestExtractSentences_Empty(t *testing.T) {
	sentences := extractSentences("")
	if sentences != nil {
		t.Error("expected nil for empty content")
	}
}

func TestCleanSentence(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"**bold text** and `code`", "bold text and code"},
		{"[link text](http://example.com)", "link text"},
		{"# Heading text", "Heading text"},
		{"- list item", "list item"},
	}

	for _, tc := range tests {
		got := cleanSentence(tc.input)
		if got != tc.want {
			t.Errorf("cleanSentence(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- Helper ------------------------------------------------------------------

func registryKeys(r map[string]*NERComponent) []string {
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
