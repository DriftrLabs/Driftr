package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/DriftrLabs/driftr/internal/ioutil"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

const pnpmGitHubRepo = "pnpm/pnpm"

// InstallPnpm downloads and installs a pnpm version.
// Uses the standalone binary from GitHub releases.
func InstallPnpm(versionStr string, verbose bool) (string, error) {
	if err := platform.EnsureToolDirs("pnpm"); err != nil {
		return "", err
	}

	v, err := version.Parse(versionStr)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	// Resolve partial versions via npm registry (pnpm publishes there too).
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

	// Download standalone binary from GitHub releases.
	archivePath, err := downloadPnpmBinary(resolvedVersion, verbose)
	if err != nil {
		return "", err
	}

	// Install: create version dir and copy binary.
	versionDir, err := platform.ToolVersionDir("pnpm", resolvedVersion)
	if err != nil {
		return "", err
	}

	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create pnpm bin dir: %w", err)
	}

	destPath := filepath.Join(binDir, "pnpm")
	if err := ioutil.CopyFile(archivePath, destPath); err != nil {
		os.RemoveAll(versionDir)
		return "", fmt.Errorf("failed to install pnpm binary: %w", err)
	}
	if err := os.Chmod(destPath, 0o755); err != nil {
		os.RemoveAll(versionDir)
		return "", err
	}

	// Create pnpx symlink pointing to pnpm.
	pnpxPath := filepath.Join(binDir, "pnpx")
	os.Remove(pnpxPath) // remove if exists
	if err := os.Symlink("pnpm", pnpxPath); err != nil {
		// Fallback: copy the binary if symlink fails.
		if err := ioutil.CopyFile(destPath, pnpxPath); err != nil {
			return "", fmt.Errorf("failed to create pnpx: %w", err)
		}
	}

	return resolvedVersion, nil
}

// pnpmBinaryURL returns the GitHub releases URL for the standalone pnpm binary.
func pnpmBinaryURL(ver string) string {
	osName := runtime.GOOS
	archName := runtime.GOARCH
	// pnpm uses "x64" for amd64.
	if archName == "amd64" {
		archName = "x64"
	}
	return fmt.Sprintf("https://github.com/%s/releases/download/v%s/pnpm-%s-%s",
		pnpmGitHubRepo, ver, osName, archName)
}

func downloadPnpmBinary(ver string, verbose bool) (string, error) {
	cacheDir, err := platform.CacheDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	filename := fmt.Sprintf("pnpm-%s-%s-%s", ver, runtime.GOOS, runtime.GOARCH)
	destPath := filepath.Join(cacheDir, filename)

	// Skip if cached.
	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		if verbose {
			fmt.Printf("  Using cached binary: %s\n", destPath)
		}
		return destPath, nil
	}

	url := pnpmBinaryURL(ver)
	if verbose {
		fmt.Printf("  Downloading: %s\n", url)
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("pnpm %s standalone binary not found for %s/%s", ver, runtime.GOOS, runtime.GOARCH)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(cacheDir, "driftr-pnpm-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if ioutil.IsTerminal(os.Stderr) {
		pw := &ioutil.ProgressWriter{Dest: tmpFile, Total: resp.ContentLength}
		_, err = io.Copy(pw, resp.Body)
		pw.Finish()
	} else {
		_, err = io.Copy(tmpFile, resp.Body)
	}
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download interrupted: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to save binary: %w", err)
	}

	return destPath, nil
}
