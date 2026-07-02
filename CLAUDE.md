# Project Instructions for AI Agents

This file provides instructions and context for AI coding agents working on this project.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->


## Build & Test

```bash
go build ./...
go vet ./...
go test ./...                    # full suite
go test ./internal/tui/...       # single package
gofmt -l <changed files>         # must be silent before committing
```

**Known pre-existing test failures** (not caused by your changes, do not try to fix unless that's the task):
- `internal/tui`: `TestUpdateGraph_DownNavigates`/`UpNavigates`/`DownWraps`/`UpWraps` — graph-view nav wrap bug
- `internal/nav`: `TestResolveLink_ExternalLink`/`ExternalLinkHTTPS`
- `internal/renderer`: `TestRenderDocument_Empty`
- `internal/knowledge`: `TestOpenClawDescriptor_Valid`/`QueryCommand`, `TestDockerfile_Valid`, `TestDeploymentDocs_Valid` — reference a Dockerfile/openclaw.yaml/DEPLOYMENT.md that don't exist in the repo

All four are tracked in `.planning/REQUIREMENTS.md` (REL-01..04) for milestone v1.1.

## Architecture Overview

Terminal markdown viewer/editor built on bubbletea + lipgloss. Three main packages:

- `internal/tui` — the interactive TUI. `Viewer` (`viewer.go`) is the top-level bubbletea model; per-mode Update/View logic lives in its own file (`edit_mode.go`, `mouse.go`, `directory.go`, `cross_search.go`, `search.go`, `replace.go`, `theme_dialog.go`, `graph.go`). Mode state (directory browser, cross-search) still lives on the `Viewer` struct rather than independent child models — a known architecture gap, see `.planning/PROJECT.md`.
- `internal/renderer` — custom ANSI-256 markdown renderer (goldmark AST → terminal output). Not glamour-based; `chroma` is an indirect dependency not currently used for syntax highlighting.
- `internal/knowledge` — BM25 full-text search index and link-based knowledge graph, persisted to SQLite (`modernc.org/sqlite`). CLI-reachable only via `bmd index`/`query`/`graph` (see `cmd/bmd/main.go`'s dispatcher) — do not assume every exported symbol here is live; a large amount of dead code (service-dependency-graph analysis, crawl, NER-based relationship discovery) was removed in v1.0 because it had already migrated to a separate `graphmd` project without the source being pruned.

`cmd/bmd/main.go` does manual `os.Args` parsing (no CLI framework) — `index`, `query`, `graph`, `--browse`, `help`, and bare `.md` file/directory args.

## Conventions & Patterns

- Per-mode TUI logic gets its own file with an `updateX(msg) (tea.Model, tea.Cmd)` method, following the existing pattern — don't add new inline blocks to `Viewer.Update()`.
- `.planning/` is gitignored in this repo — GSD planning docs (ROADMAP.md, PROJECT.md, REQUIREMENTS.md, phase PLAN/SUMMARY/VERIFICATION files) are local-only, never committed. **Do not run `phases.clear` or any other destructive `.planning/` cleanup without first confirming the target phase directory has actually been archived** — the `git status` safety guard some GSD commands rely on is a no-op here since git can't see gitignored files.
- New commits: separate unrelated changes into separate commits (bug fix vs. refactor vs. doc backfill) rather than bundling.
