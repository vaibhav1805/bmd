package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── GenerateRelationshipManifest tests ──────────────────────────────────────

func TestGenerateRelationshipManifest_Basic(t *testing.T) {
	e1, _ := NewEdge("services/auth.md", "services/db.md", EdgeReferences, 1.0, "[database](../db.md)")
	e2, _ := NewEdge("services/auth.md", "services/cache.md", EdgeMentions, 0.7, "uses cache-layer")
	e3, _ := NewEdge("services/api.md", "services/auth.md", EdgeDependsOn, 0.8, "depends on auth")

	m := GenerateRelationshipManifest([]*Edge{e1, e2, e3}, nil)

	if m.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", m.Version)
	}
	if len(m.Relationships) != 3 {
		t.Errorf("expected 3 relationships, got %d", len(m.Relationships))
	}

	// Check sorted by confidence descending.
	if m.Relationships[0].Confidence < m.Relationships[len(m.Relationships)-1].Confidence {
		t.Error("relationships should be sorted by confidence descending")
	}

	// Verify first relationship (highest confidence = 1.0).
	first := m.Relationships[0]
	if first.Source != "services/auth.md" || first.Target != "services/db.md" {
		t.Errorf("expected auth->db first, got %s->%s", first.Source, first.Target)
	}
	if first.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", first.Confidence)
	}
	if len(first.Signals) != 1 {
		t.Errorf("expected 1 signal, got %d", len(first.Signals))
	}
	if first.Status != "pending" {
		t.Errorf("expected status pending, got %s", first.Status)
	}
}

func TestGenerateRelationshipManifest_GroupsEdges(t *testing.T) {
	// Two edges between the same pair should be grouped.
	e1, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "link")
	e2, _ := NewEdge("a.md", "b.md", EdgeMentions, 0.7, "mention")

	m := GenerateRelationshipManifest([]*Edge{e1, e2}, nil)

	if len(m.Relationships) != 1 {
		t.Errorf("expected 1 grouped relationship, got %d", len(m.Relationships))
	}
	if len(m.Relationships[0].Signals) != 2 {
		t.Errorf("expected 2 signals, got %d", len(m.Relationships[0].Signals))
	}
	// Highest confidence should win.
	if m.Relationships[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", m.Relationships[0].Confidence)
	}
}

func TestGenerateRelationshipManifest_EmptyEdges(t *testing.T) {
	m := GenerateRelationshipManifest(nil, nil)
	if len(m.Relationships) != 0 {
		t.Errorf("expected 0 relationships, got %d", len(m.Relationships))
	}
	if m.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", m.Version)
	}
}

func TestGenerateRelationshipManifest_WithRegistry(t *testing.T) {
	e1, _ := NewEdge("services/auth.md", "services/db.md", EdgeReferences, 1.0, "link")

	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth", FileRef: "services/auth.md", Type: ComponentTypeService})
	_ = reg.AddComponent(&RegistryComponent{ID: "cache", Name: "Cache", FileRef: "services/cache.md", Type: ComponentTypeService})
	_ = reg.AddSignal("auth", "cache", Signal{SourceType: SignalMention, Confidence: 0.65, Evidence: "mentions cache", Weight: 1.0})

	m := GenerateRelationshipManifest([]*Edge{e1}, reg)

	// Should have 2 relationships: auth->db (from edge) and auth->cache (from registry).
	if len(m.Relationships) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(m.Relationships))
	}
}

// ─── YAML serialization tests ────────────────────────────────────────────────

func TestSaveAndLoadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	manifest := &RelationshipManifest{
		Version:   "1.0",
		Generated: "2026-03-03T12:00:00Z",
		AlgorithmVersions: AlgorithmVersions{
			Cooccurrence: "1.0",
			Structural:   "1.0",
			Semantic:     "1.0",
			NER:          "1.0",
		},
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "references",
				Confidence: 0.95,
				Signals: []ManifestSignal{
					{Type: "structural", Value: 0.95, Evidence: "link"},
				},
				Reviewed:  false,
				Status:    "pending",
				UserNotes: "",
			},
		},
	}

	if err := SaveRelationshipManifest(manifest, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadRelationshipManifest(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded manifest is nil")
	}

	if loaded.Version != "1.0" {
		t.Errorf("version: got %s", loaded.Version)
	}
	if len(loaded.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(loaded.Relationships))
	}
	r := loaded.Relationships[0]
	if r.Source != "a.md" || r.Target != "b.md" {
		t.Errorf("source/target mismatch: %s->%s", r.Source, r.Target)
	}
	if r.Confidence != 0.95 {
		t.Errorf("confidence: got %f", r.Confidence)
	}
	if r.Status != "pending" {
		t.Errorf("status: got %s", r.Status)
	}
	if len(r.Signals) != 1 {
		t.Errorf("signals count: got %d", len(r.Signals))
	}
}

func TestLoadManifest_NotFound(t *testing.T) {
	m, err := LoadRelationshipManifest("/nonexistent/path.yaml")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if m != nil {
		t.Error("expected nil manifest for missing file")
	}
}

func TestLoadManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(path, []byte("not: [valid: yaml: {broken"), 0o644)

	_, err := LoadRelationshipManifest(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// ─── Summarize tests ─────────────────────────────────────────────────────────

func TestManifestSummarize(t *testing.T) {
	m := &RelationshipManifest{
		Relationships: []ManifestRelationship{
			{Status: "accepted"},
			{Status: "accepted"},
			{Status: "rejected"},
			{Status: "pending"},
			{Status: "pending"},
		},
	}

	s := m.Summarize()
	if s.Total != 5 {
		t.Errorf("total: got %d", s.Total)
	}
	if s.Accepted != 2 {
		t.Errorf("accepted: got %d", s.Accepted)
	}
	if s.Rejected != 1 {
		t.Errorf("rejected: got %d", s.Rejected)
	}
	if s.Pending != 2 {
		t.Errorf("pending: got %d", s.Pending)
	}
	if s.Reviewed != 3 {
		t.Errorf("reviewed: got %d", s.Reviewed)
	}
}

// ─── AcceptAll / RejectAll tests ─────────────────────────────────────────────

func TestManifestAcceptAll(t *testing.T) {
	m := &RelationshipManifest{
		Relationships: []ManifestRelationship{
			{Status: "pending"},
			{Status: "rejected"},
		},
	}
	m.AcceptAll()
	for _, r := range m.Relationships {
		if r.Status != "accepted" {
			t.Errorf("expected accepted, got %s", r.Status)
		}
		if !r.Reviewed {
			t.Error("expected reviewed=true")
		}
	}
}

func TestManifestRejectAll(t *testing.T) {
	m := &RelationshipManifest{
		Relationships: []ManifestRelationship{
			{Status: "pending"},
			{Status: "accepted"},
		},
	}
	m.RejectAll()
	for _, r := range m.Relationships {
		if r.Status != "rejected" {
			t.Errorf("expected rejected, got %s", r.Status)
		}
	}
}

// ─── MergeUserEdits tests ────────────────────────────────────────────────────

func TestMergeUserEdits(t *testing.T) {
	discovered := &RelationshipManifest{
		Relationships: []ManifestRelationship{
			{Source: "a.md", Target: "b.md", Status: "pending"},
			{Source: "c.md", Target: "d.md", Status: "pending"},
			{Source: "e.md", Target: "f.md", Status: "pending"},
		},
	}

	accepted := &RelationshipManifest{
		Relationships: []ManifestRelationship{
			{Source: "a.md", Target: "b.md", Status: "accepted", Reviewed: true, UserNotes: "verified manually"},
			{Source: "c.md", Target: "d.md", Status: "rejected", Reviewed: true},
		},
	}

	discovered.MergeUserEdits(accepted)

	r0 := discovered.Relationships[0]
	if r0.Status != "accepted" || !r0.Reviewed || r0.UserNotes != "verified manually" {
		t.Errorf("first relationship not merged correctly: %+v", r0)
	}

	r1 := discovered.Relationships[1]
	if r1.Status != "rejected" || !r1.Reviewed {
		t.Errorf("second relationship not merged correctly: %+v", r1)
	}

	r2 := discovered.Relationships[2]
	if r2.Status != "pending" || r2.Reviewed {
		t.Errorf("third relationship should remain pending: %+v", r2)
	}
}

func TestMergeUserEdits_NilAccepted(t *testing.T) {
	m := &RelationshipManifest{
		Relationships: []ManifestRelationship{
			{Source: "a.md", Target: "b.md", Status: "pending"},
		},
	}
	m.MergeUserEdits(nil) // Should not panic.
	if m.Relationships[0].Status != "pending" {
		t.Error("should remain pending when accepted is nil")
	}
}

// ─── LoadAcceptedRelationships tests ─────────────────────────────────────────

func TestLoadAcceptedRelationships(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bmd-relationships.yaml")

	m := &RelationshipManifest{
		Version: "1.0",
		Relationships: []ManifestRelationship{
			{Source: "a.md", Target: "b.md", Type: "references", Confidence: 0.9, Status: "accepted"},
			{Source: "c.md", Target: "d.md", Type: "mentions", Confidence: 0.7, Status: "rejected"},
			{Source: "e.md", Target: "f.md", Type: "depends-on", Confidence: 0.8, Status: "accepted", UserNotes: "confirmed dep"},
		},
	}
	if err := SaveRelationshipManifest(m, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	edges, err := LoadAcceptedRelationships(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Only 2 accepted edges (rejected is excluded).
	if len(edges) != 2 {
		t.Fatalf("expected 2 accepted edges, got %d", len(edges))
	}

	// First edge: a.md -> b.md, references.
	if edges[0].Source != "a.md" || edges[0].Target != "b.md" {
		t.Errorf("edge 0: %s->%s", edges[0].Source, edges[0].Target)
	}
	if edges[0].Type != EdgeReferences {
		t.Errorf("edge 0 type: %s", edges[0].Type)
	}

	// Second edge: e.md -> f.md, depends-on with user notes.
	if edges[1].Source != "e.md" || edges[1].Target != "f.md" {
		t.Errorf("edge 1: %s->%s", edges[1].Source, edges[1].Target)
	}
	if edges[1].Type != EdgeDependsOn {
		t.Errorf("edge 1 type: %s", edges[1].Type)
	}
	if edges[1].Evidence != "confirmed dep" {
		t.Errorf("edge 1 evidence: %s", edges[1].Evidence)
	}
}

func TestLoadAcceptedRelationships_NoFile(t *testing.T) {
	edges, err := LoadAcceptedRelationships("/nonexistent/path.yaml")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if edges != nil {
		t.Error("expected nil edges for missing file")
	}
}

// ─── CLI integration tests ───────────────────────────────────────────────────

func TestCmdRelationshipsReview_AcceptAll(t *testing.T) {
	dir := t.TempDir()

	// Create a discovered manifest.
	e1, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "link")
	e2, _ := NewEdge("c.md", "d.md", EdgeMentions, 0.7, "mention")
	manifest := GenerateRelationshipManifest([]*Edge{e1, e2}, nil)
	discoveredPath := filepath.Join(dir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(manifest, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Run --accept-all.
	if err := CmdRelationshipsReview([]string{"--dir", dir, "--accept-all"}); err != nil {
		t.Fatalf("review --accept-all: %v", err)
	}

	// Verify the accepted manifest was created.
	acceptedPath := filepath.Join(dir, AcceptedManifestFile)
	accepted, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load accepted: %v", err)
	}
	if accepted == nil {
		t.Fatal("accepted manifest is nil")
	}

	for _, r := range accepted.Relationships {
		if r.Status != "accepted" {
			t.Errorf("expected accepted, got %s for %s->%s", r.Status, r.Source, r.Target)
		}
	}
}

func TestCmdRelationshipsReview_RejectAll(t *testing.T) {
	dir := t.TempDir()

	e1, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "link")
	manifest := GenerateRelationshipManifest([]*Edge{e1}, nil)
	if err := SaveRelationshipManifest(manifest, filepath.Join(dir, DiscoveredManifestFile)); err != nil {
		t.Fatal(err)
	}

	if err := CmdRelationshipsReview([]string{"--dir", dir, "--reject-all"}); err != nil {
		t.Fatalf("review --reject-all: %v", err)
	}

	accepted, _ := LoadRelationshipManifest(filepath.Join(dir, AcceptedManifestFile))
	if accepted == nil {
		t.Fatal("accepted is nil")
	}

	for _, r := range accepted.Relationships {
		if r.Status != "rejected" {
			t.Errorf("expected rejected, got %s", r.Status)
		}
	}
}

func TestCmdRelationshipsReview_ExportTo(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "exported.yaml")

	e1, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "link")
	manifest := GenerateRelationshipManifest([]*Edge{e1}, nil)
	if err := SaveRelationshipManifest(manifest, filepath.Join(dir, DiscoveredManifestFile)); err != nil {
		t.Fatal(err)
	}

	if err := CmdRelationshipsReview([]string{"--dir", dir, "--accept-all", "--export-to", exportPath}); err != nil {
		t.Fatalf("review --export-to: %v", err)
	}

	exported, err := LoadRelationshipManifest(exportPath)
	if err != nil {
		t.Fatalf("load exported: %v", err)
	}
	if exported == nil {
		t.Fatal("exported is nil")
	}
	if len(exported.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(exported.Relationships))
	}
}

func TestCmdRelationshipsReview_NoDiscovered(t *testing.T) {
	dir := t.TempDir()
	// Should not error, just print a message.
	if err := CmdRelationshipsReview([]string{"--dir", dir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdRelationshipsReview_MutuallyExclusive(t *testing.T) {
	err := CmdRelationshipsReview([]string{"--accept-all", "--reject-all"})
	if err == nil {
		t.Error("expected error for mutually exclusive flags")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention mutually exclusive: %v", err)
	}
}

func TestCmdRelationshipsReview_PreservesUserEdits(t *testing.T) {
	dir := t.TempDir()

	// Create discovered manifest with 2 relationships.
	e1, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "link")
	e2, _ := NewEdge("c.md", "d.md", EdgeMentions, 0.7, "mention")
	manifest := GenerateRelationshipManifest([]*Edge{e1, e2}, nil)
	if err := SaveRelationshipManifest(manifest, filepath.Join(dir, DiscoveredManifestFile)); err != nil {
		t.Fatal(err)
	}

	// Create accepted manifest where a->b is accepted with user notes.
	accepted := &RelationshipManifest{
		Version: "1.0",
		Relationships: []ManifestRelationship{
			{Source: "a.md", Target: "b.md", Status: "accepted", Reviewed: true, UserNotes: "manually verified"},
		},
	}
	if err := SaveRelationshipManifest(accepted, filepath.Join(dir, AcceptedManifestFile)); err != nil {
		t.Fatal(err)
	}

	// Run review without bulk action (should preserve previous edits).
	if err := CmdRelationshipsReview([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}

	result, _ := LoadRelationshipManifest(filepath.Join(dir, AcceptedManifestFile))
	if result == nil {
		t.Fatal("result is nil")
	}

	// Find the a->b relationship and verify user notes preserved.
	found := false
	for _, r := range result.Relationships {
		if r.Source == "a.md" && r.Target == "b.md" {
			found = true
			if r.Status != "accepted" {
				t.Errorf("expected accepted, got %s", r.Status)
			}
			if r.UserNotes != "manually verified" {
				t.Errorf("expected user notes preserved, got %q", r.UserNotes)
			}
		}
	}
	if !found {
		t.Error("a.md->b.md relationship not found in result")
	}
}

// ─── signalTypeFromEdge tests ────────────────────────────────────────────────

func TestSignalTypeFromEdge(t *testing.T) {
	tests := []struct {
		edgeType EdgeType
		expected string
	}{
		{EdgeReferences, "structural"},
		{EdgeMentions, "mention"},
		{EdgeCalls, "structural"},
		{EdgeDependsOn, "structural"},
		{EdgeImplements, "structural"},
		{EdgeType("unknown-type"), "unknown"},
	}

	for _, tc := range tests {
		e := &Edge{Type: tc.edgeType}
		got := signalTypeFromEdge(e)
		if got != tc.expected {
			t.Errorf("signalTypeFromEdge(%s): got %q, want %q", tc.edgeType, got, tc.expected)
		}
	}
}

// ─── Graph builder integration test ──────────────────────────────────────────

func TestLoadAcceptedRelationships_IntegrationWithGraph(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, AcceptedManifestFile)

	// Create manifest with accepted edges.
	m := &RelationshipManifest{
		Version: "1.0",
		Relationships: []ManifestRelationship{
			{Source: "svc/api.md", Target: "svc/db.md", Type: "depends-on", Confidence: 0.85, Status: "accepted"},
			{Source: "svc/api.md", Target: "svc/cache.md", Type: "mentions", Confidence: 0.6, Status: "accepted"},
		},
	}
	if err := SaveRelationshipManifest(m, path); err != nil {
		t.Fatal(err)
	}

	edges, err := LoadAcceptedRelationships(path)
	if err != nil {
		t.Fatal(err)
	}

	// Build a graph and add the edges.
	g := NewGraph()
	g.Nodes["svc/api.md"] = &Node{ID: "svc/api.md", Title: "API"}
	g.Nodes["svc/db.md"] = &Node{ID: "svc/db.md", Title: "DB"}
	g.Nodes["svc/cache.md"] = &Node{ID: "svc/cache.md", Title: "Cache"}

	for _, e := range edges {
		if err := g.AddEdge(e); err != nil {
			t.Errorf("AddEdge failed: %v", err)
		}
	}

	if g.EdgeCount() != 2 {
		t.Errorf("expected 2 edges in graph, got %d", g.EdgeCount())
	}
}
