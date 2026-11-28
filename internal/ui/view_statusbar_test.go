package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
)

func TestStatusBarStaysSingleLineAndExactWidth(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "void main() {}"
	}

	for _, w := range []int{24, 40, 60, 80, 110, 160} {
		m := InitialModel("lib/presentation/screens/library/library_screen.dart", lines, config.DefaultConfig())
		m.Width = w
		m.Height = 20
		m.Modified = true
		m.CursorRow = 41
		m.CursorCol = 11
		m.Config.WordWrap = true
		m.fileTree.GitBranch = "heads/main"

		bar := m.renderStatusBar()
		if gotH := lipgloss.Height(bar); gotH != 1 {
			t.Fatalf("width=%d: expected single-line status bar (height 1), got height %d:\n%s", w, gotH, stripANSI(bar))
		}
		if gotW := lipgloss.Width(bar); gotW != w {
			t.Fatalf("width=%d: expected exact bar width %d, got %d (%q)", w, w, gotW, stripANSI(bar))
		}

		wantViewport := m.Height - 1 - m.tabBarHeight()
		if got := m.editorViewportHeight(); got != wantViewport {
			t.Fatalf("width=%d: editorViewportHeight() = %d, want %d", w, got, wantViewport)
		}
	}
}

func TestStatusBarShowsTelemetrySmartPathAndModes(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}

	m := InitialModel("lib/presentation/screens/library/library_screen.dart", lines, config.DefaultConfig())
	m.Width = 110
	m.Height = 30
	m.Modified = true
	m.CursorRow = 41
	m.CursorCol = 11
	m.statusMsg = ""
	m.fileTree.GitBranch = "heads/main"

	plain := stripANSI(m.renderStatusBar())
	for _, want := range []string{"EDIT", "⎇ main", "library_screen.dart ●", "Dart", "Ln 42, Col 12", "41%", "Ctrl+h Help"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected status bar to contain %q, got %q", want, plain)
		}
	}
	if strings.Contains(plain, "heads/main") {
		t.Fatalf("expected heads/ prefix to be stripped from branch, got %q", plain)
	}

	// File tree focus switches mode badge and shows tree navigation hints
	m.fileTree.Visible = true
	m.fileTree.Focused = true
	treePlain := stripANSI(m.renderStatusBar())
	for _, want := range []string{"TREE", "Enter Open", "Esc Editor", "Ctrl+h Help"} {
		if !strings.Contains(treePlain, want) {
			t.Fatalf("expected tree status bar to contain %q, got %q", want, treePlain)
		}
	}

	// Alt leader key adapts Help pill
	m.fileTree.Focused = false
	m.Config.LeaderKey = "alt"
	altPlain := stripANSI(m.renderStatusBar())
	if !strings.Contains(altPlain, "Alt+h Help") {
		t.Fatalf("expected alt leader help pill, got %q", altPlain)
	}
}

func TestFormatCompactPath(t *testing.T) {
	dir, base := formatCompactPath("lib/presentation/screens/library/library_screen.dart", 34)
	if got := dir + base; got != "…/library/library_screen.dart" {
		t.Fatalf("formatCompactPath() = %q, want %q", got, "…/library/library_screen.dart")
	}

	dir, base = formatCompactPath("lib/presentation/screens/library/library_screen.dart", 22)
	if got := dir + base; got != "…/library_screen.dart" {
		t.Fatalf("formatCompactPath() = %q, want %q", got, "…/library_screen.dart")
	}
}

func TestEditorViewportHeightReservesPromptLines(t *testing.T) {
	m := InitialModel("file.txt", []string{"first line"}, config.DefaultConfig())
	m.Width = 24
	m.Height = 10
	m.searching = true

	want := m.Height - lipgloss.Height(m.renderStatusBar()) - 2 - m.tabBarHeight()
	if want < 1 {
		want = 1
	}

	if got := m.editorViewportHeight(); got != want {
		t.Fatalf("editorViewportHeight() = %d, want %d", got, want)
	}
}
