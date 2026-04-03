package resolver

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

// RequireInstalled verifies a version string parses correctly and the version is installed.
// For partial versions (e.g. "24", "24.14") and "latest", it finds the best matching
// installed version. Returns the normalized version string and binary path, or an actionable error.
func RequireInstalled(versionSpec string) (string, string, error) {
	return RequireToolInstalled("node", versionSpec)
}

// RequireToolInstalled verifies a tool version is installed, with partial version resolution.
func RequireToolInstalled(tool, versionSpec string) (string, string, error) {
	v, err := version.Parse(versionSpec)
	if err != nil {
		return "", "", fmt.Errorf("invalid version: %w", err)
	}

	if v.Latest || v.IsPartial() {
		return resolveInstalledPartial(tool, v)
	}

	versionStr := v.String()
	binPath, err := requireToolBinaryExists(tool, versionStr, "")
	if err != nil {
		return "", "", err
	}
	return versionStr, binPath, nil
}

// resolveInstalledPartial finds the latest installed version matching a partial spec.
func resolveInstalledPartial(tool string, v version.Version) (string, string, error) {
	installed, err := ListToolVersions(tool)
	if err != nil {
		return "", "", err
	}

	var matches []version.Version
	for _, verStr := range installed {
		iv, err := version.Parse(verStr)
		if err != nil {
			continue
		}
		if v.Matches(iv) {
			matches = append(matches, iv)
		}
	}

	if len(matches) == 0 {
		if v.Latest {
			return "", "", fmt.Errorf("no %s versions installed. Run `driftr install %s@<version>`", tool, tool)
		}
		return "", "", fmt.Errorf("no installed %s version matches %s. Run `driftr install %s@%s`", tool, v.Raw, tool, v.Raw)
	}

	// Sort descending to pick the latest.
	slices.SortFunc(matches, func(a, b version.Version) int {
		if c := cmp.Compare(b.Major, a.Major); c != 0 {
			return c
		}
		if c := cmp.Compare(b.Minor, a.Minor); c != 0 {
			return c
		}
		return cmp.Compare(b.Patch, a.Patch)
	})

	best := matches[0].String()
	binPath, err := requireToolBinaryExists(tool, best, "")
	if err != nil {
		return "", "", err
	}
	return best, binPath, nil
}

// ListToolVersions returns all installed version strings for a tool.
func ListToolVersions(tool string) ([]string, error) {
	return platform.ListToolVersions(tool)
}

// requireToolBinaryExists checks that the binary for the given tool and version is installed.
func requireToolBinaryExists(tool, ver, context string) (string, error) {
	binPath, err := platform.ToolBinary(tool, ver)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binPath); errors.Is(err, os.ErrNotExist) {
		if context != "" {
			return "", fmt.Errorf("%s %s (%s) is not installed. Run `driftr install %s@%s`", tool, ver, context, tool, ver)
		}
		return "", fmt.Errorf("%s %s is not installed. Run `driftr install %s@%s`", tool, ver, tool, ver)
	}
	return binPath, nil
}

// Source describes where a version resolution came from.
type Source int

const (
	SourceExplicit    Source = iota
	SourceProject            // .driftr.toml
	SourcePackageJSON        // package.json driftr key
	SourceNvmrc              // .nvmrc
	SourceNodeVersion        // .node-version
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
	case SourceNvmrc:
		return ".nvmrc"
	case SourceNodeVersion:
		return ".node-version"
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
func ResolveNode(explicit string) (*Resolution, error) {
	return ResolveTool("node", explicit, false)
}

// ResolveNodeVerbose determines which Node.js version to use, with optional tracing.
func ResolveNodeVerbose(explicit string, verbose bool) (*Resolution, error) {
	return ResolveTool("node", explicit, verbose)
}

// ResolveTool determines which version of a tool to use.
// Resolution order: explicit > project config > global default.
func ResolveTool(tool, explicit string, verbose bool) (*Resolution, error) {
	if verbose {
		fmt.Printf("  [resolve] Starting %s version resolution\n", tool)
	}

	if explicit != "" {
		if verbose {
			fmt.Printf("  [resolve] Step 1: Explicit override provided: %s\n", explicit)
		}
		return resolveExplicit(tool, explicit)
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

	res, err := resolveFromProject(tool, cwd, verbose)
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

	res, err = resolveFromGlobal(tool)
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

func resolveExplicit(tool, ver string) (*Resolution, error) {
	binPath, err := requireToolBinaryExists(tool, ver, "")
	if err != nil {
		return nil, err
	}
	return &Resolution{
		Tool:       tool,
		Version:    ver,
		BinaryPath: binPath,
		Source:     SourceExplicit,
	}, nil
}

const maxResolveDepth = 20

func resolveFromProject(tool, dir string, verbose bool) (*Resolution, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

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
		if cfg != nil {
			if ver := cfg.Tools.GetTool(tool); ver != "" {
				return resolveProjectVersion(tool, ver, current, SourceProject)
			}
		}

		// Check package.json driftr key.
		pkgPath := filepath.Join(current, "package.json")
		if verbose {
			fmt.Printf("  [resolve]   Checking: %s (driftr)\n", pkgPath)
		}

		pkg, err := config.LoadPackageJSON(current)
		if err != nil {
			return nil, err
		}
		if pkg != nil {
			if ver := pkg.Driftr.GetTool(tool); ver != "" {
				return resolveProjectVersion(tool, ver, current, SourcePackageJSON)
			}
		}

		// Check .nvmrc and .node-version (node only).
		if tool == "node" {
			nvmrcPath := filepath.Join(current, ".nvmrc")
			if verbose {
				fmt.Printf("  [resolve]   Checking: %s\n", nvmrcPath)
			}
			ver, err := config.LoadNvmrc(current)
			if err != nil {
				return nil, err
			}
			if ver != "" {
				return resolveProjectVersion(tool, ver, current, SourceNvmrc)
			}

			nodeVersionPath := filepath.Join(current, ".node-version")
			if verbose {
				fmt.Printf("  [resolve]   Checking: %s\n", nodeVersionPath)
			}
			ver, err = config.LoadNodeVersion(current)
			if err != nil {
				return nil, err
			}
			if ver != "" {
				return resolveProjectVersion(tool, ver, current, SourceNodeVersion)
			}
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

func resolveProjectVersion(tool, ver, dir string, source Source) (*Resolution, error) {
	binPath, err := requireToolBinaryExists(tool, ver, "pinned in "+dir)
	if err != nil {
		return nil, err
	}
	return &Resolution{
		Tool:       tool,
		Version:    ver,
		BinaryPath: binPath,
		Source:     source,
		ProjectDir: dir,
	}, nil
}

func resolveFromGlobal(tool string) (*Resolution, error) {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return nil, err
	}

	ver := cfg.Default.GetTool(tool)
	if ver == "" {
		return nil, fmt.Errorf("no %s version configured. Run `driftr install %s@<version>` and `driftr default %s@<version>`", tool, tool, tool)
	}

	binPath, err := requireToolBinaryExists(tool, ver, "global default")
	if err != nil {
		return nil, err
	}
	return &Resolution{
		Tool:       tool,
		Version:    ver,
		BinaryPath: binPath,
		Source:     SourceGlobal,
	}, nil
}

// toolParent maps tools to the parent tool whose version controls resolution.
// Tools not listed here resolve independently.
var toolParent = map[string]string{
	"npm": "node",
	"npx": "node",
}

// ResolvedBinary holds the result of resolving a tool binary.
type ResolvedBinary struct {
	ToolPath string // path to the tool binary
	NodePath string // path to node binary, set when the tool needs node to execute
}

// ResolveBinary resolves the full path to a tool binary.
// For bundled tools (npm, npx), resolves via the parent tool (node).
// For standalone tools (node, pnpm, yarn), resolves via their own version.
func ResolveBinary(tool string, explicit string) (string, error) {
	rb, err := ResolveBinaryFull(tool, explicit)
	if err != nil {
		return "", err
	}
	return rb.ToolPath, nil
}

// ResolveBinaryFull resolves a tool binary with dual resolution.
// For tools that need Node.js (e.g. yarn), it also resolves the Node binary path.
func ResolveBinaryFull(tool string, explicit string) (*ResolvedBinary, error) {
	resolveTool := tool
	if parent, ok := toolParent[tool]; ok {
		resolveTool = parent
	}

	res, err := ResolveTool(resolveTool, explicit, false)
	if err != nil {
		return nil, err
	}

	toolPath, err := platform.ToolBinary(tool, res.Version)
	if err != nil {
		return nil, err
	}

	rb := &ResolvedBinary{ToolPath: toolPath}

	// Check if this tool needs Node.js to execute.
	entry, ok := platform.LookupTool(tool)
	if ok && entry.NeedsNode {
		nodeRes, err := ResolveTool("node", "", false)
		if err != nil {
			return nil, fmt.Errorf("%s requires Node.js: %w", tool, err)
		}
		nodePath, err := platform.ToolBinary("node", nodeRes.Version)
		if err != nil {
			return nil, err
		}
		rb.NodePath = nodePath
	}

	return rb, nil
}
