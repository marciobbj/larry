<div align="center">
  
# Larry - The Text Editor

<img src="assets/larry_cat.png" alt="Larry The Cat" width="200">

A minimalist, high-performance TUI text editor written in Go.

</div>

## Features

- **Optimized File Loader**: Custom rendering engine optimizing performance for large files (O(1) input, O(H) rendering).
- **Core Functionality**:
    - Easy navigation with Larry Movements (more info in the key bindings section)
    - Standard commands like copy, cut, paste, undo, redo, select all, etc. No need to learn new commands.
    - File Loading/Saving using modern file picker
    - Very easy to use and navigate.
- **Search & Navigation**: Efficient text search using Boyer-Moore algorithm with visual highlighting and result navigation.
- **Syntax Highlighting**: Supports 200+ languages via `chroma`, automatically detected by file extension.
- **UI**:
    - Clean, distraction-free interface
    - Absolute line numbers
    - Visual cursor, selection, and search result highlighting
    - Status reporting (without clutter)

## Why should I use Larry?
Larry offers a perfect balance for users who need more power than Nano without the steep learning curve of Vim. It delivers robust features in a simple, intuitive interface. It's very fast and it runs in your terminal.

## Installation

1. Ensure Go 1.18+ is installed.
2. Clone the repository.
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Build:
   ```bash
   go build ./cmd/larry
   ```
5. Run:
   ```bash
   ./larry [OPTIONS] [FILE]
   ```

   Use `./larry --help` for detailed usage information.
6. Add Larry to your PATH if you want:
   ```bash
   export PATH="<path_to_larry_bin/:$PATH"
   ```

## Command Line Usage

Larry supports several command line options for enhanced usage:

### Arguments
- `FILE`: Optional file to open on startup

### Options
- `-config <path>`: Load configuration from a JSON file
- `-help`: Display help information and exit

### Examples
```bash
# Start with empty file
larry

# Open a specific file
larry myfile.txt

# Open with custom configuration
larry -config ~/.config/larry/config.json myfile.txt

# Show help
larry --help
```

## Key Bindings

### General
| Action | Shortcut |
|--------|----------|
| **Quit** | `Ctrl+Q` |
| **Save** | `Ctrl+S` |
| **Open File** | `Ctrl+O` |
| **Search** | `Ctrl+F` |
| **Undo** | `Ctrl+Z` |
| **Redo** (disabled)| `Ctrl+Shift+Z` |
| **Copy** | `Ctrl+C` |
| **Cut** | `Ctrl+X` |
| **Paste** | `Ctrl+V` |
| **Go to Line** | `Ctrl+G` |
| **Toggle Help** | `Ctrl+H` |
| **Select All** | `Ctrl+A` |
| **Indent** | `TAB` |
| **Dedent** | `Shift+Tab` |

### Navigation
| Action | Shortcut |
|--------|----------|
| **Move Cursor** | Arrow Keys |
| **Jump Word Left/Right** | `Ctrl+←/→` |
| **Jump 5 Lines Up/Down** | `Ctrl+↑/↓` |
| **Line Start** | `Home` |
| **Line End** | `End` |
| **File Start** | `Ctrl+Home` |
| **File End** | `Ctrl+End` |

### Selection
| Action | Shortcut |
|--------|----------|
| **Select Text** | `Shift+Arrow` |
| **Select Word Left/Right** | `Ctrl+Shift+←/→` |
| **Select 5 Lines Up/Down** | `Ctrl+Shift+↑/↓` |
| **Select to Line Start** | `Shift+Home` |
| **Select to Line End** | `Shift+End` |

## Search & Find

Larry includes an efficient text search feature powered by the **Boyer-Moore algorithm**, providing fast and responsive search capabilities across your files. This makes Larry's search extremely fast, even for large files with complex search patterns.

### Search Features

- **Case-Sensitive**: Searches exactly as typed
- **Multi-Line Support**: Finds matches across all lines in the file
- **Navigation**: Jump directly to any search result
- **Persistent**: Search state remains active until you exit search mode

## Configuration

Larry supports configuration via a JSON file.

### Usage
Run Larry with the `-config` flag:
```bash
./larry -config config.json
```

### Configuration Options (`config.json`)
```json
{
  "theme": "dracula",
  "tab_width": 4,
  "line_numbers": true
}
```
| Field | Description | Default |
|-------|-------------|---------|
| `theme` | Syntax highlighting theme (e.g., `dracula`, `monokai`, `nord`, `github`) | `dracula` |
| `tab_width` | Number of spaces for a tab character | `4` |
| `line_numbers` | Show or hide line numbers | `true` |

## Roadmap

- [x] Line numbers
- [x] Selecting text
- [x] File picker
- [x] Optimized file loading
- [x] File loading and saving
- [x] Syntax highlighting
- [x] Undo/redo functionality
- [x] Search 
- [ ] Replace
- [x] Go to line
- [ ] Markdown instant visualization
- [ ] LSP support
- [x] Config file support
- [ ] Plugin system
- [x] Theme support
- [x] Add Help Menu and Docs
- [ ] Add to a remote package manager
- [ ] Improve resizing and responsiveness (add scaling etc)
- [x] Agile navigation movements (Ctrl+arrows for word/line jumping)
