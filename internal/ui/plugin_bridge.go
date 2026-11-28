package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"larry/internal/config"
	"larry/internal/plugin"

	tea "github.com/charmbracelet/bubbletea"
)

// pluginAsyncEventMsg wraps a background plugin/LSP event for Bubble Tea's Update loop.
type pluginAsyncEventMsg struct {
	evt plugin.AsyncEvent
}

// pluginBufferDebounceMsg fires after a short typing pause to emit buffer_change
// for linters and LSP didChange without lagging every keystroke.
type pluginBufferDebounceMsg struct {
	seq int
}

const bufferChangeDebounceDelay = 250 * time.Millisecond

func waitPluginAsyncCmd(mgr *plugin.Manager) tea.Cmd {
	if mgr == nil {
		return nil
	}
	ch := mgr.AsyncEvents()
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return nil
		}
		return pluginAsyncEventMsg{evt: evt}
	}
}

func bufferChangeDebounceCmd(seq int) tea.Cmd {
	return tea.Tick(bufferChangeDebounceDelay, func(time.Time) tea.Msg {
		return pluginBufferDebounceMsg{seq: seq}
	})
}

// modelEditorAPI adapts *Model to plugin.EditorAPI during a Bubble Tea Update cycle.
type modelEditorAPI struct {
	m *Model
}

func (m *Model) editorAPI() plugin.EditorAPI {
	return &modelEditorAPI{m: m}
}

func (a *modelEditorAPI) FileName() string {
	return a.m.FileName
}

func (a *modelEditorAPI) FilePath() string {
	if a.m.FileName == "" {
		return ""
	}
	return normalizeAbsPath(a.m.FileName)
}

func (a *modelEditorAPI) FileType() string {
	return detectFileType(a.m.FileName)
}

func (a *modelEditorAPI) RootDir() string {
	if a.m.fileTree.RootPath != "" {
		return normalizeAbsPath(a.m.fileTree.RootPath)
	}
	if a.m.FileName != "" {
		return filepath.Dir(normalizeAbsPath(a.m.FileName))
	}
	wd, err := os.Getwd()
	if err == nil {
		return wd
	}
	return "."
}

func (a *modelEditorAPI) Lines() []string {
	return append([]string(nil), a.m.Lines...)
}

func (a *modelEditorAPI) LineCount() int {
	return len(a.m.Lines)
}

func (a *modelEditorAPI) GetLine(row int) string {
	if row < 0 || row >= len(a.m.Lines) {
		return ""
	}
	return a.m.Lines[row]
}

func (a *modelEditorAPI) SetLine(row int, text string) {
	if row < 0 || row >= len(a.m.Lines) {
		return
	}
	if a.m.Lines[row] == text {
		return
	}
	before := append([]string(nil), a.m.Lines...)
	beforeRow, beforeCol := a.m.CursorRow, a.m.CursorCol

	a.m.Lines[row] = text
	if a.m.CursorRow == row {
		a.m.CursorCol = clampInt(a.m.CursorCol, 0, lineRuneLen(a.m.Lines, row))
	}
	after := append([]string(nil), a.m.Lines...)
	a.m.pushUndo(EditOp{
		Type:        OpReplaceAll,
		BeforeLines: before,
		AfterLines:  after,
		BeforeRow:   beforeRow,
		BeforeCol:   beforeCol,
		AfterRow:    a.m.CursorRow,
		AfterCol:    a.m.CursorCol,
	})
	a.m.markModified()
}

func (a *modelEditorAPI) SetLines(lines []string) {
	if len(lines) == 0 {
		lines = []string{""}
	}
	// Skip no-op replacements (e.g., gofmt on already-formatted file).
	if len(lines) == len(a.m.Lines) {
		same := true
		for i := range lines {
			if lines[i] != a.m.Lines[i] {
				same = false
				break
			}
		}
		if same {
			return
		}
	}

	before := append([]string(nil), a.m.Lines...)
	beforeRow, beforeCol := a.m.CursorRow, a.m.CursorCol

	a.m.Lines = append([]string(nil), lines...)
	a.m.CursorRow = clampInt(a.m.CursorRow, 0, len(a.m.Lines)-1)
	a.m.CursorCol = clampInt(a.m.CursorCol, 0, lineRuneLen(a.m.Lines, a.m.CursorRow))
	a.m.selecting = false
	a.m.clearExtraCursors()

	after := append([]string(nil), a.m.Lines...)
	a.m.pushUndo(EditOp{
		Type:        OpReplaceAll,
		BeforeLines: before,
		AfterLines:  after,
		BeforeRow:   beforeRow,
		BeforeCol:   beforeCol,
		AfterRow:    a.m.CursorRow,
		AfterCol:    a.m.CursorCol,
	})
	a.m.markModified()
}

func (a *modelEditorAPI) InsertText(row, col int, text string) {
	if text == "" {
		return
	}
	if len(a.m.Lines) == 0 {
		a.m.Lines = []string{""}
	}
	row = clampInt(row, 0, len(a.m.Lines)-1)
	col = clampInt(col, 0, lineRuneLen(a.m.Lines, row))
	a.m.CursorRow = row
	a.m.CursorCol = col
	a.m.pushUndo(EditOp{Type: OpInsert, Row: row, Col: col, Text: text})
	*a.m = a.m.insertTextAtCursor(text)
}

func (a *modelEditorAPI) DeleteRange(startRow, startCol, endRow, endCol int) {
	if len(a.m.Lines) == 0 {
		return
	}
	startRow = clampInt(startRow, 0, len(a.m.Lines)-1)
	endRow = clampInt(endRow, 0, len(a.m.Lines)-1)
	startCol = clampInt(startCol, 0, lineRuneLen(a.m.Lines, startRow))
	endCol = clampInt(endCol, 0, lineRuneLen(a.m.Lines, endRow))

	a.m.startRow = startRow
	a.m.startCol = startCol
	a.m.CursorRow = endRow
	a.m.CursorCol = endCol
	a.m.selecting = true
	deleted := a.m.getSelectedText()
	if deleted != "" {
		sr, sc := startRow, startCol
		if sr > endRow || (sr == endRow && sc > endCol) {
			sr, sc = endRow, endCol
		}
		a.m.pushUndo(EditOp{Type: OpDelete, Row: sr, Col: sc, Text: deleted})
		*a.m = a.m.deleteSelectedText()
	} else {
		a.m.selecting = false
	}
}

func (a *modelEditorAPI) IsModified() bool {
	return a.m.Modified
}

func (a *modelEditorAPI) Cursor() (int, int) {
	return a.m.CursorRow, a.m.CursorCol
}

func (a *modelEditorAPI) SetCursor(row, col int) {
	if len(a.m.Lines) == 0 {
		a.m.CursorRow, a.m.CursorCol = 0, 0
		return
	}
	a.m.CursorRow = clampInt(row, 0, len(a.m.Lines)-1)
	a.m.CursorCol = clampInt(col, 0, lineRuneLen(a.m.Lines, a.m.CursorRow))
	a.m.selecting = false
	*a.m = a.m.updateViewport()
}

func (a *modelEditorAPI) Selection() (bool, int, int, int, int) {
	if !a.m.selecting {
		return false, a.m.CursorRow, a.m.CursorCol, a.m.CursorRow, a.m.CursorCol
	}
	sr, sc, er, ec := a.m.startRow, a.m.startCol, a.m.CursorRow, a.m.CursorCol
	if sr > er || (sr == er && sc > ec) {
		sr, er = er, sr
		sc, ec = ec, sc
	}
	return true, sr, sc, er, ec
}

func (a *modelEditorAPI) SelectedText() string {
	return a.m.getSelectedText()
}

func (a *modelEditorAPI) OpenFile(path string, row, col int) error {
	updated, err := a.m.openFile(path)
	if err != nil {
		return err
	}
	if row >= 0 && len(updated.Lines) > 0 {
		updated.CursorRow = clampInt(row, 0, len(updated.Lines)-1)
		updated.CursorCol = clampInt(col, 0, lineRuneLen(updated.Lines, updated.CursorRow))
		updated = updated.updateViewport()
	}
	updated.fileTree.Focused = false
	*a.m = updated
	return nil
}

func (a *modelEditorAPI) SaveFile() error {
	if a.m.FileName == "" {
		return fmt.Errorf("cannot save unnamed buffer")
	}
	updated, err := a.m.saveCurrentFile(a.m.FileName)
	*a.m = updated
	return err
}

func (a *modelEditorAPI) SetStatus(msg string) {
	a.m.statusMsg = msg
}

func (a *modelEditorAPI) ShowPicker(title string, items []plugin.PickerItem, onSelect func(plugin.PickerItem)) {
	entries := make([]PickerEntry, 0, len(items))
	for _, it := range items {
		entries = append(entries, PickerEntry{
			ID:     it.ID,
			Label:  it.Label,
			Detail: it.Detail,
			Path:   it.Path,
			Line:   it.Line,
			Col:    it.Col,
			Item:   it,
		})
	}
	a.m.picker = NewPickerModal(title, "Filter...", entries, false, func(chosen PickerEntry) {
		if onSelect != nil {
			onSelect(chosen.Item)
		} else if chosen.Path != "" {
			_ = a.OpenFile(chosen.Path, chosen.Line, chosen.Col)
		}
	})
	a.m.pickerActive = true
}

func (a *modelEditorAPI) ShowPanel(title string, lines []string) {
	a.m.pluginPanel = PanelState{
		Visible: true,
		Title:   title,
		Lines:   append([]string(nil), lines...),
		Scroll:  0,
	}
	a.m.invalidateVisualCache()
}

func (a *modelEditorAPI) ClosePanel() {
	a.m.pluginPanel.Visible = false
	a.m.invalidateVisualCache()
}

func (a *modelEditorAPI) ShowHover(title string, lines []string) {
	a.m.pluginHover = HoverState{
		Visible: true,
		Title:   title,
		Lines:   append([]string(nil), lines...),
		Scroll:  0,
	}
}

func (a *modelEditorAPI) CloseHover() {
	a.m.pluginHover.Visible = false
}

func (a *modelEditorAPI) ShowCompletions(items []plugin.CompletionItem, startRow, startCol int, onSelect func(plugin.CompletionItem)) {
	if len(items) == 0 {
		a.m.completion.Close()
		return
	}
	if startRow < 0 {
		startRow = a.m.CursorRow
	}
	if startRow < 0 || startRow >= len(a.m.Lines) {
		a.m.completion.Close()
		return
	}
	if startCol < 0 {
		startCol = findIdentPrefixStartCol(a.m.Lines[startRow], a.m.CursorCol)
	}
	prefix := ""
	if a.m.CursorRow == startRow && a.m.CursorCol >= startCol {
		runes := []rune(a.m.Lines[startRow])
		end := a.m.CursorCol
		if end > len(runes) {
			end = len(runes)
		}
		if startCol <= end {
			prefix = string(runes[startCol:end])
		}
	}
	a.m.completion.Open(items, startRow, startCol, prefix, onSelect)
}

func (a *modelEditorAPI) CloseCompletions() {
	a.m.completion.Close()
}

func (a *modelEditorAPI) ThemeName() string {
	return a.m.Config.Theme
}

func (a *modelEditorAPI) Palette() map[string]string {
	p := activePalette
	return map[string]string{
		"accent":       string(p.Accent),
		"accent_alt":   string(p.AccentAlt),
		"background":   string(p.Background),
		"surface":      string(p.Surface),
		"foreground":   string(p.Foreground),
		"subtle":       string(p.SubtleFg),
		"muted":        string(p.MutedFg),
		"border":       string(p.Border),
		"red":          string(p.Red),
		"green":        string(p.Green),
		"yellow":       string(p.Yellow),
		"blue":         string(p.Blue),
		"cyan":         string(p.Cyan),
		"magenta":      string(p.Magenta),
	}
}

// ── Unified Save Helper ──

// saveCurrentFile fires before_save hooks (e.g., formatters), writes the buffer
// to disk, updates tab/git/tree caches, and fires after_save hooks (e.g., linters/LSP).
func (m Model) saveCurrentFile(filename string) (Model, error) {
	if filename == "" {
		return m, fmt.Errorf("filename is empty")
	}
	m.FileName = filename

	if m.plugins != nil {
		m.plugins.Emit(m.editorAPI(), plugin.EventBeforeSave, nil)
	}

	content := strings.Join(m.Lines, "\n")
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		m.statusMsg = "Error saving: " + err.Error()
		return m, err
	}

	m.statusMsg = "Saved: " + filename
	m.Modified = false
	m.saveActiveTabState()
	m.invalidateGitCache()
	m.fileTree.RefreshGitStatus()
	m.fileTree.SetActiveFile(filename)

	if m.plugins != nil {
		m.plugins.Emit(m.editorAPI(), plugin.EventAfterSave, nil)
	}
	return m, nil
}

// ── Command Palette ──

func (m Model) openCommandPalette() Model {
	leaderKey := normalizeLeaderKey(m.Config.LeaderKey)
	leader := displayLeaderKey(leaderKey)
	secondary := "Alt"
	if leaderKey == "alt" {
		secondary = "Ctrl"
	}

	entries := []PickerEntry{
		{ID: "builtin.save", Category: "Editor", Label: "Save File", Shortcut: leader + "+s"},
		{ID: "builtin.toggle_wrap", Category: "Editor", Label: "Toggle Word Wrap", Shortcut: leader + "+l"},
		{ID: "builtin.toggle_tree", Category: "Editor", Label: "Toggle File Tree", Shortcut: leader + "+o"},
		{ID: "builtin.finder", Category: "Search", Label: "Global Finder (Files / Grep)", Shortcut: leader + "+p"},
		{ID: "builtin.global_replace", Category: "Search", Label: "Global Replace", Shortcut: leader + "+w"},
		{ID: "builtin.help", Category: "Editor", Label: "Keyboard Shortcuts Help", Shortcut: leader + "+h"},
		{ID: "builtin.diagnostics_list", Category: "Diagnostics", Label: "Show File Diagnostics"},
		{ID: "builtin.diagnostics_next", Category: "Diagnostics", Label: "Jump to Next Diagnostic"},
		{ID: "builtin.plugins_list", Category: "Plugins", Label: "List Loaded Plugins"},
		{ID: "builtin.plugins_reload", Category: "Plugins", Label: "Reload All Plugins"},
		{ID: "builtin.plugins_logs", Category: "Plugins", Label: "Show Plugin Logs"},
	}

	if !m.Config.Plugins.IsDisabled("colorschemes") {
		entries = append([]PickerEntry{
			{
				ID:       "builtin.colorschemes",
				Category: "Theme",
				Label:    "Switch Colorscheme",
				Detail:   fmt.Sprintf("%d curated themes · live preview", len(BuiltinColorSchemes())),
				Shortcut: secondary + "+t",
			},
		}, entries...)
	}

	if m.plugins != nil {
		for _, c := range m.plugins.Commands() {
			cat := c.Category
			if cat == "" {
				cat = "Plugin"
			}
			entries = append(entries, PickerEntry{
				ID:       "plugin:" + c.ID,
				Category: cat,
				Label:    c.Title,
				Detail:   c.Description,
				Shortcut: c.Key,
			})
		}
	}

	m.picker = NewPickerModal("Command Palette", "Type a command...", entries, true, nil)
	m.pickerActive = true
	return m
}

func (m Model) executePaletteEntry(chosen PickerEntry) (Model, tea.Cmd) {
	if strings.HasPrefix(chosen.ID, "plugin:") {
		cmdID := strings.TrimPrefix(chosen.ID, "plugin:")
		if m.plugins != nil {
			_ = m.plugins.RunCommand(m.editorAPI(), cmdID)
		}
		return m.updateViewport(), nil
	}

	switch chosen.ID {
	case "builtin.colorschemes":
		m = m.openThemePicker()
	case "builtin.save":
		if m.FileName != "" {
			var err error
			m, err = m.saveCurrentFile(m.FileName)
			_ = err
		} else {
			m.saving = true
			m.textInput.Focus()
			m.textInput.SetValue("")
			m.textInput.Prompt = "Filename: "
		}
	case "builtin.toggle_wrap":
		m.Config.WordWrap = !m.Config.WordWrap
		m.xOffset = 0
		m.invalidateVisualCache()
		m = m.updateViewport()
	case "builtin.toggle_tree":
		m = m.toggleFileTree()
	case "builtin.finder":
		return m.openGlobalFinder()
	case "builtin.global_replace":
		m = m.openGlobalReplace()
	case "builtin.help":
		m.showHelp = true
		m.helpScroll = 0
	case "builtin.diagnostics_list":
		m = m.openDiagnosticsPicker()
	case "builtin.diagnostics_next":
		m = m.jumpNextDiagnostic()
	case "builtin.plugins_list":
		m = m.showLoadedPluginsPanel()
	case "builtin.plugins_reload":
		if m.plugins != nil {
			m.plugins.Reload(m.editorAPI())
			m.statusMsg = m.plugins.StartupNotice()
		}
	case "builtin.plugins_logs":
		if m.plugins != nil {
			logs := m.plugins.Logs()
			if len(logs) == 0 {
				logs = []string{"(no plugin logs yet)"}
			}
			m.pluginPanel = PanelState{
				Visible: true,
				Title:   "Plugin Logs",
				Lines:   logs,
			}
			m.invalidateVisualCache()
		}
	}
	return m, nil
}

func (m Model) openDiagnosticsPicker() Model {
	if m.plugins == nil || m.FileName == "" {
		m.statusMsg = "No diagnostics"
		return m
	}
	diags := m.plugins.Diagnostics(normalizeAbsPath(m.FileName))
	if len(diags) == 0 {
		m.statusMsg = "No diagnostics in current file"
		return m
	}
	entries := make([]PickerEntry, 0, len(diags))
	for _, d := range diags {
		sevLabel := strings.ToUpper(d.Severity.String())
		detail := fmt.Sprintf("L%d:%d", d.Line+1, d.Col+1)
		if d.Source != "" {
			detail += " [" + d.Source + "]"
		}
		entries = append(entries, PickerEntry{
			ID:       fmt.Sprintf("diag:%d:%d", d.Line, d.Col),
			Category: sevLabel,
			Label:    d.Message,
			Detail:   detail,
			Path:     normalizeAbsPath(m.FileName),
			Line:     d.Line,
			Col:      d.Col,
		})
	}
	m.picker = NewPickerModal("Diagnostics", "Filter diagnostics...", entries, false, nil)
	m.pickerActive = true
	return m
}

func (m Model) jumpNextDiagnostic() Model {
	if m.plugins == nil || m.FileName == "" {
		m.statusMsg = "No diagnostics"
		return m
	}
	diags := m.plugins.Diagnostics(normalizeAbsPath(m.FileName))
	if len(diags) == 0 {
		m.statusMsg = "No diagnostics"
		return m
	}
	target := diags[0]
	for _, d := range diags {
		if d.Line > m.CursorRow || (d.Line == m.CursorRow && d.Col > m.CursorCol) {
			target = d
			break
		}
	}
	m.CursorRow = clampInt(target.Line, 0, len(m.Lines)-1)
	m.CursorCol = clampInt(target.Col, 0, lineRuneLen(m.Lines, m.CursorRow))
	m.statusMsg = fmt.Sprintf("[%s] %s", strings.ToUpper(target.Severity.String()), target.Message)
	return m.updateViewport()
}

func (m Model) showLoadedPluginsPanel() Model {
	if m.plugins == nil {
		m.statusMsg = "Plugins disabled"
		return m
	}
	list := m.plugins.Plugins()
	lines := make([]string, 0, len(list)+2)
	if len(list) == 0 {
		dir := m.plugins.PluginsDir()
		if dir == "" {
			dir = "~/.config/larry/plugins"
		}
		lines = append(lines, "No plugins installed.")
		lines = append(lines, "Expected structure: "+filepath.Join(dir, "<plugin-name>", "init.lua"))
	} else {
		for _, p := range list {
			status := "OK"
			if !p.Loaded {
				status = "ERR: " + p.Err
			}
			line := fmt.Sprintf("• %s", p.Name)
			if p.BuiltIn {
				line += " [built-in]"
			}
			if p.Version != "" {
				line += " (" + p.Version + ")"
			}
			if !p.Loaded && p.BuiltIn {
				status = "DISABLED"
			}
			line += " — " + status
			lines = append(lines, line)
			if p.Description != "" {
				lines = append(lines, "    "+p.Description)
			}
		}
	}
	m.pluginPanel = PanelState{
		Visible: true,
		Title:   "Loaded Plugins",
		Lines:   lines,
	}
	m.invalidateVisualCache()
	return m
}

// ── Built-in Colorschemes Plugin ──

func (m Model) openThemePicker() Model {
	if m.Config.Plugins.IsDisabled("colorschemes") {
		m.statusMsg = "Built-in plugin 'colorschemes' is disabled"
		return m
	}
	m.themePicker = NewThemePickerModal(m.Config.Theme)
	m.themePickerActive = true
	if hovered, ok := m.themePicker.Hovered(); ok {
		m = m.applyThemeByName(hovered.ID, false)
	}
	return m
}

func (m Model) applyThemeByName(themeName string, emitEvent bool) Model {
	SetTheme(themeName)
	m.textInput.PromptStyle = textInputStyle
	m.textInput.TextStyle = textInputStyle
	m.markdownRenderer = nil
	m.markdownCacheValid = false
	m = m.syncMarkdownForLayout()
	if emitEvent && m.plugins != nil {
		mode := "dark"
		if !activePalette.IsDark {
			mode = "light"
		}
		m.plugins.Emit(m.editorAPI(), plugin.EventThemeChange, map[string]any{
			"theme": themeName,
			"mode":  mode,
		})
	}
	return m
}

func (m Model) applyColorScheme(scheme ColorScheme, persist bool) Model {
	m.Config.Theme = scheme.ID
	m = m.applyThemeByName(scheme.ID, true)
	if persist && m.Config.ConfigPath != "" {
		if err := config.SaveTheme(m.Config.ConfigPath, scheme.ID); err != nil {
			m.statusMsg = fmt.Sprintf("Theme: %s (save failed: %v)", scheme.Name, err)
			return m
		}
		m.statusMsg = fmt.Sprintf("Theme applied & saved: %s", scheme.Name)
		return m
	}
	m.statusMsg = fmt.Sprintf("Theme applied: %s", scheme.Name)
	return m
}

// ── Filetype Detection ──

func detectFileType(filename string) string {
	if filename == "" {
		return ""
	}
	base := strings.ToLower(filepath.Base(filename))
	switch base {
	case "makefile", "gnumakefile":
		return "make"
	case "dockerfile":
		return "dockerfile"
	case "go.mod":
		return "gomod"
	case "go.sum":
		return "gosum"
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".lua":
		return "lua"
	case ".py", ".pyi":
		return "python"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".rs":
		return "rust"
	case ".md", ".markdown", ".mdown", ".mkd":
		return "markdown"
	case ".json", ".jsonc":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sh", ".bash", ".zsh":
		return "sh"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh":
		return "cpp"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss":
		return "scss"
	case ".sql":
		return "sql"
	case ".xml", ".svg":
		return "xml"
	case ".zig":
		return "zig"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}
