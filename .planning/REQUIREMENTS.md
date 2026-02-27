# Requirements: Terminal Markdown Viewer

**Defined:** 2026-02-26
**Core Value:** Render markdown files beautifully in the terminal so developers can read documentation without leaving their terminal.

## v1 Requirements

### Rendering

- [x] **REND-01**: Render markdown with proper text styling (bold, italic, strikethrough)
- [x] **REND-02**: Display headings with visual hierarchy (size/color differentiation)
- [x] **REND-03**: Render code blocks with syntax highlighting (common languages)
- [x] **REND-04**: Format inline code with distinct styling
- [x] **REND-05**: Render lists (ordered, unordered, nested)
- [x] **REND-06**: Render blockquotes with visual distinction
- [x] **REND-07**: Render tables with aligned columns

### Navigation & Search

- [ ] **NAV-01**: Follow links to navigate between markdown files
- [ ] **NAV-02**: Search within rendered content
- [ ] **NAV-03**: Return to previous file (back button/history) — History logic complete (02-02)

### Core UX

- [ ] **UX-01**: Display file path and metadata at top
- [ ] **UX-02**: Show keyboard shortcuts for navigation/search
- [ ] **UX-03**: Handle long content with scrolling/pagination

## v2 Requirements

### Advanced Features

- **ADV-01**: Image rendering (ASCII art representation)
- **ADV-02**: Auto-reload when file changes
- **ADV-03**: Syntax highlighting for more languages
- **ADV-04**: Theme customization (colors, fonts)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Markdown editing | View-only tool for v1 |
| Complex markdown extensions | Focus on CommonMark spec |
| Web version | Terminal-only experience |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| REND-01 | Phase 1 | Complete (01-01) |
| REND-02 | Phase 1 | Complete (01-02) |
| REND-03 | Phase 1 | Complete (01-02) |
| REND-04 | Phase 1 | Complete (01-01) |
| REND-05 | Phase 1 | Complete (01-02) |
| REND-06 | Phase 1 | Complete (01-02) |
| REND-07 | Phase 1 | Complete (01-03 gap closure — alignment extraction) |
| NAV-01 | Phase 2 | In Progress — path resolver logic done (02-02), TUI wiring pending (02-03) |
| NAV-02 | Phase 2 | Pending |
| NAV-03 | Phase 2 | In Progress — History stack logic done (02-02), TUI wiring pending (02-03) |
| UX-01 | Phase 3 | Pending |
| UX-02 | Phase 3 | Pending |
| UX-03 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 13 total
- Mapped to phases: 13
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-26*
