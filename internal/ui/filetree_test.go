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

func setupTestTreeDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		filepath.Join(root, "alpha"),
		filepath.Join(root, "alpha", "nested"),
		filepath.Join(root, "beta"),
		filepath.Join(root, ".hidden_dir"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "README.md"):                  "# Hello\nWorld",
		filepath.Join(root, "main.go"):                    "package main\n\nfunc main() {}",
		filepath.Join(root, ".env"):                       "SECRET=1",
		filepath.Join(root, "alpha", "a.txt"):             "alpha content",
		filepath.Join(root, "alpha", "nested", "deep.go"): "package nested",
		filepath.Join(root, "beta", "b.txt"):              "beta content",
	}
	for p, content := range files {
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	return root
}

func newTestModelInDir(t *testing.T, root string) Model {
	t.Helper()
	m := InitialModel("", []string{""}, config.DefaultConfig())
	m.Width = 100
	m.Height = 24
	m.fileTree = NewFileTreeModel(root)
	m.fileTree.Height = m.editorViewportHeight()
	return m
}

func TestOpenCommandTogglesLateralFileTree(t *testing.T) {
	root := setupTestTreeDir(t)
	m := newTestModelInDir(t, root)

	if m.fileTree.Visible || m.fileTree.Focused {
		t.Fatal("expected file tree to start collapsed and unfocused")
	}

	// Ctrl+O opens and focuses the sidebar
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	if !m.fileTree.Visible || !m.fileTree.Focused {
		t.Fatal("expected Ctrl+O to open and focus the file tree")
	}
	if !m.showWelcome {
		t.Fatal("expected welcome screen to stay visible beside sidebar until a file is opened or edited")
	}

	// Verify directories come first, sorted alphabetically, and dotfiles are hidden by default
	wantNames := []string{"alpha", "beta", "main.go", "README.md"}
	if len(m.fileTree.Nodes) != len(wantNames) {
		t.Fatalf("got %d nodes (%+v), want %d", len(m.fileTree.Nodes), m.fileTree.Nodes, len(wantNames))
	}
	for i, want := range wantNames {
		if m.fileTree.Nodes[i].Name != want {
			t.Fatalf("node[%d] = %q, want %q", i, m.fileTree.Nodes[i].Name, want)
		}
	}

	// Ctrl+O again collapses the sidebar
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	if m.fileTree.Visible || m.fileTree.Focused {
		t.Fatal("expected second Ctrl+O to collapse and unfocus the file tree")
	}
}

func TestFileTreeExpandCollapseAndOpenFile(t *testing.T) {
	root := setupTestTreeDir(t)
	m := newTestModelInDir(t, root)

	// Open tree (cursor starts at index 0 -> "alpha/")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)

	// Press Enter on "alpha" directory to expand it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if !m.fileTree.Nodes[0].Expanded {
		t.Fatal("expected alpha directory to be expanded")
	}
	// Expected nodes: alpha, nested, a.txt, beta, main.go, README.md
	if len(m.fileTree.Nodes) != 6 {
		t.Fatalf("got %d nodes after expanding alpha, want 6", len(m.fileTree.Nodes))
	}
	if m.fileTree.Nodes[1].Name != "nested" || m.fileTree.Nodes[1].Depth != 1 {
		t.Fatalf("expected nested dir at depth 1, got %+v", m.fileTree.Nodes[1])
	}
	if m.fileTree.Nodes[2].Name != "a.txt" || m.fileTree.Nodes[2].Depth != 1 {
		t.Fatalf("expected a.txt at depth 1, got %+v", m.fileTree.Nodes[2])
	}

	// Move down to a.txt (index 2) and press Left -> should jump to parent "alpha" (index 0)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.fileTree.Cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.fileTree.Cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(Model)
	if m.fileTree.Cursor != 0 {
		t.Fatalf("expected Left on child file to jump to parent alpha (0), got %d", m.fileTree.Cursor)
	}

	// Press Left again on expanded "alpha" -> collapses it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(Model)
	if len(m.fileTree.Nodes) != 4 {
		t.Fatalf("expected alpha to collapse back to 4 nodes, got %d", len(m.fileTree.Nodes))
	}

	// Expand with Right, enter first child with Right, then move down to a.txt and open with Enter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	if m.fileTree.Cursor != 1 {
		t.Fatalf("expected Right on expanded dir to move to first child (1), got %d", m.fileTree.Cursor)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if len(m.Lines) != 1 || m.Lines[0] != "alpha content" {
		t.Fatalf("opened file lines = %#v, want [\"alpha content\"]", m.Lines)
	}
	if m.showWelcome {
		t.Fatal("expected welcome screen to be dismissed after opening a file")
	}
	if !m.fileTree.Visible {
		t.Fatal("expected file tree sidebar to stay visible after opening a file (Neovim style)")
	}
	if m.fileTree.Focused {
		t.Fatal("expected focus to move to the editor after pressing Enter on a file")
	}

	// Collapse and reopen: expanded directory state must be preserved
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	if m.fileTree.Visible {
		t.Fatal("expected Ctrl+O from editor to collapse the sidebar")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	if !m.fileTree.Visible || !m.fileTree.Focused {
		t.Fatal("expected Ctrl+O to reopen and focus sidebar")
	}
	if !m.fileTree.Nodes[0].Expanded {
		t.Fatal("expected expanded directory state to persist across collapse/reopen")
	}
}

func TestFileTreeSpaceOpensWithoutLosingFocusAndEscReturnsToEditor(t *testing.T) {
	root := setupTestTreeDir(t)
	m := newTestModelInDir(t, root)

	// Open tree and move to main.go (index 2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	// Space opens file while keeping focus in tree
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	if m.Lines[0] != "package main" {
		t.Fatalf("expected main.go to load, got %#v", m.Lines)
	}
	if !m.fileTree.Focused {
		t.Fatal("expected Space to keep focus in the file tree")
	}

	// Esc unfocuses tree without collapsing it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if !m.fileTree.Visible || m.fileTree.Focused {
		t.Fatal("expected Esc to return focus to editor while keeping tree visible")
	}

	// Ctrl+E refocuses the open tree
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(Model)
	if !m.fileTree.Visible || !m.fileTree.Focused {
		t.Fatal("expected Ctrl+E to refocus the open file tree")
	}

	// 'q' in focused tree collapses it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if m.fileTree.Visible || m.fileTree.Focused {
		t.Fatal("expected q to collapse the file tree")
	}
}

func TestFileTreeHiddenToggleAndResizeAndLayout(t *testing.T) {
	root := setupTestTreeDir(t)
	m := newTestModelInDir(t, root)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)

	// Toggle hidden files on with '.'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	m = updated.(Model)
	if !m.fileTree.ShowHidden {
		t.Fatal("expected ShowHidden to be true after pressing '.'")
	}
	if len(m.fileTree.Nodes) != 6 {
		t.Fatalf("expected 6 nodes with hidden files visible, got %d", len(m.fileTree.Nodes))
	}

	// Resize wider and narrower
	beforeW := m.fileTree.Width
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = updated.(Model)
	if m.fileTree.Width <= beforeW {
		t.Fatalf("expected ']' to widen sidebar: before=%d after=%d", beforeW, m.fileTree.Width)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = updated.(Model)
	if m.fileTree.Width != beforeW {
		t.Fatalf("expected '[' to narrow sidebar back to %d, got %d", beforeW, m.fileTree.Width)
	}

	// Render View() in welcome mode and verify exact height
	rendered := m.View()
	if !strings.Contains(rendered, "alpha/") {
		t.Fatal("expected rendered view to include directory alpha/")
	}
	if h := lipgloss.Height(rendered); h != m.Height {
		t.Fatalf("rendered welcome+tree view height = %d, want %d", h, m.Height)
	}

	// Open main.go and verify editor+tree rendered view has exact height and content
	m, err := m.openFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatalf("openFile: %v", err)
	}
	m.fileTree.Focused = false
	editorRendered := m.View()
	if !strings.Contains(editorRendered, "alpha/") || !strings.Contains(editorRendered, "package") {
		t.Fatal("expected editor+tree view to contain both file tree and file contents")
	}
	if h := lipgloss.Height(editorRendered); h != m.Height {
		t.Fatalf("rendered editor+tree view height = %d, want %d", h, m.Height)
	}

	// Narrow terminal hides sidebar gracefully without panic
	m.Width = 24
	if got := m.sidebarWidth(); got != 0 {
		t.Fatalf("expected sidebarWidth() == 0 for narrow terminal (24), got %d", got)
	}
	_ = m.View()
}

func TestWelcomeScreenAlignmentLogoAndVersion(t *testing.T) {
	root := setupTestTreeDir(t)
	m := newTestModelInDir(t, root)
	m.Width = 100
	m.Height = 24

	// Open lateral tree alongside welcome screen
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)

	welcome := m.viewWelcome()

	// Full ASCII LARRY logo must remain visible even on 24-row terminals
	if !strings.Contains(welcome, "███████ ██   ██ ██   ██ ██   ██    ██") {
		t.Fatal("expected full ASCII LARRY logo to stay visible in welcome screen")
	}

	// Version must be displayed
	if !strings.Contains(welcome, "larry "+AppVersion) {
		t.Fatalf("expected version %q in welcome screen", "larry "+AppVersion)
	}

	// All Quick Actions keys must start at the exact same visual column
	var keyCols []int
	for _, line := range strings.Split(welcome, "\n") {
		idx := strings.Index(line, "Ctrl+")
		if idx >= 0 {
			// Count plain runes before "Ctrl+" by stripping ANSI codes
			prefix := stripANSI(line[:idx])
			keyCols = append(keyCols, len([]rune(prefix)))
		}
	}
	if len(keyCols) < 5 {
		t.Fatalf("expected at least 5 quick action shortcuts, found %d", len(keyCols))
	}
	for i := 1; i < len(keyCols); i++ {
		if keyCols[i] != keyCols[0] {
			t.Fatalf("quick action keys are misaligned: cols=%v", keyCols)
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			inEsc = true
			i++
			continue
		}
		if inEsc {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
