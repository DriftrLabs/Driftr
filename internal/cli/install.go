package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/installer"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <tool@version>",
		Short: "Install a Node.js version",
		Long:  "Download and install a specific Node.js version.\n\nExamples:\n  driftr install node@24\n  driftr install node@22.14.0",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec := args[0]

			fmt.Printf("Installing %s...\n", spec)

			resolved, err := installer.Install(spec, verbose)
			if err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			fmt.Printf("Installed Node.js %s\n", resolved)
			return nil
		},
	}
}
