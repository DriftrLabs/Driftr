package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/DriftrLabs/driftr/internal/platform"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the download cache",
	}

	cmd.AddCommand(newCacheCleanCmd())
	cmd.AddCommand(newCacheDirCmd())

	return cmd
}

func newCacheCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Remove all cached downloads",
		Long:  "Delete all cached archives and binaries from ~/.driftr/cache/.\nInstalled tool versions are not affected.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheDir, err := platform.CacheDir()
			if err != nil {
				return err
			}

			entries, err := os.ReadDir(cacheDir)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					fmt.Println("Cache is already empty.")
					return nil
				}
				return fmt.Errorf("failed to read cache directory: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("Cache is already empty.")
				return nil
			}

			var totalSize int64
			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				totalSize += info.Size()
			}

			if err := os.RemoveAll(cacheDir); err != nil {
				return fmt.Errorf("failed to clean cache: %w", err)
			}

			fmt.Printf("Removed %d cached file(s), freed %s.\n", len(entries), formatSize(totalSize))
			return nil
		},
	}
}

func newCacheDirCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dir",
		Short: "Print the cache directory path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheDir, err := platform.CacheDir()
			if err != nil {
				return err
			}
			fmt.Println(cacheDir)
			return nil
		},
	}
}

func formatSize(bytes int64) string {
	const (
		mb = 1024 * 1024
		kb = 1024
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
