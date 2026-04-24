package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/installer"
	"github.com/DriftrLabs/driftr/internal/ioutil"
	"github.com/DriftrLabs/driftr/internal/resolver"
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

			res, _ := resolver.ResolveTool(tool, "", false)
			activeVer := ""
			if res != nil {
				activeVer = res.Version
			}

			fmt.Printf("Installed %s versions:\n", tool)
			for _, v := range versions {
				isActive := activeVer != "" && v == activeVer
				isDefault := defaultVer != "" && v == defaultVer

				activeMark := " "
				if isActive {
					activeMark = ">"
				}
				defaultMark := " "
				if isDefault {
					defaultMark = "*"
				}
				line := activeMark + defaultMark + " " + v

				switch {
				case isActive && isDefault:
					line = ioutil.Green(ioutil.Bold(line))
				case isActive:
					line = ioutil.Green(line)
				case !isActive && !isDefault:
					line = ioutil.Dim(line)
				}

				fmt.Printf("  %s\n", line)
			}

			var legend []string
			if activeVer != "" {
				legend = append(legend, "  > = active (current directory)")
			}
			if defaultVer != "" {
				legend = append(legend, "  * = global default")
			}
			if len(legend) > 0 {
				fmt.Println()
				for _, l := range legend {
					fmt.Println(l)
				}
			}

			return nil
		},
	}
}
