package shim

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteShim_Content(t *testing.T) {
	dir := t.TempDir()
	driftrBin := "/usr/local/bin/driftr"

	if err := writeShim(dir, "node", driftrBin); err != nil {
		t.Fatalf("writeShim() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "node"))
	if err != nil {
		t.Fatalf("failed to read shim: %v", err)
	}

	shim := string(content)
	if !strings.HasPrefix(shim, "#!/bin/sh\n") {
		t.Error("shim should start with #!/bin/sh")
	}
	if !strings.Contains(shim, `exec "/usr/local/bin/driftr" shim node "$@"`) {
		t.Errorf("shim content unexpected: %s", shim)
	}
}

func TestWriteShim_Executable(t *testing.T) {
	dir := t.TempDir()

	if err := writeShim(dir, "pnpm", "/usr/local/bin/driftr"); err != nil {
		t.Fatalf("writeShim() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "pnpm"))
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("shim should be executable")
	}
}

func TestWriteShim_AllTools(t *testing.T) {
	dir := t.TempDir()

	for _, tool := range ShimTools() {
		if err := writeShim(dir, tool, "/bin/driftr"); err != nil {
			t.Fatalf("writeShim(%q) error: %v", tool, err)
		}
		if _, err := os.Stat(filepath.Join(dir, tool)); errors.Is(err, os.ErrNotExist) {
			t.Errorf("shim for %q was not created", tool)
		}
	}

	// Verify all expected tools are in ShimTools().
	expected := map[string]bool{
		"node": true, "npm": true, "npx": true,
		"pnpm": true, "pnpx": true, "yarn": true,
	}
	for _, tool := range ShimTools() {
		if !expected[tool] {
			t.Errorf("unexpected tool in ShimTools(): %q", tool)
		}
		delete(expected, tool)
	}
	for tool := range expected {
		t.Errorf("missing tool in ShimTools(): %q", tool)
	}
}
