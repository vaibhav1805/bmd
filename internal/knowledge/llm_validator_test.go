package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateRelationship_ExecutableNotFound tests that a missing validator
// binary returns ErrLLMValidatorNotFound.
func TestValidateRelationship_ExecutableNotFound(t *testing.T) {
	cfg := LLMValidatorConfig{
		ExecutablePath: "/nonexistent/path/to/validator",
		Model:          "test-model",
	}
	rel := &ManifestRelationship{
		Source: "service-a.md",
		Target: "service-b.md",
		Type:   "depends-on",
		Signals: []ManifestSignal{
			{Type: "mention", Value: 0.6, Evidence: "imports service-b"},
		},
	}

	_, err := ValidateRelationship(cfg, rel)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

// TestValidateRelationship_ValidResponse tests that a valid JSON response is
// correctly parsed.
func TestValidateRelationship_ValidResponse(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake shell script that returns a valid response.
	scriptPath := filepath.Join(tmpDir, "validator")
	scriptContent := `#!/bin/bash
echo '{"valid": true, "confidence": 0.9, "reasoning": "relationship is clearly documented"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := LLMValidatorConfig{
		ExecutablePath: scriptPath,
		Model:          "test-model",
	}
	rel := &ManifestRelationship{
		Source: "service-a.md",
		Target: "service-b.md",
		Type:   "depends-on",
		Signals: []ManifestSignal{
			{Type: "mention", Value: 0.6, Evidence: "imports service-b"},
		},
	}

	result, err := ValidateRelationship(cfg, rel)
	if err != nil {
		t.Fatalf("ValidateRelationship failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected Valid=true, got false")
	}
	if result.Confidence != 0.9 {
		t.Errorf("expected Confidence=0.9, got %v", result.Confidence)
	}
	if result.Reasoning != "relationship is clearly documented" {
		t.Errorf("expected specific reasoning, got: %q", result.Reasoning)
	}
}

// TestValidateRelationship_InvalidJSON tests graceful handling of invalid JSON.
func TestValidateRelationship_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake script that returns garbage.
	scriptPath := filepath.Join(tmpDir, "validator")
	scriptContent := `#!/bin/bash
echo "This is not JSON at all, just garbage output"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := LLMValidatorConfig{
		ExecutablePath: scriptPath,
		Model:          "test-model",
	}
	rel := &ManifestRelationship{
		Source: "service-a.md",
		Target: "service-b.md",
		Type:   "depends-on",
	}

	_, err := ValidateRelationship(cfg, rel)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	// Should NOT be ErrLLMValidatorNotFound.
	if strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected parse error, not 'not found': %v", err)
	}
}

// TestValidateRelationship_NonZeroExit tests graceful degradation on non-zero exit.
func TestValidateRelationship_NonZeroExit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake script that exits with status 1.
	scriptPath := filepath.Join(tmpDir, "validator")
	scriptContent := `#!/bin/bash
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := LLMValidatorConfig{
		ExecutablePath: scriptPath,
		Model:          "test-model",
	}
	rel := &ManifestRelationship{
		Source: "service-a.md",
		Target: "service-b.md",
		Type:   "depends-on",
	}

	result, err := ValidateRelationship(cfg, rel)
	// Graceful degradation: non-zero exit returns empty result, nil error.
	if err != nil {
		t.Fatalf("expected graceful degradation (nil error), got: %v", err)
	}
	if result.Valid || result.Confidence != 0 || result.Reasoning != "" {
		t.Errorf("expected empty ValidationResult, got: %+v", result)
	}
}

// TestBuildValidationPrompt_ContainsEvidence tests that the prompt includes
// signal evidence.
func TestBuildValidationPrompt_ContainsEvidence(t *testing.T) {
	rel := &ManifestRelationship{
		Source: "auth.md",
		Target: "db.md",
		Type:   "depends-on",
		Signals: []ManifestSignal{
			{Type: "mention", Value: 0.65, Evidence: "auth service queries user database"},
			{Type: "structural", Value: 0.75, Evidence: "database.NewConnection() called in auth init"},
		},
	}

	prompt := buildValidationPrompt(rel)

	// Verify key elements are present.
	checks := []string{
		"auth.md",
		"db.md",
		"depends-on",
		"auth service queries user database",
		"database.NewConnection",
		"mention",
		"structural",
	}
	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("prompt missing expected content: %q\nPrompt:\n%s", check, prompt)
		}
	}

	// Verify JSON structure is requested.
	if !strings.Contains(prompt, "JSON") {
		t.Error("prompt should request JSON format")
	}
}

// TestParseValidationResponse_WithProseWrapper tests that JSON extraction
// works when response is wrapped in prose.
func TestParseValidationResponse_WithProseWrapper(t *testing.T) {
	raw := []byte(`Some prose explaining the result. Here's the JSON:
{"valid": true, "confidence": 0.85, "reasoning": "test"}
And maybe more prose here.`)

	result, err := parseValidationResponse(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected Valid=true, got false")
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected Confidence=0.85, got %v", result.Confidence)
	}
	if result.Reasoning != "test" {
		t.Errorf("expected Reasoning='test', got %q", result.Reasoning)
	}
}

// TestParseValidationResponse_NoJSON tests error handling when no JSON is found.
func TestParseValidationResponse_NoJSON(t *testing.T) {
	raw := []byte("This response has no JSON object in it.")

	_, err := parseValidationResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing JSON, got nil")
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("expected JSON-related error, got: %v", err)
	}
}

// BenchmarkValidateRelationship_Subprocess measures overhead of subprocess invocation.
func BenchmarkValidateRelationship_Subprocess(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a minimal fake validator.
	scriptPath := filepath.Join(tmpDir, "validator")
	scriptContent := `#!/bin/bash
echo '{"valid":true,"confidence":0.8,"reasoning":"test"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		b.Fatalf("write script: %v", err)
	}

	cfg := LLMValidatorConfig{
		ExecutablePath: scriptPath,
		Model:          "test-model",
	}
	rel := &ManifestRelationship{
		Source: "a.md",
		Target: "b.md",
		Type:   "depends-on",
		Signals: []ManifestSignal{
			{Type: "mention", Value: 0.6, Evidence: "test signal"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateRelationship(cfg, rel)
	}
}

// TestBuildValidationPrompt_EmptySignals tests prompt generation with no signals.
func TestBuildValidationPrompt_EmptySignals(t *testing.T) {
	rel := &ManifestRelationship{
		Source: "a.md",
		Target: "b.md",
		Type:   "depends-on",
		// No signals
	}

	prompt := buildValidationPrompt(rel)

	if !strings.Contains(prompt, "a.md") || !strings.Contains(prompt, "b.md") {
		t.Error("prompt should contain source and target even with no signals")
	}
	if !strings.Contains(prompt, "no signals provided") {
		t.Error("prompt should indicate no signals when signals list is empty")
	}
}
