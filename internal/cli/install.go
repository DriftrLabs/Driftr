package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/installer"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <tool@version>",
		Short: "Install a tool version",
		Long:  "Download and install a specific tool version.\n\nExamples:\n  driftr install node@24\n  driftr install pnpm@9\n  driftr install yarn@1\n  driftr install node@latest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec := args[0]
			tool, versionSpec := parseToolVersion(spec)

			fmt.Printf("Installing %s...\n", spec)

			resolved, err := installTool(tool, versionSpec, verbose)
			if err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			fmt.Printf("Installed %s %s\n", tool, resolved)
			return nil
		},
	}
}

func installTool(tool, versionSpec string, verbose bool) (string, error) {
	// Reconstruct the spec for installers that expect "tool@version" format.
	spec := tool + "@" + versionSpec

	switch tool {
	case "node":
		return installer.Install(spec, verbose)
	case "pnpm":
		return installer.InstallPnpm(versionSpec, verbose)
	case "yarn":
		return installer.InstallYarn(versionSpec, verbose)
	default:
		return "", fmt.Errorf("unknown tool: %s. Supported tools: node, pnpm, yarn", tool)
	}
}
