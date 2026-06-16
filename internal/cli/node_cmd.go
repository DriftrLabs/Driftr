package cli

import (
	"github.com/spf13/cobra"
)

// newNodeCmd is the parent for Node.js dependency-storage optimization
// commands. It is distinct from the top-level `driftr doctor`, which checks the
// driftr installation itself; `driftr node doctor` inspects the project's
// Node.js / pnpm dependency environment.
func newNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Optimize shared Node.js dependency storage",
		Long: "Analyze and optimize how Node.js dependencies are stored across projects.\n\n" +
			"Driftr does not replace pnpm/npm/yarn — it configures and maintains pnpm's\n" +
			"shared content-addressable store to reduce duplicated dependency data.",
	}

	cmd.AddCommand(
		newNodeDoctorCmd(),
		newNodeOptimizeCmd(),
		newNodeCleanCmd(),
		newNodeReportCmd(),
	)

	return cmd
}
