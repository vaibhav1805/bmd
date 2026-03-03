package mcp

import (
	"context"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps an MCP StdioServer with knowledge command handlers.
// It exposes all bmd knowledge tools as native MCP endpoints, eliminating
// subprocess overhead when agents communicate via stdin/stdout.
type Server struct {
	// baseDir is the documentation directory to index and search.
	baseDir string
	// dbPath is the SQLite database path for the knowledge index.
	dbPath string
	// watchMgr manages active filesystem watch sessions for MCP clients.
	watchMgr *WatchSessionManager
}

// NewServer creates a new MCP server configured for the given documentation directory.
func NewServer(baseDir, dbPath string) *Server {
	return &Server{
		baseDir:  baseDir,
		dbPath:   dbPath,
		watchMgr: NewWatchSessionManager(),
	}
}

// Start initializes and runs the MCP server on stdin/stdout.
// It registers all knowledge tools and blocks until the process is killed.
func (s *Server) Start(ctx context.Context) error {
	mcpServer := server.NewMCPServer(
		"bmd",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register all 14 knowledge tools.
	s.registerQueryTool(mcpServer)
	s.registerIndexTool(mcpServer)
	s.registerDependsTool(mcpServer)
	s.registerComponentsTool(mcpServer)
	s.registerGraphTool(mcpServer)
	s.registerContextTool(mcpServer)
	s.registerGraphCrawlTool(mcpServer)
	s.registerRelationshipsValidateTool(mcpServer)
	s.registerWatchStartTool(mcpServer)
	s.registerWatchPollTool(mcpServer)
	s.registerWatchStopTool(mcpServer)
	// Component-scoped debugging tools (Phase 22).
	s.registerComponentListTool(mcpServer)
	s.registerComponentGraphTool(mcpServer)
	s.registerDebugComponentContextTool(mcpServer)

	return server.ServeStdio(mcpServer)
}

// registerQueryTool registers the bmd/query tool for full-text and semantic search.
func (s *Server) registerQueryTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/query",
		mcpsdk.WithDescription("Search documentation using BM25 full-text or PageIndex semantic search. Returns ranked results with file paths, titles, and content snippets."),
		mcpsdk.WithString("query",
			mcpsdk.Required(),
			mcpsdk.Description("The search query term or phrase"),
		),
		mcpsdk.WithString("strategy",
			mcpsdk.Description("Search strategy: 'bm25' (default, fast) or 'pageindex' (semantic, requires tree files)"),
		),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory to search (default: configured baseDir)"),
		),
		mcpsdk.WithNumber("top",
			mcpsdk.Description("Maximum number of results to return (default: 10)"),
		),
	)

	mcpServer.AddTool(tool, s.handleQuery)
}

// registerIndexTool registers the bmd/index tool for indexing documentation.
func (s *Server) registerIndexTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/index",
		mcpsdk.WithDescription("Index a documentation directory for search. Builds BM25 index and knowledge graph. Optionally generates PageIndex semantic trees."),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory to index (default: configured baseDir)"),
		),
		mcpsdk.WithString("strategy",
			mcpsdk.Description("Indexing strategy: '' or 'bm25' (default) or 'pageindex' (generates semantic tree files)"),
		),
		mcpsdk.WithString("model",
			mcpsdk.Description("LLM model for pageindex strategy (default: claude-sonnet-4-5)"),
		),
	)

	mcpServer.AddTool(tool, s.handleIndex)
}

// registerDependsTool registers the bmd/depends tool for service dependency queries.
func (s *Server) registerDependsTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/depends",
		mcpsdk.WithDescription("Query service dependencies from the knowledge graph. Shows what services a given service depends on."),
		mcpsdk.WithString("service",
			mcpsdk.Required(),
			mcpsdk.Description("The service name to query dependencies for"),
		),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory that was indexed (default: configured baseDir)"),
		),
		mcpsdk.WithBoolean("transitive",
			mcpsdk.Description("Include transitive (indirect) dependencies (default: false)"),
		),
	)

	mcpServer.AddTool(tool, s.handleDepends)
}

// registerComponentsTool registers the bmd/components tool for listing detected components.
func (s *Server) registerComponentsTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/components",
		mcpsdk.WithDescription("List all components detected in the documentation knowledge graph, with confidence scores and dependency counts."),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory that was indexed (default: configured baseDir)"),
		),
	)

	mcpServer.AddTool(tool, s.handleComponents)
}

// registerGraphTool registers the bmd/graph tool for exporting dependency graphs.
func (s *Server) registerGraphTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/graph",
		mcpsdk.WithDescription("Export the documentation knowledge graph as JSON. Optionally filter to a subgraph for a specific service."),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory that was indexed (default: configured baseDir)"),
		),
		mcpsdk.WithString("service",
			mcpsdk.Description("Export subgraph for this service only (optional)"),
		),
	)

	mcpServer.AddTool(tool, s.handleGraph)
}

// registerContextTool registers the bmd/context tool for RAG context assembly.
func (s *Server) registerContextTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/context",
		mcpsdk.WithDescription("Assemble a RAG-ready context block for a query. Returns the most relevant documentation sections formatted for LLM prompt injection."),
		mcpsdk.WithString("query",
			mcpsdk.Required(),
			mcpsdk.Description("The query to retrieve context for"),
		),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory to search (default: configured baseDir)"),
		),
		mcpsdk.WithNumber("top",
			mcpsdk.Description("Maximum number of sections to return (default: 5)"),
		),
		mcpsdk.WithString("format",
			mcpsdk.Description("Output format: 'markdown' (default) or 'json'"),
		),
	)

	mcpServer.AddTool(tool, s.handleContext)
}

// registerWatchStartTool registers the bmd/watch_start tool for starting filesystem watches.
func (s *Server) registerWatchStartTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/watch_start",
		mcpsdk.WithDescription("Start watching a directory for documentation changes. Returns a session_id for use with bmd/watch_poll and bmd/watch_stop."),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory to watch (default: configured baseDir)"),
		),
		mcpsdk.WithNumber("interval_ms",
			mcpsdk.Description("Poll interval in milliseconds (default: 500)"),
		),
	)
	mcpServer.AddTool(tool, s.handleWatchStart)
}

// registerWatchPollTool registers the bmd/watch_poll tool for polling change notifications.
func (s *Server) registerWatchPollTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/watch_poll",
		mcpsdk.WithDescription("Poll for pending graph change notifications from an active watch session. Returns all changes since last poll."),
		mcpsdk.WithString("session_id",
			mcpsdk.Required(),
			mcpsdk.Description("The watch session ID returned by bmd/watch_start"),
		),
	)
	mcpServer.AddTool(tool, s.handleWatchPoll)
}

// registerWatchStopTool registers the bmd/watch_stop tool for stopping filesystem watches.
func (s *Server) registerWatchStopTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/watch_stop",
		mcpsdk.WithDescription("Stop an active watch session and release filesystem resources."),
		mcpsdk.WithString("session_id",
			mcpsdk.Required(),
			mcpsdk.Description("The watch session ID returned by bmd/watch_start"),
		),
	)
	mcpServer.AddTool(tool, s.handleWatchStop)
}

// registerGraphCrawlTool registers the bmd/graph_crawl tool for multi-start graph traversal.
func (s *Server) registerGraphCrawlTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/graph_crawl",
		mcpsdk.WithDescription("Traverse the knowledge graph from one or more starting files, expanding all branches. Returns discovered nodes, edges, and optionally detected cycles. Useful for understanding dependency chains and impact analysis."),
		mcpsdk.WithString("start_files",
			mcpsdk.Required(),
			mcpsdk.Description("Comma-separated list of starting file paths (relative to indexed directory, e.g. 'api-gateway.md,auth-service.md')"),
		),
		mcpsdk.WithString("direction",
			mcpsdk.Description("Traversal direction: 'forward' (outgoing edges, default), 'backward' (incoming edges), or 'both'"),
		),
		mcpsdk.WithNumber("depth",
			mcpsdk.Description("Maximum traversal depth in hops (default: 10, -1 for unlimited)"),
		),
		mcpsdk.WithBoolean("include_cycles",
			mcpsdk.Description("Include cycle detection in the response (default: false)"),
		),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory that was indexed (default: configured baseDir)"),
		),
	)

	mcpServer.AddTool(tool, s.handleGraphCrawl)
}

// registerRelationshipsValidateTool registers the bmd/relationships_validate tool for LLM-powered validation.
func (s *Server) registerRelationshipsValidateTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/relationships_validate",
		mcpsdk.WithDescription("Validate pending relationships in the discovered manifest via LLM subprocess. Each pending relationship is assessed for confidence and can be auto-accepted or auto-rejected based on thresholds."),
		mcpsdk.WithString("dir",
			mcpsdk.Description("Directory containing .bmd-relationships-discovered.yaml (default: configured baseDir)"),
		),
		mcpsdk.WithString("llm_model",
			mcpsdk.Description("LLM model for validation (default: claude-sonnet-4-5)"),
		),
		mcpsdk.WithNumber("auto_accept_threshold",
			mcpsdk.Description("Auto-accept if LLM confidence >= threshold (0.0 = off, default: 0.0)"),
		),
		mcpsdk.WithNumber("auto_reject_threshold",
			mcpsdk.Description("Auto-reject if LLM confidence < threshold (0.0 = off, default: 0.0)"),
		),
	)

	mcpServer.AddTool(tool, s.handleRelationshipsValidate)
}
