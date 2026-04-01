package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DriftrLabs/driftr/internal/ioutil"
	"github.com/DriftrLabs/driftr/internal/platform"
)

const nodeDistBaseURL = "https://nodejs.org/dist"

// httpClient is the shared HTTP client for all installer network operations.
var httpClient = &http.Client{Timeout: 120 * time.Second}

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
// If cleanup is non-nil, the temp file path is registered for signal-safe removal.
func Download(version string, verbose bool, cleanup *installCleanup) (string, error) {
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

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("node.js version %s not found at %s", version, url)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(cacheDir, "driftr-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Register temp file for signal-safe cleanup.
	if cleanup != nil {
		cleanup.setTmpFile(tmpPath)
	}

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
		if cleanup != nil {
			cleanup.clearTmpFile()
		}
		return "", fmt.Errorf("download interrupted: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		if cleanup != nil {
			cleanup.clearTmpFile()
		}
		return "", fmt.Errorf("failed to save archive: %w", err)
	}

	// Temp file has been renamed to final path — no longer needs cleanup.
	if cleanup != nil {
		cleanup.clearTmpFile()
	}

	return destPath, nil
}
