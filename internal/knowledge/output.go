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
	Content        string  `json:"content,omitempty"`
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
		// penwidth: scale confidence [0.0–1.0] to line thickness [0.5–3.0].
		// Higher confidence produces a thicker edge in Graphviz DOT renderers.
		penwidth := 0.5 + e.Confidence*2.5
		fmt.Fprintf(&sb, "  %q -> %q [label=%q, weight=\"%.2f\", penwidth=\"%.2f\"];\n",
			e.Source, e.Target, string(e.Type), e.Confidence, penwidth)
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
