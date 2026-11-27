package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalReplacerPreviewAndApply(t *testing.T) {
	root := t.TempDir()

	// Create normal files, nested files, ignored dirs, and a binary file
	files := map[string]string{
		"a.go":                    "package app\n\nconst oldToken = \"oldToken\"\n",
		"sub/b.txt":               "hello oldToken\r\nsecond line\r\n",
		"untouched.txt":           "nothing to change here\n",
		"node_modules/pkg/dep.js": "const oldToken = 1;\n",
		".git/config":             "oldToken\n",
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	// Binary file containing NUL byte + query
	binPath := filepath.Join(root, "bin.dat")
	if err := os.WriteFile(binPath, []byte{0x00, 'o', 'l', 'd', 'T', 'o', 'k', 'e', 'n'}, 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	replacer := NewGlobalReplacer()

	groups, err := replacer.Preview(root, "oldToken")
	if err != nil {
		t.Fatalf("Preview() error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("Preview() found %d files (%+v), want 2", len(groups), groups)
	}
	if groups[0].DisplayPath != "a.go" || groups[0].TotalCount != 2 {
		t.Fatalf("groups[0] = %+v, want a.go with 2 matches", groups[0])
	}
	if groups[1].DisplayPath != filepath.Join("sub", "b.txt") || groups[1].TotalCount != 1 {
		t.Fatalf("groups[1] = %+v, want sub/b.txt with 1 match", groups[1])
	}

	// Apply selectively only to a.go
	allowed := map[string]bool{
		groups[0].Path: true,
		groups[1].Path: false,
	}
	res, err := replacer.Apply(root, "oldToken", "newToken", allowed)
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if res.FilesModified != 1 || res.TotalReplacements != 2 {
		t.Fatalf("Apply() result = %+v, want 1 file and 2 replacements", res)
	}

	aBytes, _ := os.ReadFile(filepath.Join(root, "a.go"))
	if string(aBytes) != "package app\n\nconst newToken = \"newToken\"\n" {
		t.Fatalf("a.go content = %q", string(aBytes))
	}

	// b.txt should remain untouched because it was unchecked
	bBytes, _ := os.ReadFile(filepath.Join(root, "sub/b.txt"))
	if string(bBytes) != "hello oldToken\r\nsecond line\r\n" {
		t.Fatalf("sub/b.txt was unexpectedly modified: %q", string(bBytes))
	}

	// Now apply to all remaining files (allowedFiles == nil) and verify CRLF preserved
	res2, err := replacer.Apply(root, "oldToken", "newToken", nil)
	if err != nil {
		t.Fatalf("Apply(all) error: %v", err)
	}
	if res2.FilesModified != 1 || res2.TotalReplacements != 1 {
		t.Fatalf("Apply(all) result = %+v, want 1 file and 1 replacement", res2)
	}

	bBytes, _ = os.ReadFile(filepath.Join(root, "sub/b.txt"))
	if string(bBytes) != "hello newToken\r\nsecond line\r\n" {
		t.Fatalf("sub/b.txt CRLF/content = %q", string(bBytes))
	}

	// Ignored files and binary file must stay untouched
	depBytes, _ := os.ReadFile(filepath.Join(root, "node_modules/pkg/dep.js"))
	if string(depBytes) != "const oldToken = 1;\n" {
		t.Fatalf("node_modules file was modified: %q", string(depBytes))
	}
	binBytes, _ := os.ReadFile(binPath)
	if len(binBytes) != 9 || binBytes[0] != 0x00 {
		t.Fatalf("binary file was modified: %v", binBytes)
	}
}

func TestGlobalReplacerBlocksOutOfScopeFiles(t *testing.T) {
	projectRoot := t.TempDir()
	outsideDir := t.TempDir()

	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("replace_me"), 0644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	replacer := NewGlobalReplacer()
	res, err := replacer.Apply(projectRoot, "replace_me", "hacked", map[string]bool{
		outsideFile: true,
	})
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if res.FilesModified != 0 {
		t.Fatalf("expected 0 modified files outside root, got %d", res.FilesModified)
	}

	data, _ := os.ReadFile(outsideFile)
	if string(data) != "replace_me" {
		t.Fatalf("outside file was modified: %q", string(data))
	}
}
