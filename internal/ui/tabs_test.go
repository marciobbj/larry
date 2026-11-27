package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
)

func TestTabLifecycleNavigationAndStatePreservation(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "one.go")
	file2 := filepath.Join(dir, "two.go")
	if err := os.WriteFile(file1, []byte("package one\nline1\nline2"), 0644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("package two\nhello"), 0644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	m := newTestModelInDir(t, dir)

	// Opening first file replaces the untouched initial empty tab
	var err error
	m, err = m.openFile(file1)
	if err != nil {
		t.Fatalf("openFile(file1): %v", err)
	}
	if len(m.tabs) != 1 || m.activeTab != 0 {
		t.Fatalf("expected 1 tab active at 0, got len=%d active=%d", len(m.tabs), m.activeTab)
	}

	// Edit tab 0 and move cursor
	m.CursorRow = 1
	m.CursorCol = 5
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	m = updated.(Model)
	if m.Lines[1] != "line1!" || !m.Modified {
		t.Fatalf("expected edited line1! and Modified=true, got %q Modified=%v", m.Lines[1], m.Modified)
	}

	// Open second file -> creates tab 1
	m, err = m.openFile(file2)
	if err != nil {
		t.Fatalf("openFile(file2): %v", err)
	}
	if len(m.tabs) != 2 || m.activeTab != 1 {
		t.Fatalf("expected 2 tabs active at 1, got len=%d active=%d", len(m.tabs), m.activeTab)
	}
	if m.Lines[0] != "package two" || m.Modified {
		t.Fatalf("expected clean file2 buffer, got %#v Modified=%v", m.Lines, m.Modified)
	}

	// Re-opening file1 switches to existing tab 0 without losing unsaved edits or cursor!
	m, err = m.openFile(file1)
	if err != nil {
		t.Fatalf("re-open file1: %v", err)
	}
	if len(m.tabs) != 2 || m.activeTab != 0 {
		t.Fatalf("expected dedup switch to tab 0, got len=%d active=%d", len(m.tabs), m.activeTab)
	}
	if m.Lines[1] != "line1!" || !m.Modified || m.CursorRow != 1 || m.CursorCol != 6 {
		t.Fatalf("expected tab 0 state preserved, got line=%q mod=%v cursor=(%d,%d)", m.Lines[1], m.Modified, m.CursorRow, m.CursorCol)
	}

	// Switch to next tab via Alt+Right (and wrap around with Alt+])
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
	m = updated.(Model)
	if m.activeTab != 1 || m.Lines[0] != "package two" {
		t.Fatalf("expected Alt+Right to switch to tab 1, got active=%d line0=%q", m.activeTab, m.Lines[0])
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}, Alt: true})
	m = updated.(Model)
	if m.activeTab != 0 {
		t.Fatalf("expected Alt+] to wrap to tab 0, got active=%d", m.activeTab)
	}

	// Create new untitled tab with Ctrl+N
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = updated.(Model)
	if len(m.tabs) != 3 || m.activeTab != 2 || m.FileName != "" {
		t.Fatalf("expected Ctrl+N to create untitled tab 2, got len=%d active=%d file=%q", len(m.tabs), m.activeTab, m.FileName)
	}

	// Close untitled tab with Ctrl+D -> returns to tab 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)
	if len(m.tabs) != 2 || m.activeTab != 1 {
		t.Fatalf("expected Ctrl+D to close tab 2 and activate tab 1, got len=%d active=%d", len(m.tabs), m.activeTab)
	}

	// Close remaining tabs -> last close resets to welcome screen
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)
	if len(m.tabs) != 1 || !m.showWelcome || m.FileName != "" {
		t.Fatalf("expected closing last tab to reset to welcome screen, got len=%d welcome=%v file=%q", len(m.tabs), m.showWelcome, m.FileName)
	}
}

func TestTabBarDisambiguatesDuplicateFilenamesAndFitsHeight(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "alpha", "main.go")
	bPath := filepath.Join(dir, "beta", "main.go")
	for _, p := range []string{aPath, bPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte("package main\n"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	m := newTestModelInDir(t, dir)
	m.Width = 100
	m.Height = 24
	m, _ = m.openFile(aPath)
	m, _ = m.openFile(bPath)

	labels := m.tabDisplayLabels()
	if len(labels) != 2 || labels[0] != "alpha/main.go" || labels[1] != "beta/main.go" {
		t.Fatalf("expected disambiguated labels [alpha/main.go beta/main.go], got %v", labels)
	}

	// Verify rendered view with sidebar + tab bar + editor + status bar has exact terminal height
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	rendered := m.View()
	if !strings.Contains(rendered, "alpha/main.go") || !strings.Contains(rendered, "beta/main.go") {
		t.Fatal("expected tab bar to render both disambiguated tab labels")
	}
	if h := lipgloss.Height(rendered); h != m.Height {
		t.Fatalf("rendered height = %d, want %d", h, m.Height)
	}
}

func TestGlobalReplaceSyncsBackgroundTabsToo(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "first.go")
	f2 := filepath.Join(dir, "second.go")
	if err := os.WriteFile(f1, []byte("oldSymbol"), 0644); err != nil {
		t.Fatalf("write f1: %v", err)
	}
	if err := os.WriteFile(f2, []byte("oldSymbol"), 0644); err != nil {
		t.Fatalf("write f2: %v", err)
	}

	m := newTestModelInDir(t, dir)
	m, _ = m.openFile(f1)
	m, _ = m.openFile(f2) // tab 1 active, tab 0 in background

	// Simulate files modified by Global Replace on disk
	if err := os.WriteFile(f1, []byte("newSymbol"), 0644); err != nil {
		t.Fatalf("rewrite f1: %v", err)
	}
	if err := os.WriteFile(f2, []byte("newSymbol"), 0644); err != nil {
		t.Fatalf("rewrite f2: %v", err)
	}

	m = m.reloadModifiedTabs([]string{f1, f2})
	if m.Lines[0] != "newSymbol" {
		t.Fatalf("active tab not reloaded: %#v", m.Lines)
	}

	m = m.prevTab()
	if m.Lines[0] != "newSymbol" {
		t.Fatalf("background tab not reloaded: %#v", m.Lines)
	}

	// Also verify InitialModel with a filename renders a 1-line tab bar
	single := InitialModel("solo.go", []string{"package solo"}, config.DefaultConfig())
	if !single.showTabBar() || single.tabBarHeight() != 1 {
		t.Fatal("expected tab bar to be visible for an opened file")
	}
}
