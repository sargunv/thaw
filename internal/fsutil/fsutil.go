// Package fsutil provides filesystem utility functions used by thaw.
package fsutil

import (
	"fmt"
	"io"
	"os"

	"github.com/google/renameio/v2"
)

// CopyFile atomically writes the contents of src to dst, preserving the source
// file's permissions. The destination is written to a temporary file first,
// then atomically renamed into place.
func CopyFile(dst string, src *os.File) error {
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("statting source: %w", err)
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("reading source: %w", err)
	}

	if err := renameio.WriteFile(dst, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("writing destination: %w", err)
	}

	return nil
}
