package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestParseReviewArgs_LLMValidateFlag tests that LLM validation flags are parsed correctly.
func TestParseReviewArgs_LLMValidateFlag(t *testing.T) {
	args := []string{
		"--llm-validate",
		"--llm-validate-bin", "/usr/local/bin/llm",
		"--llm-model", "claude-opus-4-6",
		"--auto-accept-threshold", "0.8",
		"--auto-reject-threshold", "0.3",
	}

	a, err := ParseReviewArgs(args)
	if err != nil {
		t.Fatalf("ParseReviewArgs failed: %v", err)
	}

	if !a.LLMValidate {
		t.Error("expected LLMValidate=true")
	}
	if a.LLMValidateBin != "/usr/local/bin/llm" {
		t.Errorf("expected LLMValidateBin=/usr/local/bin/llm, got %q", a.LLMValidateBin)
	}
	if a.LLMModel != "claude-opus-4-6" {
		t.Errorf("expected LLMModel=claude-opus-4-6, got %q", a.LLMModel)
	}
	if a.AutoAcceptThreshold != 0.8 {
		t.Errorf("expected AutoAcceptThreshold=0.8, got %v", a.AutoAcceptThreshold)
	}
	if a.AutoRejectThreshold != 0.3 {
		t.Errorf("expected AutoRejectThreshold=0.3, got %v", a.AutoRejectThreshold)
	}
}

// TestParseReviewArgs_DefaultValues tests that default values are set correctly.
func TestParseReviewArgs_DefaultValues(t *testing.T) {
	args := []string{} // No flags

	a, err := ParseReviewArgs(args)
	if err != nil {
		t.Fatalf("ParseReviewArgs failed: %v", err)
	}

	if a.LLMValidateBin != "pageindex" {
		t.Errorf("expected default LLMValidateBin=pageindex, got %q", a.LLMValidateBin)
	}
	if a.LLMModel != "claude-sonnet-4-5" {
		t.Errorf("expected default LLMModel=claude-sonnet-4-5, got %q", a.LLMModel)
	}
	if a.AutoAcceptThreshold != 0.0 {
		t.Errorf("expected default AutoAcceptThreshold=0.0, got %v", a.AutoAcceptThreshold)
	}
}

// TestCmdRelationshipsReview_LLMValidate_AppendsSignal tests that LLM validation
// appends an "llm" signal to pending relationships.
func TestCmdRelationshipsReview_LLMValidate_AppendsSignal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a discovered manifest with pending relationships.
	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "service-a.md",
				Target:     "service-b.md",
				Type:       "depends-on",
				Confidence: 0.65,
				Status:     "pending",
				Signals: []ManifestSignal{
					{Type: "mention", Value: 0.6, Evidence: "imports service-b"},
				},
			},
		},
	}

	// Create a fake LLM validator script.
	scriptPath := filepath.Join(tmpDir, "llm-validator")
	scriptContent := `#!/bin/bash
echo '{"valid": true, "confidence": 0.85, "reasoning": "relationship is real"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call CmdRelationshipsReview with LLM validation.
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", scriptPath,
	}
	err := CmdRelationshipsReview(args)
	if err != nil {
		t.Fatalf("CmdRelationshipsReview failed: %v", err)
	}

	// Load the result and check that an "llm" signal was appended.
	acceptedPath := filepath.Join(tmpDir, AcceptedManifestFile)
	result, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load result: %v", err)
	}
	if result == nil {
		t.Fatal("result manifest is nil")
	}
	if len(result.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(result.Relationships))
	}

	rel := result.Relationships[0]
	llmSignalFound := false
	for _, sig := range rel.Signals {
		if sig.Type == string(SignalLLM) {
			llmSignalFound = true
			if sig.Value != 0.85 {
				t.Errorf("expected LLM signal confidence=0.85, got %v", sig.Value)
			}
			if sig.Evidence != "relationship is real" {
				t.Errorf("expected LLM evidence=relationship is real, got %q", sig.Evidence)
			}
		}
	}
	if !llmSignalFound {
		t.Error("LLM signal not found in relationship")
	}
}

// TestCmdRelationshipsReview_LLMValidate_AutoAcceptThreshold tests that
// relationships are auto-accepted when LLM confidence meets threshold.
func TestCmdRelationshipsReview_LLMValidate_AutoAcceptThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "depends-on",
				Confidence: 0.5,
				Status:     "pending",
				Signals:    []ManifestSignal{},
			},
		},
	}

	// Create a validator that returns high confidence.
	scriptPath := filepath.Join(tmpDir, "llm-validator")
	scriptContent := `#!/bin/bash
echo '{"valid": true, "confidence": 0.9, "reasoning": "high confidence"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call with auto-accept threshold of 0.8.
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", scriptPath,
		"--auto-accept-threshold", "0.8",
	}
	err := CmdRelationshipsReview(args)
	if err != nil {
		t.Fatalf("CmdRelationshipsReview failed: %v", err)
	}

	// Verify relationship was auto-accepted.
	acceptedPath := filepath.Join(tmpDir, AcceptedManifestFile)
	result, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load result: %v", err)
	}
	if len(result.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(result.Relationships))
	}

	rel := result.Relationships[0]
	if rel.Status != "accepted" {
		t.Errorf("expected status=accepted, got %q", rel.Status)
	}
	if !rel.Reviewed {
		t.Error("expected Reviewed=true for auto-accepted relationship")
	}
}

// TestCmdRelationshipsReview_LLMValidate_AutoRejectThreshold tests that
// relationships are auto-rejected when LLM confidence is below threshold.
func TestCmdRelationshipsReview_LLMValidate_AutoRejectThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "depends-on",
				Confidence: 0.5,
				Status:     "pending",
				Signals:    []ManifestSignal{},
			},
		},
	}

	// Create a validator that returns low confidence.
	scriptPath := filepath.Join(tmpDir, "llm-validator")
	scriptContent := `#!/bin/bash
echo '{"valid": false, "confidence": 0.2, "reasoning": "low confidence"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call with auto-reject threshold of 0.4.
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", scriptPath,
		"--auto-reject-threshold", "0.4",
	}
	err := CmdRelationshipsReview(args)
	if err != nil {
		t.Fatalf("CmdRelationshipsReview failed: %v", err)
	}

	// Verify relationship was auto-rejected.
	acceptedPath := filepath.Join(tmpDir, AcceptedManifestFile)
	result, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load result: %v", err)
	}
	if len(result.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(result.Relationships))
	}

	rel := result.Relationships[0]
	if rel.Status != "rejected" {
		t.Errorf("expected status=rejected, got %q", rel.Status)
	}
	if !rel.Reviewed {
		t.Error("expected Reviewed=true for auto-rejected relationship")
	}
}

// TestCmdRelationshipsReview_LLMValidate_SkipsNonPending tests that
// already-reviewed relationships are not re-validated.
func TestCmdRelationshipsReview_LLMValidate_SkipsNonPending(t *testing.T) {
	tmpDir := t.TempDir()

	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "depends-on",
				Confidence: 0.5,
				Status:     "accepted", // Already accepted
				Reviewed:   true,
				Signals:    []ManifestSignal{},
			},
		},
	}

	// Create a validator that would fail if called.
	scriptPath := filepath.Join(tmpDir, "llm-validator")
	scriptContent := `#!/bin/bash
exit 1  # Fail if called
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call with LLM validation (should skip the non-pending relationship).
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", scriptPath,
	}
	err := CmdRelationshipsReview(args)
	// Should NOT error out even though validator would fail, because we skip non-pending.
	if err != nil {
		t.Fatalf("CmdRelationshipsReview failed: %v", err)
	}

	// Verify relationship status unchanged.
	acceptedPath := filepath.Join(tmpDir, AcceptedManifestFile)
	result, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load result: %v", err)
	}
	if result.Relationships[0].Status != "accepted" {
		t.Error("status should remain accepted for already-reviewed relationships")
	}
}

// TestCmdRelationshipsReview_LLMValidate_GracefulDegradation tests that
// non-fatal validator errors don't stop processing.
func TestCmdRelationshipsReview_LLMValidate_GracefulDegradation(t *testing.T) {
	tmpDir := t.TempDir()

	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "depends-on",
				Confidence: 0.5,
				Status:     "pending",
				Signals:    []ManifestSignal{},
			},
		},
	}

	// Create a validator that exits with non-zero status (but not "not found").
	scriptPath := filepath.Join(tmpDir, "llm-validator")
	scriptContent := `#!/bin/bash
exit 2  # Non-zero, but not executable-not-found
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call with LLM validation.
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", scriptPath,
	}
	// Should complete without error despite validator exit code.
	err := CmdRelationshipsReview(args)
	if err != nil {
		t.Fatalf("CmdRelationshipsReview failed: %v", err)
	}

	// Verify the manifest was created (graceful degradation).
	acceptedPath := filepath.Join(tmpDir, AcceptedManifestFile)
	result, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load result: %v", err)
	}
	if len(result.Relationships) != 1 {
		t.Fatal("manifest should be saved despite validation errors")
	}
}

// TestCmdRelationshipsReview_LLMValidate_BinaryNotFound tests that
// a missing validator binary returns an error.
func TestCmdRelationshipsReview_LLMValidate_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "depends-on",
				Confidence: 0.5,
				Status:     "pending",
				Signals:    []ManifestSignal{},
			},
		},
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call with non-existent validator binary.
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", "/nonexistent/validator",
	}
	err := CmdRelationshipsReview(args)
	if err == nil {
		t.Fatal("expected error for missing validator binary")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestCmdRelationshipsReview_LLMValidate_WithAcceptAll tests that
// --llm-validate can be combined with --accept-all.
func TestCmdRelationshipsReview_LLMValidate_WithAcceptAll(t *testing.T) {
	tmpDir := t.TempDir()

	discovered := &RelationshipManifest{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Relationships: []ManifestRelationship{
			{
				Source:     "a.md",
				Target:     "b.md",
				Type:       "depends-on",
				Confidence: 0.5,
				Status:     "pending",
				Signals:    []ManifestSignal{},
			},
		},
	}

	// Create a validator.
	scriptPath := filepath.Join(tmpDir, "llm-validator")
	scriptContent := `#!/bin/bash
echo '{"valid": true, "confidence": 0.7, "reasoning": "test"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	discoveredPath := filepath.Join(tmpDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(discovered, discoveredPath); err != nil {
		t.Fatalf("save discovered: %v", err)
	}

	// Call with both --llm-validate and --accept-all.
	args := []string{
		"--dir", tmpDir,
		"--llm-validate",
		"--llm-validate-bin", scriptPath,
		"--accept-all",
	}
	err := CmdRelationshipsReview(args)
	if err != nil {
		t.Fatalf("CmdRelationshipsReview failed: %v", err)
	}

	// Verify relationship was accepted.
	acceptedPath := filepath.Join(tmpDir, AcceptedManifestFile)
	result, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		t.Fatalf("load result: %v", err)
	}
	if result.Relationships[0].Status != "accepted" {
		t.Error("relationship should be accepted after --accept-all")
	}
}
