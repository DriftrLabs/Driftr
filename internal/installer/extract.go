package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DriftrLabs/driftr/internal/platform"
)

// extractToRoot extracts a tar entry into an os.Root-sandboxed directory.
// The Root enforces that no path can escape destDir, replacing manual prefix checks.
func extractToRoot(root *os.Root, relPath string, hdr *tar.Header, tr *tar.Reader) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := root.MkdirAll(relPath, hdr.FileInfo().Mode().Perm()); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", relPath, err)
		}

	case tar.TypeReg:
		if dir := filepath.Dir(relPath); dir != "." {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", relPath, err)
			}
		}
		outFile, err := root.OpenFile(relPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", relPath, err)
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return fmt.Errorf("failed to write file %s: %w", relPath, err)
		}
		if err := outFile.Close(); err != nil {
			return fmt.Errorf("failed to close file %s: %w", relPath, err)
		}

	case tar.TypeSymlink:
		if dir := filepath.Dir(relPath); dir != "." {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", relPath, err)
			}
		}
		if err := root.Remove(relPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing path %s before creating symlink: %w", relPath, err)
		}
		if err := root.Symlink(hdr.Linkname, relPath); err != nil {
			return fmt.Errorf("failed to create symlink %s: %w", relPath, err)
		}
	}
	return nil
}

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

	// Use os.Root to sandbox all file operations within destDir.
	// The kernel enforces that no extracted path can escape this directory.
	root, err := os.OpenRoot(destDir)
	if err != nil {
		return fmt.Errorf("failed to open root dir: %w", err)
	}
	defer root.Close()

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

		if err := extractToRoot(root, relPath, hdr, tr); err != nil {
			return err
		}
	}

	// Verify the node binary exists after extraction.
	if _, err := root.Stat(filepath.Join("bin", "node")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("extraction completed but node binary not found at %s", nodeBin)
		}
		return fmt.Errorf("failed to verify extracted node binary at %s: %w", nodeBin, err)
	}

	return nil
}
