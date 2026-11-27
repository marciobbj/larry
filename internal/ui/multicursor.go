package ui

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"larry/internal/search"
)

const maxMultiCursors = 2000

// CursorSelection represents a single cursor and optional selection range.
type CursorSelection struct {
	Row       int
	Col       int
	StartRow  int
	StartCol  int
	Selecting bool
}

func (c CursorSelection) normalizedBounds() (sRow, sCol, eRow, eCol int) {
	if !c.Selecting {
		return c.Row, c.Col, c.Row, c.Col
	}
	sRow, sCol, eRow, eCol = c.StartRow, c.StartCol, c.Row, c.Col
	if sRow > eRow || (sRow == eRow && sCol > eCol) {
		sRow, sCol, eRow, eCol = eRow, eCol, sRow, sCol
	}
	return sRow, sCol, eRow, eCol
}

// RangeEdit records a single atomic range replacement so multi-cursor undo/redo
// runs in O(k) time and O(k) memory without ever cloning the whole file buffer.
type RangeEdit struct {
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
	NewRow   int
	NewCol   int
	OldText  string
	NewText  string
}

type colSpan struct {
	start int
	end   int
}

type occurrenceCache struct {
	query      string
	contentVer int
	lineCount  int
	startRow   int
	endRow     int
	pStartRow  int
	pStartCol  int
	byRow      map[int][]colSpan
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func wordBoundsAt(line string, col int) (int, int, bool) {
	runes := []rune(line)
	if len(runes) == 0 {
		return 0, 0, false
	}
	idx := col
	if idx >= len(runes) {
		idx = len(runes) - 1
	}
	if idx < 0 {
		idx = 0
	}
	if !isWordRune(runes[idx]) {
		if idx > 0 && isWordRune(runes[idx-1]) {
			idx--
		} else {
			return 0, 0, false
		}
	}
	start := idx
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}
	end := idx + 1
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}
	return start, end, end > start
}

// findLineOccurrences uses stdlib SIMD-accelerated strings.Index and converts
// byte offsets to rune columns only when matches actually occur.
func findLineOccurrences(line, query string, qRuneLen int) []colSpan {
	if query == "" || len(line) < len(query) {
		return nil
	}
	var spans []colSpan
	bytePos := 0
	runePos := 0
	for {
		rel := strings.Index(line[bytePos:], query)
		if rel < 0 {
			break
		}
		matchByte := bytePos + rel
		runePos += utf8.RuneCountInString(line[bytePos:matchByte])
		spans = append(spans, colSpan{start: runePos, end: runePos + qRuneLen})
		bytePos = matchByte + len(query)
		runePos += qRuneLen
	}
	return spans
}

// findOccurrencesInRange finds occurrences of query ONLY within [fromRow, toRow)
// and at or after (minRow, minCol), keeping rendering strictly O(H).
func findOccurrencesInRange(lines []string, query string, fromRow, toRow, minRow, minCol int) map[int][]colSpan {
	if query == "" || len(lines) == 0 {
		return nil
	}
	if fromRow < minRow {
		fromRow = minRow
	}
	if fromRow < 0 {
		fromRow = 0
	}
	if toRow > len(lines) {
		toRow = len(lines)
	}
	if fromRow >= toRow {
		return nil
	}

	qRuneLen := utf8.RuneCountInString(query)
	var byRow map[int][]colSpan

	for row := fromRow; row < toRow; row++ {
		spans := findLineOccurrences(lines[row], query, qRuneLen)
		if len(spans) == 0 {
			continue
		}
		if row == minRow && minCol > 0 {
			filtered := spans[:0]
			for _, sp := range spans {
				if sp.start >= minCol {
					filtered = append(filtered, sp)
				}
			}
			spans = filtered
		}
		if len(spans) > 0 {
			if byRow == nil {
				byRow = make(map[int][]colSpan)
			}
			byRow[row] = spans
		}
	}
	return byRow
}

// findNextOccurrenceFrom streams forward from (fromRow, fromCol) using stdlib
// strings.Index and stops immediately on the very first unselected match.
func findNextOccurrenceFrom(lines []string, query string, fromRow, fromCol int, isAlreadySelected func(row, col, length int) bool) (search.SearchMatch, bool) {
	if query == "" || fromRow < 0 || fromRow >= len(lines) {
		return search.SearchMatch{}, false
	}
	qRuneLen := utf8.RuneCountInString(query)

	for row := fromRow; row < len(lines); row++ {
		spans := findLineOccurrences(lines[row], query, qRuneLen)
		for _, sp := range spans {
			if row == fromRow && sp.start < fromCol {
				continue
			}
			if isAlreadySelected(row, sp.start, qRuneLen) {
				continue
			}
			return search.SearchMatch{
				Line:   row,
				Col:    sp.start,
				Length: qRuneLen,
			}, true
		}
	}
	return search.SearchMatch{}, false
}

func (m Model) hasMultiCursor() bool {
	return len(m.extraCursors) > 0
}

func (m Model) primaryCursor() CursorSelection {
	return CursorSelection{
		Row:       m.CursorRow,
		Col:       m.CursorCol,
		StartRow:  m.startRow,
		StartCol:  m.startCol,
		Selecting: m.selecting,
	}
}

func (m Model) allCursors() []CursorSelection {
	res := make([]CursorSelection, 0, 1+len(m.extraCursors))
	res = append(res, m.primaryCursor())
	res = append(res, m.extraCursors...)
	return res
}

func (m *Model) setAllCursors(cursors []CursorSelection) {
	if len(cursors) == 0 {
		m.extraCursors = nil
		return
	}

	type cursorKey struct {
		row, col, sRow, sCol int
		selecting            bool
	}
	seen := make(map[cursorKey]struct{}, len(cursors))
	dedup := make([]CursorSelection, 0, len(cursors))

	maxRow := maxInt(0, len(m.Lines)-1)
	for _, c := range cursors {
		c.Row = clampInt(c.Row, 0, maxRow)
		c.Col = clampInt(c.Col, 0, lineRuneLen(m.Lines, c.Row))
		if c.Selecting {
			c.StartRow = clampInt(c.StartRow, 0, maxRow)
			c.StartCol = clampInt(c.StartCol, 0, lineRuneLen(m.Lines, c.StartRow))
			if c.StartRow == c.Row && c.StartCol == c.Col {
				c.Selecting = false
			}
		}
		k := cursorKey{row: c.Row, col: c.Col, selecting: c.Selecting}
		if c.Selecting {
			k.sRow, k.sCol = c.StartRow, c.StartCol
		}
		if _, exists := seen[k]; !exists {
			seen[k] = struct{}{}
			dedup = append(dedup, c)
		}
	}

	primary := dedup[0]
	m.CursorRow = primary.Row
	m.CursorCol = primary.Col
	m.startRow = primary.StartRow
	m.startCol = primary.StartCol
	m.selecting = primary.Selecting

	if len(dedup) > 1 {
		m.extraCursors = dedup[1:]
	} else {
		m.extraCursors = nil
	}
}

func (m *Model) clearExtraCursors() {
	m.extraCursors = nil
}

// singleLineSelectedText returns the primary selection text if it is a
// non-empty, non-whitespace single-line selection.
func (m Model) singleLineSelectedText() string {
	if !m.selecting || m.startRow != m.CursorRow {
		return ""
	}
	if m.CursorRow < 0 || m.CursorRow >= len(m.Lines) {
		return ""
	}
	text := m.getSelectedText()
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return text
}

// getVisibleOccurrences computes matching occurrences strictly within the
// visible viewport [viewStartRow, viewEndRow) AND from the primary cursor
// downward — O(H) regardless of total file size.
func (m Model) getVisibleOccurrences(viewStartRow, viewEndRow int) map[int][]colSpan {
	query := m.singleLineSelectedText()
	if query == "" {
		return nil
	}
	pStartRow, pStartCol, _, _ := m.primaryCursor().normalizedBounds()
	if m.occCache != nil {
		oc := m.occCache
		if oc.query == query &&
			oc.contentVer == m.visualLineVersion &&
			oc.lineCount == len(m.Lines) &&
			oc.startRow == viewStartRow &&
			oc.endRow == viewEndRow &&
			oc.pStartRow == pStartRow &&
			oc.pStartCol == pStartCol {
			return oc.byRow
		}
		byRow := findOccurrencesInRange(m.Lines, query, viewStartRow, viewEndRow, pStartRow, pStartCol)
		oc.query = query
		oc.contentVer = m.visualLineVersion
		oc.lineCount = len(m.Lines)
		oc.startRow = viewStartRow
		oc.endRow = viewEndRow
		oc.pStartRow = pStartRow
		oc.pStartCol = pStartCol
		oc.byRow = byRow
		return byRow
	}
	return findOccurrencesInRange(m.Lines, query, viewStartRow, viewEndRow, pStartRow, pStartCol)
}

// selectNextOccurrence selects the word under the cursor (if nothing is selected),
// or streams forward to add a cursor & selection on the next occurrence BELOW
// the last cursor (never wrapping above, never scanning past the next match).
func (m Model) selectNextOccurrence() Model {
	if len(m.Lines) == 0 {
		return m
	}

	if !m.selecting {
		start, end, ok := wordBoundsAt(m.Lines[m.CursorRow], m.CursorCol)
		if !ok {
			m.statusMsg = "No word under cursor"
			return m
		}
		m.startRow = m.CursorRow
		m.startCol = start
		m.CursorCol = end
		m.selecting = true
		m.statusMsg = fmt.Sprintf("Selected %q", m.getSelectedText())
		return m
	}

	query := m.singleLineSelectedText()
	if query == "" {
		return m
	}

	cursors := m.allCursors()
	if len(cursors) >= maxMultiCursors {
		m.statusMsg = fmt.Sprintf("Cursor limit reached (%d)", maxMultiCursors)
		return m
	}

	type selKey struct{ row, col, endCol int }
	selectedSet := make(map[selKey]struct{}, len(cursors))
	for _, c := range cursors {
		sr, sc, er, ec := c.normalizedBounds()
		if sr == er {
			selectedSet[selKey{sr, sc, ec}] = struct{}{}
		}
	}

	last := cursors[len(cursors)-1]
	_, _, lastEndRow, lastEndCol := last.normalizedBounds()

	mt, found := findNextOccurrenceFrom(m.Lines, query, lastEndRow, lastEndCol, func(row, col, length int) bool {
		_, exists := selectedSet[selKey{row, col, col + length}]
		return exists
	})
	if !found {
		m.statusMsg = fmt.Sprintf("No more occurrences below (%d selected)", len(cursors))
		return m
	}

	cursors = append(cursors, CursorSelection{
		StartRow:  mt.Line,
		StartCol:  mt.Col,
		Row:       mt.Line,
		Col:       mt.Col + mt.Length,
		Selecting: true,
	})
	m.setAllCursors(cursors)
	m.statusMsg = fmt.Sprintf("%d occurrences selected", len(m.allCursors()))
	return m
}

// selectAllOccurrences selects occurrences of the currently selected text
// (or the word under the cursor) from the cursor downward, capped at maxMultiCursors.
func (m Model) selectAllOccurrences() Model {
	if len(m.Lines) == 0 {
		return m
	}
	if !m.selecting {
		start, end, ok := wordBoundsAt(m.Lines[m.CursorRow], m.CursorCol)
		if !ok {
			m.statusMsg = "No word under cursor"
			return m
		}
		m.startRow = m.CursorRow
		m.startCol = start
		m.CursorCol = end
		m.selecting = true
	}

	query := m.singleLineSelectedText()
	if query == "" {
		return m
	}

	primary := m.primaryCursor()
	psr, psc, per, pec := primary.normalizedBounds()
	qRuneLen := utf8.RuneCountInString(query)

	cursors := []CursorSelection{primary}
	capped := false

	for row := psr; row < len(m.Lines); row++ {
		spans := findLineOccurrences(m.Lines[row], query, qRuneLen)
		for _, sp := range spans {
			if row == psr && sp.start < psc {
				continue
			}
			if row == psr && row == per && sp.start == psc && sp.end == pec {
				continue
			}
			if len(cursors) >= maxMultiCursors {
				capped = true
				break
			}
			cursors = append(cursors, CursorSelection{
				StartRow:  row,
				StartCol:  sp.start,
				Row:       row,
				Col:       sp.end,
				Selecting: true,
			})
		}
		if capped {
			break
		}
	}

	m.setAllCursors(cursors)
	if capped {
		m.statusMsg = fmt.Sprintf("Selected first %d occurrences below (limit reached)", maxMultiCursors)
	} else {
		m.statusMsg = fmt.Sprintf("Selected %d occurrences below of %q", len(m.allCursors()), query)
	}
	return m
}

func (m Model) addCursorAbove() Model {
	if len(m.Lines) == 0 {
		return m
	}
	cursors := m.allCursors()
	if len(cursors) >= maxMultiCursors {
		return m
	}
	top := cursors[0]
	for _, c := range cursors[1:] {
		if c.Row < top.Row {
			top = c
		}
	}
	if top.Row <= 0 {
		return m
	}
	newRow := top.Row - 1
	newCol := clampInt(top.Col, 0, lineRuneLen(m.Lines, newRow))
	cursors = append(cursors, CursorSelection{Row: newRow, Col: newCol})
	m.setAllCursors(cursors)
	m.statusMsg = fmt.Sprintf("%d cursors", len(m.allCursors()))
	return m
}

func (m Model) addCursorBelow() Model {
	if len(m.Lines) == 0 {
		return m
	}
	cursors := m.allCursors()
	if len(cursors) >= maxMultiCursors {
		return m
	}
	bottom := cursors[0]
	for _, c := range cursors[1:] {
		if c.Row > bottom.Row {
			bottom = c
		}
	}
	if bottom.Row >= len(m.Lines)-1 {
		return m
	}
	newRow := bottom.Row + 1
	newCol := clampInt(bottom.Col, 0, lineRuneLen(m.Lines, newRow))
	cursors = append(cursors, CursorSelection{Row: newRow, Col: newCol})
	m.setAllCursors(cursors)
	m.statusMsg = fmt.Sprintf("%d cursors", len(m.allCursors()))
	return m
}

func cloneCursors(cursors []CursorSelection) []CursorSelection {
	out := make([]CursorSelection, len(cursors))
	copy(out, cursors)
	return out
}

type indexedCursor struct {
	origIdx int
	c       CursorSelection
}

// sortedCursorsAsc returns all cursors sorted in ascending document order
// (startRow, startCol), retaining original index so primary cursor identity is preserved.
func (m Model) sortedCursorsAsc() []indexedCursor {
	all := m.allCursors()
	out := make([]indexedCursor, len(all))
	for i, c := range all {
		out[i] = indexedCursor{origIdx: i, c: c}
	}
	sort.Slice(out, func(i, j int) bool {
		sr1, sc1, _, _ := out[i].c.normalizedBounds()
		sr2, sc2, _, _ := out[j].c.normalizedBounds()
		if sr1 != sr2 {
			return sr1 < sr2
		}
		return sc1 < sc2
	})
	return out
}

func restoreOriginalOrder(sorted []indexedCursor) []CursorSelection {
	out := make([]CursorSelection, len(sorted))
	for _, ic := range sorted {
		out[ic.origIdx] = ic.c
	}
	return out
}

// applyRangeReplace replaces [sr:sc .. er:ec] in lines with replacement text.
// Single-line edits mutate lines[sr] in place in O(lineLen) without cloning the buffer.
func applyRangeReplace(lines *[]string, sr, sc, er, ec int, text string) (int, int) {
	l := *lines
	if len(l) == 0 {
		l = []string{""}
	}
	sr = clampInt(sr, 0, len(l)-1)
	er = clampInt(er, 0, len(l)-1)
	startRunes := []rune(l[sr])
	endRunes := []rune(l[er])
	sc = clampInt(sc, 0, len(startRunes))
	ec = clampInt(ec, 0, len(endRunes))

	prefix := string(startRunes[:sc])
	suffix := string(endRunes[ec:])

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	parts := strings.Split(text, "\n")

	if len(parts) == 1 {
		l[sr] = prefix + parts[0] + suffix
		if er > sr {
			l = append(l[:sr+1], l[er+1:]...)
		}
		*lines = l
		return sr, sc + len([]rune(parts[0]))
	}

	firstLine := prefix + parts[0]
	lastInsert := parts[len(parts)-1]
	lastLine := lastInsert + suffix

	newLines := make([]string, 0, len(l)-(er-sr)+len(parts)-1)
	newLines = append(newLines, l[:sr]...)
	newLines = append(newLines, firstLine)
	if len(parts) > 2 {
		newLines = append(newLines, parts[1:len(parts)-1]...)
	}
	newLines = append(newLines, lastLine)
	newLines = append(newLines, l[er+1:]...)
	*lines = newLines

	return sr + len(parts) - 1, len([]rune(lastInsert))
}

func shiftCursorAfterEdit(c *CursorSelection, sr, sc, er, ec, newRow, newCol int) {
	shiftPoint := func(row, col *int) {
		if *row > er {
			*row += newRow - er
		} else if *row == er && *col >= ec {
			*row = newRow
			*col = newCol + (*col - ec)
		}
	}
	shiftPoint(&c.Row, &c.Col)
	if c.Selecting {
		shiftPoint(&c.StartRow, &c.StartCol)
	}
}

// runMultiEdit applies edits in place with O(k) sparse RangeEdit undo records
// (never copying the whole m.Lines slice).
func (m Model) runMultiEdit(apply func(lines *[]string, sorted []indexedCursor) []RangeEdit) Model {
	beforeCursors := cloneCursors(m.allCursors())
	sorted := m.sortedCursorsAsc()

	edits := apply(&m.Lines, sorted)
	if len(edits) == 0 {
		return m
	}

	m.setAllCursors(restoreOriginalOrder(sorted))
	m.markModified()

	m.pushUndo(EditOp{
		Type:          OpMultiEdit,
		RangeEdits:    edits,
		BeforeCursors: beforeCursors,
		AfterCursors:  cloneCursors(m.allCursors()),
	})
	return m
}

func (m Model) multiInsertText(text string) Model {
	return m.runMultiEdit(func(lines *[]string, sorted []indexedCursor) []RangeEdit {
		edits := make([]RangeEdit, 0, len(sorted))
		for i := len(sorted) - 1; i >= 0; i-- {
			sr, sc, er, ec := sorted[i].c.normalizedBounds()
			oldText := extractRangeText(*lines, sr, sc, er, ec)
			newRow, newCol := applyRangeReplace(lines, sr, sc, er, ec, text)
			edits = append(edits, RangeEdit{
				StartRow: sr, StartCol: sc, EndRow: er, EndCol: ec,
				NewRow: newRow, NewCol: newCol,
				OldText: oldText, NewText: text,
			})
			sorted[i].c = CursorSelection{Row: newRow, Col: newCol}
			for j := i + 1; j < len(sorted); j++ {
				shiftCursorAfterEdit(&sorted[j].c, sr, sc, er, ec, newRow, newCol)
			}
		}
		return edits
	})
}

func (m Model) multiBackspace() Model {
	return m.runMultiEdit(func(lines *[]string, sorted []indexedCursor) []RangeEdit {
		edits := make([]RangeEdit, 0, len(sorted))
		for i := len(sorted) - 1; i >= 0; i-- {
			c := sorted[i].c
			var sr, sc, er, ec int
			if c.Selecting {
				sr, sc, er, ec = c.normalizedBounds()
			} else if c.Col > 0 {
				sr, sc, er, ec = c.Row, c.Col-1, c.Row, c.Col
			} else if c.Row > 0 {
				prevLen := lineRuneLen(*lines, c.Row-1)
				sr, sc, er, ec = c.Row-1, prevLen, c.Row, 0
			} else {
				continue
			}
			oldText := extractRangeText(*lines, sr, sc, er, ec)
			newRow, newCol := applyRangeReplace(lines, sr, sc, er, ec, "")
			edits = append(edits, RangeEdit{
				StartRow: sr, StartCol: sc, EndRow: er, EndCol: ec,
				NewRow: newRow, NewCol: newCol,
				OldText: oldText, NewText: "",
			})
			sorted[i].c = CursorSelection{Row: newRow, Col: newCol}
			for j := i + 1; j < len(sorted); j++ {
				shiftCursorAfterEdit(&sorted[j].c, sr, sc, er, ec, newRow, newCol)
			}
		}
		return edits
	})
}

func (m Model) multiMove(step func(lines []string, row, col int) (int, int), extendSelection bool) Model {
	cursors := m.allCursors()
	for i := range cursors {
		c := &cursors[i]
		if extendSelection {
			if !c.Selecting {
				c.Selecting = true
				c.StartRow, c.StartCol = c.Row, c.Col
			}
		} else {
			c.Selecting = false
		}
		c.Row, c.Col = step(m.Lines, c.Row, c.Col)
	}
	m.setAllCursors(cursors)
	return m
}

func (m Model) multiCopyText() string {
	sorted := m.sortedCursorsAsc()
	parts := make([]string, 0, len(sorted))
	for _, ic := range sorted {
		if !ic.c.Selecting {
			if ic.c.Row >= 0 && ic.c.Row < len(m.Lines) {
				parts = append(parts, m.Lines[ic.c.Row])
			}
			continue
		}
		sr, sc, er, ec := ic.c.normalizedBounds()
		parts = append(parts, extractRangeText(m.Lines, sr, sc, er, ec))
	}
	return strings.Join(parts, "\n")
}

func (m Model) multiPaste(text string) Model {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	pasteLines := strings.Split(text, "\n")

	return m.runMultiEdit(func(lines *[]string, sorted []indexedCursor) []RangeEdit {
		edits := make([]RangeEdit, 0, len(sorted))
		perCursor := len(pasteLines) == len(sorted)
		for i := len(sorted) - 1; i >= 0; i-- {
			payload := text
			if perCursor {
				payload = pasteLines[i]
			}
			sr, sc, er, ec := sorted[i].c.normalizedBounds()
			oldText := extractRangeText(*lines, sr, sc, er, ec)
			newRow, newCol := applyRangeReplace(lines, sr, sc, er, ec, payload)
			edits = append(edits, RangeEdit{
				StartRow: sr, StartCol: sc, EndRow: er, EndCol: ec,
				NewRow: newRow, NewCol: newCol,
				OldText: oldText, NewText: payload,
			})
			sorted[i].c = CursorSelection{Row: newRow, Col: newCol}
			for j := i + 1; j < len(sorted); j++ {
				shiftCursorAfterEdit(&sorted[j].c, sr, sc, er, ec, newRow, newCol)
			}
		}
		return edits
	})
}

func extractRangeText(lines []string, sr, sc, er, ec int) string {
	if len(lines) == 0 {
		return ""
	}
	sr = clampInt(sr, 0, len(lines)-1)
	er = clampInt(er, 0, len(lines)-1)
	if sr == er {
		r := []rune(lines[sr])
		sc = clampInt(sc, 0, len(r))
		ec = clampInt(ec, 0, len(r))
		return string(r[sc:ec])
	}
	var b strings.Builder
	startR := []rune(lines[sr])
	sc = clampInt(sc, 0, len(startR))
	b.WriteString(string(startR[sc:]))
	b.WriteByte('\n')
	for row := sr + 1; row < er; row++ {
		b.WriteString(lines[row])
		b.WriteByte('\n')
	}
	endR := []rune(lines[er])
	ec = clampInt(ec, 0, len(endR))
	b.WriteString(string(endR[:ec]))
	return b.String()
}

func lineRuneLen(lines []string, row int) int {
	if row < 0 || row >= len(lines) {
		return 0
	}
	return len([]rune(lines[row]))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
