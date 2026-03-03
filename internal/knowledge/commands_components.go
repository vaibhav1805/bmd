package knowledge

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// ─── argument structs ──────────────────────────────────────────────────────────

// DebugArgs holds parsed arguments for CmdDebug.
type DebugArgs struct {
	// Component is the target component name to debug (required).
	Component string

	// Query is an optional description of what is being debugged.
	Query string

	// Dir is the root directory of the monorepo to scan.
	Dir string

	// Depth is the BFS traversal depth (1-5, default: 2).
	Depth int

	// Output controls output format: "json" (default) | "text".
	Output string
}

// ComponentsGraphArgs holds parsed arguments for CmdComponentsGraph.
type ComponentsGraphArgs struct {
	Dir    string
	Format string // "ascii" | "json"
}

// ─── argument parsers ─────────────────────────────────────────────────────────

// ParseDebugArgs parses raw CLI arguments for the debug command.
//
// Usage: bmd debug --component NAME [--query QUERY] [--dir DIR] [--depth N] [--format json|text]
func ParseDebugArgs(args []string) (*DebugArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("debug", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a DebugArgs
	fs.StringVar(&a.Component, "component", "", "Component name to debug")
	fs.StringVar(&a.Query, "query", "", "Optional query / problem description")
	fs.StringVar(&a.Dir, "dir", ".", "Root directory of the monorepo")
	fs.IntVar(&a.Depth, "depth", 2, "BFS traversal depth")
	fs.StringVar(&a.Output, "format", "json", "Output format (json|text)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("debug: %w", err)
	}

	// Allow positional argument as component name.
	if len(positionals) > 0 && a.Component == "" {
		a.Component = positionals[0]
	}

	if a.Component == "" {
		return nil, fmt.Errorf("debug: --component is required")
	}

	return &a, nil
}

// CmdDebug implements `bmd debug --component NAME`.
//
// Builds a ComponentGraph, runs BFS from the named component, and outputs a
// DebugContext JSON payload (STATUS-01 compliant) or a text summary.
func CmdDebug(args []string) error {
	a, err := ParseDebugArgs(args)
	if err != nil {
		return err
	}

	isJSON := strings.ToLower(a.Output) == "json"

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("debug: resolve dir: %w", err)
	}

	cg, err := BuildComponentGraphFromConfig(absDir, nil)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("debug: build component graph: %w", err)
	}

	if _, ok := cg.Components[a.Component]; !ok {
		msg := fmt.Sprintf("component %q not found; available: %s", a.Component, listComponentNames(cg))
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeFileNotFound, msg)))
			return nil
		}
		return fmt.Errorf("debug: %s", msg)
	}

	bfs, bfsErr := NewBFS(cg, a.Component, absDir)
	if bfsErr != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, bfsErr.Error())))
			return nil
		}
		return fmt.Errorf("debug: init BFS: %w", bfsErr)
	}

	const maxDocBytes = 1 << 20 // 1 MB cap on aggregated documentation
	if travErr := bfs.Traverse(a.Depth, maxDocBytes); travErr != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, travErr.Error())))
			return nil
		}
		return fmt.Errorf("debug: traverse: %w", travErr)
	}

	dc := bfs.BuildDebugContext(a.Component, a.Query)

	jsonBytes, marshalErr := dc.ToJSON()
	if marshalErr != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, marshalErr.Error())))
			return nil
		}
		return fmt.Errorf("debug: marshal output: %w", marshalErr)
	}

	if isJSON {
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Text summary.
	fmt.Printf("Component: %s\n", dc.TargetComponent)
	if dc.QueryDescription != "" {
		fmt.Printf("Query: %s\n", dc.QueryDescription)
	}
	fmt.Printf("Components visited: %d\n", dc.Stats.ComponentsVisited)
	for _, ci := range dc.Components {
		fmt.Printf("  [%s] %s (distance=%d)\n", ci.Role, ci.Name, ci.DiscoveryDistance)
	}
	return nil
}

// listComponentNames returns a comma-separated list of component names in cg.
func listComponentNames(cg *ComponentGraph) string {
	names := make([]string, 0, len(cg.Components))
	for name := range cg.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// ParseComponentsGraphArgs parses args for CmdComponentsGraph.
//
// Usage: bmd components graph [--dir DIR] [--format ascii|json]
func ParseComponentsGraphArgs(args []string) (*ComponentsGraphArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("components graph", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ComponentsGraphArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to scan for components")
	fs.StringVar(&a.Format, "format", "ascii", "Output format (ascii|json)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("components graph: %w", err)
	}
	if len(positionals) > 0 {
		a.Dir = positionals[0]
	}

	return &a, nil
}

// ─── command implementations ──────────────────────────────────────────────────

// CmdComponentsGraph implements `bmd components graph`.
//
// Builds a component-level dependency graph and prints it in ASCII or JSON format.
//
// ASCII output example:
//
//	payment → auth (0.90)
//	payment → user (0.80)
//	auth → user (1.00)
//
// JSON output: full ComponentGraph structure with confidence scores wrapped in
// a STATUS-01 contract envelope.
func CmdComponentsGraph(args []string) error {
	a, err := ParseComponentsGraphArgs(args)
	if err != nil {
		return err
	}

	isJSON := strings.ToLower(a.Format) == "json"

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("components graph: resolve dir: %w", err)
	}

	cg, err := BuildComponentGraphFromConfig(absDir, nil)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("components graph: %w", err)
	}

	if !isJSON {
		fmt.Println(formatComponentGraphASCII(cg))
		return nil
	}

	payload := buildComponentGraphJSON(cg)
	fmt.Println(marshalContract(NewOKResponse(
		fmt.Sprintf("Component graph: %d nodes, %d edges", cg.NodeCount(), cg.EdgeCount()),
		payload,
	)))
	return nil
}

// ─── formatters ──────────────────────────────────────────────────────────────

// formatComponentGraphASCII renders the component graph in simple edge notation.
// Each line shows: "from → to (confidence)"
// Edges are sorted by from-name then to-name for deterministic output.
// When the graph has no edges, a summary line is returned instead.
func formatComponentGraphASCII(cg *ComponentGraph) string {
	if cg.EdgeCount() == 0 {
		return fmt.Sprintf("Component graph: %d components, no edges discovered.", cg.NodeCount())
	}

	// Collect all edges.
	type edge struct {
		from, to   string
		confidence float64
	}
	var edges []edge
	for fromName, adj := range cg.Edges {
		for toName, e := range adj {
			edges = append(edges, edge{from: fromName, to: toName, confidence: e.Confidence})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].from != edges[j].from {
			return edges[i].from < edges[j].from
		}
		return edges[i].to < edges[j].to
	})

	var sb strings.Builder
	for _, e := range edges {
		fmt.Fprintf(&sb, "%s → %s (%.2f)\n", e.from, e.to, e.confidence)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// componentGraphNodeJSON is the per-node structure in JSON graph output.
type componentGraphNodeJSON struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Files     []string `json:"files"`
	InDegree  int      `json:"in_degree"`
	OutDegree int      `json:"out_degree"`
}

// componentGraphEdgeJSON is the per-edge structure in JSON graph output.
type componentGraphEdgeJSON struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	Type       string   `json:"type"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence,omitempty"`
}

// componentGraphPayload is the JSON payload for a component graph response.
type componentGraphPayload struct {
	Nodes []componentGraphNodeJSON `json:"nodes"`
	Edges []componentGraphEdgeJSON `json:"edges"`
	Stats struct {
		NodeCount int `json:"node_count"`
		EdgeCount int `json:"edge_count"`
	} `json:"stats"`
}

// buildComponentGraphJSON converts a ComponentGraph into a JSON-serialisable payload.
func buildComponentGraphJSON(cg *ComponentGraph) componentGraphPayload {
	// Collect sorted node names for deterministic output.
	nodeNames := make([]string, 0, len(cg.Nodes))
	for name := range cg.Nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	nodes := make([]componentGraphNodeJSON, 0, len(nodeNames))
	for _, name := range nodeNames {
		n := cg.Nodes[name]
		files := n.Files
		if files == nil {
			files = []string{}
		}
		nodes = append(nodes, componentGraphNodeJSON{
			Name:      n.Name,
			Path:      n.Path,
			Files:     files,
			InDegree:  n.InDegree,
			OutDegree: n.OutDegree,
		})
	}

	// Collect sorted edges for deterministic output.
	type rawEdge struct {
		from string
		e    *ComponentEdge
	}
	var rawEdges []rawEdge
	for fromName, adj := range cg.Edges {
		for _, e := range adj {
			rawEdges = append(rawEdges, rawEdge{from: fromName, e: e})
		}
	}
	sort.Slice(rawEdges, func(i, j int) bool {
		if rawEdges[i].from != rawEdges[j].from {
			return rawEdges[i].from < rawEdges[j].from
		}
		return rawEdges[i].e.To < rawEdges[j].e.To
	})

	edges := make([]componentGraphEdgeJSON, 0, len(rawEdges))
	for _, re := range rawEdges {
		ev := re.e.Evidence
		if ev == nil {
			ev = []string{}
		}
		edges = append(edges, componentGraphEdgeJSON{
			From:       re.from,
			To:         re.e.To,
			Type:       re.e.Type,
			Confidence: roundFloat(re.e.Confidence, 4),
			Evidence:   ev,
		})
	}

	var payload componentGraphPayload
	payload.Nodes = nodes
	payload.Edges = edges
	payload.Stats.NodeCount = cg.NodeCount()
	payload.Stats.EdgeCount = cg.EdgeCount()
	return payload
}

// marshalComponentGraphJSON serializes a componentGraphPayload to indented JSON.
// Exported for test use.
func marshalComponentGraphJSON(payload componentGraphPayload) ([]byte, error) {
	return json.MarshalIndent(payload, "", "  ")
}
