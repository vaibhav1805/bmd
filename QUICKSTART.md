# BMD Quick Start Guide

Get up and running with BMD in 5 minutes.

## Installation

```bash
git clone https://github.com/flurryhead/bmd
cd bmd
go build -o bmd ./cmd/bmd
sudo mv bmd /usr/local/bin/
```

## View Your First File

```bash
bmd README.md
```

**Basic navigation:**
- `↓` or `j` — Scroll down
- `↑` or `k` — Scroll up
- `q` — Quit

## Common Tasks

### Read a Long Document

```bash
bmd my-doc.md
```

Press `:100` to jump to line 100, `:1` to jump to top.

### Search Within a Document

```bash
# Opens bmd
bmd README.md

# Then inside:
/search-term    # Find text
n               # Next match
N               # Previous match
```

### Navigate Between Files

```bash
bmd docs/README.md

# The README contains links to other markdown files
# Press Tab to cycle through links
# Press Enter to follow the current link
# Press Backspace to go back
```

### Switch Themes

Inside BMD, press `t` to cycle through themes:
- Default (terminal colors)
- Ocean (cool blue/cyan)
- Forest (green/brown)
- Sunset (warm orange/pink)
- Midnight (dark purple/blue)

### Change Font Size

Your terminal's font size controls BMD rendering. Change in your terminal settings:
- Alacritty: `alacritty.toml` — `font.size`
- macOS Terminal: Preferences → Font
- iTerm2: Preferences → Profiles → Text → Font

### Copy Text

Select text with mouse drag, then press `Ctrl+C` (or `Cmd+C` on macOS).

Alternatively: Use standard terminal copy (usually `Shift+Drag` or context menu).

## Knowledge System (Advanced)

### Index a Documentation Directory

```bash
# Index your project docs
bmd index ./docs

# This creates knowledge.db
```

### Search Across All Docs

```bash
# Find all mentions of "authentication"
bmd query "authentication" --dir ./docs

# Output: Ranked results with file paths
```

### Find Service Dependencies

If your docs describe microservices:

```bash
# What services depend on the auth service?
bmd depends auth-service

# Output: auth-service → [users-api, payments-api, admin-panel]
```

### List All Detected Services

```bash
bmd services

# Output: auth-service, users-api, payments-api, admin-panel, database
```

### Export Architecture Diagram

```bash
# Export dependency graph
bmd graph --format dot > architecture.dot

# Convert to image (requires graphviz)
dot -Tpng architecture.dot -o architecture.png
```

## Keyboard Reference

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `:N` | Jump to line N |
| `Ctrl+F` | Page down |
| `Ctrl+B` | Page up |

### Searching

| Key | Action |
|-----|--------|
| `/` | Search forward |
| `?` | Search backward |
| `n` | Next match |
| `N` | Previous match |

### Links

| Key | Action |
|-----|--------|
| `Tab` | Highlight next link |
| `Shift+Tab` | Highlight previous link |
| `Enter` | Follow highlighted link |
| `Backspace` | Go back to previous file |

### Themes & Help

| Key | Action |
|-----|--------|
| `t` | Cycle through themes |
| `h` or `?` | Show help overlay |
| `q` | Quit |

### Mouse

| Action | Effect |
|--------|--------|
| Click | Position cursor |
| Click link | Follow link |
| Drag | Select text |
| Scroll wheel | Navigate up/down |

## Tips & Tricks

### Open Multiple Files Quickly

```bash
# View README, then press Tab to navigate to linked files
bmd README.md
```

### Create a Simple Doc Index

```bash
# Build an index for full-text search
bmd index ./my-docs

# Later, search it
bmd query "important concept"
```

### Search from Command Line

```bash
# Search without opening viewer
bmd query "async" --dir ./docs --format json | jq '.results'
```

### View in Specific Theme

Currently, themes are selected inside BMD with `t`. To persistently use a theme, modify the code in `internal/theme/colors.go` and rebuild.

### High-DPI / Retina Display

BMD scales with your terminal font size. For HiDPI displays:
1. Increase font size in your terminal settings
2. Run BMD normally — it adapts automatically

### SSH Remote Viewing

```bash
# View files over SSH
ssh user@host bmd /path/to/file.md

# The rendering works over SSH with proper TERM setting
# Set TERM=xterm-256color for best colors
```

## Troubleshooting

### Images Not Rendering

BMD auto-detects your terminal's capabilities. If images don't appear:

```bash
# Check your terminal environment
echo $TERM
echo $KITTY_WINDOW_ID    # If using Kitty
echo $ITERM_PROGRAM       # If using iTerm2
```

Images should work in: Alacritty, Kitty, iTerm2, WezTerm, xterm with Sixel.

### Colors Look Wrong

```bash
# Ensure 256-color support
export TERM=xterm-256color
bmd file.md
```

### Scrolling Feels Slow

Scrolling is fast, but terminal rendering depends on your system. Try:
1. Update your terminal to the latest version
2. Reduce font size slightly (faster redraw)
3. Close other terminal tabs

### Theme Doesn't Change

Press `t` inside BMD. If nothing happens:
1. Ensure your terminal supports 256 colors
2. Try a different theme
3. Check that your terminal supports color output

## Next Steps

- 📖 Read [README.md](README.md) for full feature list
- 🏗️ Check [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- 💻 View [COMMANDS.md](COMMANDS.md) for knowledge system API
- 🤝 Contribute on [GitHub](https://github.com/flurryhead/bmd)

---

Happy reading! 🚀
