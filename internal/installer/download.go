package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DriftrLabs/driftr/internal/platform"
)

// isTerminal reports whether f is connected to a terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// progressWriter wraps an io.Writer to report download progress to stderr.
type progressWriter struct {
	dest      io.Writer
	total     int64 // from Content-Length; -1 if unknown
	written   int64
	lastPrint time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.dest.Write(p)
	pw.written += int64(n)

	if time.Since(pw.lastPrint) >= 100*time.Millisecond {
		pw.printProgress()
		pw.lastPrint = time.Now()
	}

	return n, err
}

func (pw *progressWriter) printProgress() {
	downloadedMB := float64(pw.written) / (1024 * 1024)
	if pw.total > 0 {
		totalMB := float64(pw.total) / (1024 * 1024)
		fmt.Fprintf(os.Stderr, "\r  Downloading: %.1f MB / %.1f MB", downloadedMB, totalMB)
	} else {
		fmt.Fprintf(os.Stderr, "\r  Downloading: %.1f MB", downloadedMB)
	}
}

func (pw *progressWriter) finish() {
	pw.printProgress()
	fmt.Fprint(os.Stderr, "\n")
}

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

	// Register temp file for signal-safe cleanup.
	if cleanup != nil {
		cleanup.setTmpFile(tmpPath)
	}

	if isTerminal(os.Stderr) {
		pw := &progressWriter{dest: tmpFile, total: resp.ContentLength}
		_, err = io.Copy(pw, resp.Body)
		pw.finish()
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
