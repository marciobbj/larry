package ui

import (
	"fmt"
	"strconv"
	"strings"

	"larry/internal/git"
	"larry/internal/plugin"

	"github.com/charmbracelet/lipgloss"
)

type editorViewConfig struct {
	width          int
	height         int
	showSearchUI   bool
	hideLeftBorder bool
}

type highlightStyleKey struct {
	fg        string
	bg        string
	bold      bool
	italic    bool
	underline bool
}

type editorRenderState struct {
	cursors         []CursorSelection
	cursorsByRow    map[int][]CursorSelection
	occByRow        map[int][]colSpan
	gitChanges      map[int]git.LineChange
	gutterSigns     map[int]plugin.GutterSign
	diagsByRow      map[int][]plugin.Diagnostic
	highlightsByRow map[int][]plugin.TextHighlight
	uiANSI          compiledUIANSI
}

func newEditorRenderState(m *Model, viewStartRow, viewEndRow int) editorRenderState {
	cursors := m.allCursors()
	byRow := make(map[int][]CursorSelection, len(cursors))
	for _, c := range cursors {
		if c.Row >= viewStartRow && c.Row < viewEndRow {
			byRow[c.Row] = append(byRow[c.Row], c)
		}
	}
	var gutterSigns map[int]plugin.GutterSign
	var diagsByRow map[int][]plugin.Diagnostic
	var highlightsByRow map[int][]plugin.TextHighlight
	if m.plugins != nil && m.FileName != "" {
		absPath := m.activeAbsPath()
		gutterSigns = m.plugins.EffectiveGutterSigns(absPath)
		diagsByRow = m.plugins.DiagnosticsByLine(absPath)
		highlightsByRow = m.plugins.HighlightsByLine(absPath)
	}
	return editorRenderState{
		cursors:         cursors,
		cursorsByRow:    byRow,
		occByRow:        m.getVisibleOccurrences(viewStartRow, viewEndRow),
		gitChanges:      m.gitLineChanges(),
		gutterSigns:     gutterSigns,
		diagsByRow:      diagsByRow,
		highlightsByRow: highlightsByRow,
		uiANSI:          getCompiledUIANSI(),
	}
}

func (rs *editorRenderState) isSelected(row, col int) bool {
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

func (rs *editorRenderState) hasNonSelectingCursorAtEOL(row, lineLen int) bool {
	for _, c := range rs.cursorsByRow[row] {
		if !c.Selecting && c.Col == lineLen {
			return true
		}
	}
	return false
}

func (m *Model) viewEditor(cfg editorViewConfig) string {
	if len(m.Lines) == 0 {
		uiANSI := getCompiledUIANSI()
		s := strings.Builder{}
		if !cfg.hideLeftBorder {
			s.WriteString(uiANSI.borderStr)
		}
		if m.Config.LineNumbers {
			s.WriteString(uiANSI.lineNum.wrap("   1 "))
		}
		s.WriteString(uiANSI.cursorSpace)
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
		return m.viewEditorNoWrap(cfg, textWidth, maxVisualLines, &rs)
	}

	lines := m.Lines
	numDigits := lineNumberDigits(len(lines))
	padTargetWidth := cfg.width
	if !cfg.hideLeftBorder {
		padTargetWidth--
	}
	if m.Config.LineNumbers {
		padTargetWidth -= 1 + numDigits + 1
	}
	if padTargetWidth < textWidth {
		padTargetWidth = textWidth
	}
	tabWidth := m.Config.TabWidth
	if tabWidth < 1 {
		tabWidth = 4
	}
	var s strings.Builder
	s.Grow(maxVisualLines * 128)
	visualLinesRendered := 0

	startLine, startVisualOffset := m.locateVisualOffset(m.yOffset, textWidth)
	endLine := startLine + maxVisualLines + 1
	if endLine > len(lines) {
		endLine = len(lines)
	}
	rs := newEditorRenderState(m, startLine, endLine)
	uiANSI := rs.uiANSI
	blankNumSpaces := strings.Repeat(" ", numDigits+1)

	for lineNum := startLine; lineNum < len(lines) && visualLinesRendered < maxVisualLines; lineNum++ {
		line := lines[lineNum]
		lineRunes := []rune(line)
		lineVisualOffset := 0
		skipVisualLines := 0
		if lineNum == startLine {
			skipVisualLines = startVisualOffset
		}

		syntaxANSI := m.getANSIStylesForLine(lineNum)
		var pSign *plugin.GutterSign
		if rs.gutterSigns != nil {
			if gs, ok := rs.gutterSigns[lineNum]; ok {
				pSign = &gs
			}
		}
		gitChange := rs.gitChanges[lineNum]
		lineHls := rs.highlightsByRow[lineNum]
		lineDiags := rs.diagsByRow[lineNum]
		lineOccs := rs.occByRow[lineNum]
		lineCursors := rs.cursorsByRow[lineNum]

		renderChunk := func(runes []rune, startIdx, endIdx int, isFirst bool, _ int) {
			if visualLinesRendered >= maxVisualLines {
				return
			}
			if lineVisualOffset < skipVisualLines {
				lineVisualOffset++
				return
			}

			if !cfg.hideLeftBorder {
				s.WriteString(uiANSI.borderStr)
			}
			if m.Config.LineNumbers {
				writeGutterANSI(&s, lineNum, numDigits, blankNumSpaces, isFirst, gitChange, pSign, uiANSI)
			}

			rw := lineRunWriter{s: &s}
			drawn := 0
			for i := startIdx; i < endIdx; i++ {
				ch := runes[i]
				style, isDirectCursor := m.resolveCharANSI(lineNum, i, syntaxANSI, lineHls, lineDiags, lineOccs, lineCursors, &rs)
				if ch == '\t' {
					rem := textWidth - drawn
					if rem <= 0 {
						break
					}
					tw := tabWidth
					if tw > rem {
						tw = rem
					}
					rw.writeSpaces(tw, style, isDirectCursor, uiANSI.cursor, uiANSI.canvasBg)
					drawn += tw
					continue
				}
				if ch >= 0x20 && ch <= 0x7e {
					if drawn+1 > textWidth {
						break
					}
					if isDirectCursor {
						rw.setStyle(uiANSI.cursor)
					} else {
						rw.setStyle(style)
					}
					s.WriteByte(byte(ch))
					drawn++
					continue
				}
				visualChar, charWidth := sanitizeDisplayRune(ch)
				if charWidth > 0 && drawn+charWidth > textWidth {
					if drawn == 0 && textWidth >= 1 {
						if isDirectCursor {
							rw.setStyle(uiANSI.cursor)
						} else {
							rw.setStyle(style)
						}
						s.WriteString("·")
						drawn++
					}
					break
				}
				if isDirectCursor {
					rw.setStyle(uiANSI.cursor)
				} else {
					rw.setStyle(style)
				}
				s.WriteString(visualChar)
				drawn += charWidth
			}
			rw.flush()

			if endIdx == len(runes) && rs.hasNonSelectingCursorAtEOL(lineNum, len(runes)) && drawn < textWidth {
				s.WriteString(uiANSI.cursorSpace)
				drawn++
			}

			if uiANSI.paintBg && drawn < padTargetWidth {
				s.WriteString(uiANSI.canvasBg.prefix)
				for pad := padTargetWidth - drawn; pad > 0; pad-- {
					s.WriteByte(' ')
				}
				s.WriteString(uiANSI.canvasBg.suffix)
			}

			s.WriteString(uiANSI.eolClear)
			visualLinesRendered++
			if visualLinesRendered < maxVisualLines {
				s.WriteByte('\n')
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

func (m *Model) viewEditorNoWrap(
	cfg editorViewConfig,
	textWidth int,
	maxVisualLines int,
	rs *editorRenderState,
) string {
	lines := m.Lines
	numDigits := lineNumberDigits(len(lines))
	padTargetWidth := cfg.width
	if !cfg.hideLeftBorder {
		padTargetWidth--
	}
	if m.Config.LineNumbers {
		padTargetWidth -= 1 + numDigits + 1
	}
	if padTargetWidth < textWidth {
		padTargetWidth = textWidth
	}
	uiANSI := rs.uiANSI
	blankNumSpaces := strings.Repeat(" ", numDigits+1)

	var s strings.Builder
	s.Grow(maxVisualLines * 128)

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
			s.WriteString(uiANSI.borderStr)
		}
		if m.Config.LineNumbers {
			var pSign *plugin.GutterSign
			if rs.gutterSigns != nil {
				if gs, ok := rs.gutterSigns[lineNum]; ok {
					pSign = &gs
				}
			}
			writeGutterANSI(&s, lineNum, numDigits, blankNumSpaces, true, rs.gitChanges[lineNum], pSign, uiANSI)
		}

		line := lines[lineNum]
		lineRunes := []rune(line)
		syntaxANSI := m.getANSIStylesForLine(lineNum)
		lineHls := rs.highlightsByRow[lineNum]
		lineDiags := rs.diagsByRow[lineNum]
		lineOccs := rs.occByRow[lineNum]
		lineCursors := rs.cursorsByRow[lineNum]

		rw := lineRunWriter{s: &s}
		visCol := 0
		drawn := 0
		for i := 0; i < len(lineRunes); i++ {
			ch := lineRunes[i]
			var charWidth int
			if ch >= 0x20 && ch <= 0x7e {
				charWidth = 1
			} else if ch == '\t' {
				charWidth = tabWidth
			} else {
				charWidth = displayRuneWidth(ch, tabWidth)
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

			style, isDirectCursor := m.resolveCharANSI(lineNum, i, syntaxANSI, lineHls, lineDiags, lineOccs, lineCursors, rs)

			if ch >= 0x20 && ch <= 0x7e {
				if isDirectCursor {
					rw.setStyle(uiANSI.cursor)
				} else {
					rw.setStyle(style)
				}
				s.WriteByte(byte(ch))
				drawn++
			} else if ch == '\t' {
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
						rw.writeSpaces(sliceLen, style, isCursorStart, uiANSI.cursor, uiANSI.canvasBg)
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
						rw.writeSpaces(rem, style, false, uiANSI.cursor, uiANSI.canvasBg)
						drawn += rem
					}
				} else if w > 0 && drawn+w > textWidth {
					// Wide rune does not fit in the remaining column(s): stop
					// before exceeding textWidth so the terminal never wraps.
					visCol = charEnd
					break
				} else {
					if isDirectCursor {
						rw.setStyle(uiANSI.cursor)
					} else {
						rw.setStyle(style)
					}
					s.WriteString(visualChar)
					drawn += w
				}
			}

			visCol = charEnd
		}
		rw.flush()

		if len(lineCursors) > 0 && rs.hasNonSelectingCursorAtEOL(lineNum, len(lineRunes)) {
			eolCol := cursorVisualCol(line, len(lineRunes), tabWidth)
			if eolCol >= xOff && eolCol < xOff+textWidth && drawn < textWidth {
				s.WriteString(uiANSI.cursorSpace)
				drawn++
			}
		}

		if uiANSI.paintBg && drawn < padTargetWidth {
			s.WriteString(uiANSI.canvasBg.prefix)
			for pad := padTargetWidth - drawn; pad > 0; pad-- {
				s.WriteByte(' ')
			}
			s.WriteString(uiANSI.canvasBg.suffix)
		}

		s.WriteString(uiANSI.eolClear)
		rendered++
		if lineNum+1 < len(lines) && rendered < maxVisualLines {
			s.WriteByte('\n')
		}
	}

	return s.String()
}

func writeGutterANSI(
	s *strings.Builder,
	lineNum, numDigits int,
	blankNumSpaces string,
	isFirst bool,
	change git.LineChange,
	pluginSign *plugin.GutterSign,
	uiANSI compiledUIANSI,
) {
	if isFirst && pluginSign != nil && pluginSign.Text != "" {
		runes := []rune(pluginSign.Text)
		s.WriteString(cachedGutterSignStr(string(runes[0]), pluginSign.Color))
	} else {
		switch change {
		case git.LineAdded:
			s.WriteString(uiANSI.gitAddedStr)
		case git.LineModified:
			s.WriteString(uiANSI.gitModifiedStr)
		case git.LineDeletedBelow:
			if isFirst {
				s.WriteString(uiANSI.gitDeletedStr)
			} else {
				s.WriteString(uiANSI.gitEmptyStr)
			}
		default:
			s.WriteString(uiANSI.gitEmptyStr)
		}
	}

	if uiANSI.lineNum.prefix != "" {
		s.WriteString(uiANSI.lineNum.prefix)
	}
	if isFirst {
		var buf [20]byte
		numBytes := strconv.AppendInt(buf[:0], int64(lineNum+1), 10)
		for pad := numDigits - len(numBytes); pad > 0; pad-- {
			s.WriteByte(' ')
		}
		s.Write(numBytes)
		s.WriteByte(' ')
	} else {
		s.WriteString(blankNumSpaces)
	}
	if uiANSI.lineNum.suffix != "" {
		s.WriteString(uiANSI.lineNum.suffix)
	}
}

func formatGutter(lineNum, numDigits int, isFirst bool, change git.LineChange) string {
	return formatGutterWithSign(lineNum, numDigits, isFirst, change, nil)
}

func formatGutterWithSign(lineNum, numDigits int, isFirst bool, change git.LineChange, pluginSign *plugin.GutterSign) string {
	numStr := fmt.Sprintf("%*d ", numDigits, lineNum+1)
	var signStyled string
	if isFirst && pluginSign != nil && pluginSign.Text != "" {
		runes := []rune(pluginSign.Text)
		glyph := string(runes[0])
		signStyled = lipgloss.NewStyle().
			Foreground(paletteColorByName(pluginSign.Color)).
			Bold(true).
			Render(glyph)
	} else {
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
	}
	if isFirst {
		return signStyled + lineNumStyle.Render(numStr)
	}
	return signStyled + lineNumStyle.Render(strings.Repeat(" ", len(numStr)))
}

func (m *Model) resolveCharANSI(
	lineNum, i int,
	syntaxANSI []ansiStyle,
	lineHls []plugin.TextHighlight,
	lineDiags []plugin.Diagnostic,
	lineOccs []colSpan,
	lineCursors []CursorSelection,
	rs *editorRenderState,
) (ansiStyle, bool) {
	var style ansiStyle

	if i < len(syntaxANSI) {
		style = syntaxANSI[i]
	}
	if style.prefix == "" && style.suffix == "" && rs.uiANSI.paintBg {
		style = rs.uiANSI.defaultText
	}

	if len(lineHls) > 0 {
		for _, h := range lineHls {
			if i >= h.Col && i < h.EndCol {
				style = cachedHighlightANSI(h)
			}
		}
	}

	if len(lineDiags) > 0 {
		for _, d := range lineDiags {
			inSpan := (d.EndCol > d.Col && i >= d.Col && i < d.EndCol) || (d.EndCol <= d.Col && i >= d.Col)
			if inSpan {
				if d.Severity == plugin.SeverityError {
					style = rs.uiANSI.diagError
					break
				} else if d.Severity == plugin.SeverityWarning {
					style = rs.uiANSI.diagWarn
				}
			}
		}
	}

	if len(lineOccs) > 0 {
		for _, sp := range lineOccs {
			if i >= sp.start && i < sp.end {
				style = rs.uiANSI.occurrence
				break
			}
		}
	}

	if len(m.searchResults) > 0 {
		for _, result := range m.searchResults {
			if result.Line == lineNum && i >= result.Col && i < result.Col+result.Length {
				style = rs.uiANSI.search
				break
			}
		}
	}

	if (m.selecting || len(m.extraCursors) > 0) && rs.isSelected(lineNum, i) {
		style = rs.uiANSI.selected
	}

	if len(lineCursors) > 0 {
		for _, c := range lineCursors {
			if c.Col == i {
				return rs.uiANSI.cursor, !c.Selecting
			}
		}
	}

	return style, false
}
