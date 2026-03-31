package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/process"
	"github.com/DriftrLabs/driftr/internal/resolver"
	"github.com/spf13/cobra"
)

// newShimCmd creates the hidden `driftr shim <tool>` command invoked by shim scripts.
func newShimCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "shim <tool>",
		Short:              "Execute a shimmed tool (internal)",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("shim: no tool specified")
			}
			tool := args[0]
			toolArgs := args[1:]

			binPath, err := resolver.ResolveBinary(tool, "")
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			// Use Exec to replace the process — preserves exit code, stdio, signals.
			return process.Exec(binPath, toolArgs)
		},
	}
}
