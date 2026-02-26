# Terminal Markdown Viewer

## What This Is

A terminal-based markdown renderer that displays markdown files with beautiful formatting, colors, and syntax highlighting — like viewing them on the web, but in the terminal. Keeps developers in their CLI workflow without context-switching to a browser.

## Core Value

Render markdown files beautifully in the terminal so developers can read documentation without leaving their terminal.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Render markdown with styling (bold, italic, colors)
- [ ] Display headings with hierarchy
- [ ] Render code blocks with syntax highlighting
- [ ] Format lists, blockquotes, and tables
- [ ] Follow links to navigate between files
- [ ] Search within rendered content

### Out of Scope

- Real-time editing — view-only for v1
- Complex markdown extensions — focus on CommonMark
- Terminal UI framework beyond basic rendering

## Context

This is a developer tool for reading documentation in the terminal. Target audience is developers who prefer CLI workflows. The tool replaces the need to open a browser for quick doc lookups.

## Constraints

- Terminal-only — no GUI
- Should work across common terminals (bash, zsh)
- Performance: fast rendering of typical doc files

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Terminal rendering focus | Keep developers in CLI workflow | — Pending |

---
*Last updated: 2026-02-26 after initialization*
