package ui

import (
	"fmt"
	"strings"

	"larry/internal/git"

	"github.com/charmbracelet/lipgloss"
)

type editorViewConfig struct {
	width          int
	height         int
	showSearchUI   bool
	hideLeftBorder bool
}

type editorRenderState struct {
	cursors     []CursorSelection
	cursorsByRow map[int][]CursorSelection
	occByRow    map[int][]colSpan
	gitChanges  map[int]git.LineChange
}

func newEditorRenderState(m Model, viewStartRow, viewEndRow int) editorRenderState {
	cursors := m.allCursors()
	byRow := make(map[int][]CursorSelection, len(cursors))
	for _, c := range cursors {
		if c.Row >= viewStartRow && c.Row < viewEndRow {
			byRow[c.Row] = append(byRow[c.Row], c)
		}
	}
	return editorRenderState{
		cursors:      cursors,
		cursorsByRow: byRow,
		occByRow:     m.getVisibleOccurrences(viewStartRow, viewEndRow),
		gitChanges:   m.gitLineChanges(),
	}
}

func (rs editorRenderState) isSelected(row, col int) bool {
	// Fast path: check cursors touching this row.
	if len(rs.cursors) <= 8 {
		for _, c := range rs.cursors {
			if !c.Selecting {
				continue
			}
			sr, sc, er, ec := c.normalizedBounds()
			if row < sr || row > er {
				continue
			}
			if row > sr && row < er {
				return true
			}
			if row == sr && row == er {
				if col >= sc && col < ec {
					return true
				}
			} else if row == sr {
				if col >= sc {
					return true
				}
			} else if row == er {
				if col < ec {
					return true
				}
			}
		}
		return false
	}
	// O(k_row) lookup when many cursors exist on a large file.
	for _, c := range rs.cursorsByRow[row] {
		if !c.Selecting {
			continue
		}
		sr, sc, er, ec := c.normalizedBounds()
		if row == sr && row == er && col >= sc && col < ec {
			return true
		}
	}
	return false
}

func (rs editorRenderState) isOccurrence(row, col int) bool {
	for _, sp := range rs.occByRow[row] {
		if col >= sp.start && col < sp.end {
			return true
		}
	}
	return false
}

func (rs editorRenderState) hasCursorAt(row, col int) (isCursor bool, isSelecting bool) {
	for _, c := range rs.cursorsByRow[row] {
		if c.Col == col {
			return true, c.Selecting
		}
	}
	return false, false
}

func (rs editorRenderState) hasNonSelectingCursorAtEOL(row, lineLen int) bool {
	for _, c := range rs.cursorsByRow[row] {
		if !c.Selecting && c.Col == lineLen {
			return true
		}
	}
	return false
}

func (m Model) viewEditor(cfg editorViewConfig) string {
	if len(m.Lines) == 0 {
		s := strings.Builder{}
		if !cfg.hideLeftBorder {
			s.WriteString(borderStyle.Render("│"))
		}
		if m.Config.LineNumbers {
			s.WriteString(lineNumStyle.Render("   1 "))
		}
		s.WriteString(styleCursor.Render(" "))
		s.WriteString("\n")
		return s.String()
	}

	textWidth := cfg.width
	if m.Config.LineNumbers {
		textWidth -= lineNumberGutterWidth(len(m.Lines))
	}
	if !cfg.hideLeftBorder {
		textWidth -= 1
	}
	if textWidth < 1 {
		textWidth = 1
	}

	maxVisualLines := cfg.height
	if cfg.showSearchUI {
		maxVisualLines -= 2
	}
	if maxVisualLines < 1 {
		maxVisualLines = 1
	}

	if !m.Config.WordWrap {
		startLine := clampInt(m.yOffset, 0, len(m.Lines)-1)
		endLine := startLine + maxVisualLines
		if endLine > len(m.Lines) {
			endLine = len(m.Lines)
		}
		rs := newEditorRenderState(m, startLine, endLine)
		return m.viewEditorNoWrap(cfg, textWidth, maxVisualLines, rs)
	}

	lines := m.Lines
	numDigits := lineNumberDigits(len(lines))
	tabWidth := m.Config.TabWidth
	if tabWidth < 1 {
		tabWidth = 4
	}
	var s strings.Builder
	visualLinesRendered := 0

	startLine, startVisualOffset := m.locateVisualOffset(m.yOffset, textWidth)
	endLine := startLine + maxVisualLines + 1
	if endLine > len(lines) {
		endLine = len(lines)
	}
	rs := newEditorRenderState(m, startLine, endLine)

	for lineNum := startLine; lineNum < len(lines) && visualLinesRendered < maxVisualLines; lineNum++ {
		line := lines[lineNum]
		lineRunes := []rune(line)
		lineVisualOffset := 0
		skipVisualLines := 0
		if lineNum == startLine {
			skipVisualLines = startVisualOffset
		}

		renderChunk := func(runes []rune, startIdx, endIdx int, isFirst bool, _ int) {
			if visualLinesRendered >= maxVisualLines {
				return
			}
			if lineVisualOffset < skipVisualLines {
				lineVisualOffset++
				return
			}

			if !cfg.hideLeftBorder {
				s.WriteString(borderStyle.Render("│"))
			}
			if m.Config.LineNumbers {
				s.WriteString(formatGutter(lineNum, numDigits, isFirst, rs.gitChanges[lineNum]))
			}

			syntaxStyles := m.getStylesForLine(lineNum)

			drawn := 0
			for i := startIdx; i < endIdx; i++ {
				ch := runes[i]
				style, applyStyle, isDirectCursor := m.resolveCharStyle(lineNum, i, syntaxStyles, rs)
				if ch == '\t' {
					rem := textWidth - drawn
					if rem <= 0 {
						break
					}
					tw := tabWidth
					if tw > rem {
						tw = rem
					}
					writeStyledSpaces(&s, tw, style, applyStyle, isDirectCursor)
					drawn += tw
					continue
				}
				visualChar, charWidth := sanitizeDisplayRune(ch)
				if charWidth > 0 && drawn+charWidth > textWidth {
					if drawn == 0 && textWidth >= 1 {
						writeStyledVisual(&s, "·", style, applyStyle, isDirectCursor)
						drawn++
					}
					break
				}
				writeStyledVisual(&s, visualChar, style, applyStyle, isDirectCursor)
				drawn += charWidth
			}

			if endIdx == len(runes) && rs.hasNonSelectingCursorAtEOL(lineNum, len(runes)) && drawn < textWidth {
				s.WriteString(styleCursor.Render(" "))
			}

			s.WriteString("\x1b[K")
			visualLinesRendered++
			if visualLinesRendered < maxVisualLines {
				s.WriteString("\n")
			}
			lineVisualOffset++
		}

		if len(lineRunes) == 0 {
			renderChunk(nil, 0, 0, true, 0)
			continue
		}

		chunkStart := 0
		currentVisualWidth := 0
		isFirst := true
		for i := 0; i < len(lineRunes); i++ {
			charWidth := displayRuneWidth(lineRunes[i], tabWidth)

			if currentVisualWidth > 0 && currentVisualWidth+charWidth > textWidth {
				renderChunk(lineRunes, chunkStart, i, isFirst, currentVisualWidth)
				chunkStart = i
				currentVisualWidth = charWidth
				isFirst = false
			} else {
				currentVisualWidth += charWidth
			}
		}
		if chunkStart <= len(lineRunes) {
			renderChunk(lineRunes, chunkStart, len(lineRunes), isFirst, currentVisualWidth)
		}
	}

	return s.String()
}

func (m Model) viewEditorNoWrap(
	cfg editorViewConfig,
	textWidth int,
	maxVisualLines int,
	rs editorRenderState,
) string {
	lines := m.Lines
	numDigits := lineNumberDigits(len(lines))
	var s strings.Builder

	startLine := m.yOffset
	if startLine < 0 {
		startLine = 0
	}
	if startLine >= len(lines) {
		startLine = len(lines) - 1
	}

	xOff := m.xOffset
	if xOff < 0 {
		xOff = 0
	}
	tabWidth := m.Config.TabWidth
	if tabWidth < 1 {
		tabWidth = 4
	}

	rendered := 0
	for lineNum := startLine; lineNum < len(lines) && rendered < maxVisualLines; lineNum++ {
		if !cfg.hideLeftBorder {
			s.WriteString(borderStyle.Render("│"))
		}
		if m.Config.LineNumbers {
			s.WriteString(formatGutter(lineNum, numDigits, true, rs.gitChanges[lineNum]))
		}

		lineRunes := []rune(lines[lineNum])
		syntaxStyles := m.getStylesForLine(lineNum)

		visCol := 0
		drawn := 0
		for i := 0; i < len(lineRunes); i++ {
			ch := lineRunes[i]
			charWidth := displayRuneWidth(ch, tabWidth)

			charEnd := visCol + charWidth
			if charEnd <= xOff {
				visCol = charEnd
				continue
			}
			if drawn >= textWidth {
				visCol = charEnd
				break
			}

			style, applyStyle, isDirectCursor := m.resolveCharStyle(lineNum, i, syntaxStyles, rs)

			if ch == '\t' {
				visStart := visCol
				if visStart < xOff {
					visStart = xOff
				}
				visEnd := charEnd
				if visEnd > xOff+textWidth {
					visEnd = xOff + textWidth
				}
				sliceLen := visEnd - visStart
				if sliceLen > 0 {
					if drawn+sliceLen > textWidth {
						sliceLen = textWidth - drawn
					}
					if sliceLen > 0 {
						isCursorStart := isDirectCursor && visStart == visCol
						writeStyledSpaces(&s, sliceLen, style, applyStyle, isCursorStart)
						drawn += sliceLen
					}
				}
			} else {
				visualChar, w := sanitizeDisplayRune(ch)
				// Wide rune straddling the left horizontal scroll boundary:
				// emit a single filler cell for its visible right half.
				if visCol < xOff && charEnd > xOff {
					visEnd := charEnd
					if visEnd > xOff+textWidth {
						visEnd = xOff + textWidth
					}
					rem := visEnd - xOff
					if rem > textWidth-drawn {
						rem = textWidth - drawn
					}
					if rem > 0 {
						writeStyledSpaces(&s, rem, style, applyStyle, false)
						drawn += rem
					}
				} else if w > 0 && drawn+w > textWidth {
					// Wide rune does not fit in the remaining column(s): stop
					// before exceeding textWidth so the terminal never wraps.
					visCol = charEnd
					break
				} else {
					writeStyledVisual(&s, visualChar, style, applyStyle, isDirectCursor)
					drawn += w
				}
			}

			visCol = charEnd
		}

		if rs.hasNonSelectingCursorAtEOL(lineNum, len(lineRunes)) {
			eolCol := cursorVisualCol(lines[lineNum], len(lineRunes), tabWidth)
			if eolCol >= xOff && eolCol < xOff+textWidth && drawn < textWidth {
				s.WriteString(styleCursor.Render(" "))
			}
		}

		s.WriteString("\x1b[K")
		rendered++
		if rendered < maxVisualLines {
			s.WriteString("\n")
		}
	}

	return s.String()
}

func formatGutter(lineNum, numDigits int, isFirst bool, change git.LineChange) string {
	numStr := fmt.Sprintf("%*d ", numDigits, lineNum+1)
	var signStyled string
	switch change {
	case git.LineAdded:
		signStyled = gitAddedStyle.Render("▎")
	case git.LineModified:
		signStyled = gitModifiedStyle.Render("▎")
	case git.LineDeletedBelow:
		if isFirst {
			signStyled = gitDeletedStyle.Render("▁")
		} else {
			signStyled = lineNumStyle.Render(" ")
		}
	default:
		signStyled = lineNumStyle.Render(" ")
	}
	if isFirst {
		return signStyled + lineNumStyle.Render(numStr)
	}
	return signStyled + lineNumStyle.Render(strings.Repeat(" ", len(numStr)))
}

func (m Model) resolveCharStyle(
	lineNum, i int,
	syntaxStyles []lipgloss.Style,
	rs editorRenderState,
) (lipgloss.Style, bool, bool) {
	var style lipgloss.Style
	applyStyle := false

	if i < len(syntaxStyles) {
		style = syntaxStyles[i]
		applyStyle = true
	}

	if len(rs.occByRow) > 0 && rs.isOccurrence(lineNum, i) {
		style = styleOccurrence
		applyStyle = true
	}

	if len(m.searchResults) > 0 {
		for _, result := range m.searchResults {
			if result.Line == lineNum && i >= result.Col && i < result.Col+result.Length {
				style = styleSearch
				applyStyle = true
				break
			}
		}
	}

	if rs.isSelected(lineNum, i) {
		style = styleSelected
		applyStyle = true
	}

	isCursor, cursorSelecting := rs.hasCursorAt(lineNum, i)
	if isCursor {
		style = styleCursor
		applyStyle = true
	}

	return style, applyStyle, isCursor && !cursorSelecting
}

func writeStyledSpaces(
	s *strings.Builder,
	count int,
	style lipgloss.Style,
	applyStyle bool,
	isCursorStart bool,
) {
	if count <= 0 {
		return
	}
	if applyStyle {
		if isCursorStart {
			s.WriteString(styleCursor.Render(" "))
			if count > 1 {
				s.WriteString(strings.Repeat(" ", count-1))
			}
		} else {
			s.WriteString(style.Render(strings.Repeat(" ", count)))
		}
	} else {
		s.WriteString(strings.Repeat(" ", count))
	}
}

func writeStyledVisual(
	s *strings.Builder,
	visualChar string,
	style lipgloss.Style,
	applyStyle bool,
	isDirectCursor bool,
) {
	if applyStyle {
		if isDirectCursor {
			s.WriteString(styleCursor.Render(visualChar))
		} else {
			s.WriteString(style.Render(visualChar))
		}
	} else {
		s.WriteString(visualChar)
	}
}
