package knowledge

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Service represents a detected microservice in the documentation graph.
// Services are identified using heuristic scoring applied to Node metadata
// and document content.
type Service struct {
	// ID is a normalised, URL-safe identifier derived from the service name
	// (e.g. "auth-service", "api-gateway").
	ID string

	// Name is the human-readable service label extracted from headings or
	// filenames (e.g. "Auth Service", "API Gateway").
	Name string

	// File is the relative path of the primary documentation file for this
	// service (matches Node.ID / Document.ID).
	File string

	// Confidence is a normalised [0.0, 1.0] score reflecting how certain the
	// detector is that this node represents a real microservice.
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

// ServiceConfig holds the optional, user-supplied service configuration loaded
// from a services.yaml file.  Configured services override auto-detection.
type ServiceConfig struct {
	// Services is the list of explicitly configured service definitions.
	Services []ServiceConfigEntry
}

// ServiceConfigEntry is a single entry in services.yaml.
type ServiceConfigEntry struct {
	// ID is the canonical service ID (e.g. "api-gateway").
	ID string

	// Patterns is the list of case-insensitive strings to match against
	// filenames and H1 headings.
	Patterns []string

	// Type describes the service category (e.g. "microservice", "database").
	Type string
}

// ConfidenceServiceFilename is assigned when the filename contains "service".
const ConfidenceServiceFilename float64 = 0.9

// ConfidenceServiceHeading is assigned when the H1 heading contains "Service".
const ConfidenceServiceHeading float64 = 0.7

// ConfidenceHighInDegree is assigned to frequently-referenced nodes that do
// not match the filename or heading heuristics.
const ConfidenceHighInDegree float64 = 0.4

// ConfidenceConfigured is assigned when a service matches a configured entry
// in services.yaml — the highest confidence tier.
const ConfidenceConfigured float64 = 1.0

// inDegreeThreshold is the minimum in-degree for a node to be considered a
// high-traffic service based on reference count alone.
const inDegreeThreshold = 3

// ServiceDetector identifies microservices from a knowledge graph using
// multiple heuristics.  An optional ServiceConfig can be loaded from a
// services.yaml file to supplement or override auto-detection.
type ServiceDetector struct {
	// config holds the optional user-supplied service configuration.
	// nil means no config file was loaded.
	config *ServiceConfig
}

// NewServiceDetector creates a ServiceDetector with no configuration.
// Call LoadServiceConfig separately if you want to use a services.yaml file.
func NewServiceDetector() *ServiceDetector {
	return &ServiceDetector{}
}

// NewServiceDetectorWithConfig creates a ServiceDetector using the supplied
// configuration.  Configured services override auto-detection results.
func NewServiceDetectorWithConfig(cfg *ServiceConfig) *ServiceDetector {
	return &ServiceDetector{config: cfg}
}

// DetectServices identifies all microservices in graph and returns them ranked
// by confidence score (highest first).
//
// The detection pipeline:
//  1. Apply per-node heuristics (filename, heading, in-degree) to collect
//     service candidates.
//  2. Merge with configured services (if a ServiceConfig is present).
//  3. Rank by confidence score.
func (sd *ServiceDetector) DetectServices(graph *Graph, docs []Document) []Service {
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
	candidateMap := make(map[string]Service, graph.NodeCount())

	for id, node := range graph.Nodes {
		svc, confidence := sd.IsService(node)

		// High in-degree heuristic: apply when no other heuristic matched OR
		// when the node is highly referenced and didn't score yet.
		if confidence <= 0 && inDegree[id] >= inDegreeThreshold {
			confidence = ConfidenceHighInDegree
			svc = Service{
				ID:   nodeToServiceID(node.ID),
				Name: node.Title,
				File: id,
			}
		}

		if confidence <= 0 {
			continue
		}

		// Extract endpoints if we have the document.
		if doc, ok := docByID[id]; ok {
			svc.Endpoints = sd.DetectEndpoints(doc)
		}

		svc.Confidence = confidence
		candidateMap[id] = svc
	}

	// Merge with configured services (override auto-detected entries).
	if sd.config != nil {
		for id, node := range graph.Nodes {
			for _, entry := range sd.config.Services {
				if matchesPatterns(node.ID, node.Title, entry.Patterns) {
					existing := candidateMap[id]
					existing.ID = entry.ID
					existing.Name = node.Title
					existing.File = id
					existing.Confidence = ConfidenceConfigured
					if doc, ok := docByID[id]; ok {
						existing.Endpoints = sd.DetectEndpoints(doc)
					}
					candidateMap[id] = existing
					break
				}
			}
		}
	}

	// Collect and rank candidates.
	services := make([]Service, 0, len(candidateMap))
	for _, svc := range candidateMap {
		services = append(services, svc)
	}
	return sd.RankServices(services)
}

// IsService applies heuristic scoring to a single Node and returns the
// candidate Service and its confidence score.
//
// Returns (Service{}, 0) when the node does not appear to be a service.
func (sd *ServiceDetector) IsService(node *Node) (Service, float64) {
	lowerID := strings.ToLower(node.ID)
	lowerTitle := strings.ToLower(node.Title)

	// Heuristic 1: filename contains "service".
	// Examples: auth-service.md, user-service.md, payment-service.md
	stem := filenameStem(node.ID)
	if strings.Contains(strings.ToLower(stem), "service") {
		return Service{
			ID:   nodeToServiceID(node.ID),
			Name: node.Title,
			File: node.ID,
		}, ConfidenceServiceFilename
	}

	// Heuristic 2: H1 heading contains "Service".
	// Examples: "# User Service", "# Auth Service"
	if strings.Contains(lowerTitle, "service") {
		return Service{
			ID:   nodeToServiceID(node.ID),
			Name: node.Title,
			File: node.ID,
		}, ConfidenceServiceHeading
	}

	// Heuristic 3: Optional — check configured patterns via IsService API.
	// High in-degree detection is handled in DetectServices (requires graph).
	_ = lowerID
	return Service{}, 0
}

// DetectEndpoints scans a Document for REST API endpoint patterns and returns
// the extracted endpoints.
//
// Recognised patterns (case-insensitive):
//   - "POST /users" — HTTP method followed by a path
//   - "# POST /users endpoint" — heading pattern
//   - "`POST /users`" — inline code pattern
//   - Code blocks containing "METHOD /path" lines
func (sd *ServiceDetector) DetectEndpoints(doc *Document) []Endpoint {
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

// RankServices sorts services by confidence (descending) and then by ID
// (ascending) for stable ordering within the same confidence tier.
func (sd *ServiceDetector) RankServices(candidates []Service) []Service {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Confidence != candidates[j].Confidence {
			return candidates[i].Confidence > candidates[j].Confidence
		}
		return candidates[i].ID < candidates[j].ID
	})
	return candidates
}

// LoadServiceConfig reads a services.yaml file from path and returns the
// parsed ServiceConfig.  Returns nil, nil when the file does not exist
// (graceful fallback — config is optional).
//
// The YAML format supported is a strict subset:
//
//	services:
//	  - id: api-gateway
//	    patterns: ["api-gateway", "API Gateway"]
//	    type: microservice
func LoadServiceConfig(path string) (*ServiceConfig, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			// Config is optional — graceful fallback.
			return nil, nil
		}
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	return parseServiceYAML(f)
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

// parseServiceYAML parses the simple services.yaml format using a line-based
// state machine.  It supports only the specific structure needed for service
// configuration.
func parseServiceYAML(r *os.File) (*ServiceConfig, error) {
	cfg := &ServiceConfig{}
	scanner := bufio.NewScanner(r)

	type state int
	const (
		stateRoot state = iota
		stateServices
		stateEntry
	)

	current := stateRoot
	var currentEntry *ServiceConfigEntry

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip blank lines and comments.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		switch current {
		case stateRoot:
			if strings.TrimRight(trimmed, ":") == "services" {
				current = stateServices
			}

		case stateServices, stateEntry:
			if strings.HasPrefix(trimmed, "- ") {
				// New list entry.
				if currentEntry != nil {
					cfg.Services = append(cfg.Services, *currentEntry)
				}
				currentEntry = &ServiceConfigEntry{}
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
		cfg.Services = append(cfg.Services, *currentEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// parseYAMLKeyValue parses a single "key: value" or "key: [val1, val2]" line
// and sets the corresponding field on entry.
func parseYAMLKeyValue(entry *ServiceConfigEntry, line string) {
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

// nodeToServiceID converts a node ID (relative file path) into a kebab-case
// service ID.  Examples:
//   - "services/auth-service.md"  → "auth-service"
//   - "docs/UserService.md"       → "userservice"
func nodeToServiceID(nodeID string) string {
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
