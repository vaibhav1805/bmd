---
phase: 12-mcp-infrastructure
plan: 02
subsystem: api
tags: [mcp, mcp-go, stdio, json-rpc, agent-integration, knowledge-graph, bm25, pageindex]

# Dependency graph
requires:
  - phase: 12-01
    provides: PageIndex UI search integration that confirmed knowledge command API stability
  - phase: 11-pageindex-integration
    provides: CmdQuery/CmdIndex/CmdContext/CmdDepends/CmdServices/CmdGraph commands wrapped by MCP handlers
  - phase: 10-agent-contracts
    provides: CONTRACT-01 JSON envelope (ContractResponse) that MCP tool responses comply with

provides:
  - MCP server (bmd serve --mcp) exposing all 6 knowledge tools as native endpoints
  - internal/mcp/server.go — Server struct with Start() method and tool registration
  - internal/mcp/handlers.go — per-tool handlers with captureOutput/captureStderr helpers
  - CLI integration: 'bmd serve --mcp' entry point in cmd/bmd/main.go
  - Documentation: MCP Server Mode section in README.md and ARCHITECTURE.md
  - 6 integration tests covering query, index, contract compliance, required params, concurrency

affects:
  - 12-03 (OpenClaw integration needs the serve --mcp endpoint)
  - agents/deployment (fleet deployment runs bmd serve --mcp)

# Tech tracking
tech-stack:
  added:
    - github.com/mark3labs/mcp-go v0.44.1 (MCP SDK for Go, stdio server)
    - github.com/invopop/jsonschema (transitive, mcp-go dependency)
    - github.com/spf13/cast (transitive, mcp-go dependency)
    - github.com/yosida95/uritemplate/v3 (transitive, mcp-go dependency)
  patterns:
    - captureOutput pattern: redirect os.Stdout to pipe, run fn, restore — enables CLI command reuse in MCP handlers
    - captureStderr pattern: same for progress messages from CmdIndex
    - Tool handler pattern: delegate to existing Cmd* functions rather than duplicating logic
    - Error handling pattern: missing required params return IsError result, not Go errors

key-files:
  created:
    - internal/mcp/server.go
    - internal/mcp/handlers.go
    - internal/mcp/server_test.go
  modified:
    - cmd/bmd/main.go (added 'serve' case + context import + bmcmp import)
    - README.md (MCP Server Mode section with tool table and integration example)
    - ARCHITECTURE.md (MCP Server component, pipeline diagram, project status update)
    - go.mod (added mark3labs/mcp-go v0.44.1)
    - go.sum (updated)

key-decisions:
  - "mark3labs/mcp-go chosen as MCP SDK — most widely-used Go MCP library, provides ServeStdio convenience function"
  - "captureOutput pattern delegates to existing Cmd* functions — reuses CONTRACT-01 compliance without duplicating logic"
  - "bmcmp import alias for internal/mcp to avoid collision with mcp-go sdk package name in main.go"
  - "Server.baseDir defaults to cwd in CLI wiring — zero-config for standard usage"
  - "MCP handlers return IsError result for missing params (not Go errors) — correct MCP protocol error semantics"

patterns-established:
  - "MCP handler pattern: parse params with mcp.ParseString/ParseInt → validate required → captureOutput(CmdX) → return text result"
  - "captureOutput/captureStderr: reusable helpers for CLI-to-MCP command bridging"

requirements-completed: [MCP-01, MCP-02]

# Metrics
duration: 18min
completed: 2026-03-01
---

# Phase 12 Plan 02: MCP Server Endpoint Exposure Summary

**Native MCP server via mark3labs/mcp-go SDK, exposing all 6 knowledge tools (query/index/depends/services/graph/context) on stdin/stdout with CONTRACT-01 compliance and zero subprocess overhead**

## Performance

- **Duration:** 18 min
- **Started:** 2026-03-01T20:30:00Z
- **Completed:** 2026-03-01T20:48:00Z
- **Tasks:** 4 of 4
- **Files modified:** 8 (3 new, 5 modified)

## Accomplishments

- Full MCP server exposing all 6 bmd knowledge tools as native endpoints
- `bmd serve --mcp` CLI entry point with proper error handling and usage message
- captureOutput/captureStderr helpers bridge existing CLI commands to MCP protocol without duplicating logic
- All responses comply with CONTRACT-01 JSON envelope (verified in TestMCPServer_ContractCompliance)
- 6 integration tests pass: query, index, contract compliance, required params, concurrency, NewServer

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement MCP Server Handler** - `37680a5` (feat)
2. **Task 2: Wire MCP Server into CLI** - `3b1f62d` (feat)
3. **Task 3: Document MCP Protocol Integration** - `6e5cc0f` (docs)
4. **Task 4: Add Integration Tests** - `b675cf1` (test)

## Files Created/Modified

- `internal/mcp/server.go` — Server struct, NewServer(), Start(), 6 registerXxxTool() methods
- `internal/mcp/handlers.go` — handleQuery/Index/Depends/Services/Graph/Context + captureOutput/captureStderr helpers
- `internal/mcp/server_test.go` — 6 tests (9 sub-tests) all passing
- `cmd/bmd/main.go` — 'serve --mcp' case, context+bmcmp imports, usage doc update
- `README.md` — MCP Server Mode section with tool table, JSON config example, CONTRACT-01 note
- `ARCHITECTURE.md` — MCP Server component with pipeline diagram, updated project status
- `go.mod` — mark3labs/mcp-go v0.44.1
- `go.sum` — updated with all transitive dependencies

## Decisions Made

- **mark3labs/mcp-go**: Most widely-used Go MCP library. Provides ServeStdio() convenience function for stdin/stdout protocol handling. Clean API with mcp.NewTool(), server.AddTool(), mcp.ParseString/Int/Boolean helpers.
- **captureOutput pattern**: Delegates to existing Cmd* functions rather than duplicating logic. Ensures CONTRACT-01 compliance is automatically inherited from the existing command implementations.
- **bmcmp import alias**: Avoids collision between `internal/mcp` package name and `mcp-go/mcp` package in main.go.
- **MCP handlers return IsError**: Missing required params return `mcp.NewToolResultError()` (not Go errors). This is the correct MCP protocol semantics — Go errors represent transport failures.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated go.sum after go.mod change**
- **Found during:** Task 1 (first build attempt)
- **Issue:** `go get github.com/mark3labs/mcp-go@latest` added the module but `go.sum` was missing transitive dependency entries (invopop/jsonschema, spf13/cast, yosida95/uritemplate)
- **Fix:** Ran `go get github.com/mark3labs/mcp-go/mcp@v0.44.1` to populate all go.sum entries
- **Files modified:** go.sum
- **Verification:** `go build ./internal/mcp/...` succeeded after fix
- **Committed in:** 37680a5 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed t.Errorf misuse in concurrency test**
- **Found during:** Task 4 (test build)
- **Issue:** `errors[idx] = t.Errorf(...)` — t.Errorf returns void, cannot be assigned to error
- **Fix:** Changed to `t.Errorf(...)` standalone call (already marks test as failed)
- **Files modified:** internal/mcp/server_test.go
- **Verification:** Test compiled and passed
- **Committed in:** b675cf1 (Task 4 commit)

**3. [Rule 1 - Bug] Removed impossible type assertion in extractText helper**
- **Found during:** Task 4 (test build)
- **Issue:** `c.(map[string]interface{})` — mcp.Content is an interface with unexported methods, map cannot implement it
- **Fix:** Removed the impossible type assertion branch; TextContent assertion is sufficient
- **Files modified:** internal/mcp/server_test.go
- **Verification:** Test compiled and passed
- **Committed in:** b675cf1 (Task 4 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking dependency, 2 compile errors from plan's test outline)
**Impact on plan:** All auto-fixes necessary for compilation. No scope creep.

## Issues Encountered

- Pre-existing test failures in `internal/knowledge` (TestSaveAndLoadTreeFile) and `internal/nav` (TestResolveLink_ExternalLink) and `internal/renderer` (TestRenderDocument_Empty) are out of scope — they predate Phase 12 and are documented in prior summaries.

## User Setup Required

None — `bmd serve --mcp` requires no external service configuration. The mark3labs/mcp-go SDK is bundled via go.sum.

## Next Phase Readiness

- Phase 12-03 (OpenClaw integration) can proceed — `bmd serve --mcp` endpoint is live
- MCP server handles all 6 tools agents need for fleet deployment
- CONTRACT-01 compliance verified via integration tests

---
*Phase: 12-mcp-infrastructure*
*Completed: 2026-03-01*
