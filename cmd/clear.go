package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear <path>",
		Short: "Remove a file from thaw's tracking without touching the filesystem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			if err := rc.store.Remove(path); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cleared %s\n", path)
			return nil
		},
	}
}
