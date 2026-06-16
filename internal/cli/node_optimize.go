package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/nodeenv"
	"github.com/DriftrLabs/driftr/internal/platform"
)

func newNodeOptimizeCmd() *cobra.Command {
	var install bool
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Configure pnpm for shared dependency storage",
		Long: "Enable corepack and point pnpm at driftr's shared content-addressable store,\n" +
			"enabling the global virtual store so dependencies are shared across projects.\n" +
			"Idempotent: already-correct settings are left untouched.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storeDir, err := platform.PnpmStoreDir()
			if err != nil {
				return err
			}
			r := nodeenv.NewExecRunner()
			return runNodeOptimize(nodeenv.NewPnpm(r), storeDir, install, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "run 'pnpm install' after configuring")
	return cmd
}

// runNodeOptimize applies the shared-store configuration idempotently. Returns
// an actionable error when pnpm is unavailable.
func runNodeOptimize(p *nodeenv.Pnpm, storeDir string, install bool, w io.Writer) error {
	fmt.Fprintln(w, "Optimizing pnpm shared dependency storage...")
	fmt.Fprintln(w)

	// corepack runs first: enabling it can make pnpm resolvable, so the pnpm
	// availability check below must come after.
	if p.CorepackAvailable() {
		if err := p.CorepackEnable(); err != nil {
			return fmt.Errorf("corepack enable failed: %w", err)
		}
		fmt.Fprintln(w, "✓ corepack enabled")
	} else {
		fmt.Fprintln(w, "• corepack not found, skipping")
	}

	if !p.Installed() {
		return fmt.Errorf("pnpm not found. Install pnpm (e.g. via 'corepack enable'), then re-run 'driftr node optimize'")
	}

	targets := []struct{ key, value string }{
		{nodeenv.StoreDirKey, storeDir},
		{nodeenv.GlobalVirtualStoreKey, "true"},
	}
	for _, t := range targets {
		current, err := p.ConfigGet(t.key)
		if err != nil {
			return fmt.Errorf("reading pnpm config %q: %w", t.key, err)
		}
		if current == t.value {
			fmt.Fprintf(w, "• %s already %s\n", t.key, t.value)
			continue
		}
		if err := p.ConfigSet(t.key, t.value); err != nil {
			return fmt.Errorf("setting pnpm config %q: %w", t.key, err)
		}
		fmt.Fprintf(w, "✓ set %s = %s\n", t.key, t.value)
	}

	if install {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Running pnpm install...")
		if _, err := p.Install(); err != nil {
			return fmt.Errorf("pnpm install failed: %w", err)
		}
		fmt.Fprintln(w, "✓ dependencies installed")
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Done. Dependencies are now stored in the shared pnpm store.")
	return nil
}
