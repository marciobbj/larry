// internal/ui/model.go
// Package ui defines the main model for the Bubble Tea TUI application.
// It manages the editor's state, including textarea for editing.
package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyMap defines key bindings for editor shortcuts.
// This allows easy customization of key mappings.
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
}

// DefaultKeyMap provides the default key bindings.
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
	Redo:               key.NewBinding(key.WithKeys("ctrl+shift+z")),
}

// Global Styles
var (
	styleCursor   = lipgloss.NewStyle().Background(lipgloss.Color("252")).Foreground(lipgloss.Color("0"))
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("0"))
	styleFile     = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	styleDir      = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	lineNumStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
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

// Model represents the state of the text editor.
type Model struct {
	TextArea   textarea.Model // Text area for editing with cursor support
	Width      int            // Terminal width
	Height     int            // Terminal height
	FileName   string         // Current file name (for save/load)
	KeyMap     KeyMap         // Key bindings for shortcuts
	Quitting   bool           // Flag to indicate if the app is quitting
	startRow   int            // selecting starting row
	startCol   int            // selecting starting col
	selecting  bool
	saving     bool             // Is the user currently saving?
	loading    bool             // Is the user currently loading?
	textInput  textinput.Model  // Input for filename
	filePicker filepicker.Model // File picker for opening files
	statusMsg  string           // Status message to display
	yOffset    int              // Vertical scroll offset (viewport)
	Lines      []string         // File content as lines
	CursorRow  int              // Cursor Row
	CursorCol  int              // Cursor Col
	UndoStack  []EditOp
	RedoStack  []EditOp
}

// InitialModel creates and returns a new initial model.
func InitialModel(filename string, content string) Model {
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
	fp.Height = 10
	fp.ShowHidden = true

	// Define styles using lipgloss
	styleCursor := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	styleSelected := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleFile := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	styleDir := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	// Apply styles (manually or via a dedicated styles struct if available in your version)
	// For basic version, we rely on default styles or set specific callbacks if needed.
	// Bubbles filepicker has specific fields for Styles. Let's set them if strictly required or rely on defaults.
	// Using basic setups for now.
	fp.Styles.Cursor = styleCursor
	fp.Styles.Selected = styleSelected
	fp.Styles.File = styleFile
	fp.Styles.Directory = styleDir

	return Model{
		TextArea:   ta,
		Width:      80,
		Height:     20,
		FileName:   filename,
		KeyMap:     DefaultKeyMap,
		Quitting:   false,
		startRow:   0,
		startCol:   0,
		selecting:  false,
		textInput:  ti,
		saving:     false,
		loading:    false,
		filePicker: fp,
		Lines:      strings.Split(content, "\n"),
		CursorRow:  0,
		CursorCol:  0,
	}
}

// getCol returns the current column (character offset) within the logical line.
func getCol(ta textarea.Model) int {
	li := ta.LineInfo()
	// StartColumn is the logical index where the current wrapped line starts.
	// CharOffset is the offset within this wrapped line.
	// Together they give the correct logical column index.
	return li.StartColumn + li.CharOffset
}

// getRow returns the current row (line number) of the cursor using the public API.
func getRow(ta textarea.Model) int {
	return ta.Line()
}

// deleteSelection removes the selected text and positions cursor at start of selection.
// Returns true if text was deleted, false if there was no selection.
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
	// CursorStart moves to start of current line, we need to go to line 0 first
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

	// Clear selection state
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

// Init initializes the model. No initial commands needed.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
// It intercepts quit commands and delegates other input to the textarea.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// Handle Loading Mode (Open)
	// Handle Loading Mode (Open)
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

	// Handle Save Mode
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
		m.TextArea.SetHeight(msg.Height)
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

// View renders the current state of the model as a string.
// It displays the textarea with cursor.
func (m Model) View() string {
	if m.Quitting {
		return "Tchau!\n"
	}

	baseView := ""

	// Use Optimized Custom View for Editor
	// val := m.TextArea.Value() // Removing redundant O(N) access

	// If empty, show placeholder or empty cursor
	// Using len(m.Lines) check instead of val == ""
	if len(m.Lines) == 0 && !m.loading {
		// Just render one empty line with cursor
		s := strings.Builder{}
		s.WriteString(borderStyle.Render("│"))
		if m.TextArea.ShowLineNumbers {
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
		if m.TextArea.ShowLineNumbers {
			textWidth -= 6
		}
		textWidth -= 1 // Border
		if textWidth < 1 {
			textWidth = 1
		}

		lines := m.Lines // Direct access! O(1) assignment.
		var s strings.Builder

		endLine := m.yOffset + m.TextArea.Height()
		if endLine > len(lines) {
			endLine = len(lines)
		}

		startLine := m.yOffset
		if startLine < 0 {
			startLine = 0
		}
		if startLine > len(lines) {
			startLine = len(lines)
		}

		for lineNum := startLine; lineNum < endLine; lineNum++ {
			line := lines[lineNum]
			lineRunes := []rune(line)

			// Process line in chunks
			chunkStart := 0
			isFirstChunk := true

			// Handle empty line case explicitly
			if len(lineRunes) == 0 {
				s.WriteString(borderStyle.Render("│"))
				if m.TextArea.ShowLineNumbers {
					ln := fmt.Sprintf(" %3d ", lineNum+1)
					s.WriteString(lineNumStyle.Render(ln))
				}
				// Render cursor if on this line
				if lineNum == cursorRow && !m.selecting {
					s.WriteString(styleCursor.Render(" "))
				}
				s.WriteString("\n")
				continue
			}

			for chunkStart < len(lineRunes) {
				chunkEnd := chunkStart + textWidth
				if chunkEnd > len(lineRunes) {
					chunkEnd = len(lineRunes)
				}

				s.WriteString(borderStyle.Render("│"))
				if m.TextArea.ShowLineNumbers {
					if isFirstChunk {
						ln := fmt.Sprintf(" %3d ", lineNum+1)
						s.WriteString(lineNumStyle.Render(ln))
					} else {
						s.WriteString(lineNumStyle.Render("      "))
					}
				}
				isFirstChunk = false

				// Syntax Highlighting
				syntaxStyles := GetLineStyles(m.Lines[lineNum], m.FileName)

				for i := chunkStart; i < chunkEnd; i++ {
					ch := lineRunes[i]

					var style lipgloss.Style
					applyStyle := false

					// Apply Syntax Style (Base)
					if i < len(syntaxStyles) {
						style = syntaxStyles[i]
						applyStyle = true
					}

					// Selection Logic (Row/Col based)
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

					// Cursor Logic
					if !m.selecting && lineNum == cursorRow && i == cursorCol {
						style = styleCursor
						applyStyle = true
					}

					// Define "visual" char (expand tab)
					visualChar := string(ch)
					if ch == '\t' {
						visualChar = "    "
					}

					if applyStyle {
						// Manual ANSI for cursor (White BG, Black FG)
						if !m.selecting && lineNum == cursorRow && i == cursorCol {
							if ch == '\t' {
								// For tabs, only highlight the first space to keep cursor width consistent (1 cell)
								s.WriteString("\x1b[47m\x1b[30m \x1b[0m   ")
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

				// Render cursor at end of line?
				// Use manual ANSI here too
				if !m.selecting && lineNum == cursorRow && cursorCol == len(lineRunes) && chunkEnd == len(lineRunes) {
					s.WriteString("\x1b[47m\x1b[30m \x1b[0m")
				}

				// Clear to end of line to remove artifacts
				s.WriteString("\x1b[K")
				s.WriteString("\n")
				chunkStart = chunkEnd
			}
		}
		baseView = s.String()
	}

	if m.saving {
		return fmt.Sprintf("%s\n\n%s", baseView, m.textInput.View())
	}
	if m.loading {
		// Calculate empty lines to push picker to bottom
		pickerView := m.filePicker.View()

		// Create a full-height string for the layout
		return lipgloss.Place(
			m.Width, m.Height,
			lipgloss.Left, lipgloss.Bottom,
			pickerView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")), // transparent/black
		)
	}
	/*
		if m.statusMsg != "" {
			return fmt.Sprintf("%s\n\n%s", baseView, m.statusMsg)
		}
	*/

	return baseView
}

// handleKey handles key messages manually
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	// Quit
	case key.Matches(msg, m.KeyMap.Quit):
		m.Quitting = true
		return m, tea.Quit

	// Save
	case key.Matches(msg, m.KeyMap.Save):
		m.saving = true
		m.textInput.Focus()
		m.textInput.SetValue(m.FileName)
		return m, nil

	// Open
	// Open
	// Open
	case key.Matches(msg, m.KeyMap.Open):
		m.loading = true
		m.filePicker.CurrentDirectory, _ = os.Getwd()
		return m, m.filePicker.Init()

	// Undo
	case key.Matches(msg, m.KeyMap.Undo):
		m = m.undo()
		return m, nil

	// Redo
	case key.Matches(msg, m.KeyMap.Redo):
		m = m.redo()
		return m, nil

	// Select All
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

	// Cut
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

	// Copy
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

	// Paste
	// Paste
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

	// Cursor Movement & Selection

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

// getAbsoluteIndex returns the absolute rune index for a given row and col.
// It performs bounds checking to avoid index out of range errors.
// This function works with rune indices (character positions), not byte indices.
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

// updateViewport adjusts the yOffset to keep the cursor in view.
func (m Model) updateViewport() Model {
	if m.CursorRow < m.yOffset { // check top
		m.yOffset = m.CursorRow
	}
	// m.TextArea.Height() logic might be off if we don't subtract status bar?
	// But let's stick to previous logic.
	if m.CursorRow >= m.yOffset+m.TextArea.Height() {
		m.yOffset = m.CursorRow - m.TextArea.Height() + 1
	}
	return m
}

// getSelectedText returns the text currently selected
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

// deleteSelectedText deletes the selected text and updates the cursor (returns modified model)
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

// clipboardWrite writes text to the system clipboard
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

// clipboardRead reads text from the system clipboard
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

// pushUndo records an operation to the undo stack and clears the redo stack
func (m *Model) pushUndo(op EditOp) {
	m.UndoStack = append(m.UndoStack, op)
	m.RedoStack = nil // Clear redo stack on new operation
}

// undo reverts the last operation
func (m Model) undo() Model {
	if len(m.UndoStack) == 0 {
		m.statusMsg = "Nothing to undo"
		return m
	}

	// Pop
	op := m.UndoStack[len(m.UndoStack)-1]
	m.UndoStack = m.UndoStack[:len(m.UndoStack)-1]

	// Inverse Operation
	var inverseOp EditOp
	inverseOp.Row = op.Row
	inverseOp.Col = op.Col
	inverseOp.Text = op.Text

	switch op.Type {
	case OpInsert:
		// To undo insertion, we delete the inserted text
		// We use `deleteSelection` logic manually or similar.
		// Since `op.Text` can be multi-line or single line.
		inverseOp.Type = OpDelete

		// Setup selection to delete
		m.startRow = op.Row
		m.startCol = op.Col

		// Calculate end position based on op.Text
		lines := strings.Split(op.Text, "\n")
		if len(lines) == 1 {
			m.CursorRow = op.Row
			m.CursorCol = op.Col + len([]rune(lines[0]))
		} else {
			m.CursorRow = op.Row + len(lines) - 1
			m.CursorCol = len([]rune(lines[len(lines)-1]))
		}

		m.selecting = true
		m = m.deleteSelectedText() // This modifies m.Lines
		m.selecting = false

	case OpDelete:
		// To undo deletion, we insert the deleted text
		inverseOp.Type = OpInsert
		m.CursorRow = op.Row
		m.CursorCol = op.Col
		m = m.insertTextAtCursor(op.Text)
	}

	// Push to Redo
	m.RedoStack = append(m.RedoStack, inverseOp)
	m.statusMsg = "Undid change"
	return m
}

// redo re-applies the last undone operation
func (m Model) redo() Model {
	if len(m.RedoStack) == 0 {
		m.statusMsg = "Nothing to redo"
		return m
	}

	// Pop
	op := m.RedoStack[len(m.RedoStack)-1]
	m.RedoStack = m.RedoStack[:len(m.RedoStack)-1]

	// Apply Operation
	// We need to push the *inverse of this* back to UndoStack?
	// The `op` in RedoStack IS the operation to perform (it was the Inverse of the Undo).
	// So if we perform it, we need to push ITS inverse to UndoStack.
	// Actually, simpler: The Op in RedoStack is "Insert X" or "Delete X".
	// We just execute it. And push it back to UndoStack.
	// BUT `op` contains the info to DO it.

	// Re-construct the Undo Op (which is the same as this Redo Op essentially)
	undoOp := op

	switch op.Type {
	case OpInsert: // Redo an Insert (which was an Undo of Delete)
		m.CursorRow = op.Row
		m.CursorCol = op.Col
		m = m.insertTextAtCursor(op.Text)
		// UndoOp should be Delete
		undoOp.Type = OpInsert // Wait. If we just Inserted, we want to Record "Inserted" so Undo can "Delete" it.
		// Yes. `pushUndo` expects what we DID.

	case OpDelete: // Redo a Delete (which was an Undo of Insert)
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
		undoOp.Type = OpDelete
	}

	m.UndoStack = append(m.UndoStack, undoOp)
	m.statusMsg = "Redid change"
	return m
}
