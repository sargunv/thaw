package cmd

import (
	"github.com/spf13/cobra"
)

func (rc *rootCmd) newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "List all materialized files",
		Example: `  thaw status`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := rc.store.List()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				rc.printer.PrintNoMaterialized()
				return nil
			}

			rc.printer.PrintStatus(entries)
			return nil
		},
	}
}
