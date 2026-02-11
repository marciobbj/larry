<div align="center">
  
# Larry - The Text Editor

<img src="assets/larry_cat.png" alt="Larry The Cat" width="200">

A minimalist, high-performance TUI text editor written in Go.

</div>

## Features

- **Optimized File Loader**: Custom rendering engine optimizing performance for large files (O(1) input, O(H) rendering).
- **Core Functionality**:
    - Navigate through your files fast like a cat with Larry Movements (more info in the key bindings section). 
    - Standard commands like copy, cut, paste, undo, redo, select all, etc. No need to learn new commands.
    - File Loading/Saving using modern file picker
    - Very easy to use and navigate.
- **Search & Navigation**: Efficient text search using Boyer-Moore algorithm with visual highlighting and result navigation.
- **Global Finder**: Powerful multi-purpose search tool (`Leader+P`) supporting both fuzzy file searching and live text grep across the entire project. It automatically ignores binary/compiled files for a cleaner search experience.
- **Syntax Highlighting**: Supports 200+ languages via `chroma`, automatically detected by file extension.
- **UI**:
    - Clean, distraction-free interface
    - Absolute line numbers
    - Visual cursor, selection, and search result highlighting
    - Status reporting with automatic text wrapping
    - Fully responsive design that handles terminal resizing elegantly

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
5. Also create the config file. If you are using macOS see "Note for macOS users" down below.
   ```bash
   touch ~/.config/larry/config.json && echo '{"theme": "monokai", "tab_width": 4, "line_numbers": true, "leader_key": "ctrl"}' > ~/.config/larry/config.json
   ```
6. Run:
   ```bash
   ./larry [OPTIONS] [FILE]
   ```

   Use `./larry --help` for detailed usage information.
7. Add Larry to your PATH if you want:
   ```bash
   export PATH="<path_to_larry_bin/:$PATH"
   ```

## Command Line Usage

Larry supports several command line options for enhanced usage:

### Arguments
- `FILE`: Optional file to open on startup

### Options
- `-config <path>`: Load a specific configuration file (overrides default `~/.config/larry/config.json`)
- `-help`: Display help information and exit

### Examples
```bash
# Start Larry (loads ~/.config/larry/config.json automatically if present)
larry

# Open a specific file
larry myfile.txt

# Override the default configuration with a specific file
larry -config ./custom_config.json myfile.txt

# Show help
larry --help
```

## Key Bindings
Leader key is ctrl by default.
### General
| Action | Shortcut |
|--------|----------|
| **Quit** | `Leader+Q` |
| **Save** | `Leader+S` |
| **Open File** | `Leader+O` |
| **Search** | `Leader+F` |
| **Replace** | `Leader+T` |
| **Global Larry Finder** | `Leader+P` |
| **Undo** | `Leader+Z` |
| **Redo** | `Leader+R` |
| **Copy** | `Leader+C` |
| **Cut** | `Leader+X` |
| **Paste** | `Leader+V` |
| **Go to Line** | `Leader+G` |
| **Toggle Help** | `Leader+H` |
| **Select All** | `Leader+A` |
| **Markdown Preview** | `Leader+U` |
| **Indent** | `TAB` |
| **Dedent** | `Shift+Tab` |

### Navigation
| Action | Shortcut |
|--------|----------|
| **Move Cursor** | Arrow Keys |
| **Jump Word Left/Right** | `Leader+←/→` |
| **Jump 5 Lines Up/Down** | `Leader+↑/↓` |
| **Line Start** | `Home` |
| **Line End** | `End` |
| **File Start** | `Leader+Home` |
| **File End** | `Leader+End` |


### Selection
| Action | Shortcut |
|--------|----------|
| **Select Text** | `Shift+Arrow` |
| **Select Word Left/Right** | `Leader+Shift+←/→` |
| **Select 5 Lines Up/Down** | `Leader+Shift+↑/↓` |
| **Select to Line Start** | `Shift+Home` |
| **Select to Line End** | `Shift+End` |

## Search & Find

Larry includes an efficient text search feature powered by the **Boyer-Moore algorithm**, providing fast and responsive search capabilities across your files. This makes Larry's search extremely fast, even for large files with complex search patterns.

### Search Features

- **FAST** like a cat.

### Global Finder

The Global Finder is a powerful tool for navigating your project. Trigger it with `Leader+P`.

- **Fuzzy Search**: Search for files by name with fuzzy matching.
- **Live Grep**: Search for text patterns across all files in your project in real-time.
- **Switch Modes**: Use `Tab` to seamlessly switch between Fuzzy Search and Live Grep modes.
- **Smart Filtering**: Automatically ignores binary and compiled files to ensure a clean search experience.
- **Navigate Results**: Use `Up`/`Down` arrows to navigate through the results and press `Enter` to open the selection.

## Configuration

Larry is designed to be easily customizable via a JSON configuration file. 

### Automatic Loading
Larry automatically looks for its configuration in the following locations:
1.  **Standard**: `~/.config/larry/config.json` (on Linux) or `~/Library/Application Support/larry/config.json` (on macOS).
2.  **Fallback**: `~/.config/larry/config.json` (common fallback for macOS users).

### Custom Configuration Override
To use a different configuration file, use the `-config` flag:

```bash
larry -config path/to/your/config.json
```

### Configuration Options (`config.json`)
```json
{
  "theme": "dracula",
  "tab_width": 4,
  "line_numbers": true,
  "leader_key": "ctrl"
}
```
| Field | Description | Default |
|-------|-------------|---------|
| `theme` | Syntax highlighting theme (e.g., `dracula`, `monokai`, `nord`, `github`) | `dracula` |
| `tab_width` | Number of spaces for a tab character | `4` |
| `line_numbers` | Show or hide line numbers | `true` |
| `leader_key` | Base key for shortcuts (e.g., `ctrl`, `alt`). | `ctrl` |

> **Note for macOS users**: The `cmd` key is generally not natively supported as a modifier by terminal emulators. We recommend setting `leader_key` to `alt` (which corresponds to the Option key) by mapping `option` to `alt` in your terminal's settings (e.g., iTerm2, Ghostty, Kitty etc).



## Roadmap

- [x] Line numbers
- [x] Selecting text
- [x] File picker
- [x] Optimized file loading
- [x] File loading and saving
- [x] Syntax highlighting
- [x] Undo/redo functionality
- [x] Search 
- [x] Global Finder (Fuzzy & Live Grep)
- [x] Leader Key config for cross-platform support
- [x] Replace
- [x] Go to line
- [x] Markdown instant visualization
- [ ] Show modified lines with git integration
- [ ] Global Replace
- [ ] LSP support
- [x] Config file support
- [ ] Plugin system
- [x] Theme support
- [x] Add Help Menu and Docs
- [ ] Add to a remote package manager
- [x] Improve resizing and responsiveness
- [x] Agile navigation movements (Leader+arrows for word/line jumping)
- [ ] Let Larry be more hackable, allowing users to add their own features, color schemes, etc
- [ ] Debugger
