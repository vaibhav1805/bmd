package knowledge

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Component represents a detected component in the documentation graph.
// Components are identified using heuristic scoring applied to Node metadata
// and document content.
type Component struct {
	// ID is a normalised, URL-safe identifier derived from the component name
	// (e.g. "auth-component", "api-gateway").
	ID string

	// Name is the human-readable component label extracted from headings or
	// filenames (e.g. "Auth Component", "API Gateway").
	Name string

	// File is the relative path of the primary documentation file for this
	// component (matches Node.ID / Document.ID).
	File string

	// Confidence is a normalised [0.0, 1.0] score reflecting how certain the
	// detector is that this node represents a real component.
	//
	// Heuristic thresholds:
	//   0.9 — filename contains "service" / configured service name
	//   0.7 — H1 heading contains "Service"
	//   0.4 — high in-degree node (frequently referenced)
	Confidence float64

	// Endpoints holds the REST API endpoints discovered in the document.
	Endpoints []Endpoint
}

// Endpoint describes a single REST API endpoint extracted from a markdown
// document (e.g. "POST /users").
type Endpoint struct {
	// Method is the HTTP verb (GET, POST, PUT, DELETE, PATCH, …).
	Method string

	// Path is the URL path (e.g. "/users", "/v1/auth/token").
	Path string

	// Evidence is the raw source line where the endpoint was found.
	Evidence string
}

// ComponentConfig holds the optional, user-supplied component configuration loaded
// from a components.yaml file.  Configured components override auto-detection.
type ComponentConfig struct {
	// Components is the list of explicitly configured component definitions.
	Components []ComponentConfigEntry
}

// ComponentConfigEntry is a single entry in components.yaml.
type ComponentConfigEntry struct {
	// ID is the canonical component ID (e.g. "api-gateway").
	ID string

	// Patterns is the list of case-insensitive strings to match against
	// filenames and H1 headings.
	Patterns []string

	// Type describes the component category (e.g. "microservice", "database").
	Type string
}

// ConfidenceComponentFilename is assigned when the filename contains "component".
const ConfidenceComponentFilename float64 = 0.9

// ConfidenceComponentHeading is assigned when the H1 heading contains "Component".
const ConfidenceComponentHeading float64 = 0.7

// ConfidenceHighInDegree is assigned to frequently-referenced nodes that do
// not match the filename or heading heuristics.
const ConfidenceHighInDegree float64 = 0.4

// ConfidenceConfigured is assigned when a service matches a configured entry
// in services.yaml — the highest confidence tier.
const ConfidenceConfigured float64 = 1.0

// inDegreeThreshold is the minimum in-degree for a node to be considered a
// high-traffic service based on reference count alone.
const inDegreeThreshold = 3

// ComponentDetector identifies components from a knowledge graph using
// multiple heuristics.  An optional ComponentConfig can be loaded from a
// components.yaml file to supplement or override auto-detection.
type ComponentDetector struct {
	// config holds the optional user-supplied component configuration.
	// nil means no config file was loaded.
	config *ComponentConfig
}

// NewComponentDetector creates a ComponentDetector with no configuration.
// Call LoadComponentConfig separately if you want to use a components.yaml file.
func NewComponentDetector() *ComponentDetector {
	return &ComponentDetector{}
}

// NewComponentDetectorWithConfig creates a ComponentDetector using the supplied
// configuration.  Configured components override auto-detection results.
func NewComponentDetectorWithConfig(cfg *ComponentConfig) *ComponentDetector {
	return &ComponentDetector{config: cfg}
}

// DetectComponents identifies all components in graph and returns them ranked
// by confidence score (highest first).
//
// The detection pipeline:
//  1. Apply per-node heuristics (filename, heading, in-degree) to collect
//     component candidates.
//  2. Merge with configured components (if a ComponentConfig is present).
//  3. Rank by confidence score.
func (cd *ComponentDetector) DetectComponents(graph *Graph, docs []Document) []Component {
	// Build a lookup from node ID to Document for endpoint extraction.
	docByID := make(map[string]*Document, len(docs))
	for i := range docs {
		docByID[docs[i].ID] = &docs[i]
	}

	// Track in-degree for high-traffic heuristic.
	inDegree := make(map[string]int, graph.NodeCount())
	for _, edges := range graph.BySource {
		for _, e := range edges {
			inDegree[e.Target]++
		}
	}

	// candidateMap collects the best candidate for each node ID.
	candidateMap := make(map[string]Component, graph.NodeCount())

	for id, node := range graph.Nodes {
		comp, confidence := cd.IsComponent(node)

		// High in-degree heuristic: apply when no other heuristic matched OR
		// when the node is highly referenced and didn't score yet.
		if confidence <= 0 && inDegree[id] >= inDegreeThreshold {
			confidence = ConfidenceHighInDegree
			comp = Component{
				ID:   nodeToComponentID(node.ID),
				Name: node.Title,
				File: id,
			}
		}

		if confidence <= 0 {
			continue
		}

		// Extract endpoints if we have the document.
		if doc, ok := docByID[id]; ok {
			comp.Endpoints = cd.DetectEndpoints(doc)
		}

		comp.Confidence = confidence
		candidateMap[id] = comp
	}

	// Merge with configured components (override auto-detected entries).
	if cd.config != nil {
		for id, node := range graph.Nodes {
			for _, entry := range cd.config.Components {
				if matchesPatterns(node.ID, node.Title, entry.Patterns) {
					existing := candidateMap[id]
					existing.ID = entry.ID
					existing.Name = node.Title
					existing.File = id
					existing.Confidence = ConfidenceConfigured
					if doc, ok := docByID[id]; ok {
						existing.Endpoints = cd.DetectEndpoints(doc)
					}
					candidateMap[id] = existing
					break
				}
			}
		}
	}

	// Collect and rank candidates.
	components := make([]Component, 0, len(candidateMap))
	for _, comp := range candidateMap {
		components = append(components, comp)
	}
	return cd.RankComponents(components)
}

// IsComponent applies heuristic scoring to a single Node and returns the
// candidate Component and its confidence score.
//
// Returns (Component{}, 0) when the node does not appear to be a component.
func (cd *ComponentDetector) IsComponent(node *Node) (Component, float64) {
	lowerID := strings.ToLower(node.ID)
	lowerTitle := strings.ToLower(node.Title)

	// Heuristic 1: filename contains "component".
	// Examples: auth-component.md, user-component.md, payment-component.md
	stem := filenameStem(node.ID)
	if strings.Contains(strings.ToLower(stem), "component") {
		return Component{
			ID:   nodeToComponentID(node.ID),
			Name: node.Title,
			File: node.ID,
		}, ConfidenceComponentFilename
	}

	// Heuristic 2: H1 heading contains "Component".
	// Examples: "# User Component", "# Auth Component"
	if strings.Contains(lowerTitle, "component") {
		return Component{
			ID:   nodeToComponentID(node.ID),
			Name: node.Title,
			File: node.ID,
		}, ConfidenceComponentHeading
	}

	// Heuristic 3: Optional — check configured patterns via IsComponent API.
	// High in-degree detection is handled in DetectComponents (requires graph).
	_ = lowerID
	return Component{}, 0
}

// DetectEndpoints scans a Document for REST API endpoint patterns and returns
// the extracted endpoints.
//
// Recognised patterns (case-insensitive):
//   - "POST /users" — HTTP method followed by a path
//   - "# POST /users endpoint" — heading pattern
//   - "`POST /users`" — inline code pattern
//   - Code blocks containing "METHOD /path" lines
func (cd *ComponentDetector) DetectEndpoints(doc *Document) []Endpoint {
	var endpoints []Endpoint
	seen := make(map[string]bool)

	lines := strings.Split(doc.Content, "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		eps := extractEndpointsFromLine(line)
		for _, ep := range eps {
			key := ep.Method + " " + ep.Path
			if !seen[key] {
				seen[key] = true
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints
}

// RankComponents sorts components by confidence (descending) and then by ID
// (ascending) for stable ordering within the same confidence tier.
func (cd *ComponentDetector) RankComponents(candidates []Component) []Component {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Confidence != candidates[j].Confidence {
			return candidates[i].Confidence > candidates[j].Confidence
		}
		return candidates[i].ID < candidates[j].ID
	})
	return candidates
}

// LoadComponentConfig reads a components.yaml file from path and returns the
// parsed ComponentConfig.  Returns nil, nil when the file does not exist
// (graceful fallback — config is optional).
//
// The YAML format supported is a strict subset:
//
//	components:
//	  - id: api-gateway
//	    patterns: ["api-gateway", "API Gateway"]
//	    type: microservice
func LoadComponentConfig(path string) (*ComponentConfig, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			// Config is optional — graceful fallback.
			return nil, nil
		}
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	return parseComponentYAML(f)
}

// --- endpoint extraction helpers --------------------------------------------

// httpMethods is the set of uppercase HTTP method names we recognise.
var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
}

// extractEndpointsFromLine attempts to extract HTTP method + path pairs from
// a single documentation line.
//
// It handles multiple formats:
//   - "POST /users"          — plain method + path
//   - "## POST /users"       — heading prefixed
//   - "`GET /health`"        — inline code span
//   - "Call `GET /health`"   — inline code within prose
func extractEndpointsFromLine(line string) []Endpoint {
	var endpoints []Endpoint

	// First pass: search the raw line for backtick-delimited inline code
	// spans and extract endpoints from within each span.
	for i := 0; i < len(line); i++ {
		if line[i] == '`' {
			end := strings.Index(line[i+1:], "`")
			if end < 0 {
				break
			}
			span := line[i+1 : i+1+end]
			eps := extractFromCleanedLine(span, line)
			endpoints = append(endpoints, eps...)
			i += end + 1
		}
	}

	// Second pass: strip markdown decorators from the whole line and search
	// for method+path pairs in the resulting text.
	cleaned := line
	cleaned = strings.TrimLeft(cleaned, "#> `*_")
	// Remove any remaining backticks.
	cleaned = strings.ReplaceAll(cleaned, "`", " ")
	cleaned = strings.TrimSpace(cleaned)
	eps := extractFromCleanedLine(cleaned, line)
	endpoints = append(endpoints, eps...)

	return endpoints
}

// extractFromCleanedLine extracts HTTP endpoint pairs from pre-cleaned text.
// evidence is the original line used to populate Endpoint.Evidence.
func extractFromCleanedLine(cleaned, evidence string) []Endpoint {
	var endpoints []Endpoint
	tokens := strings.Fields(cleaned)
	for i := 0; i+1 < len(tokens); i++ {
		method := strings.ToUpper(strings.Trim(tokens[i], "`.,;:"))
		if !httpMethods[method] {
			continue
		}
		pathToken := tokens[i+1]
		// A valid path starts with '/'.
		if !strings.HasPrefix(pathToken, "/") {
			continue
		}
		// Strip trailing punctuation (. , ; : `).
		pathToken = strings.TrimRight(pathToken, ".,;:`")
		endpoints = append(endpoints, Endpoint{
			Method:   method,
			Path:     pathToken,
			Evidence: evidence,
		})
	}
	return endpoints
}

// --- YAML config parser (minimal subset) ------------------------------------

// parseComponentYAML parses the simple components.yaml format using a line-based
// state machine.  It supports only the specific structure needed for component
// configuration.
func parseComponentYAML(r *os.File) (*ComponentConfig, error) {
	cfg := &ComponentConfig{}
	scanner := bufio.NewScanner(r)

	type state int
	const (
		stateRoot state = iota
		stateComponents
		stateEntry
	)

	current := stateRoot
	var currentEntry *ComponentConfigEntry

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip blank lines and comments.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		switch current {
		case stateRoot:
			if strings.TrimRight(trimmed, ":") == "components" {
				current = stateComponents
			}

		case stateComponents, stateEntry:
			if strings.HasPrefix(trimmed, "- ") {
				// New list entry.
				if currentEntry != nil {
					cfg.Components = append(cfg.Components, *currentEntry)
				}
				currentEntry = &ComponentConfigEntry{}
				current = stateEntry
				rest := strings.TrimPrefix(trimmed, "- ")
				trimmed = rest
				// Intentional fall-through to parse the inline key-value.
				parseYAMLKeyValue(currentEntry, trimmed)
			} else if current == stateEntry && currentEntry != nil {
				parseYAMLKeyValue(currentEntry, trimmed)
			}
		}
	}

	if currentEntry != nil {
		cfg.Components = append(cfg.Components, *currentEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// parseYAMLKeyValue parses a single "key: value" or "key: [val1, val2]" line
// and sets the corresponding field on entry.
func parseYAMLKeyValue(entry *ComponentConfigEntry, line string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])

	switch key {
	case "id":
		entry.ID = value
	case "type":
		entry.Type = value
	case "patterns":
		entry.Patterns = parseYAMLStringList(value)
	}
}

// parseYAMLStringList parses a YAML inline sequence: ["val1", "val2"] or
// [val1, val2].  Returns nil for empty / malformed input.
func parseYAMLStringList(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		// Single-value fallback.
		if s != "" {
			return []string{strings.Trim(s, `"'`)}
		}
		return nil
	}

	inner := s[1 : len(s)-1]
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// --- utility helpers --------------------------------------------------------

// nodeToComponentID converts a node ID (relative file path) into a kebab-case
// component ID.  Examples:
//   - "components/auth-component.md"  → "auth-component"
//   - "docs/UserComponent.md"       → "usercomponent"
func nodeToComponentID(nodeID string) string {
	stem := filenameStem(nodeID)
	return strings.ToLower(stem)
}

// filenameStem returns the base file name without extension.
// "services/auth-service.md" → "auth-service"
func filenameStem(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// matchesPatterns returns true when nodeID or title matches any of patterns
// (case-insensitive substring match).
func matchesPatterns(nodeID, title string, patterns []string) bool {
	lowerID := strings.ToLower(nodeID)
	lowerTitle := strings.ToLower(title)
	for _, p := range patterns {
		lp := strings.ToLower(p)
		if strings.Contains(lowerID, lp) || strings.Contains(lowerTitle, lp) {
			return true
		}
	}
	return false
}
