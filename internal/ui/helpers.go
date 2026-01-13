package ui

import (
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
)

func getCol(ta textarea.Model) int {
	li := ta.LineInfo()
	return li.StartColumn + li.CharOffset
}

func getRow(ta textarea.Model) int {
	return ta.Line()
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

func getAbsoluteIndex(value string, row, col int) int {
	if value == "" {
		return 0
	}

	lines := strings.Split(value, "\n")

	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}

	runeIndex := 0
	for i := 0; i < row; i++ {
		runeIndex += len([]rune(lines[i])) + 1
	}

	lineRunes := []rune(lines[row])
	if col < 0 {
		col = 0
	}
	if col > len(lineRunes) {
		col = len(lineRunes)
	}

	runeIndex += col

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
	textWidth -= 1
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
