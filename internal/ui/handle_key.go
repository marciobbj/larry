package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.KeyMap.ToggleMarkdownPreview):
		if isMarkdownFile(m.FileName) {
			if m.viewMode == ViewModeSplit {
				m.viewMode = ViewModeEditor
			} else {
				previewWidth := m.Width/2 - 1
				if previewWidth < 20 {
					previewWidth = 20
				}
				if m.markdownRenderer == nil {
					if err := m.initMarkdownRenderer(previewWidth); err != nil {
						m.statusMsg = "Preview error: " + err.Error()
						return m, nil
					}
				}
				m.viewMode = ViewModeSplit
			}
		}
		return m, nil

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

	case key.Matches(msg, m.KeyMap.FileStart):
		m.selecting = false
		m.CursorRow, m.CursorCol = MoveToFileStart()

	case key.Matches(msg, m.KeyMap.FileEnd):
		m.selecting = false
		m.CursorRow, m.CursorCol = MoveToFileEnd(m.Lines)

	case msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace:
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		}
		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			m.Modified = true
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

	case msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete || key.Matches(msg, m.KeyMap.Delete):
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		} else {
			m.Modified = true
			if m.CursorCol > 0 {
				line := []rune(m.Lines[m.CursorRow])
				deletedChar := string(line[m.CursorCol-1])
				m.pushUndo(EditOp{Type: OpDelete, Row: m.CursorRow, Col: m.CursorCol - 1, Text: deletedChar})

				newLine := append(line[:m.CursorCol-1], line[m.CursorCol:]...)
				m.Lines[m.CursorRow] = string(newLine)
				m.CursorCol--
			} else if m.CursorRow > 0 {
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

	case msg.Type == tea.KeyShiftTab:
		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			line := m.Lines[m.CursorRow]
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
				m.Modified = true
				dedentText := line[:spacesToRemove]
				m.pushUndo(EditOp{Type: OpDelete, Row: m.CursorRow, Col: 0, Text: dedentText})
				m.Lines[m.CursorRow] = line[spacesToRemove:]
				m.CursorCol -= spacesToRemove
				if m.CursorCol < 0 {
					m.CursorCol = 0
				}
			}
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		}

		m.Modified = true
		m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: "\n"})

		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			line := []rune(m.Lines[m.CursorRow])
			prefix := line[:m.CursorCol]
			suffix := line[m.CursorCol:]

			m.Lines[m.CursorRow] = string(prefix)
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
