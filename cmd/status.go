package cmd

import (
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List all materialized files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := rc.store.List()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No materialized files")
				return nil
			}

			paths := make([]string, 0, len(entries))
			for p := range entries {
				paths = append(paths, p)
			}
			slices.Sort(paths)

			for _, p := range paths {
				e := entries[p]
				_, _ = fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s -> %s (%s)\n",
					p, e.Target, e.MaterializedAt.Format(time.RFC3339),
				)
			}
			return nil
		},
	}
}
