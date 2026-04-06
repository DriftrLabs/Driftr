package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
)

var shimTools = []string{"node", "npm", "npx", "pnpm", "pnpx", "yarn"}

// conflicting node version managers to detect on PATH.
var conflictingManagers = []string{"nvm", "fnm", "volta", "n"}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check your driftr installation for problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := 0
			issues += checkPath()
			issues += checkShims()
			issues += checkShimBinaryPath()
			issues += checkGlobalDefault()
			issues += checkDefaultsInstalled()
			issues += checkConflictingManagers()
			issues += checkInstalledVersions()
			issues += checkNeedsNode()

			fmt.Println()
			if issues == 0 {
				fmt.Println("No issues found.")
			} else {
				fmt.Printf("Found %d issue(s).\n", issues)
			}
			return nil
		},
	}
}

func pass(msg string) {
	fmt.Printf("  ok  %s\n", msg)
}

func warn(msg string) {
	fmt.Printf("  !!  %s\n", msg)
}

func checkPath() int {
	binDir, err := platform.BinDir()
	if err != nil {
		warn("Cannot determine bin directory")
		return 1
	}

	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range pathDirs {
		if dir == binDir {
			pass(binDir + " is on PATH")
			return 0
		}
	}

	warn(binDir + " is not on PATH — shims won't be found")
	return 1
}

func checkShims() int {
	binDir, err := platform.BinDir()
	if err != nil {
		warn("Cannot determine bin directory")
		return 1
	}

	missing := 0
	for _, tool := range shimTools {
		shimPath := filepath.Join(binDir, tool)
		info, err := os.Stat(shimPath)
		if err != nil {
			warn(fmt.Sprintf("Shim missing: %s — run `driftr setup`", tool))
			missing++
			continue
		}
		if info.Mode()&0o111 == 0 {
			warn(fmt.Sprintf("Shim not executable: %s — run `driftr setup`", tool))
			missing++
		}
	}

	if missing == 0 {
		pass(fmt.Sprintf("All %d shims installed", len(shimTools)))
	}
	return missing
}

func checkShimBinaryPath() int {
	binDir, err := platform.BinDir()
	if err != nil {
		return 0
	}

	currentBin, err := os.Executable()
	if err != nil {
		return 0
	}
	currentBin, _ = filepath.EvalSymlinks(currentBin)

	// Check the first shim to see where it points.
	shimPath := filepath.Join(binDir, "node")
	data, err := os.ReadFile(shimPath)
	if err != nil {
		return 0 // already covered by checkShims
	}

	content := string(data)
	// Extract the binary path from: exec "/path/to/driftr" shim node "$@"
	if idx := strings.Index(content, "exec \""); idx >= 0 {
		rest := content[idx+6:]
		if end := strings.Index(rest, "\""); end >= 0 {
			shimBin := rest[:end]
			resolved, _ := filepath.EvalSymlinks(shimBin)
			if resolved == "" {
				resolved = shimBin
			}
			if resolved != currentBin {
				warn(fmt.Sprintf("Shims point to %s but driftr is at %s — run `driftr setup`", shimBin, currentBin))
				return 1
			}
			pass("Shims point to current driftr binary")
			return 0
		}
	}

	return 0
}

func checkGlobalDefault() int {
	cfg, err := config.LoadGlobal()
	if err != nil {
		warn(fmt.Sprintf("Cannot read global config: %s", err))
		return 1
	}

	if cfg.Default.GetTool("node") == "" {
		warn("No global default node version — run `driftr default node@<version>`")
		return 1
	}

	pass(fmt.Sprintf("Global default: node %s", cfg.Default.GetTool("node")))
	return 0
}

func checkDefaultsInstalled() int {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return 0 // already reported by checkGlobalDefault
	}

	issues := 0
	for _, tool := range []string{"node", "pnpm", "yarn"} {
		ver := cfg.Default.GetTool(tool)
		if ver == "" {
			continue
		}
		binPath, err := platform.ToolBinary(tool, ver)
		if err != nil {
			continue
		}
		if _, err := os.Stat(binPath); err != nil {
			warn(fmt.Sprintf("Default %s %s is not installed — run `driftr install %s@%s`", tool, ver, tool, ver))
			issues++
		}
	}

	if issues == 0 {
		pass("All default versions are installed")
	}
	return issues
}

func checkConflictingManagers() int {
	binDir, err := platform.BinDir()
	if err != nil {
		return 0
	}

	issues := 0
	for _, manager := range conflictingManagers {
		path, err := exec.LookPath(manager)
		if err != nil {
			continue
		}
		// Ignore if it's our own shim directory.
		if filepath.Dir(path) == binDir {
			continue
		}
		warn(fmt.Sprintf("%s detected at %s — may conflict with driftr shims", manager, path))
		issues++
	}

	if issues == 0 {
		pass("No conflicting version managers found")
	}
	return issues
}

func checkInstalledVersions() int {
	for _, tool := range []string{"node", "pnpm", "yarn"} {
		versions, err := platform.ListToolVersions(tool)
		if err != nil || len(versions) == 0 {
			continue
		}
		pass(fmt.Sprintf("%d %s version(s) installed", len(versions), tool))
	}
	return 0
}

func checkNeedsNode() int {
	nodeVersions, err := platform.ListToolVersions("node")
	if err != nil || len(nodeVersions) > 0 {
		return 0 // node is available
	}

	issues := 0
	for _, tool := range []string{"pnpm", "yarn"} {
		versions, err := platform.ListToolVersions(tool)
		if err != nil || len(versions) == 0 {
			continue
		}
		warn(fmt.Sprintf("%s is installed but no node versions found — %s requires Node.js to run", tool, tool))
		issues++
	}
	return issues
}
