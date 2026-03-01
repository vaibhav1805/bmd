---
phase: 11-pageindex-integration
plan: "02"
subsystem: knowledge
tags: [rag, context-assembly, bm25, pageindex, cli, contract]

# Dependency graph
requires:
  - phase: 11-pageindex-integration
    provides: TreeNode/FileTree types, LoadTreeFiles, RunPageIndex, ErrPageIndexNotFound, .bmd-tree.json infrastructure
  - phase: 10-agent-contracts
    provides: ContractResponse envelope, marshalContract, NewOKResponse/NewEmptyResponse, openOrBuildIndex

provides:
  - ContextSection struct (File, HeadingPath, Content, Score, ReasoningTrace)
  - AssembleContextBlock formats '# Context for:' + '## [file § heading]' citations
  - sectionsFromBM25Results converts BM25 SearchResult slice to []ContextSection
  - ContextArgs and ParseContextArgs CLI argument parser
  - CmdContext command with BM25 fallback and PageIndex tree path
  - RunPageIndexQuery subprocess wrapper for pageindex query invocations
  - 'context' case wired in cmd/bmd/main.go router

affects: [11-03, "bmd context command", "RAG pipeline consumers", "MCP server context tool (Phase 12)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "BM25 fallback pattern: try PageIndex trees first, fall back to BM25 when unavailable"
    - "Context assembly: '# Context for:' header + per-section '## [file § heading]' citations"
    - "Preamble section format: '## [file]' without § separator when HeadingPath is empty"
    - "ContractResponse envelope for JSON output: same CONTRACT-01 format as all other agent commands"
    - "Stdout capture pattern in tests: os.Pipe + io.Copy for integration test output verification"

key-files:
  created:
    - internal/knowledge/context.go
    - internal/knowledge/context_test.go
  modified:
    - cmd/bmd/main.go

key-decisions:
  - "BM25 fallback executes when no .bmd-tree.json files are present OR when RunPageIndexQuery fails — graceful degradation always wins"
  - "AssembleContextBlock uses § (U+00A7) as the heading separator for clean readability: '## [file.md § Section Name]'"
  - "Preamble sections (HeadingPath='') omit the § separator: '## [file.md]' not '## [file.md § ]'"
  - "RunPageIndexQuery passes trees as JSON via stdin to pageindex subprocess — avoids temp file creation"
  - "printContextJSON uses the same CONTRACT-01 ContractResponse envelope as all other agent commands"
  - "ContextSection.ReasoningTrace field retained from linter addition — enables future pageindex reasoning trace passthrough"

patterns-established:
  - "CmdContext BM25 fallback: LoadTreeFiles → RunPageIndexQuery → on failure → openOrBuildIndex → ScanDirectory → idx.Search"
  - "Context block assembly: '# Context for: {query}' header + blank line before each '## [...]' citation + content verbatim"

requirements-completed: [CONTEXT-01]

# Metrics
duration: 10min
completed: 2026-03-01
---

# Phase 11 Plan 02: bmd context Command Summary

**RAG-ready context block assembly via `bmd context QUERY` with PageIndex tree path and BM25 fallback, CONTRACT-01 JSON envelope support**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-01T11:25:31Z
- **Completed:** 2026-03-01T11:35:00Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- `ContextSection` struct and `AssembleContextBlock` formatter producing markdown-formatted context blocks ready for LLM injection
- `CmdContext` command with dual strategy: PageIndex tree path (when .bmd-tree.json present) with transparent BM25 fallback
- `RunPageIndexQuery` subprocess wrapper for future pageindex query invocations with graceful error handling
- `bmd context` wired into `cmd/bmd/main.go` router and usage documentation
- 12 new tests all passing: 4 assembly, 1 BM25 conversion, 4 ParseContextArgs, 3 CmdContext integration

## Task Commits

Each task was committed atomically:

1. **Task 1: ContextSection type, AssembleContextBlock, sectionsFromBM25Results, ParseContextArgs** - `4627f17` (feat)
2. **Task 2: Wire CmdContext into main.go router and usage** - `abb42f4` (feat)
3. **Task 3: Integration tests (already in Task 1 commit)**

## Files Created/Modified

- `internal/knowledge/context.go` - ContextSection struct; AssembleContextBlock markdown formatter; sectionsFromBM25Results BM25 adapter; ContextArgs/ParseContextArgs CLI parser; CmdContext command with PageIndex + BM25 fallback; RunPageIndexQuery subprocess wrapper; buildPageIndexQueryCmd helper
- `internal/knowledge/context_test.go` - 12 tests: 4 AssembleContextBlock, 1 sectionsFromBM25Results, 4 ParseContextArgs, 3 CmdContext integration
- `cmd/bmd/main.go` - Added `case "context"` routing to `knowledge.CmdContext`; updated usage() to document context command

## Decisions Made

- BM25 fallback executes when no .bmd-tree.json files are present OR when RunPageIndexQuery fails — graceful degradation always succeeds
- AssembleContextBlock uses § (U+00A7) as the heading separator for clean readability in LLM prompts
- Preamble sections (HeadingPath='') omit the § separator to avoid "file.md §" trailing noise
- RunPageIndexQuery passes trees as JSON via stdin to avoid temp file creation
- printContextJSON uses the same CONTRACT-01 ContractResponse envelope pattern for agent consistency
- ContextSection.ReasoningTrace field retained from linter suggestion — forward-compatible with pageindex reasoning traces

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] cmdQueryPageIndex function was referenced but not implemented**
- **Found during:** Task 2 (build verification)
- **Issue:** commands.go referenced `cmdQueryPageIndex` and used types `pageindexResponseJSON`, `reasoningResultJSON`, `ErrCodePageIndexNotAvailable` that didn't exist — build failed
- **Fix:** Investigation revealed these types and the partial function body were already in output.go and commands.go (modified by a prior session). The build was actually clean once all files were compiled together — the initial error was a stale build cache artifact. Build passes cleanly.
- **Files modified:** None (no fix needed, build was clean on retry)
- **Verification:** `go build ./...` passes with no errors

---

**Total deviations:** 0 (1 apparent issue resolved as false alarm — build was clean)
**Impact on plan:** No scope creep. Plan executed as specified.

## Issues Encountered

The initial build check reported `undefined: cmdQueryPageIndex` but this resolved on retry — the function and types it uses were present in commands.go and output.go from a previous session's modifications to those files. No actual fix was needed.

## User Setup Required

None - no external service configuration required at this stage. The `pageindex` CLI binary is not required for the BM25 fallback path which is the primary test path.

## Next Phase Readiness

- `bmd context` command is complete and production-ready for BM25 fallback
- `RunPageIndexQuery` is implemented and ready to activate when `pageindex` CLI is installed
- `LoadTreeFiles` from Plan 11-01 is the natural entry point for plan 11-03's live indexing
- Plan 11-03 can use `CmdContext` as the primary retrieval API for the MCP server context tool

## Self-Check: PASSED

- internal/knowledge/context.go: FOUND
- internal/knowledge/context_test.go: FOUND
- cmd/bmd/main.go: FOUND (context case added)
- Commit 4627f17: FOUND (Task 1 — ContextSection, AssembleContextBlock, tests)
- Commit abb42f4: FOUND (Task 2 — main.go wiring)
- Commit ef491c8: FOUND (docs — SUMMARY, STATE, ROADMAP)
- All 12 context tests: PASS
- Full knowledge package test suite: PASS
- go build ./...: PASS

---
*Phase: 11-pageindex-integration*
*Completed: 2026-03-01*
