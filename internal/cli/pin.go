package cli

import (
	"fmt"
	"os"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
	"github.com/spf13/cobra"
)

func newPinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin <tool@version>",
		Short: "Pin a Node.js version to the current project",
		Long:  "Create or update .driftr.toml in the current directory to pin a Node.js version.\n\nExample:\n  driftr pin node@22.14.0",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := version.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid version: %w", err)
			}

			versionStr := v.String()

			// Verify the version is installed.
			nodeBin, err := platform.NodeBinary(versionStr)
			if err != nil {
				return err
			}
			if _, err := os.Stat(nodeBin); os.IsNotExist(err) {
				return fmt.Errorf("Node %s is not installed. Run `driftr install node@%s` first", versionStr, versionStr)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine current directory: %w", err)
			}

			// Load existing config or create new one.
			cfg, err := config.LoadProject(cwd)
			if err != nil {
				return err
			}
			if cfg == nil {
				cfg = &config.ProjectConfig{}
			}

			cfg.Tools.Node = versionStr
			if err := config.SaveProject(cwd, cfg); err != nil {
				return err
			}

			fmt.Printf("Pinned Node.js %s in %s/%s\n", versionStr, cwd, config.ProjectConfigFile)
			return nil
		},
	}
}
