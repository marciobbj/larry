package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
	"larry/internal/search"
)

func TestEditorSanitizesControlCharsAndPreservesBoundaries(t *testing.T) {
	lines := []string{
		"before\rafter\r\rOVERWRITE_SIDEBAR",
		"escape\x1b[31mINJECTED\x1b[0m text",
		"nul\x00vt\x0bff\x0cbs\x08del\x7fbidi\u202ereversed",
		strings.Repeat("abc\rdef\x1b[2J", 20),
	}

	for _, wrap := range []bool{false, true} {
		cfg := config.DefaultConfig()
		cfg.WordWrap = wrap
		cfg.LineNumbers = true

		m := InitialModel("control.txt", lines, cfg)
		m.Width = 70
		m.Height = 16
		m.OpenDirectory(t.TempDir())
		m.fileTree.Focused = false

		editorW := m.mainContentWidth()
		rawEditor := m.viewEditor(editorViewConfig{
			width:          editorW,
			height:         m.editorViewportHeight(),
			hideLeftBorder: m.sidebarWidth() > 0,
		})

		if strings.ContainsRune(rawEditor, '\r') {
			t.Fatalf("wrap=%v: raw CR found in rendered editor output", wrap)
		}
		if strings.ContainsRune(rawEditor, '\x00') ||
			strings.ContainsRune(rawEditor, '\x0b') ||
			strings.ContainsRune(rawEditor, '\x0c') ||
			strings.ContainsRune(rawEditor, '\x08') ||
			strings.ContainsRune(rawEditor, '\u202e') {
			t.Fatalf("wrap=%v: unsanitized control/bidi rune found in editor output", wrap)
		}
		if strings.Contains(rawEditor, "\x1b[31m") || strings.Contains(rawEditor, "\x1b[2J") {
			t.Fatalf("wrap=%v: file-injected ANSI escape sequence leaked into output", wrap)
		}

		for idx, row := range strings.Split(rawEditor, "\n") {
			if w := lipgloss.Width(row); w > editorW {
				t.Fatalf("wrap=%v: editor row %d width %d exceeds editorW %d: %q", wrap, idx, w, editorW, stripANSI(row))
			}
		}

		fullView := m.View()
		for idx, row := range strings.Split(fullView, "\n") {
			if w := lipgloss.Width(row); w > m.Width {
				t.Fatalf("wrap=%v: full View row %d width %d exceeds terminal width %d: %q", wrap, idx, w, m.Width, stripANSI(row))
			}
		}
	}
}

func TestEditorWideRunesRespectBoundsWrapAndNoWrap(t *testing.T) {
	wideLine := strings.Repeat("日本語跡吨瀾鷏렛🚀", 12)
	mixedOdd := "a" + strings.Repeat("漢", 40) + "z"

	for _, wrap := range []bool{false, true} {
		for _, termW := range []int{49, 50, 75, 76} {
			cfg := config.DefaultConfig()
			cfg.WordWrap = wrap
			cfg.LineNumbers = true

			m := InitialModel("wide.txt", []string{wideLine, mixedOdd, "plain"}, cfg)
			m.Width = termW
			m.Height = 14
			m.OpenDirectory(t.TempDir())
			m.fileTree.Focused = false

			if !wrap {
				// Test straddling a 2-column rune on the left scroll boundary.
				m.xOffset = 1
			}

			editorW := m.mainContentWidth()
			rawEditor := m.viewEditor(editorViewConfig{
				width:          editorW,
				height:         m.editorViewportHeight(),
				hideLeftBorder: m.sidebarWidth() > 0,
			})

			for idx, row := range strings.Split(rawEditor, "\n") {
				if w := lipgloss.Width(row); w > editorW {
					t.Fatalf("wrap=%v termW=%d: editor row %d width %d exceeds editorW %d: %q",
						wrap, termW, idx, w, editorW, stripANSI(row))
				}
			}

			fullView := m.View()
			rows := strings.Split(fullView, "\n")
			for idx, row := range rows {
				if w := lipgloss.Width(row); w > m.Width {
					t.Fatalf("wrap=%v termW=%d: full View row %d width %d exceeds m.Width %d: %q",
						wrap, termW, idx, w, m.Width, stripANSI(row))
				}
			}
		}
	}
}

func TestEditorLargeLineCountGutterDoesNotOverflow(t *testing.T) {
	const total = 12000
	lines := make([]string, total)
	longText := strings.Repeat("0123456789", 20)
	for i := range lines {
		lines[i] = longText
	}

	for _, wrap := range []bool{false, true} {
		cfg := config.DefaultConfig()
		cfg.WordWrap = wrap
		cfg.LineNumbers = true

		m := InitialModel("big.txt", lines, cfg)
		m.Width = 80
		m.Height = 16
		m.OpenDirectory(t.TempDir())
		m.fileTree.Focused = false
		m.CursorRow = 11995
		m = m.updateViewport()

		editorW := m.mainContentWidth()
		rawEditor := m.viewEditor(editorViewConfig{
			width:          editorW,
			height:         m.editorViewportHeight(),
			hideLeftBorder: m.sidebarWidth() > 0,
		})

		for idx, row := range strings.Split(rawEditor, "\n") {
			if w := lipgloss.Width(row); w > editorW {
				t.Fatalf("wrap=%v: row %d width %d exceeds editorW %d (5-digit gutter overflow): %q",
					wrap, idx, w, editorW, stripANSI(row))
			}
		}
	}
}

func TestOpenFileRejectsBinaryAndNormalizesCRLF(t *testing.T) {
	dir := t.TempDir()

	// PDF whose first 10 KiB are ASCII before compressed streams begin
	pdfPath := filepath.Join(dir, "doc.pdf")
	pdfData := append([]byte("%PDF-1.7\n"), []byte(strings.Repeat("% comment line\n", 700))...)
	pdfData = append(pdfData, []byte("stream\n\x00\xff\xfe\x80endstream\n")...)
	if err := os.WriteFile(pdfPath, pdfData, 0644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	if !search.IsBinary(pdfPath) {
		t.Fatal("expected IsBinary to detect PDF file")
	}

	// Binary file with NUL byte
	binPath := filepath.Join(dir, "app.bin")
	if err := os.WriteFile(binPath, []byte{0x7f, 'E', 'L', 'F', 0x00, 0x02}, 0644); err != nil {
		t.Fatalf("write bin: %v", err)
	}

	// Normal UTF-8 text file with Portuguese accents, CJK, emoji, and Windows CRLF
	txtPath := filepath.Join(dir, "notes.txt")
	txtContent := "Olá, mundo!\r\nSegunda linha 日本語 🚀\r\nTerceira linha\r\n"
	if err := os.WriteFile(txtPath, []byte(txtContent), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if search.IsBinary(txtPath) {
		t.Fatal("expected UTF-8 CRLF text file NOT to be treated as binary")
	}

	m := newTestModelInDir(t, dir)
	m.Lines = []string{"keep me"}
	m.FileName = "existing.txt"
	m.showWelcome = false

	// Attempting to open PDF must fail gracefully without overwriting buffer
	m2, err := m.openFile(pdfPath)
	if err == nil {
		t.Fatal("expected error when opening PDF file")
	}
	if !strings.Contains(m2.statusMsg, "Binary file not supported") {
		t.Fatalf("unexpected statusMsg: %q", m2.statusMsg)
	}
	if m2.FileName != "existing.txt" || len(m2.Lines) != 1 || m2.Lines[0] != "keep me" {
		t.Fatalf("buffer was corrupted after rejected binary open: file=%q lines=%v", m2.FileName, m2.Lines)
	}

	// Attempting to open ELF binary must also fail gracefully
	if _, err := m.openFile(binPath); err == nil {
		t.Fatal("expected error when opening ELF binary file")
	}

	// Opening normal UTF-8 CRLF text file succeeds and strips \r
	m3, err := m.openFile(txtPath)
	if err != nil {
		t.Fatalf("openFile txt: %v", err)
	}
	for i, line := range m3.Lines {
		if strings.ContainsRune(line, '\r') {
			t.Fatalf("line %d still contains CR: %q", i, line)
		}
	}
	if m3.Lines[0] != "Olá, mundo!" || m3.Lines[1] != "Segunda linha 日本語 🚀" {
		t.Fatalf("unexpected loaded lines: %#v", m3.Lines)
	}
}
