package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/bmd/bmd/internal/knowledge"
)

// handleQuery handles the bmd/query MCP tool invocation.
// It delegates to knowledge.CmdQuery with a captured stdout writer.
func (s *Server) handleQuery(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	query := mcpsdk.ParseString(req, "query", "")
	if query == "" {
		return mcpsdk.NewToolResultError("query parameter is required"), nil
	}

	strategy := mcpsdk.ParseString(req, "strategy", "bm25")
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	top := mcpsdk.ParseInt(req, "top", 10)

	args := []string{query, "--dir", dir, "--format", "json",
		"--strategy", strategy, "--top", fmt.Sprintf("%d", top)}

	output, err := captureOutput(func() error {
		return knowledge.CmdQuery(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// handleIndex handles the bmd/index MCP tool invocation.
// It delegates to knowledge.CmdIndex, capturing stderr progress messages.
func (s *Server) handleIndex(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	strategy := mcpsdk.ParseString(req, "strategy", "")
	model := mcpsdk.ParseString(req, "model", "claude-sonnet-4-5")

	args := []string{"--dir", dir}
	if strategy != "" {
		args = append(args, "--strategy", strategy)
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// CmdIndex writes progress to stderr; capture it for the response.
	stderr, err := captureStderr(func() error {
		return knowledge.CmdIndex(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("index failed: %v", err)), nil
	}

	msg := "Indexing complete."
	if stderr != "" {
		msg = stderr
	}
	return mcpsdk.NewToolResultText(msg), nil
}

// handleDepends handles the bmd/depends MCP tool invocation.
// It delegates to knowledge.CmdDepends with JSON output format.
func (s *Server) handleDepends(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	service := mcpsdk.ParseString(req, "service", "")
	if service == "" {
		return mcpsdk.NewToolResultError("service parameter is required"), nil
	}

	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	transitive := mcpsdk.ParseBoolean(req, "transitive", false)

	args := []string{service, "--dir", dir, "--format", "json"}
	if transitive {
		args = append(args, "--transitive")
	}

	output, err := captureOutput(func() error {
		return knowledge.CmdDepends(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("depends failed: %v", err)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// handleServices handles the bmd/services MCP tool invocation.
// It delegates to knowledge.CmdServices with JSON output format.
func (s *Server) handleServices(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)

	args := []string{"--dir", dir, "--format", "json"}

	output, err := captureOutput(func() error {
		return knowledge.CmdServices(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("services failed: %v", err)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// handleGraph handles the bmd/graph MCP tool invocation.
// It delegates to knowledge.CmdGraph with JSON output format.
func (s *Server) handleGraph(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	service := mcpsdk.ParseString(req, "service", "")

	args := []string{"--dir", dir, "--format", "json"}
	if service != "" {
		args = append(args, "--service", service)
	}

	output, err := captureOutput(func() error {
		return knowledge.CmdGraph(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("graph failed: %v", err)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// handleContext handles the bmd/context MCP tool invocation.
// It delegates to knowledge.CmdContext with JSON output format.
func (s *Server) handleContext(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	query := mcpsdk.ParseString(req, "query", "")
	if query == "" {
		return mcpsdk.NewToolResultError("query parameter is required"), nil
	}

	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	top := mcpsdk.ParseInt(req, "top", 5)
	format := mcpsdk.ParseString(req, "format", "json")

	args := []string{query, "--dir", dir, "--top", fmt.Sprintf("%d", top), "--format", format}

	output, err := captureOutput(func() error {
		return knowledge.CmdContext(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("context failed: %v", err)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// captureOutput redirects os.Stdout to a buffer, calls fn, then restores stdout.
// Returns the captured output or an error if fn fails.
func captureOutput(fn func() error) (string, error) {
	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		// If we can't pipe, just run the function without capturing.
		return "", fn()
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&buf, r)
		done <- copyErr
	}()

	fnErr := fn()

	w.Close()         //nolint:errcheck
	<-done
	os.Stdout = origStdout
	r.Close() //nolint:errcheck

	if fnErr != nil {
		return "", fnErr
	}

	return buf.String(), nil
}

// captureStderr redirects os.Stderr to a buffer, calls fn, then restores stderr.
// Returns the captured output. Errors from fn are also returned.
func captureStderr(fn func() error) (string, error) {
	origStderr := os.Stderr
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return "", fn()
	}
	os.Stderr = w

	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&buf, r)
		done <- copyErr
	}()

	fnErr := fn()

	w.Close()         //nolint:errcheck
	<-done
	os.Stderr = origStderr
	r.Close() //nolint:errcheck

	return buf.String(), fnErr
}
