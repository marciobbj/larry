package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ReplaceLineMatch represents a single line containing one or more matches of a query.
type ReplaceLineMatch struct {
	Line    int    // 1-based line number
	Content string // trimmed line content
	Count   int    // occurrences on this line
}

// FileReplaceGroup groups all matches within a single file for global replace preview.
type FileReplaceGroup struct {
	Path        string             // absolute or clean path on disk
	DisplayPath string             // path relative to the project root
	Matches     []ReplaceLineMatch // matching lines in order
	TotalCount  int                // total occurrences across the file
}

// GlobalReplaceResult summarizes a completed global replace run.
type GlobalReplaceResult struct {
	FilesModified     int
	TotalReplacements int
	ModifiedFilePaths []string
	Errors            []string
}

// GlobalReplacer scans and replaces text across an entire project directory.
type GlobalReplacer struct {
	scanner *DirectoryScanner
}

// NewGlobalReplacer creates a GlobalReplacer backed by the default DirectoryScanner.
func NewGlobalReplacer() *GlobalReplacer {
	return &GlobalReplacer{
		scanner: NewDirectoryScanner(),
	}
}

// Preview finds all non-binary, non-symlink text files under root containing query
// and returns grouped line matches sorted by relative path.
func (gr *GlobalReplacer) Preview(root, query string) ([]FileReplaceGroup, error) {
	if query == "" {
		return nil, nil
	}

	absRoot, err := resolveRoot(root)
	if err != nil {
		return nil, err
	}

	files, err := gr.scanner.Scan(absRoot)
	if err != nil {
		return nil, err
	}

	var (
		groups []FileReplaceGroup
		mu     sync.Mutex
		wg     sync.WaitGroup
		sem    = make(chan struct{}, 10)
	)

	for _, filePath := range files {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Skip symlinks so we never leave the project scope.
			info, err := os.Lstat(p)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
				return
			}

			raw, err := os.ReadFile(p)
			if err != nil {
				return
			}

			text := string(raw)
			if !strings.Contains(text, query) {
				return
			}

			lines := strings.Split(text, "\n")
			var matches []ReplaceLineMatch
			total := 0
			for i, line := range lines {
				cleanLine := strings.TrimSuffix(line, "\r")
				c := strings.Count(cleanLine, query)
				if c > 0 {
					total += c
					matches = append(matches, ReplaceLineMatch{
						Line:    i + 1,
						Content: strings.TrimSpace(cleanLine),
						Count:   c,
					})
				}
			}

			if total == 0 {
				return
			}

			display := p
			if rel, relErr := filepath.Rel(absRoot, p); relErr == nil {
				display = rel
			}

			mu.Lock()
			groups = append(groups, FileReplaceGroup{
				Path:        filepath.Clean(p),
				DisplayPath: display,
				Matches:     matches,
				TotalCount:  total,
			})
			mu.Unlock()
		}(filePath)
	}

	wg.Wait()

	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].DisplayPath) < strings.ToLower(groups[j].DisplayPath)
	})

	return groups, nil
}

// Apply replaces all occurrences of query with replacement across files under root.
// When allowedFiles is non-nil, only files present in allowedFiles with value true are modified.
// Files outside root or symlinks are never modified.
func (gr *GlobalReplacer) Apply(root, query, replacement string, allowedFiles map[string]bool) (GlobalReplaceResult, error) {
	var result GlobalReplaceResult
	if query == "" || query == replacement {
		return result, nil
	}

	absRoot, err := resolveRoot(root)
	if err != nil {
		return result, err
	}

	var targetFiles []string
	if allowedFiles != nil {
		for p, include := range allowedFiles {
			if !include {
				continue
			}
			absPath, absErr := filepath.Abs(p)
			if absErr != nil {
				continue
			}
			absPath = filepath.Clean(absPath)
			if !isWithinRoot(absRoot, absPath) {
				continue
			}
			targetFiles = append(targetFiles, absPath)
		}
		sort.Strings(targetFiles)
	} else {
		scanned, scanErr := gr.scanner.Scan(absRoot)
		if scanErr != nil {
			return result, scanErr
		}
		sort.Strings(scanned)
		targetFiles = scanned
	}

	for _, filePath := range targetFiles {
		info, err := os.Lstat(filePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
			continue
		}
		if IsBinary(filePath) {
			continue
		}

		raw, err := os.ReadFile(filePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}

		text := string(raw)
		count := strings.Count(text, query)
		if count == 0 {
			continue
		}

		updated := strings.ReplaceAll(text, query, replacement)
		perm := info.Mode().Perm()
		if perm == 0 {
			perm = 0644
		}
		if err := os.WriteFile(filePath, []byte(updated), perm); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}

		result.FilesModified++
		result.TotalReplacements += count
		result.ModifiedFilePaths = append(result.ModifiedFilePaths, filepath.Clean(filePath))
	}

	sort.Strings(result.ModifiedFilePaths)
	return result, nil
}

func resolveRoot(root string) (string, error) {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func isWithinRoot(absRoot, absPath string) bool {
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
