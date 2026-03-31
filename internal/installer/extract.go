package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kisztof/driftr/internal/platform"
)

// Extract unpacks the downloaded archive into the tools directory.
// The Node.js archive contains a top-level directory like "node-v24.0.0-darwin-arm64/".
// We extract its contents into ~/.driftr/tools/node/<version>/.
func Extract(archivePath, version string, verbose bool) error {
	destDir, err := platform.NodeVersionDir(version)
	if err != nil {
		return err
	}

	// If already extracted, skip.
	nodeBin, err := platform.NodeBinary(version)
	if err != nil {
		return err
	}
	if _, err := os.Stat(nodeBin); err == nil {
		if verbose {
			fmt.Printf("  Already extracted: %s\n", destDir)
		}
		return nil
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create version dir: %w", err)
	}

	if verbose {
		fmt.Printf("  Extracting to: %s\n", destDir)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	// The archive contains "node-v<version>-<os>-<arch>/" as prefix.
	prefix := fmt.Sprintf("node-v%s-%s-%s/", version, platform.OS(), platform.Arch())

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		name := hdr.Name
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// Strip the archive prefix to get the relative path.
		relPath := strings.TrimPrefix(name, prefix)
		if relPath == "" {
			continue
		}

		targetPath := filepath.Join(destDir, relPath)

		// Security: prevent path traversal.
		if !strings.HasPrefix(targetPath, destDir) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent dir: %w", err)
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent dir: %w", err)
			}
			os.Remove(targetPath) // Remove existing symlink if any.
			if err := os.Symlink(hdr.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", targetPath, err)
			}
		}
	}

	// Verify the node binary exists after extraction.
	if _, err := os.Stat(nodeBin); os.IsNotExist(err) {
		return fmt.Errorf("extraction completed but node binary not found at %s", nodeBin)
	}

	return nil
}
