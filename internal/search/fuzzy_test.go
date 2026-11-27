package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFuzzyMatcher(t *testing.T) {
	fm := NewFuzzyMatcher()

	tests := []struct {
		pattern string
		text    string
		want    bool
	}{
		{"test", "test_file.txt", true},
		{"tst", "test_file.txt", true},
		{"abc", "test_file.txt", false},
		{"", "test_file.txt", true},
		{"config", "config.json", true},
		{"cfg", "config.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got, _ := fm.Match(tt.pattern, tt.text)
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.text, got, tt.want)
			}
		})
	}
}

func TestDirectoryScanner(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "larry-test-scan-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFiles := []string{"test.txt", "other.txt", "nested/file.txt"}
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	ds := NewDirectoryScanner()
	files, err := ds.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("expected %d files, got %d", len(testFiles), len(files))
	}
}

func TestLiveGrep(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "larry-test-grep-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `hello world
this is a test
another line`
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	lg := NewLiveGrep()
	results, err := lg.Search(tmpDir, "test")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	} else if results[0].Line != 2 {
		t.Errorf("expected line 2, got %d", results[0].Line)
	}
}

func TestLiveGrepSearchContextAndPureGoFallback(t *testing.T) {
	tmpDir := t.TempDir()

	// Hidden dir must be skipped
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.txt"), []byte("needle hidden\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Binary file must be skipped
	binFile := filepath.Join(tmpDir, "data.bin")
	if err := os.WriteFile(binFile, []byte("needle\x00binary\x00data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Normal file with multiple matches including uppercase (smart-case check)
	srcFile := filepath.Join(tmpDir, "app.go")
	src := "line one\nNeedle two\nline three\nneedle four\nneedle five\n"
	if err := os.WriteFile(srcFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	lg := NewLiveGrep()

	// Test SearchContext (uses ripgrep when installed) with maxResults cap = 2
	res, err := lg.SearchContext(context.Background(), tmpDir, "needle", 2)
	if err != nil {
		t.Fatalf("SearchContext error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 capped results from SearchContext, got %d: %+v", len(res), res)
	}

	// Test pure-Go SearchFiles fallback directly (including binary filtering & smart-case)
	goRes, err := lg.SearchFiles(context.Background(), []string{binFile, srcFile}, "needle", 2)
	if err != nil {
		t.Fatalf("SearchFiles error: %v", err)
	}
	if len(goRes) != 2 {
		t.Fatalf("expected 2 capped results from SearchFiles, got %d: %+v", len(goRes), goRes)
	}
	for _, r := range goRes {
		if r.Path != srcFile {
			t.Fatalf("expected only srcFile match, got %q", r.Path)
		}
	}
}
