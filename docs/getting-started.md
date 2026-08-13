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

If `vim` or `nvim` is already on your system, the installer enables bmd's vim keybindings by default (see below) — press `v` inside bmd anytime to turn them off.

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
- `b` — file browser (split pane)
- `B` — directory browser (full)
- `v` — toggle vim keybindings (edit mode)
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

Press `B` from any open file to jump straight into the full directory browser, even if the file wasn't opened from directory mode.

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

**Vim keybindings (optional):** press `v` in the file view to toggle Normal/Insert/Visual modal editing on or off — the setting is remembered across sessions. Once enabled, `e` still enters edit mode, but starts in Normal mode instead of typing directly:
- Motions: `h j k l`, `w`/`b`/`e` (word), `0`/`$` (line start/end), `gg`/`G` (top/bottom), with count prefixes (`3j`, `2dd`)
- Enter insert mode: `i a I A o O`; `Esc` returns to Normal mode without leaving edit mode
- Operators: `d`/`y`/`c` combined with a motion, or doubled for the whole line (`dd`/`yy`/`cc`)
- `x`/`X` delete a character, `p`/`P` paste, `u` undoes
- Visual selection: `v` (character) / `V` (line), then `d`/`y`/`c`
- All the shortcuts above (Ctrl+S save, Ctrl+Z/Y undo/redo, Ctrl+F search, etc.) keep working regardless of vim mode
- `Esc` from Normal mode exits edit mode, same as when vim keybindings are off

### Search Across Files
Press `Ctrl+F` in directory mode to search all markdown files. Results appear with file paths and matching content. The search index is built automatically on first use.

```bash
# Or search from command line
bmd query "authentication" --dir ./docs
```

## Next Steps

- **[Commands](./commands.md)** — Full command reference
- **[ARCHITECTURE.md](../ARCHITECTURE.md)** — Technical details
