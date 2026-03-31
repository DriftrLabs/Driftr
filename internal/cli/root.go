package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var verbose bool

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "driftr",
		Short: "Fast, cross-platform JavaScript toolchain manager",
		Long:  "Driftr manages Node.js versions with automatic per-project resolution and shim-based execution.",
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
