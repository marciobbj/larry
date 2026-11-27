package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type editorViewConfig struct {
	width          int
	height         int
	showSearchUI   bool
	hideLeftBorder bool
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

	selStartRow, selStartCol := -1, -1
	selEndRow, selEndCol := -1, -1

	if m.selecting {
		sRow, sCol := m.startRow, m.startCol
		eRow, eCol := m.CursorRow, m.CursorCol

		if sRow > eRow || (sRow == eRow && sCol > eCol) {
			sRow, sCol, eRow, eCol = eRow, eCol, sRow, sCol
		}
		selStartRow, selStartCol = sRow, sCol
		selEndRow, selEndCol = eRow, eCol
	}

	cursorRow := m.CursorRow
	cursorCol := m.CursorCol

	textWidth := cfg.width
	if m.Config.LineNumbers {
		textWidth -= 6
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
		return m.viewEditorNoWrap(cfg, textWidth, maxVisualLines, selStartRow, selStartCol, selEndRow, selEndCol, cursorRow, cursorCol)
	}

	lines := m.Lines
	var s strings.Builder
	visualLinesRendered := 0

	startLine, startVisualOffset := m.locateVisualOffset(m.yOffset, textWidth)

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
				ln := fmt.Sprintf(" %3d ", lineNum+1)
				if isFirst {
					s.WriteString(lineNumStyle.Render(ln))
				} else {
					s.WriteString(lineNumStyle.Render(strings.Repeat(" ", len(ln))))
				}
			}

			syntaxStyles := m.getStylesForLine(lineNum)

			for i := startIdx; i < endIdx; i++ {
				ch := runes[i]
				style, applyStyle := m.resolveCharStyle(lineNum, i, syntaxStyles, selStartRow, selStartCol, selEndRow, selEndCol, cursorRow, cursorCol)
				m.writeStyledChar(&s, ch, lineNum, i, cursorRow, cursorCol, style, applyStyle)
			}

			if !m.selecting && lineNum == cursorRow && cursorCol == len(runes) && endIdx == len(runes) {
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
			charWidth := 1
			if lineRunes[i] == '\t' {
				charWidth = m.Config.TabWidth
			}

			if currentVisualWidth+charWidth > textWidth {
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
	selStartRow, selStartCol, selEndRow, selEndCol int,
	cursorRow, cursorCol int,
) string {
	lines := m.Lines
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
			ln := fmt.Sprintf(" %3d ", lineNum+1)
			s.WriteString(lineNumStyle.Render(ln))
		}

		lineRunes := []rune(lines[lineNum])
		syntaxStyles := m.getStylesForLine(lineNum)

		visCol := 0
		drawn := 0
		for i := 0; i < len(lineRunes); i++ {
			ch := lineRunes[i]
			charWidth := 1
			if ch == '\t' {
				charWidth = tabWidth
			}

			charEnd := visCol + charWidth
			if charEnd <= xOff {
				visCol = charEnd
				continue
			}
			if drawn >= textWidth {
				visCol = charEnd
				break
			}

			style, applyStyle := m.resolveCharStyle(lineNum, i, syntaxStyles, selStartRow, selStartCol, selEndRow, selEndCol, cursorRow, cursorCol)

			if ch == '\t' {
				// Compute only the visible slice of expanded tab spaces
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
					isCursorStart := !m.selecting && lineNum == cursorRow && i == cursorCol && visStart == visCol
					if applyStyle {
						if isCursorStart {
							s.WriteString(styleCursor.Render(" "))
							if sliceLen > 1 {
								s.WriteString(strings.Repeat(" ", sliceLen-1))
							}
						} else {
							s.WriteString(style.Render(strings.Repeat(" ", sliceLen)))
						}
					} else {
						s.WriteString(strings.Repeat(" ", sliceLen))
					}
					drawn += sliceLen
				}
			} else {
				m.writeStyledChar(&s, ch, lineNum, i, cursorRow, cursorCol, style, applyStyle)
				drawn++
			}

			visCol = charEnd
		}

		if !m.selecting && lineNum == cursorRow && cursorCol == len(lineRunes) {
			eolCol := cursorVisualCol(lines[lineNum], len(lineRunes), tabWidth)
			if eolCol >= xOff && eolCol < xOff+textWidth {
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

func (m Model) resolveCharStyle(
	lineNum, i int,
	syntaxStyles []lipgloss.Style,
	selStartRow, selStartCol, selEndRow, selEndCol int,
	cursorRow, cursorCol int,
) (lipgloss.Style, bool) {
	var style lipgloss.Style
	applyStyle := false

	if i < len(syntaxStyles) {
		style = syntaxStyles[i]
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

	if m.selecting {
		isSelected := false
		if lineNum > selStartRow && lineNum < selEndRow {
			isSelected = true
		} else if lineNum == selStartRow && lineNum == selEndRow {
			if i >= selStartCol && i < selEndCol {
				isSelected = true
			}
		} else if lineNum == selStartRow {
			if i >= selStartCol {
				isSelected = true
			}
		} else if lineNum == selEndRow {
			if i < selEndCol {
				isSelected = true
			}
		}
		if isSelected {
			style = styleSelected
			applyStyle = true
		}
	}

	if lineNum == cursorRow && i == cursorCol {
		style = styleCursor
		applyStyle = true
	}

	return style, applyStyle
}

func (m Model) writeStyledChar(
	s *strings.Builder,
	ch rune,
	lineNum, i, cursorRow, cursorCol int,
	style lipgloss.Style,
	applyStyle bool,
) {
	visualChar := string(ch)
	if ch == '\t' {
		visualChar = strings.Repeat(" ", m.Config.TabWidth)
	}

	if applyStyle {
		if !m.selecting && lineNum == cursorRow && i == cursorCol {
			if ch == '\t' {
				s.WriteString(styleCursor.Render(" ") + strings.Repeat(" ", m.Config.TabWidth-1))
			} else {
				s.WriteString(styleCursor.Render(visualChar))
			}
		} else {
			s.WriteString(style.Render(visualChar))
		}
	} else {
		s.WriteString(visualChar)
	}
}
