package ui

import (
	"log"
	"os"
	"sort"
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

func countVisualLineCount(line string, textWidth int, tabWidth int) int {
	if line == "" {
		return 1
	}

	visualWidth := 0
	count := 1
	for _, r := range line {
		charWidth := 1
		if r == '\t' {
			charWidth = tabWidth
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

func (m *Model) invalidateVisualCache() {
	m.visualLineVersion++
}

func (m *Model) ensureVisualLineCache(textWidth int) {
	if textWidth < 1 {
		textWidth = 1
	}

	if m.visualLineWidth == textWidth &&
		m.visualLineComputed == m.visualLineVersion &&
		len(m.visualLineCounts) == len(m.Lines) &&
		len(m.visualLinePrefix) == len(m.Lines)+1 {
		return
	}

	m.visualLineCounts = make([]int, len(m.Lines))
	m.visualLinePrefix = make([]int, len(m.Lines)+1)

	for i, line := range m.Lines {
		count := countVisualLineCount(line, textWidth, m.Config.TabWidth)
		m.visualLineCounts[i] = count
		m.visualLinePrefix[i+1] = m.visualLinePrefix[i] + count
	}

	m.visualLineWidth = textWidth
	m.visualLineComputed = m.visualLineVersion
}

func (m *Model) cursorVisualAbsPos(textWidth int) int {
	m.ensureVisualLineCache(textWidth)

	if len(m.Lines) == 0 {
		return 0
	}

	row := m.CursorRow
	if row < 0 {
		row = 0
	}
	if row >= len(m.Lines) {
		row = len(m.Lines) - 1
	}

	return m.visualLinePrefix[row] + m.getCursorVisualOffset(textWidth)
}

func (m *Model) locateVisualOffset(offset int, textWidth int) (int, int) {
	m.ensureVisualLineCache(textWidth)

	if len(m.Lines) == 0 {
		return 0, 0
	}
	if offset <= 0 {
		return 0, 0
	}

	total := m.visualLinePrefix[len(m.visualLinePrefix)-1]
	if total <= 0 {
		return 0, 0
	}
	if offset >= total {
		last := len(m.Lines) - 1
		return last, m.visualLineCounts[last] - 1
	}

	row := sort.Search(len(m.visualLineCounts), func(i int) bool {
		return m.visualLinePrefix[i+1] > offset
	})
	if row >= len(m.visualLineCounts) {
		row = len(m.visualLineCounts) - 1
	}

	return row, offset - m.visualLinePrefix[row]
}

// getCursorVisualOffset returns the visual line index of the cursor RELATIVE to the start of the current line.
// To get absolute visual position relative to viewport top, we need to sum visual heights of lines between yOffset and CursorRow.
func (m Model) getCursorVisualOffset(textWidth int) int {
	if len(m.Lines) == 0 {
		return 0
	}

	row := m.CursorRow
	if row < 0 {
		row = 0
	}
	if row >= len(m.Lines) {
		row = len(m.Lines) - 1
	}

	line := []rune(m.Lines[row])
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
	viewportHeight := m.editorViewportHeight()

	if m.viewMode == ViewModeSplit {
		textWidth = m.Width / 2
	}

	if m.Config.LineNumbers {
		textWidth -= 6
	}
	textWidth -= 1
	if textWidth < 1 {
		textWidth = 1
	}

	// Calculate absolute visual position of the cursor.
	cursorVisualAbsPos := m.cursorVisualAbsPos(textWidth)

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
