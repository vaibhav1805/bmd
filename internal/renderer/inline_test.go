package renderer

import (
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/ast"
)

func TestRenderBold(t *testing.T) {
	result := RenderBold("hello")
	if !strings.HasPrefix(result, ansiBold) {
		t.Errorf("Expected bold prefix, got: %q", result)
	}
	if !strings.HasSuffix(result, ansiReset) {
		t.Errorf("Expected reset suffix, got: %q", result)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("Expected content in result, got: %q", result)
	}
}

func TestRenderItalic(t *testing.T) {
	result := RenderItalic("world")
	if !strings.HasPrefix(result, ansiItalic) {
		t.Errorf("Expected italic prefix, got: %q", result)
	}
	if !strings.HasSuffix(result, ansiReset) {
		t.Errorf("Expected reset suffix, got: %q", result)
	}
}

func TestRenderStrikethrough(t *testing.T) {
	result := RenderStrikethrough("test")
	if !strings.HasPrefix(result, ansiStrike) {
		t.Errorf("Expected strike prefix, got: %q", result)
	}
	if !strings.HasSuffix(result, ansiReset) {
		t.Errorf("Expected reset suffix, got: %q", result)
	}
}

func TestRenderInlineCode(t *testing.T) {
	result := RenderInlineCode("fmt.Println")
	if !strings.Contains(result, ansiCodeBg) {
		t.Errorf("Expected code background in result, got: %q", result)
	}
	if !strings.Contains(result, ansiCodeFg) {
		t.Errorf("Expected code foreground in result, got: %q", result)
	}
	if !strings.Contains(result, "fmt.Println") {
		t.Errorf("Expected content in result, got: %q", result)
	}
	if !strings.HasSuffix(result, ansiReset) {
		t.Errorf("Expected reset suffix, got: %q", result)
	}
}

func TestRenderText_Plain(t *testing.T) {
	n := ast.NewText("plain text")
	result := RenderText(n)
	if result != "plain text" {
		t.Errorf("Expected plain text unchanged, got: %q", result)
	}
}

func TestRenderText_Bold(t *testing.T) {
	n := ast.NewText("bold")
	n.Bold = true
	result := RenderText(n)
	if !strings.Contains(result, ansiBold) {
		t.Errorf("Expected bold code, got: %q", result)
	}
	if !strings.HasSuffix(result, ansiReset) {
		t.Errorf("Expected reset suffix, got: %q", result)
	}
}

func TestRenderText_BoldItalic(t *testing.T) {
	n := ast.NewText("bold-italic")
	n.Bold = true
	n.Italic = true
	result := RenderText(n)
	if !strings.Contains(result, ansiBold) {
		t.Errorf("Expected bold code in composed result, got: %q", result)
	}
	if !strings.Contains(result, ansiItalic) {
		t.Errorf("Expected italic code in composed result, got: %q", result)
	}
	if !strings.HasSuffix(result, ansiReset) {
		t.Errorf("Expected single reset at end, got: %q", result)
	}
}

func TestRenderText_Empty(t *testing.T) {
	n := ast.NewText("")
	n.Bold = true
	result := RenderText(n)
	if result != "" {
		t.Errorf("Expected empty result for empty text, got: %q", result)
	}
}
