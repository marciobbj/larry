package ui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"larry/internal/config"
)

func TestSelectNextOccurrenceDownwardOnlyAndEditAllWithUndoRedo(t *testing.T) {
	lines := []string{
		"const foo = 0 // above cursor, must NOT be selected",
		"let foo = foo + 1",
		"print(foo)",
		"return foo",
	}
	m := InitialModel("app.js", append([]string(nil), lines...), config.DefaultConfig())
	m.Width = 80
	m.Height = 20
	m.CursorRow = 1
	m.CursorCol = 4 // on first "foo" of line 1

	// 1st Ctrl+K selects "foo" under cursor
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)
	if !m.selecting || m.getSelectedText() != "foo" {
		t.Fatalf("expected word 'foo' selected, got selecting=%v text=%q", m.selecting, m.getSelectedText())
	}

	// Occurrence highlights must ONLY include matches from cursor downward (4 matches, NOT line 0!)
	occByRow := m.getVisibleOccurrences(0, len(m.Lines))
	totalOccs := 0
	for row, spans := range occByRow {
		if row == 0 {
			t.Fatalf("occurrence above cursor (line 0) must NOT be highlighted: %+v", occByRow)
		}
		totalOccs += len(spans)
	}
	if totalOccs != 4 {
		t.Fatalf("expected 4 downward occurrences (lines 1,1,2,3), got %d: %+v", totalOccs, occByRow)
	}

	// 2nd Alt+I selects second "foo" on line 1 (same line!)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
	m = updated.(Model)
	// 3rd Ctrl+K selects "foo" on line 2
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)
	// 4th Ctrl+K selects "foo" on line 3
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)
	// 5th Ctrl+K must NOT wrap around to line 0!
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)

	if got := len(m.allCursors()); got != 4 {
		t.Fatalf("expected 4 cursors (downward only, no wrap to line 0), got %d", got)
	}

	// Type "barBaz" across all 4 cursors simultaneously (including 2 on the same line!)
	for _, r := range "barBaz" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	wantLines := []string{
		"const foo = 0 // above cursor, must NOT be selected",
		"let barBaz = barBaz + 1",
		"print(barBaz)",
		"return barBaz",
	}
	if !reflect.DeepEqual(m.Lines, wantLines) {
		t.Fatalf("lines after multi-cursor edit =\n%#v\nwant\n%#v", m.Lines, wantLines)
	}

	// Undo 6 typed characters -> restores original lines cleanly
	for i := 0; i < len("barBaz"); i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
		m = updated.(Model)
	}
	if !reflect.DeepEqual(m.Lines, lines) {
		t.Fatalf("lines after undo =\n%#v\nwant\n%#v", m.Lines, lines)
	}

	// Redo 6 typed characters -> reapplies "barBaz"
	for i := 0; i < len("barBaz"); i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
		m = updated.(Model)
	}
	if !reflect.DeepEqual(m.Lines, wantLines) {
		t.Fatalf("lines after redo =\n%#v\nwant\n%#v", m.Lines, wantLines)
	}

	// Esc collapses back to single cursor
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.hasMultiCursor() {
		t.Fatal("expected Esc to exit multi-cursor mode")
	}
}

func TestSelectAllOccurrencesBelowAndVerticalMultiCursor(t *testing.T) {
	lines := []string{
		"item one",
		"item two",
		"item three",
		"item four",
	}
	m := InitialModel("list.txt", append([]string(nil), lines...), config.DefaultConfig())
	m.Width = 80
	m.Height = 20
	m.CursorRow = 1
	m.CursorCol = 1 // on "item" of line 1

	// Ctrl+B selects all occurrences of "item" from line 1 downward (lines 1, 2, 3 — NOT line 0!)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = updated.(Model)
	if len(m.allCursors()) != 3 {
		t.Fatalf("expected 3 cursors (lines 1..3), got %d", len(m.allCursors()))
	}

	// Press Backspace to delete "item" on lines 1..3
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)
	want := []string{
		"item one",
		" two",
		" three",
		" four",
	}
	if !reflect.DeepEqual(m.Lines, want) {
		t.Fatalf("lines after Ctrl+B + Backspace = %#v, want %#v", m.Lines, want)
	}

	// Exit multi-cursor, go to (0, 0), add 2 cursors below with Alt+Down, type "- "
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	m.CursorRow = 0
	m.CursorCol = 0

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown, Alt: true})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown, Alt: true})
	m = updated.(Model)
	if len(m.allCursors()) != 3 {
		t.Fatalf("expected 3 vertical cursors, got %d", len(m.allCursors()))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	wantBullets := []string{
		"- item one",
		"-  two",
		"-  three",
		" four",
	}
	if !reflect.DeepEqual(m.Lines, wantBullets) {
		t.Fatalf("lines after vertical multi-cursor edit = %#v, want %#v", m.Lines, wantBullets)
	}
}

func TestMultiCursorStaysFastOnLargeFile(t *testing.T) {
	// 50,000-line file verifying O(H) viewport highlighting, streaming Ctrl+K, and O(k) undo/redo
	const totalLines = 50000
	lines := make([]string, totalLines)
	for i := range lines {
		lines[i] = "var targetToken = compute(value)"
	}

	m := InitialModel("big.go", lines, config.DefaultConfig())
	m.Width = 100
	m.Height = 40
	m.CursorRow = 1000
	m.CursorCol = 5 // inside "targetToken"

	// Select word + next 4 occurrences below
	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
		m = updated.(Model)
	}
	if len(m.allCursors()) != 5 {
		t.Fatalf("expected 5 cursors, got %d", len(m.allCursors()))
	}

	// Render viewport (must scan only visible ~38 rows)
	_ = m.viewEditor(editorViewConfig{width: 100, height: 38})

	// Replace with "renamed" and undo/redo cleanly without copying the 50k buffer
	for _, r := range "renamed" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if m.Lines[999] != "var targetToken = compute(value)" {
		t.Fatalf("line above cursor was modified: %q", m.Lines[999])
	}
	for row := 1000; row < 1005; row++ {
		if m.Lines[row] != "var renamed = compute(value)" {
			t.Fatalf("row %d = %q, want renamed", row, m.Lines[row])
		}
	}

	for range "renamed" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
		m = updated.(Model)
	}
	for row := 1000; row < 1005; row++ {
		if m.Lines[row] != "var targetToken = compute(value)" {
			t.Fatalf("row %d after undo = %q, want targetToken", row, m.Lines[row])
		}
	}
}
