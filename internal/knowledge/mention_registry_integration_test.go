package knowledge

import (
	"testing"
)

// ─── BuildFromMentions ────────────────────────────────────────────────────────

func TestBuildFromMentionsBasic(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "gateway", Name: "Gateway"})

	mentions := []Mention{
		{FromFile: "gateway", ToComponent: "auth", Confidence: 0.75, EvidenceCount: 2, ExampleEvidence: "calls auth"},
	}
	reg.BuildFromMentions(mentions)

	rels := reg.FindRelationships("gateway")
	if len(rels) == 0 {
		t.Fatal("expected at least one relationship after building from mentions")
	}

	found := false
	for _, rel := range rels {
		if rel.ToComponent == "auth" {
			found = true
			if rel.AggregatedConfidence < 0.74 {
				t.Errorf("AggregatedConfidence = %.2f, want >= 0.74", rel.AggregatedConfidence)
			}
		}
	}
	if !found {
		t.Error("expected relationship from gateway to auth")
	}
}

func TestBuildFromMentionsSignalType(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "gateway", Name: "Gateway"})

	mentions := []Mention{
		{FromFile: "gateway", ToComponent: "auth", Confidence: 0.7, ExampleEvidence: "uses auth"},
	}
	reg.BuildFromMentions(mentions)

	rels := reg.FindRelationships("gateway")
	if len(rels) == 0 {
		t.Fatal("expected relationship")
	}

	mentionSignalFound := false
	for _, rel := range rels {
		for _, sig := range rel.Signals {
			if sig.SourceType == SignalMention {
				mentionSignalFound = true
			}
		}
	}
	if !mentionSignalFound {
		t.Error("expected SignalMention signal in relationship")
	}
}

func TestBuildFromMentionsDeduplication(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "gateway", Name: "Gateway"})

	mentions := []Mention{
		{FromFile: "gateway", ToComponent: "auth", Confidence: 0.75, ExampleEvidence: "calls auth"},
		{FromFile: "gateway", ToComponent: "auth", Confidence: 0.75, ExampleEvidence: "calls auth"}, // duplicate
	}
	reg.BuildFromMentions(mentions)

	rels := reg.FindRelationships("gateway")
	for _, rel := range rels {
		if rel.ToComponent == "auth" {
			// Signals should be deduplicated: same SourceType + Evidence = 1 signal.
			mentionCount := 0
			for _, sig := range rel.Signals {
				if sig.SourceType == SignalMention {
					mentionCount++
				}
			}
			if mentionCount > 1 {
				t.Errorf("expected 1 mention signal (deduped), got %d", mentionCount)
			}
		}
	}
}

func TestBuildFromMentionsMultipleSources(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "gateway", Name: "Gateway"})

	// Add a link signal first.
	_ = reg.AddSignal("gateway", "auth", Signal{
		SourceType: SignalLink,
		Confidence: 1.0,
		Evidence:   "[auth](auth.md)",
		Weight:     1.0,
	})

	// Then add a mention signal.
	reg.BuildFromMentions([]Mention{
		{FromFile: "gateway", ToComponent: "auth", Confidence: 0.75, ExampleEvidence: "calls auth"},
	})

	rels := reg.FindRelationships("gateway")
	for _, rel := range rels {
		if rel.ToComponent == "auth" {
			hasLink := false
			hasMention := false
			for _, sig := range rel.Signals {
				if sig.SourceType == SignalLink {
					hasLink = true
				}
				if sig.SourceType == SignalMention {
					hasMention = true
				}
			}
			if !hasLink {
				t.Error("expected link signal to be preserved")
			}
			if !hasMention {
				t.Error("expected mention signal to be added")
			}
			// AggregatedConfidence should be max = 1.0 (from link).
			if rel.AggregatedConfidence < 1.0 {
				t.Errorf("AggregatedConfidence = %.2f, want 1.0 (link signal wins)", rel.AggregatedConfidence)
			}
		}
	}
}

func TestBuildFromMentionsEmpty(t *testing.T) {
	reg := NewComponentRegistry()
	reg.BuildFromMentions(nil)
	if reg.RelationshipCount() != 0 {
		t.Errorf("expected 0 relationships, got %d", reg.RelationshipCount())
	}

	reg.BuildFromMentions([]Mention{})
	if reg.RelationshipCount() != 0 {
		t.Errorf("expected 0 relationships, got %d", reg.RelationshipCount())
	}
}

// ─── GetMentionsFor ───────────────────────────────────────────────────────────

func TestGetMentionsForBasic(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "gateway", Name: "Gateway"})
	_ = reg.AddComponent(&RegistryComponent{ID: "payment", Name: "Payment"})

	reg.BuildFromMentions([]Mention{
		{FromFile: "gateway", ToComponent: "auth", Confidence: 0.75},
		{FromFile: "payment", ToComponent: "auth", Confidence: 0.7},
	})

	rels := reg.GetMentionsFor("auth")
	if len(rels) < 2 {
		t.Errorf("expected >= 2 relationships mentioning auth, got %d", len(rels))
	}
}

func TestGetMentionsForExcludesLinkOnly(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "gateway", Name: "Gateway"})

	// Only a link signal — GetMentionsFor should return empty.
	_ = reg.AddSignal("gateway", "auth", Signal{
		SourceType: SignalLink,
		Confidence: 1.0,
		Evidence:   "[auth](auth.md)",
		Weight:     1.0,
	})

	rels := reg.GetMentionsFor("auth")
	if len(rels) != 0 {
		t.Errorf("GetMentionsFor should only return mention signals, got %d results", len(rels))
	}
}

func TestGetMentionsForSortsByConfidence(t *testing.T) {
	reg := NewComponentRegistry()
	_ = reg.AddComponent(&RegistryComponent{ID: "auth", Name: "Auth"})
	_ = reg.AddComponent(&RegistryComponent{ID: "a", Name: "A"})
	_ = reg.AddComponent(&RegistryComponent{ID: "b", Name: "B"})

	reg.BuildFromMentions([]Mention{
		{FromFile: "a", ToComponent: "auth", Confidence: 0.65},
		{FromFile: "b", ToComponent: "auth", Confidence: 0.75},
	})

	rels := reg.GetMentionsFor("auth")
	if len(rels) < 2 {
		t.Fatalf("expected 2 relationships, got %d", len(rels))
	}

	if rels[0].AggregatedConfidence < rels[1].AggregatedConfidence {
		t.Error("results should be sorted by confidence descending")
	}
}

func TestGetMentionsForEmpty(t *testing.T) {
	reg := NewComponentRegistry()
	rels := reg.GetMentionsFor("nonexistent")
	if len(rels) != 0 {
		t.Errorf("expected 0 results for nonexistent component, got %d", len(rels))
	}
}

// ─── InitFromGraph integration ────────────────────────────────────────────────

func TestInitFromGraphIncludesMentions(t *testing.T) {
	// Build a minimal graph with two nodes and a link edge.
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "services/auth.md", Title: "Auth Service", Type: "document"})
	_ = g.AddNode(&Node{ID: "services/gateway.md", Title: "API Gateway", Type: "document"})

	edge, _ := NewEdge("services/gateway.md", "services/auth.md", EdgeReferences, 1.0, "[auth](auth.md)")
	_ = g.AddEdge(edge)

	// Create documents — gateway mentions auth in its text.
	docs := []Document{
		{
			ID:      "services/auth.md",
			RelPath: "services/auth.md",
			Title:   "Auth Component",
			Content: "# Auth Component\n\nHandles authentication and token validation.",
		},
		{
			ID:      "services/gateway.md",
			RelPath: "services/gateway.md",
			Title:   "API Gateway",
			Content: "# API Gateway\n\nThis service calls auth to verify user tokens before routing requests.",
		},
	}

	reg := NewComponentRegistry()
	reg.InitFromGraph(g, docs)

	// Registry should have components.
	if reg.ComponentCount() == 0 {
		t.Fatal("expected components after InitFromGraph")
	}

	// Registry should have relationships.
	if reg.RelationshipCount() == 0 {
		t.Fatal("expected relationships after InitFromGraph")
	}
}

func TestInitFromGraphBackwardCompatible(t *testing.T) {
	// InitFromGraph with no docs should still work (backward-compatible).
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "auth.md", Title: "Auth", Type: "document"})
	_ = g.AddNode(&Node{ID: "gateway.md", Title: "Gateway", Type: "document"})

	edge, _ := NewEdge("gateway.md", "auth.md", EdgeReferences, 1.0, "link")
	_ = g.AddEdge(edge)

	reg := NewComponentRegistry()
	reg.InitFromGraph(g, nil) // No docs — should not panic.

	if reg.ComponentCount() == 0 {
		t.Fatal("expected components after InitFromGraph with no docs")
	}
	if reg.RelationshipCount() == 0 {
		t.Fatal("expected relationships after InitFromGraph with no docs")
	}
}
