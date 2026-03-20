package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"larry/internal/config"
)

func TestCutRemovesEmptyLine(t *testing.T) {
	tests := []struct {
		name          string
		lines         []string
		cursorRow     int
		wantLines     []string
		wantCursorRow int
		wantCursorCol int
	}{
		{
			name:          "first empty line",
			lines:         []string{"", "omega"},
			cursorRow:     0,
			wantLines:     []string{"omega"},
			wantCursorRow: 0,
			wantCursorCol: 0,
		},
		{
			name:          "middle empty line",
			lines:         []string{"alpha", "", "omega"},
			cursorRow:     1,
			wantLines:     []string{"alpha", "omega"},
			wantCursorRow: 1,
			wantCursorCol: 0,
		},
		{
			name:          "last empty line",
			lines:         []string{"alpha", ""},
			cursorRow:     1,
			wantLines:     []string{"alpha"},
			wantCursorRow: 0,
			wantCursorCol: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitialModel("file.txt", append([]string(nil), tt.lines...), config.DefaultConfig())
			m.Width = 80
			m.Height = 20
			m.CursorRow = tt.cursorRow
			m.CursorCol = 0

			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
			got := updated.(Model)

			if !reflect.DeepEqual(got.Lines, tt.wantLines) {
				t.Fatalf("lines after cut = %#v, want %#v", got.Lines, tt.wantLines)
			}
			if got.CursorRow != tt.wantCursorRow || got.CursorCol != tt.wantCursorCol {
				t.Fatalf("cursor after cut = (%d,%d), want (%d,%d)", got.CursorRow, got.CursorCol, tt.wantCursorRow, tt.wantCursorCol)
			}
			if !got.Modified {
				t.Fatal("expected buffer to be marked modified")
			}
			if len(got.UndoStack) != 1 {
				t.Fatalf("undo stack size = %d, want 1", len(got.UndoStack))
			}

			undone := got.undo()
			if !reflect.DeepEqual(undone.Lines, tt.lines) {
				t.Fatalf("lines after undo = %#v, want %#v", undone.Lines, tt.lines)
			}

			redone := undone.redo()
			if !reflect.DeepEqual(redone.Lines, tt.wantLines) {
				t.Fatalf("lines after redo = %#v, want %#v", redone.Lines, tt.wantLines)
			}
		})
	}
}

func TestCutRemovesCurrentLineContents(t *testing.T) {
	originalWriteClipboard := writeClipboard
	var clipboardText string
	writeClipboard = func(_ Model, text string) error {
		clipboardText = text
		return nil
	}
	t.Cleanup(func() { writeClipboard = originalWriteClipboard })

	m := InitialModel("file.txt", []string{"alpha", "omega"}, config.DefaultConfig())
	m.Width = 80
	m.Height = 20
	m.CursorRow = 1
	m.CursorCol = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	got := updated.(Model)

	if clipboardText != "omega" {
		t.Fatalf("clipboard = %q, want %q", clipboardText, "omega")
	}
	if got.Lines[1] != "" {
		t.Fatalf("line after cut = %q, want empty line", got.Lines[1])
	}
	if got.CursorRow != 1 || got.CursorCol != 0 {
		t.Fatalf("cursor after cut = (%d,%d), want (1,0)", got.CursorRow, got.CursorCol)
	}
	if len(got.UndoStack) != 1 {
		t.Fatalf("undo stack size = %d, want 1", len(got.UndoStack))
	}

	undone := got.undo()
	if !reflect.DeepEqual(undone.Lines, []string{"alpha", "omega"}) {
		t.Fatalf("lines after undo = %#v, want %#v", undone.Lines, []string{"alpha", "omega"})
	}

	redone := undone.redo()
	if redone.Lines[1] != "" {
		t.Fatalf("lines after redo = %#v, want empty second line", redone.Lines)
	}
}

func TestSaveAutoSavesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("setup file: %v", err)
	}

	m := InitialModel(path, []string{"new content"}, config.DefaultConfig())
	m.Width = 80
	m.Height = 20
	m.Modified = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	got := updated.(Model)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "new content" {
		t.Fatalf("saved file = %q, want %q", string(data), "new content")
	}
	if got.saving {
		t.Fatal("expected save flow to stay closed")
	}
	if got.Modified {
		t.Fatal("expected buffer to be marked clean")
	}
	if got.statusMsg != "Saved: "+path {
		t.Fatalf("status = %q, want %q", got.statusMsg, "Saved: "+path)
	}
}

func TestSavePromptsForNewFile(t *testing.T) {
	m := InitialModel("", []string{"draft"}, config.DefaultConfig())
	m.Width = 80
	m.Height = 20

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	got := updated.(Model)

	if !got.saving {
		t.Fatal("expected save flow to open for unnamed file")
	}
	if got.textInput.Value() != "" {
		t.Fatalf("filename prompt value = %q, want empty", got.textInput.Value())
	}
}
