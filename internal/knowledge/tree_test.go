package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadTreeFile(t *testing.T) {
	dir := t.TempDir()

	// Build a FileTree with nested children.
	ft := FileTree{
		File: "docs/api.md",
		Root: &TreeNode{
			Heading:   "",
			Summary:   "Top-level API documentation",
			LineStart: 0,
			LineEnd:   100,
			Children: []*TreeNode{
				{
					Heading:   "## Installation",
					Summary:   "How to install the package",
					LineStart: 10,
					LineEnd:   30,
					Children: []*TreeNode{
						{
							Heading:   "### From Source",
							Summary:   "Build from source instructions",
							LineStart: 15,
							LineEnd:   28,
						},
					},
				},
				{
					Heading:   "## Usage",
					Summary:   "How to use the package",
					LineStart: 31,
					LineEnd:   80,
				},
			},
		},
	}

	// Save the tree.
	if err := SaveTreeFile(dir, ft); err != nil {
		t.Fatalf("SaveTreeFile: %v", err)
	}

	// Verify the file was created.
	expected := filepath.Join(dir, "api.bmd-tree.json")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("expected tree file %q not created", expected)
	}

	// Load it back.
	trees, err := LoadTreeFiles(dir)
	if err != nil {
		t.Fatalf("LoadTreeFiles: %v", err)
	}
	if len(trees) != 1 {
		t.Fatalf("expected 1 tree, got %d", len(trees))
	}

	loaded := trees[0]

	// Assert File field round-trips.
	if loaded.File != ft.File {
		t.Errorf("File: got %q, want %q", loaded.File, ft.File)
	}

	// Assert root heading and summary.
	if loaded.Root == nil {
		t.Fatal("loaded Root is nil")
	}
	if loaded.Root.Heading != ft.Root.Heading {
		t.Errorf("Root.Heading: got %q, want %q", loaded.Root.Heading, ft.Root.Heading)
	}
	if loaded.Root.Summary != ft.Root.Summary {
		t.Errorf("Root.Summary: got %q, want %q", loaded.Root.Summary, ft.Root.Summary)
	}

	// Assert line offsets.
	if loaded.Root.LineStart != ft.Root.LineStart {
		t.Errorf("Root.LineStart: got %d, want %d", loaded.Root.LineStart, ft.Root.LineStart)
	}
	if loaded.Root.LineEnd != ft.Root.LineEnd {
		t.Errorf("Root.LineEnd: got %d, want %d", loaded.Root.LineEnd, ft.Root.LineEnd)
	}

	// Assert child count and nested child.
	if len(loaded.Root.Children) != 2 {
		t.Fatalf("Root.Children count: got %d, want 2", len(loaded.Root.Children))
	}
	installChild := loaded.Root.Children[0]
	if installChild.Heading != "## Installation" {
		t.Errorf("Children[0].Heading: got %q, want %q", installChild.Heading, "## Installation")
	}
	if len(installChild.Children) != 1 {
		t.Fatalf("Children[0].Children count: got %d, want 1", len(installChild.Children))
	}
	grandchild := installChild.Children[0]
	if grandchild.Heading != "### From Source" {
		t.Errorf("grandchild.Heading: got %q, want %q", grandchild.Heading, "### From Source")
	}
}

func TestLoadTreeFiles_Empty(t *testing.T) {
	dir := t.TempDir()

	trees, err := LoadTreeFiles(dir)
	if err != nil {
		t.Fatalf("LoadTreeFiles on empty dir should not error: %v", err)
	}
	if len(trees) != 0 {
		t.Errorf("expected 0 trees in empty dir, got %d", len(trees))
	}
}

func TestLoadTreeFiles_SkipsMalformed(t *testing.T) {
	dir := t.TempDir()

	// Write one valid tree file.
	validFT := FileTree{
		File: "valid.md",
		Root: &TreeNode{
			Heading:   "",
			Summary:   "Valid file",
			LineStart: 0,
			LineEnd:   50,
		},
	}
	if err := SaveTreeFile(dir, validFT); err != nil {
		t.Fatalf("SaveTreeFile: %v", err)
	}

	// Write one malformed JSON file with .bmd-tree.json extension.
	malformedPath := filepath.Join(dir, "broken.bmd-tree.json")
	if err := os.WriteFile(malformedPath, []byte("not valid json {{{"), 0644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	// Load: should return only the valid tree, skipping the malformed one.
	trees, err := LoadTreeFiles(dir)
	if err != nil {
		t.Fatalf("LoadTreeFiles: %v", err)
	}
	if len(trees) != 1 {
		t.Errorf("expected 1 valid tree (malformed skipped), got %d", len(trees))
	}
	if trees[0].File != "valid.md" {
		t.Errorf("expected valid.md, got %q", trees[0].File)
	}
}

func TestTreeNode_NilChildren(t *testing.T) {
	node := TreeNode{
		Heading:   "## Section",
		Summary:   "A section with no children",
		LineStart: 5,
		LineEnd:   20,
		Children:  nil, // omitempty should suppress this field
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// "children" key should not appear in the JSON output.
	jsonStr := string(data)
	if contains := func(s, sub string) bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}; contains(jsonStr, `"children"`) {
		t.Errorf("JSON should not contain 'children' key when nil, got: %s", jsonStr)
	}
}
