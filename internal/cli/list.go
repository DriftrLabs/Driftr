package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/installer"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [tool]",
		Aliases: []string{"ls"},
		Short:   "List installed versions",
		Long:    "List installed versions for a tool. Defaults to node.\n\nExamples:\n  driftr list\n  driftr list node",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool := "node"
			if len(args) > 0 {
				tool = args[0]
			}

			versions, err := installer.ListInstalledToolVersions(tool)
			if err != nil {
				return fmt.Errorf("failed to list versions: %w", err)
			}

			if len(versions) == 0 {
				fmt.Printf("No %s versions installed.\n", tool)
				fmt.Printf("Run `driftr install %s@<version>` to get started.\n", tool)
				return nil
			}

			cfg, err := config.LoadGlobal()
			if err != nil {
				return fmt.Errorf("failed to load global config: %w", err)
			}

			defaultVer := cfg.Default.GetTool(tool)

			fmt.Printf("Installed %s versions:\n", tool)
			for _, v := range versions {
				marker := "  "
				if v == defaultVer {
					marker = "* "
				}
				fmt.Printf("  %s%s\n", marker, v)
			}

			if defaultVer != "" {
				fmt.Printf("\n  * = global default\n")
			}

			return nil
		},
	}
}
