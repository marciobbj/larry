package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
)

func TestEditorViewportHeightUsesWrappedStatusBarHeight(t *testing.T) {
	m := InitialModel("very-long-file-name.txt", []string{"first line", "second line"}, config.DefaultConfig())
	m.Width = 24
	m.Height = 10

	statusHeight := lipgloss.Height(m.renderStatusBar())
	if statusHeight < 2 {
		t.Fatalf("expected wrapped status bar for narrow width, got height %d", statusHeight)
	}

	want := m.Height - statusHeight
	if got := m.editorViewportHeight(); got != want {
		t.Fatalf("editorViewportHeight() = %d, want %d", got, want)
	}
}

func TestEditorViewportHeightReservesPromptLines(t *testing.T) {
	m := InitialModel("file.txt", []string{"first line"}, config.DefaultConfig())
	m.Width = 24
	m.Height = 10
	m.searching = true

	want := m.Height - lipgloss.Height(m.renderStatusBar()) - 2
	if want < 1 {
		want = 1
	}

	if got := m.editorViewportHeight(); got != want {
		t.Fatalf("editorViewportHeight() = %d, want %d", got, want)
	}
}
