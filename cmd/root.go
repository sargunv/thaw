package cmd

import (
	"cmp"

	"github.com/sargunv/thaw/internal/state"
	"github.com/spf13/cobra"
)

type rootCmd struct {
	stateDir string
	store    *state.Store
}

func New() *cobra.Command {
	rc := &rootCmd{}

	cmd := &cobra.Command{
		Use:   "thaw",
		Short: "Temporarily materialize immutable symlinked config files",
		Long: `thaw temporarily replaces symlinked config files with mutable copies,
letting applications write to them. After you're done, diff the changes
against the original or restore the symlink.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			rc.store = state.NewStore(cmp.Or(rc.stateDir, state.DefaultDir()))
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rc.stateDir, "state-dir", "", "override state directory")
	_ = cmd.PersistentFlags().MarkHidden("state-dir")

	cmd.AddCommand(
		rc.newMaterializeCmd(),
		rc.newRestoreCmd(),
		rc.newClearCmd(),
	)

	return cmd
}
