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
```

**Options:**
- `--dir DIR` — Directory to index (default: .)
- `--db PATH` — Database path (default: .bmd/knowledge.db)
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
- `--strategy bm25|pageindex` — Search type (default: bm25)
- `--format json|text|csv` — Output format (default: json)
- `--top N` — Max results (default: 10)

## bmd graph [DIR]
Export link graph as JSON.

```bash
bmd graph --dir ./docs --format json
```

**Options:**
- `--dir DIR` — Directory to graph (default: .)
- `--format dot|json` — Output format (default: json)
