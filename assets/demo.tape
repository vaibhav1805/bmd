# VHS tape for bmd's README demo GIF.
# Regenerate with: brew install vhs && vhs assets/demo.tape
# Run from the repo root; uses the repo's own real docs as demo content
# (read-only the whole way through -- never saves, never edits).

Output assets/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "clear"
Enter
Show

Type "./bmd README.md"
Sleep 500ms
Enter
Sleep 2s

Down 12
Sleep 1.5s
Down 12
Sleep 1.5s

# Cross-document search across the whole repo's docs
Type "/"
Sleep 500ms
Type "vim keybindings"
Sleep 500ms
Enter
Sleep 2s

Escape
Sleep 500ms

# Split-pane browser: file list + live preview
Type "b"
Sleep 1s
Down 2
Sleep 800ms
Down 2
Sleep 1.2s
Escape
Sleep 500ms

# Full directory browser
Type "B"
Sleep 1.5s
Down 3
Sleep 1.5s

Type "q"
Sleep 500ms
