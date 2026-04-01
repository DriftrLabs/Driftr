package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DriftrHome returns the root directory for Driftr storage (~/.driftr).
func DriftrHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".driftr"), nil
}

// BinDir returns the shim directory (~/.driftr/bin).
func BinDir() (string, error) {
	home, err := DriftrHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "bin"), nil
}

// ToolsDir returns the tools root (~/.driftr/tools).
func ToolsDir() (string, error) {
	home, err := DriftrHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "tools"), nil
}

// ToolVersionDir returns the directory for a specific tool and version.
func ToolVersionDir(tool, version string) (string, error) {
	tools, err := ToolsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(tools, tool, version), nil
}

// NodeVersionDir returns the directory for a specific Node version.
func NodeVersionDir(version string) (string, error) {
	return ToolVersionDir("node", version)
}

// NodeBinary returns the path to the node binary for a given version.
func NodeBinary(version string) (string, error) {
	return ToolBinary("node", version)
}

// NpmBinary returns the path to the npm binary for a given version.
func NpmBinary(version string) (string, error) {
	return ToolBinary("npm", version)
}

// NpxBinary returns the path to the npx binary for a given version.
func NpxBinary(version string) (string, error) {
	return ToolBinary("npx", version)
}

// CacheDir returns the cache directory (~/.driftr/cache).
func CacheDir() (string, error) {
	home, err := DriftrHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "cache"), nil
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() (string, error) {
	home, err := DriftrHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "config", "config.toml"), nil
}

// EnsureDirs creates all required Driftr directories.
func EnsureDirs() error {
	return EnsureToolDirs("node")
}

// EnsureToolDirs creates all required Driftr directories including tool-specific ones.
func EnsureToolDirs(tools ...string) error {
	home, err := DriftrHome()
	if err != nil {
		return err
	}

	dirs := []string{
		filepath.Join(home, "bin"),
		filepath.Join(home, "config"),
		filepath.Join(home, "cache"),
	}

	for _, tool := range tools {
		dirs = append(dirs, filepath.Join(home, "tools", tool))
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	return nil
}

// Arch returns the architecture string used by Node.js distribution.
func Arch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return runtime.GOARCH
	}
}

// OS returns the OS string used by Node.js distribution.
func OS() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

// toolBinaryMap maps tool names to their parent tool (for tools bundled with node)
// and binary name within the version directory.
var toolBinaryMap = map[string]struct {
	parent string // which tool's version dir contains this binary
	binary string // binary name under bin/
}{
	"node": {parent: "node", binary: "node"},
	"npm":  {parent: "node", binary: "npm"},
	"npx":  {parent: "node", binary: "npx"},
	"pnpm": {parent: "pnpm", binary: "pnpm"},
	"pnpx": {parent: "pnpm", binary: "pnpx"},
	"yarn": {parent: "yarn", binary: "yarn"},
}

// ToolBinary returns the binary path for a given tool and version.
// For bundled tools (npm, npx), the version refers to the parent tool (node).
func ToolBinary(tool, version string) (string, error) {
	entry, ok := toolBinaryMap[tool]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
	dir, err := ToolVersionDir(entry.parent, version)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bin", entry.binary), nil
}

// ArchiveExt returns the archive extension for the current platform.
func ArchiveExt() string {
	return "tar.gz"
}
