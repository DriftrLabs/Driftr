package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PackageJSON represents the relevant fields from a package.json file.
type PackageJSON struct {
	Driftr DriftrConfig `json:"driftr"`
}

// DriftrConfig holds the Driftr tool pinning from package.json.
type DriftrConfig struct {
	Node string `json:"node"`
}

// LoadPackageJSON reads the driftr.node version from package.json in the given directory.
// Returns (nil, nil) if the file doesn't exist or has no driftr.node field.
func LoadPackageJSON(dir string) (*PackageJSON, error) {
	path := filepath.Join(dir, "package.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if pkg.Driftr.Node == "" {
		return nil, nil
	}

	return &pkg, nil
}

// SavePackageJSON writes the driftr.node version into an existing package.json.
// Returns an error if package.json does not exist in the directory.
func SavePackageJSON(dir string, nodeVersion string) error {
	path := filepath.Join(dir, "package.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no package.json found in %s. Run `npm init` first", dir)
		}
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	driftrValue, err := json.Marshal(DriftrConfig{Node: nodeVersion})
	if err != nil {
		return err
	}
	raw["driftr"] = driftrValue

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	out = append(out, '\n')

	return os.WriteFile(path, out, 0o644)
}

// RemoveDriftrFromPackageJSON removes the driftr key from package.json.
// Returns nil if package.json doesn't exist or has no driftr key.
func RemoveDriftrFromPackageJSON(dir string) error {
	path := filepath.Join(dir, "package.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if _, exists := raw["driftr"]; !exists {
		return nil
	}
	delete(raw, "driftr")

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	out = append(out, '\n')

	return os.WriteFile(path, out, 0o644)
}
