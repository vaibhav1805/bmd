package renderer

import (
	"testing"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

func TestCheckboxRendering_Unchecked(t *testing.T) {
	th := theme.NewThemeForScheme(theme.Dark)
	r := NewRenderer(th, 80)

	li := ast.NewListItem()
	checked := false
	li.Checkbox = &checked
	li.AddChild(ast.NewText("Task to do"))

	result := r.renderListItem(li, false, 0)

	if !contains(result, "☐") {
		t.Errorf("Expected unchecked checkbox ☐ in output, got: %s", result)
	}
}

func TestCheckboxRendering_Checked(t *testing.T) {
	th := theme.NewThemeForScheme(theme.Dark)
	r := NewRenderer(th, 80)

	li := ast.NewListItem()
	checked := true
	li.Checkbox = &checked
	li.AddChild(ast.NewText("Task done"))

	result := r.renderListItem(li, false, 0)

	if !contains(result, "☑") {
		t.Errorf("Expected checked checkbox ☑ in output, got: %s", result)
	}
}

func TestCheckboxRendering_NoCheckbox(t *testing.T) {
	th := theme.NewThemeForScheme(theme.Dark)
	r := NewRenderer(th, 80)

	li := ast.NewListItem()
	// No checkbox field set (nil)
	li.AddChild(ast.NewText("Regular item"))

	result := r.renderListItem(li, false, 0)

	// Should not contain checkbox characters
	if contains(result, "☐") || contains(result, "☑") {
		t.Errorf("Expected no checkboxes in output, got: %s", result)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
