package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func initUITestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	run("init")
	run("config", "user.email", "test@larry.dev")
	run("config", "user.name", "Larry Test")
	return dir
}

func TestGitMarkersInFileTreeEditorGutterAndStatusBar(t *testing.T) {
	repo := initUITestGitRepo(t)

	sub := filepath.Join(repo, "src")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(sub, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {\n\tprintln(\"hi\")\n}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := exec.Command("git", "-C", repo, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "-C", repo, "commit", "-m", "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, string(out))
	}

	// Modify file on disk and add an untracked file
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {\n\tprintln(\"edited\")\n}"), 0644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	untracked := filepath.Join(repo, "notes.md")
	if err := os.WriteFile(untracked, []byte("# notes"), 0644); err != nil {
		t.Fatalf("write untracked: %v", err)
	}

	m := newTestModelInDir(t, repo)
	m.Width = 100
	m.Height = 24

	// Open sidebar and verify dirty folder indicator '•' and untracked '?' badge
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	treeOut := stripANSI(m.fileTree.View(30, 20))
	if !strings.Contains(treeOut, "src/") || !strings.Contains(treeOut, "•") {
		t.Fatalf("expected dirty directory badge '•' on src/:\n%s", treeOut)
	}
	if !strings.Contains(treeOut, "notes.md") || !strings.Contains(treeOut, "?") {
		t.Fatalf("expected untracked badge '?' on notes.md:\n%s", treeOut)
	}

	// Open src/main.go and verify file badge 'M' and editor gutter marker '▎' on line 4
	var err error
	m, err = m.openFile(filePath)
	if err != nil {
		t.Fatalf("openFile: %v", err)
	}
	m.fileTree.Focused = false

	treeExpanded := stripANSI(m.fileTree.View(30, 20))
	if !strings.Contains(treeExpanded, "main.go") || !strings.Contains(treeExpanded, "M") {
		t.Fatalf("expected modified badge 'M' on main.go in tree:\n%s", treeExpanded)
	}

	editorOut := stripANSI(m.viewEditor(editorViewConfig{
		width:          m.mainContentWidth(),
		height:         m.editorViewportHeight(),
		hideLeftBorder: true,
	}))
	rows := strings.Split(strings.TrimSuffix(editorOut, "\n"), "\n")
	if len(rows) < 4 {
		t.Fatalf("expected at least 4 rendered lines, got:\n%s", editorOut)
	}
	if strings.Contains(rows[0], "▎") {
		t.Fatalf("expected unchanged line 1 to have no git gutter sign, got %q", rows[0])
	}
	if !strings.Contains(rows[3], "▎") {
		t.Fatalf("expected modified line 4 to show git gutter sign '▎', got %q", rows[3])
	}

	// Status bar should show git branch
	statusOut := stripANSI(m.renderStatusBar())
	if !strings.Contains(statusOut, "⎇ ") {
		t.Fatalf("expected status bar to display git branch icon, got %q", statusOut)
	}
}
