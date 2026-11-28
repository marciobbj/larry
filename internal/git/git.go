package git

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileStatus represents a file's git working-tree/index status.
type FileStatus string

const (
	StatusNone      FileStatus = ""
	StatusModified  FileStatus = "M"
	StatusAdded     FileStatus = "A"
	StatusUntracked FileStatus = "?"
	StatusDeleted   FileStatus = "D"
	StatusRenamed   FileStatus = "R"
	StatusConflict  FileStatus = "U"
)

// LineChange describes the git diff state of a single line in the editor buffer.
type LineChange int

const (
	LineUnchanged LineChange = iota
	LineAdded
	LineModified
	LineDeletedBelow
)

// RepoStatus summarizes repository-level git state for a workspace directory.
type RepoStatus struct {
	InRepo       bool
	RepoRoot     string
	Branch       string
	FileStatuses map[string]FileStatus // clean abs path -> status
	DirtyDirs    map[string]bool       // clean abs dir path -> has changed descendants
}

// FileBase stores the HEAD baseline of a file for fast live diffing in memory.
type FileBase struct {
	Loaded  bool
	InRepo  bool
	Tracked bool
	Lines   []string
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

// GetRepoStatus inspects the git repository containing dir (if any).
func GetRepoStatus(dir string) RepoStatus {
	if dir == "" {
		return RepoStatus{}
	}
	topOut, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return RepoStatus{}
	}
	repoRoot := filepath.Clean(strings.TrimSpace(topOut))
	if repoRoot == "" {
		return RepoStatus{}
	}

	branchOut, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	branch := ""
	if err == nil {
		branch = strings.TrimPrefix(strings.TrimSpace(branchOut), "heads/")
	}

	statuses := make(map[string]FileStatus)
	dirtyDirs := make(map[string]bool)

	porcelain, err := runGit(repoRoot, "status", "--porcelain=v1", "-uall", "--no-renames")
	if err == nil {
		for _, rawLine := range strings.Split(porcelain, "\n") {
			if len(rawLine) < 4 {
				continue
			}
			x := rawLine[0]
			y := rawLine[1]
			relPath := strings.TrimSpace(rawLine[3:])
			relPath = unquoteGitPath(relPath)
			if relPath == "" {
				continue
			}

			st := classifyPorcelain(x, y)
			if st == StatusNone {
				continue
			}

			absPath := filepath.Clean(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
			statuses[absPath] = st

			// Mark ancestor directories up to repoRoot as having git changes.
			curr := filepath.Dir(absPath)
			for {
				if !strings.HasPrefix(curr, repoRoot) {
					break
				}
				dirtyDirs[curr] = true
				if curr == repoRoot {
					break
				}
				parent := filepath.Dir(curr)
				if parent == curr {
					break
				}
				curr = parent
			}
		}
	}

	return RepoStatus{
		InRepo:       true,
		RepoRoot:     repoRoot,
		Branch:       branch,
		FileStatuses: statuses,
		DirtyDirs:    dirtyDirs,
	}
}

// LoadFileBase loads the HEAD version of filePath so line-level diffs can be
// computed live in memory without spawning git on every keystroke.
func LoadFileBase(filePath string) FileBase {
	if filePath == "" {
		return FileBase{Loaded: true}
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return FileBase{Loaded: true}
	}
	absPath = filepath.Clean(absPath)
	dir := filepath.Dir(absPath)
	if _, err := os.Stat(dir); err != nil {
		return FileBase{Loaded: true}
	}

	topOut, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return FileBase{Loaded: true, InRepo: false}
	}
	repoRoot := filepath.Clean(strings.TrimSpace(topOut))
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return FileBase{Loaded: true, InRepo: false}
	}

	gitRel := filepath.ToSlash(rel)
	headOut, err := runGit(repoRoot, "show", "HEAD:"+gitRel)
	if err != nil {
		// File is in repo but not yet committed to HEAD (untracked or newly staged).
		return FileBase{
			Loaded:  true,
			InRepo:  true,
			Tracked: false,
		}
	}

	headOut = strings.ReplaceAll(headOut, "\r\n", "\n")
	headLines := strings.Split(headOut, "\n")
	return FileBase{
		Loaded:  true,
		InRepo:  true,
		Tracked: true,
		Lines:   headLines,
	}
}

// ComputeLineChanges compares base (HEAD) against current editor buffer lines
// and returns a map of 0-based line index -> LineChange.
func ComputeLineChanges(base FileBase, current []string) map[int]LineChange {
	if !base.InRepo || len(current) == 0 {
		return nil
	}

	if !base.Tracked {
		// Untracked or newly added file: mark all lines as added (skip pure empty single-line buffer).
		if len(current) == 1 && current[0] == "" {
			return nil
		}
		changes := make(map[int]LineChange, len(current))
		for i := range current {
			changes[i] = LineAdded
		}
		return changes
	}

	oldLines := base.Lines
	// Normalize trailing empty line difference caused by POSIX trailing newline
	// vs editor Lines split when comparing identical content.
	if len(oldLines) > 1 && oldLines[len(oldLines)-1] == "" && (len(current) == 0 || current[len(current)-1] != "") {
		// Keep exact split semantics: os.ReadFile + strings.Split("\n") also keeps trailing ""
		// so only trim if oldLines has an extra trailing empty element beyond current.
	}

	// Fast trim of identical prefix and suffix.
	prefix := 0
	for prefix < len(oldLines) && prefix < len(current) && oldLines[prefix] == current[prefix] {
		prefix++
	}
	if prefix == len(oldLines) && prefix == len(current) {
		return nil
	}

	oldEnd := len(oldLines)
	newEnd := len(current)
	for oldEnd > prefix && newEnd > prefix && oldLines[oldEnd-1] == current[newEnd-1] {
		oldEnd--
		newEnd--
	}

	oldMid := oldLines[prefix:oldEnd]
	newMid := current[prefix:newEnd]

	changes := make(map[int]LineChange)

	if len(oldMid) == 0 {
		for i := prefix; i < newEnd; i++ {
			changes[i] = LineAdded
		}
		return changes
	}
	if len(newMid) == 0 {
		markRow := prefix - 1
		if markRow < 0 {
			markRow = 0
		}
		if markRow < len(current) {
			changes[markRow] = LineDeletedBelow
		}
		return changes
	}

	// Myers diff on the differing middle window.
	ops := myersDiff(oldMid, newMid)
	i := 0
	for i < len(ops) {
		if ops[i].kind == diffEqual {
			i++
			continue
		}
		delCount := 0
		addStart := -1
		addCount := 0
		delAnchorNew := ops[i].newIdx

		for i < len(ops) && ops[i].kind != diffEqual {
			if ops[i].kind == diffDelete {
				delCount++
			} else {
				if addStart == -1 {
					addStart = ops[i].newIdx
				}
				addCount++
			}
			i++
		}

		if delCount == 0 && addCount > 0 {
			for k := 0; k < addCount; k++ {
				changes[prefix+addStart+k] = LineAdded
			}
		} else if delCount > 0 && addCount == 0 {
			markRow := prefix + delAnchorNew - 1
			if markRow < 0 {
				markRow = 0
			}
			if markRow < len(current) {
				if _, exists := changes[markRow]; !exists {
					changes[markRow] = LineDeletedBelow
				}
			}
		} else {
			modCount := delCount
			if addCount < modCount {
				modCount = addCount
			}
			for k := 0; k < modCount; k++ {
				changes[prefix+addStart+k] = LineModified
			}
			for k := modCount; k < addCount; k++ {
				changes[prefix+addStart+k] = LineAdded
			}
		}
	}

	if len(changes) == 0 {
		return nil
	}
	return changes
}

type diffKind int

const (
	diffEqual diffKind = iota
	diffInsert
	diffDelete
)

type diffOp struct {
	kind   diffKind
	newIdx int
}

// myersDiff computes a shortest edit script between a and b.
// Falls back to a simple hunk replacement when the edit distance exceeds maxD.
func myersDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	maxD := n + m
	if maxD > 2000 {
		// Guard against pathological diffs on huge generated files.
		ops := make([]diffOp, 0, n+m)
		for i := 0; i < n; i++ {
			ops = append(ops, diffOp{kind: diffDelete, newIdx: 0})
		}
		for j := 0; j < m; j++ {
			ops = append(ops, diffOp{kind: diffInsert, newIdx: j})
		}
		return ops
	}

	offset := maxD
	v := make([]int, 2*maxD+1)
	trace := make([][]int, 0, 32)

	foundD := -1
	for d := 0; d <= maxD; d++ {
		snap := make([]int, len(v))
		copy(snap, v)
		trace = append(trace, snap)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+offset] < v[k+1+offset]) {
				x = v[k+1+offset]
			} else {
				x = v[k-1+offset] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[k+offset] = x
			if x >= n && y >= m {
				foundD = d
				break
			}
		}
		if foundD >= 0 {
			break
		}
	}

	if foundD < 0 {
		return nil
	}

	// Backtrack to construct operations in forward order.
	x, y := n, m
	rev := make([]diffOp, 0, n+m)
	for d := foundD; d > 0; d-- {
		prev := trace[d]
		k := x - y
		var prevK int
		if k == -d || (k != d && prev[k-1+offset] < prev[k+1+offset]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := prev[prevK+offset]
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			x--
			y--
			rev = append(rev, diffOp{kind: diffEqual, newIdx: y})
		}

		if x == prevX {
			y--
			rev = append(rev, diffOp{kind: diffInsert, newIdx: y})
		} else {
			x--
			rev = append(rev, diffOp{kind: diffDelete, newIdx: y})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		rev = append(rev, diffOp{kind: diffEqual, newIdx: y})
	}

	// Reverse in place.
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

func classifyPorcelain(x, y byte) FileStatus {
	if x == '?' && y == '?' {
		return StatusUntracked
	}
	if x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D') {
		return StatusConflict
	}
	if y == 'M' || x == 'M' {
		return StatusModified
	}
	if x == 'A' || y == 'A' {
		return StatusAdded
	}
	if x == 'R' || y == 'R' {
		return StatusRenamed
	}
	if y == 'D' || x == 'D' {
		return StatusDeleted
	}
	return StatusNone
}

func unquoteGitPath(p string) string {
	if len(p) >= 2 && p[0] == '"' && p[len(p)-1] == '"' {
		return p[1 : len(p)-1]
	}
	return p
}
