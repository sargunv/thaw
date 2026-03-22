package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sargunv/thaw/internal/fsutil"
	"github.com/sargunv/thaw/internal/state"
	"github.com/spf13/cobra"
)

func (rc *rootCmd) newMaterializeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "materialize <path>",
		Short: "Replace a symlink with a mutable copy of its target",
		Example: `  thaw materialize ~/.config/foo/config.toml
  # app writes to the file
  thaw diff ~/.config/foo/config.toml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := absPath(args[0])
			if err != nil {
				return err
			}

			fi, err := os.Lstat(path)
			if err != nil {
				return fmt.Errorf("reading path info: %w", err)
			}
			if fi.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("not a symlink: %s", path)
			}

			rawTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("reading symlink: %w", err)
			}
			target := rawTarget
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}

			// Open the target file once to avoid TOCTOU races.
			targetFile, err := os.Open(target)
			if err != nil {
				return fmt.Errorf("opening symlink target: %w", err)
			}
			defer func() { _ = targetFile.Close() }()

			targetInfo, err := targetFile.Stat()
			if err != nil {
				return fmt.Errorf("reading symlink target info: %w", err)
			}
			if !targetInfo.Mode().IsRegular() {
				return fmt.Errorf("symlink target is not a regular file: %s", target)
			}

			if err := rc.store.Add(path, target, rawTarget, time.Now()); err != nil {
				if !errors.Is(err, state.ErrAlreadyMaterialized) {
					return err
				}
				// Path is a symlink but state has a stale entry — replace it.
				if err := rc.store.Remove(path); err != nil {
					return err
				}
				if err := rc.store.Add(path, target, rawTarget, time.Now()); err != nil {
					return err
				}
			}

			if err := os.Remove(path); err != nil {
				if rbErr := rc.store.Remove(path); rbErr != nil {
					return errors.Join(fmt.Errorf("removing symlink: %w", err), fmt.Errorf("rolling back state: %w", rbErr))
				}
				return fmt.Errorf("removing symlink: %w", err)
			}

			if err := fsutil.CopyFile(path, targetFile); err != nil {
				copyErr := fmt.Errorf("copying file: %w", err)
				if rbErr := rc.store.Remove(path); rbErr != nil {
					copyErr = errors.Join(copyErr, fmt.Errorf("rolling back state: %w", rbErr))
				}
				// Best-effort: restore the original symlink
				if linkErr := os.Symlink(rawTarget, path); linkErr != nil {
					copyErr = errors.Join(copyErr, fmt.Errorf("restoring symlink: %w", linkErr))
				}
				return copyErr
			}

			rc.printer.PrintMaterialized(path)
			return nil
		},
	}
}
