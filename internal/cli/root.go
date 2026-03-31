package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build metadata, injected via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var verbose bool

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "driftr",
		Short:   "Fast, cross-platform JavaScript toolchain manager",
		Long:    "Driftr manages Node.js versions with automatic per-project resolution and shim-based execution.",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	root.AddCommand(
		newInstallCmd(),
		newDefaultCmd(),
		newPinCmd(),
		newListCmd(),
		newWhichCmd(),
		newRunCmd(),
		newShimCmd(),
		newSetupCmd(),
	)

	return root
}

func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
