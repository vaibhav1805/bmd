package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/bmd/bmd/internal/knowledge"
)

// registerComponentListTool registers the bmd/component_list tool for listing
// all discovered components in a codebase.
func (s *Server) registerComponentListTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/component_list",
		mcpsdk.WithDescription("List all discovered components in the codebase. Returns component names, paths, files, and discovery method."),
		mcpsdk.WithBoolean("include_hidden",
			mcpsdk.Description("Include hidden components (default: false)"),
		),
		mcpsdk.WithString("root_dir",
			mcpsdk.Description("Root directory to scan for components (default: .)"),
		),
	)

	mcpServer.AddTool(tool, s.handleComponentList)
}

// registerComponentGraphTool registers the bmd/component_graph tool for
// building and visualizing the dependency graph between components.
func (s *Server) registerComponentGraphTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/component_graph",
		mcpsdk.WithDescription("Build and visualize the dependency graph between components. Returns graph structure with nodes and edges."),
		mcpsdk.WithString("format",
			mcpsdk.Description("Output format: 'json' (default) or 'ascii'"),
		),
		mcpsdk.WithString("root_dir",
			mcpsdk.Description("Root directory to scan for components (default: .)"),
		),
	)

	mcpServer.AddTool(tool, s.handleComponentGraph)
}

// registerDebugComponentContextTool registers the bmd/debug_component_context
// tool for aggregating documentation and relationships for debugging a component.
func (s *Server) registerDebugComponentContextTool(mcpServer *server.MCPServer) {
	tool := mcpsdk.NewTool(
		"bmd/debug_component_context",
		mcpsdk.WithDescription("Get aggregated documentation and relationships for debugging a specific component. Returns STATUS-01 compliant DebugContext with docs from the component and all related components discovered via BFS traversal."),
		mcpsdk.WithString("component",
			mcpsdk.Required(),
			mcpsdk.Description("Component name to debug"),
		),
		mcpsdk.WithString("query",
			mcpsdk.Description("What are you debugging? (optional context for retrieval)"),
		),
		mcpsdk.WithNumber("depth",
			mcpsdk.Description("BFS traversal depth for related components (1-5, default: 2)"),
		),
		mcpsdk.WithString("root_dir",
			mcpsdk.Description("Root directory to scan for components (default: .)"),
		),
	)

	mcpServer.AddTool(tool, s.handleDebugComponentContext)
}

// handleComponentList handles the bmd/component_list MCP tool invocation.
// It calls CmdComponentsList and returns a JSON list of all discovered components.
func (s *Server) handleComponentList(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	rootDir := mcpsdk.ParseString(req, "root_dir", s.baseDir)
	if rootDir == "" {
		rootDir = s.baseDir
	}

	args := []string{"--dir", rootDir, "--format", "json"}

	output, err := captureOutput(func() error {
		return knowledge.CmdComponentsList(args)
	})
	if err != nil {
		resp := knowledge.NewErrorResponse(knowledge.ErrCodeInternalError, fmt.Sprintf("component_list failed: %v", err))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// handleComponentGraph handles the bmd/component_graph MCP tool invocation.
// It calls CmdGraph and returns the formatted component dependency graph.
func (s *Server) handleComponentGraph(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	format := mcpsdk.ParseString(req, "format", "json")
	rootDir := mcpsdk.ParseString(req, "root_dir", s.baseDir)
	if rootDir == "" {
		rootDir = s.baseDir
	}

	// Validate format.
	switch format {
	case "json", "ascii", "":
		// valid; empty falls back to json
	default:
		resp := knowledge.NewErrorResponse("INVALID_PARAMS", fmt.Sprintf("invalid format %q: must be 'json' or 'ascii'", format))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	if format == "" || format == "ascii" {
		// CmdGraph uses "dot" for non-JSON; treat ascii as a text table output.
		// Fall through to CmdGraph with format passthrough.
		args := []string{"--dir", rootDir, "--format", format}
		output, err := captureOutput(func() error {
			return knowledge.CmdGraph(args)
		})
		if err != nil {
			resp := knowledge.NewErrorResponse(knowledge.ErrCodeInternalError, fmt.Sprintf("component_graph failed: %v", err))
			data, _ := json.MarshalIndent(resp, "", "  ")
			return mcpsdk.NewToolResultText(string(data)), nil
		}
		return mcpsdk.NewToolResultText(output), nil
	}

	// Default: JSON format.
	args := []string{"--dir", rootDir, "--format", "json"}

	output, err := captureOutput(func() error {
		return knowledge.CmdGraph(args)
	})
	if err != nil {
		resp := knowledge.NewErrorResponse(knowledge.ErrCodeInternalError, fmt.Sprintf("component_graph failed: %v", err))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}

// handleDebugComponentContext handles the bmd/debug_component_context MCP tool invocation.
// It aggregates documentation and relationships for a specific component using BFS traversal,
// returning a STATUS-01 compliant DebugContext for agent troubleshooting.
func (s *Server) handleDebugComponentContext(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	component := mcpsdk.ParseString(req, "component", "")
	if component == "" {
		return mcpsdk.NewToolResultError("component parameter is required"), nil
	}

	query := mcpsdk.ParseString(req, "query", "")
	depth := mcpsdk.ParseInt(req, "depth", 2)
	rootDir := mcpsdk.ParseString(req, "root_dir", s.baseDir)
	if rootDir == "" {
		rootDir = s.baseDir
	}

	// Validate depth bounds.
	if depth < 1 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}

	args := []string{
		"--component", component,
		"--dir", rootDir,
		"--depth", fmt.Sprintf("%d", depth),
		"--format", "json",
	}
	if query != "" {
		args = append(args, "--query", query)
	}

	output, err := captureOutput(func() error {
		return knowledge.CmdDebug(args)
	})
	if err != nil {
		resp := knowledge.NewErrorResponse(knowledge.ErrCodeInternalError, fmt.Sprintf("debug_component_context failed: %v", err))
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcpsdk.NewToolResultText(string(data)), nil
	}

	return mcpsdk.NewToolResultText(output), nil
}
