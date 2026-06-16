package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/ioutil"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/shim"
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

			fmt.Println(ioutil.Success(ioutil.Bold("Driftr setup complete!")))
			fmt.Println()
			fmt.Println("Add the following to your shell profile (~/.zshrc, ~/.bashrc, etc.):")
			fmt.Println()
			fmt.Printf("  %s\n", ioutil.Bold(fmt.Sprintf("export PATH=\"%s:$PATH\"", binDir)))
			fmt.Println()
			fmt.Println(ioutil.Dim("Then restart your shell or run: source ~/.zshrc"))

			return nil
		},
	}
}
