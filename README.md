# Larry - The Text Editor

![Larry The Cat](assets/larry_cat.png)

A minimalist, high-performance TUI text editor written in Go.

## Features

- **OptimizedFile Loader**: Custom rendering engine optimizing performance for large files (O(1) input, O(H) rendering).
- **Core Functionality**:
    - Undo/Redo (`Ctrl+Z`, `Ctrl+Shift+Z`)
    - Clipboard Support (`Ctrl+C`, `Ctrl+X`, `Ctrl+V`)
    - Selection with Shift+Arrows
    - File Loading/Saving using modern file picker
- **Syntax Highlighting**: Supports 200+ languages via `chroma`, automatically detected by file extension.
- **UI**: 
    - Clean, distraction-free interface
    - Absolute line numbers
    - Visual cursor and selection highlighting
    - Status reporting (without clutter)

## Installation

1. Ensure Go 1.18+ is installed.
2. Clone the repository.
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Build:
   ```bash
   go build ./cmd/larry-text-editor
   ```
5. Run:
   ```bash
   ./larry-text-editor [filename]
   ```

## Key Bindings

| Action | Shortcut |
|--------|----------|
| **Quit** | `Ctrl+Q` |
| **Save** | `Ctrl+S` |
| **Open File** | `Ctrl+O` |
| **Undo** | `Ctrl+Z` |
| **Redo** | `Ctrl+Shift+Z` |
| **Copy** | `Ctrl+C` |
| **Cut** | `Ctrl+X` |
| **Paste** | `Ctrl+V` |
| **Select All** | `Ctrl+A` |
| **Select Text**| `Shift + Arrays` |
| **Indent** | `TAB` |
| **Dedent** | `Shift+Tab` |
| **Navigation** | Arrow Keys |

## Roadmap

- [x] Line numbers
- [x] Selecting text
- [x] File picker
- [x] Optimized file loading (Engine 2.0)
- [x] File loading and saving
- [x] Syntax highlighting
- [x] Undo/redo functionality
- [ ] Search / Find & Replace
- [ ] Markdown instant visualization
- [ ] LSP support
- [ ] Config file support
- [ ] Plugin system
- [ ] Theme support