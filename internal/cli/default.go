package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/resolver"
	"github.com/spf13/cobra"
)

func newDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <tool@version>",
		Short: "Set the global default Node.js version",
		Long:  "Set which Node.js version is used outside of pinned projects.\n\nExample:\n  driftr default node@24.0.0",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			versionStr, _, err := resolver.RequireInstalled(args[0])
			if err != nil {
				return err
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
