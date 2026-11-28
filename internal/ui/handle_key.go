package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// Built-in Quit & Command Palette always remain accessible.
	if key.Matches(msg, m.KeyMap.Quit) {
		m.Quitting = true
		m.ClosePlugins()
		return m, tea.Quit
	}
	if key.Matches(msg, m.KeyMap.CommandPalette) {
		m = m.openCommandPalette()
		return m, nil
	}

	// Plugin-registered keymaps (e.g. alt+f, alt+k, alt+d, f2, leader+y).
	if m.plugins != nil && m.plugins.HandleKey(m.editorAPI(), msg.String()) {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.KeyMap.ThemePicker):
		m = m.openThemePicker()
		return m, nil

	case key.Matches(msg, m.KeyMap.ToggleMarkdownPreview):
		if isMarkdownFile(m.FileName) {
			if m.viewMode == ViewModeSplit {
				m.viewMode = ViewModeEditor
			} else {
				mainW := m.mainContentWidth()
				previewWidth := mainW - mainW/2 - 1
				if previewWidth < minPreviewWidth {
					m.statusMsg = "Preview error: " + ErrPreviewTooNarrow.Error()
					return m, nil
				}
				if err := m.initMarkdownRenderer(previewWidth); err != nil {
					m.statusMsg = "Preview error: " + err.Error()
					return m, nil
				}
				m.viewMode = ViewModeSplit
			}
			m.invalidateVisualCache()
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.NewTab):
		m = m.newTab()
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

	case msg.Type == tea.KeyEsc:
		if m.hasMultiCursor() {
			m.clearExtraCursors()
			m.statusMsg = "Single cursor"
		} else if m.selecting {
			m.selecting = false
		} else if m.pluginPanel.Visible {
			m.pluginPanel.Visible = false
			m.invalidateVisualCache()
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.SelectNextOccurrence):
		m = m.selectNextOccurrence()
		return m, nil

	case key.Matches(msg, m.KeyMap.SelectAllOccurrences):
		m = m.selectAllOccurrences()
		return m, nil

	case key.Matches(msg, m.KeyMap.AddCursorUp):
		m = m.addCursorAbove()
		return m, nil

	case key.Matches(msg, m.KeyMap.AddCursorDown):
		m = m.addCursorBelow()
		return m, nil

	case key.Matches(msg, m.KeyMap.ToggleWordWrap):
		m.Config.WordWrap = !m.Config.WordWrap
		m.xOffset = 0
		m.invalidateVisualCache()
		if m.Config.WordWrap {
			m.statusMsg = "Word wrap: ON"
		} else {
			m.statusMsg = "Word wrap: OFF"
		}
		m = m.updateViewport()
		return m, nil

	case key.Matches(msg, m.KeyMap.ToggleHelp):
		m.showHelp = !m.showHelp
		m.helpScroll = 0
		return m, nil

	case key.Matches(msg, m.KeyMap.Save):
		if m.FileName != "" {
			m, _ = m.saveCurrentFile(m.FileName)
			return m, nil
		}

		m.saving = true
		m.textInput.Focus()
		m.textInput.SetValue("")
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
		m.replacing = false
		m.replaceResults = nil
		m.textInput.Focus()
		m.textInput.SetValue(m.searchQuery)
		m.textInput.Prompt = "Search: "
		return m, nil
	case key.Matches(msg, m.KeyMap.Replace):
		m.replacing = true
		m.searching = false
		m.searchResults = nil
		m.replaceStep = 1
		m.textInput.Focus()
		m.textInput.SetValue(m.replaceQuery)
		m.textInput.Prompt = "Find: "
		return m, nil

	case key.Matches(msg, m.KeyMap.GlobalReplace):
		m = m.openGlobalReplace()
		return m, nil

	case key.Matches(msg, m.KeyMap.GlobalFinder):
		return m.openGlobalFinder()

	case key.Matches(msg, m.KeyMap.Open):
		m = m.toggleFileTree()
		return m, nil

	case key.Matches(msg, m.KeyMap.FocusTree):
		if !m.fileTree.Visible {
			m = m.toggleFileTree()
		} else {
			m.fileTree.Focused = !m.fileTree.Focused
			if m.fileTree.Focused {
				m.fileTree.Height = m.workspaceHeight()
				m.fileTree.Rebuild()
			}
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.Undo):
		m = m.undo()
		return m, nil

	case key.Matches(msg, m.KeyMap.Redo):
		m = m.redo()
		return m, nil

	case key.Matches(msg, m.KeyMap.SelectAll):
		m.clearExtraCursors()
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
		if m.hasMultiCursor() {
			text := m.multiCopyText()
			if err := writeClipboard(m, text); err != nil {
				m.statusMsg = "Cut Error: " + err.Error()
			} else {
				m = m.multiInsertText("")
				m.statusMsg = "Cut to clipboard"
			}
			return m, nil
		}
		if m.selecting {
			text := m.getSelectedText()
			err := writeClipboard(m, text)
			if err != nil {
				m.statusMsg = "Cut Error: " + err.Error()
			} else {
				m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
				m = m.deleteSelectedText()
				m.statusMsg = "Cut to clipboard"
			}
		} else if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) && m.Lines[m.CursorRow] == "" {
			m = m.deleteEmptyLineAtCursor()
		} else if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) && m.Lines[m.CursorRow] != "" {
			text := m.Lines[m.CursorRow]
			err := writeClipboard(m, text)
			if err != nil {
				m.statusMsg = "Cut Error: " + err.Error()
				return m, nil
			}
			m.markModified()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.CursorRow, Col: 0, Text: text})
			m.Lines[m.CursorRow] = ""
			m.CursorCol = 0
			m.selecting = false
			m.statusMsg = "Cut to clipboard"
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.Copy):
		if m.hasMultiCursor() {
			text := m.multiCopyText()
			if err := m.clipboardWrite(text); err != nil {
				m.statusMsg = "Copy Error: " + err.Error()
			} else {
				m.statusMsg = "Copied to clipboard"
			}
			return m, nil
		}
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
		text, err := readClipboard(m)
		if err != nil {
			m.statusMsg = "Paste Error: " + err.Error()
		} else if m.hasMultiCursor() {
			m = m.multiPaste(text)
			m.statusMsg = "Pasted from clipboard"
		} else {
			m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: text})
			m = m.insertTextAtCursor(text)
			m.statusMsg = "Pasted from clipboard"
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.CursorUp) || key.Matches(msg, m.KeyMap.MoveSelectionUp):
		if m.hasMultiCursor() {
			m = m.multiMove(func(lines []string, row, col int) (int, int) {
				if row > 0 {
					row--
					col = clampInt(col, 0, lineRuneLen(lines, row))
				}
				return row, col
			}, key.Matches(msg, m.KeyMap.MoveSelectionUp))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(func(lines []string, row, col int) (int, int) {
				if row < len(lines)-1 {
					row++
					col = clampInt(col, 0, lineRuneLen(lines, row))
				}
				return row, col
			}, key.Matches(msg, m.KeyMap.MoveSelectionDown))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(func(lines []string, row, col int) (int, int) {
				if col > 0 {
					col--
				} else if row > 0 {
					row--
					col = lineRuneLen(lines, row)
				}
				return row, col
			}, key.Matches(msg, m.KeyMap.MoveSelectionLeft))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(func(lines []string, row, col int) (int, int) {
				if col < lineRuneLen(lines, row) {
					col++
				} else if row < len(lines)-1 {
					row++
					col = 0
				}
				return row, col
			}, key.Matches(msg, m.KeyMap.MoveSelectionRight))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(FindNextWordBoundary, key.Matches(msg, m.KeyMap.SelectWordRight))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(FindPrevWordBoundary, key.Matches(msg, m.KeyMap.SelectWordLeft))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(JumpLinesUp, key.Matches(msg, m.KeyMap.SelectLinesUp))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(JumpLinesDown, key.Matches(msg, m.KeyMap.SelectLinesDown))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(func(_ []string, row, col int) (int, int) {
				return MoveToLineStart(row, col)
			}, key.Matches(msg, m.KeyMap.SelectToLineStart))
			return m, nil
		}
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
		if m.hasMultiCursor() {
			m = m.multiMove(func(lines []string, row, _ int) (int, int) {
				return MoveToLineEnd(lines, row)
			}, key.Matches(msg, m.KeyMap.SelectToLineEnd))
			return m, nil
		}
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
		m.clearExtraCursors()
		m.selecting = false
		m.CursorRow, m.CursorCol = MoveToFileStart()

	case key.Matches(msg, m.KeyMap.FileEnd):
		m.clearExtraCursors()
		m.selecting = false
		m.CursorRow, m.CursorCol = MoveToFileEnd(m.Lines)

	case !msg.Alt && (msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace):
		if m.hasMultiCursor() {
			text := string(msg.Runes)
			if msg.Type == tea.KeySpace {
				text = " "
			}
			m = m.multiInsertText(text)
			return m, nil
		}
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		}
		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			m.markModified()
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
		if m.hasMultiCursor() {
			m = m.multiBackspace()
			return m, nil
		}
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		} else {
			m.markModified()
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
		if m.hasMultiCursor() {
			m = m.multiInsertText(strings.Repeat(" ", m.Config.TabWidth))
			return m, nil
		}
		if m.selecting {
			m.markModified()
			startRow, _ := m.startRow, m.startCol
			endRow, _ := m.CursorRow, m.CursorCol
			if startRow > endRow {
				startRow, endRow = endRow, startRow
			}

			tab := strings.Repeat(" ", m.Config.TabWidth)
			for i := startRow; i <= endRow; i++ {
				m.pushUndo(EditOp{Type: OpInsert, Row: i, Col: 0, Text: tab})
				line := m.Lines[i]
				m.Lines[i] = tab + line
			}
			m.startCol += len(tab)
			m.CursorCol += len(tab)
			return m, nil
		}

		tab := strings.Repeat(" ", m.Config.TabWidth)
		m.pushUndo(EditOp{Type: OpInsert, Row: m.CursorRow, Col: m.CursorCol, Text: tab})
		m = m.insertTextAtCursor(tab)
		return m, nil

	case msg.Type == tea.KeyShiftTab:
		if m.selecting {
			m.markModified()
			startRow, _ := m.startRow, m.startCol
			endRow, _ := m.CursorRow, m.CursorCol
			if startRow > endRow {
				startRow, endRow = endRow, startRow
			}

			for i := startRow; i <= endRow; i++ {
				line := m.Lines[i]
				spacesToRemove := 0
				for j, r := range line {
					if j >= m.Config.TabWidth {
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
					m.pushUndo(EditOp{Type: OpDelete, Row: i, Col: 0, Text: dedentText})
					m.Lines[i] = line[spacesToRemove:]
					if i == m.startRow {
						m.startCol -= spacesToRemove
						if m.startCol < 0 {
							m.startCol = 0
						}
					}
					if i == m.CursorRow {
						m.CursorCol -= spacesToRemove
						if m.CursorCol < 0 {
							m.CursorCol = 0
						}
					}
				}
			}
			return m, nil
		}
		if m.CursorRow >= 0 && m.CursorRow < len(m.Lines) {
			line := m.Lines[m.CursorRow]
			spacesToRemove := 0
			for i, r := range line {
				if i >= m.Config.TabWidth {
					break
				}
				if r == ' ' {
					spacesToRemove++
				} else {
					break
				}
			}

			if spacesToRemove > 0 {
				m.markModified()
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
		if m.hasMultiCursor() {
			m = m.multiInsertText("\n")
			return m, nil
		}
		if m.selecting {
			text := m.getSelectedText()
			m.pushUndo(EditOp{Type: OpDelete, Row: m.startRow, Col: m.startCol, Text: text})
			m = m.deleteSelectedText()
		}

		m.markModified()
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
