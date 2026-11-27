package ui

import (
	"log"
	"os"
	"sort"
	"strconv"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// isUnsafeTerminalRune reports whether r is a control, bidi/format, or
// separator rune that must never be written raw to the terminal because it
// can move the cursor (e.g. \r, \b, \x0b, \x0c), inject ANSI escape sequences
// (\x1b), or corrupt terminal column layout.
func isUnsafeTerminalRune(r rune) bool {
	if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
		return true
	}
	if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
		return true
	}
	return false
}

// sanitizeDisplayRune converts a non-tab rune into a terminal-safe printable
// string and returns its exact terminal cell width.
func sanitizeDisplayRune(r rune) (string, int) {
	// Fast path for printable ASCII
	if r >= 0x20 && r <= 0x7e {
		return string(r), 1
	}
	if r == '\t' {
		return " ", 1
	}
	if isUnsafeTerminalRune(r) {
		return "·", 1
	}
	if r == unicode.ReplacementChar {
		return "\uFFFD", 1
	}
	w := runewidth.RuneWidth(r)
	if w <= 0 {
		if unicode.Is(unicode.Mn, r) {
			return string(r), 0
		}
		return "·", 1
	}
	return string(r), w
}

// displayRuneWidth returns the exact number of terminal cells occupied by r.
func displayRuneWidth(r rune, tabWidth int) int {
	if r == '\t' {
		if tabWidth < 1 {
			return 4
		}
		return tabWidth
	}
	if r >= 0x20 && r <= 0x7e {
		return 1
	}
	_, w := sanitizeDisplayRune(r)
	return w
}

// lineNumberDigits returns the digit width used to format line numbers
// (minimum 3 so lines 1..999 keep compact 3-digit padding).
func lineNumberDigits(lineCount int) int {
	if lineCount < 1000 {
		return 3
	}
	return len(strconv.Itoa(lineCount))
}

// lineNumberGutterWidth returns the total columns reserved for the line-number
// gutter (1 git sign + digits + 1 trailing space, minimum 6).
func lineNumberGutterWidth(lineCount int) int {
	w := 1 + lineNumberDigits(lineCount) + 1
	if w < 6 {
		return 6
	}
	return w
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

func debugKeysEnabled() bool {
	return os.Getenv("LARRY_DEBUG_KEYS") != ""
}

func countVisualLineCount(line string, textWidth int, tabWidth int) int {
	if line == "" {
		return 1
	}
	if tabWidth < 1 {
		tabWidth = 4
	}

	visualWidth := 0
	count := 1
	for _, r := range line {
		charWidth := displayRuneWidth(r, tabWidth)
		if visualWidth > 0 && visualWidth+charWidth > textWidth {
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
	tabWidth := m.Config.TabWidth
	if tabWidth < 1 {
		tabWidth = 4
	}
	currentLineVisualLine := 0
	visualWidth := 0
	for i := 0; i < m.CursorCol && i < len(line); i++ {
		charWidth := displayRuneWidth(line[i], tabWidth)
		if visualWidth > 0 && visualWidth+charWidth > textWidth {
			currentLineVisualLine++
			visualWidth = charWidth
		} else {
			visualWidth += charWidth
		}
	}
	return currentLineVisualLine
}

func (m Model) sidebarWidth() int {
	return m.fileTree.EffectiveWidth(m.Width)
}

func (m Model) mainContentWidth() int {
	sw := m.sidebarWidth()
	if sw > 0 {
		w := m.Width - sw - 1 // 1 column for vertical divider
		if w < 1 {
			return 1
		}
		return w
	}
	if m.Width < 1 {
		return 1
	}
	return m.Width
}

func (m Model) editorTextWidth() int {
	textWidth := m.mainContentWidth()
	if m.pluginPanel.Visible {
		ew := textWidth * 60 / 100
		if ew < 20 {
			ew = textWidth / 2
		}
		textWidth = ew
	} else if m.viewMode == ViewModeSplit {
		textWidth = textWidth / 2
	}

	if m.Config.LineNumbers {
		textWidth -= lineNumberGutterWidth(len(m.Lines))
	}
	if m.sidebarWidth() == 0 {
		textWidth -= 1
	}
	if textWidth < 1 {
		textWidth = 1
	}
	return textWidth
}

func cursorVisualCol(line string, col int, tabWidth int) int {
	if tabWidth < 1 {
		tabWidth = 4
	}
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}
	if col < 0 {
		col = 0
	}
	visCol := 0
	for i := 0; i < col; i++ {
		visCol += displayRuneWidth(runes[i], tabWidth)
	}
	return visCol
}

func (m Model) updateViewport() Model {
	textWidth := m.editorTextWidth()
	viewportHeight := m.editorViewportHeight()

	if len(m.Lines) == 0 {
		m.yOffset = 0
		m.xOffset = 0
		return m
	}

	if !m.Config.WordWrap {
		row := m.CursorRow
		if row < 0 {
			row = 0
		}
		if row >= len(m.Lines) {
			row = len(m.Lines) - 1
		}

		if row < m.yOffset {
			m.yOffset = row
		}
		if row >= m.yOffset+viewportHeight {
			m.yOffset = row - viewportHeight + 1
		}
		if m.yOffset < 0 {
			m.yOffset = 0
		}
		if row == 0 {
			m.yOffset = 0
		}

		visCol := cursorVisualCol(m.Lines[row], m.CursorCol, m.Config.TabWidth)
		if visCol < m.xOffset {
			m.xOffset = visCol
		}
		if visCol >= m.xOffset+textWidth {
			m.xOffset = visCol - textWidth + 1
		}
		if m.xOffset < 0 {
			m.xOffset = 0
		}
		return m
	}

	m.xOffset = 0

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
