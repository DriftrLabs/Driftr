package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kisztof/driftr/internal/platform"
)

const nodeDistBaseURL = "https://nodejs.org/dist"

// DownloadURL returns the download URL for a given Node.js version.
func DownloadURL(version string) string {
	filename := ArchiveFilename(version)
	return fmt.Sprintf("%s/v%s/%s", nodeDistBaseURL, version, filename)
}

// ArchiveFilename returns the expected archive filename.
func ArchiveFilename(version string) string {
	return fmt.Sprintf("node-v%s-%s-%s.%s",
		version, platform.OS(), platform.Arch(), platform.ArchiveExt())
}

// Download fetches the Node.js archive to the cache directory.
// Returns the path to the downloaded file.
func Download(version string, verbose bool) (string, error) {
	cacheDir, err := platform.CacheDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	filename := ArchiveFilename(version)
	destPath := filepath.Join(cacheDir, filename)

	// Skip download if already cached.
	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		if verbose {
			fmt.Printf("  Using cached archive: %s\n", destPath)
		}
		return destPath, nil
	}

	url := DownloadURL(version)
	if verbose {
		fmt.Printf("  Downloading: %s\n", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("Node.js version %s not found at %s", version, url)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(cacheDir, "driftr-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download interrupted: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to save archive: %w", err)
	}

	return destPath, nil
}
