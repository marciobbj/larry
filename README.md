# Larry Text Editor

A minimalist general purpose terminal-based text editor written in Go.

## Features (MVP)

- Visual cursor that follows text input and navigation
- Character insertion and backspace deletion
- Enter for new lines
- Automatic scrolling
- Simple status line showing file name and cursor position
- Quit with Ctrl+C
- Modular keymap for easy shortcut customization
- Selecting text with shift + arrow keys

## Building and Running

1. Ensure Go is installed (1.18+ recommended).
2. Clone or navigate to the project directory.
3. Install dependencies: `go mod tidy`
4. Build: `go build ./cmd/larry-text-editor`
5. Run: `./larry-text-editor`

## Usage

- Type characters to insert text.
- Use arrow keys to move the cursor.
- Backspace to delete characters.
- Enter to create new lines.
- Ctrl+C or 'q' to quit.

## Key Bindings

Key bindings are defined in `internal/ui/model.go` and can be easily customized:

- Up/Down/Left/Right: Arrow keys
- Backspace: Delete character
- Enter: New line
- Ctrl+C or 'q': Quit

To modify shortcuts, edit the `DefaultKeyMap` struct and update the keys as needed.

## Future Enhancements

- [x] Line numbers
- [x] Selecting text
- [x] File picker
- [x] Optimized file loading
- [] Layout and UI improvements
- [x] File loading and saving
- [x] Word wrapping
- [] Syntax highlighting
- [] Undo/redo functionality
- [] Markdown instant visualization
