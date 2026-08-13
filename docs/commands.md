# Commands

## bmd
Browse current directory in split-pane mode with file list and preview.

```bash
bmd
```

## bmd [FILE]
View a markdown file.

```bash
bmd README.md
```

## bmd index [DIR]
Build search index for a directory.

```bash
bmd index ./docs
bmd index ./docs --ignore-dirs vendor,build
bmd index ./docs --watch          # rebuild automatically on file changes
```

**Options:**
- `--dir DIR` — Directory to index (default: .)
- `--db PATH` — Database path (default: .bmd/knowledge.db)
- `--watch` — Rebuild the index automatically on file changes
- `--poll-interval N` — Polling interval in seconds, watch mode (default: 5)
- `--ignore-dirs DIRS` — Skip directories (comma-separated)
- `--ignore-files PATTERNS` — Skip file patterns
- `-A` — Include hidden directories
- `--no-ignore-defaults` — Disable default ignores

## bmd query TERM [DIR]
Search across all markdown files.

```bash
bmd query "authentication" --dir ./docs
```

**Options:**
- `--dir DIR` — Directory to search (default: .)
- `--format json|text|csv` — Output format (default: json)
- `--top N` — Max results (default: 10)

## bmd graph [DIR]
Export the link graph.

```bash
bmd graph --dir ./docs --format json
```

**Options:**
- `--dir DIR` — Directory to graph (default: .)
- `--format dot|json` — Output format (default: dot)
- `--service NAME` — Export only the subgraph reachable from this node (matched by ID or filename)
