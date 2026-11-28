package ui

import (
	"fmt"
	"strings"
	"unicode"

	"larry/internal/plugin"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

const (
	completionMaxVisibleRows = 8
	completionMinBoxWidth    = 26
	completionMaxBoxWidth    = 58
)

// CompletionPopup holds the state for Larry's floating, cursor-anchored
// autocomplete menu populated by LSP/plugin providers.
type CompletionPopup struct {
	Visible  bool
	AllItems []plugin.CompletionItem
	Filtered []plugin.CompletionItem
	StartRow int // 0-based buffer row where completion was opened
	StartCol int // 0-based rune column where identifier prefix starts
	Cursor   int
	Scroll   int
	OnSelect func(plugin.CompletionItem)
}

func (c *CompletionPopup) Open(items []plugin.CompletionItem, startRow, startCol int, prefix string, onSelect func(plugin.CompletionItem)) {
	if len(items) == 0 {
		c.Close()
		return
	}
	c.AllItems = append([]plugin.CompletionItem(nil), items...)
	c.StartRow = startRow
	c.StartCol = startCol
	c.Cursor = 0
	c.Scroll = 0
	c.OnSelect = onSelect
	c.Visible = true
	c.Filter(prefix)
}

func (c *CompletionPopup) Close() {
	c.Visible = false
	c.AllItems = nil
	c.Filtered = nil
	c.Cursor = 0
	c.Scroll = 0
	c.OnSelect = nil
}

// Filter narrows AllItems against prefix (ranking prefix matches ahead of
// substring matches) and hides the popup when no items match.
func (c *CompletionPopup) Filter(prefix string) {
	if !c.Visible {
		return
	}
	q := strings.ToLower(strings.TrimSpace(prefix))
	if q == "" {
		c.Filtered = append([]plugin.CompletionItem(nil), c.AllItems...)
	} else {
		var prefixMatches []plugin.CompletionItem
		var containsMatches []plugin.CompletionItem
		for _, it := range c.AllItems {
			low := strings.ToLower(it.Label)
			if strings.HasPrefix(low, q) {
				prefixMatches = append(prefixMatches, it)
			} else if strings.Contains(low, q) {
				containsMatches = append(containsMatches, it)
			}
		}
		c.Filtered = make([]plugin.CompletionItem, 0, len(prefixMatches)+len(containsMatches))
		c.Filtered = append(c.Filtered, prefixMatches...)
		c.Filtered = append(c.Filtered, containsMatches...)
	}

	if len(c.Filtered) == 0 {
		c.Close()
		return
	}
	if c.Cursor >= len(c.Filtered) {
		c.Cursor = len(c.Filtered) - 1
	}
	if c.Cursor < 0 {
		c.Cursor = 0
	}
	c.Scroll = 0
}

func (c *CompletionPopup) MoveUp() {
	if len(c.Filtered) == 0 {
		return
	}
	if c.Cursor > 0 {
		c.Cursor--
	} else {
		c.Cursor = len(c.Filtered) - 1
	}
}

func (c *CompletionPopup) MoveDown() {
	if len(c.Filtered) == 0 {
		return
	}
	c.Cursor = (c.Cursor + 1) % len(c.Filtered)
}

func (c *CompletionPopup) Selected() (plugin.CompletionItem, bool) {
	if !c.Visible || len(c.Filtered) == 0 || c.Cursor < 0 || c.Cursor >= len(c.Filtered) {
		return plugin.CompletionItem{}, false
	}
	return c.Filtered[c.Cursor], true
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func findIdentPrefixStartCol(line string, cursorCol int) int {
	runes := []rune(line)
	if cursorCol > len(runes) {
		cursorCol = len(runes)
	}
	if cursorCol < 0 {
		return 0
	}
	i := cursorCol
	for i > 0 && isIdentRune(runes[i-1]) {
		i--
	}
	return i
}

// syncCompletionPrefix re-filters or dismisses the completion popup after an
// in-place buffer edit (typing identifier chars or backspace).
func (m *Model) syncCompletionPrefix() {
	if !m.completion.Visible {
		return
	}
	if m.CursorRow != m.completion.StartRow || m.CursorRow < 0 || m.CursorRow >= len(m.Lines) {
		m.completion.Close()
		return
	}
	if m.CursorCol < m.completion.StartCol {
		m.completion.Close()
		return
	}
	runes := []rune(m.Lines[m.CursorRow])
	end := m.CursorCol
	if end > len(runes) {
		end = len(runes)
	}
	prefix := string(runes[m.completion.StartCol:end])
	for _, r := range prefix {
		if !isIdentRune(r) {
			m.completion.Close()
			return
		}
	}
	m.completion.Filter(prefix)
}

// acceptCompletion applies the currently selected completion item at the cursor.
func (m Model) acceptCompletion() Model {
	chosen, ok := m.completion.Selected()
	if !ok {
		m.completion.Close()
		return m
	}
	onSelect := m.completion.OnSelect
	startRow := m.completion.StartRow
	startCol := m.completion.StartCol
	m.completion.Close()

	if onSelect != nil {
		onSelect(chosen)
		return m.updateViewport()
	}

	insert := chosen.InsertText
	if insert == "" {
		insert = chosen.Label
	}
	if startRow < 0 || startRow >= len(m.Lines) {
		return m
	}

	api := m.editorAPI()
	if m.CursorRow == startRow && m.CursorCol > startCol {
		api.DeleteRange(startRow, startCol, startRow, m.CursorCol)
	}
	api.InsertText(startRow, startCol, insert)
	return m.updateViewport()
}

// cursorScreenCoords returns the 0-based (row, col) of the completion anchor
// inside baseView (which spans totalW x wsHeight).
func (m *Model) cursorScreenCoords(wsHeight int) (screenRow, screenCol int, ok bool) {
	if m.showWelcome || m.fileTree.Focused || len(m.Lines) == 0 {
		return 0, 0, false
	}
	if m.CursorRow < 0 || m.CursorRow >= len(m.Lines) {
		return 0, 0, false
	}

	tabH := m.tabBarHeight()
	textW := m.editorTextWidth()
	if textW < 1 {
		textW = 1
	}

	var editorRelRow int
	var textVisCol int
	anchorCol := m.completion.StartCol
	if anchorCol < 0 || anchorCol > m.CursorCol {
		anchorCol = m.CursorCol
	}

	if m.Config.WordWrap {
		editorRelRow = m.cursorVisualAbsPos(textW) - m.yOffset
		vis := cursorVisualCol(m.Lines[m.CursorRow], anchorCol, m.Config.TabWidth)
		textVisCol = vis % textW
	} else {
		editorRelRow = m.CursorRow - m.yOffset
		vis := cursorVisualCol(m.Lines[m.CursorRow], anchorCol, m.Config.TabWidth)
		textVisCol = vis - m.xOffset
	}

	screenRow = tabH + editorRelRow
	if screenRow < tabH || screenRow >= wsHeight {
		return 0, 0, false
	}

	leftChrome := 0
	if sw := m.sidebarWidth(); sw > 0 {
		leftChrome += sw + 1 // sidebar + divider
	} else {
		leftChrome += 1 // left border │
	}
	if m.Config.LineNumbers {
		leftChrome += lineNumberGutterWidth(len(m.Lines))
	}

	if textVisCol < 0 {
		textVisCol = 0
	}
	screenCol = leftChrome + textVisCol
	return screenRow, screenCol, true
}

func completionKindBadge(kind string, p UIPalette) (tag string, color lipgloss.Color) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "method":
		return "mth", p.Blue
	case "function", "func":
		return "fn ", p.Blue
	case "constructor":
		return "new", p.Yellow
	case "class", "struct":
		return "cls", p.Yellow
	case "interface":
		return "ifc", p.Yellow
	case "variable", "var":
		return "var", p.Cyan
	case "field", "property", "prop":
		return "fld", p.Cyan
	case "module", "package", "namespace":
		return "mod", p.Green
	case "keyword":
		return "kw ", p.Magenta
	case "constant", "const":
		return "cst", p.AccentAlt
	case "enum", "enummember":
		return "enm", p.Yellow
	case "snippet":
		return "snp", p.Green
	default:
		return "sym", p.MutedFg
	}
}

func (c *CompletionPopup) renderBox(maxW, maxH int) ([]string, int) {
	if !c.Visible || len(c.Filtered) == 0 || maxW < 18 || maxH < 3 {
		return nil, 0
	}

	showFooter := maxH >= 5
	chromeRows := 2 // top & bottom border
	if showFooter {
		chromeRows += 2 // divider + footer line
	}
	maxItems := maxH - chromeRows
	if maxItems > completionMaxVisibleRows {
		maxItems = completionMaxVisibleRows
	}
	if maxItems < 1 {
		showFooter = false
		maxItems = maxH - 2
		if maxItems < 1 {
			return nil, 0
		}
	}

	visibleCount := len(c.Filtered)
	if visibleCount > maxItems {
		visibleCount = maxItems
	}

	// Keep cursor inside visible scroll window
	scroll := c.Scroll
	if c.Cursor < scroll {
		scroll = c.Cursor
	}
	if c.Cursor >= scroll+visibleCount {
		scroll = c.Cursor - visibleCount + 1
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll+visibleCount > len(c.Filtered) {
		scroll = len(c.Filtered) - visibleCount
		if scroll < 0 {
			scroll = 0
		}
	}

	// Measure ideal inner width from visible items
	maxLabelW := 0
	maxDetailW := 0
	for i := scroll; i < scroll+visibleCount; i++ {
		it := c.Filtered[i]
		if lw := runewidth.StringWidth(it.Label); lw > maxLabelW {
			maxLabelW = lw
		}
		if dw := runewidth.StringWidth(it.Detail); dw > maxDetailW {
			maxDetailW = dw
		}
	}
	if maxLabelW > 28 {
		maxLabelW = 28
	}
	if maxDetailW > 24 {
		maxDetailW = 24
	}

	// Row format: " " + kind(3) + " " + label + "  " + detail + " "
	idealInner := 1 + 3 + 1 + maxLabelW + 1
	if maxDetailW > 0 {
		idealInner += 2 + maxDetailW
	}
	if showFooter && idealInner < 30 {
		idealInner = 30
	}

	boxW := idealInner + 2 // left & right border
	if boxW < completionMinBoxWidth {
		boxW = completionMinBoxWidth
	}
	if boxW > completionMaxBoxWidth {
		boxW = completionMaxBoxWidth
	}
	if boxW > maxW {
		boxW = maxW
	}
	innerW := boxW - 2
	if innerW < 12 {
		return nil, 0
	}

	p := activePalette
	bg := p.Background
	bgFill := lipgloss.NewStyle().Background(bg)
	borderSt := lipgloss.NewStyle().Foreground(p.Accent).Background(bg)
	ruleSt := lipgloss.NewStyle().Foreground(p.Border).Background(bg)
	labelSt := lipgloss.NewStyle().Foreground(p.Foreground).Background(bg)
	detailSt := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)
	selBg := p.Accent
	selFg := p.AccentFg
	selFill := lipgloss.NewStyle().Background(selBg)
	selLabelSt := lipgloss.NewStyle().Background(selBg).Foreground(selFg).Bold(true)
	selKindSt := lipgloss.NewStyle().Background(selBg).Foreground(selFg)
	selDetailSt := lipgloss.NewStyle().Background(selBg).Foreground(selFg)
	hintKeySt := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)
	hintMutedSt := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)

	lines := make([]string, 0, visibleCount+4)
	lines = append(lines, borderSt.Render("╭"+strings.Repeat("─", innerW)+"╮"))

	vBorder := borderSt.Render("│")

	for i := scroll; i < scroll+visibleCount; i++ {
		it := c.Filtered[i]
		selected := i == c.Cursor
		kindTag, kindColor := completionKindBadge(it.Kind, p)

		// Layout: " " (1) + kindTag (3) + " " (1) + label + [gap + detail] + " " (1)
		prefixW := 5 // " " + 3 + " "
		avail := innerW - prefixW - 1
		if avail < 4 {
			avail = 4
		}

		labelMax := avail
		hasDetail := it.Detail != "" && avail >= 14
		if hasDetail {
			// Reserve up to ~40% for detail while giving label at least 60%
			lblCap := avail * 60 / 100
			if lblCap < 10 {
				lblCap = 10
			}
			labelMax = lblCap
		}
		labelTxt := truncateDisplay(it.Label, labelMax)
		labelW := runewidth.StringWidth(labelTxt)

		detailTxt := ""
		detailW := 0
		if hasDetail {
			detBudget := avail - labelW - 2
			if detBudget >= 4 {
				detailTxt = truncateDisplay(it.Detail, detBudget)
				detailW = runewidth.StringWidth(detailTxt)
			}
		}

		gap := innerW - prefixW - labelW - detailW - 1
		if gap < 0 {
			gap = 0
		}

		var row strings.Builder
		row.WriteString(vBorder)
		if selected {
			row.WriteString(selKindSt.Render(" " + kindTag + " "))
			row.WriteString(selLabelSt.Render(labelTxt))
			if detailTxt != "" {
				row.WriteString(selFill.Render(strings.Repeat(" ", gap)))
				row.WriteString(selDetailSt.Render(detailTxt))
				row.WriteString(selFill.Render(" "))
			} else {
				row.WriteString(selFill.Render(strings.Repeat(" ", gap+1)))
			}
		} else {
			kindSt := lipgloss.NewStyle().Foreground(kindColor).Background(bg).Bold(true)
			row.WriteString(bgFill.Render(" "))
			row.WriteString(kindSt.Render(kindTag))
			row.WriteString(bgFill.Render(" "))
			row.WriteString(labelSt.Render(labelTxt))
			if detailTxt != "" {
				row.WriteString(bgFill.Render(strings.Repeat(" ", gap)))
				row.WriteString(detailSt.Render(detailTxt))
				row.WriteString(bgFill.Render(" "))
			} else {
				row.WriteString(bgFill.Render(strings.Repeat(" ", gap+1)))
			}
		}
		row.WriteString(vBorder)
		lines = append(lines, row.String())
	}

	if showFooter {
		lines = append(lines, vBorder+ruleSt.Render(strings.Repeat("─", innerW))+vBorder)
		counter := fmt.Sprintf(" %d/%d ", c.Cursor+1, len(c.Filtered))
		counterW := runewidth.StringWidth(counter)
		leftHint := bgFill.Render(" ") + hintKeySt.Render("Tab") + hintMutedSt.Render("/") +
			hintKeySt.Render("↵") + hintMutedSt.Render(" accept · ") +
			hintKeySt.Render("Esc") + hintMutedSt.Render(" close")
		if lipgloss.Width(leftHint)+counterW > innerW {
			leftHint = bgFill.Render(" ") + hintKeySt.Render("Tab") + hintMutedSt.Render(" accept")
		}
		if lipgloss.Width(leftHint)+counterW > innerW {
			leftHint = ""
		}
		fGap := innerW - lipgloss.Width(leftHint) - counterW
		if fGap < 0 {
			fGap = 0
		}
		footerLine := leftHint + bgFill.Render(strings.Repeat(" ", fGap)) + hintMutedSt.Render(counter)
		lines = append(lines, vBorder+footerLine+vBorder)
	}

	lines = append(lines, borderSt.Render("╰"+strings.Repeat("─", innerW)+"╯"))
	return lines, boxW
}

// overlayCompletionPopup composites the floating completion box onto baseView
// directly below (or above) the active cursor using ANSI-aware cell slicing.
func (m *Model) overlayCompletionPopup(baseView string, wsHeight int, uiANSI compiledUIANSI) string {
	if !m.completion.Visible || len(m.completion.Filtered) == 0 {
		return baseView
	}
	cursorRow, cursorCol, ok := m.cursorScreenCoords(wsHeight)
	if !ok {
		return baseView
	}

	rows := strings.Split(baseView, "\n")
	if len(rows) == 0 {
		return baseView
	}

	availBelow := len(rows) - (cursorRow + 1)
	availAbove := cursorRow

	placeAbove := false
	maxH := availBelow
	if availBelow < 5 && availAbove > availBelow {
		placeAbove = true
		maxH = availAbove
	}
	if maxH < 3 {
		return baseView
	}

	maxW := m.Width - 2
	if maxW < 18 {
		return baseView
	}

	boxLines, boxW := m.completion.renderBox(maxW, maxH)
	if len(boxLines) == 0 || boxW <= 0 {
		return baseView
	}

	boxTop := cursorRow + 1
	if placeAbove {
		boxTop = cursorRow - len(boxLines)
	}
	if boxTop < 0 {
		boxTop = 0
	}
	if boxTop+len(boxLines) > len(rows) {
		boxTop = len(rows) - len(boxLines)
		if boxTop < 0 {
			return baseView
		}
	}

	// Align completion labels directly under the identifier start (accounting for border + kind prefix)
	boxLeft := cursorCol - 6
	if boxLeft < 0 {
		boxLeft = 0
	}
	if boxLeft+boxW > m.Width {
		boxLeft = m.Width - boxW
		if boxLeft < 0 {
			boxLeft = 0
		}
	}

	for i, popLine := range boxLines {
		rIdx := boxTop + i
		if rIdx < 0 || rIdx >= len(rows) {
			break
		}
		bgLine := rows[rIdx]
		hasEOLClear := uiANSI.eolClear != "" && strings.HasSuffix(bgLine, uiANSI.eolClear)
		if hasEOLClear {
			bgLine = strings.TrimSuffix(bgLine, uiANSI.eolClear)
		}

		leftPart := ansi.Cut(bgLine, 0, boxLeft)
		if lw := ansi.StringWidth(leftPart); lw < boxLeft {
			padSpaces := strings.Repeat(" ", boxLeft-lw)
			if uiANSI.paintBg {
				leftPart += uiANSI.canvasBg.wrap(padSpaces)
			} else {
				leftPart += padSpaces
			}
		}

		rightStart := boxLeft + boxW
		rightPart := ""
		if rightStart < m.Width {
			rightPart = ansi.Cut(bgLine, rightStart, m.Width)
		}

		composed := leftPart + "\x1b[0m" + popLine + "\x1b[0m" + rightPart
		if hasEOLClear {
			composed += uiANSI.eolClear
		}
		rows[rIdx] = composed
	}

	return strings.Join(rows, "\n")
}
