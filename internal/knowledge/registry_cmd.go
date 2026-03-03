package knowledge

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ─── argument structs ─────────────────────────────────────────────────────────

// ComponentsListArgs holds parsed arguments for CmdComponentsList.
type ComponentsListArgs struct {
	Dir    string
	Format string // "table" | "json"
}

// ComponentsSearchArgs holds parsed arguments for CmdComponentsSearch.
type ComponentsSearchArgs struct {
	Query  string
	Dir    string
	Format string // "table" | "json"
}

// ComponentsInspectArgs holds parsed arguments for CmdComponentsInspect.
type ComponentsInspectArgs struct {
	ComponentID string
	Dir         string
	Format      string // "table" | "json"
}

// RelationshipsArgs holds parsed arguments for CmdRelationships.
type RelationshipsArgs struct {
	Dir            string
	From           string  // --from: show downstream deps of this component
	To             string  // --to: show upstream deps (who depends on this component)
	MinConfidence  float64 // --confidence: minimum confidence threshold
	IncludeSignals bool    // --include-signals: show signal breakdown
	Format         string  // "table" | "json" | "dot"
}

// ReviewArgs holds parsed arguments for CmdRelationshipsReview.
type ReviewArgs struct {
	Dir       string
	AcceptAll bool   // --accept-all: auto-accept all suggestions
	RejectAll bool   // --reject-all: reject all suggestions
	Edit      bool   // --edit: open in $EDITOR
	ExportTo  string // --export-to: save to specific location
}

// ─── argument parsers ─────────────────────────────────────────────────────────

// ParseComponentsListArgs parses args for CmdComponentsList.
//
// Usage: bmd components list [--dir DIR] [--format table|json]
func ParseComponentsListArgs(args []string) (*ComponentsListArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("components list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ComponentsListArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to load registry from")
	fs.StringVar(&a.Format, "format", "table", "Output format (table|json)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("components list: %w", err)
	}
	if len(positionals) > 0 {
		a.Dir = positionals[0]
	}

	return &a, nil
}

// ParseComponentsSearchArgs parses args for CmdComponentsSearch.
//
// Usage: bmd components search QUERY [--dir DIR] [--format table|json]
func ParseComponentsSearchArgs(args []string) (*ComponentsSearchArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("components search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ComponentsSearchArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to load registry from")
	fs.StringVar(&a.Format, "format", "table", "Output format (table|json)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("components search: %w", err)
	}
	if len(positionals) == 0 {
		return nil, fmt.Errorf("components search: QUERY argument required")
	}
	a.Query = positionals[0]
	if len(positionals) > 1 {
		a.Dir = positionals[1]
	}

	return &a, nil
}

// ParseComponentsInspectArgs parses args for CmdComponentsInspect.
//
// Usage: bmd components inspect COMPONENT_ID [--dir DIR] [--format table|json]
func ParseComponentsInspectArgs(args []string) (*ComponentsInspectArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("components inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ComponentsInspectArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to load registry from")
	fs.StringVar(&a.Format, "format", "table", "Output format (table|json)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("components inspect: %w", err)
	}
	if len(positionals) == 0 {
		return nil, fmt.Errorf("components inspect: COMPONENT_ID argument required")
	}
	a.ComponentID = positionals[0]
	if len(positionals) > 1 {
		a.Dir = positionals[1]
	}

	return &a, nil
}

// ParseRelationshipsArgs parses args for CmdRelationships.
//
// Usage: bmd relationships [--from COMPONENT] [--to COMPONENT] [--confidence 0.0] [--include-signals] [--format table|json|dot]
func ParseRelationshipsArgs(args []string) (*RelationshipsArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("relationships", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a RelationshipsArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to load registry from")
	fs.StringVar(&a.From, "from", "", "Show relationships from this component (downstream deps)")
	fs.StringVar(&a.To, "to", "", "Show relationships to this component (upstream deps)")
	fs.Float64Var(&a.MinConfidence, "confidence", 0.0, "Minimum confidence threshold (0.0–1.0)")
	fs.BoolVar(&a.IncludeSignals, "include-signals", false, "Show signal breakdown for each relationship")
	fs.StringVar(&a.Format, "format", "table", "Output format (table|json|dot)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("relationships: %w", err)
	}
	if len(positionals) > 0 {
		a.Dir = positionals[0]
	}

	if a.MinConfidence < 0.0 || a.MinConfidence > 1.0 {
		return nil, fmt.Errorf("relationships: --confidence must be in [0.0, 1.0]")
	}

	return &a, nil
}

// ParseReviewArgs parses args for CmdRelationshipsReview.
//
// Usage: bmd relationships-review [--dir DIR] [--accept-all] [--reject-all] [--edit] [--export-to PATH]
func ParseReviewArgs(args []string) (*ReviewArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("relationships-review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ReviewArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory containing manifest files")
	fs.BoolVar(&a.AcceptAll, "accept-all", false, "Auto-accept all discovered relationships")
	fs.BoolVar(&a.RejectAll, "reject-all", false, "Reject all discovered relationships")
	fs.BoolVar(&a.Edit, "edit", false, "Open manifest in $EDITOR for review")
	fs.StringVar(&a.ExportTo, "export-to", "", "Save accepted relationships to a specific path")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("relationships-review: %w", err)
	}
	if len(positionals) > 0 {
		a.Dir = positionals[0]
	}

	if a.AcceptAll && a.RejectAll {
		return nil, fmt.Errorf("relationships-review: --accept-all and --reject-all are mutually exclusive")
	}

	return &a, nil
}

// ─── loadOrBuildRegistry loads the registry, building from graph if missing ──

func loadOrBuildRegistry(absDir string) (*ComponentRegistry, error) {
	registryPath := filepath.Join(absDir, RegistryFileName)
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}
	if reg != nil {
		return reg, nil
	}

	// No registry file: bootstrap from graph.
	db, graph, _, graphErr := loadGraphAndServices(absDir)
	if graphErr != nil {
		return nil, graphErr
	}
	defer db.Close() //nolint:errcheck

	docs, _ := ScanDirectory(absDir)
	reg = NewComponentRegistry()
	reg.InitFromGraph(graph, docs)
	return reg, nil
}

// ─── command implementations ──────────────────────────────────────────────────

// CmdComponents is the router for `bmd components` subcommands.
//
// Subcommands: list, search, inspect
// Falls back to the legacy behavior (cmdComponentsLegacy) when no subcommand
// is given or when a flag is passed as the first argument.
func CmdComponents(args []string) error {
	if len(args) == 0 {
		// No subcommand: fall back to legacy behavior.
		return cmdComponentsLegacy(args)
	}

	switch args[0] {
	case "list":
		return CmdComponentsList(args[1:])
	case "search":
		return CmdComponentsSearch(args[1:])
	case "inspect":
		return CmdComponentsInspect(args[1:])
	default:
		// If first arg starts with '-' or is a flag, treat as legacy invocation.
		if strings.HasPrefix(args[0], "-") {
			return cmdComponentsLegacy(args)
		}
		return fmt.Errorf("components: unknown subcommand %q (use list, search, or inspect)", args[0])
	}
}

// CmdComponentsList implements `bmd components list`.
//
// Lists all components with their type, file, and relationship counts.
func CmdComponentsList(args []string) error {
	a, err := ParseComponentsListArgs(args)
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
		return fmt.Errorf("components list: resolve dir: %w", err)
	}

	reg, err := loadOrBuildRegistry(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("components list: %w", err)
	}

	// Compute incoming/outgoing counts per component.
	incoming, outgoing := computeRelationshipCounts(reg)

	// Sort component IDs for deterministic output.
	ids := sortedComponentIDs(reg)

	if !isJSON {
		fmt.Println(formatComponentsListTable(reg, ids, incoming, outgoing))
		return nil
	}

	items := buildComponentListJSON(reg, ids, incoming, outgoing)
	payload := map[string]interface{}{
		"type":  "components_list",
		"data":  items,
		"count": len(items),
	}
	fmt.Println(marshalContract(NewOKResponse("Components listed", payload)))
	return nil
}

// CmdComponentsSearch implements `bmd components search QUERY`.
//
// Searches components by name/ID using case-insensitive partial matching.
func CmdComponentsSearch(args []string) error {
	a, err := ParseComponentsSearchArgs(args)
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
		return fmt.Errorf("components search: resolve dir: %w", err)
	}

	reg, err := loadOrBuildRegistry(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("components search: %w", err)
	}

	query := strings.ToLower(a.Query)
	var matchIDs []string
	for id, comp := range reg.Components {
		if strings.Contains(strings.ToLower(id), query) ||
			strings.Contains(strings.ToLower(comp.Name), query) {
			matchIDs = append(matchIDs, id)
		}
	}
	sort.Strings(matchIDs)

	incoming, outgoing := computeRelationshipCounts(reg)

	if !isJSON {
		if len(matchIDs) == 0 {
			fmt.Printf("No components matching %q found.\n", a.Query)
			return nil
		}
		fmt.Println(formatComponentsListTable(reg, matchIDs, incoming, outgoing))
		return nil
	}

	items := buildComponentListJSON(reg, matchIDs, incoming, outgoing)
	payload := map[string]interface{}{
		"type":  "components_search",
		"query": a.Query,
		"data":  items,
		"count": len(items),
	}
	if len(items) == 0 {
		fmt.Println(marshalContract(NewEmptyResponse("No components matching query", payload)))
		return nil
	}
	fmt.Println(marshalContract(NewOKResponse("Components found", payload)))
	return nil
}

// CmdComponentsInspect implements `bmd components inspect COMPONENT_ID`.
//
// Shows detailed information about a single component including all relationships.
func CmdComponentsInspect(args []string) error {
	a, err := ParseComponentsInspectArgs(args)
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
		return fmt.Errorf("components inspect: resolve dir: %w", err)
	}

	reg, err := loadOrBuildRegistry(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("components inspect: %w", err)
	}

	comp := reg.GetComponent(a.ComponentID)
	if comp == nil {
		// Try case-insensitive match.
		for id, c := range reg.Components {
			if strings.EqualFold(id, a.ComponentID) {
				comp = c
				break
			}
		}
	}
	if comp == nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeFileNotFound,
				fmt.Sprintf("Component %q not found. Run `bmd components list` to see available components", a.ComponentID))))
			return nil
		}
		return fmt.Errorf("components inspect: component %q not found", a.ComponentID)
	}

	// Collect outgoing (what this component depends on) and incoming (what depends on this).
	var outgoing []RegistryRelationship
	var incoming []RegistryRelationship
	for _, rel := range reg.Relationships {
		if rel.FromComponent == comp.ID {
			outgoing = append(outgoing, rel)
		}
		if rel.ToComponent == comp.ID {
			incoming = append(incoming, rel)
		}
	}
	sort.Slice(outgoing, func(i, j int) bool { return outgoing[i].AggregatedConfidence > outgoing[j].AggregatedConfidence })
	sort.Slice(incoming, func(i, j int) bool { return incoming[i].AggregatedConfidence > incoming[j].AggregatedConfidence })

	if !isJSON {
		fmt.Println(formatComponentInspectTable(comp, incoming, outgoing))
		return nil
	}

	type signalJSON struct {
		Type       string  `json:"type"`
		Confidence float64 `json:"confidence"`
		Evidence   string  `json:"evidence,omitempty"`
	}
	type relJSON struct {
		Component  string       `json:"component"`
		Confidence float64      `json:"confidence"`
		Signals    []signalJSON `json:"signals,omitempty"`
	}

	buildRelJSON := func(rels []RegistryRelationship, useFrom bool) []relJSON {
		result := make([]relJSON, len(rels))
		for i, r := range rels {
			compID := r.ToComponent
			if useFrom {
				compID = r.FromComponent
			}
			sigs := make([]signalJSON, len(r.Signals))
			for j, s := range r.Signals {
				sigs[j] = signalJSON{
					Type:       string(s.SourceType),
					Confidence: roundFloat(s.Confidence, 4),
					Evidence:   s.Evidence,
				}
			}
			result[i] = relJSON{
				Component:  compID,
				Confidence: roundFloat(r.AggregatedConfidence, 4),
				Signals:    sigs,
			}
		}
		return result
	}

	payload := map[string]interface{}{
		"id":              comp.ID,
		"name":            comp.Name,
		"type":            string(comp.Type),
		"file":            comp.FileRef,
		"detected_at":     comp.DetectedAt.Format("2006-01-02"),
		"outgoing_count":  len(outgoing),
		"incoming_count":  len(incoming),
		"depends_on":      buildRelJSON(outgoing, false),
		"depended_on_by":  buildRelJSON(incoming, true),
	}
	fmt.Println(marshalContract(NewOKResponse("Component details", payload)))
	return nil
}

// CmdRelationships implements `bmd relationships`.
//
// Queries relationships from the component registry with optional filtering
// by source/destination component and confidence threshold.
func CmdRelationships(args []string) error {
	a, err := ParseRelationshipsArgs(args)
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
		return fmt.Errorf("relationships: resolve dir: %w", err)
	}

	reg, err := loadOrBuildRegistry(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("relationships: %w", err)
	}

	// Filter relationships based on flags.
	var rels []RegistryRelationship
	switch {
	case a.From != "":
		rels = reg.FindRelationships(a.From)
	case a.To != "":
		// Incoming: find all rels where ToComponent == a.To.
		for _, r := range reg.Relationships {
			if r.ToComponent == a.To {
				rels = append(rels, r)
			}
		}
	default:
		rels = reg.QueryByConfidence(a.MinConfidence)
	}

	// Apply confidence filter.
	if a.MinConfidence > 0 {
		var filtered []RegistryRelationship
		for _, r := range rels {
			if r.AggregatedConfidence >= a.MinConfidence {
				filtered = append(filtered, r)
			}
		}
		rels = filtered
	}

	// Sort by confidence descending.
	sort.Slice(rels, func(i, j int) bool {
		if rels[i].AggregatedConfidence != rels[j].AggregatedConfidence {
			return rels[i].AggregatedConfidence > rels[j].AggregatedConfidence
		}
		if rels[i].FromComponent != rels[j].FromComponent {
			return rels[i].FromComponent < rels[j].FromComponent
		}
		return rels[i].ToComponent < rels[j].ToComponent
	})

	if strings.ToLower(a.Format) == "dot" {
		fmt.Println(formatRelationshipsDOT(rels))
		return nil
	}

	if !isJSON {
		fmt.Println(formatRelationshipsTable(rels, a.From, a.To, a.IncludeSignals))
		return nil
	}

	type signalJSON struct {
		Type       string  `json:"type"`
		Confidence float64 `json:"confidence"`
	}
	type relJSON struct {
		From       string       `json:"from"`
		To         string       `json:"to"`
		Confidence float64      `json:"confidence"`
		Signals    []signalJSON `json:"signals,omitempty"`
	}

	component := a.From
	if component == "" {
		component = a.To
	}

	relItems := make([]relJSON, len(rels))
	for i, r := range rels {
		item := relJSON{
			From:       r.FromComponent,
			To:         r.ToComponent,
			Confidence: roundFloat(r.AggregatedConfidence, 4),
		}
		if a.IncludeSignals {
			sigs := make([]signalJSON, len(r.Signals))
			for j, s := range r.Signals {
				sigs[j] = signalJSON{
					Type:       string(s.SourceType),
					Confidence: roundFloat(s.Confidence, 4),
				}
			}
			item.Signals = sigs
		}
		relItems[i] = item
	}

	payload := map[string]interface{}{
		"type":          "relationships",
		"component":     component,
		"relationships": relItems,
		"count":         len(relItems),
	}
	if len(relItems) == 0 {
		fmt.Println(marshalContract(NewEmptyResponse("No relationships found", payload)))
		return nil
	}
	fmt.Println(marshalContract(NewOKResponse("Relationships found", payload)))
	return nil
}

// CmdRelationshipsReview implements `bmd relationships-review`.
//
// Loads discovered relationships, applies user review decisions, and saves
// the accepted manifest.
func CmdRelationshipsReview(args []string) error {
	a, err := ParseReviewArgs(args)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("relationships-review: resolve dir: %w", err)
	}

	discoveredPath := filepath.Join(absDir, DiscoveredManifestFile)
	acceptedPath := filepath.Join(absDir, AcceptedManifestFile)

	// Load discovered manifest.
	discovered, err := LoadRelationshipManifest(discoveredPath)
	if err != nil {
		return fmt.Errorf("relationships-review: %w", err)
	}
	if discovered == nil {
		fmt.Fprintf(os.Stderr, "No discovered relationships found at %s\n", discoveredPath)
		fmt.Fprintf(os.Stderr, "Run 'bmd index' first to discover relationships.\n")
		return nil
	}

	// Load existing user edits if present.
	accepted, err := LoadRelationshipManifest(acceptedPath)
	if err != nil {
		return fmt.Errorf("relationships-review: %w", err)
	}

	// Merge previous review decisions into the discovered manifest.
	discovered.MergeUserEdits(accepted)

	// Apply bulk actions.
	switch {
	case a.AcceptAll:
		discovered.AcceptAll()
		fmt.Fprintf(os.Stderr, "Accepted all %d relationships.\n", len(discovered.Relationships))
	case a.RejectAll:
		discovered.RejectAll()
		fmt.Fprintf(os.Stderr, "Rejected all %d relationships.\n", len(discovered.Relationships))
	case a.Edit:
		// Save to a temp file, open in editor, then load back.
		tmpFile := acceptedPath + ".edit.yaml"
		if err := SaveRelationshipManifest(discovered, tmpFile); err != nil {
			return fmt.Errorf("relationships-review: save temp: %w", err)
		}
		defer os.Remove(tmpFile) //nolint:errcheck

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, tmpFile)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("relationships-review: editor failed: %w", err)
		}

		edited, err := LoadRelationshipManifest(tmpFile)
		if err != nil {
			return fmt.Errorf("relationships-review: reload after edit: %w", err)
		}
		if edited != nil {
			discovered = edited
		}
	}

	// Print summary.
	summary := discovered.Summarize()
	fmt.Fprintf(os.Stderr, "%d relationships discovered, %d reviewed, %d accepted, %d rejected, %d pending\n",
		summary.Total, summary.Reviewed, summary.Accepted, summary.Rejected, summary.Pending)

	// Save the accepted manifest.
	savePath := acceptedPath
	if a.ExportTo != "" {
		savePath, err = filepath.Abs(a.ExportTo)
		if err != nil {
			return fmt.Errorf("relationships-review: resolve export path: %w", err)
		}
	}

	if err := SaveRelationshipManifest(discovered, savePath); err != nil {
		return fmt.Errorf("relationships-review: save: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Saved to %s\n", savePath)

	return nil
}

// ─── formatters ───────────────────────────────────────────────────────────────

// componentListEntryJSON is the per-component data in list/search JSON output.
type componentListEntryJSON struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	File          string `json:"file"`
	IncomingCount int    `json:"incoming_count"`
	OutgoingCount int    `json:"outgoing_count"`
	DetectedAt    string `json:"detected_at"`
}

// computeRelationshipCounts returns incoming/outgoing relationship counts per component ID.
func computeRelationshipCounts(reg *ComponentRegistry) (incoming, outgoing map[string]int) {
	incoming = make(map[string]int)
	outgoing = make(map[string]int)
	for _, rel := range reg.Relationships {
		outgoing[rel.FromComponent]++
		incoming[rel.ToComponent]++
	}
	return incoming, outgoing
}

// sortedComponentIDs returns all component IDs from the registry sorted alphabetically.
func sortedComponentIDs(reg *ComponentRegistry) []string {
	ids := make([]string, 0, len(reg.Components))
	for id := range reg.Components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// buildComponentListJSON converts component IDs to JSON entries.
func buildComponentListJSON(reg *ComponentRegistry, ids []string, incoming, outgoing map[string]int) []componentListEntryJSON {
	items := make([]componentListEntryJSON, 0, len(ids))
	for _, id := range ids {
		comp := reg.Components[id]
		if comp == nil {
			continue
		}
		items = append(items, componentListEntryJSON{
			ID:            comp.ID,
			Name:          comp.Name,
			Type:          string(comp.Type),
			File:          comp.FileRef,
			IncomingCount: incoming[comp.ID],
			OutgoingCount: outgoing[comp.ID],
			DetectedAt:    comp.DetectedAt.Format("2006-01-02"),
		})
	}
	return items
}

// formatComponentsListTable renders the component list in table format.
func formatComponentsListTable(reg *ComponentRegistry, ids []string, incoming, outgoing map[string]int) string {
	if len(ids) == 0 {
		return "No components found."
	}

	// Column widths.
	const colComponent = 24
	const colType = 10
	const colFile = 22
	const colRels = 20

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
		colComponent, "Component",
		colType, "Type",
		colFile, "File",
		"Relationships")

	sep := strings.Repeat("\u2500", colComponent+2+colType+2+colFile+2+colRels)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(sep)
	sb.WriteString("\n")

	for _, id := range ids {
		comp := reg.Components[id]
		if comp == nil {
			continue
		}
		name := comp.ID
		typ := string(comp.Type)
		f := comp.FileRef
		if len(name) > colComponent {
			name = name[:colComponent-1] + "\u2026"
		}
		if len(f) > colFile {
			f = f[:colFile-1] + "\u2026"
		}
		rels := fmt.Sprintf("%d incoming, %d outgoing", incoming[comp.ID], outgoing[comp.ID])
		fmt.Fprintf(&sb, "%-*s  %-*s  %-*s  %s\n", colComponent, name, colType, typ, colFile, f, rels)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// formatComponentInspectTable renders detailed component info in table format.
func formatComponentInspectTable(comp *RegistryComponent, incoming, outgoing []RegistryRelationship) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Component: %s\n", comp.ID)
	fmt.Fprintf(&sb, "  Name:     %s\n", comp.Name)
	fmt.Fprintf(&sb, "  Type:     %s\n", comp.Type)
	fmt.Fprintf(&sb, "  File:     %s\n", comp.FileRef)
	fmt.Fprintf(&sb, "  Detected: %s\n", comp.DetectedAt.Format("2006-01-02"))

	fmt.Fprintf(&sb, "\nDepends on (%d):\n", len(outgoing))
	if len(outgoing) == 0 {
		sb.WriteString("  (none)\n")
	}
	for _, r := range outgoing {
		sig := buildSignalSummary(r.Signals)
		fmt.Fprintf(&sb, "  %s [%.2f] %s\n", r.ToComponent, r.AggregatedConfidence, sig)
	}

	fmt.Fprintf(&sb, "\nDepended on by (%d):\n", len(incoming))
	if len(incoming) == 0 {
		sb.WriteString("  (none)\n")
	}
	for _, r := range incoming {
		sig := buildSignalSummary(r.Signals)
		fmt.Fprintf(&sb, "  %s [%.2f] %s\n", r.FromComponent, r.AggregatedConfidence, sig)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// formatRelationshipsTable renders relationships in human-readable table format.
func formatRelationshipsTable(rels []RegistryRelationship, from, to string, includeSignals bool) string {
	if len(rels) == 0 {
		return "No relationships found."
	}

	var sb strings.Builder

	switch {
	case from != "":
		fmt.Fprintf(&sb, "Relationships for %s:\n\nDepends on:\n", from)
		for _, r := range rels {
			sig := buildSignalSummary(r.Signals)
			fmt.Fprintf(&sb, "  %s [%.2f] %s\n", r.ToComponent, r.AggregatedConfidence, sig)
			if includeSignals {
				for _, s := range r.Signals {
					fmt.Fprintf(&sb, "    signal: %s (%.2f)\n", s.SourceType, s.Confidence)
				}
			}
		}
	case to != "":
		fmt.Fprintf(&sb, "Relationships for %s:\n\nDepended on by:\n", to)
		for _, r := range rels {
			sig := buildSignalSummary(r.Signals)
			fmt.Fprintf(&sb, "  %s [%.2f] %s\n", r.FromComponent, r.AggregatedConfidence, sig)
			if includeSignals {
				for _, s := range r.Signals {
					fmt.Fprintf(&sb, "    signal: %s (%.2f)\n", s.SourceType, s.Confidence)
				}
			}
		}
	default:
		fmt.Fprintf(&sb, "All relationships (%d):\n", len(rels))
		for _, r := range rels {
			sig := buildSignalSummary(r.Signals)
			fmt.Fprintf(&sb, "  %s → %s [%.2f] %s\n", r.FromComponent, r.ToComponent, r.AggregatedConfidence, sig)
			if includeSignals {
				for _, s := range r.Signals {
					fmt.Fprintf(&sb, "    signal: %s (%.2f)\n", s.SourceType, s.Confidence)
				}
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// formatRelationshipsDOT renders relationships as a Graphviz DOT digraph.
func formatRelationshipsDOT(rels []RegistryRelationship) string {
	var sb strings.Builder
	sb.WriteString("digraph relationships {\n")
	for _, r := range rels {
		penwidth := 0.5 + r.AggregatedConfidence*2.5
		fmt.Fprintf(&sb, "  %q -> %q [label=\"%.2f\", penwidth=\"%.2f\"];\n",
			r.FromComponent, r.ToComponent, r.AggregatedConfidence, penwidth)
	}
	sb.WriteString("}\n")
	return sb.String()
}

// buildSignalSummary returns a compact signal type listing.
func buildSignalSummary(signals []Signal) string {
	if len(signals) == 0 {
		return ""
	}
	types := make([]string, 0, len(signals))
	seen := make(map[string]bool)
	for _, s := range signals {
		t := string(s.SourceType)
		if !seen[t] {
			seen[t] = true
			types = append(types, t)
		}
	}
	return fmt.Sprintf("(%s)", strings.Join(types, "+"))
}

