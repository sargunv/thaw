package cmd

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/sargunv/thaw/internal/state"
	"github.com/sargunv/thaw/internal/ui"
	"github.com/spf13/cobra"
)

type rootCmd struct {
	stateDir string
	store    *state.Store
	printer  *ui.Printer
}

func New() *cobra.Command {
	rc := &rootCmd{}

	cmd := &cobra.Command{
		Use:   "thaw",
		Short: "Temporarily materialize immutable symlinked config files",
		Long: `thaw temporarily replaces symlinked config files with mutable copies,
letting applications write to them. After you're done, diff the changes
against the original or restore the symlink.

Discover a config change and feed it back:

  thaw materialize ~/.config/foo/config.toml
  # app writes to the file
  thaw diff ~/.config/foo/config.toml
  # update your dotfile source based on the diff
  thaw clear ~/.config/foo/config.toml

Temporarily materialize, then restore:

  thaw materialize ~/.config/foo/config.toml
  # poke around, let the app write
  thaw restore ~/.config/foo/config.toml`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			rc.store = state.NewStore(cmp.Or(rc.stateDir, state.DefaultDir()))

			w := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
			var isDark bool
			if term.IsTerminal(os.Stdout.Fd()) {
				isDark = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
			}
			rc.printer = ui.NewPrinter(w, isDark)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rc.stateDir, "state-dir", "", "override state directory")
	_ = cmd.PersistentFlags().MarkHidden("state-dir")

	cmd.AddCommand(
		rc.newMaterializeCmd(),
		rc.newRestoreCmd(),
		rc.newClearCmd(),
		rc.newDiffCmd(),
		rc.newStatusCmd(),
	)

	return cmd
}

func absPath(arg string) (string, error) {
	path, err := filepath.Abs(arg)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	return path, nil
}

func (rc *rootCmd) getTrackedEntry(path string) (state.Entry, error) {
	entry, found, err := rc.store.Get(path)
	if err != nil {
		return state.Entry{}, err
	}
	if !found {
		return state.Entry{}, fmt.Errorf("path not tracked: %s", path)
	}
	return entry, nil
}
