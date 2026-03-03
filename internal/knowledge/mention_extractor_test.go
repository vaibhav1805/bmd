package knowledge

import (
	"strings"
	"testing"
	"time"
)

// makeMentionTestDoc builds a simple Document for use in mention tests.
func makeMentionTestDoc(id, content string) Document {
	return Document{
		ID:      id,
		RelPath: id,
		Content: content,
		Title:   id,
	}
}

// makeComponent builds a Component for use in tests.
func makeComponent(id, name, file string) Component {
	return Component{
		ID:         id,
		Name:       name,
		File:       file,
		Confidence: 0.9,
	}
}

// ─── ExtractMentionsFromDocument ─────────────────────────────────────────────

func TestExtractMentionsFromDocumentBasic(t *testing.T) {
	doc := makeMentionTestDoc("services/gateway.md",
		"This service calls auth to verify tokens and depends on payment for billing.")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
		makeComponent("payment", "Payment Service", "services/payment.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)

	if len(mentions) == 0 {
		t.Fatal("expected at least one mention, got none")
	}

	byComp := make(map[string]Mention)
	for _, m := range mentions {
		byComp[m.ToComponent] = m
	}

	if _, ok := byComp["auth"]; !ok {
		t.Error("expected auth to be mentioned")
	}
	if _, ok := byComp["payment"]; !ok {
		t.Error("expected payment to be mentioned")
	}
}

func TestExtractMentionsFromDocumentConfidenceRange(t *testing.T) {
	doc := makeMentionTestDoc("services/gateway.md",
		"calls auth to verify the user identity")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) == 0 {
		t.Fatal("expected mention, got none")
	}

	m := mentions[0]
	if m.Confidence < 0.6 || m.Confidence > 0.8 {
		t.Errorf("confidence = %.2f, want in [0.6, 0.8]", m.Confidence)
	}
}

func TestExtractMentionsFromDocumentExcludesSelf(t *testing.T) {
	// The document belongs to the "auth" component; it should NOT mention itself.
	doc := makeMentionTestDoc("services/auth.md",
		"The auth service handles authentication. calls auth to process tokens.")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
		makeComponent("payment", "Payment Service", "services/payment.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	for _, m := range mentions {
		if m.ToComponent == "auth" {
			t.Errorf("auth should not mention itself, but got mention with confidence %.2f", m.Confidence)
		}
	}
}

func TestExtractMentionsFromDocumentEvidenceTracking(t *testing.T) {
	doc := makeMentionTestDoc("services/gateway.md",
		"calls auth to verify\nauth-service handles tokens\ndepends on auth for all requests")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) == 0 {
		t.Fatal("expected mention")
	}

	m := mentions[0]
	if m.EvidenceCount < 1 {
		t.Errorf("EvidenceCount = %d, want >= 1", m.EvidenceCount)
	}
	if m.ExampleEvidence == "" {
		t.Error("ExampleEvidence should not be empty")
	}
}

func TestExtractMentionsFromDocumentMultipleMatches(t *testing.T) {
	// Multiple lines mentioning the same component should be counted.
	doc := makeMentionTestDoc("services/gateway.md",
		"calls auth\nuses auth-service\ndepends on auth for processing")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) != 1 {
		t.Fatalf("expected 1 deduped mention, got %d", len(mentions))
	}

	m := mentions[0]
	if m.EvidenceCount < 2 {
		t.Errorf("EvidenceCount = %d, want >= 2 (multiple matching lines)", m.EvidenceCount)
	}
}

func TestExtractMentionsFromDocumentHighestConfidenceRetained(t *testing.T) {
	// "calls auth" (0.75) and "uses auth" (0.7) — should keep 0.75.
	doc := makeMentionTestDoc("services/gateway.md",
		"uses auth for validation\ncalls auth to verify identity")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) == 0 {
		t.Fatal("expected mention")
	}

	m := mentions[0]
	if m.Confidence < 0.74 {
		t.Errorf("confidence = %.2f, want >= 0.74 (calls pattern should win)", m.Confidence)
	}
}

func TestExtractMentionsFromDocumentEmptyDoc(t *testing.T) {
	emptyDoc := Document{ID: "empty.md"}
	components := []Component{makeComponent("auth", "Auth", "auth.md")}

	mentions := ExtractMentionsFromDocument(emptyDoc, components)
	if len(mentions) != 0 {
		t.Errorf("expected no mentions for empty doc, got %d", len(mentions))
	}
}

func TestExtractMentionsFromDocumentNoComponents(t *testing.T) {
	doc := makeMentionTestDoc("gateway.md", "calls auth to verify")
	mentions := ExtractMentionsFromDocument(doc, nil)
	if len(mentions) != 0 {
		t.Errorf("expected no mentions with no components, got %d", len(mentions))
	}
}

func TestExtractMentionsFromDocumentNoMatch(t *testing.T) {
	doc := makeMentionTestDoc("gateway.md", "This document does not mention any known services")
	components := []Component{makeComponent("auth", "Auth Service", "auth.md")}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) != 0 {
		t.Errorf("expected no mentions, got %d", len(mentions))
	}
}

func TestExtractMentionsFromDocumentAPIPattern(t *testing.T) {
	doc := makeMentionTestDoc("services/gateway.md",
		"Requests are forwarded to the auth API for token validation.")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) == 0 {
		t.Fatal("expected mention via X API pattern")
	}
}

func TestExtractMentionsFromDocumentServiceSuffixPattern(t *testing.T) {
	doc := makeMentionTestDoc("services/gateway.md",
		"Requests are forwarded to auth-service for validation.")

	components := []Component{
		makeComponent("auth", "Auth Service", "services/auth.md"),
	}

	mentions := ExtractMentionsFromDocument(doc, components)
	if len(mentions) == 0 {
		t.Fatal("expected mention via service-suffix pattern")
	}
}

// ─── ExtractMentionsFromDocuments ────────────────────────────────────────────

func TestExtractMentionsFromDocumentsBatch(t *testing.T) {
	docs := []Document{
		makeMentionTestDoc("a.md", "calls auth for verification"),
		makeMentionTestDoc("b.md", "depends on payment for billing"),
		makeMentionTestDoc("c.md", "no relevant mentions here"),
	}
	components := []Component{
		makeComponent("auth", "Auth", "auth.md"),
		makeComponent("payment", "Payment", "payment.md"),
	}

	mentions := ExtractMentionsFromDocuments(docs, components)
	if len(mentions) < 2 {
		t.Errorf("expected at least 2 mentions, got %d", len(mentions))
	}
}

func TestExtractMentionsFromDocumentsEmpty(t *testing.T) {
	mentions := ExtractMentionsFromDocuments(nil, nil)
	if len(mentions) != 0 {
		t.Errorf("expected no mentions, got %d", len(mentions))
	}

	mentions = ExtractMentionsFromDocuments([]Document{makeMentionTestDoc("a.md", "hello")}, nil)
	if len(mentions) != 0 {
		t.Errorf("expected no mentions with no components, got %d", len(mentions))
	}
}

// ─── buildComponentLookup ────────────────────────────────────────────────────

func TestBuildComponentLookupBasic(t *testing.T) {
	components := []Component{
		makeComponent("auth", "Auth Service", "auth.md"),
		makeComponent("payment", "Payment", "payment.md"),
	}
	m := buildComponentLookup(components, "other.md")
	if _, ok := m["auth"]; !ok {
		t.Error("expected auth in lookup")
	}
	if _, ok := m["payment"]; !ok {
		t.Error("expected payment in lookup")
	}
}

func TestBuildComponentLookupExcludesSelf(t *testing.T) {
	components := []Component{
		makeComponent("auth", "Auth Service", "auth.md"),
		makeComponent("payment", "Payment", "payment.md"),
	}
	// selfDocID = "auth.md" — auth should be excluded
	m := buildComponentLookup(components, "auth.md")
	if _, ok := m["auth"]; ok {
		t.Error("auth should be excluded (self-document)")
	}
	if _, ok := m["payment"]; !ok {
		t.Error("payment should remain in lookup")
	}
}

func TestBuildComponentLookupNormalizesNames(t *testing.T) {
	components := []Component{
		makeComponent("api-gateway", "API Gateway", "gateway.md"),
	}
	m := buildComponentLookup(components, "other.md")

	// Should have both the id and a normalized multi-word name.
	if _, ok := m["api-gateway"]; !ok {
		t.Error("expected api-gateway (from ID)")
	}
}

// ─── normalizeComponentName ───────────────────────────────────────────────────

func TestNormalizeComponentName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Auth Service", "auth-service"},
		{"API Gateway", "api-gateway"},
		{"UserDB", "userdb"},
		{"", ""},
		{"Auth", "auth"},
	}
	for _, tc := range tests {
		got := normalizeComponentName(tc.input)
		if got != tc.want {
			t.Errorf("normalizeComponentName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── Performance test ─────────────────────────────────────────────────────────

func TestExtractMentionsFromDocumentPerformance(t *testing.T) {
	// Build a large document with many lines.
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("This service calls auth to verify tokens. ")
		sb.WriteString("It also depends on payment for billing.\n")
	}

	doc := makeMentionTestDoc("perf.md", sb.String())
	components := []Component{
		makeComponent("auth", "Auth Service", "auth.md"),
		makeComponent("payment", "Payment Service", "payment.md"),
	}

	start := time.Now()
	mentions := ExtractMentionsFromDocument(doc, components)
	elapsed := time.Since(start)

	if len(mentions) == 0 {
		t.Error("expected mentions from large document")
	}
	if elapsed.Milliseconds() > 500 {
		t.Errorf("extraction took %dms, want < 500ms", elapsed.Milliseconds())
	}
}

// ─── truncateLine ─────────────────────────────────────────────────────────────

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		input    string
		maxChars int
		want     string
	}{
		{"hello world", 20, "hello world"},
		{"hello world", 5, "hello..."},
		{"", 10, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}
	for _, tc := range tests {
		got := truncateLine(tc.input, tc.maxChars)
		if got != tc.want {
			t.Errorf("truncateLine(%q, %d) = %q, want %q", tc.input, tc.maxChars, got, tc.want)
		}
	}
}
