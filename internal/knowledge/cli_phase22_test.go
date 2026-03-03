package knowledge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// captureKnowledgeOutput redirects stdout and calls fn, returning captured output.
func captureKnowledgeOutput(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("pipe: %w", err)
	}
	os.Stdout = w

	fnErr := fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String(), fnErr
}

// ─── ParseDebugArgs CLI tests ─────────────────────────────────────────────────

func TestCLI_ParseDebugArgs_AllFlags(t *testing.T) {
	args := []string{
		"--component", "auth",
		"--query", "why is auth failing",
		"--dir", "/tmp/repo",
		"--depth", "4",
		"--format", "text",
	}
	a, err := ParseDebugArgs(args)
	if err != nil {
		t.Fatalf("ParseDebugArgs: %v", err)
	}
	if a.Component != "auth" {
		t.Errorf("Component = %q, want 'auth'", a.Component)
	}
	if a.Query != "why is auth failing" {
		t.Errorf("Query = %q, want 'why is auth failing'", a.Query)
	}
	if a.Dir != "/tmp/repo" {
		t.Errorf("Dir = %q, want '/tmp/repo'", a.Dir)
	}
	if a.Depth != 4 {
		t.Errorf("Depth = %d, want 4", a.Depth)
	}
	if a.Output != "text" {
		t.Errorf("Output = %q, want 'text'", a.Output)
	}
}

func TestCLI_ParseDebugArgs_DefaultDepth(t *testing.T) {
	a, err := ParseDebugArgs([]string{"--component", "payment"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Depth != 2 {
		t.Errorf("default Depth = %d, want 2", a.Depth)
	}
}

func TestCLI_ParseDebugArgs_DefaultFormat(t *testing.T) {
	a, err := ParseDebugArgs([]string{"--component", "payment"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Output != "json" {
		t.Errorf("default Output = %q, want 'json'", a.Output)
	}
}

func TestCLI_ParseDebugArgs_MissingComponentError(t *testing.T) {
	_, err := ParseDebugArgs([]string{"--depth", "2"})
	if err == nil {
		t.Fatal("expected error for missing --component, got nil")
	}
	if !strings.Contains(err.Error(), "--component") {
		t.Errorf("error %q should mention --component", err.Error())
	}
}

func TestCLI_ParseDebugArgs_UnknownFlagError(t *testing.T) {
	_, err := ParseDebugArgs([]string{"--unknown-flag", "x", "--component", "svc"})
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

// ─── ParseComponentsGraphArgs CLI tests ──────────────────────────────────────

func TestCLI_ParseComponentsGraphArgs_AllFlags(t *testing.T) {
	a, err := ParseComponentsGraphArgs([]string{"--dir", "/repo", "--format", "json"})
	if err != nil {
		t.Fatalf("ParseComponentsGraphArgs: %v", err)
	}
	if a.Dir != "/repo" {
		t.Errorf("Dir = %q, want '/repo'", a.Dir)
	}
	if a.Format != "json" {
		t.Errorf("Format = %q, want 'json'", a.Format)
	}
}

func TestCLI_ParseComponentsGraphArgs_PositionalDir(t *testing.T) {
	a, err := ParseComponentsGraphArgs([]string{"/my/dir"})
	if err != nil {
		t.Fatalf("ParseComponentsGraphArgs positional: %v", err)
	}
	if a.Dir != "/my/dir" {
		t.Errorf("Dir = %q, want '/my/dir'", a.Dir)
	}
}

// ─── CmdDebug output tests ────────────────────────────────────────────────────

func TestCLI_CmdDebug_JSONOutput_ValidEnvelope(t *testing.T) {
	tmpDir := t.TempDir()
	// Build a minimal monorepo with one component.
	mustMkdir(t, tmpDir, "services/payment")
	mustWriteFile(t, tmpDir, "services/payment/go.mod", "module payment")
	mustWriteFile(t, tmpDir, "services/payment/README.md", "# Payment Service\n\nProcesses payments.\n")

	output, err := captureKnowledgeOutput(func() error {
		return CmdDebug([]string{
			"--component", "payment",
			"--dir", tmpDir,
			"--depth", "1",
			"--format", "json",
		})
	})

	if err != nil {
		t.Fatalf("CmdDebug: %v", err)
	}

	// Output must be valid JSON.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("CmdDebug output is not valid JSON: %v\n\nOutput:\n%s", err, output)
	}

	// Must contain STATUS-01 fields.
	for _, field := range []string{"status", "code", "message"} {
		if _, ok := envelope[field]; !ok {
			t.Errorf("envelope missing field %q", field)
		}
	}
}

func TestCLI_CmdDebug_TextOutput(t *testing.T) {
	// Text mode with unknown component returns a Go error (non-JSON path).
	// Verify the error message is human-readable, not JSON.
	tmpDir := t.TempDir()
	mustMkdir(t, tmpDir, "services/payment")
	mustWriteFile(t, tmpDir, "services/payment/go.mod", "module payment")
	mustWriteFile(t, tmpDir, "services/payment/README.md", "# Payment Service\n\nProcesses payments.\n")

	err := CmdDebug([]string{
		"--component", "nonexistent",
		"--dir", tmpDir,
		"--depth", "1",
		"--format", "text",
	})

	// In text mode, errors propagate as Go errors (not JSON envelopes).
	if err == nil {
		t.Fatal("expected error for unknown component in text mode, got nil")
	}
	// The error should be a plain string message, not JSON.
	errStr := err.Error()
	if strings.HasPrefix(strings.TrimSpace(errStr), "{") {
		t.Errorf("text mode error should be plain string, got JSON: %s", errStr)
	}
}

func TestCLI_CmdDebug_UnknownComponentJSONError(t *testing.T) {
	tmpDir := t.TempDir()
	mustMkdir(t, tmpDir, "services/api")
	mustWriteFile(t, tmpDir, "services/api/go.mod", "module api")
	mustWriteFile(t, tmpDir, "services/api/README.md", "# API\n\nREST api.\n")

	output, err := captureKnowledgeOutput(func() error {
		return CmdDebug([]string{
			"--component", "nonexistent-component",
			"--dir", tmpDir,
			"--format", "json",
		})
	})

	// CmdDebug in JSON mode should not return an error — it writes JSON instead.
	if err != nil {
		t.Fatalf("CmdDebug JSON error mode: unexpected go error: %v", err)
	}

	// Output should be a JSON error envelope.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("error output is not valid JSON: %v\n\nOutput:\n%s", err, output)
	}
	if status, _ := envelope["status"].(string); status == "ok" {
		t.Errorf("status should not be 'ok' for unknown component, got: %s", status)
	}
}

// ─── CmdComponentsGraph output tests ─────────────────────────────────────────

func TestCLI_CmdComponentsGraph_ASCIIOutputDefault(t *testing.T) {
	// formatComponentGraphASCII is already tested via unit tests.
	// Here we verify the ASCII output path returns an error (not JSON) for
	// an empty dir, i.e., the text error path works correctly.
	tmpDir := t.TempDir()

	err := CmdComponentsGraph([]string{"--dir", tmpDir})

	// ASCII mode propagates errors as Go errors (not JSON envelopes).
	if err == nil {
		t.Fatal("expected error for empty directory in ASCII mode, got nil")
	}
	// Error should be a plain string, not JSON.
	errStr := err.Error()
	if strings.HasPrefix(strings.TrimSpace(errStr), "{") {
		t.Errorf("ASCII mode error should be plain string, got JSON: %s", errStr)
	}
}

func TestCLI_CmdComponentsGraph_JSONOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	mustMkdir(t, tmpDir, "services/api")
	mustWriteFile(t, tmpDir, "services/api/go.mod", "module api")
	mustWriteFile(t, tmpDir, "services/api/README.md", "# API\n\nEntrypoint.\n")

	output, err := captureKnowledgeOutput(func() error {
		return CmdComponentsGraph([]string{"--dir", tmpDir, "--format", "json"})
	})

	if err != nil {
		t.Fatalf("CmdComponentsGraph JSON: %v", err)
	}

	// Must be valid JSON.
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("JSON output is not valid: %v\n\nOutput:\n%s", err, output)
	}
}

// ─── listComponentNames helper test ──────────────────────────────────────────

func TestCLI_ListComponentNames_Sorted(t *testing.T) {
	comps := []Component{
		{ID: "z-svc", Name: "Z Svc", File: "z.md"},
		{ID: "a-svc", Name: "A Svc", File: "a.md"},
		{ID: "m-svc", Name: "M Svc", File: "m.md"},
	}
	cg := NewComponentGraph(comps)

	result := listComponentNames(cg)
	parts := strings.Split(result, ", ")
	if len(parts) != 3 {
		t.Fatalf("expected 3 component names, got %d: %v", len(parts), parts)
	}
	if parts[0] != "a-svc" || parts[1] != "m-svc" || parts[2] != "z-svc" {
		t.Errorf("expected sorted [a-svc, m-svc, z-svc], got %v", parts)
	}
}
