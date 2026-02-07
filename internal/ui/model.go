package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"larry/internal/config"
	"larry/internal/search"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
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
	finding            bool
	finder             FinderModel
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
	Modified           bool
	viewMode           ViewMode
	markdownRenderer   *glamour.TermRenderer
}

func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown" || ext == ".mdown" || ext == ".mkd"
}

func InitialModel(filename string, content string, cfg config.Config) Model {
	SetTheme(cfg.Theme)
	initStyles()

	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.Placeholder = "Digite algo..."
	ta.SetValue(content)
	ta.Focus()

	ti := textinput.New()
	ti.Placeholder = "filename.txt"
	ti.Prompt = "Filename: "
	ti.CharLimit = 156
	ti.Width = 20
	ti.PromptStyle = textInputStyle
	ti.TextStyle = textInputStyle

	fp := filepicker.New()
	fp.AllowedTypes = nil // All files
	fp.CurrentDirectory, _ = os.Getwd()
	fp.Height = 15
	fp.ShowHidden = true
	fp.Styles.Cursor = styleCursor
	fp.Styles.Selected = styleSelected
	fp.Styles.File = styleFile
	fp.Styles.Directory = styleDir
	fp.Styles.Permission = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(modalStyle.GetBackground())
	fp.Styles.FileSize = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(modalStyle.GetBackground())
	fp.Styles.EmptyDirectory = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(modalStyle.GetBackground())
	fp.Styles.Symlink = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Background(modalStyle.GetBackground())
	fp.Styles.Selected = styleSelected

	return Model{
		TextArea:           ta,
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
		loading:            false,
		filePicker:         fp,
		Lines:              strings.Split(content, "\n"),
		CursorRow:          0,
		CursorCol:          0,
		Config:             cfg,
		showHelp:           false,
		searching:          false,
		finding:            false,
		finder:             NewFinderModel(80, 20),
		searchQuery:        "",
		searchResults:      nil,
		currentResultIndex: -1,
		Modified:           false,
		viewMode:           ViewModeEditor,
		markdownRenderer:   nil,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

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

					content, err := os.ReadFile(path)
					if err == nil {
						m.Lines = strings.Split(string(content), "\n")
						m.FileName = path
						m.CursorRow = targetRow
						m.CursorCol = 0
						m = m.updateViewport()
						m.TextArea.SetValue(string(content))
						idx := getAbsoluteIndex(string(content), m.CursorRow, m.CursorCol)
						m.TextArea.SetCursor(idx)
						m.Modified = false
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

	if m.loading {
		var cmd tea.Cmd
		m.filePicker, cmd = m.filePicker.Update(msg)
		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			content, err := os.ReadFile(path)
			if err != nil {
				m.statusMsg = "Error opening: " + err.Error()
			} else {
				m.Lines = strings.Split(string(content), "\n")
				m.statusMsg = "Opened: " + path
				m.FileName = path
				m.CursorRow = 0
				m.CursorCol = 0
				m.yOffset = 0
				m.selecting = false
				m.Modified = false
				m.viewMode = ViewModeEditor
				m.markdownRenderer = nil
			}
			m.loading = false
			return m, cmd
		}
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
				content := strings.Join(m.Lines, "\n")
				err := os.WriteFile(filename, []byte(content), 0644)
				if err != nil {
					m.statusMsg = "Error saving: " + err.Error()
				} else {
					m.statusMsg = "Saved: " + filename
					m.FileName = filename
					m.Modified = false
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
					m.searchQuery = query
					searcher := search.NewBoyerMooreSearch(query)
					m.searchResults = searcher.SearchInLines(m.Lines)
					m.currentResultIndex = -1
				}
				if len(m.searchResults) > 0 {
					m.currentResultIndex = (m.currentResultIndex + 1) % len(m.searchResults)
					result := m.searchResults[m.currentResultIndex]
					m.CursorRow = result.Line
					m.CursorCol = result.Col

					idx := getAbsoluteIndex(strings.Join(m.Lines, "\n"), m.CursorRow, m.CursorCol)
					m.TextArea.SetCursor(idx)

					textWidth := m.TextArea.Width()
					if m.Config.LineNumbers {
						textWidth -= 6
					}
					textWidth -= 1
					if textWidth < 1 {
						textWidth = 1
					}

					cursorVisualLine := m.getCursorVisualOffset(textWidth)
					availableHeight := m.TextArea.Height() - 2
					targetOffset := cursorVisualLine - (availableHeight / 2)
					if targetOffset < 0 {
						targetOffset = 0
					}
					m.yOffset = targetOffset
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
	case searchMsg:
		var cmd tea.Cmd
		m.finder, cmd = m.finder.Update(msg)
		return m, cmd
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
		m.TextArea.SetHeight(msg.Height - 1)

		if m.viewMode == ViewModeSplit {
			previewWidth := msg.Width/2 - 1
			m.initMarkdownRenderer(previewWidth)
		}

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
		return m, finderCmd
	}

	var taCmd tea.Cmd
	if _, ok := msg.(tea.KeyMsg); !ok {
		m.TextArea, taCmd = m.TextArea.Update(msg)
	}

	return m, taCmd
}

func (m Model) View() string {
	if m.Quitting {
		return "Tchau!\n"
	}

	baseView := ""

	if m.viewMode == ViewModeSplit && isMarkdownFile(m.FileName) {
		baseView = m.viewSplit()
	} else if len(m.Lines) == 0 && !m.loading {
		s := strings.Builder{}
		s.WriteString(borderStyle.Render("│"))
		if m.Config.LineNumbers {
			s.WriteString(lineNumStyle.Render("   1 "))
		}
		s.WriteString(styleCursor.Render(" "))
		s.WriteString("\n")
		baseView = s.String()
	} else {
		baseView = m.viewEditor(editorViewConfig{
			width:        m.TextArea.Width(),
			height:       m.TextArea.Height(),
			showSearchUI: m.searching,
		})
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
	if m.loading {
		pickerView := m.filePicker.View()
		w := lipgloss.Width(pickerView)
		if w < 40 {
			w = 40
		}

		bg := modalStyle.GetBackground()
		spacerStyle := lipgloss.NewStyle().Background(bg)

		titleText := "Open File"
		titleGap := w - lipgloss.Width(titleText)
		leftGap := titleGap / 2
		rightGap := titleGap - leftGap
		title := modalTitleStyle.Width(w).Render(strings.Repeat(" ", leftGap) + titleText + strings.Repeat(" ", rightGap))

		var allLines []string
		allLines = append(allLines, title)
		allLines = append(allLines, spacerStyle.Copy().Width(w).Render(""))

		pickerLines := strings.Split(pickerView, "\n")
		for _, line := range pickerLines {
			styledLine := strings.ReplaceAll(line, " ", spacerStyle.Render(" "))
			allLines = append(allLines, spacerStyle.Width(w).Render(styledLine))
		}

		maxWidth := 0
		for _, line := range allLines {
			lw := lipgloss.Width(line)
			if lw > maxWidth {
				maxWidth = lw
			}
		}

		for i, line := range allLines {
			allLines[i] = spacerStyle.Copy().Width(maxWidth).Render(line)
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

	leader := strings.Title(m.Config.LeaderKey)
	if leader == "" {
		leader = "Leader"
	}

	msg := m.statusMsg
	if msg == "" {
		if len(m.searchResults) > 0 {
			msg = fmt.Sprintf("Search: %s (%d/%d) | %s+h: Help | %s+q: Quit | %s+s: Save | %s+f: Search File | %s+p: Larry Finder",
				m.searchQuery, m.currentResultIndex+1, len(m.searchResults), leader, leader, leader, leader, leader)
		} else if isMarkdownFile(m.FileName) {
			if m.viewMode == ViewModeSplit {
				msg = fmt.Sprintf("%s+m: Close Preview | %s+h: Help | %s+q: Quit | %s+s: Save | %s+p: Larry Finder",
					leader, leader, leader, leader, leader)
			} else {
				msg = fmt.Sprintf("%s+m: Preview | %s+h: Help | %s+q: Quit | %s+s: Save | %s+p: Larry Finder",
					leader, leader, leader, leader, leader)
			}
		} else {
			msg = fmt.Sprintf("%s+o: Open File | %s+h: Help | %s+q: Quit | %s+s: Save | %s+f: Search File | %s+p: Larry Finder",
				leader, leader, leader, leader, leader, leader)
		}
	}

	fileStatus := m.FileName
	if fileStatus == "" {
		fileStatus = "[No Name]"
	}
	if m.Modified {
		fileStatus += " [+]"
	}

	fullStatus := fmt.Sprintf(" %s │ %s", fileStatus, msg)

	width := m.Width
	if width < 20 {
		width = 20
	}

	wrappedMsg := lipgloss.NewStyle().Width(width - 2).Render(fullStatus)
	status := statusBarStyle.Width(width).Render(wrappedMsg)

	return lipgloss.JoinVertical(lipgloss.Left, baseView, status)
}
