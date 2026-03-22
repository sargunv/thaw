package fsutil

import (
	"fmt"
	"io"
	"os"
)

// CopyFile copies src to dst, preserving the source file's permissions.
// dst must not already exist.
func CopyFile(dst, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("statting source: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = dstFile.Close()
		_ = os.Remove(dst)
		return fmt.Errorf("copying data: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("closing destination: %w", err)
	}

	if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	return nil
}
