// internal/ui/model.go
// Package ui defines the main model for the Bubble Tea TUI application.
// It manages the editor's state, including textarea for editing.
package ui

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
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
}

// Model represents the state of the text editor.
type Model struct {
	TextArea  textarea.Model // Text area for editing with cursor support
	Width     int            // Terminal width
	Height    int            // Terminal height
	FileName  string         // Current file name (for save/load)
	KeyMap    KeyMap         // Key bindings for shortcuts
	Quitting  bool           // Flag to indicate if the app is quitting
	startRow  int            // selecting starting row
	startCol  int            // selecting starting col
	selecting bool
}

// InitialModel creates and returns a new initial model.
func InitialModel() Model {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.Placeholder = "Digite algo..."
	ta.Focus()
	//styles.setupStyle()

	return Model{
		TextArea:  ta,
		Width:     80,
		Height:    20,
		FileName:  "untitled.txt",
		KeyMap:    DefaultKeyMap,
		Quitting:  false,
		startRow:  0,
		startCol:  0,
		selecting: false,
	}
}

// getCol returns the current column (character offset) within the logical line.
// This handles wrapped lines correctly by calculating the position in the unwrapped line.
func getCol(ta textarea.Model) int {
	// LineInfo().CharOffset gives the offset within the current wrapped segment
	// We need to calculate the actual offset in the logical line
	li := ta.LineInfo()

	// Get the text width to understand wrapping
	width := ta.Width()
	if ta.ShowLineNumbers {
		width -= 6
	}
	if width < 1 {
		width = 1
	}

	// RowOffset tells us which wrapped row we're on within the current logical line
	// CharOffset tells us the position within that wrapped row
	return li.RowOffset*width + li.CharOffset
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

// getSelectedText returns the currently selected text.
func (m *Model) getSelectedText() string {
	if !m.selecting {
		return ""
	}

	val := m.TextArea.Value()
	if val == "" {
		return ""
	}

	startIdx := getAbsoluteIndex(val, m.startRow, m.startCol)
	endIdx := getAbsoluteIndex(val, getRow(m.TextArea), getCol(m.TextArea))

	if startIdx > endIdx {
		startIdx, endIdx = endIdx, startIdx
	}

	runes := []rune(val)
	if startIdx >= len(runes) {
		return ""
	}
	if endIdx > len(runes) {
		endIdx = len(runes)
	}

	return string(runes[startIdx:endIdx])
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
	handled := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Intercept quit before passing to textarea
		if key.Matches(msg, m.KeyMap.Quit) {
			m.Quitting = true
			return m, tea.Quit
		}
		// Select all text
		if key.Matches(msg, m.KeyMap.SelectAll) {
			m.selecting = true
			m.startRow = 0
			m.startCol = 0
			m.TextArea.CursorEnd()
		}
		// Copy selected text to clipboard
		if key.Matches(msg, m.KeyMap.Copy) && m.selecting {
			text := m.getSelectedText()
			if text != "" {
				clipboard.Write(clipboard.FmtText, []byte(text))
			}
			handled = true
		}
		// Cut selected text (copy + delete)
		if key.Matches(msg, m.KeyMap.Cut) && m.selecting {
			text := m.getSelectedText()
			if text != "" {
				clipboard.Write(clipboard.FmtText, []byte(text))
			}
			m.deleteSelection()
			handled = true
		}
		// Paste from clipboard
		if key.Matches(msg, m.KeyMap.Paste) {
			// If there's a selection, delete it first
			if m.selecting {
				m.deleteSelection()
			}
			// Get text from clipboard and insert it
			data := clipboard.Read(clipboard.FmtText)
			if len(data) > 0 {
				m.TextArea.InsertString(string(data))
			}
			handled = true
		}
		// Handle delete/backspace when text is selected
		if m.selecting && (msg.Type == tea.KeyDelete || msg.Type == tea.KeyBackspace) {
			m.deleteSelection()
			handled = true
		}
		// Handle character input when text is selected - replace selection with typed char
		if m.selecting && msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.deleteSelection()
			// Insert the typed character(s) - let textarea handle it
			handled = false // Let textarea process the input
		}
		if key.Matches(msg, m.KeyMap.MoveSelectionDown) {
			handled = true
			if !m.selecting {
				m.selecting = true
				// Salva a posição de ancoragem
				m.startRow = getRow(m.TextArea)
				m.startCol = getCol(m.TextArea)
			}
			m.TextArea.CursorDown()
		}
		if key.Matches(msg, m.KeyMap.MoveSelectionUp) {
			handled = true
			if !m.selecting {
				m.selecting = true
				// Salva a posição de ancoragem
				m.startRow = getRow(m.TextArea)
				m.startCol = getCol(m.TextArea)
			}
			m.TextArea.CursorUp()
		}
		if key.Matches(msg, m.KeyMap.MoveSelectionLeft) {
			handled = true
			if !m.selecting {
				m.selecting = true
				// Salva a posição de ancoragem
				m.startRow = getRow(m.TextArea)
				m.startCol = getCol(m.TextArea)
			}
			// Move cursor left by setting cursor position
			col := getCol(m.TextArea)
			if col > 0 {
				m.TextArea.SetCursor(col - 1)
			} else {
				// Move to end of previous line
				row := getRow(m.TextArea)
				if row > 0 {
					m.TextArea.CursorUp()
					m.TextArea.CursorEnd()
				}
			}
		}
		if key.Matches(msg, m.KeyMap.MoveSelectionRight) {
			handled = true
			if !m.selecting {
				m.selecting = true
				// Salva a posição de ancoragem
				m.startRow = getRow(m.TextArea)
				m.startCol = getCol(m.TextArea)
			}
			// Move cursor right
			lines := strings.Split(m.TextArea.Value(), "\n")
			row := getRow(m.TextArea)
			col := getCol(m.TextArea)
			if row < len(lines) && col < len(lines[row]) {
				m.TextArea.SetCursor(col + 1)
			} else if row < len(lines)-1 {
				// Move to start of next line
				m.TextArea.CursorDown()
				m.TextArea.CursorStart()
			}
		}
		// Cancel selection on arrow keys without shift
		// Check if it's a plain arrow key (not with shift modifier)
		if !m.selecting {
			// Already not selecting, nothing to do
		} else if msg.Type == tea.KeyLeft || msg.Type == tea.KeyRight || msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
			// Check if shift is NOT pressed - plain arrow keys cancel selection
			if !key.Matches(msg, m.KeyMap.MoveSelectionDown) &&
				!key.Matches(msg, m.KeyMap.MoveSelectionUp) &&
				!key.Matches(msg, m.KeyMap.MoveSelectionLeft) &&
				!key.Matches(msg, m.KeyMap.MoveSelectionRight) {
				m.selecting = false
			}
		}
		// TODO add partial text selection
	case tea.WindowSizeMsg:
		// Update textarea size on terminal resize
		m.Width = msg.Width
		m.Height = msg.Height
		m.TextArea.SetWidth(msg.Width)
		m.TextArea.SetHeight(msg.Height)
	}

	var taCmd tea.Cmd
	if !handled {
		m.TextArea, taCmd = m.TextArea.Update(msg)
	}
	return m, taCmd
}

// View renders the current state of the model as a string.
// It displays the textarea with cursor.
func (m Model) View() string {
	if m.Quitting {
		return "Tchau!\n"
	}
	if m.selecting {
		val := m.TextArea.Value()
		if val == "" {
			return m.TextArea.View()
		}

		startIdx := getAbsoluteIndex(val, m.startRow, m.startCol)
		endIdx := getAbsoluteIndex(val, getRow(m.TextArea), getCol(m.TextArea))

		if startIdx > endIdx {
			startIdx, endIdx = endIdx, startIdx
		}

		// Calculate available width for text (accounting for line numbers)
		textWidth := m.TextArea.Width()
		if m.TextArea.ShowLineNumbers {
			textWidth -= 6 // " 123 " = 6 characters for line numbers
		}
		if textWidth < 10 {
			textWidth = 10
		}

		// Process line by line to properly handle line numbers
		lines := strings.Split(val, "\n")
		var s strings.Builder
		runeIdx := 0

		for lineNum, line := range lines {
			lineRunes := []rune(line)

			// Handle word wrap - break long lines into multiple visual lines
			if len(lineRunes) == 0 {
				// Empty line
				if m.TextArea.ShowLineNumbers {
					ln := fmt.Sprintf(" %3d ", lineNum+1)
					s.WriteString(ln)
				}
				s.WriteString("\n")
				runeIdx++ // Count the \n character
				continue
			}

			// Process the line in chunks of textWidth
			chunkStart := 0
			isFirstChunk := true
			for chunkStart < len(lineRunes) {
				chunkEnd := chunkStart + textWidth
				if chunkEnd > len(lineRunes) {
					chunkEnd = len(lineRunes)
				}

				// Add line number only for the first chunk of each logical line
				if m.TextArea.ShowLineNumbers {
					if isFirstChunk {
						ln := fmt.Sprintf(" %3d ", lineNum+1)
						s.WriteString(ln)
					} else {
						s.WriteString("      ") // Padding for wrapped lines
					}
				}
				isFirstChunk = false

				// Process each character in the chunk
				for i := chunkStart; i < chunkEnd; i++ {
					ch := lineRunes[i]
					charIdx := runeIdx + i
					if charIdx >= startIdx && charIdx < endIdx {
						// Use orange background (color 208) and black text (color 0)
						s.WriteString("\x1b[48;5;208;38;5;0m")
						s.WriteRune(ch)
						s.WriteString("\x1b[0m")
					} else {
						s.WriteRune(ch)
					}
				}
				s.WriteString("\n")
				chunkStart = chunkEnd
			}

			runeIdx += len(lineRunes)
			runeIdx++ // Count the \n character
		}

		return s.String()
	}
	// Render the textarea (cursor is handled by textarea)
	return m.TextArea.View()
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
