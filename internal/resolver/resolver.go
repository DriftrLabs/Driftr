package resolver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

// RequireInstalled verifies a version string parses correctly and the version is installed.
// Returns the normalized version string and binary path, or an actionable error.
func RequireInstalled(versionSpec string) (string, string, error) {
	v, err := version.Parse(versionSpec)
	if err != nil {
		return "", "", fmt.Errorf("invalid version: %w", err)
	}
	versionStr := v.String()
	binPath, err := requireBinaryExists(versionStr, "")
	if err != nil {
		return "", "", err
	}
	return versionStr, binPath, nil
}

// requireBinaryExists checks that the node binary for the given version is installed.
// The context string (e.g. "pinned in /path") is included in the error message if non-empty.
func requireBinaryExists(ver, context string) (string, error) {
	binPath, err := platform.NodeBinary(ver)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		if context != "" {
			return "", fmt.Errorf("Node %s (%s) is not installed. Run `driftr install node@%s`", ver, context, ver)
		}
		return "", fmt.Errorf("Node %s is not installed. Run `driftr install node@%s`", ver, ver)
	}
	return binPath, nil
}

// Source describes where a version resolution came from.
type Source int

const (
	SourceExplicit    Source = iota
	SourceProject            // .driftr.toml
	SourcePackageJSON        // package.json driftr.node
	SourceGlobal
)

func (s Source) String() string {
	switch s {
	case SourceExplicit:
		return "explicit override"
	case SourceProject:
		return "project config"
	case SourcePackageJSON:
		return "package.json (driftr)"
	case SourceGlobal:
		return "global default"
	default:
		return "unknown"
	}
}

// Resolution holds the result of resolving a tool version.
type Resolution struct {
	Tool       string
	Version    string
	BinaryPath string
	Source     Source
	ProjectDir string // set when Source == SourceProject
}

// ResolveNode determines which Node.js version to use.
// Resolution order: explicit > project config > global default.
func ResolveNode(explicit string) (*Resolution, error) {
	return ResolveNodeVerbose(explicit, false)
}

// ResolveNodeVerbose determines which Node.js version to use, with optional tracing.
func ResolveNodeVerbose(explicit string, verbose bool) (*Resolution, error) {
	if verbose {
		fmt.Println("  [resolve] Starting Node.js version resolution")
	}

	if explicit != "" {
		if verbose {
			fmt.Printf("  [resolve] Step 1: Explicit override provided: %s\n", explicit)
		}
		return resolveExplicit(explicit)
	}
	if verbose {
		fmt.Println("  [resolve] Step 1: No explicit override")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("cannot determine working directory: %w", err)
	}

	if verbose {
		fmt.Printf("  [resolve] Step 2: Searching for project config from %s\n", cwd)
	}

	res, err := resolveFromProject(cwd, verbose)
	if err != nil {
		return nil, err
	}
	if res != nil {
		if verbose {
			fmt.Printf("  [resolve] Resolved: %s from %s (%s)\n", res.Version, res.Source, res.ProjectDir)
		}
		return res, nil
	}

	if verbose {
		fmt.Println("  [resolve] Step 3: No project config found, checking global default")
	}

	res, err = resolveFromGlobal()
	if err != nil {
		if verbose {
			fmt.Printf("  [resolve] Global default failed: %v\n", err)
		}
		return nil, err
	}

	if verbose {
		fmt.Printf("  [resolve] Resolved: %s from %s\n", res.Version, res.Source)
	}

	return res, nil
}

func resolveExplicit(ver string) (*Resolution, error) {
	binPath, err := requireBinaryExists(ver, "")
	if err != nil {
		return nil, err
	}
	return &Resolution{
		Tool:       "node",
		Version:    ver,
		BinaryPath: binPath,
		Source:     SourceExplicit,
	}, nil
}

const maxResolveDepth = 20

func resolveFromProject(dir string, verbose bool) (*Resolution, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Walk up directories looking for .driftr.toml or package.json (volta),
	// checking both in each directory before moving to the parent.
	current := absDir
	depth := 0
	for {
		if depth >= maxResolveDepth {
			if verbose {
				fmt.Printf("  [resolve]   Reached max depth (%d), stopping search\n", maxResolveDepth)
			}
			break
		}
		// Check .driftr.toml first.
		cfgPath := filepath.Join(current, config.ProjectConfigFile)
		if verbose {
			fmt.Printf("  [resolve]   Checking: %s\n", cfgPath)
		}

		cfg, err := config.LoadProject(current)
		if err != nil {
			return nil, err
		}
		if cfg != nil && cfg.Tools.Node != "" {
			return resolveProjectVersion(cfg.Tools.Node, current, SourceProject)
		}

		// Check package.json volta.node.
		pkgPath := filepath.Join(current, "package.json")
		if verbose {
			fmt.Printf("  [resolve]   Checking: %s (driftr)\n", pkgPath)
		}

		pkg, err := config.LoadPackageJSON(current)
		if err != nil {
			return nil, err
		}
		if pkg != nil {
			return resolveProjectVersion(pkg.Driftr.Node, current, SourcePackageJSON)
		}

		depth++
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil, nil
}

func resolveProjectVersion(ver, dir string, source Source) (*Resolution, error) {
	binPath, err := requireBinaryExists(ver, "pinned in "+dir)
	if err != nil {
		return nil, err
	}
	return &Resolution{
		Tool:       "node",
		Version:    ver,
		BinaryPath: binPath,
		Source:     source,
		ProjectDir: dir,
	}, nil
}

func resolveFromGlobal() (*Resolution, error) {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return nil, err
	}

	if cfg.Default.Node == "" {
		return nil, fmt.Errorf("no Node.js version configured. Run `driftr install node@<version>` and `driftr default node@<version>`")
	}

	ver := cfg.Default.Node
	binPath, err := requireBinaryExists(ver, "global default")
	if err != nil {
		return nil, err
	}
	return &Resolution{
		Tool:       "node",
		Version:    ver,
		BinaryPath: binPath,
		Source:     SourceGlobal,
	}, nil
}

// ResolveBinary resolves the full path to a tool binary (node, npm, npx).
func ResolveBinary(tool string, explicit string) (string, error) {
	res, err := ResolveNode(explicit)
	if err != nil {
		return "", err
	}
	return platform.ToolBinary(tool, res.Version)
}
