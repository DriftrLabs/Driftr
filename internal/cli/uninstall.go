package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
)

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <tool@version>",
		Short: "Remove an installed tool version",
		Long:  "Remove a previously installed tool version and free disk space.\n\nExamples:\n  driftr uninstall node@22.14.0\n  driftr uninstall pnpm@9.15.0\n  driftr uninstall yarn@1.22.22",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool, versionSpec := parseToolVersion(args[0])

			if _, ok := platform.LookupTool(tool); !ok {
				return fmt.Errorf("unknown tool: %s. Supported tools: node, pnpm, yarn", tool)
			}

			if versionSpec == "" {
				return fmt.Errorf("version required. Usage: driftr uninstall %s@<version>", tool)
			}

			// Verify the version is installed.
			versionDir, err := platform.ToolVersionDir(tool, versionSpec)
			if err != nil {
				return err
			}
			if _, err := os.Stat(versionDir); errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%s %s is not installed", tool, versionSpec)
			}

			// Warn if this is the global default.
			cfg, err := config.LoadGlobal()
			if err == nil && cfg.Default.GetTool(tool) == versionSpec {
				fmt.Printf("Warning: %s %s is the current global default. Run `driftr default %s@<version>` to set a new one.\n", tool, versionSpec, tool)
			}

			if verbose {
				fmt.Printf("  Removing: %s\n", versionDir)
			}

			if err := os.RemoveAll(versionDir); err != nil {
				return fmt.Errorf("failed to remove %s %s: %w", tool, versionSpec, err)
			}

			fmt.Printf("Uninstalled %s %s\n", tool, versionSpec)
			return nil
		},
	}
}
