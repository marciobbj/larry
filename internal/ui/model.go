package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"larry/internal/config"
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
	Lines              []string
	CursorRow          int
	CursorCol          int
	UndoStack          []EditOp
	RedoStack          []EditOp
	Config             config.Config
	showHelp           bool
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
		saving:             false,
		fileTree:           ft,
		syntaxCache:        &fileSyntaxCache{},
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
	}
}

// OpenDirectory opens the collapsible lateral file tree rooted at dirPath.
func (m *Model) OpenDirectory(dirPath string) {
	m.fileTree = NewFileTreeModel(dirPath)
	m.fileTree.Visible = true
	m.fileTree.Focused = true
	m.fileTree.Height = m.editorViewportHeight()
}

func (m Model) openFile(path string) (Model, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		m.statusMsg = "Error opening: " + err.Error()
		return m, err
	}

	displayPath := path
	if wd, wdErr := os.Getwd(); wdErr == nil {
		if abs, absErr := filepath.Abs(path); absErr == nil {
			if rel, relErr := filepath.Rel(wd, abs); relErr == nil && !strings.HasPrefix(rel, "..") {
				displayPath = rel
			}
		}
	}

	m.Lines = strings.Split(string(content), "\n")
	m.FileName = displayPath
	m.CursorRow = 0
	m.CursorCol = 0
	m.yOffset = 0
	m.xOffset = 0
	m.UndoStack = nil
	m.RedoStack = nil
	m.selecting = false
	m.Modified = false
	m.showWelcome = false
	m.viewMode = ViewModeEditor
	m.markdownRenderer = nil
	m.markdownCache = ""
	m.markdownCacheValid = false
	m.searchResults = nil
	m.replaceResults = nil
	m.currentResultIndex = -1
	m.currReplaceIndex = -1
	m.invalidateVisualCache()
	m.fileTree.SetActiveFile(path)
	m.statusMsg = "Opened: " + displayPath
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

func (m Model) toggleFileTree() Model {
	if m.fileTree.Visible {
		m.fileTree.Visible = false
		m.fileTree.Focused = false
	} else {
		m.fileTree.Visible = true
		m.fileTree.Focused = true
		m.fileTree.Height = m.editorViewportHeight()
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
		return m, tea.Quit

	case key.Matches(msg, m.KeyMap.Open):
		m = m.toggleFileTree()
		return m, nil

	case key.Matches(msg, m.KeyMap.FocusTree):
		m.fileTree.Focused = false
		return m, nil

	case key.Matches(msg, m.KeyMap.ToggleHelp):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.KeyMap.GlobalReplace):
		m = m.openGlobalReplace()
		return m, nil

	case key.Matches(msg, m.KeyMap.GlobalFinder):
		m.finding = true
		finderWidth := m.Width
		if finderWidth > 120 {
			finderWidth = 120
		}
		finderHeight := m.Height
		if finderHeight > 25 {
			finderHeight = 25
		}
		m.finder = NewFinderModel(finderWidth, finderHeight)
		return m, m.finder.performSearch()

	case key.Matches(msg, m.KeyMap.Save):
		return m.handleKey(msg)

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

func (m Model) reloadActiveFileIfModified(modifiedPaths []string) Model {
	if m.FileName == "" || len(modifiedPaths) == 0 {
		return m
	}
	absCurrent, err := filepath.Abs(m.FileName)
	if err != nil {
		absCurrent = filepath.Clean(m.FileName)
	} else {
		absCurrent = filepath.Clean(absCurrent)
	}

	matched := false
	for _, p := range modifiedPaths {
		if filepath.Clean(p) == absCurrent {
			matched = true
			break
		}
	}
	if !matched {
		return m
	}

	raw, err := os.ReadFile(absCurrent)
	if err != nil {
		return m
	}
	m.Lines = strings.Split(string(raw), "\n")
	if len(m.Lines) == 0 {
		m.Lines = []string{""}
	}
	if m.CursorRow >= len(m.Lines) {
		m.CursorRow = len(m.Lines) - 1
	}
	if m.CursorRow < 0 {
		m.CursorRow = 0
	}
	lineLen := len([]rune(m.Lines[m.CursorRow]))
	if m.CursorCol > lineLen {
		m.CursorCol = lineLen
	}
	m.UndoStack = nil
	m.RedoStack = nil
	m.selecting = false
	m.Modified = false
	m.markdownCacheValid = false
	m.invalidateVisualCache()
	return m.updateViewport()
}

type omarchyThemeTickMsg struct{}

func omarchyWatchTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return omarchyThemeTickMsg{}
	})
}

func (m Model) Init() tea.Cmd {
	if isOmarchyThemeName(m.Config.Theme) {
		return omarchyWatchTickCmd()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(omarchyThemeTickMsg); ok {
		if isOmarchyThemeName(m.Config.Theme) {
			path := ResolveOmarchyColorsPath()
			sig := OmarchyThemeSignature(path)
			if sig != "" && sig != omarchySignature {
				SetTheme(m.Config.Theme)
				m.textInput.PromptStyle = textInputStyle
				m.textInput.TextStyle = textInputStyle
				m.markdownRenderer = nil
				m.markdownCacheValid = false
				m = m.syncMarkdownForLayout()
			}
			return m, omarchyWatchTickCmd()
		}
		return m, nil
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
			m = m.reloadActiveFileIfModified(res.ModifiedFilePaths)
			m.fileTree.Rebuild()
			return m, nil
		}
		return m, cmd
	}

	if m.finding {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
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
				content := strings.Join(m.Lines, "\n")
				err := os.WriteFile(filename, []byte(content), 0644)
				if err != nil {
					m.statusMsg = "Error saving: " + err.Error()
				} else {
					m.statusMsg = "Saved: " + filename
					m.FileName = filename
					m.Modified = false
					m.fileTree.Rebuild()
					m.fileTree.SetActiveFile(filename)
				}
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
			if keyMsg.Type == tea.KeyEsc || key.Matches(keyMsg, m.KeyMap.ToggleHelp) {
				m.showHelp = false
			}
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
			!key.Matches(msg, m.KeyMap.Quit) {
			m.showWelcome = false
		}

		var cmd tea.Cmd
		m, cmd = m.handleKey(msg)
		if cmd != nil {
			return m, cmd
		}
		m = m.updateViewport()
		return m, cmd

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.fileTree.Height = m.editorViewportHeight()

		// Always update markdown renderer if it exists or if we are in split mode
		m = m.syncMarkdownForLayout()

		finderWidth := msg.Width
		if finderWidth > 120 {
			finderWidth = 120
		}
		finderHeight := msg.Height
		if finderHeight > 25 {
			finderHeight = 25
		}

		var finderCmd tea.Cmd
		m.finder, finderCmd = m.finder.Update(tea.WindowSizeMsg{Width: finderWidth, Height: finderHeight})

		m = m.updateViewport()

		return m, finderCmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return "Tchau!\n"
	}

	status := m.renderStatusBar()

	viewportHeight := m.editorViewportHeight()
	treeW := m.sidebarWidth()
	mainW := m.mainContentWidth()

	mainView := ""

	if m.showWelcome {
		mainView = m.viewWelcome()
	} else if m.viewMode == ViewModeSplit && isMarkdownFile(m.FileName) {
		mainView = m.viewSplit()
	} else if len(m.Lines) == 0 {
		s := strings.Builder{}
		if treeW == 0 {
			s.WriteString(borderStyle.Render("│"))
		}
		if m.Config.LineNumbers {
			s.WriteString(lineNumStyle.Render("   1 "))
		}
		s.WriteString(styleCursor.Render(" "))
		s.WriteString("\n")
		mainView = s.String()
	} else {
		mainView = m.viewEditor(editorViewConfig{
			width:          mainW,
			height:         viewportHeight,
			showSearchUI:   m.searching,
			hideLeftBorder: treeW > 0,
		})
	}

	baseView := mainView
	if treeW > 0 {
		treeView := m.fileTree.View(treeW, viewportHeight)
		divider := m.fileTree.DividerView(viewportHeight)
		baseView = lipgloss.JoinHorizontal(lipgloss.Top, treeView, divider, mainView)
	}

	if m.saving {
		return fmt.Sprintf("%s\n\n%s", baseView, m.textInput.View())
	}
	if m.goToLine {
		return fmt.Sprintf("%s\n\n%s", baseView, m.textInput.View())
	}
	if m.searching {
		var counter string
		if len(m.searchResults) > 0 {
			counter = fmt.Sprintf(" (%d/%d)", m.currentResultIndex+1, len(m.searchResults))
		} else if m.searchQuery != "" {
			counter = " (no results)"
		}
		searchView := fmt.Sprintf("%s%s", m.textInput.View(), counter)
		return fmt.Sprintf("%s\n\n%s", baseView, searchView)
	}
	if m.replacing {
		var counter string
		if len(m.replaceResults) > 0 {
			counter = fmt.Sprintf(" (%d/%d)", m.currReplaceIndex+1, len(m.replaceResults))
		} else if m.replaceQuery != "" && m.replaceStep >= 2 {
			counter = " (no results)"
		}
		replaceView := fmt.Sprintf("%s%s", m.textInput.View(), counter)
		return fmt.Sprintf("%s\n\n%s", baseView, replaceView)
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
			allLines = append(allLines, spacerStyle.Width(w).Render(line))
		}

		modal := modalStyle.Render(strings.Join(allLines, "\n"))

		return lipgloss.Place(
			m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			modal,
		)
	}

	if m.finding {
		finderView := m.finder.View()
		w := m.Width
		if w > 120 {
			w = 120
		}
		h := lipgloss.Height(finderView)

		maxW := m.Width - 8
		maxH := m.Height - 4

		if w > maxW {
			w = maxW
		}
		if h > maxH {
			h = maxH
		}

		bg := modalStyle.GetBackground()
		spacerStyle := lipgloss.NewStyle().Background(bg)

		titleText := "Global Larry Finder (Tab: Switch Mode)"
		titleGap := w - lipgloss.Width(titleText)
		if titleGap < 0 {
			titleGap = 0
		}
		leftGap := titleGap / 2
		rightGap := titleGap - leftGap
		title := modalTitleStyle.Width(w).Render(strings.Repeat(" ", leftGap) + titleText + strings.Repeat(" ", rightGap))

		var allLines []string
		allLines = append(allLines, title)
		allLines = append(allLines, spacerStyle.Copy().Width(w).Render(""))

		finderLines := strings.Split(finderView, "\n")
		for i, line := range finderLines {
			if i >= maxH-2 {
				break
			}
			styledLine := strings.ReplaceAll(line, " ", spacerStyle.Render(" "))
			allLines = append(allLines, spacerStyle.Width(w).Render(styledLine))
		}

		modal := modalStyle.Render(strings.Join(allLines, "\n"))

		return lipgloss.Place(
			m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			modal,
		)
	}

	if m.showHelp {
		return m.viewHelpMenu(baseView)
	}

	// Pad baseView so the status bar is always at the bottom of the terminal
	baseHeight := lipgloss.Height(baseView)
	statusHeight := lipgloss.Height(status)
	totalUsed := baseHeight + statusHeight
	if totalUsed < m.Height {
		padding := strings.Repeat("\n", m.Height-totalUsed)
		baseView = baseView + padding
	}

	return lipgloss.JoinVertical(lipgloss.Left, baseView, status)
}
