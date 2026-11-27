package search

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
			"target":       true,
			".cache":       true,
			".next":        true,
			".nuxt":        true,
			".venv":        true,
			"venv":         true,
			"__pycache__":  true,
			".gradle":      true,
			".cargo":       true,
		},
	}
}

var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".bmp": true, ".ico": true, ".tiff": true, ".psd": true, ".avif": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".zip": true, ".tar": true, ".gz": true, ".tgz": true, ".bz2": true, ".xz": true, ".7z": true, ".rar": true, ".zst": true,
	".mp3": true, ".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".webm": true, ".wav": true, ".flac": true, ".ogg": true, ".m4a": true,
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".a": true, ".o": true, ".obj": true,
	".pyc": true, ".pyo": true, ".class": true, ".jar": true, ".war": true, ".wasm": true,
	".db": true, ".sqlite": true, ".sqlite3": true, ".bin": true, ".iso": true, ".img": true, ".deb": true, ".rpm": true,
}

// HasBinaryExtension returns true if the filename extension is a known binary format.
func HasBinaryExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext != "" && binaryExtensions[ext]
}

// Scan returns non-binary files under root (checking content when needed).
func (ds *DirectoryScanner) Scan(root string) ([]string, error) {
	var files []string
	err := ds.walk(root, &files, true)
	return files, err
}

// ScanPaths quickly walks root collecting candidate file paths without opening
// every file on disk, skipping ignored directories and known binary extensions.
func (ds *DirectoryScanner) ScanPaths(root string) ([]string, error) {
	var files []string
	err := ds.walk(root, &files, false)
	return files, err
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
	if HasBinaryExtension(filePath) {
		return true
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	return IsBinaryData(buf[:n])
}

func (ds *DirectoryScanner) walk(root string, files *[]string, checkContent bool) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := strings.Join([]string{root, name}, string(os.PathSeparator))

		if entry.IsDir() {
			if ds.ignoreDirs[name] {
				continue
			}
			_ = ds.walk(fullPath, files, checkContent)
		} else {
			if HasBinaryExtension(fullPath) {
				continue
			}
			if checkContent && IsBinary(fullPath) {
				continue
			}
			*files = append(*files, fullPath)
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

const maxGrepFileSize = 2 * 1024 * 1024 // 2 MiB

func (lg *LiveGrep) Search(root, pattern string) ([]GrepResult, error) {
	if pattern == "" {
		return nil, nil
	}
	files, err := lg.scanner.ScanPaths(root)
	if err != nil {
		return nil, err
	}
	return lg.SearchFiles(context.Background(), files, pattern, 0)
}

// SearchFiles greps pattern across a pre-scanned list of file paths using a
// bounded worker pool, cancelling early once maxResults (> 0) are found or ctx
// is cancelled.
func (lg *LiveGrep) SearchFiles(ctx context.Context, files []string, pattern string, maxResults int) ([]GrepResult, error) {
	if pattern == "" || len(files) == 0 {
		return nil, nil
	}

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	bm := NewBoyerMooreSearch(pattern)
	patternBytes := []byte(pattern)

	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	if numWorkers < 2 {
		numWorkers = 2
	}

	jobs := make(chan string, numWorkers*4)
	var results []GrepResult
	var mu sync.Mutex
	var found int64

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				if searchCtx.Err() != nil {
					return
				}
				info, err := os.Stat(filePath)
				if err != nil || info.IsDir() || info.Size() == 0 || info.Size() > maxGrepFileSize {
					continue
				}
				content, err := os.ReadFile(filePath)
				if err != nil || IsBinaryData(content) {
					continue
				}
				// Fast pre-check before splitting lines
				if !bytes.Contains(content, patternBytes) {
					continue
				}

				lines := strings.Split(string(content), "\n")
				for lineIdx, line := range lines {
					if searchCtx.Err() != nil {
						return
					}
					if bm.SearchInText(line) != nil {
						mu.Lock()
						if maxResults <= 0 || len(results) < maxResults {
							results = append(results, GrepResult{
								Path:    filePath,
								Line:    lineIdx + 1,
								Content: strings.TrimSpace(line),
							})
							n := atomic.AddInt64(&found, 1)
							if maxResults > 0 && int(n) >= maxResults {
								cancel()
							}
						}
						mu.Unlock()
						if maxResults > 0 && int(atomic.LoadInt64(&found)) >= maxResults {
							return
						}
					}
				}
			}
		}()
	}

	for _, f := range files {
		if searchCtx.Err() != nil {
			break
		}
		select {
		case jobs <- f:
		case <-searchCtx.Done():
		}
	}
	close(jobs)
	wg.Wait()

	return results, nil
}
