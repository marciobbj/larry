package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGlobalReplaceWorkflowAndBufferSync(t *testing.T) {
	root := t.TempDir()

	fileA := filepath.Join(root, "a.go")
	fileB := filepath.Join(root, "sub", "b.go")
	if err := os.MkdirAll(filepath.Dir(fileB), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fileA, []byte("package main\n\nvar oldVal = \"oldVal\"\n"), 0644); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("package sub\n\nconst ref = oldVal\n"), 0644); err != nil {
		t.Fatalf("write b.go: %v", err)
	}

	m := newTestModelInDir(t, root)
	var err error
	m, err = m.openFile(fileA)
	if err != nil {
		t.Fatalf("openFile: %v", err)
	}
	m.CursorRow = 2
	m.CursorCol = 4

	// Trigger Global Replace with Ctrl+W
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	m = updated.(Model)
	if !m.globalReplacing {
		t.Fatal("expected Ctrl+W to open Global Replace modal")
	}
	if m.globalReplace.Root() != m.fileTree.RootPath {
		t.Fatalf("globalReplace root = %q, want %q", m.globalReplace.Root(), m.fileTree.RootPath)
	}

	// Type "oldVal" into FIND and run the returned async preview command
	var lastCmd tea.Cmd
	for _, r := range "oldVal" {
		updated, lastCmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if lastCmd == nil {
		t.Fatal("expected async preview cmd after typing query")
	}
	previewMsg := lastCmd()
	updated, _ = m.Update(previewMsg)
	m = updated.(Model)

	selFiles, selMatches := m.globalReplace.SelectedCounts()
	if selFiles != 2 || selMatches != 3 {
		t.Fatalf("selected = (%d files, %d matches), want (2, 3)", selFiles, selMatches)
	}

	// Press Enter in FIND to move focus to REPLACE
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.globalReplace.focus != grFocusReplace {
		t.Fatalf("expected focus on REPLACE, got %v", m.globalReplace.focus)
	}

	// Type "newVal" into REPLACE
	for _, r := range "newVal" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	// Press Tab to focus RESULTS list, move down to sub/b.go, press Space to uncheck it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.globalReplace.focus != grFocusList {
		t.Fatalf("expected focus on RESULTS list, got %v", m.globalReplace.focus)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	selFiles, selMatches = m.globalReplace.SelectedCounts()
	if selFiles != 1 || selMatches != 2 {
		t.Fatalf("after unchecking b.go: selected = (%d files, %d matches), want (1, 2)", selFiles, selMatches)
	}

	// Render view while modal is open to verify no panics and expected text
	rendered := m.View()
	if !strings.Contains(rendered, "Global Replace") || !strings.Contains(rendered, "a.go") {
		t.Fatal("expected rendered modal to display Global Replace title and matching file")
	}

	// Press Enter -> enters confirmation step
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.globalReplace.confirming {
		t.Fatal("expected Enter to enter confirmation step")
	}

	// Press 'y' -> executes global replace
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if m.globalReplacing {
		t.Fatal("expected modal to close after executing global replace")
	}
	if !strings.Contains(m.statusMsg, "Replaced 2 occurrences in 1 files") {
		t.Fatalf("unexpected statusMsg = %q", m.statusMsg)
	}

	// Verify a.go was updated both on disk AND in the active editor buffer
	aDisk, _ := os.ReadFile(fileA)
	if string(aDisk) != "package main\n\nvar newVal = \"newVal\"\n" {
		t.Fatalf("a.go on disk = %q", string(aDisk))
	}
	if len(m.Lines) < 3 || m.Lines[2] != "var newVal = \"newVal\"" {
		t.Fatalf("active editor buffer not synced: %#v", m.Lines)
	}

	// Verify unchecked sub/b.go was NOT modified
	bDisk, _ := os.ReadFile(fileB)
	if string(bDisk) != "package sub\n\nconst ref = oldVal\n" {
		t.Fatalf("unchecked b.go was modified: %q", string(bDisk))
	}
}

func TestGlobalReplaceCancelWithEscDoesNotModifyFiles(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "note.txt")
	if err := os.WriteFile(f, []byte("keep me"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := newTestModelInDir(t, root)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	m = updated.(Model)

	var cmd tea.Cmd
	for _, r := range "keep" {
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(cmd())
	m = updated.(Model)

	// Enter -> REPLACE -> Enter -> confirm -> Esc cancels confirm -> Esc closes modal
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.globalReplace.confirming {
		t.Fatal("expected confirmation prompt")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.globalReplace.confirming || !m.globalReplacing {
		t.Fatal("expected first Esc to cancel confirmation while keeping modal open")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.globalReplacing {
		t.Fatal("expected second Esc to close modal")
	}

	data, _ := os.ReadFile(f)
	if string(data) != "keep me" {
		t.Fatalf("file modified despite cancel: %q", string(data))
	}
}
