<div align="center">

<img src="assets/larry.svg" alt="Larry" width="160" height="160">

# Larry

**A minimalist, high-performance TUI text editor written in Go.**<br />
*More power than Nano, without the steep learning curve of Vim — made for humans.*

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-a6e3a1?style=flat-square)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v1.0.0-c6a0f6?style=flat-square)](https://github.com/marciobbj/larry/releases)

</div>

---

```text
 ▾ my-project/              │  main.go ●  │  README.md  │  config.json             1/3
────────────────────────────│──────────────────────────────────────────────────────────
 ▾ cmd/                   • │    1 package main
 │ ▾ larry/               • │    2
 │ │ ● main.go            M │ ▎  3 import "fmt"
 ▾ internal/                │    4
 │ ▸ ui/                    │ ▎  5 func main() {
   README.md              M │ ▎  6     fmt.Println("Hello from Larry!")
   notes.md               ? │ ▁  7 }
────────────────────────────┴──────────────────────────────────────────────────────────
  main.go  ●  ⎇ main                              Ctrl+o Hide Tree │ Ctrl+p Finder │ …
```

## Why Larry?

Larry sits in the sweet spot between **Nano** and **Neovim**: familiar shortcuts (`Ctrl+C/V/X/Z/S/F`), zero modal friction, and modern IDE ergonomics right inside your terminal — collapsible file tree, multi-file tabs, live Git diff signs, project-wide search & replace, split Markdown preview, and live Omarchy theming.

## Features

- **Collapsible Lateral File Tree (`Leader+O`)**: Neovim-style sidebar with expandable/collapsible directories, indent guides, hidden file toggle (`.`), resizing (`[` / `]`), and persistent state across collapses.
- **Multi-File Tabs (`Leader+N` / `Leader+D`)**: Work across multiple files simultaneously with independent cursors, scroll positions, undo/redo stacks, modified indicators (`●`), duplicate filename disambiguation (`dir/file`), and horizontal tab overflow scrolling.
- **Live Git Integration**:
  - Real-time line diff markers right in the line-number gutter (`▎` added in green, `▎` modified in yellow, `▁` deleted in red) computed live against `HEAD` as you type.
  - File status badges (`M`, `A`, `?`, `U`, `D`) and dirty folder indicators (`•`) in the lateral file tree.
  - Active branch indicator (`⎇ branch`) in the status bar.
- **Global Finder (`Leader+P`)**: Fast project-wide fuzzy file search and concurrent live grep (switch with `Tab`), automatically ignoring binary files and directories like `.git` or `node_modules`.
- **Global Replace (`Leader+W`)**: Project-scoped search & replace with live `before → after` diff previews, per-file `[x]`/`[ ]` selection toggles, confirmation guard, and automatic buffer sync across all open tabs.
- **Syntax Highlighting**: 200+ languages via `chroma` with cached full-file tokenization for multi-line constructs (XML tags/comments, Markdown fenced code blocks, embedded HTML in Markdown, and block comments), with O(H) fallback for huge files.
- **Word Wrap Toggle (`Leader+L`)**: No-wrap with smooth horizontal scrolling by default (clean consecutive line numbers with zero gaps), toggleable live to aligned word wrap anytime.
- **Instant Markdown Preview (`Leader+U`)**: Side-by-side live rendered Markdown split view synchronized with your cursor position.
- **Omarchy Native Theming**: Set `"theme": "omarchy"` (automatic default on Omarchy systems) to adopt your desktop color scheme across the entire UI and syntax highlighting — with **live reload** when you switch Omarchy themes.

## Installation

### Pre-built Binaries (GitHub Releases)

Download the latest release archive for Linux, macOS, or Windows from [GitHub Releases](https://github.com/marciobbj/larry/releases/latest), extract it, and place `larry` on your `PATH`:

```bash
tar -xzf larry_v1.0.0_linux_amd64.tar.gz
sudo mv larry /usr/local/bin/
larry -version
```

### Build from Source

```bash
git clone https://github.com/marciobbj/larry.git
cd larry
go mod tidy
go build -o larry ./cmd/larry
```

Optionally install to your `PATH`:

```bash
sudo mv larry /usr/local/bin/
# or
mkdir -p ~/.local/bin && mv larry ~/.local/bin/
```

## Command Line Usage

```bash
larry [OPTIONS] [FILE... | DIRECTORY]
```

### Options

| Flag | Description |
|------|-------------|
| `-config <path>` | Load a custom JSON configuration file |
| `-theme <name>` | Override the configured color theme (e.g. `omarchy`, `dracula`, `nord`, `monokai`, `github`) |
| `-help` | Show CLI usage and configuration help |

### Examples

```bash
# Start Larry (opens welcome screen)
larry

# Open a directory directly with the lateral file tree focused
larry .

# Open a single file
larry main.go

# Open multiple files directly in tabs
larry cmd/larry/main.go internal/ui/model.go README.md

# Preview with Omarchy live color scheme without changing config.json
larry -theme omarchy .

# Use a custom config file
larry -config ./custom_config.json myfile.txt
```

## Key Bindings

`Leader` defaults to `Ctrl` (configurable to `alt` via `"leader_key"` in `config.json`).

### General & Editing

| Action | Shortcut |
|--------|----------|
| **Quit** | `Leader+Q` |
| **Save** | `Leader+S` |
| **Search (Boyer-Moore)** | `Leader+F` |
| **Replace (Current File)** | `Leader+T` |
| **Global Replace (Project)** | `Leader+W` |
| **Global Finder (Files / Grep)** | `Leader+P` |
| **Go to Line** | `Leader+G` |
| **Undo / Redo** | `Leader+Z` / `Leader+R` |
| **Copy / Cut / Paste** | `Leader+C` / `Leader+X` / `Leader+V` |
| **Select All** | `Leader+A` |
| **Indent / Dedent** | `Tab` / `Shift+Tab` |
| **Toggle Word Wrap** | `Leader+L` |
| **Markdown Split Preview** | `Leader+U` |
| **Toggle Help Menu** | `Leader+H` |

### Tabs

| Action | Shortcut |
|--------|----------|
| **New Tab** | `Leader+N` |
| **Close Current Tab** | `Leader+D` |
| **Previous Tab** | `Alt+←` or `Alt+[` *(or `Ctrl+←` when `leader_key` is `alt`)* |
| **Next Tab** | `Alt+→` or `Alt+]` *(or `Ctrl+→` when `leader_key` is `alt`)* |

### Lateral File Tree

| Action | Shortcut |
|--------|----------|
| **Toggle File Tree (Open / Collapse)** | `Leader+O` *(or `q` while tree is focused)* |
| **Switch Focus (Tree ↔ Editor)** | `Leader+E` *(or `Esc` / `Tab` to return to editor)* |
| **Move Selection** | `↑`/`↓`, `k`/`j`, `PgUp`/`PgDown`, `g`/`G` |
| **Open File / Toggle Folder** | `Enter` *(moves focus to editor when opening a file)* |
| **Quick Peek (Open & Keep Tree Focus)** | `Space` or `o` |
| **Expand / Enter Child** | `→` or `l` |
| **Collapse Folder / Jump to Parent** | `←` or `h` |
| **Go Up Root Directory** | `-` or `Backspace` |
| **Toggle Hidden Files (Dotfiles)** | `.` or `H` |
| **Refresh Tree & Git Status** | `r` |
| **Resize Sidebar Width** | `[` / `]` *(or `<` / `>`)* |

### Navigation & Selection

| Action | Shortcut |
|--------|----------|
| **Move Cursor** | Arrow Keys |
| **Jump Word Left / Right** | `Leader+←` / `Leader+→` |
| **Jump 5 Lines Up / Down** | `Leader+↑` / `Leader+↓` |
| **Line Start / End** | `Home` / `End` |
| **File Start / End** | `Leader+Home` / `Leader+End` |
| **Select Text** | `Shift+Arrow` |
| **Select Word Left / Right** | `Leader+Shift+←/→` *(or `Alt+Shift+←/→`)* |
| **Select 5 Lines Up / Down** | `Leader+Shift+↑/↓` *(or `Alt+Shift+↑/↓`)* |
| **Select to Line Start / End** | `Shift+Home` / `Shift+End` |

## Search, Finder & Global Replace

- **In-File Search (`Leader+F`)**: Powered by the **Boyer-Moore** algorithm with real-time highlighting and jump-to-match centering (including horizontal scrolling when word wrap is off).
- **Global Finder (`Leader+P`)**: Press `Tab` to switch between **Fuzzy File Search** and **Live Grep** across your project, then `Enter` to open the file (and jump directly to the matching line).
- **Global Replace (`Leader+W`)**: Scoped strictly to the open project root. Type in **FIND** and **REPLACE** to preview matching files and inline `old → new` diffs, press `Tab` to toggle individual files with `Space` (or `Ctrl+A` / `a` for all), and confirm before applying. All affected open tabs reload automatically.

## Configuration

Larry loads its configuration automatically from:
1. `~/.config/larry/config.json` (Linux) or `~/Library/Application Support/larry/config.json` (macOS)
2. Fallback: `~/.config/larry/config.json`

### Example `config.json`

```json
{
  "theme": "omarchy",
  "tab_width": 4,
  "line_numbers": true,
  "word_wrap": false,
  "leader_key": "ctrl"
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `theme` | Syntax & UI theme (`omarchy`, `dracula`, `monokai`, `nord`, `github`, etc.). `omarchy` follows your active Omarchy desktop theme live. | `omarchy` (on Omarchy) / `dracula` |
| `tab_width` | Number of spaces per tab | `4` |
| `line_numbers` | Show or hide absolute line numbers (and Git gutter signs) | `true` |
| `word_wrap` | Wrap long lines instead of scrolling horizontally (toggle live with `Leader+L`) | `false` |
| `leader_key` | Base modifier key for shortcuts (`ctrl` or `alt`) | `ctrl` |

### Platform & Terminal Notes

- **Omarchy Linux**: Larry automatically reads `~/.local/state/omarchy/current/theme/colors.toml` (and legacy `~/.config/omarchy/current/theme/colors.toml`) and hot-reloads colors live when you switch themes.
- **Ghostty Users**: Ghostty binds `Ctrl+Shift+Arrow` by default (`jump_to_prompt` / tab switching). You can either use `Alt+Shift+Arrow` directly in Larry or unbind them in `~/.config/ghostty/config`:
  ```ini
  keybind = ctrl+shift+arrow_up=unbind
  keybind = ctrl+shift+arrow_down=unbind
  keybind = ctrl+shift+arrow_left=unbind
  keybind = ctrl+shift+arrow_right=unbind
  ```
- **macOS Users**: Since terminals rarely forward `Cmd`, set `"leader_key": "alt"` and map Option to Alt in your terminal preferences (Ghostty, Kitty, iTerm2, Alacritty).

## Roadmap

- [x] Line numbers & live Git diff gutter markers
- [x] Text selection & agile navigation (`Leader+Arrows`, `Shift+Arrows`)
- [x] Collapsible Neovim-style lateral file tree with Git status indicators
- [x] Multi-file tab system
- [x] Optimized file loading & syntax highlighting (200+ languages, Markdown + HTML, multi-line XML)
- [x] Configurable Word Wrap / horizontal scrolling
- [x] Undo / Redo & clipboard operations
- [x] Fast Boyer-Moore Search & Replace
- [x] Global Finder (Fuzzy Search & Live Grep)
- [x] Global Replace (project-scoped with live diff preview)
- [x] Instant split Markdown preview
- [x] Config file & theme support (including Omarchy live color scheme)
- [x] Built-in Help Menu & responsive resizing
- [ ] Publish to package managers (AUR / Homebrew)
- [ ] Extensibility & plugin / custom theme authoring

## License

MIT — see [LICENSE](LICENSE) for details.
