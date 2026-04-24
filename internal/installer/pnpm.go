package installer

import (
	"fmt"
	"os"

	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

// InstallPnpm downloads and installs a pnpm version from the npm registry.
// Uses the npm registry tarball with SHA-512 SRI verification.
func InstallPnpm(versionStr string, verbose bool) (string, error) {
	if err := platform.EnsureToolDirs("pnpm"); err != nil {
		return "", err
	}

	v, err := version.Parse(versionStr)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	// Resolve partial versions via npm registry.
	resolvedVersion := v.String()
	if v.Latest || v.IsPartial() {
		resolved, err := ResolveRegistryLatest("pnpm", v)
		if err != nil {
			return "", err
		}
		resolvedVersion = resolved
		if verbose {
			fmt.Printf("  Resolved %s to %s\n", versionStr, resolvedVersion)
		}
	}

	// Check if already installed.
	binPath, err := platform.ToolBinary("pnpm", resolvedVersion)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binPath); err == nil {
		if verbose {
			fmt.Printf("  pnpm %s is already installed\n", resolvedVersion)
		}
		return resolvedVersion, nil
	}

	// Download from npm registry.
	archivePath, rv, err := DownloadRegistryPackage("pnpm", resolvedVersion, verbose)
	if err != nil {
		return "", err
	}

	// Verify integrity.
	if rv.Dist.Integrity == "" {
		os.Remove(archivePath)
		return "", fmt.Errorf("registry metadata for pnpm %s is missing integrity data; refusing to install unverified package", resolvedVersion)
	}
	if verbose {
		fmt.Println("  Verifying integrity...")
	}
	if err := VerifyIntegrity(archivePath, rv.Dist.Integrity); err != nil {
		os.Remove(archivePath)
		return "", err
	}

	// Extract to version directory.
	versionDir, err := platform.ToolVersionDir("pnpm", resolvedVersion)
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("  Extracting to: %s\n", versionDir)
	}
	if err := ExtractRegistryPackage(archivePath, versionDir, binPath); err != nil {
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	// Ensure the binary is executable.
	if err := os.Chmod(binPath, 0o755); err != nil {
		return "", err
	}

	return resolvedVersion, nil
}
