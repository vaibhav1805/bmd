# Getting Started with BMD

Beautiful Markdown (BMD) is a terminal-based markdown editor with integrated search.

## Installation

```bash
# Download and install
curl -fsSL https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh | bash

# Or build from source
git clone https://github.com/vaibhav1805/bmd && cd bmd
go build -o bmd ./cmd/bmd && sudo mv bmd /usr/local/bin/
```

## First Steps

### View a File
```bash
bmd README.md
```
**Keyboard shortcuts:**
- `j`/`k` — scroll down/up
- `gg` — jump to top
- `G` — jump to bottom
- `e` — enter edit mode
- `/` — search within file
- `t` — change theme
- `?` — show help
- `q` — quit

### Browse a Directory
```bash
bmd
```
Opens the current directory in split-pane mode:
- Arrow keys — navigate files
- `Enter` — open selected file
- `s` — toggle split pane
- `Ctrl+F` — search across all files
- `h`/`Backspace` — back to directory view

### Edit a File
```bash
bmd file.md
# Press 'e' to enter edit mode
# Type to edit, Ctrl+S to save
# Esc to return to view mode
```

**Edit shortcuts:**
- `Ctrl+S` — save
- `Ctrl+Z` — undo
- `Ctrl+Y` — redo
- `Ctrl+Home`/`Ctrl+End` — jump to start/end
- `Esc` — exit edit mode

### Search Across Files
Press `Ctrl+F` in directory mode to search all markdown files. Results appear with file paths and matching content. The search index is built automatically on first use.

```bash
# Or search from command line
bmd query "authentication" --dir ./docs
```

## Next Steps

- **[Commands](./commands.md)** — Full command reference
- **[ARCHITECTURE.md](../ARCHITECTURE.md)** — Technical details
