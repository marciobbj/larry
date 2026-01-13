package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
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

func (m Model) getSelectedText() string {
	if !m.selecting {
		return ""
	}

	startRow, startCol := m.startRow, m.startCol
	endRow, endCol := m.CursorRow, m.CursorCol

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
	line := []rune(m.Lines[startRow])
	if startCol < len(line) {
		builder.WriteString(string(line[startCol:]))
	}
	builder.WriteString("\n")

	for i := startRow + 1; i < endRow; i++ {
		builder.WriteString(m.Lines[i])
		builder.WriteString("\n")
	}

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

	m.Modified = true

	startRow, startCol := m.startRow, m.startCol
	endRow, endCol := m.CursorRow, m.CursorCol

	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	if startRow == endRow {
		line := []rune(m.Lines[startRow])
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

	startLine := []rune(m.Lines[startRow])
	if startCol > len(startLine) {
		startCol = len(startLine)
	}
	prefix := string(startLine[:startCol])

	endLine := []rune(m.Lines[endRow])
	if endCol > len(endLine) {
		endCol = len(endLine)
	}
	suffix := string(endLine[endCol:])

	m.Lines[startRow] = prefix + suffix
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

	m.Modified = true
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	linesToInsert := strings.Split(text, "\n")

	row := m.CursorRow
	col := m.CursorCol
	if row >= len(m.Lines) {
		row = len(m.Lines) - 1
	}
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
		m.Lines[row] = prefix + linesToInsert[0] + suffix
		m.CursorCol += len([]rune(linesToInsert[0]))
	} else {
		m.Lines[row] = prefix + linesToInsert[0]
		var middleLines []string
		for i := 1; i < len(linesToInsert)-1; i++ {
			middleLines = append(middleLines, linesToInsert[i])
		}

		lastInsertLine := linesToInsert[len(linesToInsert)-1]
		lastLineContent := lastInsertLine + suffix

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
	m.RedoStack = nil
}

func (m Model) undo() Model {
	if len(m.UndoStack) == 0 {
		m.statusMsg = "Nothing to undo"
		return m
	}

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
