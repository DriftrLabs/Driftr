package installer

import (
	"errors"
	"fmt"
	"os"

	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

// InstallYarn downloads and installs a yarn version from the npm registry.
func InstallYarn(versionStr string, verbose bool) (string, error) {
	if err := platform.EnsureToolDirs("yarn"); err != nil {
		return "", err
	}

	v, err := version.Parse(versionStr)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	// Resolve partial versions via npm registry.
	resolvedVersion := v.String()
	if v.Latest || v.IsPartial() {
		resolved, err := ResolveRegistryLatest("yarn", v)
		if err != nil {
			return "", err
		}
		resolvedVersion = resolved
		if verbose {
			fmt.Printf("  Resolved %s to %s\n", versionStr, resolvedVersion)
		}
	}

	// Check if already installed.
	binPath, err := platform.ToolBinary("yarn", resolvedVersion)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binPath); err == nil {
		if verbose {
			fmt.Printf("  yarn %s is already installed\n", resolvedVersion)
		}
		return resolvedVersion, nil
	}

	// Download from npm registry.
	archivePath, rv, err := DownloadRegistryPackage("yarn", resolvedVersion, verbose)
	if err != nil {
		return "", err
	}

	// Verify integrity.
	if rv.Dist.Integrity != "" {
		if verbose {
			fmt.Println("  Verifying integrity...")
		}
		if err := VerifyIntegrity(archivePath, rv.Dist.Integrity); err != nil {
			os.Remove(archivePath)
			return "", err
		}
	}

	// Extract to version directory.
	versionDir, err := platform.ToolVersionDir("yarn", resolvedVersion)
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("  Extracting to: %s\n", versionDir)
	}
	if err := ExtractRegistryPackage(archivePath, versionDir); err != nil {
		os.RemoveAll(versionDir)
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	// Verify the binary exists after extraction.
	if _, err := os.Stat(binPath); errors.Is(err, os.ErrNotExist) {
		os.RemoveAll(versionDir)
		return "", fmt.Errorf("yarn binary not found after extraction at %s", binPath)
	}

	// Ensure the binary is executable.
	if err := os.Chmod(binPath, 0o755); err != nil {
		return "", err
	}

	return resolvedVersion, nil
}
