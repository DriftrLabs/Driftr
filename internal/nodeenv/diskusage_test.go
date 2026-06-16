package nodeenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirSize_SumsRegularFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), 100)
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), 250)
	writeFile(t, filepath.Join(dir, "sub", "deep", "c.txt"), 50)

	got, err := DirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 400 {
		t.Errorf("DirSize = %d, want 400", got)
	}
}

func TestDirSize_MissingPathIsZero(t *testing.T) {
	got, err := DirSize(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("DirSize of missing path = %d, want 0", got)
	}
}

func TestDirSize_EmptyDir(t *testing.T) {
	got, err := DirSize(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("DirSize of empty dir = %d, want 0", got)
	}
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}
