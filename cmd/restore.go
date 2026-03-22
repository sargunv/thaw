package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <path>",
		Short: "Restore a materialized file back to its original symlink",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			entry, found, err := rc.store.Get(path)
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("path not tracked: %s", path)
			}

			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing file: %w", err)
			}

			if err := os.Symlink(entry.Target, path); err != nil {
				return fmt.Errorf("creating symlink: %w", err)
			}

			if err := rc.store.Remove(path); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Restored %s -> %s\n", path, entry.Target)
			return nil
		},
	}
}
