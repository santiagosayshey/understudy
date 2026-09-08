package plex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearCache(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "Cache", "PhotoTranscoder")
	os.MkdirAll(filepath.Join(cache, "1b"), 0o755)
	os.WriteFile(filepath.Join(cache, "1b", "x.jpg"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(cache, "loose.jpg"), []byte("y"), 0o644)
	n, err := ClearCache(cache)
	if err != nil || n != 2 {
		t.Fatalf("clear: %d %v", n, err)
	}
	if left, _ := os.ReadDir(cache); len(left) != 0 {
		t.Fatalf("not empty: %v", left)
	}
	if _, err := ClearCache(root); err == nil {
		t.Fatal("must refuse a directory that is not PhotoTranscoder")
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatal("the directory itself must survive")
	}
}
