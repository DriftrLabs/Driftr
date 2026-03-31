package installer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

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

	archivePath, err := Download(resolvedVersion, verbose)
	if err != nil {
		return "", err
	}

	if err := VerifyChecksum(archivePath, resolvedVersion, verbose); err != nil {
		// Remove corrupted cached archive so next attempt re-downloads.
		os.Remove(archivePath)
		return "", fmt.Errorf("checksum verification failed: %w", err)
	}

	if err := Extract(archivePath, resolvedVersion, verbose); err != nil {
		// Clean up partial extraction so it doesn't look installed.
		cleanupFailedInstall(resolvedVersion, verbose)
		return "", err
	}

	return resolvedVersion, nil
}

// cleanupFailedInstall removes a partially extracted version directory.
func cleanupFailedInstall(version string, verbose bool) {
	dir, err := platform.NodeVersionDir(version)
	if err != nil {
		return
	}
	if verbose {
		fmt.Printf("  Cleaning up partial install: %s\n", dir)
	}
	os.RemoveAll(dir)
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
	resp, err := http.Get(nodeIndexURL)
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
