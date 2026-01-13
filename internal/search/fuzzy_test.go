package search

import (
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
