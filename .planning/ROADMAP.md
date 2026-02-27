# Roadmap: Terminal Markdown Viewer

## Overview

Three phases deliver a beautiful, navigable markdown viewer for the terminal. Phase 1 establishes the core rendering engine that makes markdown readable. Phase 2 adds navigation and search so users can move between files and find content. Phase 3 completes the UX layer so the tool is polished and self-explanatory.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Core Rendering** - Render all markdown elements beautifully in the terminal
- [x] **Phase 2: Navigation & Search** - Move between files and find content within them
- [ ] **Phase 3: Polish & UX** - Headers, keyboard hints, scrolling — tool feels complete

## Phase Details

### Phase 1: Core Rendering
**Goal**: Users can open a markdown file and read it with beautiful, legible formatting in the terminal
**Depends on**: Nothing (first phase)
**Requirements**: REND-01, REND-02, REND-03, REND-04, REND-05, REND-06, REND-07
**Success Criteria** (what must be TRUE):
  1. User runs `bmd README.md` and sees styled output — headings visually distinct, bold/italic text rendered, inline code highlighted
  2. Code blocks display with syntax highlighting for common languages (JS, Python, Go, etc.)
  3. Lists, blockquotes, and tables render with correct structure and visual alignment
**Plans**:
- [x] Plan 01: Foundation & Architecture — Go project, goldmark parser, ANSI renderer, CLI pipeline
- [x] Plan 02: Block-Level Rendering — headings H1-H6, code blocks with chroma highlighting, lists, blockquotes, tables
- [x] Plan 03: Gap Closure — table column alignment extraction from goldmark AST (REND-07 fully verified)

### Phase 2: Navigation & Search
**Goal**: Users can follow links between markdown files and search rendered content without leaving the terminal
**Depends on**: Phase 1
**Requirements**: NAV-01, NAV-02, NAV-03
**Success Criteria** (what must be TRUE):
  1. User clicks/selects a link to another `.md` file and that file renders in place
  2. User can search for a term and matching text is highlighted in the rendered output
  3. User can navigate back to the previously viewed file
**Plans:** 6/6 plans executed

Plans:
- [x] 02-01-PLAN.md — TUI foundation: bubbletea viewer, scrollable display, keyboard/quit
- [x] 02-02-PLAN.md — TDD: navigation history stack + path security resolver (pure logic)
- [x] 02-03-PLAN.md — Link navigation: link registry, Tab/mouse/follow, history wiring, file browser
- [x] 02-04-PLAN.md — TDD: search matcher (FindMatches, StripANSI, case-insensitive)
- [x] 02-05-PLAN.md — Search UI: Ctrl+F prompt, all-match highlighting, n/N jump, match counter
- [x] 02-06-PLAN.md — Human verification checkpoint: all NAV-01, NAV-02, NAV-03 checks

### Phase 3: Polish & UX
**Goal**: Users understand what the tool can do immediately and can comfortably read any-length document
**Depends on**: Phase 2
**Requirements**: UX-01, UX-02, UX-03
**Success Criteria** (what must be TRUE):
  1. File path and basic metadata appear at the top of every rendered view
  2. Keyboard shortcuts for navigation and search are visible on screen
  3. Long documents scroll or paginate — no content is cut off or lost
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core Rendering | 3/3 | Complete    | 2026-02-26 |
| 2. Navigation & Search | 6/6 | Complete   | 2026-02-27 |
| 3. Polish & UX | 0/? | Not started | - |
