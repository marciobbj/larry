package search

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

const maxScanPaths = 50000

var (
	rgPathOnce sync.Once
	rgPathVal  string
)

func findRipgrep() string {
	rgPathOnce.Do(func() {
		if p, err := exec.LookPath("rg"); err == nil {
			rgPathVal = p
		}
	})
	return rgPathVal
}

// Scan returns non-binary files under root (checking content when needed).
func (ds *DirectoryScanner) Scan(root string) ([]string, error) {
	var files []string
	err := ds.walk(root, &files, true, 0)
	return files, err
}

// ScanPaths quickly walks root collecting candidate file paths without opening
// every file on disk, skipping hidden/ignored directories and known binary extensions.
func (ds *DirectoryScanner) ScanPaths(root string) ([]string, error) {
	var files []string
	err := ds.walk(root, &files, false, maxScanPaths)
	return files, err
}

// ScanPathsFast uses ripgrep (--files) when installed for fast gitignore-aware
// listing, falling back to the pure-Go walker when ripgrep is unavailable.
func (ds *DirectoryScanner) ScanPathsFast(ctx context.Context, root string) ([]string, error) {
	rg := findRipgrep()
	if rg == "" {
		return ds.ScanPaths(root)
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, rg, "--files", "--color=never", root)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ds.ScanPaths(root)
	}
	if err := cmd.Start(); err != nil {
		return ds.ScanPaths(root)
	}

	files := make([]string, 0, 256)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			cancel()
			break
		}
		p := scanner.Text()
		if p == "" || HasBinaryExtension(p) {
			continue
		}
		files = append(files, p)
		if len(files) >= maxScanPaths {
			cancel()
			break
		}
	}
	_ = cmd.Wait()

	if ctx.Err() != nil {
		return files, ctx.Err()
	}
	return files, nil
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

func (ds *DirectoryScanner) walk(root string, files *[]string, checkContent bool, maxFiles int) error {
	if maxFiles > 0 && len(*files) >= maxFiles {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if maxFiles > 0 && len(*files) >= maxFiles {
			return nil
		}
		name := entry.Name()
		fullPath := strings.Join([]string{root, name}, string(os.PathSeparator))

		if entry.IsDir() {
			if ds.ignoreDirs[name] || (strings.HasPrefix(name, ".") && name != "." && name != "..") {
				continue
			}
			_ = ds.walk(fullPath, files, checkContent, maxFiles)
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
	return lg.SearchContext(context.Background(), root, pattern, 0)
}

// SearchContext searches for pattern under root, using ripgrep when available
// for streaming SIMD + gitignore-aware grep with immediate cancellation at
// maxResults, and falling back to a fast pure-Go worker pool otherwise.
func (lg *LiveGrep) SearchContext(ctx context.Context, root, pattern string, maxResults int) ([]GrepResult, error) {
	if pattern == "" {
		return nil, nil
	}
	if rg := findRipgrep(); rg != "" {
		results, ok := searchWithRipgrep(ctx, rg, root, pattern, maxResults)
		if ok {
			return results, nil
		}
	}
	files, err := lg.scanner.ScanPaths(root)
	if err != nil {
		return nil, err
	}
	return lg.SearchFiles(ctx, files, pattern, maxResults)
}

func searchWithRipgrep(ctx context.Context, rg, root, pattern string, maxResults int) ([]GrepResult, bool) {
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	args := []string{
		"--no-heading",
		"--line-number",
		"--with-filename",
		"--null",
		"--color=never",
		"--fixed-strings",
		"--smart-case",
		"--max-filesize", "2M",
		"--",
		pattern,
		root,
	}
	cmd := exec.CommandContext(cmdCtx, rg, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false
	}
	if err := cmd.Start(); err != nil {
		return nil, false
	}

	capHint := 32
	if maxResults > 0 && maxResults < 256 {
		capHint = maxResults
	}
	results := make([]GrepResult, 0, capHint)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			cancel()
			break
		}
		raw := scanner.Bytes()
		nulIdx := bytes.IndexByte(raw, 0)
		if nulIdx <= 0 || nulIdx+1 >= len(raw) {
			continue
		}
		path := string(raw[:nulIdx])
		rest := raw[nulIdx+1:]
		colonIdx := bytes.IndexByte(rest, ':')
		if colonIdx <= 0 {
			continue
		}
		lineNum, err := strconv.Atoi(string(rest[:colonIdx]))
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(rest[colonIdx+1:]))
		results = append(results, GrepResult{
			Path:    path,
			Line:    lineNum,
			Content: content,
		})
		if maxResults > 0 && len(results) >= maxResults {
			cancel()
			break
		}
	}
	_ = cmd.Wait()
	return results, true
}

// SearchFiles greps pattern across a pre-scanned list of file paths using a
// bounded worker pool, SIMD bytes.Index pre-filtering, zero-split line scanning,
// and early cancellation once maxResults (> 0) are found or ctx is cancelled.
func (lg *LiveGrep) SearchFiles(ctx context.Context, files []string, pattern string, maxResults int) ([]GrepResult, error) {
	if pattern == "" || len(files) == 0 {
		return nil, nil
	}

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	patternBytes := []byte(pattern)
	// Smart-case: lowercase patterns match case-insensitively, uppercase makes it exact.
	caseInsensitive := strings.ToLower(pattern) == pattern
	lowerPattern := []byte(strings.ToLower(pattern))

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
				content, ok := readSmallFile(filePath, maxGrepFileSize)
				if !ok || len(content) == 0 {
					continue
				}

				haystack := content
				needle := patternBytes
				if caseInsensitive {
					// Fast exact check first; only lowercase if exact bytes aren't found
					if !bytes.Contains(content, patternBytes) {
						haystack = bytes.ToLower(content)
						needle = lowerPattern
						if !bytes.Contains(haystack, needle) {
							continue
						}
					}
				} else if !bytes.Contains(content, patternBytes) {
					continue
				}

				// Only pay the binary heuristic cost on files that actually matched!
				if IsBinaryData(content) {
					continue
				}

				// Jump directly between matching lines using SIMD bytes.Index
				lineNum := 1
				scanPos := 0
				for scanPos < len(haystack) {
					if searchCtx.Err() != nil {
						return
					}
					rel := bytes.Index(haystack[scanPos:], needle)
					if rel < 0 {
						break
					}
					matchPos := scanPos + rel
					lineNum += bytes.Count(content[scanPos:matchPos], []byte{'\n'})

					lineStart := bytes.LastIndexByte(content[:matchPos], '\n') + 1
					lineEnd := bytes.IndexByte(content[matchPos:], '\n')
					if lineEnd < 0 {
						lineEnd = len(content)
					} else {
						lineEnd = matchPos + lineEnd
					}

					lineText := strings.TrimSpace(string(content[lineStart:lineEnd]))
					mu.Lock()
					if maxResults <= 0 || len(results) < maxResults {
						results = append(results, GrepResult{
							Path:    filePath,
							Line:    lineNum,
							Content: lineText,
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

					// Advance past the end of this line so we report at most one match per line
					if lineEnd < len(content) {
						scanPos = lineEnd + 1
						lineNum++
					} else {
						break
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

func readSmallFile(filePath string, maxSize int64) ([]byte, bool) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() || info.Size() == 0 || info.Size() > maxSize {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil || int64(len(data)) > maxSize {
		return nil, false
	}
	return data, true
}
