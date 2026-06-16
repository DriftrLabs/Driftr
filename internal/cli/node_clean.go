package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/nodeenv"
)

func newNodeCleanCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove node_modules and prune the shared store",
		Long: "Reclaim disk space by removing the project's node_modules, reinstalling from\n" +
			"the shared store, and pruning orphaned packages.\n\n" +
			"Runs as a dry-run by default; pass --yes to perform the destructive operations.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			r := nodeenv.NewExecRunner()
			return runNodeClean(nodeenv.NewPnpm(r), dir, yes, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "actually perform the cleanup (otherwise dry-run)")
	return cmd
}

// runNodeClean removes node_modules, reinstalls, and prunes the store. With
// doExecute=false it only reports what would happen and mutates nothing.
func runNodeClean(p *nodeenv.Pnpm, projectDir string, doExecute bool, w io.Writer) error {
	nodeModules := filepath.Join(projectDir, "node_modules")
	// Presence is decided by stat, not size: an empty node_modules (or one
	// holding only symlinks) still needs removing and reinstalling.
	hasNodeModules := false
	if info, err := os.Stat(nodeModules); err == nil && info.IsDir() {
		hasNodeModules = true
	}
	size, _ := nodeenv.DirSize(nodeModules)

	if !doExecute {
		fmt.Fprintln(w, "Dry run — no changes made. Re-run with --yes to execute.")
		fmt.Fprintln(w)
		if hasNodeModules {
			fmt.Fprintf(w, "Would remove: %s (%s)\n", nodeModules, formatSize(size))
			fmt.Fprintln(w, "Would run:    pnpm install")
		} else {
			fmt.Fprintln(w, "No node_modules to remove.")
		}
		fmt.Fprintln(w, "Would run:    pnpm store prune")
		return nil
	}

	if hasNodeModules {
		if err := os.RemoveAll(nodeModules); err != nil {
			return fmt.Errorf("removing node_modules: %w", err)
		}
		fmt.Fprintf(w, "✓ removed node_modules (%s)\n", formatSize(size))
	} else {
		fmt.Fprintln(w, "• no node_modules to remove")
	}

	// Reinstall and prune require pnpm. If it is missing, the node_modules
	// removal above still succeeded — warn rather than fail outright.
	if !p.Installed() {
		fmt.Fprintln(w, "⚠ pnpm not found — skipped reinstall and store prune.")
		fmt.Fprintln(w, "  Run 'corepack enable', then 'pnpm install' to restore dependencies.")
		return nil
	}

	if hasNodeModules {
		fmt.Fprintln(w, "Running pnpm install...")
		if _, err := p.Install(); err != nil {
			return fmt.Errorf("pnpm install failed: %w", err)
		}
		fmt.Fprintln(w, "✓ dependencies reinstalled")
	}

	if _, err := p.StorePrune(); err != nil {
		return fmt.Errorf("pnpm store prune failed: %w", err)
	}
	fmt.Fprintln(w, "✓ pruned orphaned packages from the shared store")
	return nil
}
