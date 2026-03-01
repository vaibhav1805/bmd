package knowledge

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ─── agent contract types ─────────────────────────────────────────────────────

// Error code constants for ContractResponse.Code.
const (
	ErrCodeIndexNotFound       = "INDEX_NOT_FOUND"
	ErrCodeFileNotFound        = "FILE_NOT_FOUND"
	ErrCodeInvalidQuery        = "INVALID_QUERY"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodePageIndexNotAvailable = "PAGEINDEX_NOT_AVAILABLE"
)

// reasoningResultJSON is the per-section output when strategy=pageindex.
// It extends searchResultJSON with a reasoning_trace field explaining why
// the section was selected by the LLM.
type reasoningResultJSON struct {
	Rank           int     `json:"rank"`
	File           string  `json:"file"`
	HeadingPath    string  `json:"heading_path,omitempty"`
	StartLine      int     `json:"start_line,omitempty"`
	EndLine        int     `json:"end_line,omitempty"`
	ContentPreview string  `json:"content_preview,omitempty"`
	Score          float64 `json:"score"`
	ReasoningTrace string  `json:"reasoning_trace,omitempty"`
}

// pageindexResponseJSON is the data payload for strategy=pageindex results.
// It is wrapped in a ContractResponse envelope (CONTRACT-01).
type pageindexResponseJSON struct {
	Query       string                `json:"query"`
	Strategy    string                `json:"strategy"`
	Model       string                `json:"model"`
	Results     []reasoningResultJSON `json:"results"`
	Count       int                   `json:"count"`
	QueryTimeMs int64                 `json:"query_time_ms"`
}

// ContractResponse is the top-level JSON envelope for all agent-facing commands.
// status: "ok" | "error" | "empty"
// code: nil when status="ok"/"empty"; a string constant from ErrCode* when status="error"
// message: human-readable summary
// data: command-specific payload; nil when status="error"
type ContractResponse struct {
	Status  string      `json:"status"`
	Code    *string     `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// NewOKResponse constructs a ContractResponse with status="ok" and nil code.
func NewOKResponse(message string, data interface{}) ContractResponse {
	return ContractResponse{Status: "ok", Code: nil, Message: message, Data: data}
}

// NewEmptyResponse constructs a ContractResponse with status="empty" and nil code.
func NewEmptyResponse(message string, data interface{}) ContractResponse {
	return ContractResponse{Status: "empty", Code: nil, Message: message, Data: data}
}

// NewErrorResponse constructs a ContractResponse with status="error" and the given code.
func NewErrorResponse(code, message string) ContractResponse {
	c := code
	return ContractResponse{Status: "error", Code: &c, Message: message, Data: nil}
}

// marshalContract serializes a ContractResponse to indented JSON.
// Error is silenced because ContractResponse contains only JSON-safe types.
func marshalContract(resp ContractResponse) string {
	data, _ := json.MarshalIndent(resp, "", "  ")
	return string(data)
}

// FormatSearchResults formats a slice of SearchResult values in the requested
// output format.  Supported formats: "json" (default), "text", "csv".
// Unknown formats fall back to JSON.
func FormatSearchResults(results []SearchResult, query string, format string, queryTimeMs int64) string {
	switch strings.ToLower(format) {
	case "text":
		return formatSearchResultsText(results)
	case "csv":
		return formatSearchResultsCSV(results)
	default:
		return formatSearchResultsJSON(results, query, queryTimeMs)
	}
}

// FormatServices formats a slice of Service values in the requested output
// format.  Supported formats: "json" (default), "text".
func FormatServices(services []Service, depCounts map[string]int, format string) string {
	switch strings.ToLower(format) {
	case "text":
		return formatServicesText(services, depCounts)
	default:
		return formatServicesJSON(services, depCounts)
	}
}

// FormatDependencies formats dependency information in the requested output
// format.  Supported formats: "json" (default), "text", "dot".
// When transitive is true, transitivePaths is expected; otherwise refs is used.
func FormatDependencies(
	serviceID string,
	refs []ServiceRef,
	transitive bool,
	transitivePaths [][]string,
	cycles [][]string,
	format string,
) string {
	switch strings.ToLower(format) {
	case "text":
		return formatDependenciesText(serviceID, refs, transitive, transitivePaths, cycles)
	case "dot":
		return formatDependenciesDOT(serviceID, refs, transitivePaths)
	default:
		return formatDependenciesJSON(serviceID, refs, transitive, transitivePaths, cycles)
	}
}

// FormatGraph formats a knowledge Graph in the requested output format.
// Supported formats: "dot" (default), "json".
func FormatGraph(graph *Graph, format string) string {
	switch strings.ToLower(format) {
	case "json":
		return formatGraphJSON(graph)
	default:
		return formatGraphDOT(graph)
	}
}

// ─── search result formatters ─────────────────────────────────────────────────

type searchResultJSON struct {
	Rank           int     `json:"rank"`
	File           string  `json:"file"`
	Title          string  `json:"title"`
	Score          float64 `json:"score"`
	Snippet        string  `json:"snippet"`
	HeadingPath    string  `json:"heading_path,omitempty"`
	StartLine      int     `json:"start_line,omitempty"`
	EndLine        int     `json:"end_line,omitempty"`
	ContentPreview string  `json:"content_preview,omitempty"`
}

type searchResponseJSON struct {
	Query       string             `json:"query"`
	Results     []searchResultJSON `json:"results"`
	Count       int                `json:"count"`
	QueryTimeMs int64              `json:"query_time_ms"`
}

func formatSearchResultsJSON(results []SearchResult, query string, queryTimeMs int64) string {
	items := make([]searchResultJSON, len(results))
	for i, r := range results {
		items[i] = searchResultJSON{
			Rank:           i + 1,
			File:           r.RelPath,
			Title:          r.Title,
			Score:          roundFloat(r.Score, 4),
			Snippet:        r.Snippet,
			HeadingPath:    r.HeadingPath,
			StartLine:      r.StartLine,
			EndLine:        r.EndLine,
			ContentPreview: r.ContentPreview,
		}
	}
	resp := searchResponseJSON{
		Query:       query,
		Results:     items,
		Count:       len(items),
		QueryTimeMs: queryTimeMs,
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return string(data)
}

func formatSearchResultsText(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}
	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "%d. %s (score: %.4f)\n", i+1, r.RelPath, r.Score)
		if r.Title != "" {
			fmt.Fprintf(&sb, "   %s\n", r.Title)
		}
		if r.Snippet != "" {
			snippet := r.Snippet
			if len([]rune(snippet)) > 120 {
				runes := []rune(snippet)
				snippet = string(runes[:120]) + "..."
			}
			fmt.Fprintf(&sb, "   %s\n", snippet)
		}
		if i < len(results)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func formatSearchResultsCSV(results []SearchResult) string {
	var sb strings.Builder
	sb.WriteString("rank|file|title|score|snippet\n")
	for i, r := range results {
		snippet := strings.ReplaceAll(r.Snippet, "|", "/")
		snippet = strings.ReplaceAll(snippet, "\n", " ")
		title := strings.ReplaceAll(r.Title, "|", "/")
		fmt.Fprintf(&sb, "%d|%s|%s|%.4f|%s\n", i+1, r.RelPath, title, r.Score, snippet)
	}
	return sb.String()
}

// ─── services formatters ──────────────────────────────────────────────────────

type serviceEntryJSON struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	File            string  `json:"file"`
	Confidence      float64 `json:"confidence"`
	DependencyCount int     `json:"dependency_count"`
}

type servicesResponseJSON struct {
	Services []serviceEntryJSON `json:"services"`
}

func formatServicesJSON(services []Service, depCounts map[string]int) string {
	items := make([]serviceEntryJSON, len(services))
	for i, s := range services {
		cnt := 0
		if depCounts != nil {
			cnt = depCounts[s.ID]
		}
		items[i] = serviceEntryJSON{
			ID:              s.ID,
			Name:            s.Name,
			File:            s.File,
			Confidence:      roundFloat(s.Confidence, 2),
			DependencyCount: cnt,
		}
	}
	resp := servicesResponseJSON{Services: items}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return string(data)
}

func formatServicesText(services []Service, depCounts map[string]int) string {
	if len(services) == 0 {
		return "No services detected."
	}
	var sb strings.Builder
	for i, s := range services {
		cnt := 0
		if depCounts != nil {
			cnt = depCounts[s.ID]
		}
		fmt.Fprintf(&sb, "%s (%.2f)\n", s.ID, s.Confidence)
		if s.File != "" {
			fmt.Fprintf(&sb, "  File: %s\n", s.File)
		}
		fmt.Fprintf(&sb, "  Dependencies: %d\n", cnt)
		if i < len(services)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// ─── dependency formatters ────────────────────────────────────────────────────

type depRefJSON struct {
	Service    string  `json:"service"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

type depTransitiveJSON struct {
	Path     []string `json:"path"`
	Distance int      `json:"distance"`
}

type depsDirectResponseJSON struct {
	Service      string       `json:"service"`
	Dependencies []depRefJSON `json:"dependencies"`
	Cycles       [][]string   `json:"cycles,omitempty"`
}

type depsTransitiveResponseJSON struct {
	Service                 string              `json:"service"`
	TransitiveDependencies  []depTransitiveJSON `json:"transitive_dependencies"`
	Cycles                  [][]string          `json:"cycles,omitempty"`
}

func formatDependenciesJSON(
	serviceID string,
	refs []ServiceRef,
	transitive bool,
	transitivePaths [][]string,
	cycles [][]string,
) string {
	var cleanCycles [][]string
	if len(cycles) > 0 {
		cleanCycles = cycles
	}

	if transitive {
		items := make([]depTransitiveJSON, 0, len(transitivePaths))
		for _, path := range transitivePaths {
			items = append(items, depTransitiveJSON{
				Path:     path,
				Distance: len(path) - 1,
			})
		}
		resp := depsTransitiveResponseJSON{
			Service:                serviceID,
			TransitiveDependencies: items,
			Cycles:                 cleanCycles,
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		return string(data)
	}

	items := make([]depRefJSON, len(refs))
	for i, r := range refs {
		items[i] = depRefJSON{
			Service:    r.ServiceID,
			Type:       r.Type,
			Confidence: roundFloat(r.Confidence, 2),
		}
	}
	resp := depsDirectResponseJSON{
		Service:      serviceID,
		Dependencies: items,
		Cycles:       cleanCycles,
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return string(data)
}

func formatDependenciesText(
	serviceID string,
	refs []ServiceRef,
	transitive bool,
	transitivePaths [][]string,
	cycles [][]string,
) string {
	var sb strings.Builder

	if transitive {
		fmt.Fprintf(&sb, "%s (transitive)\n", serviceID)
		if len(transitivePaths) == 0 {
			sb.WriteString("  No transitive dependencies found.\n")
		}
		for _, path := range transitivePaths {
			indent := ""
			for i, node := range path {
				if i == 0 {
					// serviceID already printed
					continue
				}
				fmt.Fprintf(&sb, "%s-> %s\n", indent, node)
				indent += "  "
			}
		}
	} else {
		fmt.Fprintf(&sb, "%s\n", serviceID)
		if len(refs) == 0 {
			sb.WriteString("  No dependencies found.\n")
		}
		for _, r := range refs {
			fmt.Fprintf(&sb, "  -> %s (%.2f)\n", r.ServiceID, r.Confidence)
		}
	}

	if len(cycles) > 0 {
		sb.WriteString("\nCycles detected:\n")
		for _, cycle := range cycles {
			fmt.Fprintf(&sb, "  %s\n", strings.Join(cycle, " -> "))
		}
	}

	return sb.String()
}

func formatDependenciesDOT(serviceID string, refs []ServiceRef, transitivePaths [][]string) string {
	var sb strings.Builder
	sb.WriteString("digraph dependencies {\n")
	fmt.Fprintf(&sb, "  %q;\n", serviceID)
	added := make(map[string]bool)
	if len(transitivePaths) > 0 {
		for _, path := range transitivePaths {
			for i := 0; i+1 < len(path); i++ {
				key := path[i] + "->" + path[i+1]
				if !added[key] {
					added[key] = true
					fmt.Fprintf(&sb, "  %q -> %q;\n", path[i], path[i+1])
				}
			}
		}
	} else {
		for _, r := range refs {
			key := serviceID + "->" + r.ServiceID
			if !added[key] {
				added[key] = true
				fmt.Fprintf(&sb, "  %q -> %q [label=%q];\n", serviceID, r.ServiceID, r.Type)
			}
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

// ─── graph formatters ─────────────────────────────────────────────────────────

type graphNodeJSON struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type graphEdgeJSON struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

type graphResponseJSON struct {
	Nodes []graphNodeJSON `json:"nodes"`
	Edges []graphEdgeJSON `json:"edges"`
}

func formatGraphJSON(graph *Graph) string {
	// Sort nodes and edges for deterministic output.
	nodeIDs := make([]string, 0, len(graph.Nodes))
	for id := range graph.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	edgeIDs := make([]string, 0, len(graph.Edges))
	for id := range graph.Edges {
		edgeIDs = append(edgeIDs, id)
	}
	sort.Strings(edgeIDs)

	nodes := make([]graphNodeJSON, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		n := graph.Nodes[id]
		label := n.Title
		if label == "" {
			label = n.ID
		}
		nodes = append(nodes, graphNodeJSON{
			ID:    n.ID,
			Type:  n.Type,
			Label: label,
		})
	}

	edges := make([]graphEdgeJSON, 0, len(edgeIDs))
	for _, id := range edgeIDs {
		e := graph.Edges[id]
		edges = append(edges, graphEdgeJSON{
			Source:     e.Source,
			Target:     e.Target,
			Type:       string(e.Type),
			Confidence: roundFloat(e.Confidence, 4),
		})
	}

	resp := graphResponseJSON{Nodes: nodes, Edges: edges}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return string(data)
}

func formatGraphDOT(graph *Graph) string {
	var sb strings.Builder
	sb.WriteString("digraph knowledge_graph {\n")

	// Sort for deterministic output.
	nodeIDs := make([]string, 0, len(graph.Nodes))
	for id := range graph.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		n := graph.Nodes[id]
		label := n.Title
		if label == "" {
			label = n.ID
		}
		fmt.Fprintf(&sb, "  %q [label=%q];\n", n.ID, label)
	}

	edgeIDs := make([]string, 0, len(graph.Edges))
	for id := range graph.Edges {
		edgeIDs = append(edgeIDs, id)
	}
	sort.Strings(edgeIDs)

	for _, id := range edgeIDs {
		e := graph.Edges[id]
		fmt.Fprintf(&sb, "  %q -> %q [label=%q, weight=\"%.2f\"];\n",
			e.Source, e.Target, string(e.Type), e.Confidence)
	}

	sb.WriteString("}\n")
	return sb.String()
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// roundFloat rounds f to the given number of decimal places.
func roundFloat(f float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}
