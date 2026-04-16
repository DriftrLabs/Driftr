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
	"github.com/DriftrLabs/driftr/internal/shim"
)

// versionedTools are tools that have independently installed versions.
var versionedTools = []string{"node", "pnpm", "yarn"}

// conflicting node version managers to detect on PATH.
var conflictingBinaries = []string{"fnm", "volta", "n"}

const shimExecPrefix = `exec "`

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check your driftr installation for problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			binDir, err := platform.BinDir()
			if err != nil {
				return fmt.Errorf("cannot determine driftr bin directory: %w", err)
			}
			cfg, cfgErr := config.LoadGlobal()

			toolVersions := make(map[string][]string)
			issues := 0
			for _, tool := range versionedTools {
				versions, err := platform.ListToolVersions(tool)
				if err != nil {
					warn(fmt.Sprintf("Cannot list installed versions for %s: %s", tool, err))
					issues++
					continue
				}
				toolVersions[tool] = versions
			}

			issues += checkPath(binDir)
			issues += checkShims(binDir)
			issues += checkShimBinaryPath(binDir)
			issues += checkGlobalDefault(cfg, cfgErr)
			issues += checkDefaultsInstalled(cfg)
			issues += checkConflictingManagers(binDir)
			issues += checkInstalledVersions(toolVersions)
			issues += checkNeedsNode(toolVersions)

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

func checkPath(binDir string) int {
	cleanBinDir := filepath.Clean(binDir)
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(dir) == cleanBinDir {
			pass(binDir + " is on PATH")
			return 0
		}
	}

	warn(binDir + " is not on PATH — shims won't be found")
	return 1
}

func checkShims(binDir string) int {
	missing := 0
	for _, tool := range shim.ShimTools() {
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
		pass(fmt.Sprintf("All %d shims installed", len(shim.ShimTools())))
	}
	return missing
}

func checkShimBinaryPath(binDir string) int {
	currentBin, err := os.Executable()
	if err != nil {
		return 0
	}
	currentBin, _ = filepath.EvalSymlinks(currentBin)

	shimPath := filepath.Join(binDir, "node")
	data, err := os.ReadFile(shimPath)
	if err != nil {
		return 0 // already covered by checkShims
	}

	content := string(data)
	if idx := strings.Index(content, shimExecPrefix); idx >= 0 {
		rest := content[idx+len(shimExecPrefix):]
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

func checkGlobalDefault(cfg *config.GlobalConfig, cfgErr error) int {
	if cfgErr != nil {
		warn(fmt.Sprintf("Cannot read global config: %s", cfgErr))
		return 1
	}

	if cfg.Default.GetTool("node") == "" {
		warn("No global default node version — run `driftr default node@<version>`")
		return 1
	}

	pass(fmt.Sprintf("Global default: node %s", cfg.Default.GetTool("node")))
	return 0
}

func checkDefaultsInstalled(cfg *config.GlobalConfig) int {
	if cfg == nil {
		return 0
	}

	issues := 0
	for _, tool := range versionedTools {
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

func checkConflictingManagers(binDir string) int {
	issues := 0

	// nvm is a shell function, not a binary — check $NVM_DIR instead.
	if nvmDir := os.Getenv("NVM_DIR"); nvmDir != "" {
		if _, err := os.Stat(filepath.Join(nvmDir, "nvm.sh")); err == nil {
			warn(fmt.Sprintf("nvm detected ($NVM_DIR=%s) — may conflict with driftr shims", nvmDir))
			issues++
		}
	}

	for _, manager := range conflictingBinaries {
		path, err := exec.LookPath(manager)
		if err != nil {
			continue
		}
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

func checkInstalledVersions(toolVersions map[string][]string) int {
	for _, tool := range versionedTools {
		if versions := toolVersions[tool]; len(versions) > 0 {
			pass(fmt.Sprintf("%d %s version(s) installed", len(versions), tool))
		}
	}
	return 0
}

func checkNeedsNode(toolVersions map[string][]string) int {
	if len(toolVersions["node"]) > 0 {
		return 0
	}

	issues := 0
	for _, tool := range []string{"pnpm", "yarn"} {
		if len(toolVersions[tool]) > 0 {
			warn(fmt.Sprintf("%s is installed but no node versions found — %s requires Node.js to run", tool, tool))
			issues++
		}
	}
	return issues
}
