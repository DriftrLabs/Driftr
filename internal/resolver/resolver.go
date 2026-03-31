package resolver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
)

// Source describes where a version resolution came from.
type Source int

const (
	SourceExplicit    Source = iota
	SourceProject           // .driftr.toml
	SourcePackageJSON       // package.json driftr.node
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

func resolveExplicit(version string) (*Resolution, error) {
	binPath, err := platform.NodeBinary(version)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Node %s is not installed. Run `driftr install node@%s`", version, version)
	}

	return &Resolution{
		Tool:       "node",
		Version:    version,
		BinaryPath: binPath,
		Source:     SourceExplicit,
	}, nil
}

func resolveFromProject(dir string, verbose bool) (*Resolution, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Walk up directories looking for .driftr.toml or package.json (volta),
	// checking both in each directory before moving to the parent.
	current := absDir
	for {
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

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil, nil
}

func resolveProjectVersion(version, dir string, source Source) (*Resolution, error) {
	binPath, err := platform.NodeBinary(version)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Node %s (pinned in %s) is not installed. Run `driftr install node@%s`",
			version, dir, version)
	}

	return &Resolution{
		Tool:       "node",
		Version:    version,
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

	version := cfg.Default.Node
	binPath, err := platform.NodeBinary(version)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Node %s (global default) is not installed. Run `driftr install node@%s`",
			version, version)
	}

	return &Resolution{
		Tool:       "node",
		Version:    version,
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

	switch tool {
	case "node":
		return res.BinaryPath, nil
	case "npm":
		return platform.NpmBinary(res.Version)
	case "npx":
		return platform.NpxBinary(res.Version)
	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}
