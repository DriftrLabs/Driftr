package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// writeExe creates an executable file named tool in dir.
func writeExe(t *testing.T, dir, tool string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, tool), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestCheckShimShadowing_ShimWins(t *testing.T) {
	shimDir := t.TempDir()
	writeExe(t, shimDir, "node")
	t.Setenv("PATH", shimDir)

	if got := checkShimShadowing(shimDir); got != 0 {
		t.Errorf("shim is first on PATH, want 0 shadowed, got %d", got)
	}
}

func TestCheckShimShadowing_OtherInstallShadows(t *testing.T) {
	shimDir := t.TempDir()
	otherDir := t.TempDir()
	writeExe(t, shimDir, "node")
	writeExe(t, otherDir, "node") // earlier in PATH → wins
	t.Setenv("PATH", otherDir+string(os.PathListSeparator)+shimDir)

	if got := checkShimShadowing(shimDir); got != 1 {
		t.Errorf("node shadowed by earlier install, want 1, got %d", got)
	}
}

func TestCheckShimShadowing_SkipsWhenBinDirNotOnPath(t *testing.T) {
	shimDir := t.TempDir()
	otherDir := t.TempDir()
	writeExe(t, otherDir, "node")
	// shimDir absent from PATH — checkPath owns that diagnosis, so shadowing
	// check must stay silent (return 0).
	t.Setenv("PATH", otherDir)

	if got := checkShimShadowing(shimDir); got != 0 {
		t.Errorf("bin dir not on PATH, want 0 (skip), got %d", got)
	}
}

func TestCheckShimShadowing_NoToolsOnPath(t *testing.T) {
	shimDir := t.TempDir() // on PATH but contains no tool binaries
	t.Setenv("PATH", shimDir)

	if got := checkShimShadowing(shimDir); got != 0 {
		t.Errorf("no managed tools on PATH, want 0, got %d", got)
	}
}
