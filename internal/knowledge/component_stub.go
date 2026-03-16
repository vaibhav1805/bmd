package knowledge

// Component is a stub definition for backward compatibility.
// The full component detection has been moved to the graphmd project.
type Component struct {
	ID         string
	Name       string
	File       string
	Confidence float64
	DetectedAt string
}

// ComponentRegistry is a stub for backward compatibility
type ComponentRegistry struct {
}

// LoadRegistry is a stub
func LoadRegistry(path string) (*ComponentRegistry, error) {
	return nil, nil
}

// SaveRegistry is a stub
func SaveRegistry(reg *ComponentRegistry, path string) error {
	return nil
}

// LoadComponentConfig is a stub
func LoadComponentConfig(path string) (*interface{}, error) {
	return nil, nil
}

// LoadAcceptedRelationships is a stub
func LoadAcceptedRelationships(path string) ([]*Edge, error) {
	return nil, nil
}

const (
	RegistryFileName         = ".bmd-registry.json"
	DiscoveredManifestFile   = ".bmd-relationships-discovered.yaml"
	AcceptedManifestFile     = ".bmd-relationships-accepted.yaml"
)

// Stubs for PageIndex and LLM-related functions moved to graphmd

type FileTree struct {
	ID       string
	Children []*FileTree
}

type PageIndexConfig struct {
	Model          string
	Bin            string
	CacheFile      string
	Timeout        int
}

func DefaultPageIndexConfig() *PageIndexConfig {
	return &PageIndexConfig{
		Model: "claude-sonnet-4-5",
		Bin: "pageindex",
		Timeout: 30,
	}
}

func LoadTreeFiles(dir string) ([]FileTree, error) {
	return nil, nil
}

func RunPageIndexQuery(config *PageIndexConfig, query string, trees []FileTree) (*interface{}, error) {
	return nil, nil
}

var ErrPageIndexNotFound error

func NewComponentDetector() *interface{} {
	return nil
}

func NewComponentDetectorWithConfig(cfg *interface{}) *interface{} {
	return nil
}

func filenameStem(path string) string {
	return path
}
