package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

// ExtractRegistryPackage extracts an npm registry tarball to the destination directory.
// npm tarballs contain a top-level "package/" prefix which is stripped during extraction.
func ExtractRegistryPackage(archivePath, destDir string) error {
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

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	// Use os.Root to sandbox all file operations within destDir.
	root, err := os.OpenRoot(destDir)
	if err != nil {
		return fmt.Errorf("failed to open root dir: %w", err)
	}
	defer root.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Strip the "package/" prefix that all npm tarballs use.
		name := hdr.Name
		if i := strings.Index(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if name == "" {
			continue
		}

		if err := extractToRoot(root, name, hdr, tr); err != nil {
			return err
		}
	}

	return nil
}
