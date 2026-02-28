package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ServiceDetector — IsService heuristic tests
// ---------------------------------------------------------------------------

func TestIsService_FilenameHeuristic(t *testing.T) {
	sd := NewServiceDetector()

	tests := []struct {
		nodeID     string
		title      string
		wantDetect bool
		wantConf   float64
	}{
		{"services/auth-service.md", "Auth Service", true, ConfidenceServiceFilename},
		{"user-service.md", "User Service", true, ConfidenceServiceFilename},
		{"payment-service.md", "Payment", true, ConfidenceServiceFilename},
		{"services/gateway-service.md", "Gateway", true, ConfidenceServiceFilename},
	}

	for _, tc := range tests {
		t.Run(tc.nodeID, func(t *testing.T) {
			node := &Node{ID: tc.nodeID, Title: tc.title, Type: "document"}
			svc, conf := sd.IsService(node)
			if tc.wantDetect && conf <= 0 {
				t.Errorf("IsService(%q): expected detection, got confidence=%.2f", tc.nodeID, conf)
			}
			if tc.wantDetect && conf != tc.wantConf {
				t.Errorf("IsService(%q): confidence=%.2f, want %.2f", tc.nodeID, conf, tc.wantConf)
			}
			if tc.wantDetect && svc.File != tc.nodeID {
				t.Errorf("IsService(%q): svc.File=%q, want %q", tc.nodeID, svc.File, tc.nodeID)
			}
		})
	}
}

func TestIsService_HeadingHeuristic(t *testing.T) {
	sd := NewServiceDetector()

	tests := []struct {
		nodeID   string
		title    string
		wantConf float64
	}{
		{"docs/auth.md", "Auth Service", ConfidenceServiceHeading},
		{"gateway.md", "API Gateway Service", ConfidenceServiceHeading},
		{"docs/users.md", "User Service", ConfidenceServiceHeading},
	}

	for _, tc := range tests {
		t.Run(tc.title, func(t *testing.T) {
			node := &Node{ID: tc.nodeID, Title: tc.title, Type: "document"}
			svc, conf := sd.IsService(node)
			if conf == 0 {
				t.Errorf("IsService: heading %q should be detected as service", tc.title)
			}
			if conf != tc.wantConf {
				t.Errorf("IsService: heading heuristic confidence=%.2f, want %.2f", conf, tc.wantConf)
			}
			if svc.Name != tc.title {
				t.Errorf("IsService: svc.Name=%q, want %q", svc.Name, tc.title)
			}
		})
	}
}

func TestIsService_NoFalsePositives(t *testing.T) {
	sd := NewServiceDetector()

	nonServices := []struct {
		nodeID string
		title  string
	}{
		{"docs/overview.md", "Overview"},
		{"README.md", "Getting Started"},
		{"docs/architecture.md", "Architecture"},
		{"guide/setup.md", "Setup Guide"},
		{"changelog.md", "Changelog"},
	}

	for _, tc := range nonServices {
		t.Run(tc.nodeID, func(t *testing.T) {
			node := &Node{ID: tc.nodeID, Title: tc.title, Type: "document"}
			_, conf := sd.IsService(node)
			if conf >= ConfidenceHighInDegree {
				t.Errorf("IsService(%q, %q): false positive, confidence=%.2f", tc.nodeID, tc.title, conf)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ServiceDetector — DetectServices tests
// ---------------------------------------------------------------------------

func makeServiceGraph(t *testing.T, nodes []struct{ id, title string }) *Graph {
	t.Helper()
	g := NewGraph()
	for _, n := range nodes {
		_ = g.AddNode(&Node{ID: n.id, Title: n.title, Type: "document"})
	}
	return g
}

func TestDetectServices_FilenamePattern(t *testing.T) {
	sd := NewServiceDetector()
	g := makeServiceGraph(t, []struct{ id, title string }{
		{"auth-service.md", "Auth Service"},
		{"user-service.md", "User Service"},
		{"overview.md", "Overview"},
	})

	services := sd.DetectServices(g, nil)
	if len(services) < 2 {
		t.Errorf("DetectServices: expected >=2 services, got %d", len(services))
	}

	// Verify services have expected IDs.
	byID := make(map[string]Service)
	for _, s := range services {
		byID[s.ID] = s
	}
	if _, ok := byID["auth-service"]; !ok {
		t.Error("expected auth-service to be detected")
	}
	if _, ok := byID["user-service"]; !ok {
		t.Error("expected user-service to be detected")
	}
}

func TestDetectServices_HighInDegree(t *testing.T) {
	sd := NewServiceDetector()
	g := NewGraph()

	// db-adapter.md is referenced by 4 other docs — high in-degree.
	for _, id := range []string{"a.md", "b.md", "c.md", "d.md", "db-adapter.md"} {
		_ = g.AddNode(&Node{ID: id, Title: id, Type: "document"})
	}
	for _, src := range []string{"a.md", "b.md", "c.md", "d.md"} {
		e, _ := NewEdge(src, "db-adapter.md", EdgeReferences, 1.0, "")
		_ = g.AddEdge(e)
	}

	services := sd.DetectServices(g, nil)
	byFile := make(map[string]Service)
	for _, s := range services {
		byFile[s.File] = s
	}
	if _, ok := byFile["db-adapter.md"]; !ok {
		t.Error("high in-degree node db-adapter.md should be detected as service")
	}
}

func TestDetectServices_RankedByConfidence(t *testing.T) {
	sd := NewServiceDetector()
	g := makeServiceGraph(t, []struct{ id, title string }{
		{"auth-service.md", "Auth Service"},   // filename heuristic (0.9)
		{"docs/gateway.md", "API Gateway Service"}, // heading heuristic (0.7)
	})

	services := sd.DetectServices(g, nil)
	if len(services) < 2 {
		t.Fatalf("expected >=2 services, got %d", len(services))
	}

	// First should have higher confidence.
	if services[0].Confidence < services[len(services)-1].Confidence {
		t.Errorf("services not ranked by confidence: first=%.2f, last=%.2f",
			services[0].Confidence, services[len(services)-1].Confidence)
	}
}

// ---------------------------------------------------------------------------
// ServiceDetector — DetectEndpoints tests
// ---------------------------------------------------------------------------

func TestDetectEndpoints_BasicPatterns(t *testing.T) {
	sd := NewServiceDetector()

	tests := []struct {
		name      string
		content   string
		wantMethod string
		wantPath  string
	}{
		{
			"plain method path",
			"# Auth\nPOST /users/login\n",
			"POST", "/users/login",
		},
		{
			"heading pattern",
			"# Auth\n## POST /users endpoint\n",
			"POST", "/users",
		},
		{
			"inline code",
			"Call `GET /health` to check status.\n",
			"GET", "/health",
		},
		{
			"DELETE endpoint",
			"DELETE /users/{id} removes a user.\n",
			"DELETE", "/users/{id}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := &Document{
				ID:      "service.md",
				Content: tc.content,
			}
			endpoints := sd.DetectEndpoints(doc)
			found := false
			for _, ep := range endpoints {
				if ep.Method == tc.wantMethod && strings.HasPrefix(ep.Path, tc.wantPath) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("DetectEndpoints: did not find %s %s in %v", tc.wantMethod, tc.wantPath, endpoints)
			}
		})
	}
}

func TestDetectEndpoints_Deduplication(t *testing.T) {
	sd := NewServiceDetector()
	// Same endpoint mentioned twice.
	doc := &Document{
		ID:      "svc.md",
		Content: "POST /users creates a user.\n\nAlso POST /users is documented.\n",
	}
	endpoints := sd.DetectEndpoints(doc)
	count := 0
	for _, ep := range endpoints {
		if ep.Method == "POST" && ep.Path == "/users" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("DetectEndpoints: duplicate endpoint found %d times, want 1", count)
	}
}

func TestDetectEndpoints_EmptyDoc(t *testing.T) {
	sd := NewServiceDetector()
	doc := &Document{ID: "svc.md", Content: "# No endpoints here\n"}
	eps := sd.DetectEndpoints(doc)
	if len(eps) != 0 {
		t.Errorf("DetectEndpoints: expected 0 endpoints for non-API doc, got %d", len(eps))
	}
}

// ---------------------------------------------------------------------------
// ServiceDetector — RankServices tests
// ---------------------------------------------------------------------------

func TestRankServices_OrderedByConfidence(t *testing.T) {
	sd := NewServiceDetector()
	candidates := []Service{
		{ID: "c", Confidence: 0.4},
		{ID: "a", Confidence: 0.9},
		{ID: "b", Confidence: 0.7},
	}
	ranked := sd.RankServices(candidates)
	if ranked[0].Confidence < ranked[1].Confidence {
		t.Errorf("RankServices: first=%.2f should be >= second=%.2f",
			ranked[0].Confidence, ranked[1].Confidence)
	}
	if ranked[1].Confidence < ranked[2].Confidence {
		t.Errorf("RankServices: second=%.2f should be >= third=%.2f",
			ranked[1].Confidence, ranked[2].Confidence)
	}
}

func TestRankServices_StableByID(t *testing.T) {
	sd := NewServiceDetector()
	// Same confidence: alphabetical by ID.
	candidates := []Service{
		{ID: "z-svc", Confidence: 0.9},
		{ID: "a-svc", Confidence: 0.9},
		{ID: "m-svc", Confidence: 0.9},
	}
	ranked := sd.RankServices(candidates)
	if ranked[0].ID != "a-svc" || ranked[1].ID != "m-svc" || ranked[2].ID != "z-svc" {
		t.Errorf("RankServices same confidence: want a,m,z got %v,%v,%v",
			ranked[0].ID, ranked[1].ID, ranked[2].ID)
	}
}

// ---------------------------------------------------------------------------
// ServiceConfig — LoadServiceConfig tests
// ---------------------------------------------------------------------------

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "services.yaml")
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatalf("writeTemp: %v", err)
	}
	return p
}

func TestLoadServiceConfig_MissingFile(t *testing.T) {
	cfg, err := LoadServiceConfig("/tmp/does-not-exist-999/services.yaml")
	if err != nil {
		t.Errorf("LoadServiceConfig missing file: expected nil error, got %v", err)
	}
	if cfg != nil {
		t.Errorf("LoadServiceConfig missing file: expected nil config, got %v", cfg)
	}
}

func TestLoadServiceConfig_ValidYAML(t *testing.T) {
	content := `services:
  - id: api-gateway
    patterns: ["api-gateway", "API Gateway"]
    type: microservice
  - id: user-service
    patterns: ["user-service", "User Service"]
    type: microservice
`
	p := writeTemp(t, content)
	cfg, err := LoadServiceConfig(p)
	if err != nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadServiceConfig: expected non-nil config")
	}
	if len(cfg.Services) != 2 {
		t.Errorf("Services count = %d, want 2", len(cfg.Services))
	}
	if cfg.Services[0].ID != "api-gateway" {
		t.Errorf("Services[0].ID = %q, want %q", cfg.Services[0].ID, "api-gateway")
	}
	if len(cfg.Services[0].Patterns) != 2 {
		t.Errorf("Services[0].Patterns = %v, want 2 patterns", cfg.Services[0].Patterns)
	}
	if cfg.Services[0].Type != "microservice" {
		t.Errorf("Services[0].Type = %q, want %q", cfg.Services[0].Type, "microservice")
	}
}

func TestLoadServiceConfig_CaseInsensitivePatterns(t *testing.T) {
	content := `services:
  - id: auth
    patterns: ["AUTH-SERVICE", "Auth Service"]
    type: microservice
`
	p := writeTemp(t, content)
	cfg, err := LoadServiceConfig(p)
	if err != nil || cfg == nil {
		t.Fatalf("LoadServiceConfig: %v, cfg=%v", err, cfg)
	}

	// Test that pattern matching is case-insensitive.
	matched := matchesPatterns("services/auth-service.md", "Auth Service", cfg.Services[0].Patterns)
	if !matched {
		t.Error("matchesPatterns should match case-insensitively")
	}
}

func TestLoadServiceConfig_ConfiguredServicesHighConfidence(t *testing.T) {
	content := `services:
  - id: gateway
    patterns: ["gateway"]
    type: microservice
`
	p := writeTemp(t, content)
	cfg, err := LoadServiceConfig(p)
	if err != nil || cfg == nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}

	sd := NewServiceDetectorWithConfig(cfg)
	g := makeServiceGraph(t, []struct{ id, title string }{
		{"api-gateway.md", "API Gateway"},
	})

	services := sd.DetectServices(g, nil)
	if len(services) == 0 {
		t.Fatal("expected at least one service from configured pattern")
	}
	if services[0].Confidence != ConfidenceConfigured {
		t.Errorf("configured service confidence=%.2f, want %.2f", services[0].Confidence, ConfidenceConfigured)
	}
}

// ---------------------------------------------------------------------------
// DependencyAnalyzer — BuildServiceGraph tests
// ---------------------------------------------------------------------------

func buildTestServiceGraph(t *testing.T) (*Graph, []Service) {
	t.Helper()
	g := NewGraph()
	for _, id := range []string{
		"auth-service.md",
		"user-service.md",
		"payment-service.md",
		"db.md",
	} {
		_ = g.AddNode(&Node{ID: id, Title: strings.TrimSuffix(id, ".md"), Type: "document"})
	}

	services := []Service{
		{ID: "auth-service", File: "auth-service.md", Confidence: 0.9},
		{ID: "user-service", File: "user-service.md", Confidence: 0.9},
		{ID: "payment-service", File: "payment-service.md", Confidence: 0.9},
	}
	// Edges: auth -> user, user -> payment
	e1, _ := NewEdge("auth-service.md", "user-service.md", EdgeCalls, 0.9, "calls user")
	e2, _ := NewEdge("user-service.md", "payment-service.md", EdgeCalls, 0.9, "calls payment")
	// Non-service node — should be excluded from service graph.
	e3, _ := NewEdge("auth-service.md", "db.md", EdgeReferences, 1.0, "db link")
	_ = g.AddEdge(e1)
	_ = g.AddEdge(e2)
	_ = g.AddEdge(e3)

	return g, services
}

func TestBuildServiceGraph_DirectDeps(t *testing.T) {
	g, services := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, services)

	deps := da.GetDirectDeps("auth-service")
	if len(deps) != 1 || deps[0] != "user-service" {
		t.Errorf("GetDirectDeps(auth-service) = %v, want [user-service]", deps)
	}
}

func TestBuildServiceGraph_NonServiceEdgeExcluded(t *testing.T) {
	g, services := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, services)

	// db.md is not in services list — should not appear as a dependency.
	deps := da.GetDirectDeps("auth-service")
	for _, d := range deps {
		if d == "db" || d == "db.md" {
			t.Errorf("non-service node db.md should not appear in service dependencies")
		}
	}
}

func TestGetDirectDeps_Unknown(t *testing.T) {
	g, services := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, services)

	deps := da.GetDirectDeps("unknown-service")
	if deps != nil {
		t.Errorf("GetDirectDeps(unknown) = %v, want nil", deps)
	}
}

func TestGetDirectDeps_NoDeps(t *testing.T) {
	g, services := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, services)

	// payment-service has no outgoing edges.
	deps := da.GetDirectDeps("payment-service")
	if len(deps) != 0 {
		t.Errorf("GetDirectDeps(payment-service) = %v, want empty", deps)
	}
}

// ---------------------------------------------------------------------------
// DependencyAnalyzer — Transitive dependency tests
// ---------------------------------------------------------------------------

func TestGetTransitiveDeps_LinearChain(t *testing.T) {
	// A -> B -> C -> D
	g, svcs := makeLinearChain(t, []string{"a", "b", "c", "d"})
	da := NewDependencyAnalyzer(g, svcs)

	deps := da.GetTransitiveDeps("a")
	if len(deps) != 3 {
		t.Errorf("GetTransitiveDeps(a) in chain a->b->c->d: got %v (len=%d), want 3", deps, len(deps))
	}
}

func TestGetTransitiveDeps_Branching(t *testing.T) {
	// A -> {B, C, D}
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b", "c", "d"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	for _, tgt := range []string{"b.md", "c.md", "d.md"} {
		e, _ := NewEdge("a.md", tgt, EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	da := NewDependencyAnalyzer(g, svcs)

	deps := da.GetTransitiveDeps("a")
	if len(deps) != 3 {
		t.Errorf("GetTransitiveDeps(a) branching: got %v, want [b,c,d]", deps)
	}
}

func TestGetTransitiveDeps_Diamond(t *testing.T) {
	// A -> {B, C}, B -> D, C -> D
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b", "c", "d"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	edges := [][2]string{{"a.md", "b.md"}, {"a.md", "c.md"}, {"b.md", "d.md"}, {"c.md", "d.md"}}
	for _, pair := range edges {
		e, _ := NewEdge(pair[0], pair[1], EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	da := NewDependencyAnalyzer(g, svcs)

	deps := da.GetTransitiveDeps("a")
	// Should contain b, c, d — exactly 3, not 4 (d should not appear twice).
	if len(deps) != 3 {
		t.Errorf("GetTransitiveDeps diamond: got %v (len=%d), want 3", deps, len(deps))
	}
}

func TestGetTransitiveDeps_Unknown(t *testing.T) {
	g, svcs := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, svcs)
	if da.GetTransitiveDeps("nonexistent") != nil {
		t.Error("GetTransitiveDeps(nonexistent) should return nil")
	}
}

// ---------------------------------------------------------------------------
// DependencyAnalyzer — Path finding tests
// ---------------------------------------------------------------------------

func TestFindPath_LinearChain(t *testing.T) {
	// A -> B -> C -> D
	g, svcs := makeLinearChain(t, []string{"a", "b", "c", "d"})
	da := NewDependencyAnalyzer(g, svcs)

	paths := da.FindPath("a", "d")
	if len(paths) == 0 {
		t.Fatal("FindPath(a,d): expected at least one path, got none")
	}
	p := paths[0]
	if p[0] != "a" || p[len(p)-1] != "d" {
		t.Errorf("FindPath(a,d): path should start at a and end at d: %v", p)
	}
}

func TestFindPath_DiamondTwoPaths(t *testing.T) {
	// A -> {B, C}, B -> D, C -> D
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b", "c", "d"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	edges := [][2]string{{"a.md", "b.md"}, {"a.md", "c.md"}, {"b.md", "d.md"}, {"c.md", "d.md"}}
	for _, pair := range edges {
		e, _ := NewEdge(pair[0], pair[1], EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	da := NewDependencyAnalyzer(g, svcs)

	paths := da.FindPath("a", "d")
	if len(paths) != 2 {
		t.Errorf("FindPath diamond a->d: got %d paths, want 2", len(paths))
	}
}

func TestFindPath_Unreachable(t *testing.T) {
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	// No edges between a and b.
	da := NewDependencyAnalyzer(g, svcs)

	paths := da.FindPath("a", "b")
	if len(paths) != 0 {
		t.Errorf("FindPath(a,b) unreachable: expected 0 paths, got %d", len(paths))
	}
}

func TestFindPath_UnknownService(t *testing.T) {
	g, svcs := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, svcs)

	if da.FindPath("auth-service", "nonexistent") != nil {
		t.Error("FindPath to unknown service should return nil")
	}
	if da.FindPath("nonexistent", "user-service") != nil {
		t.Error("FindPath from unknown service should return nil")
	}
}

func TestFindPath_SameService(t *testing.T) {
	g, svcs := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, svcs)

	if da.FindPath("auth-service", "auth-service") != nil {
		t.Error("FindPath(x, x) should return nil")
	}
}

// ---------------------------------------------------------------------------
// DependencyAnalyzer — Cycle detection tests
// ---------------------------------------------------------------------------

func TestDetectCycles_NoCycles(t *testing.T) {
	// A -> B -> C (DAG, no cycles)
	g, svcs := makeLinearChain(t, []string{"a", "b", "c"})
	da := NewDependencyAnalyzer(g, svcs)

	cycles := da.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("DetectCycles DAG: expected 0 cycles, got %d: %v", len(cycles), cycles)
	}
}

func TestDetectCycles_SimpleCycle(t *testing.T) {
	// A -> B -> A
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	e1, _ := NewEdge("a.md", "b.md", EdgeCalls, 0.9, "")
	e2, _ := NewEdge("b.md", "a.md", EdgeCalls, 0.9, "")
	_ = g.AddEdge(e1)
	_ = g.AddEdge(e2)
	da := NewDependencyAnalyzer(g, svcs)

	cycles := da.DetectCycles()
	if len(cycles) == 0 {
		t.Error("DetectCycles: should detect cycle a->b->a")
	}
}

func TestDetectCycles_LongerCycle(t *testing.T) {
	// A -> B -> C -> A
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b", "c"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	pairs := [][2]string{{"a.md", "b.md"}, {"b.md", "c.md"}, {"c.md", "a.md"}}
	for _, p := range pairs {
		e, _ := NewEdge(p[0], p[1], EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	da := NewDependencyAnalyzer(g, svcs)

	cycles := da.DetectCycles()
	if len(cycles) == 0 {
		t.Error("DetectCycles: should detect cycle a->b->c->a")
	}
	// Cycle path must start and end with same node.
	for _, cycle := range cycles {
		if len(cycle) < 2 {
			t.Errorf("cycle too short: %v", cycle)
			continue
		}
		if cycle[0] != cycle[len(cycle)-1] {
			t.Errorf("cycle should start and end at same node: %v", cycle)
		}
	}
}

func TestDetectCycles_MultipleCycles(t *testing.T) {
	// A -> B -> A (cycle 1) and C -> D -> C (cycle 2)
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b", "c", "d"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	for _, pair := range [][2]string{
		{"a.md", "b.md"}, {"b.md", "a.md"},
		{"c.md", "d.md"}, {"d.md", "c.md"},
	} {
		e, _ := NewEdge(pair[0], pair[1], EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	da := NewDependencyAnalyzer(g, svcs)

	cycles := da.DetectCycles()
	if len(cycles) < 2 {
		t.Errorf("DetectCycles: expected >=2 cycles for two independent cycles, got %d: %v", len(cycles), cycles)
	}
}

func TestDetectCycles_EmptyGraph(t *testing.T) {
	da := NewDependencyAnalyzer(NewGraph(), nil)
	cycles := da.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("DetectCycles empty graph: expected 0 cycles, got %d", len(cycles))
	}
}

// ---------------------------------------------------------------------------
// DependencyAnalyzer — FindDependencyChain (BFS shortest path) tests
// ---------------------------------------------------------------------------

func TestFindDependencyChain_LinearChain(t *testing.T) {
	// A -> B -> C -> D
	g, svcs := makeLinearChain(t, []string{"a", "b", "c", "d"})
	da := NewDependencyAnalyzer(g, svcs)

	chain := da.FindDependencyChain("a", "d")
	if len(chain.Path) == 0 {
		t.Fatal("FindDependencyChain(a,d): expected non-empty path")
	}
	if chain.Path[0] != "a" || chain.Path[len(chain.Path)-1] != "d" {
		t.Errorf("chain path should go from a to d: %v", chain.Path)
	}
	if chain.Distance != len(chain.Path)-1 {
		t.Errorf("Distance=%d, but path length=%d", chain.Distance, len(chain.Path))
	}
}

func TestFindDependencyChain_ShortestPath(t *testing.T) {
	// A -> B -> C and A -> C (direct).  Shortest path is A -> C (distance=1).
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b", "c"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	for _, pair := range [][2]string{{"a.md", "b.md"}, {"b.md", "c.md"}, {"a.md", "c.md"}} {
		e, _ := NewEdge(pair[0], pair[1], EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	da := NewDependencyAnalyzer(g, svcs)

	chain := da.FindDependencyChain("a", "c")
	if chain.Distance != 1 {
		t.Errorf("FindDependencyChain shortest: Distance=%d, want 1", chain.Distance)
	}
}

func TestFindDependencyChain_Unreachable(t *testing.T) {
	g := NewGraph()
	svcs := []Service{}
	for _, id := range []string{"a", "b"} {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs = append(svcs, Service{ID: id, File: id + ".md", Confidence: 0.9})
	}
	da := NewDependencyAnalyzer(g, svcs)

	chain := da.FindDependencyChain("a", "b")
	if len(chain.Path) != 0 {
		t.Errorf("FindDependencyChain unreachable: expected empty path, got %v", chain.Path)
	}
	if chain.Distance != 0 {
		t.Errorf("FindDependencyChain unreachable: Distance=%d, want 0", chain.Distance)
	}
}

func TestFindDependencyChain_SameService(t *testing.T) {
	g, svcs := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, svcs)

	chain := da.FindDependencyChain("auth-service", "auth-service")
	if len(chain.Path) != 0 {
		t.Errorf("FindDependencyChain same service: expected empty path, got %v", chain.Path)
	}
}

func TestFindDependencyChain_DepthLimit(t *testing.T) {
	// Chain of 7 services: a->b->c->d->e->f->g (exceeds depth limit of 5).
	g, svcs := makeLinearChain(t, []string{"a", "b", "c", "d", "e", "f", "g"})
	da := NewDependencyAnalyzer(g, svcs)

	chain := da.FindDependencyChain("a", "g")
	// Path requires 6 hops; depth limit is 5 — should return empty.
	if len(chain.Path) != 0 {
		t.Errorf("FindDependencyChain depth limit: expected empty path for 6-hop chain (limit=5), got %v", chain.Path)
	}
}

// ---------------------------------------------------------------------------
// DependencyAnalyzer — GetServiceGraph and edge type mapping tests
// ---------------------------------------------------------------------------

func TestGetServiceGraph_NotNil(t *testing.T) {
	g, svcs := buildTestServiceGraph(t)
	da := NewDependencyAnalyzer(g, svcs)
	sg := da.GetServiceGraph()
	if sg == nil {
		t.Error("GetServiceGraph should return non-nil service graph")
	}
	if len(sg.Services) != len(svcs) {
		t.Errorf("ServiceGraph.Services count = %d, want %d", len(sg.Services), len(svcs))
	}
}

func TestEdgeTypeToDepType_AllEdgeTypes(t *testing.T) {
	// Ensure all EdgeType constants map to non-empty dep types.
	for _, et := range []EdgeType{
		EdgeCalls, EdgeDependsOn, EdgeMentions, EdgeReferences, EdgeImplements, "unknown",
	} {
		depType := edgeTypeToDepType(et)
		if depType == "" {
			t.Errorf("edgeTypeToDepType(%q) returned empty string", et)
		}
	}
}

func TestEdgeTypeToDepType_CallsMapsToDirect(t *testing.T) {
	if edgeTypeToDepType(EdgeCalls) != "direct-call" {
		t.Errorf("EdgeCalls should map to 'direct-call'")
	}
	if edgeTypeToDepType(EdgeDependsOn) != "direct-call" {
		t.Errorf("EdgeDependsOn should map to 'direct-call'")
	}
}

// ---------------------------------------------------------------------------
// Benchmark — 100-service graph performance
// ---------------------------------------------------------------------------

func BenchmarkDetectCycles_100Services(b *testing.B) {
	const n = 100
	g := NewGraph()
	svcs := make([]Service, n)
	for i := 0; i < n; i++ {
		id := serviceID(i)
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs[i] = Service{ID: id, File: id + ".md", Confidence: 0.9}
	}
	// Build a linear chain: 0->1->2->...->99 (no cycles).
	for i := 0; i < n-1; i++ {
		e, _ := NewEdge(serviceID(i)+".md", serviceID(i+1)+".md", EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		da := NewDependencyAnalyzer(g, svcs)
		_ = da.DetectCycles()
	}
}

func BenchmarkGetTransitiveDeps_100Services(b *testing.B) {
	const n = 100
	g := NewGraph()
	svcs := make([]Service, n)
	for i := 0; i < n; i++ {
		id := serviceID(i)
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs[i] = Service{ID: id, File: id + ".md", Confidence: 0.9}
	}
	for i := 0; i < n-1; i++ {
		e, _ := NewEdge(serviceID(i)+".md", serviceID(i+1)+".md", EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}

	da := NewDependencyAnalyzer(g, svcs)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = da.GetTransitiveDeps(serviceID(0))
	}
}

// ---------------------------------------------------------------------------
// helpers shared by tests
// ---------------------------------------------------------------------------

// makeLinearChain builds a graph with a linear service chain: a->b->c->...
// node IDs are from ids (e.g. []string{"a","b","c"}).
func makeLinearChain(t *testing.T, ids []string) (*Graph, []Service) {
	t.Helper()
	g := NewGraph()
	svcs := make([]Service, len(ids))
	for i, id := range ids {
		_ = g.AddNode(&Node{ID: id + ".md", Title: id, Type: "document"})
		svcs[i] = Service{ID: id, File: id + ".md", Confidence: 0.9}
	}
	for i := 0; i < len(ids)-1; i++ {
		e, _ := NewEdge(ids[i]+".md", ids[i+1]+".md", EdgeCalls, 0.9, "")
		_ = g.AddEdge(e)
	}
	return g, svcs
}

// serviceID returns a zero-padded service ID for benchmarks.
func serviceID(i int) string {
	return "svc" + padInt(i, 3)
}

// padInt formats i as a zero-padded decimal string of width w.
func padInt(i, w int) string {
	s := strings.Repeat("0", w) + intToStr(i)
	return s[len(s)-w:]
}

// intToStr is a simple integer-to-string helper that avoids importing strconv.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
