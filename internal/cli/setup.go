package cli

import (
	"fmt"

	"github.com/kisztof/driftr/internal/platform"
	"github.com/kisztof/driftr/internal/shim"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Initialize Driftr directories and generate shims",
		Long:  "Create the Driftr directory structure and generate shim scripts.\nAfter running setup, add ~/.driftr/bin to your PATH.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := platform.EnsureDirs(); err != nil {
				return fmt.Errorf("failed to create directories: %w", err)
			}

			if err := shim.GenerateShims(); err != nil {
				return fmt.Errorf("failed to generate shims: %w", err)
			}

			binDir, err := platform.BinDir()
			if err != nil {
				return err
			}

			fmt.Println("Driftr setup complete!")
			fmt.Println()
			fmt.Println("Add the following to your shell profile (~/.zshrc, ~/.bashrc, etc.):")
			fmt.Println()
			fmt.Printf("  export PATH=\"%s:$PATH\"\n", binDir)
			fmt.Println()
			fmt.Println("Then restart your shell or run: source ~/.zshrc")

			return nil
		},
	}
}
