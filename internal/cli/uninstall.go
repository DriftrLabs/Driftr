package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
)

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <tool@version>",
		Short: "Remove an installed tool version",
		Long:  "Remove a previously installed tool version and free disk space.\n\nExamples:\n  driftr uninstall node@22.14.0\n  driftr uninstall pnpm@9.15.0\n  driftr uninstall yarn@1.22.22",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool, versionSpec := parseToolVersion(args[0])

			// Resolve bundled tools to their parent (e.g. npm → node, pnpx → pnpm).
			entry, ok := platform.LookupTool(tool)
			if !ok {
				return fmt.Errorf("unknown tool: %s. Supported tools: node, pnpm, yarn", tool)
			}
			tool = entry.Parent

			if versionSpec == "" {
				return fmt.Errorf("version required. Usage: driftr uninstall %s@<version>", tool)
			}

			// Validate and normalize the version string to prevent path traversal.
			v, err := version.Parse(versionSpec)
			if err != nil {
				return fmt.Errorf("invalid version: %w", err)
			}
			versionStr := v.String()

			// Verify the version is installed.
			versionDir, err := platform.ToolVersionDir(tool, versionStr)
			if err != nil {
				return err
			}
			if _, err := os.Stat(versionDir); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("%s %s is not installed", tool, versionStr)
				}
				return fmt.Errorf("failed to check installed version %s %s: %w", tool, versionStr, err)
			}

			// Warn if this is the global default.
			cfg, err := config.LoadGlobal()
			if err == nil && cfg.Default.GetTool(tool) == versionStr {
				fmt.Printf("Warning: %s %s is the current global default. Run `driftr default %s@<version>` to set a new one.\n", tool, versionStr, tool)
			}

			// Warn if this version is pinned in the project config.
			cwd, cwdErr := os.Getwd()
			if cwdErr == nil {
				if proj, err := config.LoadProject(cwd); err == nil && proj != nil && proj.Tools.GetTool(tool) == versionStr {
					fmt.Printf("Warning: %s@%s is pinned in .driftr.toml — uninstalling will break this project until you run 'driftr install %s@%s' or update the pin\n", tool, versionStr, tool, versionStr)
				}
				if pkg, err := config.LoadPackageJSON(cwd); err == nil && pkg != nil && pkg.Driftr.GetTool(tool) == versionStr {
					fmt.Printf("Warning: %s@%s is pinned in package.json — uninstalling will break this project until you run 'driftr install %s@%s' or update the pin\n", tool, versionStr, tool, versionStr)
				}
			}

			if verbose {
				fmt.Printf("  Removing: %s\n", versionDir)
			}

			if err := os.RemoveAll(versionDir); err != nil {
				return fmt.Errorf("failed to remove %s %s: %w", tool, versionStr, err)
			}

			fmt.Printf("Uninstalled %s %s\n", tool, versionStr)
			return nil
		},
	}
}
