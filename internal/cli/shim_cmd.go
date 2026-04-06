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
				rb, err = HandleShimError(err, tool)
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

// HandleShimError checks if the error is a NotInstalledError and offers to install.
// Returns the resolved binary on successful install, or the original error.
// Called from the fast path in main.go and from the cobra shim command.
func HandleShimError(err error, tool string) (*resolver.ResolvedBinary, error) {
	var notInstalled *resolver.NotInstalledError
	if !errors.As(err, &notInstalled) {
		return nil, err
	}

	// Auto-install if configured.
	cfg, cfgErr := config.LoadGlobal()
	if cfgErr == nil && cfg.AutoInstall {
		return autoInstall(notInstalled, tool)
	}

	// Prompt only in interactive terminals.
	if !ioutil.IsTerminal(os.Stdin) {
		return nil, err
	}

	if !promptInstall(notInstalled) {
		os.Exit(1)
	}

	return autoInstall(notInstalled, tool)
}

func autoInstall(e *resolver.NotInstalledError, tool string) (*resolver.ResolvedBinary, error) {
	fmt.Fprintf(os.Stderr, "Installing %s@%s...\n", e.Tool, e.Version)
	if _, err := installTool(e.Tool, e.Version, false); err != nil {
		return nil, fmt.Errorf("auto-install failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Installed %s %s\n", e.Tool, e.Version)

	// Retry resolution after install.
	return resolver.ResolveBinaryFull(tool, "")
}

// promptInstall asks the user whether to install a missing version.
func promptInstall(e *resolver.NotInstalledError) bool {
	msg := fmt.Sprintf("%s %s is not installed", e.Tool, e.Version)
	if e.Context != "" {
		msg = fmt.Sprintf("%s %s (%s) is not installed", e.Tool, e.Version, e.Context)
	}
	fmt.Fprintf(os.Stderr, "%s\nInstall now? [Y/n] ", msg)

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}
