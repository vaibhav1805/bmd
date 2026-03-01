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
}

// NewServer creates a new MCP server configured for the given documentation directory.
func NewServer(baseDir, dbPath string) *Server {
	return &Server{baseDir: baseDir, dbPath: dbPath}
}

// Start initializes and runs the MCP server on stdin/stdout.
// It registers all knowledge tools and blocks until the process is killed.
func (s *Server) Start(ctx context.Context) error {
	mcpServer := server.NewMCPServer(
		"bmd",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register all 6 knowledge tools.
	s.registerQueryTool(mcpServer)
	s.registerIndexTool(mcpServer)
	s.registerDependsTool(mcpServer)
	s.registerComponentsTool(mcpServer)
	s.registerGraphTool(mcpServer)
	s.registerContextTool(mcpServer)

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
