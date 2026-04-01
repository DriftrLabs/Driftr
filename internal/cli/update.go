package cli

import (
	"fmt"

	"github.com/DriftrLabs/driftr/internal/updater"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
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
			}
			return nil
		},
	}
}
