package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/config"
	"github.com/DriftrLabs/driftr/internal/ioutil"
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
				rb, err = handleNotInstalled(err, tool)
				if err != nil {
					return fmt.Errorf("error: %w", err)
				}
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

// handleNotInstalled checks if the error is a NotInstalledError and offers to install.
// Returns the resolved binary on successful install, or the original error.
func handleNotInstalled(err error, tool string) (*resolver.ResolvedBinary, error) {
	var notInstalled *resolver.NotInstalledError
	if !errors.As(err, &notInstalled) {
		return nil, err
	}

	if !shouldAutoInstall(notInstalled) {
		return nil, fmt.Errorf("%s %s is not installed", notInstalled.Tool, notInstalled.Version)
	}

	fmt.Fprintf(os.Stderr, "Installing %s@%s...\n", notInstalled.Tool, notInstalled.Version)
	if _, installErr := installTool(notInstalled.Tool, notInstalled.Version, false); installErr != nil {
		return nil, fmt.Errorf("auto-install failed: %w", installErr)
	}
	fmt.Fprintf(os.Stderr, "Installed %s %s\n", notInstalled.Tool, notInstalled.Version)

	// Retry resolution after install.
	return resolver.ResolveBinaryFull(tool, "")
}

// shouldAutoInstall decides whether to install a missing version.
// Returns true if auto_install is enabled in config, or the user confirms at an interactive prompt.
func shouldAutoInstall(e *resolver.NotInstalledError) bool {
	cfg, err := config.LoadGlobal()
	if err == nil && cfg.AutoInstall {
		return true
	}

	if !ioutil.IsTerminal(os.Stdin) {
		return false
	}

	fmt.Fprintf(os.Stderr, "%s\nInstall now? [Y/n] ", e.Error())
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}
