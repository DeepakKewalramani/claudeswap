package chat

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ImportArchive extracts a chat archive (.tar.gz) into destDir safely.
// It strictly validates paths against directory traversal attacks.
func ImportArchive(archivePath, destDir string) (int, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	fileCount := 0

	cleanDestDir := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fileCount, fmt.Errorf("error reading tar archive: %w", err)
		}

		// Security check: Path traversal prevention
		relPath := filepath.Clean(header.Name)
		if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			return fileCount, fmt.Errorf("archive contains unsafe path: %q", header.Name)
		}

		target := filepath.Join(cleanDestDir, relPath)

		// Extra check: ensure target is strictly within destDir
		if !strings.HasPrefix(target, cleanDestDir) {
			return fileCount, fmt.Errorf("archive entry escapes destination: %q", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0700); err != nil {
				return fileCount, fmt.Errorf("failed to create directory %q: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return fileCount, fmt.Errorf("failed to create parent directory for %q: %w", target, err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				return fileCount, fmt.Errorf("failed to create file %q: %w", target, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fileCount, fmt.Errorf("failed to extract file %q: %w", target, err)
			}
			outFile.Close()
			fileCount++
		}
	}

	return fileCount, nil
}
