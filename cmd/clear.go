package cmd

import (
	"github.com/spf13/cobra"
)

func (rc *rootCmd) newClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "clear <path>",
		Short:   "Remove a file from thaw's tracking without touching the filesystem",
		Example: `  thaw clear ~/.config/foo/config.toml`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := absPath(args[0])
			if err != nil {
				return err
			}

			if err := rc.store.Remove(path); err != nil {
				return err
			}

			rc.printer.PrintCleared(path)
			return nil
		},
	}
}
