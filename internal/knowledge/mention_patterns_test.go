package knowledge

import (
	"testing"
)

// ─── BuiltInPatterns ─────────────────────────────────────────────────────────

func TestBuiltInPatternsNotEmpty(t *testing.T) {
	p := BuiltInPatterns()
	if len(p.ServicePatterns) == 0 {
		t.Error("ServicePatterns should not be empty")
	}
	if len(p.ApiPatterns) == 0 {
		t.Error("ApiPatterns should not be empty")
	}
	if len(p.ConfigPatterns) == 0 {
		t.Error("ConfigPatterns should not be empty")
	}
}

func TestAllPatternsReturnsAll(t *testing.T) {
	p := BuiltInPatterns()
	all := p.AllPatterns()
	want := len(p.ServicePatterns) + len(p.ApiPatterns) + len(p.ConfigPatterns)
	if len(all) != want {
		t.Errorf("AllPatterns() = %d, want %d", len(all), want)
	}
}

// ─── IsComponentMention ───────────────────────────────────────────────────────

func TestIsComponentMentionCallsPattern(t *testing.T) {
	tests := []struct {
		text      string
		component string
		wantMatch bool
		minConf   float64
	}{
		{"calls auth to verify", "auth", true, 0.7},
		{"calls the auth service", "auth", true, 0.7},
		{"depends on auth", "auth", true, 0.7},
		{"depends on the auth service", "auth", true, 0.7},
		{"uses auth", "auth", true, 0.6},
		{"integrates with auth", "auth", true, 0.7},
		{"requires auth", "auth", true, 0.6},
		{"sends a request to auth", "auth", true, 0.7},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			got, conf := IsComponentMention(tc.text, tc.component)
			if got != tc.wantMatch {
				t.Errorf("IsComponentMention(%q, %q) match = %v, want %v", tc.text, tc.component, got, tc.wantMatch)
			}
			if got && conf < tc.minConf {
				t.Errorf("confidence = %.2f, want >= %.2f", conf, tc.minConf)
			}
		})
	}
}

func TestIsComponentMentionAPIPatterns(t *testing.T) {
	tests := []struct {
		text      string
		component string
		wantMatch bool
	}{
		{"auth API", "auth", true},
		{"the auth API is available", "auth", true},
		{"api/auth/v1", "auth", true},
		{"auth.internal.com", "auth", true},
		{"auth.svc.com is the endpoint", "auth", true},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			got, _ := IsComponentMention(tc.text, tc.component)
			if got != tc.wantMatch {
				t.Errorf("IsComponentMention(%q, %q) = %v, want %v", tc.text, tc.component, got, tc.wantMatch)
			}
		})
	}
}

func TestIsComponentMentionConfigPatterns(t *testing.T) {
	tests := []struct {
		text      string
		component string
		wantMatch bool
		minConf   float64
	}{
		{"auth-service handles tokens", "auth", true, 0.7},
		{"connect to auth:8080", "auth", true, 0.6},
		{"auth:443 endpoint", "auth", true, 0.6},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			got, conf := IsComponentMention(tc.text, tc.component)
			if got != tc.wantMatch {
				t.Errorf("IsComponentMention(%q, %q) match = %v, want %v", tc.text, tc.component, got, tc.wantMatch)
			}
			if got && conf < tc.minConf {
				t.Errorf("confidence = %.2f, want >= %.2f", conf, tc.minConf)
			}
		})
	}
}

func TestIsComponentMentionCaseInsensitive(t *testing.T) {
	tests := []struct {
		text      string
		component string
	}{
		{"CALLS AUTH to verify", "auth"},
		{"Calls Auth-Service", "auth"},
		{"depends on AUTH", "auth"},
		{"Uses Auth API", "auth"},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			got, _ := IsComponentMention(tc.text, tc.component)
			if !got {
				t.Errorf("IsComponentMention(%q, %q) should match (case-insensitive)", tc.text, tc.component)
			}
		})
	}
}

func TestIsComponentMentionFalsePositiveAvoidance(t *testing.T) {
	// "auth" should NOT match "authentication" as a standalone word mention
	// when the text only mentions "authentication" but not "auth" as a distinct word.
	// Note: this tests that "authentication" is not captured as "auth".
	// However we only check that captured == name in isExactMatch, so if pattern
	// captures "authentication" it won't equal "auth".
	tests := []struct {
		text      string
		component string
		wantMatch bool
	}{
		// Simple prose that doesn't contain the component name as a pattern match
		{"the system does authentication", "auth", false},
		{"it handles authorization", "auth", false},
		// Empty inputs
		{"", "auth", false},
		{"calls auth", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.text+"_"+tc.component, func(t *testing.T) {
			got, _ := IsComponentMention(tc.text, tc.component)
			if got != tc.wantMatch {
				t.Errorf("IsComponentMention(%q, %q) = %v, want %v", tc.text, tc.component, got, tc.wantMatch)
			}
		})
	}
}

// ─── ExtractMentionsFromLine ──────────────────────────────────────────────────

func TestExtractMentionsFromLineBasic(t *testing.T) {
	known := map[string]string{
		"auth":         "auth",
		"api-gateway":  "api-gateway",
		"user-service": "user",
	}

	line := "This service calls auth to verify tokens"
	mentions := ExtractMentionsFromLine(line, known)

	if len(mentions) == 0 {
		t.Fatal("expected at least one mention, got none")
	}

	found := false
	for _, m := range mentions {
		if m.ComponentName == "auth" {
			found = true
			if m.Confidence < 0.7 {
				t.Errorf("auth mention confidence = %.2f, want >= 0.7", m.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected auth to be mentioned")
	}
}

func TestExtractMentionsFromLineMultipleComponents(t *testing.T) {
	known := map[string]string{
		"auth":    "auth",
		"payment": "payment",
	}

	line := "calls auth and depends on payment for processing"
	mentions := ExtractMentionsFromLine(line, known)

	foundAuth := false
	foundPayment := false
	for _, m := range mentions {
		if m.ComponentName == "auth" {
			foundAuth = true
		}
		if m.ComponentName == "payment" {
			foundPayment = true
		}
	}

	if !foundAuth {
		t.Error("expected auth to be mentioned")
	}
	if !foundPayment {
		t.Error("expected payment to be mentioned")
	}
}

func TestExtractMentionsFromLineDeduplication(t *testing.T) {
	known := map[string]string{
		"auth": "auth",
	}

	// "calls auth" and "auth-service" both match auth — should only get one result
	line := "calls auth and uses auth-service"
	mentions := ExtractMentionsFromLine(line, known)

	authCount := 0
	for _, m := range mentions {
		if m.ComponentName == "auth" {
			authCount++
		}
	}
	if authCount != 1 {
		t.Errorf("expected 1 auth mention (deduplicated), got %d", authCount)
	}
}

func TestExtractMentionsFromLineHighestConfidenceWins(t *testing.T) {
	known := map[string]string{
		"auth": "auth",
	}

	// Both "calls auth" (0.75) and "uses auth" (0.7) match. Should keep 0.75.
	line := "calls auth and uses auth for everything"
	mentions := ExtractMentionsFromLine(line, known)

	for _, m := range mentions {
		if m.ComponentName == "auth" {
			if m.Confidence < 0.74 {
				t.Errorf("expected highest confidence (>=0.74), got %.2f", m.Confidence)
			}
			return
		}
	}
	t.Error("expected auth mention")
}

func TestExtractMentionsFromLineEmpty(t *testing.T) {
	tests := []struct {
		line  string
		known map[string]string
	}{
		{"", map[string]string{"auth": "auth"}},
		{"calls auth", nil},
		{"calls auth", map[string]string{}},
	}
	for _, tc := range tests {
		mentions := ExtractMentionsFromLine(tc.line, tc.known)
		if len(mentions) != 0 {
			t.Errorf("expected no mentions for empty input, got %d", len(mentions))
		}
	}
}

func TestExtractMentionsFromLineNoMatch(t *testing.T) {
	known := map[string]string{
		"auth": "auth",
	}
	line := "The system processes requests efficiently"
	mentions := ExtractMentionsFromLine(line, known)
	if len(mentions) != 0 {
		t.Errorf("expected no mentions, got %d", len(mentions))
	}
}

// ─── isExactMatch ─────────────────────────────────────────────────────────────

func TestIsExactMatchDirect(t *testing.T) {
	if !isExactMatch("auth", "auth") {
		t.Error("exact same name should match")
	}
}

func TestIsExactMatchWithSuffix(t *testing.T) {
	tests := []struct {
		captured string
		name     string
		want     bool
	}{
		{"auth-service", "auth", true},
		{"auth-api", "auth", true},
		{"auth-server", "auth", true},
		{"auth-backend", "auth", true},
		{"auth-svc", "auth", true},
		// Name has suffix, captured doesn't
		{"auth", "auth-service", true},
		// Different names
		{"payment", "auth", false},
		{"authentication", "auth", false},
		// Empty
		{"", "auth", false},
		{"auth", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.captured+"_"+tc.name, func(t *testing.T) {
			got := isExactMatch(tc.captured, tc.name)
			if got != tc.want {
				t.Errorf("isExactMatch(%q, %q) = %v, want %v", tc.captured, tc.name, got, tc.want)
			}
		})
	}
}

// ─── Confidence constant checks ───────────────────────────────────────────────

func TestConfidenceConstants(t *testing.T) {
	// Verify the confidence constants are in the expected ranges per plan spec.
	constants := map[string]struct {
		val float64
		min float64
		max float64
	}{
		"ConfidenceMentionCalls":      {ConfidenceMentionCalls, 0.7, 0.8},
		"ConfidenceMentionDependsOn":  {ConfidenceMentionDependsOn, 0.7, 0.8},
		"ConfidenceMentionServiceSuffix": {ConfidenceMentionServiceSuffix, 0.7, 0.8},
		"ConfidenceMentionUses":       {ConfidenceMentionUses, 0.65, 0.75},
		"ConfidenceMentionAPI":        {ConfidenceMentionAPI, 0.65, 0.75},
		"ConfidenceMentionPortColon":  {ConfidenceMentionPortColon, 0.6, 0.7},
		"ConfidenceMentionAPIPath":    {ConfidenceMentionAPIPath, 0.6, 0.7},
		"ConfidenceMentionHostname":   {ConfidenceMentionHostname, 0.6, 0.7},
		"ConfidenceMentionHTTP":       {ConfidenceMentionHTTP, 0.55, 0.65},
	}
	for name, c := range constants {
		if c.val < c.min || c.val > c.max {
			t.Errorf("%s = %.2f, expected in [%.2f, %.2f]", name, c.val, c.min, c.max)
		}
	}
}
