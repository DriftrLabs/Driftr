package installer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

// installCleanup tracks resources that need cleanup if the install is
// interrupted by a signal. All fields are guarded by mu.
type installCleanup struct {
	mu      sync.Mutex
	tmpFile string // temp download file (driftr-download-*)
	version string // version being installed (partial extraction dir)
	verbose bool
}

func (c *installCleanup) setTmpFile(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tmpFile = path
}

func (c *installCleanup) clearTmpFile() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tmpFile = ""
}

func (c *installCleanup) setVersion(v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version = v
}

// run removes any tracked temp files and partial installs.
// Safe to call multiple times or concurrently.
func (c *installCleanup) run() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tmpFile != "" {
		os.Remove(c.tmpFile)
		c.tmpFile = ""
	}
	if c.version != "" {
		dir, err := platform.NodeVersionDir(c.version)
		if err == nil {
			if c.verbose {
				fmt.Fprintf(os.Stderr, "  Cleaning up partial install: %s\n", dir)
			}
			os.RemoveAll(dir)
		}
		c.version = ""
	}
}

const nodeIndexURL = "https://nodejs.org/dist/index.json"

// NodeRelease represents a single Node.js release from the index.
type NodeRelease struct {
	Version string `json:"version"`
	LTS     any    `json:"lts"`
}

// Install downloads and installs a Node.js version.
func Install(versionStr string, verbose bool) (string, error) {
	if err := platform.EnsureDirs(); err != nil {
		return "", err
	}

	v, err := version.Parse(versionStr)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	// If partial version (e.g. "24"), resolve to latest matching release.
	resolvedVersion := v.String()
	if v.IsPartial() {
		resolved, err := resolveLatestVersion(v)
		if err != nil {
			return "", err
		}
		resolvedVersion = resolved
		if verbose {
			fmt.Printf("  Resolved node@%d to %s\n", v.Major, resolvedVersion)
		}
	}

	// Check if already installed.
	nodeBin, err := platform.NodeBinary(resolvedVersion)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(nodeBin); err == nil {
		return resolvedVersion, nil
	}

	// Set up signal-aware cleanup for temp files and partial installs.
	cleanup := &installCleanup{verbose: verbose}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-sigChan:
			fmt.Fprintf(os.Stderr, "\nInterrupted, cleaning up...\n")
			cleanup.run()
			os.Exit(1)
		case <-done:
		}
	}()
	defer func() {
		signal.Stop(sigChan)
		close(done)
	}()

	cleanup.setVersion(resolvedVersion)

	archivePath, err := Download(resolvedVersion, verbose, cleanup)
	if err != nil {
		cleanup.run()
		return "", err
	}

	if err := VerifyChecksum(archivePath, resolvedVersion, verbose); err != nil {
		// Remove corrupted cached archive so next attempt re-downloads.
		os.Remove(archivePath)
		cleanup.run()
		return "", fmt.Errorf("checksum verification failed: %w", err)
	}

	if err := Extract(archivePath, resolvedVersion, verbose); err != nil {
		cleanup.run()
		return "", err
	}

	// Success — nothing to clean up.
	cleanup.setVersion("")

	return resolvedVersion, nil
}

// resolveLatestVersion finds the latest Node.js release matching a partial version.
func resolveLatestVersion(v version.Version) (string, error) {
	releases, err := fetchNodeIndex()
	if err != nil {
		return "", err
	}

	for _, rel := range releases {
		rv, err := version.Parse(rel.Version)
		if err != nil {
			continue
		}
		if rv.Major == v.Major {
			return rv.String(), nil
		}
	}

	return "", fmt.Errorf("no Node.js release found for major version %d", v.Major)
}

// ListInstalledVersions returns all installed Node.js versions.
func ListInstalledVersions() ([]string, error) {
	toolsDir, err := platform.ToolsDir()
	if err != nil {
		return nil, err
	}

	nodeDir := toolsDir + "/node"
	entries, err := os.ReadDir(nodeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read node versions: %w", err)
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}

	return versions, nil
}

func fetchNodeIndex() ([]NodeRelease, error) {
	resp, err := httpClient.Get(nodeIndexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Node.js release index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch release index: HTTP %d", resp.StatusCode)
	}

	var releases []NodeRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse release index: %w", err)
	}

	// Strip "v" prefix from versions.
	for i := range releases {
		releases[i].Version = strings.TrimPrefix(releases[i].Version, "v")
	}

	return releases, nil
}
