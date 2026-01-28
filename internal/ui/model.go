package ui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"larry/internal/config"
	"larry/internal/search"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	loading            bool
	goToLine           bool
	searching          bool
	replacing          bool
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
	replaceQuery       string
	replaceWith        string
	replaceStep        int // 1: Find, 2: Replace with, 3: Replace loop
	searchResults      []search.SearchMatch
	replaceResults     []search.SearchMatch
	currentResultIndex int
	currReplaceIndex   int
	Modified           bool
	searchCancel       context.CancelFunc
	replaceCancel      context.CancelFunc
}

func InitialModel(filename string, lines []string, cfg config.Config) Model {
	SetTheme(cfg.Theme)
	initStyles()

	ti := textinput.New()
	ti.Placeholder = "..."
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
					viewportHeight := m.Height - 3 // Status bar etc
					m.yOffset = m.CursorRow - (viewportHeight / 2)
					if m.yOffset < 0 {
						m.yOffset = 0
					}
					// Ensure m.yOffset is valid (redundant check but safe)
					if m.yOffset >= len(m.Lines) {
						m.yOffset = len(m.Lines) - 1
					}

					// We don't strictly need updateViewport here if we manually set yOffset correctly,
					// but it handles bounds and bottom-clamping too.
					// However, updateViewport might override our "center" preference
					// if it thinks the cursor is visible "enough" (at the very bottom or top).
					// So let's force the center first, then call updateViewport to fix edge cases.
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

		if len(m.Lines) <= m.Height-1 {
			m.yOffset = 0
		} else {
			maxOffset := len(m.Lines) - (m.Height - 1)
			if m.yOffset > maxOffset {
				m.yOffset = maxOffset
			}
			if m.yOffset < 0 {
				m.yOffset = 0
			}
		}

		m.yOffset = 0
		return m, finderCmd
	}

	return m, nil
}

func (m Model) View() string {
	leader := strings.Title(m.Config.LeaderKey)
	if leader == "" {
		leader = "Leader"
	}

	msg := m.statusMsg
	if msg == "" {
		if len(m.searchResults) > 0 {
			msg = fmt.Sprintf("Search: %s (%d/%d) | %s+h: Help | %s+q: Quit | %s+s: Save | %s+f: Search File | %s+p: Larry Finder",
				m.searchQuery, m.currentResultIndex+1, len(m.searchResults), leader, leader, leader, leader, leader)
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
	statusBarHeight := lipgloss.Height(status)

	if m.Quitting {
		return "Tchau!\n"
	}

	baseView := ""

	if len(m.Lines) == 0 && !m.loading {
		s := strings.Builder{}
		s.WriteString(borderStyle.Render("│"))
		if m.Config.LineNumbers {
			s.WriteString(lineNumStyle.Render("   1 "))
		}
		s.WriteString(styleCursor.Render(" ")) // Cursor
		s.WriteString("\n")
		baseView = s.String()
	} else {
		selStartRow, selStartCol := -1, -1
		selEndRow, selEndCol := -1, -1

		if m.selecting {
			sRow, sCol := m.startRow, m.startCol
			eRow, eCol := m.CursorRow, m.CursorCol

			if sRow > eRow || (sRow == eRow && sCol > eCol) {
				sRow, sCol, eRow, eCol = eRow, eCol, sRow, sCol
			}
			selStartRow, selStartCol = sRow, sCol
			selEndRow, selEndCol = eRow, eCol
		}

		cursorRow := m.CursorRow
		cursorCol := m.CursorCol

		textWidth := m.Width
		if m.Config.LineNumbers {
			textWidth -= 6
		}
		textWidth -= 1 // Border
		if textWidth < 1 {
			textWidth = 1
		}

		lines := m.Lines
		var s strings.Builder

		maxVisualLines := m.Height - statusBarHeight
		if m.searching || m.saving || m.goToLine || m.replacing {
			maxVisualLines -= 2
		}

		visualLinesRendered := 0

		// Start rendering from m.yOffset (Logical Row Index)
		// We iterate through lines starting from the first visible logical line.
		for lineNum := m.yOffset; lineNum < len(lines) && visualLinesRendered < maxVisualLines; lineNum++ {
			line := lines[lineNum]
			lineRunes := []rune(line)

			renderChunk := func(runes []rune, startIdx, endIdx int, isFirst bool, currentLineVisualWidth int) {
				if visualLinesRendered >= maxVisualLines {
					return
				}

				// No need to skip lines based on visual index anymore,
				// since we start from the correct logical line.
				// Exception: Partial scrolling of wrapped lines is not supported yet
				// (we always snap to top of logical line), so this simplified logic is correct for O(1).

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

				// Helper to find matches for current line
				findMatchesOnLine := func(allMatches []search.SearchMatch, lineNum int) []search.SearchMatch {
					if len(allMatches) == 0 {
						return nil
					}
					// Binary search for the first match on this line
					idx := sort.Search(len(allMatches), func(i int) bool {
						return allMatches[i].Line >= lineNum
					})
					if idx < len(allMatches) && allMatches[idx].Line == lineNum {
						// Found matches, collect all for this line
						var matches []search.SearchMatch
						for i := idx; i < len(allMatches); i++ {
							if allMatches[i].Line != lineNum {
								break
							}
							matches = append(matches, allMatches[i])
						}
						return matches
					}
					return nil
				}

				lineSearchResults := findMatchesOnLine(m.searchResults, lineNum)
				lineReplaceResults := findMatchesOnLine(m.replaceResults, lineNum)

				for i := startIdx; i < endIdx; i++ {
					ch := runes[i]
					var style lipgloss.Style
					applyStyle := false

					if i < len(syntaxStyles) {
						style = syntaxStyles[i]
						applyStyle = true
					}

					if len(lineSearchResults) > 0 {
						for _, result := range lineSearchResults {
							if i >= result.Col && i < result.Col+result.Length {
								style = styleSearch
								applyStyle = true
								break
							}
						}
					}

					if len(lineReplaceResults) > 0 {
						for _, result := range lineReplaceResults {
							if i >= result.Col && i < result.Col+result.Length {
								style = styleSearch
								applyStyle = true
								break
							}
						}
					}

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
								s.WriteString(styleCursor.Render(" ") + strings.Repeat(" ", m.Config.TabWidth-1))
							} else {
								s.WriteString(styleCursor.Render(visualChar))
							}
						} else {
							s.WriteString(style.Render(visualChar))
						}
					} else {
						s.WriteString(visualChar)
					}
				}

				if !m.selecting && lineNum == cursorRow && cursorCol == len(runes) && endIdx == len(runes) {
					s.WriteString(styleCursor.Render(" "))
				}

				s.WriteString("\x1b[K")
				visualLinesRendered++
				if visualLinesRendered < maxVisualLines {
					s.WriteString("\n")
				}
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

	return lipgloss.JoinVertical(lipgloss.Left, baseView, status)
}
