package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sargunv/thaw/internal/fsutil"
	"github.com/spf13/cobra"
)

func (rc *rootCmd) newMaterializeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "materialize <path>",
		Short: "Replace a symlink with a mutable copy of its target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			fi, err := os.Lstat(path)
			if err != nil {
				return fmt.Errorf("statting path: %w", err)
			}
			if fi.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("not a symlink: %s", path)
			}

			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("reading symlink: %w", err)
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}

			targetInfo, err := os.Stat(target)
			if err != nil {
				return fmt.Errorf("statting symlink target: %w", err)
			}
			if !targetInfo.Mode().IsRegular() {
				return fmt.Errorf("symlink target is not a regular file: %s", target)
			}

			if err := rc.store.Add(path, target, time.Now()); err != nil {
				return err
			}

			if err := os.Remove(path); err != nil {
				_ = rc.store.Remove(path)
				return fmt.Errorf("removing symlink: %w", err)
			}

			if err := fsutil.CopyFile(path, target); err != nil {
				_ = rc.store.Remove(path)
				// Best-effort: restore the original symlink
				_ = os.Symlink(target, path)
				return fmt.Errorf("copying file: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Materialized %s\n", path)
			return nil
		},
	}
}
