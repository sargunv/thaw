package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "restore <path>",
		Short:   "Restore a materialized file back to its original symlink",
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

			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing file: %w", err)
			}

			if err := os.Symlink(entry.Target, path); err != nil {
				return fmt.Errorf("creating symlink: %w", err)
			}

			if err := rc.store.Remove(path); err != nil {
				return err
			}

			rc.printer.PrintRestored(path, entry.Target)
			return nil
		},
	}
}
