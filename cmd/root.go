package cmd

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thaw",
		Short: "Temporarily materialize immutable symlinked config files",
		Long: `thaw temporarily replaces symlinked config files with mutable copies,
letting applications write to them. After you're done, diff the changes
against the original or restore the symlink.`,
	}
	return cmd
}
