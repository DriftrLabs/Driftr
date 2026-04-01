package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/process"
	"github.com/DriftrLabs/driftr/internal/resolver"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var nodeVersion string

	cmd := &cobra.Command{
		Use:                "run [flags] -- <command> [args...]",
		Short:              "Run a command under a specific Node.js version",
		Long:               "Execute a command using an explicitly specified Node.js version without changing defaults.\n\nExamples:\n  driftr run --node 24.0.0 -- npm test\n  driftr run --node 24 -- node -v",
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no command specified. Usage: driftr run --node <version> -- <command> [args...]")
			}

			tool := args[0]
			toolArgs := args[1:]

			binPath, err := resolver.ResolveBinary(tool, nodeVersion)
			if err != nil {
				return err
			}

			exitCode, err := process.Run(binPath, toolArgs)
			if err != nil {
				return err
			}

			return &ExitError{Code: exitCode}
		},
	}

	cmd.Flags().StringVar(&nodeVersion, "node", "", "Node.js version to use")

	return cmd
}
