package cmd

import (
	"errors"
	"fmt"

	"github.com/sargunv/thaw/internal/state"
	"github.com/spf13/cobra"
)

func (rc *rootCmd) newForgetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "forget <path>",
		Short:   "Stop tracking a file without modifying the filesystem",
		Example: `  thaw forget ~/.config/foo/config.toml`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := absPath(args[0])
			if err != nil {
				return err
			}

			if err := rc.store.Remove(path); err != nil {
				if errors.Is(err, state.ErrNotTracked) {
					return fmt.Errorf("%s is not tracked by thaw; run \"thaw status\" to see tracked files", path)
				}
				return err
			}

			rc.printer.PrintForgotten(path)
			return nil
		},
	}
}
