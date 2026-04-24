package installer

import (
	"cmp"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DriftrLabs/driftr/internal/ioutil"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

const registryBaseURL = "https://registry.npmjs.org"

const maxRegistryDownloadBytes = 50 * 1024 * 1024 // 50 MB

// registryPackage represents the top-level npm registry response for a package.
type registryPackage struct {
	Name     string                     `json:"name"`
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]registryVersion `json:"versions"`
}

// registryVersion represents a single version from the npm registry.
type registryVersion struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Dist    registryDist      `json:"dist"`
	Bin     map[string]string `json:"bin,omitempty"`
}

// registryDist holds the distribution metadata for an npm package version.
type registryDist struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"` // SRI format: "sha512-<base64>"
	Shasum    string `json:"shasum"`    // sha1 hex (fallback)
}

// FetchRegistryVersion fetches metadata for a specific package version from the npm registry.
func FetchRegistryVersion(pkg, ver string) (*registryVersion, error) {
	url := fmt.Sprintf("%s/%s/%s", registryBaseURL, pkg, ver)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s@%s from registry: %w", pkg, ver, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s@%s not found in npm registry", pkg, ver)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned status %d for %s@%s", resp.StatusCode, pkg, ver)
	}

	var rv registryVersion
	if err := json.NewDecoder(resp.Body).Decode(&rv); err != nil {
		return nil, fmt.Errorf("failed to parse registry response for %s@%s: %w", pkg, ver, err)
	}
	return &rv, nil
}

// ResolveRegistryLatest finds the latest version of an npm package matching a partial version spec.
// For "latest" or when no constraint is given, returns the dist-tag "latest".
func ResolveRegistryLatest(pkg string, v version.Version) (string, error) {
	url := fmt.Sprintf("%s/%s", registryBaseURL, pkg)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s from registry: %w", pkg, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry returned status %d for %s", resp.StatusCode, pkg)
	}

	var rp registryPackage
	if err := json.NewDecoder(resp.Body).Decode(&rp); err != nil {
		return "", fmt.Errorf("failed to parse registry response for %s: %w", pkg, err)
	}

	// For "latest", use the dist-tag.
	if v.Latest {
		latest, ok := rp.DistTags["latest"]
		if !ok {
			return "", fmt.Errorf("no 'latest' dist-tag for %s", pkg)
		}
		return latest, nil
	}

	// Find the highest version matching the partial spec.
	// Collect all matching versions, then pick the highest.
	var best *version.Version
	for verStr := range rp.Versions {
		rv, err := version.Parse(verStr)
		if err != nil {
			continue
		}
		if !v.Matches(rv) {
			continue
		}
		if best == nil || versionCompare(rv, *best) > 0 {
			rv := rv // copy
			best = &rv
		}
	}

	if best == nil {
		return "", fmt.Errorf("no %s version found matching %s", pkg, v.Raw)
	}
	return best.String(), nil
}

// versionCompare returns a positive value if a > b, negative if a < b, zero if equal.
func versionCompare(a, b version.Version) int {
	if c := cmp.Compare(a.Major, b.Major); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Minor, b.Minor); c != 0 {
		return c
	}
	return cmp.Compare(a.Patch, b.Patch)
}

// DownloadRegistryPackage downloads an npm package tarball to the cache directory.
// Returns the path to the downloaded file and the registry version metadata.
func DownloadRegistryPackage(pkg, ver string, verbose bool) (string, *registryVersion, error) {
	rv, err := FetchRegistryVersion(pkg, ver)
	if err != nil {
		return "", nil, err
	}

	cacheDir, err := platform.CacheDir()
	if err != nil {
		return "", nil, err
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.tgz", pkg, ver)
	destPath := filepath.Join(cacheDir, filename)

	// Skip download if already cached.
	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		if verbose {
			fmt.Printf("  Using cached archive: %s\n", destPath)
		}
		return destPath, rv, nil
	}

	tarballURL := rv.Dist.Tarball
	if verbose {
		fmt.Printf("  Downloading: %s\n", tarballURL)
	}

	resp, err := httpClient.Get(tarballURL)
	if err != nil {
		return "", nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(cacheDir, "driftr-registry-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	limited := io.LimitReader(resp.Body, maxRegistryDownloadBytes)
	if ioutil.IsTerminal(os.Stderr) {
		pw := &ioutil.ProgressWriter{Dest: tmpFile, Total: resp.ContentLength}
		_, err = io.Copy(pw, limited)
		pw.Finish()
	} else {
		_, err = io.Copy(tmpFile, limited)
	}
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("download interrupted: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("failed to save archive: %w", err)
	}

	return destPath, rv, nil
}

// VerifyIntegrity verifies a file against an SRI integrity string (e.g. "sha512-<base64>").
func VerifyIntegrity(filePath, integrity string) error {
	algo, expectedHash, err := parseSRI(integrity)
	if err != nil {
		return err
	}

	if algo != "sha512" {
		return fmt.Errorf("unsupported integrity algorithm: %s (expected sha512)", algo)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for integrity check: %w", err)
	}
	defer f.Close()

	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	actual := h.Sum(nil)
	if subtle.ConstantTimeCompare(actual, expectedHash) != 1 {
		return fmt.Errorf("integrity mismatch: expected sha512-%s, got sha512-%s",
			base64.StdEncoding.EncodeToString(expectedHash),
			base64.StdEncoding.EncodeToString(actual))
	}

	return nil
}

// parseSRI parses an SRI integrity string like "sha512-<base64>" into algorithm and raw hash bytes.
func parseSRI(sri string) (string, []byte, error) {
	parts := strings.SplitN(sri, "-", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid SRI format: %q", sri)
	}

	algo := parts[0]
	hashBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("invalid base64 in SRI: %w", err)
	}

	return algo, hashBytes, nil
}
