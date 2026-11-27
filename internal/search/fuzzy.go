package search

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"unicode/utf8"
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

// IsBinaryData inspects a byte slice (up to the first 8 KiB) to determine
// whether the content is binary rather than plain text.
func IsBinaryData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Well-known binary file headers whose first bytes may be ASCII before
	// compressed/binary streams (e.g. PDFs often start streams past 8 KiB).
	if bytes.HasPrefix(data, []byte("%PDF-")) ||
		bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) ||
		bytes.HasPrefix(data, []byte("\xff\xd8\xff")) ||
		bytes.HasPrefix(data, []byte("GIF87a")) ||
		bytes.HasPrefix(data, []byte("GIF89a")) ||
		bytes.HasPrefix(data, []byte("PK\x03\x04")) ||
		bytes.HasPrefix(data, []byte("\x7fELF")) ||
		bytes.HasPrefix(data, []byte("\x1f\x8b")) {
		return true
	}

	limit := len(data)
	if limit > 8192 {
		limit = 8192
	}
	sample := data[:limit]

	// Any NUL byte in the first 8 KiB indicates binary data.
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}

	checkLen := len(sample)
	if len(data) > limit && checkLen > 3 {
		checkLen -= 3
	}

	suspicious := 0
	for i := 0; i < checkLen; {
		b := sample[i]
		if b < 0x20 {
			if b != '\t' && b != '\n' && b != '\r' && b != '\f' && b != 0x1b {
				suspicious++
			}
			i++
			continue
		}
		if b == 0x7f {
			suspicious++
			i++
			continue
		}
		if b < 0x80 {
			i++
			continue
		}
		r, size := utf8.DecodeRune(sample[i:checkLen])
		if r == utf8.RuneError && size == 1 {
			suspicious++
			i++
			continue
		}
		i += size
	}

	return checkLen > 0 && suspicious*100/checkLen > 30
}

func IsBinary(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	return IsBinaryData(buf[:n])
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

	// can only process 10 files at a time, this saves memory and prevents race conditions
	semaphore := make(chan struct{}, 10)

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // release execution
			semaphore <- struct{}{} // start execution
			
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
