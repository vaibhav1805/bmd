package knowledge

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// projectRoot returns the root of the bmd project by walking up from this file's location.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine source file location")
	}
	// filename is .../internal/knowledge/openclaw_test.go — go up 3 levels
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("cannot resolve project root: %v", err)
	}
	return abs
}

// TestOpenClawDescriptor_Valid validates that openclaw.yaml exists and contains
// the required fields: apiVersion, kind, all 4 commands, MCP config, and capabilities.
func TestOpenClawDescriptor_Valid(t *testing.T) {
	root := projectRoot(t)
	descriptorPath := filepath.Join(root, "openclaw.yaml")

	data, err := os.ReadFile(descriptorPath)
	if err != nil {
		t.Fatalf("openclaw.yaml not found at %s: %v", descriptorPath, err)
	}

	content := string(data)

	requiredFields := []struct {
		field string
		desc  string
	}{
		{"apiVersion: openclaw.ai/v1", "apiVersion"},
		{"kind: Plugin", "kind"},
		{"name: bmd-documentation-service", "plugin name"},
		{"version: 1.0.0", "plugin version"},
		{"- name: query", "query command"},
		{"- name: index", "index command"},
		{"- name: context", "context command"},
		{"- name: depends", "depends command"},
		{"mcp:", "MCP configuration"},
		{"enabled: true", "MCP enabled"},
		{"protocol: stdio", "MCP protocol"},
		{"command: [\"bmd\", \"serve\", \"--mcp\"]", "MCP command"},
		{"capabilities:", "capabilities section"},
		{"- semantic-search", "semantic-search capability"},
		{"- knowledge-graphs", "knowledge-graphs capability"},
		{"- service-detection", "service-detection capability"},
		{"- documentation-indexing", "documentation-indexing capability"},
		{"- rag-assembly", "rag-assembly capability"},
	}

	for _, rf := range requiredFields {
		if !strings.Contains(content, rf.field) {
			t.Errorf("openclaw.yaml missing %s: expected to contain %q", rf.desc, rf.field)
		}
	}
}

// TestOpenClawDescriptor_QueryCommand checks the query command has required args.
func TestOpenClawDescriptor_QueryCommand(t *testing.T) {
	root := projectRoot(t)
	descriptorPath := filepath.Join(root, "openclaw.yaml")

	data, err := os.ReadFile(descriptorPath)
	if err != nil {
		t.Fatalf("openclaw.yaml not found: %v", err)
	}

	content := string(data)

	// Query command must have: query arg (required), dir arg, strategy arg with bm25/pageindex enum
	queryRequired := []string{
		"name: query",
		"default: bm25",
		"enum: [bm25, pageindex]",
	}
	for _, req := range queryRequired {
		if !strings.Contains(content, req) {
			t.Errorf("query command missing field: %q", req)
		}
	}
}

// TestDockerfile_Valid validates that the Dockerfile exists and contains required
// build stages and runtime configuration.
func TestDockerfile_Valid(t *testing.T) {
	root := projectRoot(t)
	dockerfilePath := filepath.Join(root, "Dockerfile")

	data, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("Dockerfile not found at %s: %v", dockerfilePath, err)
	}

	content := string(data)

	requiredFields := []struct {
		field string
		desc  string
	}{
		{"FROM golang:", "Go builder stage"},
		{"AS builder", "named builder stage"},
		{"FROM alpine:", "Alpine runtime stage"},
		{"CGO_ENABLED=0", "static binary build"},
		{"./cmd/bmd", "correct build target"},
		{"/usr/local/bin/bmd", "binary install location"},
		{"pageindex", "PageIndex installation"},
		{"HEALTHCHECK", "health check directive"},
		{"bmd serve --mcp", "MCP server health check"},
		{`ENTRYPOINT ["bmd"]`, "entrypoint"},
		{`CMD ["serve", "--mcp"]`, "default MCP serve command"},
	}

	for _, rf := range requiredFields {
		if !strings.Contains(content, rf.field) {
			t.Errorf("Dockerfile missing %s: expected to contain %q", rf.desc, rf.field)
		}
	}
}

// TestDockerCompose_Valid validates that docker-compose.yaml exists and has
// required volume mounts and environment variables.
func TestDockerCompose_Valid(t *testing.T) {
	root := projectRoot(t)
	composePath := filepath.Join(root, "docker-compose.yaml")

	data, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("docker-compose.yaml not found at %s: %v", composePath, err)
	}

	content := string(data)

	requiredFields := []struct {
		field string
		desc  string
	}{
		{"version: '3.8'", "compose version"},
		{"./docs:/docs:ro", "docs volume (read-only)"},
		{"./data:/data", "data volume (persistent)"},
		{"BMD_STRATEGY=pageindex", "strategy env var"},
		{"BMD_DB=/data/bmd.db", "database path env var"},
	}

	for _, rf := range requiredFields {
		if !strings.Contains(content, rf.field) {
			t.Errorf("docker-compose.yaml missing %s: expected to contain %q", rf.desc, rf.field)
		}
	}
}

// TestDeploymentDocs_Valid validates that DEPLOYMENT.md exists and covers
// both fleet and self-hosted deployment options.
func TestDeploymentDocs_Valid(t *testing.T) {
	root := projectRoot(t)
	docsPath := filepath.Join(root, "DEPLOYMENT.md")

	data, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatalf("DEPLOYMENT.md not found at %s: %v", docsPath, err)
	}

	content := string(data)

	requiredSections := []struct {
		field string
		desc  string
	}{
		{"openclaw plugin register", "plugin registration command"},
		{"openclaw fleet deploy", "fleet deployment command"},
		{"docker-compose up", "self-hosted deployment"},
		{"BMD_STRATEGY", "BMD_STRATEGY env var documentation"},
		{"BMD_DB", "BMD_DB env var documentation"},
		{"BMD_MODEL", "BMD_MODEL env var documentation"},
	}

	for _, rs := range requiredSections {
		if !strings.Contains(content, rs.field) {
			t.Errorf("DEPLOYMENT.md missing %s: expected to contain %q", rs.desc, rs.field)
		}
	}
}
