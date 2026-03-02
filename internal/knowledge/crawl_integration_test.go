package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupIntegrationDocs creates a temp directory with a richer set of markdown
// files forming a known graph suitable for integration testing.
//
// Graph:
//
//	api-gateway.md -> user-service.md -> auth-service.md
//	api-gateway.md -> payment-service.md -> auth-service.md
//	payment-service.md -> notification-service.md
func setupIntegrationDocs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"api-gateway.md": `# API Gateway

Routes requests to downstream services.

See [user-service](./user-service.md) for user management.
See [payment-service](./payment-service.md) for payments.

## Endpoints

- GET /health
- POST /api/v1/users
- POST /api/v1/payments
`,
		"user-service.md": `# User Service

Handles user management and profiles.

Depends on [auth-service](./auth-service.md) for token validation.

## Endpoints

- GET /users/:id
- PUT /users/:id
`,
		"auth-service.md": `# Auth Service

Provides JWT token validation and issuance.

## Endpoints

- POST /auth/token
- GET /auth/validate
`,
		"payment-service.md": `# Payment Service

Handles payment processing.

Depends on [auth-service](./auth-service.md) for authorization.
Sends receipts via [notification-service](./notification-service.md).

## Endpoints

- POST /payments
- GET /payments/:id
`,
		"notification-service.md": `# Notification Service

Sends email and push notifications.

## Endpoints

- POST /notify/email
- POST /notify/push
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("create %q: %v", path, err)
		}
	}

	// Build the index.
	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir .bmd: %v", err)
	}
	if err := CmdIndex([]string{"--dir", dir, "--db", dbPath}); err != nil {
		t.Fatalf("CmdIndex: %v", err)
	}

	return dir
}

// captureStdoutIntegration runs fn while capturing stdout.
func captureStdoutIntegration(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}

// parseCrawlJSON parses a JSON crawl response into envelope + crawl data.
func parseCrawlJSON(t *testing.T, output string) (ContractResponse, crawlResponseJSON) {
	t.Helper()

	var envelope ContractResponse
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("output not valid JSON: %v\nOutput: %s", err, output)
	}

	dataBytes, _ := json.Marshal(envelope.Data)
	var data crawlResponseJSON
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("data not valid crawl JSON: %v", err)
	}
	return envelope, data
}

// ---------------------------------------------------------------------------
// Integration Test 1: Search Then Crawl (Agent Workflow Simulation)
// ---------------------------------------------------------------------------

func TestIntegration_SearchThenCrawl(t *testing.T) {
	dir := setupIntegrationDocs(t)

	// Step 1: Agent searches for "token" using BM25.
	searchOutput := captureStdoutIntegration(t, func() {
		err := CmdQuery([]string{"token", "--dir", dir, "--format", "json", "--top", "3"})
		if err != nil {
			t.Fatalf("CmdQuery error: %v", err)
		}
	})

	// Parse the search JSON output.
	var searchEnvelope map[string]interface{}
	if err := json.Unmarshal([]byte(searchOutput), &searchEnvelope); err != nil {
		t.Fatalf("search output not valid JSON: %v", err)
	}
	status := searchEnvelope["status"]
	if status != "ok" && status != "empty" {
		t.Fatalf("search failed: status=%v", status)
	}
	if status == "empty" {
		t.Fatal("search returned no results for 'token' — test data may need updating")
	}

	// Extract file paths from search results to use as crawl start files.
	searchData, ok := searchEnvelope["data"].(map[string]interface{})
	if !ok {
		t.Fatal("search data is not an object")
	}

	results, ok := searchData["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("expected search to return at least one result")
	}

	// Collect file paths from top search results.
	var startFiles []string
	for _, r := range results {
		entry, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		file, ok := entry["file"].(string)
		if !ok {
			continue
		}
		startFiles = append(startFiles, file)
	}

	if len(startFiles) == 0 {
		t.Fatal("no file paths extracted from search results")
	}

	// Step 2: Agent crawls from the search result files.
	crawlOutput := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", strings.Join(startFiles, ","),
			"--dir", dir,
			"--direction", "forward",
			"--depth", "5",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	envelope, data := parseCrawlJSON(t, crawlOutput)

	if envelope.Status != "ok" {
		t.Errorf("crawl status: %q, want ok", envelope.Status)
	}

	if data.TotalNodes < 1 {
		t.Error("crawl should discover at least 1 node from search results")
	}

	// Verify the workflow produces actionable results: nodes have depth info.
	for nodeID, node := range data.Nodes {
		if node.Depth < 0 {
			t.Errorf("node %q has negative depth: %d", nodeID, node.Depth)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration Test 2: Crawl With Real Graph (Repository Integration)
// ---------------------------------------------------------------------------

func TestIntegration_CrawlWithRealGraph(t *testing.T) {
	dir := setupIntegrationDocs(t)

	// Crawl the full graph from api-gateway.md forward.
	output := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "10",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, data := parseCrawlJSON(t, output)

	// api-gateway links to user-service and payment-service,
	// which link to auth-service and notification-service.
	// We should discover all 5 nodes.
	if data.TotalNodes != 5 {
		t.Errorf("TotalNodes = %d, want 5 (all docs reachable from api-gateway)", data.TotalNodes)
	}

	// Verify specific nodes exist.
	expectedNodes := []string{
		"api-gateway.md",
		"user-service.md",
		"auth-service.md",
		"payment-service.md",
		"notification-service.md",
	}
	for _, n := range expectedNodes {
		if _, ok := data.Nodes[n]; !ok {
			t.Errorf("expected node %q in crawl results", n)
		}
	}

	// api-gateway should be at depth 0.
	if node, ok := data.Nodes["api-gateway.md"]; ok && node.Depth != 0 {
		t.Errorf("api-gateway.md depth = %d, want 0", node.Depth)
	}

	// auth-service should be at depth 2 (gateway -> user/payment -> auth).
	if node, ok := data.Nodes["auth-service.md"]; ok && node.Depth != 2 {
		t.Errorf("auth-service.md depth = %d, want 2", node.Depth)
	}

	// auth-service should have two parents (user-service and payment-service).
	if node, ok := data.Nodes["auth-service.md"]; ok {
		parents := node.Parents
		sort.Strings(parents)
		if len(parents) != 2 {
			t.Errorf("auth-service parents = %v, want 2 parents", parents)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration Test 3: Crawl Performance
// ---------------------------------------------------------------------------

func TestIntegration_CrawlPerformance(t *testing.T) {
	// Build a 50-node graph in memory and measure crawl time.
	nodes := make([]string, 50)
	for i := range nodes {
		nodes[i] = fmt.Sprintf("node-%03d.md", i)
	}

	// Create a chain + fan-out pattern: 0->1->2->...->49, plus 0->10, 0->20, etc.
	var edges [][2]string
	for i := 0; i < 49; i++ {
		edges = append(edges, [2]string{nodes[i], nodes[i+1]})
	}
	for i := 10; i < 50; i += 10 {
		edges = append(edges, [2]string{nodes[0], nodes[i]})
	}

	g := NewGraph()
	for _, id := range nodes {
		_ = g.AddNode(&Node{ID: id, Title: "Title-" + id, Type: "document"})
	}
	for _, pair := range edges {
		e, err := NewEdge(pair[0], pair[1], EdgeReferences, ConfidenceLink, "")
		if err != nil {
			t.Fatalf("NewEdge: %v", err)
		}
		_ = g.AddEdge(e)
	}

	start := time.Now()

	result := g.CrawlMulti(CrawlOptions{
		FromFiles:     []string{nodes[0]},
		Direction:     "forward",
		MaxDepth:      -1,
		IncludeCycles: true,
	})

	elapsed := time.Since(start)

	if result.TotalNodes != 50 {
		t.Errorf("TotalNodes = %d, want 50", result.TotalNodes)
	}

	// Performance target: < 100ms for 50-node graph.
	if elapsed > 100*time.Millisecond {
		t.Errorf("crawl took %v, want < 100ms", elapsed)
	}

	t.Logf("50-node crawl completed in %v", elapsed)
}

// ---------------------------------------------------------------------------
// Integration Test 4: MCP and CLI Match
// ---------------------------------------------------------------------------

func TestIntegration_MCP_and_CLI_Match(t *testing.T) {
	dir := setupIntegrationDocs(t)

	// Run CLI crawl.
	cliOutput := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "10",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, cliData := parseCrawlJSON(t, cliOutput)

	// Run the same crawl using the internal API (simulating MCP path).
	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close() //nolint:errcheck

	graph := NewGraph()
	if err := db.LoadGraph(graph); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	mcpResult := graph.CrawlMulti(CrawlOptions{
		FromFiles:     []string{"api-gateway.md"},
		Direction:     "forward",
		MaxDepth:      10,
		IncludeCycles: true,
	})

	// Both should discover the same number of nodes.
	if cliData.TotalNodes != mcpResult.TotalNodes {
		t.Errorf("node count mismatch: CLI=%d, MCP=%d", cliData.TotalNodes, mcpResult.TotalNodes)
	}

	// Both should discover the same nodes.
	for nodeID := range mcpResult.Nodes {
		if _, ok := cliData.Nodes[nodeID]; !ok {
			t.Errorf("node %q found via MCP but not CLI", nodeID)
		}
	}
	for nodeID := range cliData.Nodes {
		if _, ok := mcpResult.Nodes[nodeID]; !ok {
			t.Errorf("node %q found via CLI but not MCP", nodeID)
		}
	}

	// Edge counts should match.
	if cliData.TotalEdges != mcpResult.TotalEdges {
		t.Errorf("edge count mismatch: CLI=%d, MCP=%d", cliData.TotalEdges, mcpResult.TotalEdges)
	}
}

// ---------------------------------------------------------------------------
// Integration Test 5: Crawl Depth Limiting
// ---------------------------------------------------------------------------

func TestIntegration_CrawlDepthLimiting(t *testing.T) {
	dir := setupIntegrationDocs(t)

	// Depth 1: api-gateway + its direct dependencies (user-service, payment-service).
	output := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "1",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, data := parseCrawlJSON(t, output)

	// At depth 1, should see api-gateway + user-service + payment-service = 3 nodes.
	if data.TotalNodes != 3 {
		t.Errorf("depth=1: TotalNodes = %d, want 3", data.TotalNodes)
	}

	// auth-service should NOT be present (depth 2).
	if _, ok := data.Nodes["auth-service.md"]; ok {
		t.Error("depth=1: auth-service.md should not be discovered")
	}

	// notification-service should NOT be present (depth 2).
	if _, ok := data.Nodes["notification-service.md"]; ok {
		t.Error("depth=1: notification-service.md should not be discovered")
	}

	// Depth 2: should get all 5.
	output2 := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "2",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, data2 := parseCrawlJSON(t, output2)

	if data2.TotalNodes != 5 {
		t.Errorf("depth=2: TotalNodes = %d, want 5", data2.TotalNodes)
	}
}

// ---------------------------------------------------------------------------
// Integration Test 6: Fan-Out Expansion
// ---------------------------------------------------------------------------

func TestIntegration_CrawlFanOutExpansion(t *testing.T) {
	dir := setupIntegrationDocs(t)

	// api-gateway has 2 outgoing edges. Test that all branches are expanded.
	output := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "api-gateway.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "10",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, data := parseCrawlJSON(t, output)

	// api-gateway should have edges to user-service and payment-service.
	gateway := data.Nodes["api-gateway.md"]
	edgesOut := gateway.EdgesOut
	sort.Strings(edgesOut)

	if len(edgesOut) < 2 {
		t.Errorf("api-gateway edges_out = %v, want at least 2", edgesOut)
	}

	// Both branches should be fully expanded.
	if _, ok := data.Nodes["user-service.md"]; !ok {
		t.Error("user-service.md not discovered via fan-out")
	}
	if _, ok := data.Nodes["payment-service.md"]; !ok {
		t.Error("payment-service.md not discovered via fan-out")
	}
	if _, ok := data.Nodes["notification-service.md"]; !ok {
		t.Error("notification-service.md not discovered via payment-service branch")
	}
}

// ---------------------------------------------------------------------------
// Integration Test 7: Cycle Detection
// ---------------------------------------------------------------------------

func TestIntegration_CrawlCycleDetection(t *testing.T) {
	// Create a graph with a cycle: A -> B -> C -> A.
	dir := t.TempDir()

	files := map[string]string{
		"svc-a.md": `# Service A

Calls [svc-b](./svc-b.md).
`,
		"svc-b.md": `# Service B

Calls [svc-c](./svc-c.md).
`,
		"svc-c.md": `# Service C

Calls [svc-a](./svc-a.md).
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	dbPath := filepath.Join(dir, ".bmd", "knowledge.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := CmdIndex([]string{"--dir", dir, "--db", dbPath}); err != nil {
		t.Fatalf("CmdIndex: %v", err)
	}

	output := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "svc-a.md",
			"--dir", dir,
			"--direction", "forward",
			"--depth", "10",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, data := parseCrawlJSON(t, output)

	// All 3 nodes should be discovered.
	if data.TotalNodes != 3 {
		t.Errorf("TotalNodes = %d, want 3", data.TotalNodes)
	}

	// Cycles should be detected (CmdCrawl enables IncludeCycles).
	if len(data.Cycles) == 0 {
		t.Error("expected at least one cycle to be detected in A->B->C->A graph")
	}

	// Verify cycle is closed (first == last).
	for _, cycle := range data.Cycles {
		if len(cycle.Path) < 3 {
			t.Errorf("cycle path too short: %v", cycle.Path)
			continue
		}
		if cycle.Path[0] != cycle.Path[len(cycle.Path)-1] {
			t.Errorf("cycle not closed: %v", cycle.Path)
		}
		if cycle.Type == "" {
			t.Error("cycle type should not be empty")
		}
	}
}

// ---------------------------------------------------------------------------
// Integration Test 8: All Formats Produce Valid Output
// ---------------------------------------------------------------------------

func TestIntegration_AllFormats(t *testing.T) {
	dir := setupIntegrationDocs(t)

	formats := []struct {
		name   string
		format string
		check  func(t *testing.T, output string)
	}{
		{
			name:   "json",
			format: "json",
			check: func(t *testing.T, output string) {
				var env ContractResponse
				if err := json.Unmarshal([]byte(output), &env); err != nil {
					t.Fatalf("JSON parse error: %v", err)
				}
				if env.Status != "ok" {
					t.Errorf("JSON status = %q, want ok", env.Status)
				}
			},
		},
		{
			name:   "tree",
			format: "tree",
			check: func(t *testing.T, output string) {
				if strings.HasPrefix(output, "{") {
					t.Error("tree output should not be JSON")
				}
				if !strings.Contains(output, "api-gateway.md") {
					t.Error("tree should contain start node")
				}
			},
		},
		{
			name:   "dot",
			format: "dot",
			check: func(t *testing.T, output string) {
				if !strings.HasPrefix(output, "digraph") {
					t.Error("DOT should start with 'digraph'")
				}
				if !strings.Contains(output, "->") {
					t.Error("DOT should contain edges")
				}
				if !strings.HasSuffix(strings.TrimSpace(output), "}") {
					t.Error("DOT should end with '}'")
				}
			},
		},
		{
			name:   "list",
			format: "list",
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "depth=") {
					t.Error("list should contain depth labels")
				}
				if !strings.Contains(output, "depth=0") {
					t.Error("list should have depth=0 for start node")
				}
			},
		},
	}

	for _, tt := range formats {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdoutIntegration(t, func() {
				err := CmdCrawl([]string{
					"--from-multiple", "api-gateway.md",
					"--dir", dir,
					"--direction", "forward",
					"--depth", "10",
					"--format", tt.format,
				})
				if err != nil {
					t.Fatalf("CmdCrawl error: %v", err)
				}
			})

			if output == "" {
				t.Fatal("expected non-empty output")
			}
			tt.check(t, output)
		})
	}
}

// ---------------------------------------------------------------------------
// Edge Case: Deep Graph (50+ levels)
// ---------------------------------------------------------------------------

func TestEdgeCase_DeepGraph(t *testing.T) {
	// Build a 60-node linear chain.
	nodes := make([]string, 60)
	for i := range nodes {
		nodes[i] = fmt.Sprintf("deep-%03d.md", i)
	}

	var edges [][2]string
	for i := 0; i < 59; i++ {
		edges = append(edges, [2]string{nodes[i], nodes[i+1]})
	}

	g := crawlTestGraph(t, nodes, edges)

	// Unlimited depth should discover all 60 nodes.
	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{nodes[0]},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result.TotalNodes != 60 {
		t.Errorf("unlimited depth: TotalNodes = %d, want 60", result.TotalNodes)
	}

	// Last node should be at depth 59.
	if info, ok := result.Nodes[nodes[59]]; ok && info.Depth != 59 {
		t.Errorf("last node depth = %d, want 59", info.Depth)
	}

	// Depth limit of 10 should cap at 11 nodes (0..10).
	limited := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{nodes[0]},
		Direction: "forward",
		MaxDepth:  10,
	})
	if limited.TotalNodes != 11 {
		t.Errorf("depth=10: TotalNodes = %d, want 11", limited.TotalNodes)
	}
}

// ---------------------------------------------------------------------------
// Edge Case: Wide Graph (100 edges from one node)
// ---------------------------------------------------------------------------

func TestEdgeCase_WideGraph(t *testing.T) {
	// Build a star graph: hub -> 100 leaf nodes.
	hub := "hub.md"
	nodes := []string{hub}
	var edges [][2]string
	for i := 0; i < 100; i++ {
		leaf := fmt.Sprintf("leaf-%03d.md", i)
		nodes = append(nodes, leaf)
		edges = append(edges, [2]string{hub, leaf})
	}

	g := crawlTestGraph(t, nodes, edges)

	start := time.Now()
	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{hub},
		Direction: "forward",
		MaxDepth:  -1,
	})
	elapsed := time.Since(start)

	// Should discover all 101 nodes.
	if result.TotalNodes != 101 {
		t.Errorf("TotalNodes = %d, want 101", result.TotalNodes)
	}

	// All leaves should be at depth 1.
	for i := 0; i < 100; i++ {
		leaf := fmt.Sprintf("leaf-%03d.md", i)
		if info, ok := result.Nodes[leaf]; ok && info.Depth != 1 {
			t.Errorf("leaf %s depth = %d, want 1", leaf, info.Depth)
			break
		}
	}

	// Hub should have 100 edges out.
	if hubInfo, ok := result.Nodes[hub]; ok {
		if len(hubInfo.EdgesOut) != 100 {
			t.Errorf("hub edges_out = %d, want 100", len(hubInfo.EdgesOut))
		}
	}

	// Performance: should complete in < 100ms.
	if elapsed > 100*time.Millisecond {
		t.Errorf("wide graph crawl took %v, want < 100ms", elapsed)
	}

	t.Logf("101-node star crawl completed in %v", elapsed)
}

// ---------------------------------------------------------------------------
// Edge Case: Disconnected Nodes
// ---------------------------------------------------------------------------

func TestEdgeCase_DisconnectedNodes(t *testing.T) {
	// Graph with 5 nodes but no edges between clusters.
	// Cluster 1: A -> B
	// Cluster 2: C -> D
	// Isolated: E
	g := crawlTestGraph(t,
		[]string{"A", "B", "C", "D", "E"},
		[][2]string{{"A", "B"}, {"C", "D"}},
	)

	// Crawl from A should only discover A and B.
	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result.TotalNodes != 2 {
		t.Errorf("crawl from A: TotalNodes = %d, want 2", result.TotalNodes)
	}
	if _, ok := result.Nodes["C"]; ok {
		t.Error("disconnected node C should not be discovered from A")
	}
	if _, ok := result.Nodes["E"]; ok {
		t.Error("isolated node E should not be discovered from A")
	}

	// Multi-start from A and C should discover both clusters but not E.
	result2 := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A", "C"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result2.TotalNodes != 4 {
		t.Errorf("crawl from A,C: TotalNodes = %d, want 4", result2.TotalNodes)
	}
	if _, ok := result2.Nodes["E"]; ok {
		t.Error("isolated node E should not be discovered")
	}
}

// ---------------------------------------------------------------------------
// Edge Case: Empty Graph
// ---------------------------------------------------------------------------

func TestEdgeCase_EmptyGraph(t *testing.T) {
	g := NewGraph()

	result := g.CrawlMulti(CrawlOptions{
		FromFiles:     []string{"nonexistent.md"},
		Direction:     "forward",
		MaxDepth:      -1,
		IncludeCycles: true,
	})

	if result.TotalNodes != 0 {
		t.Errorf("empty graph: TotalNodes = %d, want 0", result.TotalNodes)
	}
	if result.TotalEdges != 0 {
		t.Errorf("empty graph: TotalEdges = %d, want 0", result.TotalEdges)
	}
	if len(result.StartNodes) != 0 {
		t.Errorf("empty graph: StartNodes = %v, want []", result.StartNodes)
	}
	if len(result.Cycles) != 0 {
		t.Errorf("empty graph: Cycles count = %d, want 0", len(result.Cycles))
	}
}

// ---------------------------------------------------------------------------
// Edge Case: Large Multi-Start (10+ files)
// ---------------------------------------------------------------------------

func TestEdgeCase_LargeMultiStart(t *testing.T) {
	// Build a graph with 20 independent chains of length 3.
	var allNodes []string
	var edges [][2]string
	var startFiles []string

	for i := 0; i < 20; i++ {
		a := fmt.Sprintf("chain%02d-a.md", i)
		b := fmt.Sprintf("chain%02d-b.md", i)
		c := fmt.Sprintf("chain%02d-c.md", i)
		allNodes = append(allNodes, a, b, c)
		edges = append(edges, [2]string{a, b}, [2]string{b, c})
		startFiles = append(startFiles, a)
	}

	g := crawlTestGraph(t, allNodes, edges)

	start := time.Now()
	result := g.CrawlMulti(CrawlOptions{
		FromFiles: startFiles,
		Direction: "forward",
		MaxDepth:  -1,
	})
	elapsed := time.Since(start)

	// Should discover all 60 nodes.
	if result.TotalNodes != 60 {
		t.Errorf("multi-start: TotalNodes = %d, want 60", result.TotalNodes)
	}

	// Should have 20 start nodes.
	if len(result.StartNodes) != 20 {
		t.Errorf("StartNodes = %d, want 20", len(result.StartNodes))
	}

	// Performance should be acceptable.
	if elapsed > 100*time.Millisecond {
		t.Errorf("20-chain multi-start crawl took %v, want < 100ms", elapsed)
	}

	t.Logf("20-chain multi-start (60 nodes) completed in %v", elapsed)
}

// ---------------------------------------------------------------------------
// Edge Case: Backward Traversal Integration
// ---------------------------------------------------------------------------

func TestEdgeCase_BackwardTraversal(t *testing.T) {
	dir := setupIntegrationDocs(t)

	// Crawl backward from auth-service: should discover who depends on it.
	output := captureStdoutIntegration(t, func() {
		err := CmdCrawl([]string{
			"--from-multiple", "auth-service.md",
			"--dir", dir,
			"--direction", "backward",
			"--depth", "10",
			"--format", "json",
		})
		if err != nil {
			t.Fatalf("CmdCrawl error: %v", err)
		}
	})

	_, data := parseCrawlJSON(t, output)

	// auth-service is depended on by user-service and payment-service,
	// which are depended on by api-gateway. Should find 4 nodes backward.
	if data.TotalNodes < 3 {
		t.Errorf("backward from auth-service: TotalNodes = %d, want >= 3", data.TotalNodes)
	}

	// Strategy should be backward.
	if data.Strategy != "backward" {
		t.Errorf("Strategy = %q, want backward", data.Strategy)
	}

	// api-gateway should be reachable backward from auth-service.
	if _, ok := data.Nodes["api-gateway.md"]; !ok {
		t.Error("api-gateway.md should be reachable backward from auth-service")
	}
}
