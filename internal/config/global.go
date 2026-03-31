package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/kisztof/driftr/internal/platform"
)

// GlobalConfig represents ~/.driftr/config/config.toml
type GlobalConfig struct {
	Default DefaultConfig `toml:"default"`
}

// DefaultConfig holds the default tool versions.
type DefaultConfig struct {
	Node string `toml:"node"`
}

// LoadGlobal reads the global configuration file.
func LoadGlobal() (*GlobalConfig, error) {
	path, err := platform.GlobalConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &GlobalConfig{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	return cfg, nil
}

// SaveGlobal writes the global configuration file.
func SaveGlobal(cfg *GlobalConfig) error {
	path, err := platform.GlobalConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create global config: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}

	return nil
}
