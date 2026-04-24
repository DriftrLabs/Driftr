package config

import (
	"errors"
	"os"
	"path/filepath"
)

// LoadNodeVersion reads the .node-version file from the given directory.
// Returns the version string, or ("", nil) if the file doesn't exist.
func LoadNodeVersion(dir string) (string, error) {
	path := filepath.Join(dir, ".node-version")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	return parseVersionFile(string(data))
}
