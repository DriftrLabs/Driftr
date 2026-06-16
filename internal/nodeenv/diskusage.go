package nodeenv

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// DirSize returns the total size in bytes of all regular files under path,
// walking recursively. A non-existent path returns (0, nil) — callers treat a
// missing node_modules or store as "empty", not an error.
//
// Symlinks are not followed (filepath.WalkDir uses Lstat), so linked store
// content is not double-counted.
func DirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	return total, nil
}
