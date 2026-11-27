package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init")
	run("config", "user.email", "test@larry.dev")
	run("config", "user.name", "Larry Test")
	return dir
}

func TestGetRepoStatusAndLoadFileBaseAndLiveDiff(t *testing.T) {
	repo := initTestRepo(t)

	subDir := filepath.Join(repo, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	trackedFile := filepath.Join(subDir, "app.go")
	initial := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(trackedFile, []byte(initial), 0644); err != nil {
		t.Fatalf("write tracked: %v", err)
	}

	cmd := exec.Command("git", "-C", repo, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "-C", repo, "commit", "-m", "initial")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, string(out))
	}

	// Modify tracked file and create an untracked file
	if err := os.WriteFile(trackedFile, []byte("line1\nline2-edited\nline3\nline4\nline5"), 0644); err != nil {
		t.Fatalf("rewrite tracked: %v", err)
	}
	untrackedFile := filepath.Join(repo, "new.txt")
	if err := os.WriteFile(untrackedFile, []byte("hello\nworld"), 0644); err != nil {
		t.Fatalf("write untracked: %v", err)
	}

	status := GetRepoStatus(repo)
	if !status.InRepo {
		t.Fatal("expected InRepo=true")
	}
	if status.Branch == "" {
		t.Fatal("expected non-empty branch name")
	}
	if got := status.FileStatuses[trackedFile]; got != StatusModified {
		t.Fatalf("tracked file status = %q, want %q", got, StatusModified)
	}
	if got := status.FileStatuses[untrackedFile]; got != StatusUntracked {
		t.Fatalf("untracked file status = %q, want %q", got, StatusUntracked)
	}
	if !status.DirtyDirs[subDir] {
		t.Fatalf("expected parent dir %s to be marked dirty", subDir)
	}

	// Load HEAD base and test live line-level diff: modify line 1, delete line 3, add a new line at the end
	base := LoadFileBase(trackedFile)
	if !base.InRepo || !base.Tracked {
		t.Fatalf("expected tracked file base, got %+v", base)
	}

	liveLines := []string{"line1", "line2-edited", "line3", "line5", "line6-added"}
	changes := ComputeLineChanges(base, liveLines)

	if changes[0] != LineUnchanged {
		t.Fatalf("line 0 = %v, want Unchanged", changes[0])
	}
	if changes[1] != LineModified {
		t.Fatalf("line 1 = %v, want Modified", changes[1])
	}
	if changes[2] != LineDeletedBelow {
		t.Fatalf("line 2 = %v, want DeletedBelow (line4 deleted after line3)", changes[2])
	}
	if changes[4] != LineAdded {
		t.Fatalf("line 4 = %v, want Added", changes[4])
	}

	// Untracked file marks all lines as added
	untrackedBase := LoadFileBase(untrackedFile)
	if !untrackedBase.InRepo || untrackedBase.Tracked {
		t.Fatalf("expected untracked base, got %+v", untrackedBase)
	}
	uChanges := ComputeLineChanges(untrackedBase, []string{"hello", "world"})
	if uChanges[0] != LineAdded || uChanges[1] != LineAdded {
		t.Fatalf("expected untracked lines to be LineAdded, got %+v", uChanges)
	}
}

func TestNonGitDirectoryReturnsCleanEmptyStatus(t *testing.T) {
	dir := t.TempDir()
	status := GetRepoStatus(dir)
	if status.InRepo {
		t.Fatal("expected InRepo=false outside git repo")
	}
	f := filepath.Join(dir, "plain.txt")
	_ = os.WriteFile(f, []byte("hello"), 0644)
	base := LoadFileBase(f)
	if base.InRepo {
		t.Fatal("expected base.InRepo=false outside git repo")
	}
	if changes := ComputeLineChanges(base, []string{"changed"}); changes != nil {
		t.Fatalf("expected nil changes outside git repo, got %+v", changes)
	}
}
