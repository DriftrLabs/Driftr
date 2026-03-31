package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/installer"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed Node.js versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			versions, err := installer.ListInstalledVersions()
			if err != nil {
				return fmt.Errorf("failed to list versions: %w", err)
			}

			if len(versions) == 0 {
				fmt.Println("No Node.js versions installed.")
				fmt.Println("Run `driftr install node@<version>` to get started.")
				return nil
			}

			cfg, err := config.LoadGlobal()
			if err != nil {
				return fmt.Errorf("failed to load global config: %w", err)
			}

			fmt.Println("Installed Node.js versions:")
			for _, v := range versions {
				marker := "  "
				if v == cfg.Default.Node {
					marker = "* "
				}
				fmt.Printf("  %s%s\n", marker, v)
			}

			if cfg.Default.Node != "" {
				fmt.Printf("\n  * = global default\n")
			}

			return nil
		},
	}
}
