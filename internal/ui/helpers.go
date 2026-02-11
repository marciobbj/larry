package ui

import (
	"log"
	"os"
)

func Write(errorMessage string) {
	f, err := os.OpenFile("larry.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	defer f.Close()

	log.SetOutput(f)
	log.Println(errorMessage)
}

func debugKeysEnabled() bool {
	return os.Getenv("LARRY_DEBUG_KEYS") != ""
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

// getCursorVisualOffset returns the visual line index of the cursor RELATIVE to the start of the current line.
// To get absolute visual position relative to viewport top, we need to sum visual heights of lines between yOffset and CursorRow.
func (m Model) getCursorVisualOffset(textWidth int) int {
	line := []rune(m.Lines[m.CursorRow])
	currentLineVisualLine := 0
	visualWidth := 0
	for i := 0; i < m.CursorCol && i < len(line); i++ {
		charWidth := 1
		if line[i] == '\t' {
			charWidth = m.Config.TabWidth
		}
		if visualWidth+charWidth > textWidth {
			currentLineVisualLine++
			visualWidth = charWidth
		} else {
			visualWidth += charWidth
		}
	}
	return currentLineVisualLine
}

func (m Model) updateViewport() Model {
	textWidth := m.Width
	viewportHeight := m.Height - 1

	if m.viewMode == ViewModeSplit {
		textWidth = m.Width / 2
		viewportHeight = m.Height - 1
	}

	if m.saving || m.goToLine || m.searching || m.replacing {
		viewportHeight -= 2
	}
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	if m.Config.LineNumbers {
		textWidth -= 6
	}
	textWidth -= 1
	if textWidth < 1 {
		textWidth = 1
	}

	// Calculate absolute visual position of the cursor
	cursorVisualAbsPos := 0
	// 1. Sum visual height of all lines BEFORE CursorRow
	for i := 0; i < m.CursorRow; i++ {
		cursorVisualAbsPos += m.getVisualLineCount(i, textWidth)
	}
	// 2. Add cursor's visual offset within the current line
	cursorVisualAbsPos += m.getCursorVisualOffset(textWidth)

	// Adjust yOffset to keep cursor in view
	// yOffset represents the absolute visual line index at the top of the viewport

	// If cursor is above the viewport, scroll up
	if cursorVisualAbsPos < m.yOffset {
		m.yOffset = cursorVisualAbsPos
	}

	// If cursor is below the viewport, scroll down
	// Visible range is [yOffset, yOffset + viewportHeight)
	// So we need: cursorVisualAbsPos < yOffset + viewportHeight
	// => yOffset > cursorVisualAbsPos - viewportHeight
	if cursorVisualAbsPos >= m.yOffset+viewportHeight {
		m.yOffset = cursorVisualAbsPos - viewportHeight + 1
	}

	// Ensure yOffset is not negative
	if m.yOffset < 0 {
		m.yOffset = 0
	}

	if m.CursorRow == 0 {
		m.yOffset = 0
	}

	return m
}
