package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const ProjectConfigFile = ".driftr.toml"

// ProjectConfig represents .driftr.toml in a project root.
type ProjectConfig struct {
	Tools ToolsConfig `toml:"tools"`
}

// ToolsConfig holds pinned tool versions for a project.
type ToolsConfig struct {
	Node string `toml:"node"`
}

// LoadProject reads the project config from the given directory.
func LoadProject(dir string) (*ProjectConfig, error) {
	path := filepath.Join(dir, ProjectConfigFile)

	cfg := &ProjectConfig{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse project config %s: %w", path, err)
	}

	return cfg, nil
}

// SaveProject writes a project config file in the given directory.
func SaveProject(dir string, cfg *ProjectConfig) error {
	path := filepath.Join(dir, ProjectConfigFile)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create project config: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to write project config: %w", err)
	}

	return nil
}
