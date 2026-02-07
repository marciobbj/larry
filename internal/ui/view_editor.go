package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type editorViewConfig struct {
	width        int
	height       int
	showSearchUI bool
}

func (m Model) viewEditor(cfg editorViewConfig) string {
	if len(m.Lines) == 0 {
		s := strings.Builder{}
		s.WriteString(borderStyle.Render("│"))
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
	textWidth -= 1
	if textWidth < 1 {
		textWidth = 1
	}

	lines := m.Lines
	var s strings.Builder

	maxVisualLines := cfg.height
	visualLinesRendered := 0
	currentVisualLineIndex := 0

	if cfg.showSearchUI {
		maxVisualLines -= 2
	}

	for lineNum := 0; lineNum < len(lines) && visualLinesRendered < maxVisualLines; lineNum++ {
		line := lines[lineNum]
		lineRunes := []rune(line)

		renderChunk := func(runes []rune, startIdx, endIdx int, isFirst bool, _ int) {
			if visualLinesRendered >= maxVisualLines {
				return
			}
			if currentVisualLineIndex < m.yOffset {
				currentVisualLineIndex++
				return
			}

			s.WriteString(borderStyle.Render("│"))
			if m.Config.LineNumbers {
				if isFirst {
					ln := fmt.Sprintf(" %3d ", lineNum+1)
					s.WriteString(lineNumStyle.Render(ln))
				} else {
					s.WriteString(lineNumStyle.Render("      "))
				}
			}

			syntaxStyles := GetLineStyles(m.Lines[lineNum], m.FileName)

			for i := startIdx; i < endIdx; i++ {
				ch := runes[i]
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

			if !m.selecting && lineNum == cursorRow && cursorCol == len(runes) && endIdx == len(runes) {
				s.WriteString(styleCursor.Render(" "))
			}

			s.WriteString("\x1b[K")
			visualLinesRendered++
			if visualLinesRendered < maxVisualLines {
				s.WriteString("\n")
			}
			currentVisualLineIndex++
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
