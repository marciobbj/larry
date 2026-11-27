package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"larry/internal/config"
)

func TestNoWrapKeepsConsecutiveLineNumbersAndScrollsHorizontally(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WordWrap = false
	cfg.LineNumbers = true

	longLine := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	m := InitialModel("doc.md", []string{"first", longLine, "third"}, cfg)
	m.Width = 30 // textWidth = 30 - 6 - 1 = 23
	m.Height = 10

	view := stripANSI(m.viewEditor(editorViewConfig{
		width:  m.Width,
		height: m.editorViewportHeight(),
	}))
	renderedLines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")

	// With WordWrap=false, 3 logical lines must render as exactly 3 visual lines (no gaps between line numbers)
	if len(renderedLines) != 3 {
		t.Fatalf("expected 3 visual lines without wrap, got %d:\n%s", len(renderedLines), view)
	}
	if !strings.Contains(renderedLines[0], " 1 ") || !strings.Contains(renderedLines[1], " 2 ") || !strings.Contains(renderedLines[2], " 3 ") {
		t.Fatalf("expected consecutive line numbers 1, 2, 3 without gaps, got:\n%s", view)
	}

	// Move cursor to end of long line -> xOffset should scroll right so the end of the line is visible
	m.CursorRow = 1
	m.CursorCol = len(longLine)
	m = m.updateViewport()
	if m.xOffset <= 0 {
		t.Fatalf("expected positive xOffset after moving to end of long line, got %d", m.xOffset)
	}

	scrolledView := stripANSI(m.viewEditor(editorViewConfig{
		width:  m.Width,
		height: m.editorViewportHeight(),
	}))
	if !strings.Contains(scrolledView, "XYZ") {
		t.Fatalf("expected horizontally scrolled view to show tail of line (XYZ), got:\n%s", scrolledView)
	}
}

func TestToggleWordWrapCommandAndAlignedContinuationGutter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WordWrap = false
	cfg.LineNumbers = true

	longLine := "OpenTune is a high-performance audio playback application specifically engineered"
	m := InitialModel("README.md", []string{longLine, "next line"}, cfg)
	m.Width = 35 // textWidth = 35 - 6 - 1 = 28
	m.Height = 12

	// Ctrl+L enables word wrap
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = updated.(Model)
	if !m.Config.WordWrap {
		t.Fatal("expected Ctrl+L to enable word wrap")
	}
	if m.statusMsg != "Word wrap: ON" {
		t.Fatalf("statusMsg = %q, want %q", m.statusMsg, "Word wrap: ON")
	}

	wrapped := stripANSI(m.viewEditor(editorViewConfig{
		width:  m.Width,
		height: m.editorViewportHeight(),
	}))
	rows := strings.Split(wrapped, "\n")
	if len(rows) <= 2 {
		t.Fatalf("expected long line to wrap into multiple visual rows, got %d:\n%s", len(rows), wrapped)
	}

	// Verify continuation line text aligns flush with first line text (same starting rune column after gutter)
	firstTextCol := len([]rune(rows[0][:strings.Index(rows[0], "OpenTune")]))
	secondTextCol := len([]rune(rows[1][:strings.Index(rows[1], "ce ")]))
	if firstTextCol != secondTextCol {
		t.Fatalf("wrapped continuation line misaligned: line0 col=%d, line1 col=%d\n%s", firstTextCol, secondTextCol, wrapped)
	}

	// Ctrl+L again disables word wrap
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = updated.(Model)
	if m.Config.WordWrap {
		t.Fatal("expected second Ctrl+L to disable word wrap")
	}
	if m.statusMsg != "Word wrap: OFF" {
		t.Fatalf("statusMsg = %q, want %q", m.statusMsg, "Word wrap: OFF")
	}
}
