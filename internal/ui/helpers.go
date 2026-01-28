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
	if m.Config.LineNumbers {
		textWidth -= 6
	}
	textWidth -= 1
	if textWidth < 1 {
		textWidth = 1
	}

	viewportHeight := m.Height - 2 // -2 for status bar and border

	// 1. Ensure yOffset is within bounds [0, len(Lines)-1]
	if m.yOffset < 0 {
		m.yOffset = 0
	}
	if m.yOffset >= len(m.Lines) {
		m.yOffset = len(m.Lines) - 1
	}

	// 2. If CursorRow is above yOffset, simply scroll up to CursorRow
	if m.CursorRow < m.yOffset {
		m.yOffset = m.CursorRow
	}

	// 3. If CursorRow is below yOffset, we need to make sure it fits in the viewport.
	// We might need to increment yOffset until the cursor is visible.
	// Calculate total visual lines from yOffset to CursorRow

	// Quick check: if CursorRow is WAY far down, jump closer first to avoid huge loop
	if m.CursorRow > m.yOffset+viewportHeight {
		m.yOffset = m.CursorRow - viewportHeight + 1
		if m.yOffset < 0 {
			m.yOffset = 0
		}
	}

	for {
		// Calculate visual height of the range [yOffset, CursorRow]
		totalVisualHeight := 0

		// Optimization: We only care if it EXCEEDS viewportHeight.
		// We can stop summing once we pass it.
		for i := m.yOffset; i < m.CursorRow; i++ {
			totalVisualHeight += m.getVisualLineCount(i, textWidth)
			if totalVisualHeight > viewportHeight {
				break
			}
		}

		// Add the cursor's visual offset within its own line
		cursorVisualLine := m.getCursorVisualOffset(textWidth)

		// The cursor is at `totalVisualHeight + cursorVisualLine` (0-indexed visual row relative to viewport top)
		// We want this value to be < viewportHeight

		cursorPosInViewport := totalVisualHeight + cursorVisualLine

		if cursorPosInViewport < viewportHeight {
			break // It fits!
		}

		// Doesn't fit, scroll down (increment yOffset)
		m.yOffset++

		// Safety break
		if m.yOffset >= len(m.Lines) {
			m.yOffset = len(m.Lines) - 1
			break
		}
		if m.yOffset > m.CursorRow {
			m.yOffset = m.CursorRow
			break
		}
	}

	return m
}
