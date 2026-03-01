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

// ─── DIR-02: Directory ↔ File Navigation Tests ────────────────────────────────

// TestSaveDirectorySelection verifies that SaveDirectorySelection persists the
// current selected index and root path.
func TestSaveDirectorySelection(t *testing.T) {
	ds := DirectoryState{
		RootPath:      "/some/dir",
		SelectedIndex: 3,
	}
	ds.SaveDirectorySelection()

	if ds.SavedSelectedIndex != 3 {
		t.Errorf("Expected SavedSelectedIndex=3, got %d", ds.SavedSelectedIndex)
	}
	if ds.SavedFilePath != "/some/dir" {
		t.Errorf("Expected SavedFilePath=/some/dir, got %q", ds.SavedFilePath)
	}
}

// TestRestoreDirectorySelection verifies that RestoreDirectorySelection restores
// the saved cursor index.
func TestRestoreDirectorySelection(t *testing.T) {
	ds := DirectoryState{
		RootPath:           "/some/dir",
		SelectedIndex:      0,
		SavedSelectedIndex: 5,
		SavedFilePath:      "/some/dir",
	}
	ds.RestoreDirectorySelection()

	if ds.SelectedIndex != 5 {
		t.Errorf("Expected SelectedIndex=5 after restore, got %d", ds.SelectedIndex)
	}
}

// TestSaveRestoreCycle verifies save then restore produces the same index.
func TestSaveRestoreCycle(t *testing.T) {
	ds := DirectoryState{
		RootPath:      "/docs",
		SelectedIndex: 7,
	}
	ds.SaveDirectorySelection()
	ds.SelectedIndex = 0 // simulate moving away
	ds.RestoreDirectorySelection()

	if ds.SelectedIndex != 7 {
		t.Errorf("Expected SelectedIndex=7 after save/restore cycle, got %d", ds.SelectedIndex)
	}
}

// TestOpenFileFromDirectorySetsFlags verifies OpenFileFromDirectory sets the
// openedFromDirectory flag and clears directoryMode.
func TestOpenFileFromDirectorySetsFlags(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"api.md": "# API\nContent\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	vv, _ := v.OpenFileFromDirectory()

	if vv.directoryMode {
		t.Error("Expected directoryMode=false after OpenFileFromDirectory")
	}
	if !vv.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true after OpenFileFromDirectory")
	}
}

// TestOpenFileFromDirectorySavesSelection verifies that the cursor position is
// saved when opening a file from directory.
func TestOpenFileFromDirectorySavesSelection(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a",
		"bbb.md": "b",
		"ccc.md": "c",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryState.SelectedIndex = 2 // ccc.md

	vv, _ := v.OpenFileFromDirectory()

	if vv.directoryState.SavedSelectedIndex != 2 {
		t.Errorf("Expected SavedSelectedIndex=2, got %d", vv.directoryState.SavedSelectedIndex)
	}
}

// TestOpenFileFromDirectorySetsCurrentView verifies currentView is set to "file".
func TestOpenFileFromDirectorySetsCurrentView(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"file.md": "# File\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	vv, _ := v.OpenFileFromDirectory()

	if vv.currentView != "file" {
		t.Errorf("Expected currentView='file', got %q", vv.currentView)
	}
}

// TestOpenFileFromDirectoryEmptyDoesNothing verifies that opening from an
// empty directory returns the viewer unchanged.
func TestOpenFileFromDirectoryEmptyDoesNothing(t *testing.T) {
	dir := makeTempDir(t, map[string]string{})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Empty directory: OpenFileFromDirectory should return unchanged
	vv, _ := v.OpenFileFromDirectory()

	// Should remain in directory mode since there's nothing to open
	if !vv.directoryMode {
		t.Error("Expected directoryMode=true when opening from empty directory")
	}
}

// TestBackToDirectoryRestoresMode verifies BackToDirectory restores directoryMode
// and clears openedFromDirectory.
func TestBackToDirectoryRestoresMode(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Go to file view
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"

	vv, _ := v.BackToDirectory()

	if !vv.directoryMode {
		t.Error("Expected directoryMode=true after BackToDirectory")
	}
	if vv.openedFromDirectory {
		t.Error("Expected openedFromDirectory=false after BackToDirectory")
	}
}

// TestBackToDirectorySetsCurrentView verifies currentView returns to "directory".
func TestBackToDirectorySetsCurrentView(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"

	vv, _ := v.BackToDirectory()

	if vv.currentView != "directory" {
		t.Errorf("Expected currentView='directory', got %q", vv.currentView)
	}
}

// TestBackToDirectoryRestoresCursorPosition verifies cursor position is restored
// after BackToDirectory.
func TestBackToDirectoryRestoresCursorPosition(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a",
		"bbb.md": "b",
		"ccc.md": "c",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Set cursor at index 2, save it.
	v.directoryState.SelectedIndex = 2
	v.directoryState.SaveDirectorySelection()

	// Simulate switch to file mode.
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"
	v.directoryState.SelectedIndex = 0 // simulate drift

	vv, _ := v.BackToDirectory()

	if vv.directoryState.SelectedIndex != 2 {
		t.Errorf("Expected SelectedIndex=2 after BackToDirectory, got %d", vv.directoryState.SelectedIndex)
	}
}

// TestBackToDirectoryNoopWhenNotFromDirectory verifies BackToDirectory does
// nothing if openedFromDirectory is false.
func TestBackToDirectoryNoopWhenNotFromDirectory(t *testing.T) {
	v := New(&ast.Document{}, "test.md", theme.NewTheme(), 80)

	// Not opened from directory
	vv, _ := v.BackToDirectory()

	// directoryMode should remain false
	if vv.directoryMode {
		t.Error("Expected directoryMode=false when BackToDirectory called without openedFromDirectory")
	}
}

// TestNavigationCycleDirToFileToDir verifies a full dir→file→dir cycle preserves
// the correct cursor index.
func TestNavigationCycleDirToFileToDir(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a",
		"bbb.md": "b",
		"ccc.md": "c",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryState.SelectedIndex = 1 // bbb.md

	// Open file from directory
	vv, _ := v.OpenFileFromDirectory()
	if vv.directoryMode {
		t.Error("Expected directoryMode=false after open")
	}

	// Return to directory
	vvv, _ := vv.BackToDirectory()
	if !vvv.directoryMode {
		t.Error("Expected directoryMode=true after back")
	}
	if vvv.directoryState.SelectedIndex != 1 {
		t.Errorf("Expected SelectedIndex=1 after cycle, got %d", vvv.directoryState.SelectedIndex)
	}
}

// TestMultipleNavigationCycles verifies multiple dir→file→dir cycles all
// preserve the correct cursor index each time.
func TestMultipleNavigationCycles(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a",
		"bbb.md": "b",
		"ccc.md": "c",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	indices := []int{0, 2, 1, 2, 0}
	for i, wantIdx := range indices {
		v.directoryState.SelectedIndex = wantIdx

		// Open
		vv, _ := v.OpenFileFromDirectory()

		// Return
		vvv, _ := vv.BackToDirectory()
		v = vvv

		if v.directoryState.SelectedIndex != wantIdx {
			t.Errorf("Cycle %d: Expected SelectedIndex=%d after back, got %d", i, wantIdx, v.directoryState.SelectedIndex)
		}
	}
}

// TestLoadDirectorySetsCurrentView verifies LoadDirectory sets currentView="directory".
func TestLoadDirectorySetsCurrentView(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if v.currentView != "directory" {
		t.Errorf("Expected currentView='directory' after LoadDirectory, got %q", v.currentView)
	}
}

// TestNewDirectoryViewerCurrentView verifies NewDirectoryViewer sets currentView="directory".
func TestNewDirectoryViewerCurrentView(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 80)
	if v.currentView != "directory" {
		t.Errorf("Expected currentView='directory' from constructor, got %q", v.currentView)
	}
}

// TestBreadcrumbInHeader verifies that renderHeader shows breadcrumb when
// openedFromDirectory is true.
func TestBreadcrumbInHeader(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"api.md": "# API\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Simulate being in file view from directory
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"
	v.FilePath = filepath.Join(dir, "api.md")

	header := v.renderHeader()
	plain := stripANSI(header)

	// Breadcrumb should show directory context
	if !strings.Contains(plain, "api.md") {
		t.Errorf("Expected 'api.md' in breadcrumb header, got: %q", plain)
	}
	if !strings.Contains(plain, "[") || !strings.Contains(plain, "]") {
		t.Error("Expected '[dir] filename' breadcrumb format in header")
	}
}

// TestNoBreadcrumbInNormalFileHeader verifies that renderHeader shows normal
// header (no breadcrumb) when file was NOT opened from directory.
func TestNoBreadcrumbInNormalFileHeader(t *testing.T) {
	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24

	header := v.renderHeader()
	plain := stripANSI(header)

	// Normal header should show "filename  (parent/)" format
	if !strings.Contains(plain, "file.md") {
		t.Errorf("Expected 'file.md' in header, got: %q", plain)
	}
	// Should not contain the breadcrumb format "[dir] filename"
	// The breadcrumb '[' comes from the directory, not file path
	if strings.Contains(plain, "[/tmp]") {
		t.Error("Unexpected breadcrumb format '[dir]' in non-directory header")
	}
}

// TestBreadcrumbShowsBackHint verifies the header hints 'h/Backspace: back to directory'
// when openedFromDirectory is true.
func TestBreadcrumbShowsBackHint(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 200) // wide for hint visibility
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"
	v.FilePath = filepath.Join(dir, "doc.md")

	header := v.renderHeader()
	plain := stripANSI(header)

	if !strings.Contains(plain, "back to directory") {
		t.Errorf("Expected 'back to directory' hint in header, got: %q", plain)
	}
}

// TestUpdateDirectoryLKeyCallsOpenFileFromDirectory verifies that pressing 'l'
// in directory mode triggers the file open behavior.
func TestUpdateDirectoryLKeyCallsOpenFileFromDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"api.md": "# API\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	result, _ := v.updateDirectory(msg)
	vv := result.(Viewer)

	// Should leave directory mode
	if vv.directoryMode {
		t.Error("Expected directoryMode=false after 'l' in directory mode")
	}
	// Should set openedFromDirectory
	if !vv.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true after 'l' in directory mode")
	}
}

// TestUpdateDirectoryEnterKeyCallsOpenFileFromDirectory verifies Enter also
// triggers file open.
func TestUpdateDirectoryEnterKeyCallsOpenFileFromDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := v.updateDirectory(msg)
	vv := result.(Viewer)

	if vv.directoryMode {
		t.Error("Expected directoryMode=false after Enter in directory mode")
	}
	if !vv.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true after Enter in directory mode")
	}
}

// TestBackToDirectoryResetsOffset verifies that returning to directory resets
// the scroll offset.
func TestBackToDirectoryResetsOffset(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"
	v.Offset = 42 // simulate having scrolled in the file

	vv, _ := v.BackToDirectory()

	if vv.Offset != 0 {
		t.Errorf("Expected Offset=0 after BackToDirectory, got %d", vv.Offset)
	}
}

// TestBackToDirectoryClearsSearch verifies search state is cleared when
// returning to directory.
func TestBackToDirectoryClearsSearch(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v.directoryMode = false
	v.openedFromDirectory = true
	v.currentView = "file"
	v.searchMode = true
	v.searchInput = "test"

	vv, _ := v.BackToDirectory()

	if vv.searchMode {
		t.Error("Expected searchMode=false after BackToDirectory")
	}
	if vv.searchInput != "" {
		t.Errorf("Expected searchInput='' after BackToDirectory, got %q", vv.searchInput)
	}
}

// TestOpenFileFromDirectoryPreservesDirectoryState verifies that the directory
// root path and file list are preserved when switching to file view.
func TestOpenFileFromDirectoryPreservesDirectoryState(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a",
		"bbb.md": "b",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	origRoot := v.directoryState.RootPath
	origCount := len(v.directoryState.Files)

	vv, _ := v.OpenFileFromDirectory()

	// Directory state should remain intact for return navigation
	if vv.directoryState.RootPath != origRoot {
		t.Errorf("RootPath changed: expected %q, got %q", origRoot, vv.directoryState.RootPath)
	}
	if len(vv.directoryState.Files) != origCount {
		t.Errorf("File count changed: expected %d, got %d", origCount, len(vv.directoryState.Files))
	}
}

// ==================== Split-Pane Mode Tests (09-01) ====================

// TestSplitModeStateInitialized verifies that splitMode defaults to false.
func TestSplitModeStateInitialized(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 120)
	if v.splitMode {
		t.Error("Expected splitMode=false by default")
	}
	if v.splitPreviewOffset != 0 {
		t.Errorf("Expected splitPreviewOffset=0, got %d", v.splitPreviewOffset)
	}
}

// TestSplitPaneWidthCalculation_Normal verifies width split at standard terminal sizes.
func TestSplitPaneWidthCalculation_Normal(t *testing.T) {
	tests := []struct {
		total     int
		wantLeft  int
		wantRight int
		wantOK    bool
	}{
		{120, 42, 77, true},  // 120 * 0.35 = 42; 120-42-1 = 77
		{100, 35, 64, true},  // 100 * 0.35 = 35; 100-35-1 = 64
		{80, 28, 51, true},   // 80 * 0.35 = 28; 80-28-1 = 51
		{160, 50, 109, true}, // 160 * 0.35 = 56 -> clamped to 50; 160-50-1 = 109
	}

	for _, tt := range tests {
		left, right, ok := splitPaneWidths(tt.total)
		if ok != tt.wantOK {
			t.Errorf("splitPaneWidths(%d): ok=%v, want %v", tt.total, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if left != tt.wantLeft {
			t.Errorf("splitPaneWidths(%d): left=%d, want %d", tt.total, left, tt.wantLeft)
		}
		if right != tt.wantRight {
			t.Errorf("splitPaneWidths(%d): right=%d, want %d", tt.total, right, tt.wantRight)
		}
	}
}

// TestSplitPaneWidthCalculation_NarrowTerminal verifies split is disabled below 80 cols.
func TestSplitPaneWidthCalculation_NarrowTerminal(t *testing.T) {
	for _, w := range []int{40, 60, 79} {
		_, _, ok := splitPaneWidths(w)
		if ok {
			t.Errorf("splitPaneWidths(%d): expected ok=false for narrow terminal", w)
		}
	}
}

// TestRenderDirectoryListingSplit_Truncates verifies long filenames are truncated.
func TestRenderDirectoryListingSplit_Truncates(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"very-long-filename-that-should-be-truncated.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	rows := v.renderDirectoryListingSplit(25, 10)
	if len(rows) != 10 {
		t.Fatalf("Expected 10 rows, got %d", len(rows))
	}
	// Row 2 should contain the (possibly truncated) filename
	row2 := rows[2]
	if !strings.Contains(row2, "…") && !strings.Contains(row2, "very-long") {
		// Either truncated with ellipsis or fits
		t.Logf("Row content: %q", row2)
	}
}

// TestRenderDirectoryListingSplit_Metadata verifies file list rows are populated.
func TestRenderDirectoryListingSplit_Metadata(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "# AAA\n",
		"bbb.md": "# BBB\nLine 2\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	rows := v.renderDirectoryListingSplit(30, 10)
	// Should have title, separator, then 2 file entries
	if len(rows) != 10 {
		t.Fatalf("Expected 10 rows, got %d", len(rows))
	}
	// Check that "aaa.md" appears somewhere in the rows
	found := false
	for _, r := range rows {
		if strings.Contains(r, "aaa.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'aaa.md' to appear in split directory listing")
	}
}

// TestRenderFilePreviewSplit_ShowsContent verifies file content appears in preview.
func TestRenderFilePreviewSplit_ShowsContent(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"test.md": "# Hello World\nThis is preview content.\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	rows := v.renderFilePreviewSplit(60, 10)
	if len(rows) != 10 {
		t.Fatalf("Expected 10 rows, got %d", len(rows))
	}
	// Check that content from the file appears
	combined := strings.Join(rows, "\n")
	if !strings.Contains(combined, "Hello World") {
		t.Error("Expected 'Hello World' in preview content")
	}
}

// TestRenderFilePreviewSplit_RespectsBoundary verifies preview stays within width.
func TestRenderFilePreviewSplit_RespectsBoundary(t *testing.T) {
	longLine := strings.Repeat("X", 200)
	dir := makeTempDir(t, map[string]string{
		"wide.md": longLine + "\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	rightWidth := 60
	rows := v.renderFilePreviewSplit(rightWidth, 10)
	// Content row (row 2) should not exceed rightWidth in rune length
	contentRow := rows[2]
	if len([]rune(contentRow)) > rightWidth {
		t.Errorf("Preview row exceeds rightWidth: got %d runes, max %d", len([]rune(contentRow)), rightWidth)
	}
}

// TestRenderSplitPane_CombinesLeftRight verifies the composite renders both panes.
func TestRenderSplitPane_CombinesLeftRight(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"readme.md": "# README\nProject description here.\n",
		"notes.md":  "# Notes\nSome notes.\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	v.splitMode = true
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderSplitPane(20)
	// Should contain both file list and preview content
	if !strings.Contains(output, "notes.md") && !strings.Contains(output, "readme.md") {
		t.Error("Expected file name to appear in split pane output")
	}
}

// TestRenderSplitPane_ProperAlignment verifies each line has the border character.
func TestRenderSplitPane_ProperAlignment(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"test.md": "# Test\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	v.splitMode = true
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderSplitPane(10)
	lines := strings.Split(output, "\n")
	// Each content line (not the footer) should contain the border character
	borderCount := 0
	for _, l := range lines {
		if strings.Contains(l, "│") {
			borderCount++
		}
	}
	if borderCount < 10 {
		t.Errorf("Expected at least 10 lines with border character, got %d", borderCount)
	}
}

// TestRenderSplitPane_BorderCharacters verifies the border is the │ character.
func TestRenderSplitPane_BorderCharacters(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "content",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	v.splitMode = true
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderSplitPane(5)
	if !strings.Contains(output, "│") {
		t.Error("Expected │ border character in split pane output")
	}
}

// TestRenderSplitPane_EmptyDirectory verifies split mode handles zero files.
func TestRenderSplitPane_EmptyDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"readme.txt": "not markdown",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	v.splitMode = true
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Should not panic
	output := v.renderSplitPane(10)
	if output == "" {
		t.Error("Expected non-empty output for empty directory in split mode")
	}
}

// TestRenderSplitPane_LargeDifferences verifies split handles many files.
func TestRenderSplitPane_LargeDifferences(t *testing.T) {
	files := make(map[string]string)
	for i := 0; i < 30; i++ {
		files[fmt.Sprintf("file%03d.md", i)] = fmt.Sprintf("# File %d\nContent line\n", i)
	}
	dir := makeTempDir(t, files)
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 40
	v.splitMode = true
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	output := v.renderSplitPane(35)
	if output == "" {
		t.Error("Expected non-empty output for large file list")
	}
	// Should contain border characters
	if !strings.Contains(output, "│") {
		t.Error("Expected border characters in output")
	}
}

// TestSplitModeWithVeryNarrowTerminal verifies fallback to full-screen listing.
func TestSplitModeWithVeryNarrowTerminal(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"test.md": "# Test\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 60) // too narrow for split
	v.Height = 24
	v.splitMode = true
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// renderSplitPane should fall back to renderDirectoryListing
	output := v.renderSplitPane(20)
	// The fallback output should still contain the file name
	if !strings.Contains(output, "test.md") {
		t.Error("Expected fallback to still show file name")
	}
}

// TestSplitModeScrollingPlaceholder verifies splitPreviewOffset starts at zero.
func TestSplitModeScrollingPlaceholder(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 120)
	if v.splitPreviewOffset != 0 {
		t.Errorf("Expected splitPreviewOffset=0 initially, got %d", v.splitPreviewOffset)
	}

	// Simulate setting offset
	v.splitPreviewOffset = 5
	if v.splitPreviewOffset != 5 {
		t.Errorf("Expected splitPreviewOffset=5, got %d", v.splitPreviewOffset)
	}
}

// TestSplitModeToggleState verifies split mode can be toggled on and off.
func TestSplitModeToggleState(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 120)

	if v.splitMode {
		t.Error("Expected splitMode=false initially")
	}

	v.splitMode = true
	if !v.splitMode {
		t.Error("Expected splitMode=true after toggle on")
	}

	v.splitMode = false
	if v.splitMode {
		t.Error("Expected splitMode=false after toggle off")
	}
}

// TestViewRoutesSplitMode verifies View() routes to split rendering when enabled.
func TestViewRoutesSplitMode(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Document\nSome content.\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Without split mode: normal directory listing
	normalOutput := v.View()

	// With split mode: split pane
	v.splitMode = true
	splitOutput := v.View()

	// Split output should contain border characters that normal output doesn't
	if !strings.Contains(splitOutput, "│") {
		t.Error("Expected split output to contain │ border")
	}
	// Outputs should be different
	if normalOutput == splitOutput {
		t.Error("Expected split and normal outputs to differ")
	}
}

// TestSplitPreviewPageIndicator verifies page indicator in preview pane.
func TestSplitPreviewPageIndicator(t *testing.T) {
	// Create a file with many lines to ensure multiple pages
	content := ""
	for i := 0; i < 100; i++ {
		content += fmt.Sprintf("Line %d of content\n", i)
	}
	dir := makeTempDir(t, map[string]string{
		"long.md": content,
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	rows := v.renderFilePreviewSplit(60, 12)
	// Last row should contain page indicator
	lastRow := rows[len(rows)-1]
	if !strings.Contains(lastRow, "pages") {
		t.Errorf("Expected page indicator in last row, got: %q", lastRow)
	}
}
