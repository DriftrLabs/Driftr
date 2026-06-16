package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/ioutil"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/resolver"
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

			fmt.Printf("%s %s\n", ioutil.Label("Tool:   "), tool)
			fmt.Printf("%s %s\n", ioutil.Label("Version:"), ioutil.Bold(res.Version))
			fmt.Printf("%s %s\n", ioutil.Label("Binary: "), ioutil.Green(binPath))
			fmt.Printf("%s %s\n", ioutil.Label("Source: "), res.Source)
			if res.ProjectDir != "" {
				fmt.Printf("%s %s\n", ioutil.Label("Project:"), res.ProjectDir)
			}

			return nil
		},
	}
}
