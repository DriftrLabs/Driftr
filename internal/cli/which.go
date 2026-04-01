package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/resolver"
	"github.com/spf13/cobra"
)

func newWhichCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "which <tool>",
		Short: "Show which binary Driftr would execute",
		Long:  "Display the resolved binary path and the source of the version decision.\n\nExample:\n  driftr which node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool := args[0]

			res, err := resolver.ResolveTool(tool, "", verbose)
			if err != nil {
				return err
			}

			binPath, err := platform.ToolBinary(tool, res.Version)
			if err != nil {
				return err
			}

			fmt.Printf("Tool:    %s\n", tool)
			fmt.Printf("Version: %s\n", res.Version)
			fmt.Printf("Binary:  %s\n", binPath)
			fmt.Printf("Source:  %s\n", res.Source)
			if res.ProjectDir != "" {
				fmt.Printf("Project: %s\n", res.ProjectDir)
			}

			return nil
		},
	}
}
