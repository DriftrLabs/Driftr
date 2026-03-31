package cli

import (
	"fmt"
	"os"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/version"
	"github.com/spf13/cobra"
)

func newDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <tool@version>",
		Short: "Set the global default Node.js version",
		Long:  "Set which Node.js version is used outside of pinned projects.\n\nExample:\n  driftr default node@24.0.0",
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

			cfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}

			cfg.Default.Node = versionStr
			if err := config.SaveGlobal(cfg); err != nil {
				return err
			}

			fmt.Printf("Set global default to Node.js %s\n", versionStr)
			return nil
		},
	}
}
