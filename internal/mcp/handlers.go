package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// handleComponents handles the bmd/components MCP tool invocation.
// It delegates to knowledge.CmdComponents with JSON output format.
func (s *Server) handleComponents(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)

	args := []string{"--dir", dir, "--format", "json"}

	output, err := captureOutput(func() error {
		return knowledge.CmdComponents(args)
	})
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("components failed: %v", err)), nil
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

// handleGraphCrawl handles the bmd/graph_crawl MCP tool invocation.
// It loads the knowledge graph, parses crawl options from the MCP request,
// calls Graph.CrawlMulti(), and returns the result as a ContractResponse.
func (s *Server) handleGraphCrawl(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	startFilesStr := mcpsdk.ParseString(req, "start_files", "")
	if startFilesStr == "" {
		return mcpsdk.NewToolResultError("start_files parameter is required"), nil
	}

	// Parse comma-separated start files.
	var startFiles []string
	for _, f := range strings.Split(startFilesStr, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			startFiles = append(startFiles, f)
		}
	}
	if len(startFiles) == 0 {
		return mcpsdk.NewToolResultError("start_files parameter must contain at least one file path"), nil
	}

	direction := mcpsdk.ParseString(req, "direction", "forward")
	depth := mcpsdk.ParseInt(req, "depth", 10)
	includeCycles := mcpsdk.ParseBoolean(req, "include_cycles", false)
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)

	// Validate direction.
	switch direction {
	case "forward", "backward", "both":
		// valid
	default:
		resp := knowledge.NewErrorResponse("INVALID_PARAMS", fmt.Sprintf("invalid direction %q: must be 'forward', 'backward', or 'both'", direction))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	// Load the graph from the indexed directory.
	absDir, err := absPath(dir)
	if err != nil {
		resp := knowledge.NewErrorResponse(knowledge.ErrCodeInternalError, err.Error())
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	graph, db, err := loadGraph(absDir)
	if err != nil {
		resp := knowledge.NewErrorResponse("GRAPH_NOT_FOUND", fmt.Sprintf("failed to load graph: %v", err))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}
	defer db.Close() //nolint:errcheck

	// Validate start files exist in the graph.
	for _, sf := range startFiles {
		if _, ok := graph.Nodes[sf]; !ok {
			resp := knowledge.NewErrorResponse("GRAPH_NOT_FOUND", fmt.Sprintf("start file %q not found in graph", sf))
			data, _ := json.MarshalIndent(resp, "", "  ")
			return mcpsdk.NewToolResultText(string(data)), nil
		}
	}

	// Execute crawl.
	opts := knowledge.CrawlOptions{
		FromFiles:     startFiles,
		Direction:     direction,
		MaxDepth:      depth,
		IncludeCycles: includeCycles,
	}

	crawlResult := graph.CrawlMulti(opts)

	// Format result as ContractResponse.
	resp := knowledge.NewOKResponse("Graph crawl completed", crawlResult)
	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcpsdk.NewToolResultText(string(data)), nil
}

// handleRelationshipsValidate handles the bmd/relationships_validate MCP tool invocation.
// It validates pending relationships in the discovered manifest via LLM subprocess.
func (s *Server) handleRelationshipsValidate(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	llmModel := mcpsdk.ParseString(req, "llm_model", "claude-sonnet-4-5")
	// Parse thresholds - use ParseString and manual float conversion for compatibility
	autoAcceptThresholdStr := mcpsdk.ParseString(req, "auto_accept_threshold", "0.0")
	autoRejectThresholdStr := mcpsdk.ParseString(req, "auto_reject_threshold", "0.0")
	autoAcceptThreshold := parseThreshold(autoAcceptThresholdStr)
	autoRejectThreshold := parseThreshold(autoRejectThresholdStr)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		errResp := knowledge.NewErrorResponse("INVALID_PATH", fmt.Sprintf("failed to resolve dir: %v", err))
		data, _ := json.MarshalIndent(errResp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	// Build args for CmdRelationshipsReview with validation.
	args := []string{
		"--dir", absDir,
		"--llm-validate",
		"--llm-model", llmModel,
	}
	if autoAcceptThreshold > 0.0 {
		args = append(args, "--auto-accept-threshold", fmt.Sprintf("%v", autoAcceptThreshold))
	}
	if autoRejectThreshold > 0.0 {
		args = append(args, "--auto-reject-threshold", fmt.Sprintf("%v", autoRejectThreshold))
	}

	// Run validation.
	stderr, err := captureStderr(func() error {
		return knowledge.CmdRelationshipsReview(args)
	})
	if err != nil {
		errResp := knowledge.NewErrorResponse("VALIDATION_FAILED", fmt.Sprintf("validation failed: %v", err))
		data, _ := json.MarshalIndent(errResp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	// Load the result manifest to extract summary stats.
	acceptedPath := filepath.Join(absDir, knowledge.AcceptedManifestFile)
	manifest, err := knowledge.LoadRelationshipManifest(acceptedPath)
	if err != nil {
		errResp := knowledge.NewErrorResponse("LOAD_FAILED", fmt.Sprintf("failed to load result manifest: %v", err))
		data, _ := json.MarshalIndent(errResp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}
	if manifest == nil {
		errResp := knowledge.NewErrorResponse("NO_MANIFEST", "no relationships manifest found")
		data, _ := json.MarshalIndent(errResp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	summary := manifest.Summarize()
	resultData := map[string]interface{}{
		"validated": summary.Total,
		"accepted":  summary.Accepted,
		"rejected":  summary.Rejected,
		"pending":   summary.Pending,
		"message":   stderr,
	}

	okResp := knowledge.NewOKResponse("Relationships validated", resultData)
	data, _ := json.MarshalIndent(okResp, "", "  ")
	return mcpsdk.NewToolResultText(string(data)), nil
}

// handleWatchStart handles the bmd/watch_start MCP tool invocation.
// It creates a FileWatcher and IncrementalUpdater for the requested directory,
// registers a new WatchSession, and returns the session ID.
func (s *Server) handleWatchStart(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	dir := mcpsdk.ParseString(req, "dir", s.baseDir)
	intervalMs := mcpsdk.ParseInt(req, "interval_ms", 500)

	absDir, err := absPath(dir)
	if err != nil {
		resp := knowledge.NewErrorResponse(knowledge.ErrCodeInternalError, fmt.Sprintf("resolve dir: %v", err))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	// Load or build the graph and index for the directory.
	graph, db, err := loadGraph(absDir)
	if err != nil {
		// Continue with an empty graph — the watch will still detect changes.
		graph = knowledge.NewGraph()
		db = nil
	}

	// Load the index from scratch for incremental updates.
	idx := knowledge.NewIndex()
	if db != nil {
		// Scan to populate index for incremental-update hash comparison.
		if docs, scanErr := knowledge.ScanDirectory(absDir); scanErr == nil {
			_ = idx.Build(docs)
		}
	}

	// Load or initialise the component registry.
	reg, _ := knowledge.LoadRegistry(filepath.Join(absDir, knowledge.RegistryFileName))
	if reg == nil {
		reg = knowledge.NewComponentRegistry()
	}

	// Create watcher and updater.
	watcher := knowledge.NewFileWatcher(absDir, time.Duration(intervalMs)*time.Millisecond)
	updater := knowledge.NewIncrementalUpdater(absDir, watcher, idx, graph, reg, db, nil)

	// Register session (wires onChange callback).
	session := s.watchMgr.Create(absDir, updater)

	// Start watching.
	watcher.Start()
	updater.Start()

	type watchStartResult struct {
		SessionID  string `json:"session_id"`
		Dir        string `json:"dir"`
		IntervalMs int    `json:"interval_ms"`
		Status     string `json:"status"`
	}
	resp := knowledge.NewOKResponse("Watch session started", watchStartResult{
		SessionID:  session.ID,
		Dir:        absDir,
		IntervalMs: intervalMs,
		Status:     "watching",
	})
	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcpsdk.NewToolResultText(string(data)), nil
}

// handleWatchPoll handles the bmd/watch_poll MCP tool invocation.
// It drains the pending notification queue for the session and returns all
// change events since the last poll.
func (s *Server) handleWatchPoll(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	sessionID := mcpsdk.ParseString(req, "session_id", "")
	if sessionID == "" {
		return mcpsdk.NewToolResultError("session_id parameter is required"), nil
	}

	session, ok := s.watchMgr.Get(sessionID)
	if !ok {
		resp := knowledge.NewErrorResponse("SESSION_NOT_FOUND", fmt.Sprintf("watch session %q not found", sessionID))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	notifications := session.DrainPending()

	type watchPollResult struct {
		SessionID     string               `json:"session_id"`
		Notifications []WatchNotification  `json:"notifications"`
		Count         int                  `json:"count"`
	}
	resp := knowledge.NewOKResponse("Poll complete", watchPollResult{
		SessionID:     sessionID,
		Notifications: notifications,
		Count:         len(notifications),
	})
	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcpsdk.NewToolResultText(string(data)), nil
}

// handleWatchStop handles the bmd/watch_stop MCP tool invocation.
// It terminates the watch session, stopping the watcher and releasing resources.
func (s *Server) handleWatchStop(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	sessionID := mcpsdk.ParseString(req, "session_id", "")
	if sessionID == "" {
		return mcpsdk.NewToolResultError("session_id parameter is required"), nil
	}

	_, ok := s.watchMgr.Get(sessionID)
	if !ok {
		resp := knowledge.NewErrorResponse("SESSION_NOT_FOUND", fmt.Sprintf("watch session %q not found", sessionID))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	s.watchMgr.Delete(sessionID)

	type watchStopResult struct {
		SessionID string `json:"session_id"`
		Status    string `json:"status"`
	}
	resp := knowledge.NewOKResponse("Watch session stopped", watchStopResult{
		SessionID: sessionID,
		Status:    "stopped",
	})
	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcpsdk.NewToolResultText(string(data)), nil
}

// parseThreshold parses a threshold string to a float64, defaulting to 0.0 on error.
func parseThreshold(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// absPath resolves a directory path to an absolute path.
func absPath(dir string) (string, error) {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve dir %q: %w", dir, err)
	}
	return abs, nil
}

// loadGraph opens the knowledge database for a directory and loads the graph.
func loadGraph(absDir string) (*knowledge.Graph, *knowledge.Database, error) {
	dbPath := filepath.Join(absDir, ".bmd", "knowledge.db")

	// Auto-build if needed (captures stderr so the MCP response is clean).
	_, buildErr := captureStderr(func() error {
		return knowledge.CmdIndex([]string{"--dir", absDir, "--db", dbPath})
	})
	if buildErr != nil {
		return nil, nil, buildErr
	}

	db, err := knowledge.OpenDB(dbPath)
	if err != nil {
		return nil, nil, err
	}

	graph := knowledge.NewGraph()
	if err := db.LoadGraph(graph); err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	return graph, db, nil
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
