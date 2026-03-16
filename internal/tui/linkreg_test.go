package tui

import (
	"testing"
)

// helper: build a sentinel-wrapped link string as the renderer would produce it.
func makeLinkSentinel(url, visibleText string) string {
	return linkSentinelPrefix + url + linkSentinelSep + visibleText + linkSentinelEnd
}

func TestBuildRegistry_Empty(t *testing.T) {
	reg := BuildRegistry([]string{"no links here", "plain text"})
	if len(reg.Links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(reg.Links))
	}
	if reg.Focused() != -1 {
		t.Errorf("Expected focused=-1, got %d", reg.Focused())
	}
}

func TestBuildRegistry_SingleLink(t *testing.T) {
	line := "See " + makeLinkSentinel("docs/guide.md", "Guide") + " for details"
	reg := BuildRegistry([]string{line})
	if len(reg.Links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(reg.Links))
	}
	if reg.Links[0].URL != "docs/guide.md" {
		t.Errorf("Expected URL=docs/guide.md, got %q", reg.Links[0].URL)
	}
	if reg.Links[0].LineIndex != 0 {
		t.Errorf("Expected LineIndex=0, got %d", reg.Links[0].LineIndex)
	}
}

func TestBuildRegistry_MultipleLines(t *testing.T) {
	lines := []string{
		"First line: " + makeLinkSentinel("a.md", "A"),
		"Second line has no link",
		"Third line: " + makeLinkSentinel("b.md", "B") + " and " + makeLinkSentinel("c.md", "C"),
	}
	reg := BuildRegistry(lines)
	if len(reg.Links) != 3 {
		t.Fatalf("Expected 3 links, got %d", len(reg.Links))
	}
	if reg.Links[0].LineIndex != 0 || reg.Links[0].URL != "a.md" {
		t.Errorf("Link 0: unexpected %+v", reg.Links[0])
	}
	if reg.Links[1].LineIndex != 2 || reg.Links[1].URL != "b.md" {
		t.Errorf("Link 1: unexpected %+v", reg.Links[1])
	}
	if reg.Links[2].LineIndex != 2 || reg.Links[2].URL != "c.md" {
		t.Errorf("Link 2: unexpected %+v", reg.Links[2])
	}
}

func TestFocusNext_Cycles(t *testing.T) {
	lines := []string{
		makeLinkSentinel("a.md", "A"),
		makeLinkSentinel("b.md", "B"),
	}
	reg := BuildRegistry(lines)

	reg.FocusNext()
	if reg.FocusedURL() != "a.md" {
		t.Errorf("Expected a.md after first FocusNext, got %q", reg.FocusedURL())
	}
	reg.FocusNext()
	if reg.FocusedURL() != "b.md" {
		t.Errorf("Expected b.md after second FocusNext, got %q", reg.FocusedURL())
	}
	// Should wrap around
	reg.FocusNext()
	if reg.FocusedURL() != "a.md" {
		t.Errorf("Expected a.md after wrap, got %q", reg.FocusedURL())
	}
}

func TestFocusPrev_Cycles(t *testing.T) {
	lines := []string{
		makeLinkSentinel("a.md", "A"),
		makeLinkSentinel("b.md", "B"),
	}
	reg := BuildRegistry(lines)

	// FocusPrev from no focus should go to last
	reg.FocusPrev()
	if reg.FocusedURL() != "b.md" {
		t.Errorf("Expected b.md on first FocusPrev (wrap), got %q", reg.FocusedURL())
	}
	reg.FocusPrev()
	if reg.FocusedURL() != "a.md" {
		t.Errorf("Expected a.md, got %q", reg.FocusedURL())
	}
}

func TestFocusedURL_NoFocus(t *testing.T) {
	reg := BuildRegistry([]string{makeLinkSentinel("x.md", "X")})
	if reg.FocusedURL() != "" {
		t.Errorf("Expected empty URL with no focus, got %q", reg.FocusedURL())
	}
}

func TestFocusedLine(t *testing.T) {
	lines := []string{
		"plain",
		"has " + makeLinkSentinel("link.md", "Link"),
	}
	reg := BuildRegistry(lines)
	reg.FocusNext()
	if reg.FocusedLine() != 1 {
		t.Errorf("Expected FocusedLine=1, got %d", reg.FocusedLine())
	}
}

func TestClear(t *testing.T) {
	reg := BuildRegistry([]string{makeLinkSentinel("a.md", "A")})
	reg.FocusNext()
	if reg.FocusedURL() == "" {
		t.Fatalf("Expected focused URL after FocusNext")
	}
	reg.Clear()
	if reg.FocusedURL() != "" {
		t.Errorf("Expected empty URL after Clear, got %q", reg.FocusedURL())
	}
	if reg.FocusedLine() != -1 {
		t.Errorf("Expected FocusedLine=-1 after Clear, got %d", reg.FocusedLine())
	}
}

func TestStripSentinels(t *testing.T) {
	line := "See " + makeLinkSentinel("docs.md", "the docs") + " for more"
	stripped := StripSentinels(line)
	expected := "See the docs for more"
	if stripped != expected {
		t.Errorf("Expected %q, got %q", expected, stripped)
	}
}

func TestStripSentinels_MultiplLinks(t *testing.T) {
	line := makeLinkSentinel("a.md", "A") + " and " + makeLinkSentinel("b.md", "B")
	stripped := StripSentinels(line)
	expected := "A and B"
	if stripped != expected {
		t.Errorf("Expected %q, got %q", expected, stripped)
	}
}

func TestFocusNext_NoLinks(t *testing.T) {
	reg := BuildRegistry([]string{"plain text"})
	idx := reg.FocusNext()
	if idx != -1 {
		t.Errorf("Expected -1 with no links, got %d", idx)
	}
}
