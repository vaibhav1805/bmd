package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/editor"
	"github.com/bmd/bmd/internal/theme"
)

// Helper function to create a test document with given lines.
func createTestDocument(lines []string) *ast.Document {
	return &ast.Document{}
}

// TestEditModeToggle tests entering and exiting edit mode.
func TestEditModeToggle(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"# Header", "This is a test."}

	// Initially not in edit mode
	if v.editMode {
		t.Error("Expected editMode to be false initially")
	}

	// Manually set edit mode (since we can't easily test key presses)
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	output := v.renderEditMode()
	if !strings.Contains(output, "[EDIT MODE]") {
		t.Error("Expected [EDIT MODE] indicator in edit mode output")
	}

	// Exit edit mode
	v.editMode = false
	// Output should no longer contain [EDIT MODE] (it will show rendered view)
}

// TestEditModeTextInsertion tests character insertion in edit mode.
func TestEditModeTextInsertion(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	v.editBuffer.Insert('X')

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "Xhello" {
		t.Errorf("Expected 'Xhello', got '%s'", resultLines[0])
	}
}

// TestEditModeUndo tests undo in edit mode.
func TestEditModeUndo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	v.editBuffer.Insert('X')
	v.editBuffer.Undo()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "hello" {
		t.Errorf("Expected 'hello' after undo, got '%s'", resultLines[0])
	}
}

// TestEditModeSave tests file persistence setup.
func TestEditModeSave(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"original content"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Make an edit
	v.editBuffer.Insert('X')

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "Xoriginal content" {
		t.Errorf("Expected 'Xoriginal content', got '%s'", resultLines[0])
	}
}

// TestEditModeNavigation tests cursor movement in edit mode.
func TestEditModeNavigation(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Test CursorDown
	v.editBuffer.CursorDown()
	if v.editBuffer.CursorLine() != 1 {
		t.Errorf("Expected cursor at line 1, got %d", v.editBuffer.CursorLine())
	}

	// Test CursorRight
	v.editBuffer.CursorRight()
	if v.editBuffer.CursorCol() != 1 {
		t.Errorf("Expected cursor at col 1, got %d", v.editBuffer.CursorCol())
	}
}

// TestEditModeDeleteKey tests delete key functionality in edit mode.
func TestEditModeDeleteKey(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Delete at position 0
	v.editBuffer.Delete()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "ello" {
		t.Errorf("Expected 'ello' after delete, got '%s'", resultLines[0])
	}
}

// TestEditModeBackspaceKey tests backspace key functionality in edit mode.
func TestEditModeBackspaceKey(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to position 2
	v.editBuffer.CursorRight()
	v.editBuffer.CursorRight()

	// Backspace to delete 'e'
	v.editBuffer.Backspace()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "hllo" {
		t.Errorf("Expected 'hllo' after backspace, got '%s'", resultLines[0])
	}
}

// TestEditModeEnterNewLine tests Enter key functionality in edit mode.
func TestEditModeEnterNewLine(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to position 2 (between 'l' and 'l')
	v.editBuffer.CursorRight()
	v.editBuffer.CursorRight()

	// Enter new line
	v.editBuffer.EnterNewLine()

	resultLines := v.editBuffer.GetLines()
	if len(resultLines) != 2 {
		t.Errorf("Expected 2 lines after EnterNewLine, got %d", len(resultLines))
	}

	if resultLines[0] != "he" {
		t.Errorf("Expected 'he', got '%s'", resultLines[0])
	}

	if resultLines[1] != "llo" {
		t.Errorf("Expected 'llo', got '%s'", resultLines[1])
	}
}

// TestEditModeRedo tests redo functionality in edit mode.
func TestEditModeRedo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Type, undo, then redo
	v.editBuffer.Insert('X')
	v.editBuffer.Undo()
	v.editBuffer.Redo()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "Xhello" {
		t.Errorf("Expected 'Xhello' after undo then redo, got '%s'", resultLines[0])
	}
}

// TestEditModeMultilineEdit tests editing with multiple lines.
func TestEditModeMultilineEdit(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to second line
	v.editBuffer.CursorDown()

	// Insert character
	v.editBuffer.Insert('X')

	resultLines := v.editBuffer.GetLines()
	if resultLines[1] != "Xline2" {
		t.Errorf("Expected 'Xline2' on line 2, got '%s'", resultLines[1])
	}

	// Line 1 should be unchanged
	if resultLines[0] != "line1" {
		t.Errorf("Expected 'line1' on line 1, got '%s'", resultLines[0])
	}
}

// TestEditModeCanUndo tests the CanUndo query method.
func TestEditModeCanUndo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Initially no undo available
	if v.editBuffer.CanUndo() {
		t.Error("Expected CanUndo to be false initially")
	}

	// After insert, undo should be available
	v.editBuffer.Insert('X')
	if !v.editBuffer.CanUndo() {
		t.Error("Expected CanUndo to be true after insert")
	}

	// After undo, undo should not be available
	v.editBuffer.Undo()
	if v.editBuffer.CanUndo() {
		t.Error("Expected CanUndo to be false after undo")
	}
}

// TestEditModeCanRedo tests the CanRedo query method.
func TestEditModeCanRedo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Initially no redo available
	if v.editBuffer.CanRedo() {
		t.Error("Expected CanRedo to be false initially")
	}

	// After insert and undo, redo should be available
	v.editBuffer.Insert('X')
	v.editBuffer.Undo()
	if !v.editBuffer.CanRedo() {
		t.Error("Expected CanRedo to be true after undo")
	}

	// After redo, redo should not be available
	v.editBuffer.Redo()
	if v.editBuffer.CanRedo() {
		t.Error("Expected CanRedo to be false after redo")
	}
}

// TestEditModeJumpToStart tests jumping to document start.
func TestEditModeJumpToStart(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to end
	v.editBuffer.JumpToEnd()

	// Jump back to start
	v.editBuffer.JumpToStart()

	if v.editBuffer.CursorLine() != 0 || v.editBuffer.CursorCol() != 0 {
		t.Errorf("Expected cursor at (0, 0) after JumpToStart, got (%d, %d)",
			v.editBuffer.CursorLine(), v.editBuffer.CursorCol())
	}
}

// TestEditModeJumpToEnd tests jumping to document end.
func TestEditModeJumpToEnd(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Jump to end
	v.editBuffer.JumpToEnd()

	if v.editBuffer.CursorLine() != 2 {
		t.Errorf("Expected cursor at line 2 after JumpToEnd, got %d", v.editBuffer.CursorLine())
	}

	if v.editBuffer.CursorCol() != 5 { // "line3" is 5 chars
		t.Errorf("Expected cursor at col 5 after JumpToEnd, got %d", v.editBuffer.CursorCol())
	}
}

// TestEditModeJumpToLine tests jumping to a specific line.
func TestEditModeJumpToLine(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Jump to line 1 (0-based)
	v.editBuffer.JumpToLine(1)

	if v.editBuffer.CursorLine() != 1 {
		t.Errorf("Expected cursor at line 1 after JumpToLine(1), got %d", v.editBuffer.CursorLine())
	}

	if v.editBuffer.CursorCol() != 0 {
		t.Errorf("Expected cursor at col 0 after JumpToLine(1), got %d", v.editBuffer.CursorCol())
	}
}

// TestEditModeRenderEditMode tests the renderEditMode output.
func TestEditModeRenderEditMode(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello world", "second line"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	output := v.renderEditMode()

	// Check that output contains expected elements
	if !strings.Contains(output, "[EDIT MODE]") {
		t.Error("Expected [EDIT MODE] in output")
	}

	if !strings.Contains(output, "test.md") {
		t.Error("Expected filename 'test.md' in output")
	}

	// Should contain at least one line number
	if !strings.Contains(output, "|") {
		t.Error("Expected line number separator '|' in output")
	}
}

// TestEditModeSetLines tests the SetLines method (for undo/redo).
func TestEditModeSetLines(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"original1", "original2"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Change the lines
	newLines := []string{"new1", "new2", "new3"}
	v.editBuffer.SetLines(newLines)

	resultLines := v.editBuffer.GetLines()
	if len(resultLines) != 3 {
		t.Errorf("Expected 3 lines after SetLines, got %d", len(resultLines))
	}

	if resultLines[0] != "new1" {
		t.Errorf("Expected 'new1', got '%s'", resultLines[0])
	}

	if resultLines[2] != "new3" {
		t.Errorf("Expected 'new3', got '%s'", resultLines[2])
	}
}

// TestEditModeGetLines tests the GetLines method returns a copy.
func TestEditModeGetLines(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Get lines and modify the returned slice
	retrieved := v.editBuffer.GetLines()
	retrieved[0] = "modified"

	// Original should be unchanged
	newRetrieved := v.editBuffer.GetLines()
	if newRetrieved[0] != "line1" {
		t.Errorf("Expected 'line1', but GetLines returned modified value")
	}
}

// ─── Directory Browser Tests (DIR-01) ────────────────────────────────────────

// makeTempDir creates a temporary directory with optional .md files for testing.
// Returns the directory path; caller is responsible for cleanup (os.RemoveAll).
func makeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "bmd-dir-test-*")
	if err != nil {
		t.Fatalf("makeTempDir: %v", err)
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if parent := filepath.Dir(p); parent != dir {
			if mkErr := os.MkdirAll(parent, 0o755); mkErr != nil {
				t.Fatalf("makeTempDir MkdirAll: %v", mkErr)
			}
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("makeTempDir WriteFile: %v", err)
		}
	}
	return dir
}

// TestNewDirectoryViewer verifies the constructor produces a viewer in directory mode.
func TestNewDirectoryViewer(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 80)
	if !v.directoryMode {
		t.Error("Expected directoryMode=true after NewDirectoryViewer")
	}
	if v.directoryState.RootPath != "/tmp" {
		t.Errorf("Expected RootPath=/tmp, got %q", v.directoryState.RootPath)
	}
	if v.directoryState.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex=0, got %d", v.directoryState.SelectedIndex)
	}
}

// TestLoadDirectoryBasic verifies that LoadDirectory discovers .md files.
func TestLoadDirectoryBasic(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"README.md":  "# README\nHello world\n",
		"notes.md":   "# Notes\nSome notes here\n",
		"ignore.txt": "not a markdown file",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) != 2 {
		t.Errorf("Expected 2 .md files, got %d", len(v.directoryState.Files))
	}
}

// TestLoadDirectoryIgnoresNonMd verifies only .md files are collected.
func TestLoadDirectoryIgnoresNonMd(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md":     "markdown",
		"b.txt":    "text",
		"c.go":     "go code",
		"d.json":   "{}",
		"e.md":     "also markdown",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) != 2 {
		t.Errorf("Expected 2 .md files, got %d", len(v.directoryState.Files))
	}
	for _, f := range v.directoryState.Files {
		if filepath.Ext(f.Path) != ".md" {
			t.Errorf("Non-.md file found: %s", f.Path)
		}
	}
}

// TestLoadDirectoryRecursive verifies recursive directory scanning.
func TestLoadDirectoryRecursive(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"root.md":          "root level",
		"docs/api.md":      "api docs",
		"docs/guide.md":    "guide",
		"docs/sub/deep.md": "deep nested",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) != 4 {
		t.Errorf("Expected 4 .md files (recursive), got %d", len(v.directoryState.Files))
	}
}

// TestLoadDirectorySortedByName verifies files are sorted alphabetically.
func TestLoadDirectorySortedByName(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"zoo.md":   "z",
		"alpha.md": "a",
		"beta.md":  "b",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	names := make([]string, len(v.directoryState.Files))
	for i, f := range v.directoryState.Files {
		names[i] = f.Name
	}
	if names[0] != "alpha.md" || names[1] != "beta.md" || names[2] != "zoo.md" {
		t.Errorf("Expected alphabetical sort, got %v", names)
	}
}

// TestLoadDirectoryMetadataSize verifies that Size is set from file content.
func TestLoadDirectoryMetadataSize(t *testing.T) {
	content := "# Hello World\nThis is a test file.\n"
	dir := makeTempDir(t, map[string]string{
		"test.md": content,
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) == 0 {
		t.Fatal("No files loaded")
	}
	f := v.directoryState.Files[0]
	if f.Size != int64(len(content)) {
		t.Errorf("Expected Size=%d, got %d", len(content), f.Size)
	}
}

// TestLoadDirectoryMetadataLineCount verifies line count computation.
func TestLoadDirectoryMetadataLineCount(t *testing.T) {
	content := "line1\nline2\nline3\n"
	dir := makeTempDir(t, map[string]string{
		"test.md": content,
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) == 0 {
		t.Fatal("No files loaded")
	}
	f := v.directoryState.Files[0]
	if f.LineCount != 3 {
		t.Errorf("Expected LineCount=3 for 3-line file, got %d", f.LineCount)
	}
}

// TestLoadDirectoryMetadataLineCountNoTrailingNewline verifies line count without trailing newline.
func TestLoadDirectoryMetadataLineCountNoTrailingNewline(t *testing.T) {
	content := "line1\nline2\nline3"
	dir := makeTempDir(t, map[string]string{
		"test.md": content,
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	f := v.directoryState.Files[0]
	if f.LineCount != 3 {
		t.Errorf("Expected LineCount=3 for 3 lines without trailing newline, got %d", f.LineCount)
	}
}

// TestLoadDirectoryMetadataModTime verifies ModTime is populated and recent.
func TestLoadDirectoryMetadataModTime(t *testing.T) {
	before := time.Now().Add(-time.Second)
	dir := makeTempDir(t, map[string]string{
		"test.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	f := v.directoryState.Files[0]
	if f.ModTime.Before(before) {
		t.Errorf("ModTime %v is before test start %v", f.ModTime, before)
	}
}

// TestLoadDirectoryMetadataPreview verifies the first-100-chars preview.
func TestLoadDirectoryMetadataPreview(t *testing.T) {
	longContent := strings.Repeat("x", 200)
	dir := makeTempDir(t, map[string]string{
		"test.md": longContent,
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	f := v.directoryState.Files[0]
	if len(f.Preview) != 100 {
		t.Errorf("Expected Preview length 100, got %d", len(f.Preview))
	}
}

// TestLoadDirectoryRelativeName verifies Name is relative to root.
func TestLoadDirectoryRelativeName(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"docs/api.md": "api",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) == 0 {
		t.Fatal("No files loaded")
	}
	f := v.directoryState.Files[0]
	// Name should be "docs/api.md" (relative path)
	if f.Name != filepath.Join("docs", "api.md") {
		t.Errorf("Expected relative name %q, got %q", filepath.Join("docs", "api.md"), f.Name)
	}
}

// TestLoadDirectoryEmptyDirectory verifies empty directory results in zero files.
func TestLoadDirectoryEmptyDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"notes.txt": "not markdown",
		"data.json": "{}",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) != 0 {
		t.Errorf("Expected 0 files in dir with no .md files, got %d", len(v.directoryState.Files))
	}
}

// TestRenderDirectoryListingBasic verifies the directory listing renders without crashing.
func TestRenderDirectoryListingBasic(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"README.md": "# README\nContent here\n",
		"notes.md":  "# Notes\nMore content\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderDirectoryListing(22)
	if output == "" {
		t.Error("Expected non-empty render output")
	}
	// Should contain at least one filename
	if !strings.Contains(output, "README.md") && !strings.Contains(output, "notes.md") {
		t.Error("Expected at least one filename in render output")
	}
}

// TestRenderDirectoryListingCursorVisible verifies cursor ">" is shown for selected item.
func TestRenderDirectoryListingCursorVisible(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "first",
		"bbb.md": "second",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// SelectedIndex is 0 (aaa.md) by default
	output := v.renderDirectoryListing(22)
	if !strings.Contains(output, "> ") {
		t.Error("Expected '>' cursor prefix for selected item")
	}
}

// TestRenderDirectoryListingEmptyDir verifies "no files" message when no .md files.
func TestRenderDirectoryListingEmptyDir(t *testing.T) {
	dir := makeTempDir(t, map[string]string{})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderDirectoryListing(22)
	if !strings.Contains(output, "No markdown files") {
		t.Errorf("Expected 'No markdown files' message, got: %q", output)
	}
}

// TestRenderDirectoryListingHeader verifies header shows file count.
func TestRenderDirectoryListingHeader(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "first",
		"b.md": "second",
		"c.md": "third",
	})
	defer os.RemoveAll(dir)

	// Use a wide terminal width so the header is not truncated.
	v := NewDirectoryViewer(dir, theme.NewTheme(), 200)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderDirectoryListing(22)
	if !strings.Contains(output, "3 files") {
		t.Errorf("Expected '3 files' in header, got: %q", stripANSI(output[:min(300, len(output))]))
	}
}

// TestRenderDirectoryListingFooterHints verifies footer has keyboard hints.
func TestRenderDirectoryListingFooterHints(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderDirectoryListing(22)
	if !strings.Contains(output, "Navigate") && !strings.Contains(output, "↑") && !strings.Contains(output, "Open") {
		t.Error("Expected keyboard hints in footer")
	}
}

// TestNavigationMoveDown verifies ↓ moves the selection cursor down.
func TestNavigationMoveDown(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "first",
		"bbb.md": "second",
		"ccc.md": "third",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Initial state: index 0
	if v.directoryState.SelectedIndex != 0 {
		t.Fatalf("Expected initial SelectedIndex=0, got %d", v.directoryState.SelectedIndex)
	}

	// Simulate pressing "down"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	result, _ := v.updateDirectory(msg)
	vv := result.(Viewer)

	if vv.directoryState.SelectedIndex != 1 {
		t.Errorf("Expected SelectedIndex=1 after down, got %d", vv.directoryState.SelectedIndex)
	}
}

// TestNavigationMoveUp verifies ↑ moves the selection cursor up.
func TestNavigationMoveUp(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "first",
		"bbb.md": "second",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryState.SelectedIndex = 1 // start at second file

	// Simulate pressing "up"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	result, _ := v.updateDirectory(msg)
	vv := result.(Viewer)

	if vv.directoryState.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex=0 after up, got %d", vv.directoryState.SelectedIndex)
	}
}

// TestNavigationWrapBottom verifies cursor wraps from last file to first.
func TestNavigationWrapBottom(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "first",
		"bbb.md": "second",
		"ccc.md": "third",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryState.SelectedIndex = 2 // last file

	// Press down: should wrap to 0
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	result, _ := v.updateDirectory(msg)
	vv := result.(Viewer)

	if vv.directoryState.SelectedIndex != 0 {
		t.Errorf("Expected wraparound to 0, got %d", vv.directoryState.SelectedIndex)
	}
}

// TestNavigationWrapTop verifies cursor wraps from first file to last.
func TestNavigationWrapTop(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "first",
		"bbb.md": "second",
		"ccc.md": "third",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryState.SelectedIndex = 0 // first file

	// Press up: should wrap to last (index 2)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	result, _ := v.updateDirectory(msg)
	vv := result.(Viewer)

	if vv.directoryState.SelectedIndex != 2 {
		t.Errorf("Expected wraparound to 2 (last), got %d", vv.directoryState.SelectedIndex)
	}
}

// TestNavigationEmptyDirNocrash verifies navigation with empty file list doesn't crash.
func TestNavigationEmptyDirNocrash(t *testing.T) {
	dir := makeTempDir(t, map[string]string{})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Pressing down/up with no files should not panic
	for _, key := range []string{"j", "k"} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		result, _ := v.updateDirectory(msg)
		vv := result.(Viewer)
		if vv.directoryState.SelectedIndex != 0 {
			t.Errorf("Empty dir: expected SelectedIndex=0 after %q, got %d", key, vv.directoryState.SelectedIndex)
		}
	}
}

// TestStressLoadDirectory verifies 50+ files load and render without issues.
func TestStressLoadDirectory(t *testing.T) {
	files := make(map[string]string)
	for i := 0; i < 55; i++ {
		name := fmt.Sprintf("file%03d.md", i)
		content := fmt.Sprintf("# File %d\n%s\n", i, strings.Repeat("content line\n", 10))
		files[name] = content
	}

	dir := makeTempDir(t, files)
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) != 55 {
		t.Errorf("Expected 55 files, got %d", len(v.directoryState.Files))
	}

	// Render should not panic
	output := v.renderDirectoryListing(22)
	if output == "" {
		t.Error("Expected non-empty render for 55 files")
	}

	// Navigate through all files without panic
	for i := 0; i < 60; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		result, _ := v.updateDirectory(msg)
		v = result.(Viewer)
	}
	if v.directoryState.SelectedIndex < 0 || v.directoryState.SelectedIndex >= 55 {
		t.Errorf("SelectedIndex %d out of range after stress navigation", v.directoryState.SelectedIndex)
	}
}

// TestDirectoryModeInView verifies View() routes to directory listing when directoryMode=true.
func TestDirectoryModeInView(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"test.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.View()
	// In directory mode the output should contain file names, not a rendered document
	if !strings.Contains(output, "test.md") {
		t.Error("Expected 'test.md' in View() output when in directory mode")
	}
	// Should not contain [EDIT MODE]
	if strings.Contains(output, "[EDIT MODE]") {
		t.Error("Unexpected [EDIT MODE] in directory view")
	}
}

// TestDirectoryModeSelectedIndexStaysInBounds verifies SelectedIndex is always valid.
func TestDirectoryModeSelectedIndexStaysInBounds(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "a",
		"b.md": "b",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Navigate 100 times in each direction — index should always be in [0, n-1]
	for i := 0; i < 50; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		result, _ := v.updateDirectory(msg)
		v = result.(Viewer)
		if v.directoryState.SelectedIndex < 0 || v.directoryState.SelectedIndex >= 2 {
			t.Errorf("SelectedIndex %d out of bounds at iteration %d (down)", v.directoryState.SelectedIndex, i)
		}
	}
	for i := 0; i < 50; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
		result, _ := v.updateDirectory(msg)
		v = result.(Viewer)
		if v.directoryState.SelectedIndex < 0 || v.directoryState.SelectedIndex >= 2 {
			t.Errorf("SelectedIndex %d out of bounds at iteration %d (up)", v.directoryState.SelectedIndex, i)
		}
	}
}

// TestDirectoryMetadataNameIsSet verifies Name field is populated.
func TestDirectoryMetadataNameIsSet(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"my-doc.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(v.directoryState.Files) == 0 {
		t.Fatal("No files loaded")
	}
	f := v.directoryState.Files[0]
	if f.Name == "" {
		t.Error("Expected non-empty Name for discovered file")
	}
	if f.Name != "my-doc.md" {
		t.Errorf("Expected Name='my-doc.md', got %q", f.Name)
	}
}
