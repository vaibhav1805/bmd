package knowledge

import (
	"sort"
	"strings"
)

// ServiceRef describes a dependency that one service has on another.
// It captures not only the target service but also how the dependency was
// discovered and how confident the system is in it.
type ServiceRef struct {
	// ServiceID is the ID of the depended-upon service.
	ServiceID string

	// Type describes the nature of the dependency.  Common values:
	//   "direct-call" — synchronous RPC / HTTP call
	//   "queue"       — asynchronous message through a queue
	//   "database"    — shared database dependency
	//   "cache"       — shared cache layer
	//   "reference"   — document link / generic reference
	Type string

	// Evidence is the human-readable source of this dependency relationship
	// (e.g. the edge Evidence string or a markdown link).
	Evidence string

	// Confidence is a [0.0, 1.0] score reflecting extraction certainty.
	Confidence float64
}

// ServiceGraph is a directed dependency graph whose nodes are Services
// (detected or configured) and whose edges represent service-to-service
// dependencies.
type ServiceGraph struct {
	// Services maps service ID → *Service.
	Services map[string]*Service

	// Dependencies maps service ID → list of its outgoing ServiceRefs.
	Dependencies map[string][]ServiceRef
}

// newServiceGraph returns an empty, ready-to-use ServiceGraph.
func newServiceGraph() *ServiceGraph {
	return &ServiceGraph{
		Services:     make(map[string]*Service),
		Dependencies: make(map[string][]ServiceRef),
	}
}

// DependencyChain describes the shortest dependency path between two services.
type DependencyChain struct {
	// Path is the ordered list of service IDs from the source to the
	// destination (inclusive of both endpoints).
	Path []string

	// Distance is the number of hops (len(Path) - 1).  Zero when From == To
	// or no path exists.
	Distance int

	// HasCycle indicates that a cycle was detected along the path search.
	// This flag is informational; the path itself is still the shortest one
	// found before the cycle was encountered.
	HasCycle bool

	// Evidence is a short description of how the path was found (e.g. edge
	// evidence strings joined by " → ").
	Evidence string
}

// DependencyAnalyzer extracts and queries service-to-service dependencies
// from a knowledge Graph and a set of detected Services.
//
// It operates on the service-level view of the graph: only nodes that
// correspond to known services are included in the analysis.
type DependencyAnalyzer struct {
	// serviceGraph is the computed service-level dependency graph.
	serviceGraph *ServiceGraph
}

// NewDependencyAnalyzer creates a DependencyAnalyzer and builds the service
// dependency graph from graph and services.
//
// This is the primary entry point.  All subsequent query methods operate on
// the pre-built ServiceGraph so they run in O(degree) time.
func NewDependencyAnalyzer(graph *Graph, services []Service) *DependencyAnalyzer {
	da := &DependencyAnalyzer{}
	da.serviceGraph = da.BuildServiceGraph(graph, services)
	return da
}

// BuildServiceGraph extracts a service-only subgraph from graph using services
// as the set of known service nodes.
//
// Algorithm:
//  1. Index services by file path (Node ID).
//  2. For each edge in the full graph, check whether both Source and Target
//     map to known services.
//  3. Add qualifying edges to the ServiceGraph with appropriate type/confidence.
func (da *DependencyAnalyzer) BuildServiceGraph(graph *Graph, services []Service) *ServiceGraph {
	sg := newServiceGraph()

	// Index services by their document file path.
	byFile := make(map[string]*Service, len(services))
	for i := range services {
		s := &services[i]
		sg.Services[s.ID] = s
		byFile[s.File] = s
	}

	// Iterate every edge in the full knowledge graph and check whether both
	// endpoints correspond to known services.
	for _, edge := range graph.Edges {
		srcSvc, srcOK := byFile[edge.Source]
		tgtSvc, tgtOK := byFile[edge.Target]
		if !srcOK || !tgtOK {
			continue
		}

		ref := ServiceRef{
			ServiceID:  tgtSvc.ID,
			Type:       edgeTypeToDepType(edge.Type),
			Evidence:   edge.Evidence,
			Confidence: edge.Confidence,
		}

		// Avoid duplicating refs for the same (src, tgt, type) triple.
		if !hasRef(sg.Dependencies[srcSvc.ID], ref) {
			sg.Dependencies[srcSvc.ID] = append(sg.Dependencies[srcSvc.ID], ref)
		}
	}

	return sg
}

// GetDirectDeps returns the IDs of services that serviceID directly depends on.
// Returns nil when serviceID is unknown or has no dependencies.
func (da *DependencyAnalyzer) GetDirectDeps(serviceID string) []string {
	refs, ok := da.serviceGraph.Dependencies[serviceID]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(refs))
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if !seen[ref.ServiceID] {
			seen[ref.ServiceID] = true
			ids = append(ids, ref.ServiceID)
		}
	}
	sort.Strings(ids)
	return ids
}

// GetTransitiveDeps returns the IDs of all services reachable from serviceID
// by following dependency edges (BFS, no depth limit).
//
// The starting service itself is NOT included.  Returns nil when serviceID is
// unknown or has no outgoing dependencies.
func (da *DependencyAnalyzer) GetTransitiveDeps(serviceID string) []string {
	if _, ok := da.serviceGraph.Services[serviceID]; !ok {
		return nil
	}

	visited := make(map[string]bool)
	visited[serviceID] = true
	queue := []string{serviceID}
	var result []string

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, ref := range da.serviceGraph.Dependencies[cur] {
			if !visited[ref.ServiceID] {
				visited[ref.ServiceID] = true
				result = append(result, ref.ServiceID)
				queue = append(queue, ref.ServiceID)
			}
		}
	}

	sort.Strings(result)
	return result
}

// FindPath returns all simple paths from service from to service to, limited
// to a maximum of maxDepth hops.
//
// Each path is a slice of service IDs starting with from and ending with to.
// Returns nil when no path exists or either service is unknown.
func (da *DependencyAnalyzer) FindPath(from, to string) [][]string {
	const maxDepth = 10 // guard against explosion in dense graphs
	if from == to {
		return nil
	}
	if _, ok := da.serviceGraph.Services[from]; !ok {
		return nil
	}
	if _, ok := da.serviceGraph.Services[to]; !ok {
		return nil
	}

	var results [][]string
	visited := make(map[string]bool)

	var dfs func(cur string, path []string)
	dfs = func(cur string, path []string) {
		if len(path)-1 >= maxDepth {
			return
		}
		for _, ref := range da.serviceGraph.Dependencies[cur] {
			next := ref.ServiceID
			if visited[next] {
				continue
			}
			newPath := append(append([]string{}, path...), next)
			if next == to {
				results = append(results, newPath)
				continue
			}
			visited[next] = true
			dfs(next, newPath)
			visited[next] = false
		}
	}

	visited[from] = true
	dfs(from, []string{from})
	return results
}

// DetectCycles finds all circular dependencies in the service graph using
// iterative DFS with three-colour marking (white/gray/black).
//
// Returns a slice of cycles; each cycle is a slice of service IDs where the
// first and last element are the same service.  Returns nil when the graph
// has no cycles.
func (da *DependencyAnalyzer) DetectCycles() [][]string {
	const (
		white = 0
		gray  = 1
		black = 2
	)

	colour := make(map[string]int, len(da.serviceGraph.Services))
	parent := make(map[string]string, len(da.serviceGraph.Services))

	var cycles [][]string
	seen := make(map[string]bool) // dedup identical cycles

	var dfs func(u string)
	dfs = func(u string) {
		colour[u] = gray

		for _, ref := range da.serviceGraph.Dependencies[u] {
			v := ref.ServiceID
			switch colour[v] {
			case white:
				parent[v] = u
				dfs(v)
			case gray:
				// Back edge — reconstruct cycle path.
				cycle := reconstructServiceCycle(parent, v, u)
				key := cycleKey(cycle)
				if !seen[key] {
					seen[key] = true
					cycles = append(cycles, cycle)
				}
			}
		}

		colour[u] = black
	}

	for id := range da.serviceGraph.Services {
		if colour[id] == white {
			dfs(id)
		}
	}

	return cycles
}

// FindDependencyChain finds the shortest dependency path from service from to
// service to using BFS.  The search is limited to maxChainDepth hops to
// prevent combinatorial explosion.
//
// Returns a DependencyChain with an empty Path when no route exists.
func (da *DependencyAnalyzer) FindDependencyChain(from, to string) DependencyChain {
	const maxChainDepth = 5

	if from == to {
		return DependencyChain{}
	}
	if _, ok := da.serviceGraph.Services[from]; !ok {
		return DependencyChain{}
	}
	if _, ok := da.serviceGraph.Services[to]; !ok {
		return DependencyChain{}
	}

	type bfsEntry struct {
		id       string
		path     []string
		evidence []string
	}

	visited := make(map[string]bool)
	visited[from] = true
	queue := []bfsEntry{{id: from, path: []string{from}}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if len(cur.path)-1 >= maxChainDepth {
			continue
		}

		for _, ref := range da.serviceGraph.Dependencies[cur.id] {
			if visited[ref.ServiceID] {
				continue
			}
			newPath := append(append([]string{}, cur.path...), ref.ServiceID)
			newEvidence := append(append([]string{}, cur.evidence...), ref.Evidence)

			if ref.ServiceID == to {
				return DependencyChain{
					Path:     newPath,
					Distance: len(newPath) - 1,
					Evidence: strings.Join(newEvidence, " -> "),
				}
			}
			visited[ref.ServiceID] = true
			queue = append(queue, bfsEntry{
				id:       ref.ServiceID,
				path:     newPath,
				evidence: newEvidence,
			})
		}
	}

	// No path found.
	return DependencyChain{}
}

// ServiceGraph returns a read-only view of the computed service dependency
// graph.  Callers should not mutate the returned value.
func (da *DependencyAnalyzer) GetServiceGraph() *ServiceGraph {
	return da.serviceGraph
}

// --- helpers ----------------------------------------------------------------

// edgeTypeToDepType maps knowledge graph EdgeType values to dependency type
// strings used in ServiceRef.Type.
func edgeTypeToDepType(et EdgeType) string {
	switch et {
	case EdgeCalls:
		return "direct-call"
	case EdgeDependsOn:
		return "direct-call"
	case EdgeMentions:
		return "reference"
	case EdgeReferences:
		return "reference"
	case EdgeImplements:
		return "reference"
	default:
		return "reference"
	}
}

// hasRef returns true when refs already contains a ServiceRef with the same
// ServiceID and Type as ref.
func hasRef(refs []ServiceRef, ref ServiceRef) bool {
	for _, r := range refs {
		if r.ServiceID == ref.ServiceID && r.Type == ref.Type {
			return true
		}
	}
	return false
}

// reconstructServiceCycle builds a cycle path starting and ending at
// cycleRoot by following the parent map backwards from tail.
//
// When the DFS detects a back edge tail→cycleRoot, the cycle is:
//
//	cycleRoot → … → tail → cycleRoot
//
// We reconstruct by walking parent[] from tail back to cycleRoot, building
// the path in reverse, then closing it.
func reconstructServiceCycle(parent map[string]string, cycleRoot, tail string) []string {
	// Collect intermediate nodes from tail back to (but not including) cycleRoot.
	var middle []string
	cur := tail
	for cur != cycleRoot {
		middle = append([]string{cur}, middle...)
		p, ok := parent[cur]
		if !ok {
			break
		}
		cur = p
	}
	// Build: cycleRoot + middle + cycleRoot (closed cycle).
	path := make([]string, 0, len(middle)+2)
	path = append(path, cycleRoot)
	path = append(path, middle...)
	path = append(path, cycleRoot)
	return path
}

// cycleKey returns a canonical string key for a cycle so duplicates can be
// detected regardless of rotation.
func cycleKey(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	// Use the lexicographically smallest rotation as canonical form.
	n := len(cycle) - 1 // last element == first element; work with n distinct nodes
	if n <= 0 {
		return strings.Join(cycle, "|")
	}
	min := 0
	for i := 1; i < n; i++ {
		if cycle[i] < cycle[min] {
			min = i
		}
	}
	rotated := make([]string, n)
	for i := 0; i < n; i++ {
		rotated[i] = cycle[(min+i)%n]
	}
	return strings.Join(rotated, "|")
}
