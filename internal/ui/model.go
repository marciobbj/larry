package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"larry/internal/config"
	"larry/internal/git"
	"larry/internal/plugin"
	"larry/internal/search"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type ViewMode int

const (
	ViewModeEditor ViewMode = iota
	ViewModeSplit
)

type fileGitDiffCache struct {
	absPath    string
	contentVer int
	lineCount  int
	base       git.FileBase
	changes    map[int]git.LineChange
}

type Model struct {
	Width              int
	Height             int
	FileName           string
	KeyMap             KeyMap
	Quitting           bool
	startRow           int
	startCol           int
	selecting          bool
	saving             bool
	goToLine           bool
	searching          bool
	replacing          bool
	globalReplacing    bool
	finding            bool
	finder             FinderModel
	globalReplace      GlobalReplaceModel
	fileTree           FileTreeModel
	textInput          textinput.Model
	statusMsg          string
	yOffset            int
	xOffset            int
	visualLineCounts   []int
	visualLinePrefix   []int
	visualLineWidth    int
	visualLineVersion  int
	visualLineComputed int
	syntaxCache        *fileSyntaxCache
	gitDiffCache       *fileGitDiffCache
	occCache           *occurrenceCache
	statusCache        *statusBarCache
	extraCursors       []CursorSelection
	Lines              []string
	CursorRow          int
	CursorCol          int
	UndoStack          []EditOp
	RedoStack          []EditOp
	Config             config.Config
	showHelp           bool
	helpScroll         int
	searchQuery        string
	replaceQuery       string
	replaceWith        string
	replaceStep        int // 1: Find, 2: Replace with, 3: Replace loop
	searchResults      []search.SearchMatch
	replaceResults     []search.SearchMatch
	currentResultIndex int
	currReplaceIndex   int
	Modified           bool
	viewMode           ViewMode
	markdownRenderer   *glamour.TermRenderer
	markdownCache      string
	markdownCacheValid bool
	showWelcome        bool
	tabs               []EditorTab
	activeTab          int
	plugins            *plugin.Manager
	pickerActive       bool
	picker             PickerModal
	themePickerActive  bool
	themePicker        ThemePickerModal
	pluginHover        HoverState
	pluginPanel        PanelState
	bufferChangeSeq    int
	lastModifiedVer    int
	lastCursorRow      int
	omarchyWatcherGen  int
}

func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown" || ext == ".mdown" || ext == ".mkd"
}

func InitialModel(filename string, lines []string, cfg config.Config) Model {
	SetTheme(cfg.Theme)

	ti := textinput.New()
	ti.Placeholder = "..."
	ti.Prompt = "Filename: "
	ti.CharLimit = 156
	ti.Width = 20
	ti.PromptStyle = textInputStyle
	ti.TextStyle = textInputStyle

	ft := NewFileTreeModel("")
	if filename != "" {
		ft.SetActiveFile(filename)
	}

	initialTab := EditorTab{
		FileName: filename,
		AbsPath:  normalizeAbsPath(filename),
		Lines:    lines,
		ViewMode: ViewModeEditor,
	}

	var pluginMgr *plugin.Manager
	startupStatus := ""
	if cfg.Plugins.IsEnabled() {
		pluginsDir := config.PluginsDir()
		pluginMgr = plugin.NewManager(cfg, pluginsDir)
		pluginMgr.RegisterBuiltIn(plugin.PluginInfo{
			Name:        "colorschemes",
			Version:     "1.0.0",
			Description: fmt.Sprintf("Interactive colorscheme switcher (%d curated themes with live preview)", len(BuiltinColorSchemes())),
			Author:      "Larry",
		})
		pluginMgr.LoadAll()
		startupStatus = pluginMgr.StartupNotice()
	} else {
		startupStatus = "Plugins disabled"
	}

	return Model{
		Width:              80,
		Height:             20,
		FileName:           filename,
		KeyMap:             NewKeyMap(cfg.LeaderKey),
		Quitting:           false,
		startRow:           0,
		startCol:           0,
		selecting:          false,
		textInput:          ti,
		statusMsg:          startupStatus,
		saving:             false,
		fileTree:           ft,
		syntaxCache:        &fileSyntaxCache{},
		gitDiffCache:       &fileGitDiffCache{},
		occCache:           &occurrenceCache{},
		statusCache:        &statusBarCache{},
		Lines:              lines,
		CursorRow:          0,
		CursorCol:          0,
		Config:             cfg,
		showHelp:           false,
		searching:          false,
		replacing:          false,
		finding:            false,
		finder:             NewFinderModel(80, 20),
		replaceResults:     nil,
		searchQuery:        "",
		searchResults:      nil,
		currentResultIndex: -1, // search
		currReplaceIndex:   -1, // replace
		Modified:           false,
		viewMode:           ViewModeEditor,
		markdownRenderer:   nil,
		showWelcome:        filename == "",
		tabs:               []EditorTab{initialTab},
		activeTab:          0,
		plugins:            pluginMgr,
	}
}

// Plugins returns the active Lua plugin manager (may be nil if plugins are disabled).
func (m *Model) Plugins() *plugin.Manager {
	return m.plugins
}

// SetPluginManager replaces the active plugin manager (useful for tests or custom setups).
func (m *Model) SetPluginManager(mgr *plugin.Manager) {
	if m.plugins != nil && m.plugins != mgr {
		m.plugins.Close()
	}
	m.plugins = mgr
}

// ClosePlugins cleanly terminates any running plugin resources (LSP servers, timers, VM).
func (m *Model) ClosePlugins() {
	if m.plugins != nil {
		m.plugins.Emit(m.editorAPI(), plugin.EventQuit, nil)
		m.plugins.Close()
	}
}

// OpenDirectory opens the collapsible lateral file tree rooted at dirPath.
func (m *Model) OpenDirectory(dirPath string) {
	m.fileTree = NewFileTreeModel(dirPath)
	m.fileTree.Visible = true
	m.fileTree.Focused = true
	m.fileTree.Height = m.workspaceHeight()
}

func (m Model) openFile(path string) (Model, error) {
	m.ensureTabsInitialized()
	absPath := normalizeAbsPath(path)

	displayPath := path
	if wd, wdErr := os.Getwd(); wdErr == nil {
		if rel, relErr := filepath.Rel(wd, absPath); relErr == nil && !strings.HasPrefix(rel, "..") {
			displayPath = rel
		}
	}

	// If already open in a tab, switch to it without losing unsaved edits or cursor position.
	if existingIdx := m.findTabByAbsPath(absPath); existingIdx >= 0 {
		m = m.switchToTab(existingIdx)
		m.statusMsg = "Switched to: " + displayPath
		return m, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		m.statusMsg = "Error opening: " + err.Error()
		return m, err
	}

	if search.IsBinaryData(content) {
		m.statusMsg = "Binary file not supported: " + filepath.Base(path)
		return m, fmt.Errorf("binary file not supported: %s", displayPath)
	}

	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	newLines := strings.Split(normalized, "\n")
	if len(newLines) == 0 {
		newLines = []string{""}
	}

	newTab := EditorTab{
		FileName: displayPath,
		AbsPath:  absPath,
		Lines:    newLines,
		ViewMode: ViewModeEditor,
	}

	if m.isEmptyInitialTab() {
		m.tabs[0] = newTab
		m.activeTab = 0
	} else {
		m.saveActiveTabState()
		m.tabs = append(m.tabs, newTab)
		m.activeTab = len(m.tabs) - 1
	}

	m.showWelcome = false
	m = m.loadTabState(m.activeTab)
	m.statusMsg = "Opened: " + displayPath
	if m.plugins != nil {
		m.plugins.Emit(m.editorAPI(), plugin.EventFileOpen, nil)
	}
	return m, nil
}

func (m Model) syncMarkdownForLayout() Model {
	if m.viewMode != ViewModeSplit && m.markdownRenderer == nil {
		return m
	}
	mainW := m.mainContentWidth()
	previewWidth := mainW - mainW/2 - 1
	if err := m.initMarkdownRenderer(previewWidth); err != nil {
		if m.viewMode == ViewModeSplit {
			m.viewMode = ViewModeEditor
			m.statusMsg = "Preview closed: " + err.Error()
		}
		m.markdownRenderer = nil
	}
	return m
}

func (m Model) finderModalDims() (innerW, innerH int) {
	outerW := m.Width - 6
	if outerW > 96 {
		outerW = 96
	}
	if outerW < 40 {
		outerW = m.Width - 2
	}
	innerW = outerW - 6 // 2 border + 4 horizontal padding
	if innerW < 34 {
		innerW = 34
	}

	outerH := m.Height - 4
	if outerH > 26 {
		outerH = 26
	}
	if outerH < 13 {
		outerH = m.Height - 2
	}
	innerH = outerH - 4 // 2 border + 2 vertical padding
	if innerH < 9 {
		innerH = 9
	}
	return innerW, innerH
}

func (m Model) openGlobalFinder() (Model, tea.Cmd) {
	m.finding = true
	innerW, innerH := m.finderModalDims()
	root := m.fileTree.RootPath
	if root == "" {
		root = "."
	}
	m.finder = NewFinderModelInRoot(root, innerW, innerH)
	return m, m.finder.performSearch()
}

func (m Model) toggleFileTree() Model {
	if m.fileTree.Visible {
		m.fileTree.Visible = false
		m.fileTree.Focused = false
	} else {
		m.fileTree.Visible = true
		m.fileTree.Focused = true
		m.fileTree.Height = m.workspaceHeight()
		m.fileTree.Rebuild()
		if m.FileName != "" {
			m.fileTree.SetActiveFile(m.FileName)
		}
	}
	m.invalidateVisualCache()
	m = m.syncMarkdownForLayout()
	m = m.updateViewport()
	return m
}

func (m Model) handleFileTreeKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.KeyMap.Quit):
		m.Quitting = true
		m.ClosePlugins()
		return m, tea.Quit

	case key.Matches(msg, m.KeyMap.CommandPalette):
		m = m.openCommandPalette()
		return m, nil

	case key.Matches(msg, m.KeyMap.ThemePicker):
		m = m.openThemePicker()
		return m, nil

	case key.Matches(msg, m.KeyMap.Open):
		m = m.toggleFileTree()
		return m, nil

	case key.Matches(msg, m.KeyMap.FocusTree):
		m.fileTree.Focused = false
		return m, nil

	case key.Matches(msg, m.KeyMap.ToggleHelp):
		m.showHelp = !m.showHelp
		m.helpScroll = 0
		return m, nil

	case key.Matches(msg, m.KeyMap.GlobalReplace):
		m = m.openGlobalReplace()
		return m, nil

	case key.Matches(msg, m.KeyMap.GlobalFinder):
		return m.openGlobalFinder()

	case key.Matches(msg, m.KeyMap.Save):
		return m.handleKey(msg)

	case key.Matches(msg, m.KeyMap.NewTab):
		m = m.newTab()
		m.fileTree.Focused = false
		return m, nil

	case key.Matches(msg, m.KeyMap.CloseTab):
		m = m.closeActiveTab()
		return m, nil

	case key.Matches(msg, m.KeyMap.NextTab):
		m = m.nextTab()
		return m, nil

	case key.Matches(msg, m.KeyMap.PrevTab):
		m = m.prevTab()
		return m, nil

	case key.Matches(msg, m.KeyMap.JumpLinesUp) || msg.Type == tea.KeyPgUp:
		m.fileTree.MoveUp(5)
		return m, nil

	case key.Matches(msg, m.KeyMap.JumpLinesDown) || msg.Type == tea.KeyPgDown:
		m.fileTree.MoveDown(5)
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEsc, tea.KeyTab:
		m.fileTree.Focused = false
		return m, nil

	case tea.KeyUp:
		m.fileTree.MoveUp(1)
		return m, nil

	case tea.KeyDown:
		m.fileTree.MoveDown(1)
		return m, nil

	case tea.KeyHome:
		m.fileTree.MoveTop()
		return m, nil

	case tea.KeyEnd:
		m.fileTree.MoveBottom()
		return m, nil

	case tea.KeyRight:
		node, ok := m.fileTree.SelectedNode()
		if !ok {
			return m, nil
		}
		if node.IsDir {
			m.fileTree.ExpandOrEnterChild()
		} else {
			var err error
			m, err = m.openFile(node.Path)
			if err == nil {
				m.fileTree.Focused = false
				m = m.updateViewport()
			}
		}
		return m, nil

	case tea.KeyLeft:
		m.fileTree.CollapseOrParent()
		return m, nil

	case tea.KeyEnter:
		node, ok := m.fileTree.SelectedNode()
		if !ok {
			return m, nil
		}
		if node.IsDir {
			m.fileTree.ToggleExpand()
		} else {
			var err error
			m, err = m.openFile(node.Path)
			if err == nil {
				m.fileTree.Focused = false
				m = m.updateViewport()
			}
		}
		return m, nil

	case tea.KeySpace:
		node, ok := m.fileTree.SelectedNode()
		if !ok {
			return m, nil
		}
		if node.IsDir {
			m.fileTree.ToggleExpand()
		} else {
			var err error
			m, err = m.openFile(node.Path)
			if err == nil {
				m = m.updateViewport()
			}
		}
		return m, nil

	case tea.KeyBackspace:
		m.fileTree.GoUpDirectory()
		return m, nil
	}

	switch msg.String() {
	case "j":
		m.fileTree.MoveDown(1)
	case "k":
		m.fileTree.MoveUp(1)
	case "l":
		node, ok := m.fileTree.SelectedNode()
		if ok {
			if node.IsDir {
				m.fileTree.ExpandOrEnterChild()
			} else {
				var err error
				m, err = m.openFile(node.Path)
				if err == nil {
					m.fileTree.Focused = false
					m = m.updateViewport()
				}
			}
		}
	case "h":
		m.fileTree.CollapseOrParent()
	case "o":
		node, ok := m.fileTree.SelectedNode()
		if ok {
			if node.IsDir {
				m.fileTree.ToggleExpand()
			} else {
				var err error
				m, err = m.openFile(node.Path)
				if err == nil {
					m = m.updateViewport()
				}
			}
		}
	case "g":
		m.fileTree.MoveTop()
	case "G":
		m.fileTree.MoveBottom()
	case ".", "H":
		m.fileTree.ToggleHidden()
		if m.fileTree.ShowHidden {
			m.statusMsg = "Showing hidden files"
		} else {
			m.statusMsg = "Hiding hidden files"
		}
	case "r":
		m.fileTree.Rebuild()
		m.statusMsg = "Tree refreshed"
	case "-":
		m.fileTree.GoUpDirectory()
	case "]", ">":
		m.fileTree.Resize(2, m.Width)
		m.invalidateVisualCache()
		m = m.syncMarkdownForLayout()
		m = m.updateViewport()
	case "[", "<":
		m.fileTree.Resize(-2, m.Width)
		m.invalidateVisualCache()
		m = m.syncMarkdownForLayout()
		m = m.updateViewport()
	case "q":
		m = m.toggleFileTree()
	}

	return m, nil
}

func (m Model) openGlobalReplace() Model {
	w := m.Width - 8
	if w > 110 {
		w = 110
	}
	if w < 40 {
		w = 40
	}
	h := m.Height - 4
	if h > 28 {
		h = 28
	}
	if h < 14 {
		h = 14
	}
	m.globalReplace = NewGlobalReplaceModel(m.fileTree.RootPath, w, h)
	m.globalReplacing = true
	return m
}

type omarchyThemeTickMsg struct {
	gen int
}

func omarchyWatchTickCmd(gen int) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return omarchyThemeTickMsg{gen: gen}
	})
}

func (m Model) gitLineChanges() map[int]git.LineChange {
	if m.FileName == "" || len(m.Lines) == 0 || m.gitDiffCache == nil {
		return nil
	}
	absPath := m.activeAbsPath()
	gc := m.gitDiffCache

	if gc.absPath != absPath || !gc.base.Loaded {
		// Reuse per-tab cached HEAD baseline if available.
		var base git.FileBase
		if m.activeTab >= 0 && m.activeTab < len(m.tabs) && m.tabs[m.activeTab].AbsPath == absPath && m.tabs[m.activeTab].GitBase.Loaded {
			base = m.tabs[m.activeTab].GitBase
		} else {
			base = git.LoadFileBase(absPath)
			if m.activeTab >= 0 && m.activeTab < len(m.tabs) && m.tabs[m.activeTab].AbsPath == absPath {
				m.tabs[m.activeTab].GitBase = base
			}
		}
		gc.absPath = absPath
		gc.base = base
		gc.contentVer = -1
		gc.lineCount = -1
	}

	if gc.contentVer != m.visualLineVersion || gc.lineCount != len(m.Lines) {
		gc.changes = git.ComputeLineChanges(gc.base, m.Lines)
		gc.contentVer = m.visualLineVersion
		gc.lineCount = len(m.Lines)
	}

	return gc.changes
}

func (m *Model) invalidateGitCache() {
	if m.gitDiffCache != nil {
		m.gitDiffCache.absPath = ""
		m.gitDiffCache.base = git.FileBase{}
		m.gitDiffCache.changes = nil
	}
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].GitBase = git.FileBase{}
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if isOmarchyThemeName(m.Config.Theme) {
		cmds = append(cmds, omarchyWatchTickCmd(m.omarchyWatcherGen))
	}
	if m.plugins != nil {
		m.plugins.Emit(m.editorAPI(), plugin.EventInit, nil)
		if m.FileName != "" {
			m.plugins.Emit(m.editorAPI(), plugin.EventFileOpen, nil)
		}
		cmds = append(cmds, waitPluginAsyncCmd(m.plugins))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if evtMsg, ok := msg.(pluginAsyncEventMsg); ok {
		if m.plugins != nil {
			m.plugins.HandleAsyncEvent(m.editorAPI(), evtMsg.evt)
			m = m.updateViewport()
			return m, waitPluginAsyncCmd(m.plugins)
		}
		return m, nil
	}

	if debMsg, ok := msg.(pluginBufferDebounceMsg); ok {
		if m.plugins != nil && debMsg.seq == m.bufferChangeSeq {
			m.plugins.Emit(m.editorAPI(), plugin.EventBufferChange, nil)
		}
		return m, nil
	}

	if tickMsg, ok := msg.(omarchyThemeTickMsg); ok {
		if isOmarchyThemeName(m.Config.Theme) && tickMsg.gen == m.omarchyWatcherGen {
			path := ResolveOmarchyColorsPath()
			sig := OmarchyThemeSignature(path)
			if sig != "" && sig != omarchySignature {
				SetTheme(m.Config.Theme)
				m.textInput.PromptStyle = textInputStyle
				m.textInput.TextStyle = textInputStyle
				m.markdownRenderer = nil
				m.markdownCacheValid = false
				m = m.syncMarkdownForLayout()
				if m.plugins != nil {
					m.plugins.Emit(m.editorAPI(), plugin.EventThemeChange, map[string]any{"theme": m.Config.Theme})
				}
			}
			return m, omarchyWatchTickCmd(m.omarchyWatcherGen)
		}
		return m, nil
	}

	if m.themePickerActive {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.Width = ws.Width
			m.Height = ws.Height
			m.fileTree.Height = m.workspaceHeight()
			return m, nil
		}
		var cmd tea.Cmd
		var chosen *ColorScheme
		var closed, hoveredChanged bool
		m.themePicker, cmd, chosen, closed, hoveredChanged = m.themePicker.Update(msg)
		if closed {
			m.themePickerActive = false
			if chosen != nil {
				m = m.applyColorScheme(*chosen, true)
				if isOmarchyThemeName(m.Config.Theme) {
					m.omarchyWatcherGen++
					return m, omarchyWatchTickCmd(m.omarchyWatcherGen)
				}
			} else {
				// Revert live preview back to the original theme.
				m = m.applyThemeByName(m.themePicker.OriginalTheme, false)
			}
			return m, nil
		}
		if hoveredChanged {
			if hovered, ok := m.themePicker.Hovered(); ok {
				m = m.applyThemeByName(hovered.ID, false)
			}
		}
		return m, cmd
	}

	if m.pickerActive {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.Width = ws.Width
			m.Height = ws.Height
			m.fileTree.Height = m.workspaceHeight()
			return m, nil
		}
		var cmd tea.Cmd
		var chosen *PickerEntry
		var closed bool
		m.picker, cmd, chosen, closed = m.picker.Update(msg)
		if closed {
			m.pickerActive = false
			if chosen != nil {
				if m.picker.IsPalette {
					return m.executePaletteEntry(*chosen)
				}
				if m.picker.OnSelect != nil {
					m.picker.OnSelect(*chosen)
					return m.updateViewport(), nil
				}
				if chosen.Path != "" {
					if updated, err := m.openFile(chosen.Path); err == nil {
						m = updated
						m.CursorRow = clampInt(chosen.Line, 0, len(m.Lines)-1)
						m.CursorCol = clampInt(chosen.Col, 0, lineRuneLen(m.Lines, m.CursorRow))
						m.fileTree.Focused = false
						m = m.updateViewport()
					}
				}
			}
			return m, nil
		}
		return m, cmd
	}

	if m.pluginHover.Visible {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.Type {
			case tea.KeyEsc:
				m.pluginHover.Visible = false
				return m, nil
			case tea.KeyUp:
				if m.pluginHover.Scroll > 0 {
					m.pluginHover.Scroll--
				}
				return m, nil
			case tea.KeyDown:
				m.pluginHover.Scroll++
				return m, nil
			default:
				if km.String() == "k" {
					if m.pluginHover.Scroll > 0 {
						m.pluginHover.Scroll--
					}
					return m, nil
				}
				if km.String() == "j" {
					m.pluginHover.Scroll++
					return m, nil
				}
				if km.String() == "q" {
					m.pluginHover.Visible = false
					return m, nil
				}
				// Any other key dismisses hover and proceeds normally.
				m.pluginHover.Visible = false
			}
		}
	}

	if m.globalReplacing {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.Width = ws.Width
			m.Height = ws.Height
			m.fileTree.Height = m.editorViewportHeight()
			grW := ws.Width - 8
			if grW > 110 {
				grW = 110
			}
			if grW < 40 {
				grW = 40
			}
			grH := ws.Height - 4
			if grH > 28 {
				grH = 28
			}
			if grH < 14 {
				grH = 14
			}
			m.globalReplace, _, _, _ = m.globalReplace.Update(tea.WindowSizeMsg{Width: grW, Height: grH})
			return m, nil
		}

		var cmd tea.Cmd
		var executed, closed bool
		m.globalReplace, cmd, executed, closed = m.globalReplace.Update(msg)
		if closed {
			m.globalReplacing = false
			return m, nil
		}
		if executed {
			res, err := m.globalReplace.Execute()
			m.globalReplacing = false
			if err != nil {
				m.statusMsg = "Global replace error: " + err.Error()
			} else if len(res.Errors) > 0 {
				m.statusMsg = fmt.Sprintf("Replaced %d in %d files (%d errors)", res.TotalReplacements, res.FilesModified, len(res.Errors))
			} else {
				m.statusMsg = fmt.Sprintf("Replaced %d occurrences in %d files", res.TotalReplacements, res.FilesModified)
			}
			m = m.reloadModifiedTabs(res.ModifiedFilePaths)
			m.fileTree.Rebuild()
			return m, nil
		}
		return m, cmd
	}

	if m.finding {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height
			m.fileTree.Height = m.workspaceHeight()
			innerW, innerH := m.finderModalDims()
			var cmd tea.Cmd
			m.finder, cmd = m.finder.Update(tea.WindowSizeMsg{Width: innerW, Height: innerH})
			return m, cmd
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.finder.Cancel()
				m.finding = false
				return m, nil
			case "enter":
				if len(m.finder.results) > 0 {
					res := m.finder.results[m.finder.cursor]
					var path string
					var targetRow int
					if res.Mode == search.ModeFiles {
						path = res.File.Path
						targetRow = 0
					} else {
						path = res.Grep.Path
						targetRow = res.Grep.Line - 1
					}

					var err error
					m, err = m.openFile(path)
					if err == nil {
						m.CursorRow = targetRow
						m.CursorCol = 0
						m = m.updateViewport()
						m.fileTree.Focused = false
					}
					m.finder.Cancel()
					m.finding = false
					return m, nil
				}
			}
		}
		var cmd tea.Cmd
		m.finder, cmd = m.finder.Update(msg)
		return m, cmd
	}

	if m.saving {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.saving = false
				return m, nil
			case tea.KeyEnter:
				filename := m.textInput.Value()
				if filename == "" {
					filename = "untitled.txt"
				}
				m, _ = m.saveCurrentFile(filename)
				m.fileTree.Rebuild()
				m.saving = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if m.goToLine {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.goToLine = false
				return m, nil
			case tea.KeyEnter:
				lineStr := m.textInput.Value()
				var targetLine int
				_, err := fmt.Sscanf(lineStr, "%d", &targetLine)
				if err == nil {
					targetLine--
					if targetLine < 0 {
						targetLine = 0
					}
					if targetLine >= len(m.Lines) {
						targetLine = len(m.Lines) - 1
					}
					if targetLine < 0 {
						targetLine = 0
					}
					m.CursorRow = targetLine
					m.CursorCol = 0
					m = m.updateViewport()
				}
				m.goToLine = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if m.replacing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.replacing = false
				m.replaceStep = 0
				m.replaceQuery = ""
				m.replaceWith = ""
				m.replaceResults = nil
				return m, nil
			case tea.KeyEnter:
				if m.replaceStep == 1 {
					m.replaceQuery = m.textInput.Value()
					if m.replaceQuery == "" {
						m.replacing = false
						return m, nil
					}
					m.replaceStep = 2
					m.textInput.SetValue("")
					m.textInput.Prompt = "With: "
					return m, nil
				} else if m.replaceStep == 2 {
					m.replaceWith = m.textInput.Value()
					searcher := search.NewBoyerMooreSearch(m.replaceQuery)
					m.replaceResults = searcher.SearchInLines(context.Background(), m.Lines)
					m.currReplaceIndex = -1
					if len(m.replaceResults) > 0 {
						m.replaceStep = 3
						m.currReplaceIndex = 0
						result := m.replaceResults[m.currReplaceIndex]
						m.CursorRow = result.Line
						m.CursorCol = result.Col
						m = m.updateViewport()
					} else {
						m.statusMsg = "No matches found"
						m.replacing = false
					}
					return m, nil
				} else if m.replaceStep == 3 {
					if m.currReplaceIndex >= 0 && m.currReplaceIndex < len(m.replaceResults) {
						match := m.replaceResults[m.currReplaceIndex]

						m.startRow, m.startCol = match.Line, match.Col
						m.CursorRow, m.CursorCol = match.Line, match.Col+match.Length
						m.selecting = true

						m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: m.replaceQuery})
						m = m.deleteSelectedText()
						m.pushUndo(EditOp{Type: OpInsert, Row: m.startRow, Col: m.startCol, Text: m.replaceWith})
						m = m.insertTextAtCursor(m.replaceWith)

						searcher := search.NewBoyerMooreSearch(m.replaceQuery)
						m.replaceResults = searcher.SearchInLines(context.Background(), m.Lines)

						if len(m.replaceResults) > 0 {
							if m.currReplaceIndex >= len(m.replaceResults) {
								m.currReplaceIndex = 0
							}
							result := m.replaceResults[m.currReplaceIndex]
							m.CursorRow = result.Line
							m.CursorCol = result.Col
							m = m.updateViewport()
						} else {
							m.statusMsg = "Done replacing"
							m.replacing = false
						}
					}
					return m, nil
				}
			}
		}

		if m.replaceStep == 1 {
			query := m.textInput.Value()
			if query != m.replaceQuery {
				m.replaceQuery = query
				if query != "" {
					searcher := search.NewBoyerMooreSearch(query)
					m.replaceResults = searcher.SearchInLines(context.Background(), m.Lines)
				} else {
					m.replaceResults = nil
				}
				m.currReplaceIndex = -1
			}
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if m.searching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.searching = false
				m.searchQuery = ""
				m.searchResults = nil
				m.currentResultIndex = -1
				return m, nil
			case tea.KeyEnter:
				if len(m.searchResults) > 0 {
					m.currentResultIndex = (m.currentResultIndex + 1) % len(m.searchResults)
					result := m.searchResults[m.currentResultIndex]
					m.CursorRow = result.Line
					m.CursorCol = result.Col

					// Center the result in the viewport
					viewportHeight := m.editorViewportHeight()
					if m.Config.WordWrap {
						m.yOffset = m.cursorVisualAbsPos(m.editorTextWidth()) - (viewportHeight / 2)
					} else {
						m.yOffset = m.CursorRow - (viewportHeight / 2)
					}
					if m.yOffset < 0 {
						m.yOffset = 0
					}
					m = m.updateViewport()
				}
				return m, nil
			}
		}

		query := m.textInput.Value()
		if query != m.searchQuery {
			m.searchQuery = query
			if query != "" {
				searcher := search.NewBoyerMooreSearch(query)
				m.searchResults = searcher.SearchInLines(context.Background(), m.Lines)
			} else {
				m.searchResults = nil
			}
			m.currentResultIndex = -1
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if m.showHelp {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case keyMsg.Type == tea.KeyEsc || key.Matches(keyMsg, m.KeyMap.ToggleHelp) || keyMsg.String() == "q":
				m.showHelp = false
				m.helpScroll = 0
			case keyMsg.String() == "up" || keyMsg.String() == "k":
				if m.helpScroll > 0 {
					m.helpScroll--
				}
			case keyMsg.String() == "down" || keyMsg.String() == "j":
				m.helpScroll++
			case keyMsg.String() == "pgup":
				m.helpScroll -= 6
				if m.helpScroll < 0 {
					m.helpScroll = 0
				}
			case keyMsg.String() == "pgdown":
				m.helpScroll += 6
			case keyMsg.String() == "home" || keyMsg.String() == "g":
				m.helpScroll = 0
			case keyMsg.String() == "end" || keyMsg.String() == "G":
				m.helpScroll = 999
			}
			return m, nil
		}
		if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
			m.Width = sizeMsg.Width
			m.Height = sizeMsg.Height
			m.fileTree.Height = m.workspaceHeight()
			m = m.syncMarkdownForLayout()
			m = m.updateViewport()
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case globalReplacePreviewMsg:
		return m, nil

	case searchMsg:
		var cmd tea.Cmd
		m.finder, cmd = m.finder.Update(msg)
		return m, cmd

	case SearchResultsMsg:
		if msg.IsReplace {
			// Verify if the query matches current replaceQuery
			// (User might have typed more since this search started)
			if msg.Query != m.replaceQuery {
				return m, nil
			}
			m.replaceResults = msg.Results
			m.currReplaceIndex = -1
			if len(m.replaceResults) > 0 {
				m.replaceStep = 3
				m.currReplaceIndex = 0
				result := m.replaceResults[m.currReplaceIndex]
				m.CursorRow = result.Line
				m.CursorCol = result.Col
				m = m.updateViewport()
			} else {
				// Only finish if we are in step 2 (waiting for search)
				if m.replaceStep == 2 {
					m.statusMsg = "No matches found"
					// Stay in replace mode or exit?
					// m.replacing = false // Maybe let them try another query
				}
			}
		} else {
			if msg.Query != m.searchQuery {
				return m, nil
			}
			m.searchResults = msg.Results
			m.currentResultIndex = -1
		}
		return m, nil

	case tea.KeyMsg:
		// Clear transient status messages on any key press
		m.statusMsg = ""

		if m.fileTree.Visible && m.fileTree.Focused {
			return m.handleFileTreeKey(msg)
		}

		// Dismiss the welcome screen on editor key presses (keep it alongside the sidebar until a file is opened or edited)
		if m.showWelcome &&
			!key.Matches(msg, m.KeyMap.Open) &&
			!key.Matches(msg, m.KeyMap.FocusTree) &&
			!key.Matches(msg, m.KeyMap.ToggleHelp) &&
			!key.Matches(msg, m.KeyMap.GlobalFinder) &&
			!key.Matches(msg, m.KeyMap.GlobalReplace) &&
			!key.Matches(msg, m.KeyMap.CommandPalette) &&
			!key.Matches(msg, m.KeyMap.ThemePicker) &&
			!key.Matches(msg, m.KeyMap.Quit) {
			m.showWelcome = false
		}

		prevVer := m.visualLineVersion
		prevRow := m.CursorRow

		var cmd tea.Cmd
		m, cmd = m.handleKey(msg)

		var extraCmds []tea.Cmd
		if cmd != nil {
			extraCmds = append(extraCmds, cmd)
		}
		if m.plugins != nil && !m.Quitting {
			if m.visualLineVersion != prevVer {
				m.bufferChangeSeq++
				extraCmds = append(extraCmds, bufferChangeDebounceCmd(m.bufferChangeSeq))
			}
			if m.CursorRow != prevRow {
				m.lastCursorRow = m.CursorRow
				if m.plugins.HasHandlers(plugin.EventCursorMove) {
					m.plugins.Emit(m.editorAPI(), plugin.EventCursorMove, nil)
				}
			}
		}

		m = m.updateViewport()
		if len(extraCmds) == 0 {
			return m, nil
		}
		if len(extraCmds) == 1 {
			return m, extraCmds[0]
		}
		return m, tea.Batch(extraCmds...)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.fileTree.Height = m.workspaceHeight()

		// Always update markdown renderer if it exists or if we are in split mode
		m = m.syncMarkdownForLayout()

		innerW, innerH := m.finderModalDims()
		var finderCmd tea.Cmd
		m.finder, finderCmd = m.finder.Update(tea.WindowSizeMsg{Width: innerW, Height: innerH})

		m = m.updateViewport()

		return m, finderCmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return "Tchau!\n"
	}

	// Fast path for full-screen modals: skip rendering the editor, syntax
	// highlighting, git diff, file tree, and status bar when they are hidden
	// behind a centered modal overlay.
	if m.finding {
		innerW, innerH := m.finderModalDims()
		body := m.finder.Render(innerW, innerH)
		modal := modalStyle.Render(body)

		return placeCentered(m.Width, m.Height, modal)
	}

	if m.globalReplacing {
		w := m.Width - 10
		if w > 100 {
			w = 100
		}
		if w < 36 {
			w = 36
		}
		maxH := m.Height - 4
		if maxH < 10 {
			maxH = 10
		}

		bg := modalStyle.GetBackground()
		spacerStyle := lipgloss.NewStyle().Background(bg)

		titleText := "Global Replace"
		title := modalTitleStyle.Width(w).Render(titleText)

		body := m.globalReplace.View(w)
		var allLines []string
		allLines = append(allLines, title)
		allLines = append(allLines, spacerStyle.Width(w).Render(""))

		for i, line := range strings.Split(body, "\n") {
			if i >= maxH-2 {
				break
			}
			allLines = append(allLines, line)
		}

		modal := modalStyle.Render(strings.Join(allLines, "\n"))

		return placeCentered(m.Width, m.Height, modal)
	}

	if m.themePickerActive {
		return m.themePicker.Render(m.Width, m.Height)
	}

	if m.pickerActive {
		return m.picker.Render(m.Width, m.Height)
	}

	if m.pluginHover.Visible {
		return m.pluginHover.Render(m.Width, m.Height)
	}

	if m.showHelp {
		return m.viewHelpMenu("")
	}

	status := m.renderStatusBar()
	statusHeight := strings.Count(status, "\n") + 1
	if statusHeight < 1 {
		statusHeight = 1
	}

	hasPrompt := m.saving || m.goToLine || m.searching || m.replacing
	promptHeight := 0
	if hasPrompt {
		promptHeight = 2 // 1 divider rule + 1 full-width styled prompt bar
	}

	wsHeight := m.Height - statusHeight - promptHeight
	if wsHeight < 1 {
		wsHeight = 1
	}
	editorHeight := wsHeight - m.tabBarHeight()
	if editorHeight < 1 {
		editorHeight = 1
	}
	treeW := m.sidebarWidth()
	mainW := m.mainContentWidth()

	mainView := ""

	if m.showWelcome {
		mainView = m.viewWelcome()
	} else {
		var editorArea string
		if m.pluginPanel.Visible {
			editorArea = m.viewWithPluginPanel()
		} else if m.viewMode == ViewModeSplit && isMarkdownFile(m.FileName) {
			editorArea = m.viewSplit()
		} else if len(m.Lines) == 0 {
			uiANSI := getCompiledUIANSI()
			s := strings.Builder{}
			if treeW == 0 {
				s.WriteString(uiANSI.borderStr)
			}
			if m.Config.LineNumbers {
				s.WriteString(uiANSI.lineNum.wrap("   1 "))
			}
			s.WriteString(uiANSI.cursorSpace)
			s.WriteString("\n")
			editorArea = s.String()
		} else {
			editorArea = m.viewEditor(editorViewConfig{
				width:          mainW,
				height:         editorHeight,
				showSearchUI:   false,
				hideLeftBorder: treeW > 0,
			})
		}

		if m.showTabBar() {
			tabBar := m.renderTabBar(mainW)
			mainView = tabBar + "\n" + editorArea
		} else {
			mainView = editorArea
		}
	}

	uiANSI := getCompiledUIANSI()
	baseView := mainView
	if treeW > 0 {
		treeView := m.fileTree.View(treeW, wsHeight)
		divider := m.fileTree.DividerView(wsHeight)
		baseView = joinSidebarAndMain(treeView, divider, mainView, treeW, mainW, wsHeight, uiANSI)
	}

	// Pad baseView so the bottom prompt bar (if active) and status bar sit flush at the bottom.
	baseHeight := strings.Count(baseView, "\n") + 1
	totalUsed := baseHeight + promptHeight + statusHeight
	if totalUsed < m.Height {
		missing := m.Height - totalUsed
		if uiANSI.paintBg && m.Width > 0 {
			blankRow := uiANSI.canvasBg.wrap(strings.Repeat(" ", m.Width)) + uiANSI.eolClear
			var pad strings.Builder
			pad.Grow(missing * (len(blankRow) + 1))
			for i := 0; i < missing; i++ {
				pad.WriteByte('\n')
				pad.WriteString(blankRow)
			}
			baseView += pad.String()
		} else {
			baseView += strings.Repeat("\n", missing)
		}
	}

	if hasPrompt {
		return baseView + "\n" + m.renderPromptBar() + "\n" + status
	}

	return baseView + "\n" + status
}

func (m *Model) renderPromptBar() string {
	width := m.Width
	if width < 20 {
		width = 20
	}

	p := activePalette
	barBg := p.Surface

	rule := withCanvasBg(lipgloss.NewStyle().Foreground(p.Border)).Render(strings.Repeat("─", width))

	// Style a value copy of textInput so prompt, text, and placeholder blend cleanly into barBg.
	ti := m.textInput
	ti.PromptStyle = lipgloss.NewStyle().Foreground(p.Accent).Background(barBg).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(p.Foreground).Background(barBg)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(p.MutedFg).Background(barBg)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(p.AccentFg).Background(p.Accent)
	ti.Cursor.TextStyle = lipgloss.NewStyle().Foreground(p.Foreground).Background(barBg)

	var counter, hint string
	switch {
	case m.searching:
		if len(m.searchResults) > 0 {
			counter = fmt.Sprintf(" (%d/%d)", m.currentResultIndex+1, len(m.searchResults))
		} else if m.searchQuery != "" {
			counter = " (no results)"
		}
		hint = "Enter Next · Esc Close"
	case m.replacing:
		if len(m.replaceResults) > 0 {
			counter = fmt.Sprintf(" (%d/%d)", m.currReplaceIndex+1, len(m.replaceResults))
		} else if m.replaceQuery != "" && m.replaceStep >= 2 {
			counter = " (no results)"
		}
		if m.replaceStep == 3 {
			hint = "Enter Replace · Esc Done"
		} else {
			hint = "Enter Confirm · Esc Cancel"
		}
	case m.goToLine:
		hint = "Enter Jump · Esc Cancel"
	case m.saving:
		hint = "Enter Save · Esc Cancel"
	}

	counterStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(barBg).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(p.MutedFg).Background(barBg)
	spacerStyle := lipgloss.NewStyle().Background(barBg)

	hintRendered := ""
	hintW := 0
	if width >= 50 && hint != "" {
		hintRendered = hintStyle.Render(hint + " ")
		hintW = lipgloss.Width(hintRendered)
	}

	counterRendered := ""
	counterW := 0
	if counter != "" {
		counterRendered = counterStyle.Render(counter)
		counterW = lipgloss.Width(counterRendered)
	}

	// Give textInput enough width while keeping everything on a single row.
	promptW := lipgloss.Width(ti.Prompt)
	maxInputW := width - promptW - counterW - hintW - 4
	if maxInputW < 8 {
		maxInputW = 8
	}
	ti.Width = maxInputW

	left := spacerStyle.Render(" ") + ti.View() + counterRendered
	leftW := lipgloss.Width(left)
	gap := width - leftW - hintW
	if gap < 0 {
		gap = 0
		hintRendered = ""
	}

	promptRow := lipgloss.NewStyle().Background(barBg).Width(width).Render(
		left + spacerStyle.Render(strings.Repeat(" ", gap)) + hintRendered,
	)

	return rule + "\n" + promptRow
}
