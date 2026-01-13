package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"larry/internal/config"
	"larry/internal/search"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KeyMap struct {
	Quit               key.Binding
	SelectAll          key.Binding
	MoveSelectionDown  key.Binding
	MoveSelectionUp    key.Binding
	MoveSelectionLeft  key.Binding
	MoveSelectionRight key.Binding
	Copy               key.Binding
	Paste              key.Binding
	Cut                key.Binding
	Save               key.Binding
	Open               key.Binding
	CursorUp           key.Binding
	CursorDown         key.Binding
	CursorLeft         key.Binding
	CursorRight        key.Binding
	Delete             key.Binding
	Undo               key.Binding
	Redo               key.Binding
	GoToLine           key.Binding
	ToggleHelp         key.Binding
	Search             key.Binding
	// Agile Navigation
	JumpWordLeft      key.Binding
	JumpWordRight     key.Binding
	JumpLinesUp       key.Binding
	JumpLinesDown     key.Binding
	SelectWordLeft    key.Binding
	SelectWordRight   key.Binding
	SelectLinesUp     key.Binding
	SelectLinesDown   key.Binding
	LineStart         key.Binding
	LineEnd           key.Binding
	FileStart         key.Binding
	FileEnd           key.Binding
	SelectToLineStart key.Binding
	SelectToLineEnd   key.Binding
}

var DefaultKeyMap = KeyMap{
	Quit:               key.NewBinding(key.WithKeys("ctrl+q")),
	SelectAll:          key.NewBinding(key.WithKeys("ctrl+a")),
	MoveSelectionDown:  key.NewBinding(key.WithKeys("shift+down")),
	MoveSelectionLeft:  key.NewBinding(key.WithKeys("shift+left")),
	MoveSelectionRight: key.NewBinding(key.WithKeys("shift+right")),
	MoveSelectionUp:    key.NewBinding(key.WithKeys("shift+up")),
	Copy:               key.NewBinding(key.WithKeys("ctrl+c")),
	Paste:              key.NewBinding(key.WithKeys("ctrl+v")),
	Cut:                key.NewBinding(key.WithKeys("ctrl+x")),
	Save:               key.NewBinding(key.WithKeys("ctrl+s")),
	Open:               key.NewBinding(key.WithKeys("ctrl+o")),
	CursorUp:           key.NewBinding(key.WithKeys("up")),
	CursorDown:         key.NewBinding(key.WithKeys("down")),
	CursorLeft:         key.NewBinding(key.WithKeys("left")),
	CursorRight:        key.NewBinding(key.WithKeys("right")),
	Delete:             key.NewBinding(key.WithKeys("backspace", "delete")),
	Undo:               key.NewBinding(key.WithKeys("ctrl+z")),
	Redo:               key.NewBinding(key.WithKeys("ctrl+shift+z", "ctrl+r")),
	GoToLine:           key.NewBinding(key.WithKeys("ctrl+g")),
	ToggleHelp:         key.NewBinding(key.WithKeys("ctrl+h")),
	Search:             key.NewBinding(key.WithKeys("ctrl+f")),
	// Agile Navigation
	JumpWordLeft:      key.NewBinding(key.WithKeys("ctrl+left")),
	JumpWordRight:     key.NewBinding(key.WithKeys("ctrl+right")),
	JumpLinesUp:       key.NewBinding(key.WithKeys("ctrl+up")),
	JumpLinesDown:     key.NewBinding(key.WithKeys("ctrl+down")),
	SelectWordLeft:    key.NewBinding(key.WithKeys("ctrl+shift+left")),
	SelectWordRight:   key.NewBinding(key.WithKeys("ctrl+shift+right")),
	SelectLinesUp:     key.NewBinding(key.WithKeys("ctrl+shift+up")),
	SelectLinesDown:   key.NewBinding(key.WithKeys("ctrl+shift+down")),
	LineStart:         key.NewBinding(key.WithKeys("home")),
	LineEnd:           key.NewBinding(key.WithKeys("end")),
	FileStart:         key.NewBinding(key.WithKeys("ctrl+home")),
	FileEnd:           key.NewBinding(key.WithKeys("ctrl+end")),
	SelectToLineStart: key.NewBinding(key.WithKeys("shift+home")),
	SelectToLineEnd:   key.NewBinding(key.WithKeys("shift+end")),
}

var (
	styleCursor   = lipgloss.NewStyle().Background(lipgloss.Color("252")).Foreground(lipgloss.Color("0"))
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("0"))
	styleSearch   = lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0")) // Yellow background for search
	styleFile     = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	styleDir      = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	lineNumStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	borderStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("235"))

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			MarginBottom(1).
			Align(lipgloss.Center)
)

type OpType int

const (
	OpInsert OpType = iota
	OpDelete
)

type EditOp struct {
	Type OpType
	Row  int
	Col  int
	Text string
}

type Model struct {
	TextArea           textarea.Model
	Width              int
	Height             int
	FileName           string
	KeyMap             KeyMap
	Quitting           bool
	startRow           int
	startCol           int
	selecting          bool
	saving             bool
	loading            bool
	goToLine           bool
	searching          bool
	textInput          textinput.Model
	filePicker         filepicker.Model
	statusMsg          string
	yOffset            int
	Lines              []string
	CursorRow          int
	CursorCol          int
	UndoStack          []EditOp
	RedoStack          []EditOp
	Config             config.Config
	showHelp           bool
	searchQuery        string
	searchResults      []search.SearchMatch
	currentResultIndex int
}

func InitialModel(filename string, content string, cfg config.Config) Model {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.Placeholder = "Digite algo..."
	ta.SetValue(content)
	ta.Focus()
	//styles.setupStyle()

	ti := textinput.New()
	ti.Placeholder = "filename.txt"
	ti.Prompt = "Filename: "
	ti.CharLimit = 156
	ti.Width = 20

	fp := filepicker.New()
	fp.AllowedTypes = nil // All files
	fp.CurrentDirectory, _ = os.Getwd()
	fp.Height = 15
	fp.ShowHidden = true
	fp.Styles.Cursor = styleCursor
	fp.Styles.Selected = styleSelected
	fp.Styles.File = styleFile
	fp.Styles.Directory = styleDir

	SetTheme(cfg.Theme)

	return Model{
		TextArea:           ta,
		Width:              80,
		Height:             20,
		FileName:           filename,
		KeyMap:             DefaultKeyMap,
		Quitting:           false,
		startRow:           0,
		startCol:           0,
		selecting:          false,
		textInput:          ti,
		saving:             false,
		loading:            false,
		filePicker:         fp,
		Lines:              strings.Split(content, "\n"),
		CursorRow:          0,
		CursorCol:          0,
		Config:             cfg,
		showHelp:           false,
		searching:          false,
		searchQuery:        "",
		searchResults:      nil,
		currentResultIndex: -1,
	}
}

func getCol(ta textarea.Model) int {
	li := ta.LineInfo()
	// Together they give the correct logical column index.
	return li.StartColumn + li.CharOffset
}

func getRow(ta textarea.Model) int {
	return ta.Line()
}

// deleteSelection removes the selected text and positions cursor at start of selection.
func (m *Model) deleteSelection() bool {
	if !m.selecting {
		return false
	}

	val := m.TextArea.Value()
	if val == "" {
		m.selecting = false
		return false
	}

	// Get current cursor position
	curRow := getRow(m.TextArea)
	curCol := getCol(m.TextArea)

	startIdx := getAbsoluteIndex(val, m.startRow, m.startCol)
	endIdx := getAbsoluteIndex(val, curRow, curCol)

	// Determine which position is the "start" (leftmost) of the selection
	targetRow := m.startRow
	targetCol := m.startCol
	if startIdx > endIdx {
		startIdx, endIdx = endIdx, startIdx
		targetRow = curRow
		targetCol = curCol
	}

	// Convert to rune slice to handle unicode correctly
	runes := []rune(val)
	if startIdx >= len(runes) {
		startIdx = len(runes)
	}
	if endIdx > len(runes) {
		endIdx = len(runes)
	}

	// Create new content without selected text
	newRunes := append(runes[:startIdx], runes[endIdx:]...)
	newVal := string(newRunes)

	// Set the new value
	m.TextArea.SetValue(newVal)

	// Position cursor at the start of where selection was
	lines := strings.Split(newVal, "\n")
	if targetRow >= len(lines) {
		targetRow = len(lines) - 1
	}
	if targetRow < 0 {
		targetRow = 0
	}

	// Move cursor to beginning of document (line 0, col 0)
	for getRow(m.TextArea) > 0 {
		m.TextArea.CursorUp()
	}
	m.TextArea.CursorStart()

	// Navigate to target row
	for i := 0; i < targetRow; i++ {
		m.TextArea.CursorDown()
	}

	// Set column position - ensure it's within bounds of the new line
	if targetRow < len(lines) && targetCol > len([]rune(lines[targetRow])) {
		targetCol = len([]rune(lines[targetRow]))
	}
	m.TextArea.SetCursor(targetCol)

	m.selecting = false
	return true
}

func Write(errorMessage string) {
	f, err := os.OpenFile("larry.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	defer f.Close()

	log.SetOutput(f)
	log.Println(errorMessage)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	if m.loading {
		var cmd tea.Cmd
		m.filePicker, cmd = m.filePicker.Update(msg)
		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			content, err := os.ReadFile(path)
			if err != nil {
				m.statusMsg = "Error opening: " + err.Error()
			} else {
				// Replaced TextArea logic with manual Lines logic
				m.Lines = strings.Split(string(content), "\n")
				m.statusMsg = "Opened: " + path
				m.FileName = path
				m.CursorRow = 0
				m.CursorCol = 0
				m.yOffset = 0
				m.selecting = false
			}
			m.loading = false
			return m, cmd
		}
		// Handle user manual quit via Esc
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEsc {
			m.loading = false
		}
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
				// Save from m.Lines
				content := strings.Join(m.Lines, "\n")
				err := os.WriteFile(filename, []byte(content), 0644)
				if err != nil {
					m.statusMsg = "Error saving: " + err.Error()
				} else {
					m.statusMsg = "Saved: " + filename
					m.FileName = filename
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
					targetLine-- // Adjust from 1-indexed UI to 0-indexed code
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
				query := m.textInput.Value()
				if query != "" && query != m.searchQuery {
					// New search
					m.searchQuery = query
					searcher := search.NewBoyerMooreSearch(query)
					m.searchResults = searcher.SearchInLines(m.Lines)
					m.currentResultIndex = -1
				}
				// Navigate to next result or first result
				if len(m.searchResults) > 0 {
					m.currentResultIndex = (m.currentResultIndex + 1) % len(m.searchResults)
					result := m.searchResults[m.currentResultIndex]
					m.CursorRow = result.Line
					m.CursorCol = result.Col
					m = m.updateViewport()
				}
				return m, nil
			}
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
	case tea.KeyMsg:
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
		m.TextArea.SetWidth(msg.Width)
		m.TextArea.SetHeight(msg.Height - 1) // Reserve 1 line for status bar

		// Reset yOffset to valid bounds after resize
		if len(m.Lines) <= m.TextArea.Height() {
			m.yOffset = 0
		} else {
			maxOffset := len(m.Lines) - m.TextArea.Height()
			if m.yOffset > maxOffset {
				m.yOffset = maxOffset
			}
			if m.yOffset < 0 {
				m.yOffset = 0
			}
		}

		m = m.updateViewport()
		return m, nil
	}

	var taCmd tea.Cmd

	// Fallback for non-key messages
	if _, ok := msg.(tea.KeyMsg); !ok {
		// Only pass non-key messages (like blink)
		m.TextArea, taCmd = m.TextArea.Update(msg)
	}

	// Adjust Viewport Y-Offset based on CursorRow
	if m.CursorRow < m.yOffset {
		m.yOffset = m.CursorRow
	}
	if m.CursorRow >= m.yOffset+m.TextArea.Height() {
		m.yOffset = m.CursorRow - m.TextArea.Height() + 1
	}

	return m, taCmd
}

func (m Model) View() string {
	if m.Quitting {
		return "Tchau!\n"
	}

	baseView := ""

	// Use Optimized Custom View for Editor
	// val := m.TextArea.Value()

	if len(m.Lines) == 0 && !m.loading {
		// Just render one empty line with cursor
		s := strings.Builder{}
		s.WriteString(borderStyle.Render("│"))
		if m.Config.LineNumbers {
			s.WriteString(lineNumStyle.Render("   1 "))
		}
		s.WriteString(styleCursor.Render(" ")) // Cursor
		s.WriteString("\n")
		baseView = s.String()
	} else {
		// Prepare selection ranges (Row/Col based)
		selStartRow, selStartCol := -1, -1
		selEndRow, selEndCol := -1, -1

		if m.selecting {
			// Normalize start/end
			sRow, sCol := m.startRow, m.startCol
			eRow, eCol := m.CursorRow, m.CursorCol

			if sRow > eRow || (sRow == eRow && sCol > eCol) {
				sRow, sCol, eRow, eCol = eRow, eCol, sRow, sCol
			}
			selStartRow, selStartCol = sRow, sCol
			selEndRow, selEndCol = eRow, eCol
		}

		// Cursor tracking
		cursorRow := m.CursorRow
		cursorCol := m.CursorCol

		// Calculate available width
		textWidth := m.TextArea.Width()
		if m.Config.LineNumbers {
			textWidth -= 6
		}
		textWidth -= 1 // Border
		if textWidth < 1 {
			textWidth = 1
		}

		lines := m.Lines
		var s strings.Builder

		maxVisualLines := m.TextArea.Height()
		visualLinesRendered := 0
		currentVisualLineIndex := 0

		for lineNum := 0; lineNum < len(lines) && visualLinesRendered < maxVisualLines; lineNum++ {
			line := lines[lineNum]
			lineRunes := []rune(line)

			// Helper to render a chunk
			renderChunk := func(runes []rune, startIdx, endIdx int, isFirst bool, currentLineVisualWidth int) {
				if visualLinesRendered >= maxVisualLines {
					return
				}
				if currentVisualLineIndex < m.yOffset {
					currentVisualLineIndex++
					return
				}

				s.WriteString(borderStyle.Render("│"))
				if m.Config.LineNumbers {
					if isFirst {
						ln := fmt.Sprintf(" %3d ", lineNum+1)
						s.WriteString(lineNumStyle.Render(ln))
					} else {
						s.WriteString(lineNumStyle.Render("      "))
					}
				}

				syntaxStyles := GetLineStyles(m.Lines[lineNum], m.FileName)

				for i := startIdx; i < endIdx; i++ {
					ch := runes[i]
					var style lipgloss.Style
					applyStyle := false

					// Base syntax highlighting
					if i < len(syntaxStyles) {
						style = syntaxStyles[i]
						applyStyle = true
					}

					// Search highlighting (can override syntax)
					if len(m.searchResults) > 0 {
						for _, result := range m.searchResults {
							if result.Line == lineNum && i >= result.Col && i < result.Col+result.Length {
								style = styleSearch
								applyStyle = true
								break
							}
						}
					}

					// Selection highlighting (can override search and syntax)
					if m.selecting {
						isSelected := false
						if lineNum > selStartRow && lineNum < selEndRow {
							isSelected = true
						} else if lineNum == selStartRow && lineNum == selEndRow {
							if i >= selStartCol && i < selEndCol {
								isSelected = true
							}
						} else if lineNum == selStartRow {
							if i >= selStartCol {
								isSelected = true
							}
						} else if lineNum == selEndRow {
							if i < selEndCol {
								isSelected = true
							}
						}
						if isSelected {
							style = styleSelected
							applyStyle = true
						}
					}

					// Cursor highlighting (highest priority, overrides everything)
					if lineNum == cursorRow && i == cursorCol {
						style = styleCursor
						applyStyle = true
					}

					visualChar := string(ch)
					if ch == '\t' {
						visualChar = strings.Repeat(" ", m.Config.TabWidth)
					}

					if applyStyle {
						if !m.selecting && lineNum == cursorRow && i == cursorCol {
							if ch == '\t' {
								s.WriteString("\x1b[47m\x1b[30m \x1b[0m" + strings.Repeat(" ", m.Config.TabWidth-1))
							} else {
								s.WriteString("\x1b[47m\x1b[30m" + visualChar + "\x1b[0m")
							}
						} else {
							s.WriteString(style.Render(visualChar))
						}
					} else {
						s.WriteString(visualChar)
					}
				}

				if !m.selecting && lineNum == cursorRow && cursorCol == len(runes) && endIdx == len(runes) {
					s.WriteString("\x1b[47m\x1b[30m \x1b[0m")
				}

				s.WriteString("\x1b[K")
				visualLinesRendered++
				if visualLinesRendered < maxVisualLines {
					s.WriteString("\n")
				}
				currentVisualLineIndex++
			}

			if len(lineRunes) == 0 {
				renderChunk(nil, 0, 0, true, 0)
				continue
			}

			chunkStart := 0
			currentVisualWidth := 0
			isFirst := true
			for i := 0; i < len(lineRunes); i++ {
				charWidth := 1
				if lineRunes[i] == '\t' {
					charWidth = m.Config.TabWidth
				}

				if currentVisualWidth+charWidth > textWidth {
					renderChunk(lineRunes, chunkStart, i, isFirst, currentVisualWidth)
					chunkStart = i
					currentVisualWidth = charWidth
					isFirst = false
				} else {
					currentVisualWidth += charWidth
				}
			}
			if chunkStart <= len(lineRunes) {
				renderChunk(lineRunes, chunkStart, len(lineRunes), isFirst, currentVisualWidth)
			}
		}
		baseView = s.String()
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
	if m.loading {
		title := modalTitleStyle.Render("Open File")
		pickerView := m.filePicker.View()

		content := lipgloss.JoinVertical(lipgloss.Left, title, pickerView)
		modal := modalStyle.Render(content)

		return lipgloss.Place(
			m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			modal,
		)
	}

	if m.showHelp {
		return m.viewHelpMenu(baseView)
	}

	// Status Bar
	msg := m.statusMsg
	if msg == "" {
		if len(m.searchResults) > 0 {
			msg = fmt.Sprintf("Search: %s (%d/%d) | Ctrl+h: Help | Ctrl+q: Quit | Ctrl+s: Save | Ctrl+f: Search",
				m.searchQuery, m.currentResultIndex+1, len(m.searchResults))
		} else {
			msg = "Ctrl+h: Help | Ctrl+q: Quit | Ctrl+s: Save | Ctrl+f: Search"
		}
	}
	// Pad status bar
	width := m.Width
	if width < len(msg) {
		width = len(msg)
	}
	status := statusBarStyle.Width(width).Render(" " + msg)

	return lipgloss.JoinVertical(lipgloss.Left, baseView, status)
}

func (m Model) viewHelpMenu(base string) string {
	width := m.Width
	height := m.Height

	// Ensure we have valid dimensions
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	// Define visual style for the help menu
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252"))

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)

	categoryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("117")).
		Bold(true).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("154")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248"))

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	// Left column: General shortcuts
	generalShortcuts := []struct {
		Key  string
		Desc string
	}{
		{"Ctrl+q", "Quit"},
		{"Ctrl+s", "Save"},
		{"Ctrl+o", "Open File"},
		{"Ctrl+g", "Go to Line"},
		{"Ctrl+f", "Search"},
		{"Ctrl+h", "Toggle Help"},
		{"Ctrl+z", "Undo"},
		{"Ctrl+R", "Redo"},
		{"Ctrl+c", "Copy"},
		{"Ctrl+v", "Paste"},
		{"Ctrl+x", "Cut"},
		{"Ctrl+a", "Select All"},
	}

	// Right column: Navigation shortcuts
	navShortcuts := []struct {
		Key  string
		Desc string
	}{
		{"←/→/↑/↓", "Move Cursor"},
		{"Shift+Arrow", "Select Text"},
		{"Ctrl+←/→", "Jump Word"},
		{"Ctrl+↑/↓", "Jump 5 Lines"},
		{"Ctrl+Shift+←/→", "Select Word"},
		{"Ctrl+Shift+↑/↓", "Select Lines"},
		{"Home", "Line Start"},
		{"End", "Line End"},
		{"Shift+Home/End", "Select to Start/End"},
		{"Ctrl+Home", "File Start"},
		{"Ctrl+End", "File End"},
	}

	// Build left column
	var leftCol strings.Builder
	leftCol.WriteString(categoryStyle.Render("General"))
	leftCol.WriteString("\n")
	for _, s := range generalShortcuts {
		leftCol.WriteString(fmt.Sprintf("%-14s %s\n", keyStyle.Render(s.Key), descStyle.Render(s.Desc)))
	}

	// Build right column
	var rightCol strings.Builder
	rightCol.WriteString(categoryStyle.Render("Navigation"))
	rightCol.WriteString("\n")
	for _, s := range navShortcuts {
		rightCol.WriteString(fmt.Sprintf("%-18s %s\n", keyStyle.Render(s.Key), descStyle.Render(s.Desc)))
	}

	// Combine columns side by side
	leftColStyled := lipgloss.NewStyle().MarginRight(3).Render(leftCol.String())
	rightColStyled := lipgloss.NewStyle().Render(rightCol.String())
	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftColStyled, rightColStyled)

	// Calculate content width for centering title
	contentWidth := lipgloss.Width(columns)

	// Build final menu with centered title
	title := "Larry - Help Menu"
	centeredTitle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(titleStyle.Render(title))
	footer := "Press Esc or Ctrl+h to close"
	centeredFooter := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(footerStyle.Render(footer))

	var sb strings.Builder
	sb.WriteString(centeredTitle)
	sb.WriteString("\n")
	sb.WriteString(columns)
	sb.WriteString("\n")
	sb.WriteString(centeredFooter)

	helpMenu := helpStyle.Render(sb.String())

	// Center the help menu on screen
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, helpMenu)
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.KeyMap.ToggleHelp):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.KeyMap.Quit):
		m.Quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.KeyMap.Save):
		m.saving = true
		m.textInput.Focus()
		m.textInput.SetValue(m.FileName)
		m.textInput.Prompt = "Filename: "
		return m, nil

	case key.Matches(msg, m.KeyMap.GoToLine):
		m.goToLine = true
		m.textInput.Focus()
		m.textInput.SetValue("")
		lineCount := len(m.Lines)
		if lineCount == 0 {
			lineCount = 1
		}
		m.textInput.Prompt = fmt.Sprintf("Go to line (1-%d): ", lineCount)
		return m, nil

	case key.Matches(msg, m.KeyMap.Search):
		m.searching = true
		m.textInput.Focus()
		m.textInput.SetValue(m.searchQuery)
		m.textInput.Prompt = "Search: "
		return m, nil

	case key.Matches(msg, m.KeyMap.Open):
		m.loading = true
		m.filePicker.CurrentDirectory, _ = os.Getwd()
		return m, m.filePicker.Init()

	case key.Matches(msg, m.KeyMap.Undo):
		m = m.undo()
		return m, nil

	case key.Matches(msg, m.KeyMap.Redo):
		m = m.redo()
		return m, nil

	case key.Matches(msg, m.KeyMap.SelectAll):
		m.startRow = 0
		m.startCol = 0
		m.CursorRow = len(m.Lines) - 1
		// Check valid cursor row
		if m.CursorRow < 0 {
			m.CursorRow = 0
		}
		m.CursorCol = len([]rune(m.Lines[m.CursorRow]))
		m.selecting = true
		return m, nil

	case key.Matches(msg, m.KeyMap.Cut):
		if m.selecting {
			text := m.getSelectedText()
			err := m.clipboardWrite(text)
			if err != nil {
				m.statusMsg = "Cut Error: " + err.Error()
			} else {
				m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
				m = m.deleteSelectedText()
				m.statusMsg = "Cut to clipboard"
			}
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.Copy):
		if m.selecting {
			text := m.getSelectedText()
			err := m.clipboardWrite(text)
			if err != nil {
				m.statusMsg = "Copy Error: " + err.Error()
			} else {
				m.selecting = false
				m.statusMsg = "Copied to clipboard"
			}
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.Paste):
		text, err := m.clipboardRead()
		if err != nil {
			m.statusMsg = "Paste Error: " + err.Error()
		} else {
			m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: text})
			m = m.insertTextAtCursor(text)
			m.statusMsg = "Pasted from clipboard"
		}
		return m, nil

	// UP
	case key.Matches(msg, m.KeyMap.CursorUp) || key.Matches(msg, m.KeyMap.MoveSelectionUp):
		if key.Matches(msg, m.KeyMap.MoveSelectionUp) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		if m.CursorRow > 0 {
			m.CursorRow--
			if m.CursorRow < len(m.Lines) {
				lineLen := len([]rune(m.Lines[m.CursorRow]))
				if m.CursorCol > lineLen {
					m.CursorCol = lineLen
				}
			}
		}

	// DOWN
	case key.Matches(msg, m.KeyMap.CursorDown) || key.Matches(msg, m.KeyMap.MoveSelectionDown):
		if key.Matches(msg, m.KeyMap.MoveSelectionDown) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		if m.CursorRow < len(m.Lines)-1 {
			m.CursorRow++
			lineLen := len([]rune(m.Lines[m.CursorRow]))
			if m.CursorCol > lineLen {
				m.CursorCol = lineLen
			}
		}

	// LEFT
	case key.Matches(msg, m.KeyMap.CursorLeft) || key.Matches(msg, m.KeyMap.MoveSelectionLeft):
		if key.Matches(msg, m.KeyMap.MoveSelectionLeft) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		if m.CursorCol > 0 {
			m.CursorCol--
		} else if m.CursorRow > 0 {
			m.CursorRow--
			m.CursorCol = len([]rune(m.Lines[m.CursorRow]))
		}

	// RIGHT
	case key.Matches(msg, m.KeyMap.CursorRight) || key.Matches(msg, m.KeyMap.MoveSelectionRight):
		if key.Matches(msg, m.KeyMap.MoveSelectionRight) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		lineLen := len([]rune(m.Lines[m.CursorRow]))
		if m.CursorCol < lineLen {
			m.CursorCol++
		} else if m.CursorRow < len(m.Lines)-1 {
			m.CursorRow++
			m.CursorCol = 0
		}

	// JUMP WORD RIGHT (Ctrl+Right)
	case key.Matches(msg, m.KeyMap.JumpWordRight) || key.Matches(msg, m.KeyMap.SelectWordRight):
		if key.Matches(msg, m.KeyMap.SelectWordRight) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		m.CursorRow, m.CursorCol = FindNextWordBoundary(m.Lines, m.CursorRow, m.CursorCol)

	// JUMP WORD LEFT (Ctrl+Left)
	case key.Matches(msg, m.KeyMap.JumpWordLeft) || key.Matches(msg, m.KeyMap.SelectWordLeft):
		if key.Matches(msg, m.KeyMap.SelectWordLeft) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		m.CursorRow, m.CursorCol = FindPrevWordBoundary(m.Lines, m.CursorRow, m.CursorCol)

	// JUMP LINES UP (Ctrl+Up)
	case key.Matches(msg, m.KeyMap.JumpLinesUp) || key.Matches(msg, m.KeyMap.SelectLinesUp):
		if key.Matches(msg, m.KeyMap.SelectLinesUp) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		m.CursorRow, m.CursorCol = JumpLinesUp(m.Lines, m.CursorRow, m.CursorCol)

	// JUMP LINES DOWN (Ctrl+Down)
	case key.Matches(msg, m.KeyMap.JumpLinesDown) || key.Matches(msg, m.KeyMap.SelectLinesDown):
		if key.Matches(msg, m.KeyMap.SelectLinesDown) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		m.CursorRow, m.CursorCol = JumpLinesDown(m.Lines, m.CursorRow, m.CursorCol)

	// HOME (Line Start)
	case key.Matches(msg, m.KeyMap.LineStart) || key.Matches(msg, m.KeyMap.SelectToLineStart):
		if key.Matches(msg, m.KeyMap.SelectToLineStart) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		m.CursorRow, m.CursorCol = MoveToLineStart(m.CursorRow, m.CursorCol)

	// END (Line End)
	case key.Matches(msg, m.KeyMap.LineEnd) || key.Matches(msg, m.KeyMap.SelectToLineEnd):
		if key.Matches(msg, m.KeyMap.SelectToLineEnd) {
			if !m.selecting {
				m.selecting = true
				m.startRow, m.startCol = m.CursorRow, m.CursorCol
			}
		} else {
			m.selecting = false
		}
		m.CursorRow, m.CursorCol = MoveToLineEnd(m.Lines, m.CursorRow)

	// FILE START (Ctrl+Home)
	case key.Matches(msg, m.KeyMap.FileStart):
		m.selecting = false
		m.CursorRow, m.CursorCol = MoveToFileStart()

	// FILE END (Ctrl+End)
	case key.Matches(msg, m.KeyMap.FileEnd):
		m.selecting = false
		m.CursorRow, m.CursorCol = MoveToFileEnd(m.Lines)

	// Typing (Chars) and Space
	case msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace:
		// Logic to handle deleting selection before typing
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
			// We should probably group this with the insert?
			// For now, it's 2 atomic ops: Delete Selection, then Insert Char.
			// Ideally they should be 1 transaction.
			// But sticking to simple 1:1 for now.
		}
		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: string(msg.Runes)})

			line := []rune(m.Lines[m.CursorRow])
			prefix := line[:m.CursorCol]
			suffix := line[m.CursorCol:]

			var runes []rune
			if msg.Type == tea.KeySpace {
				runes = []rune{' '}
			} else {
				runes = msg.Runes
			}

			newLine := append(prefix, append(runes, suffix...)...)
			m.Lines[m.CursorRow] = string(newLine)
			m.CursorCol += len(runes)
		}

	// Backspace
	case msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete || key.Matches(msg, m.KeyMap.Delete):
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		} else {
			if m.CursorCol > 0 {
				line := []rune(m.Lines[m.CursorRow])
				deletedChar := string(line[m.CursorCol-1])
				m.pushUndo(EditOp{Type: OpDelete, Row: m.CursorRow, Col: m.CursorCol - 1, Text: deletedChar})

				newLine := append(line[:m.CursorCol-1], line[m.CursorCol:]...)
				m.Lines[m.CursorRow] = string(newLine)
				m.CursorCol--
			} else if m.CursorRow > 0 {
				// Deleting newline from previous line
				prevLine := m.Lines[m.CursorRow-1]
				currLine := m.Lines[m.CursorRow]
				m.pushUndo(EditOp{Type: OpDelete, Row: m.CursorRow - 1, Col: len([]rune(prevLine)), Text: "\n"})

				newCol := len([]rune(prevLine))
				m.Lines[m.CursorRow-1] = prevLine + currLine
				m.Lines = append(m.Lines[:m.CursorRow], m.Lines[m.CursorRow+1:]...)
				m.CursorRow--
				m.CursorCol = newCol
			}
		}

	// Tab
	case msg.Type == tea.KeyTab:
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		}

		tab := "    "
		m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: tab})
		m = m.insertTextAtCursor(tab)
		return m, nil

	// Shift+Tab (Dedent)
	case msg.Type == tea.KeyShiftTab:
		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			line := m.Lines[m.CursorRow]
			// Check for leading spaces
			spacesToRemove := 0
			for i, r := range line {
				if i >= 4 {
					break
				}
				if r == ' ' {
					spacesToRemove++
				} else {
					break
				}
			}

			if spacesToRemove > 0 {
				dedentText := line[:spacesToRemove]

				// Record Undo (removed text at start of line: Col 0)
				m.pushUndo(EditOp{Type: OpDelete, Row: m.CursorRow, Col: 0, Text: dedentText})

				// Remove from line
				m.Lines[m.CursorRow] = line[spacesToRemove:]

				// Adjust cursor
				m.CursorCol -= spacesToRemove
				if m.CursorCol < 0 {
					m.CursorCol = 0
				}
			}
		}
		return m, nil

	// Enter
	case msg.Type == tea.KeyEnter:
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		}

		m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: "\n"})

		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			line := []rune(m.Lines[m.CursorRow])
			prefix := line[:m.CursorCol]
			suffix := line[m.CursorCol:]

			m.Lines[m.CursorRow] = string(prefix)
			// Efficiently insert new line
			newLines := make([]string, 0, len(m.Lines)+1)
			newLines = append(newLines, m.Lines[:m.CursorRow+1]...)
			newLines = append(newLines, string(suffix))
			newLines = append(newLines, m.Lines[m.CursorRow+1:]...)
			m.Lines = newLines

			m.CursorRow++
			m.CursorCol = 0
		}
	}
	return m, cmd
}

func getAbsoluteIndex(value string, row, col int) int {
	if value == "" {
		return 0
	}

	lines := strings.Split(value, "\n")

	// Ensure row is within bounds
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}

	// Calculate rune index
	runeIndex := 0
	for i := 0; i < row; i++ {
		runeIndex += len([]rune(lines[i])) + 1 // +1 for the \n
	}

	// Ensure col is within bounds of the current line (in runes)
	lineRunes := []rune(lines[row])
	if col < 0 {
		col = 0
	}
	if col > len(lineRunes) {
		col = len(lineRunes)
	}

	runeIndex += col

	// Final bounds check against total rune count
	totalRunes := len([]rune(value))
	if runeIndex > totalRunes {
		runeIndex = totalRunes
	}

	return runeIndex
}

func (m Model) getVisualLineCount(lineNum int, textWidth int) int {
	if lineNum < 0 || lineNum >= len(m.Lines) {
		return 0
	}
	line := m.Lines[lineNum]
	if line == "" {
		return 1
	}

	visualWidth := 0
	count := 1
	for _, r := range line {
		charWidth := 1
		if r == '\t' {
			charWidth = 4
		}
		if visualWidth+charWidth > textWidth {
			count++
			visualWidth = charWidth
		} else {
			visualWidth += charWidth
		}
	}
	return count
}

func (m Model) getCursorVisualOffset(textWidth int) int {
	totalVisualLines := 0
	for i := 0; i < m.CursorRow; i++ {
		totalVisualLines += m.getVisualLineCount(i, textWidth)
	}

	// Visual offset within the current line
	line := []rune(m.Lines[m.CursorRow])
	currentLineVisualLine := 0
	visualWidth := 0
	for i := 0; i < m.CursorCol && i < len(line); i++ {
		charWidth := 1
		if line[i] == '\t' {
			charWidth = 4
		}
		if visualWidth+charWidth > textWidth {
			currentLineVisualLine++
			visualWidth = charWidth
		} else {
			visualWidth += charWidth
		}
	}

	return totalVisualLines + currentLineVisualLine
}

func (m Model) updateViewport() Model {
	textWidth := m.TextArea.Width()
	if m.TextArea.ShowLineNumbers {
		textWidth -= 6
	}
	textWidth -= 1 // Border
	if textWidth < 1 {
		textWidth = 1
	}

	cursorVisualLine := m.getCursorVisualOffset(textWidth)

	if cursorVisualLine < m.yOffset {
		m.yOffset = cursorVisualLine
	}
	if cursorVisualLine >= m.yOffset+m.TextArea.Height() {
		m.yOffset = cursorVisualLine - m.TextArea.Height() + 1
	}
	return m
}

func (m Model) getSelectedText() string {
	if !m.selecting {
		return ""
	}

	startRow, startCol := m.startRow, m.startCol
	endRow, endCol := m.CursorRow, m.CursorCol

	// Normalize order
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	if startRow == endRow {
		if startCol < 0 {
			startCol = 0
		}
		line := []rune(m.Lines[startRow])
		if endCol > len(line) {
			endCol = len(line)
		}
		if startCol > len(line) {
			startCol = len(line)
		}
		return string(line[startCol:endCol])
	}

	var builder strings.Builder
	// First line
	line := []rune(m.Lines[startRow])
	if startCol < len(line) {
		builder.WriteString(string(line[startCol:]))
	}
	builder.WriteString("\n")

	// Middle lines
	for i := startRow + 1; i < endRow; i++ {
		builder.WriteString(m.Lines[i])
		builder.WriteString("\n")
	}

	// Last line
	line = []rune(m.Lines[endRow])
	if endCol > len(line) {
		endCol = len(line)
	}
	if endCol > 0 {
		builder.WriteString(string(line[:endCol]))
	}

	return builder.String()
}

func (m Model) deleteSelectedText() Model {
	if !m.selecting {
		return m
	}

	startRow, startCol := m.startRow, m.startCol
	endRow, endCol := m.CursorRow, m.CursorCol

	// Normalize order
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	// Single line deletion
	if startRow == endRow {
		line := []rune(m.Lines[startRow])
		// Check bounds
		if startCol < 0 {
			startCol = 0
		}
		if endCol > len(line) {
			endCol = len(line)
		}

		newLine := append(line[:startCol], line[endCol:]...)
		m.Lines[startRow] = string(newLine)
		m.CursorRow = startRow
		m.CursorCol = startCol
		m.selecting = false
		return m
	}

	// Multi-line deletion
	// Start line prefix
	startLine := []rune(m.Lines[startRow])
	if startCol > len(startLine) {
		startCol = len(startLine)
	}
	prefix := string(startLine[:startCol])

	// End line suffix
	endLine := []rune(m.Lines[endRow])
	if endCol > len(endLine) {
		endCol = len(endLine)
	}
	suffix := string(endLine[endCol:])

	// Merge
	m.Lines[startRow] = prefix + suffix

	// Delete intermediate lines
	m.Lines = append(m.Lines[:startRow+1], m.Lines[endRow+1:]...)

	m.CursorRow = startRow
	m.CursorCol = startCol
	m.selecting = false
	return m
}

func (m Model) insertTextAtCursor(text string) Model {
	if text == "" {
		return m
	}

	// Split input text by newline
	// Note: Normalize CRLF?
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	linesToInsert := strings.Split(text, "\n")

	// Current Line separation
	row := m.CursorRow
	col := m.CursorCol
	if row >= len(m.Lines) {
		row = len(m.Lines) - 1
	} // Safety
	if row < 0 {
		row = 0
		m.Lines = []string{""}
	}

	line := []rune(m.Lines[row])
	if col > len(line) {
		col = len(line)
	}

	prefix := string(line[:col])
	suffix := string(line[col:])

	if len(linesToInsert) == 1 {
		// Single line insert
		m.Lines[row] = prefix + linesToInsert[0] + suffix
		m.CursorCol += len([]rune(linesToInsert[0]))
	} else {
		// Multi-line insert
		// 1. Current row becomes Prefix + First Insert Line
		m.Lines[row] = prefix + linesToInsert[0]

		// 2. Middle lines are inserted as-is
		// 3. Last inserted line + Suffix becomes a new line
		var middleLines []string
		for i := 1; i < len(linesToInsert)-1; i++ {
			middleLines = append(middleLines, linesToInsert[i])
		}

		lastInsertLine := linesToInsert[len(linesToInsert)-1]
		lastLineContent := lastInsertLine + suffix

		// Reconstruct slice
		newLines := make([]string, 0)
		newLines = append(newLines, m.Lines[:row+1]...)
		newLines = append(newLines, middleLines...)
		newLines = append(newLines, lastLineContent)
		newLines = append(newLines, m.Lines[row+1:]...)
		m.Lines = newLines

		m.CursorRow += len(linesToInsert) - 1
		m.CursorCol = len([]rune(lastInsertLine))
	}
	return m
}

func (m Model) clipboardWrite(text string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("pbcopy")
	} else {
		cmd = exec.Command("xclip", "-selection", "clipboard", "-in")
	}

	cmd.Stdin = strings.NewReader(text)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Try wl-copy on failure (Wayland)
		if runtime.GOOS == "linux" {
			wlCmd := exec.Command("wl-copy")
			wlCmd.Stdin = strings.NewReader(text)
			if errWl := wlCmd.Run(); errWl == nil {
				return nil
			}
		}
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}

func (m Model) clipboardRead() (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("pbpaste")
	} else {
		cmd = exec.Command("xclip", "-selection", "clipboard", "-out")
	}

	out, err := cmd.Output()
	if err != nil {
		// Try wl-paste (Wayland)
		if runtime.GOOS == "linux" {
			wlCmd := exec.Command("wl-paste")
			outWl, errWl := wlCmd.Output()
			if errWl == nil {
				return string(outWl), nil
			}
		}
		return "", err
	}
	return string(out), nil
}

func (m *Model) pushUndo(op EditOp) {
	m.UndoStack = append(m.UndoStack, op)
	m.RedoStack = nil // Clear redo stack on new operation
}

func (m Model) undo() Model {
	if len(m.UndoStack) == 0 {
		m.statusMsg = "Nothing to undo"
		return m
	}

	// Pop
	op := m.UndoStack[len(m.UndoStack)-1]
	m.UndoStack = m.UndoStack[:len(m.UndoStack)-1]

	switch op.Type {
	case OpInsert:
		m.startRow = op.Row
		m.startCol = op.Col

		lines := strings.Split(op.Text, "\n")
		if len(lines) == 1 {
			m.CursorRow = op.Row
			m.CursorCol = op.Col + len([]rune(lines[0]))
		} else {
			m.CursorRow = op.Row + len(lines) - 1
			m.CursorCol = len([]rune(lines[len(lines)-1]))
		}

		m.selecting = true
		m = m.deleteSelectedText()
		m.selecting = false

	case OpDelete:
		m.CursorRow = op.Row
		m.CursorCol = op.Col
		m = m.insertTextAtCursor(op.Text)
	}

	m.RedoStack = append(m.RedoStack, op)
	m.statusMsg = "Undid change"
	return m
}

func (m Model) redo() Model {
	if len(m.RedoStack) == 0 {
		m.statusMsg = "Nothing to redo"
		return m
	}

	// Pop
	op := m.RedoStack[len(m.RedoStack)-1]
	m.RedoStack = m.RedoStack[:len(m.RedoStack)-1]

	switch op.Type {
	case OpInsert:
		m.CursorRow = op.Row
		m.CursorCol = op.Col
		m = m.insertTextAtCursor(op.Text)

	case OpDelete:
		m.startRow = op.Row
		m.startCol = op.Col
		lines := strings.Split(op.Text, "\n")
		if len(lines) == 1 {
			m.CursorRow = op.Row
			m.CursorCol = op.Col + len([]rune(lines[0]))
		} else {
			m.CursorRow = op.Row + len(lines) - 1
			m.CursorCol = len([]rune(lines[len(lines)-1]))
		}
		m.selecting = true
		m = m.deleteSelectedText()
		m.selecting = false
	}

	m.UndoStack = append(m.UndoStack, op)
	m.statusMsg = "Redid change"
	return m
}
