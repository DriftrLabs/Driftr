package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/nodeenv"
)

func newNodeReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Report dependency storage usage",
		Long:  "Show the current project's node_modules size alongside the shared pnpm store size.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			r := nodeenv.NewExecRunner()
			return runNodeReport(nodeenv.NewPnpm(r), dir, cmd.OutOrStdout())
		},
	}
}

// runNodeReport prints node_modules and shared-store sizes for the current
// project. Scope is intentionally single-project: a cross-project "savings"
// figure would be fabricated here, so the report states real sizes plus a note
// that the store is shared machine-wide.
func runNodeReport(p *nodeenv.Pnpm, projectDir string, w io.Writer) error {
	nodeModulesSize, _ := nodeenv.DirSize(filepath.Join(projectDir, "node_modules"))

	var storeSize int64
	storeKnown := false
	if p.Installed() {
		if path, err := p.StorePath(); err == nil && path != "" {
			if s, err := nodeenv.DirSize(path); err == nil {
				storeSize = s
				storeKnown = true
			}
		}
	}

	fmt.Fprintln(w, "Driftr Storage Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Project:               %s\n", projectDir)
	fmt.Fprintf(w, "Project node_modules:  %s\n", formatSize(nodeModulesSize))
	if storeKnown {
		fmt.Fprintf(w, "Shared pnpm store:     %s\n", formatSize(storeSize))
	} else {
		fmt.Fprintln(w, "Shared pnpm store:     unknown (pnpm not found)")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The shared pnpm store is reused across every project on this machine,")
	fmt.Fprintln(w, "so its cost is amortized — more projects means greater savings overall.")
	return nil
}
