package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/installer"
	"github.com/DriftrLabs/driftr/internal/ioutil"
)

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <tool[@version]>",
		Short: "Install a tool version",
		Long:  "Download and install a tool version.\n\nA bare tool name installs the newest release.\n\nExamples:\n  driftr install pnpm        # latest pnpm\n  driftr install node        # latest node\n  driftr install node@24\n  driftr install pnpm@9\n  driftr install yarn@1\n  driftr install node@latest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw := args[0]
			tool, versionSpec := parseToolVersion(raw)
			if versionSpec == "" {
				// "tool@" is a malformed spec (the user wrote "@" but omitted the
				// version); reject it rather than silently installing latest. A
				// bare tool name ("driftr install pnpm") has no "@" and means
				// "install the newest release".
				if strings.Contains(raw, "@") {
					return fmt.Errorf("version required after '@'. Use 'driftr install %s' for the latest, or '%s@<version>'", tool, tool)
				}
				versionSpec = "latest"
			}

			fmt.Println(ioutil.Dim(fmt.Sprintf("Installing %s@%s...", tool, versionSpec)))

			resolved, err := installTool(tool, versionSpec, verbose)
			if err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			fmt.Println(ioutil.Success(fmt.Sprintf("Installed %s %s", tool, ioutil.Bold(resolved))))
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
