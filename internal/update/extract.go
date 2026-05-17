package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/danieljustus/OpenPass/internal/pathutil"
)

var (
	// ErrBinaryNotFound is returned when the expected binary is not found
	// inside the archive.
	ErrBinaryNotFound = errors.New("binary not found in archive")

	// ErrPathTraversal is returned when an archive entry attempts to escape
	// the destination directory.
	ErrPathTraversal = errors.New("archive entry attempts path traversal")
)

// safeArchivePath validates that entryPath does not escape destDir. It checks
// for parent-directory traversal segments and verifies that the cleaned,
// resolved path is a child of destDir.
func safeArchivePath(destDir, entryPath string) (string, error) {
	if pathutil.HasTraversal(entryPath) {
		return "", fmt.Errorf("%w: %q", ErrPathTraversal, entryPath)
	}

	cleanDest := filepath.Clean(destDir)
	fullPath := filepath.Clean(filepath.Join(cleanDest, entryPath))

	// Ensure the resolved path is within destDir (or is destDir itself for
	// the root directory entry).
	if !strings.HasPrefix(fullPath, cleanDest+string(filepath.Separator)) && fullPath != cleanDest {
		return "", fmt.Errorf("%w: %q resolves outside %q", ErrPathTraversal, entryPath, destDir)
	}

	return fullPath, nil
}

// ExtractTarGz extracts a gzip-compressed tar archive to destDir and returns
// the path to the extracted binary matching expectedBinaryName. It protects
// against path traversal attacks and skips symlinks for safety.
func ExtractTarGz(data []byte, destDir, expectedBinaryName string) (string, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("decompress gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var binaryPath string

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read tar header: %w", err)
		}

		safePath, err := safeArchivePath(destDir, header.Name)
		if err != nil {
			return "", err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(safePath, 0o755); err != nil {
				return "", fmt.Errorf("create directory %q: %w", safePath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(safePath), 0o755); err != nil {
				return "", fmt.Errorf("create parent dir for %q: %w", safePath, err)
			}

			f, err := os.OpenFile(safePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", fmt.Errorf("create file %q: %w", safePath, err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return "", fmt.Errorf("write file %q: %w", safePath, err)
			}
			_ = f.Close()

			if filepath.Base(header.Name) == expectedBinaryName {
				binaryPath = safePath
			}

		default:
			// Skip symlinks, hard links, special devices, etc.
			continue
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, expectedBinaryName)
	}

	return binaryPath, nil
}

// ExtractZip extracts a zip archive to destDir and returns the path to the
// extracted binary matching expectedBinaryName. It protects against path
// traversal attacks.
func ExtractZip(data []byte, destDir, expectedBinaryName string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open zip archive: %w", err)
	}

	var binaryPath string

	for _, f := range zr.File {
		safePath, err := safeArchivePath(destDir, f.Name)
		if err != nil {
			return "", err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(safePath, 0o755); err != nil {
				return "", fmt.Errorf("create directory %q: %w", safePath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(safePath), 0o755); err != nil {
			return "", fmt.Errorf("create parent dir for %q: %w", safePath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("open zip entry %q: %w", f.Name, err)
		}

		out, err := os.OpenFile(safePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			_ = rc.Close()
			return "", fmt.Errorf("create file %q: %w", safePath, err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return "", fmt.Errorf("write file %q: %w", safePath, err)
		}

		_ = out.Close()
		_ = rc.Close()

		if filepath.Base(f.Name) == expectedBinaryName {
			binaryPath = safePath
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, expectedBinaryName)
	}

	return binaryPath, nil
}
