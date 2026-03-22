package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "restore <path>",
		Short:   "Restore a materialized file to its original symlink",
		Example: `  thaw restore ~/.config/foo/config.toml`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := absPath(args[0])
			if err != nil {
				return err
			}

			entry, err := rc.getTrackedEntry(path)
			if err != nil {
				return err
			}

			// Read content and mode before removing so we can roll back.
			data, readErr := os.ReadFile(path)
			var mode os.FileMode
			if readErr == nil {
				fi, err := os.Lstat(path)
				if err != nil {
					return fmt.Errorf("reading file info: %w", err)
				}
				mode = fi.Mode().Perm()
			}

			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("removing file: %w", err)
			}

			linkTarget := entry.RawTarget
			if linkTarget == "" {
				linkTarget = entry.Target
			}

			if err := os.Symlink(linkTarget, path); err != nil {
				// Best-effort: restore the file we just deleted
				if readErr == nil {
					_ = os.WriteFile(path, data, mode)
				}
				return fmt.Errorf("creating symlink: %w", err)
			}

			if err := rc.store.Remove(path); err != nil {
				return fmt.Errorf("removing from state: %w", err)
			}

			rc.printer.PrintRestored(path, entry.Target)
			return nil
		},
	}
}
