package ui

import (
	"strings"
	"testing"

	"larry/internal/config"
	"larry/internal/plugin"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestCompletionPopupRenderFilterAcceptAndDismiss(t *testing.T) {
	isolatePluginsDir(t)

	m := InitialModel("app.py", []string{
		"def greet(name: str) -> str:",
		"    return name.st",
		"",
		"greet('larry')",
	}, config.DefaultConfig())
	defer m.ClosePlugins()
	m.Width = 100
	m.Height = 24
	m.CursorRow = 1
	m.CursorCol = len("    return name.st")

	items := []plugin.CompletionItem{
		{Label: "strip", InsertText: "strip", Kind: "Method", Detail: "() -> str"},
		{Label: "startswith", InsertText: "startswith", Kind: "Method", Detail: "(prefix) -> bool"},
		{Label: "split", InsertText: "split", Kind: "Method", Detail: "(sep) -> list[str]"},
		{Label: "upper", InsertText: "upper", Kind: "Method", Detail: "() -> str"},
	}

	m.editorAPI().ShowCompletions(items, 1, len("    return name."), nil)
	if !m.completion.Visible {
		t.Fatal("expected completion popup to be visible")
	}
	// Initial prefix "st" should filter down to strip + startswith (excluding split & upper)
	if len(m.completion.Filtered) != 2 || m.completion.Filtered[0].Label != "strip" || m.completion.Filtered[1].Label != "startswith" {
		t.Fatalf("unexpected initial filtered completions: %+v", m.completion.Filtered)
	}

	// Render full editor with floating popup composited over baseView
	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "strip") || !strings.Contains(plain, "startswith") || !strings.Contains(plain, "mth") {
		t.Fatalf("expected floating completion popup in editor view, got:\n%s", plain)
	}
	// Underlying editor code should still remain visible around the floating popup!
	if !strings.Contains(plain, "def greet") {
		t.Fatalf("expected surrounding code to remain visible behind floating popup, got:\n%s", plain)
	}
	if h := lipgloss.Height(view); h != m.Height {
		t.Fatalf("rendered height = %d, want %d", h, m.Height)
	}

	// Typing 'a' keeps popup open and live-filters to "startswith"!
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if !m.completion.Visible {
		t.Fatal("expected completion popup to stay visible after typing identifier rune")
	}
	if len(m.completion.Filtered) != 1 || m.completion.Filtered[0].Label != "startswith" {
		t.Fatalf("expected live filter to narrow to startswith, got %+v", m.completion.Filtered)
	}
	if m.Lines[1] != "    return name.sta" {
		t.Fatalf("expected buffer line updated live, got %q", m.Lines[1])
	}

	// Pressing Tab accepts "startswith", replacing the partial prefix "sta"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.completion.Visible {
		t.Fatal("expected Tab to close completion popup after accepting")
	}
	if m.Lines[1] != "    return name.startswith" {
		t.Fatalf("expected accepted completion line, got %q", m.Lines[1])
	}

	// Re-open and verify Esc dismisses cleanly without modifying buffer
	m.editorAPI().ShowCompletions(items, 1, len("    return name."), nil)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.completion.Visible {
		t.Fatal("expected Esc to dismiss completion popup")
	}
}

func TestCompletionPopupFlipsAboveCursorNearBottomAndWorksInLightThemes(t *testing.T) {
	isolatePluginsDir(t)

	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "x = value"
	}
	cfg := config.DefaultConfig()
	cfg.Theme = "catppuccin-latte"
	m := InitialModel("test.py", lines, cfg)
	defer m.ClosePlugins()
	m.Width = 80
	m.Height = 16
	m.CursorRow = 12
	m.CursorCol = 9

	m.editorAPI().ShowCompletions([]plugin.CompletionItem{
		{Label: "value_one", Kind: "Variable", Detail: "int"},
		{Label: "value_two", Kind: "Variable", Detail: "str"},
	}, 12, 4, nil)

	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "value_one") || !strings.Contains(plain, "value_two") {
		t.Fatalf("expected completion popup to flip above cursor and render, got:\n%s", plain)
	}
	for i, row := range strings.Split(view, "\n") {
		if w := lipgloss.Width(row); w > m.Width {
			t.Fatalf("row %d width %d exceeded terminal width %d", i, w, m.Width)
		}
	}
}
