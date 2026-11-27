package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
	"larry/internal/search"
)

func TestGlobalFinderModalLayoutAndModes(t *testing.T) {
	cfg := config.DefaultConfig()
	m := InitialModel("main.go", []string{"package main"}, cfg)
	m.Width = 100
	m.Height = 30
	m.showWelcome = false

	m, _ = m.openGlobalFinder()
	m.finder.loading = false
	m.finder.results = []search.FinderResult{
		{Mode: search.ModeFiles, File: &search.FileResult{Path: "internal/ui/view_editor.go"}},
		{Mode: search.ModeFiles, File: &search.FileResult{Path: "internal/ui/finder.go"}},
		{Mode: search.ModeFiles, File: &search.FileResult{Path: "cmd/larry/main.go"}},
	}
	m.finder.cursor = 1

	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "Global Finder") {
		t.Fatalf("expected Global Finder header, got:\n%s", plain)
	}
	if !strings.Contains(plain, "FILES") || !strings.Contains(plain, "GREP") {
		t.Fatalf("expected FILES and GREP mode pills, got:\n%s", plain)
	}
	if !strings.Contains(plain, "2/3") {
		t.Fatalf("expected result counter 2/3, got:\n%s", plain)
	}
	if !strings.Contains(plain, "▸") {
		t.Fatalf("expected active selection indicator, got:\n%s", plain)
	}

	rows := strings.Split(view, "\n")
	if len(rows) > m.Height {
		t.Fatalf("finder modal height %d exceeded terminal height %d", len(rows), m.Height)
	}
	for i, r := range rows {
		if w := lipgloss.Width(r); w > m.Width {
			t.Fatalf("finder modal row %d width %d exceeded terminal width %d", i, w, m.Width)
		}
	}

	// Switch to grep mode with control chars & wide runes in content
	m.finder.mode = FinderModeGrep
	m.finder.textInput.SetValue("hello")
	m.finder.results = []search.FinderResult{
		{
			Mode: search.ModeGrep,
			Grep: &search.GrepResult{
				Path:    "internal/ui/very/deeply/nested/path/example.go",
				Line:    42,
				Content: "\t// hello 日本語\r\x1b[31m world",
			},
		},
	}
	m.finder.cursor = 0

	grepView := m.View()
	if strings.ContainsRune(grepView, '\r') || strings.Contains(grepView, "\x1b[31m") {
		t.Fatalf("grep preview leaked raw CR or ANSI escape sequence")
	}
	for i, r := range strings.Split(grepView, "\n") {
		if w := lipgloss.Width(r); w > m.Width {
			t.Fatalf("grep finder row %d width %d exceeded terminal width %d", i, w, m.Width)
		}
	}
}

func TestHelpMenuLayoutResponsiveAndScrollable(t *testing.T) {
	cfg := config.DefaultConfig()

	// 1. Tall wide terminal: 2-column layout fits completely without clipping
	m := InitialModel("main.go", []string{"package main"}, cfg)
	m.Width = 110
	m.Height = 36
	m.showHelp = true

	view := m.View()
	plain := stripANSI(view)
	for _, want := range []string{"LARRY", "Keyboard Shortcuts", "Files & Tabs", "Search & Replace", "File Tree", "Editing", "Multi-Cursor", "Navigation & Selection"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected help menu to contain %q, got:\n%s", want, plain)
		}
	}
	rows := strings.Split(view, "\n")
	if len(rows) > m.Height {
		t.Fatalf("help menu height %d exceeded terminal height %d", len(rows), m.Height)
	}
	for i, r := range rows {
		if w := lipgloss.Width(r); w > m.Width {
			t.Fatalf("help menu row %d width %d exceeded terminal width %d", i, w, m.Width)
		}
	}

	// 2. Standard 24-row terminal: never clips border/header/footer and supports scrolling
	mShort := InitialModel("main.go", []string{"package main"}, cfg)
	mShort.Width = 80
	mShort.Height = 24
	mShort.showHelp = true

	shortView := mShort.View()
	shortPlain := stripANSI(shortView)
	if !strings.Contains(shortPlain, "Keyboard Shortcuts") || !strings.Contains(shortPlain, "Scroll") {
		t.Fatalf("expected scrollable header & footer on 24-row terminal, got:\n%s", shortPlain)
	}
	if h := len(strings.Split(shortView, "\n")); h > mShort.Height {
		t.Fatalf("24-row help menu overflowed height: got %d > %d", h, mShort.Height)
	}

	// Scrolling down with 'j' advances viewport cleanly
	updated, _ := mShort.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	mShort = updated.(Model)
	if mShort.helpScroll != 1 {
		t.Fatalf("expected helpScroll=1 after pressing j, got %d", mShort.helpScroll)
	}

	// Pressing 'q' closes help and resets scroll
	updated, _ = mShort.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mShort = updated.(Model)
	if mShort.showHelp || mShort.helpScroll != 0 {
		t.Fatalf("expected q to close help and reset scroll, got showHelp=%v scroll=%d", mShort.showHelp, mShort.helpScroll)
	}

	// 3. Narrow terminal (< 62 inner width): collapses cleanly to single column without overflow
	mNarrow := InitialModel("main.go", []string{"package main"}, cfg)
	mNarrow.Width = 50
	mNarrow.Height = 24
	mNarrow.showHelp = true

	narrowView := mNarrow.View()
	for i, r := range strings.Split(narrowView, "\n") {
		if w := lipgloss.Width(r); w > mNarrow.Width {
			t.Fatalf("narrow help row %d width %d exceeded m.Width %d", i, w, mNarrow.Width)
		}
	}
}

func TestGlobalFinderCachesFilesAndNavigatesFastWithLargeBuffer(t *testing.T) {
	cfg := config.DefaultConfig()
	bigLines := make([]string, 20000)
	for i := range bigLines {
		bigLines[i] = fmt.Sprintf("func line_%d() { println(\"value_%d\") }", i, i)
	}

	m := InitialModel("huge.go", bigLines, cfg)
	m.Width = 110
	m.Height = 32
	m.showWelcome = false

	m, cmd := m.openGlobalFinder()
	if cmd == nil {
		t.Fatal("expected initial finder scan command")
	}

	// Simulate initial scan completion with 2000 candidate paths
	fakeFiles := make([]string, 2000)
	for i := range fakeFiles {
		fakeFiles[i] = fmt.Sprintf("pkg/module_%02d/file_%04d.go", i%20, i)
	}
	updated, _ := m.Update(searchMsg{
		seq:   m.finder.searchSeq,
		files: fakeFiles,
		results: []search.FinderResult{
			{Mode: search.ModeFiles, File: &search.FileResult{Path: fakeFiles[0]}},
		},
	})
	m = updated.(Model)
	if !m.finder.allFilesLoaded || len(m.finder.allFiles) != len(fakeFiles) {
		t.Fatalf("expected allFiles cache to persist, loaded=%v len=%d", m.finder.allFilesLoaded, len(m.finder.allFiles))
	}

	// Typing a query must filter synchronously in-memory (nil cmd)
	updated, nextCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(Model)
	if nextCmd != nil {
		t.Fatalf("expected synchronous cached filter (nil cmd), got non-nil cmd")
	}
	if len(m.finder.results) != finderMaxResults {
		t.Fatalf("expected %d capped results, got %d", finderMaxResults, len(m.finder.results))
	}

	// Navigating 50 steps down + full View() render must stay blazing fast even with 20k-line buffer open
	start := time.Now()
	for i := 0; i < 50; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
		_ = m.View()
	}
	elapsed := time.Since(start)
	if m.finder.cursor != 50 {
		t.Fatalf("expected cursor at 50, got %d", m.finder.cursor)
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("finder navigation + View() too slow: 50 frames took %v", elapsed)
	}
}

