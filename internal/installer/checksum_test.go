package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFileChecksum(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello driftr\n")
	path := filepath.Join(dir, "testfile")

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := ComputeFileChecksum(path)
	if err != nil {
		t.Fatalf("ComputeFileChecksum() error: %v", err)
	}

	h := sha256.Sum256(content)
	want := hex.EncodeToString(h[:])

	if got != want {
		t.Errorf("ComputeFileChecksum() = %q, want %q", got, want)
	}
}

func TestComputeFileChecksum_MissingFile(t *testing.T) {
	_, err := ComputeFileChecksum("/nonexistent/path/file.tar.gz")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestComputeFileChecksum_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty")

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	got, err := ComputeFileChecksum(path)
	if err != nil {
		t.Fatalf("ComputeFileChecksum() error: %v", err)
	}

	h := sha256.Sum256([]byte{})
	want := hex.EncodeToString(h[:])

	if got != want {
		t.Errorf("ComputeFileChecksum() = %q, want %q", got, want)
	}
}
