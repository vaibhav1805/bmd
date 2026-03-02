package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContainerFiles validates that all container deployment files exist and
// contain expected content. These are static validation tests that don't
// require a Docker daemon.
// Uses projectRoot(t) from openclaw_test.go.

func TestDockerfileExists(t *testing.T) {
	root := projectRoot(t)

	path := filepath.Join(root, "Dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Dockerfile not found: %v", err)
	}

	content := string(data)

	// Verify multi-stage build.
	if !strings.Contains(content, "AS builder") {
		t.Error("Dockerfile should use multi-stage build (AS builder)")
	}

	// Verify it builds from Go alpine.
	if !strings.Contains(content, "golang:") {
		t.Error("Dockerfile should use golang base image")
	}

	// Verify stripped binary flags.
	if !strings.Contains(content, "-ldflags") || !strings.Contains(content, "-s -w") {
		t.Error("Dockerfile should build with stripped binary flags (-s -w)")
	}

	// Verify CGO disabled for static binary.
	if !strings.Contains(content, "CGO_ENABLED=0") {
		t.Error("Dockerfile should set CGO_ENABLED=0")
	}

	// Verify non-root user.
	if !strings.Contains(content, "adduser") || !strings.Contains(content, "USER") {
		t.Error("Dockerfile should create and use a non-root user")
	}

	// Verify health check.
	if !strings.Contains(content, "HEALTHCHECK") {
		t.Error("Dockerfile should include a HEALTHCHECK")
	}

	// Verify knowledge tar packaging.
	if !strings.Contains(content, "knowledge.tar.gz") {
		t.Error("Dockerfile should reference knowledge.tar.gz")
	}

	// Verify headless mode in default CMD.
	if !strings.Contains(content, "--headless") {
		t.Error("Dockerfile CMD should include --headless flag")
	}
}

func TestDockerComposeExists(t *testing.T) {
	root := projectRoot(t)

	path := filepath.Join(root, "docker-compose.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("docker-compose.yaml not found: %v", err)
	}

	content := string(data)

	// Verify bmd service exists.
	if !strings.Contains(content, "bmd:") {
		t.Error("docker-compose.yaml should define a bmd service")
	}

	// Verify agent service exists (sidecar pattern).
	if !strings.Contains(content, "agent:") {
		t.Error("docker-compose.yaml should define an agent service")
	}

	// Verify depends_on for ordering.
	if !strings.Contains(content, "depends_on") {
		t.Error("docker-compose.yaml agent should depend on bmd")
	}

	// Verify resource limits.
	if !strings.Contains(content, "memory:") {
		t.Error("docker-compose.yaml should set memory limits")
	}

	// Verify restart policy.
	if !strings.Contains(content, "restart:") {
		t.Error("docker-compose.yaml should set restart policy")
	}

	// Verify healthcheck.
	if !strings.Contains(content, "healthcheck:") {
		t.Error("docker-compose.yaml should include healthcheck")
	}
}

func TestKubernetesManifestsExist(t *testing.T) {
	root := projectRoot(t)

	k8sDir := filepath.Join(root, "kubernetes")

	expectedFiles := []string{
		"namespace.yaml",
		"configmap.yaml",
		"deployment.yaml",
		"service.yaml",
		"kustomization.yaml",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(k8sDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Kubernetes manifest missing: %s", name)
		}
	}
}

func TestKubernetesDeploymentContent(t *testing.T) {
	root := projectRoot(t)

	path := filepath.Join(root, "kubernetes", "deployment.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("deployment.yaml not found: %v", err)
	}

	content := string(data)

	// Verify it's a Deployment.
	if !strings.Contains(content, "kind: Deployment") {
		t.Error("deployment.yaml should be kind: Deployment")
	}

	// Verify namespace.
	if !strings.Contains(content, "namespace: bmd") {
		t.Error("deployment.yaml should use namespace: bmd")
	}

	// Verify init container for knowledge extraction.
	if !strings.Contains(content, "initContainers") {
		t.Error("deployment.yaml should have an initContainer")
	}

	// Verify resource limits.
	if !strings.Contains(content, "limits:") && !strings.Contains(content, "requests:") {
		t.Error("deployment.yaml should set resource limits")
	}

	// Verify probes.
	if !strings.Contains(content, "livenessProbe") {
		t.Error("deployment.yaml should have a liveness probe")
	}
	if !strings.Contains(content, "readinessProbe") {
		t.Error("deployment.yaml should have a readiness probe")
	}

	// Verify security context.
	if !strings.Contains(content, "runAsNonRoot") {
		t.Error("deployment.yaml should set runAsNonRoot")
	}

	// Verify headless mode.
	if !strings.Contains(content, "--headless") {
		t.Error("deployment.yaml should use --headless flag")
	}

	// Verify volume mount.
	if !strings.Contains(content, "volumeMounts") {
		t.Error("deployment.yaml should have volume mounts")
	}
}

func TestKubernetesServiceContent(t *testing.T) {
	root := projectRoot(t)

	path := filepath.Join(root, "kubernetes", "service.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("service.yaml not found: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "kind: Service") {
		t.Error("service.yaml should be kind: Service")
	}
	if !strings.Contains(content, "ClusterIP") {
		t.Error("service.yaml should be type: ClusterIP")
	}
	if !strings.Contains(content, "namespace: bmd") {
		t.Error("service.yaml should use namespace: bmd")
	}
}

func TestDockerignoreExists(t *testing.T) {
	root := projectRoot(t)

	path := filepath.Join(root, ".dockerignore")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(".dockerignore not found: %v", err)
	}

	content := string(data)

	// Should exclude .git for smaller build context.
	if !strings.Contains(content, ".git") {
		t.Error(".dockerignore should exclude .git")
	}
	// Should exclude test data.
	if !strings.Contains(content, "test-data") {
		t.Error(".dockerignore should exclude test-data")
	}
}
