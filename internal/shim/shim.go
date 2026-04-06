package shim

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DriftrLabs/driftr/internal/platform"
)

// ShimTools lists the tools for which shims are created.
var ShimTools = []string{"node", "npm", "npx", "pnpm", "pnpx", "yarn"}

// GenerateShims creates shim shell scripts in ~/.driftr/bin/.
// Each shim invokes `driftr shim <tool>` to resolve and exec the real binary.
func GenerateShims() error {
	binDir, err := platform.BinDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("failed to create bin dir: %w", err)
	}

	driftrBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine driftr executable path: %w", err)
	}
	// Resolve symlinks to get the real path.
	driftrBin, err = filepath.EvalSymlinks(driftrBin)
	if err != nil {
		return fmt.Errorf("cannot resolve driftr executable path: %w", err)
	}

	for _, tool := range ShimTools {
		if err := writeShim(binDir, tool, driftrBin); err != nil {
			return fmt.Errorf("failed to create shim for %s: %w", tool, err)
		}
	}

	return nil
}

func writeShim(binDir, tool, driftrBin string) error {
	shimPath := filepath.Join(binDir, tool)

	content := fmt.Sprintf(`#!/bin/sh
exec "%s" shim %s "$@"
`, driftrBin, tool)

	return os.WriteFile(shimPath, []byte(content), 0o755)
}

// ShimDir returns the path to the shim directory for display purposes.
func ShimDir() (string, error) {
	return platform.BinDir()
}
