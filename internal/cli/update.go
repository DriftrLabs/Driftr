package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/pathsetup"
	"github.com/DriftrLabs/driftr/internal/platform"
	"github.com/DriftrLabs/driftr/internal/updater"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update driftr to the latest version",
		Long:  "Check for a newer version of driftr and replace the current binary.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			newVersion, err := updater.Update(Version, verbose)
			if err != nil {
				return fmt.Errorf("update failed: %w", err)
			}

			if newVersion == "" {
				fmt.Printf("driftr v%s is already the latest version.\n", Version)
			} else {
				fmt.Printf("Updated successfully to driftr v%s!\n", newVersion)
				migratePathConfig()
			}
			return nil
		},
	}
}

// migratePathConfig is a best-effort repair of legacy PATH placement after
// a successful self-update. Older installers wrote the PATH export to .zshrc
// (interactive only); this moves the configuration to a file that every shell
// invocation sources, so driftr works in scripts and non-interactive shells.
//
// Silent on any failure — self-update's primary job (binary swap) already
// succeeded and we don't want to mask that with rc-file errors.
func migratePathConfig() {
	binDir, err := platform.BinDir()
	if err != nil {
		return
	}
	r, err := pathsetup.Detect(binDir)
	if err != nil || !r.NeedsFix() {
		return
	}
	wrote, file, err := pathsetup.Apply(r)
	if err != nil || !wrote {
		return
	}
	fmt.Printf("Migrated PATH config to %s for universal shell coverage.\n", file)
	if len(r.StaleFiles) > 0 {
		fmt.Printf("  Note: legacy entries remain in %s — safe to remove.\n", strings.Join(r.StaleFiles, ", "))
	}
}
