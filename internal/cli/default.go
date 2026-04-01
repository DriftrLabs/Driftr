package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/resolver"
)

func newDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <tool@version>",
		Short: "Set the global default version for a tool",
		Long:  "Set which version is used outside of pinned projects.\n\nExamples:\n  driftr default node@24.0.0\n  driftr default node@24",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool, versionSpec := parseToolVersion(args[0])

			versionStr, _, err := resolver.RequireToolInstalled(tool, versionSpec)
			if err != nil {
				return err
			}

			cfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}

			cfg.Default.SetTool(tool, versionStr)
			if err := config.SaveGlobal(cfg); err != nil {
				return err
			}

			fmt.Printf("Set global default to %s %s\n", tool, versionStr)
			return nil
		},
	}
}
