package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/process"
	"github.com/DriftrLabs/driftr/internal/resolver"
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

			rb, err := resolver.ResolveBinaryFull(tool, "")
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			// For tools that need Node.js (e.g. yarn), exec node with the tool script.
			if rb.NodePath != "" {
				nodeArgs := append([]string{rb.ToolPath}, toolArgs...)
				return process.Exec(rb.NodePath, nodeArgs)
			}

			// Standalone tools exec directly.
			return process.Exec(rb.ToolPath, toolArgs)
		},
	}
}
