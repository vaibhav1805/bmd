---
phase: 10-agent-contracts
plan: "01"
subsystem: internal/knowledge
tags: [agent-contracts, json-envelope, error-codes, reliability]
dependency_graph:
  requires: []
  provides: [ContractResponse, ErrCode constants, marshalContract]
  affects: [CmdQuery, CmdGraph, CmdServices, CmdDepends]
tech_stack:
  added: []
  patterns: [JSON envelope, factory constructors, error classification]
key_files:
  created: []
  modified:
    - internal/knowledge/output.go
    - internal/knowledge/commands.go
    - internal/knowledge/commands_test.go
decisions:
  - "ContractResponse wraps all JSON output from agent commands; text/CSV/DOT paths are byte-for-byte unchanged"
  - "classifyIndexError() maps error message content to INDEX_NOT_FOUND or INTERNAL_ERROR"
  - "CmdDepends with json format returns nil (envelope on stdout) for service-not-found, text format still returns error"
  - "Existing integration tests updated to verify ContractResponse envelope structure rather than raw inner fields"
metrics:
  duration: "4 minutes"
  completed: "2026-03-01T10:43:53Z"
  tasks_completed: 3
  files_modified: 3
---

# Phase 10 Plan 01: Agent Output Contracts Summary

JSON envelope (ContractResponse) added to all agent-facing bmd commands so agents can programmatically handle errors without parsing stderr or guessing from output shape.

## What Was Built

### ContractResponse Struct (internal/knowledge/output.go)

Four error code constants:
- `ErrCodeIndexNotFound = "INDEX_NOT_FOUND"`
- `ErrCodeFileNotFound  = "FILE_NOT_FOUND"`
- `ErrCodeInvalidQuery  = "INVALID_QUERY"`
- `ErrCodeInternalError = "INTERNAL_ERROR"`

`ContractResponse` struct with `status`, `code`, `message`, `data` fields.

Three constructor functions:
- `NewOKResponse(message, data)` — status="ok", code=nil
- `NewEmptyResponse(message, data)` — status="empty", code=nil
- `NewErrorResponse(code, message)` — status="error", code=&code, data=nil

`marshalContract(resp)` helper for indented JSON serialization.

### Command Wiring (internal/knowledge/commands.go)

`classifyIndexError(err)` helper maps error messages to INDEX_NOT_FOUND or INTERNAL_ERROR.

All four agent-facing commands updated for json format:

- **CmdQuery**: INVALID_QUERY (empty term), INDEX_NOT_FOUND/INTERNAL_ERROR (index errors), empty/ok envelopes for 0/N results
- **CmdGraph**: INDEX_NOT_FOUND/INTERNAL_ERROR (index errors), FILE_NOT_FOUND (service not in graph), ok envelope
- **CmdServices**: INDEX_NOT_FOUND/INTERNAL_ERROR (index errors), ok envelope
- **CmdDepends**: INDEX_NOT_FOUND/INTERNAL_ERROR (index errors), FILE_NOT_FOUND (service not found), ok envelope

Text/CSV/DOT paths are completely unchanged.

### Tests (internal/knowledge/commands_test.go)

`TestContractResponsePaths` with 8 sub-tests:
1. ok response has nil code
2. empty response has nil code
3. error response INDEX_NOT_FOUND
4. error response FILE_NOT_FOUND
5. error response INVALID_QUERY
6. error response INTERNAL_ERROR
7. marshalContract produces valid JSON
8. empty response serializes code as JSON null

Existing integration tests updated to check ContractResponse envelope structure.

## Files Modified

| File | Changes |
|------|---------|
| `internal/knowledge/output.go` | Added ErrCode constants, ContractResponse struct, 3 constructors, marshalContract (+45 lines) |
| `internal/knowledge/commands.go` | Added sort import, classifyIndexError(), wired envelope into 4 Cmd functions (+318 lines) |
| `internal/knowledge/commands_test.go` | Updated 4 integration tests, added TestContractResponsePaths (+90 lines) |

## Test Count

- TestContractResponsePaths: 8 new sub-tests (all PASS)
- 4 existing integration tests updated (all PASS)
- Full knowledge package: ok (0 failures)
- Pre-existing failures in nav/renderer are unchanged and out of scope

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated existing integration tests to match new ContractResponse contract**

- **Found during:** Task 2 verification
- **Issue:** TestCmdQuery_JSON, TestCmdServices_JSON, TestCmdGraph_JSON expected raw inner fields (query, results, services, nodes, edges) directly in output. TestCmdDepends_MissingService expected `err != nil` but json format now emits envelope and returns nil.
- **Fix:** Updated 4 tests to verify ContractResponse envelope structure with status/data fields, then verify inner payload fields inside data. TestCmdDepends_MissingService now checks JSON envelope for status=error/code=FILE_NOT_FOUND, plus verifies text format still returns an error.
- **Files modified:** internal/knowledge/commands_test.go
- **Commit:** 51ea23a

## Self-Check: PASSED

- internal/knowledge/output.go: FOUND (ContractResponse appears 15 times)
- internal/knowledge/commands.go: FOUND (NewOKResponse appears 4 times)
- internal/knowledge/commands_test.go: FOUND (INDEX_NOT_FOUND appears 2 times)
- Commits 6c4b41e, 51ea23a, a7dd69f: ALL FOUND
