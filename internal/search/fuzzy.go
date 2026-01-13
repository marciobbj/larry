package search

import (
	"os"
	"strings"
	"sync"
)

type FinderMode int

const (
	ModeFiles FinderMode = iota
	ModeGrep
)

type FileResult struct {
	Path string
}

type GrepResult struct {
	Path    string
	Line    int
	Content string
}

type FinderResult struct {
	File *FileResult
	Grep *GrepResult
	Mode FinderMode
}

type FuzzyMatcher struct{}

func NewFuzzyMatcher() *FuzzyMatcher {
	return &FuzzyMatcher{}
}

func (fm *FuzzyMatcher) Match(pattern, text string) (bool, int) {
	if pattern == "" {
		return true, 0
	}

	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	patternIdx := 0
	score := 0
	matched := 0

	for i := 0; i < len(text) && patternIdx < len(pattern); i++ {
		if text[i] == pattern[patternIdx] {
			patternIdx++
			score += i * 10
			matched++
		}
	}

	return patternIdx == len(pattern), score
}

type DirectoryScanner struct {
	ignoreDirs map[string]bool
}

func NewDirectoryScanner() *DirectoryScanner {
	return &DirectoryScanner{
		ignoreDirs: map[string]bool{
			".git":         true,
			"node_modules": true,
			"vendor":       true,
			".vscode":      true,
			".idea":        true,
			"dist":         true,
			"build":        true,
		},
	}
}

func (ds *DirectoryScanner) Scan(root string) ([]string, error) {
	var files []string
	var mu sync.Mutex

	var walkErr error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		walkErr = ds.walk(root, &files, &mu)
	}()

	wg.Wait()
	return files, walkErr
}

func IsBinary(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

func (ds *DirectoryScanner) walk(root string, files *[]string, mu *sync.Mutex) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := strings.Join([]string{root, entry.Name()}, string(os.PathSeparator))

		if entry.IsDir() {
			if ds.ignoreDirs[entry.Name()] {
				continue
			}
			if err := ds.walk(fullPath, files, mu); err != nil {
				return err
			}
		} else {
			if IsBinary(fullPath) {
				continue
			}
			mu.Lock()
			*files = append(*files, fullPath)
			mu.Unlock()
		}
	}

	return nil
}

type LiveGrep struct {
	scanner *DirectoryScanner
	bm      *BoyerMooreSearch
}

func NewLiveGrep() *LiveGrep {
	return &LiveGrep{
		scanner: NewDirectoryScanner(),
		bm:      &BoyerMooreSearch{},
	}
}

func (lg *LiveGrep) Search(root, pattern string) ([]GrepResult, error) {
	if pattern == "" {
		return nil, nil
	}

	files, err := lg.scanner.Scan(root)
	if err != nil {
		return nil, err
	}

	bm := NewBoyerMooreSearch(pattern)

	var results []GrepResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	resultsChan := make(chan GrepResult, 100)

	done := make(chan struct{})
	go func() {
		for res := range resultsChan {
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}
		close(done)
	}()

	semaphore := make(chan struct{}, 10)

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			content, err := os.ReadFile(filePath)
			if err != nil {
				return
			}

			lines := strings.Split(string(content), "\n")
			for lineIdx, line := range lines {
				if bm.SearchInText(line) != nil {
					resultsChan <- GrepResult{
						Path:    filePath,
						Line:    lineIdx + 1,
						Content: strings.TrimSpace(line),
					}
				}
			}
		}(file)
	}

	wg.Wait()
	close(resultsChan)
	<-done

	return results, nil
}
