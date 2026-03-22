package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <path>",
		Short: "Diff a materialized file against its original symlink target",
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

			toolStr := cmp.Or(os.Getenv("THAW_DIFF"), "diff -u")
			parts := strings.Fields(toolStr)
			if len(parts) == 0 {
				return fmt.Errorf("THAW_DIFF is set but empty")
			}

			diffArgs := make([]string, 0, len(parts)-1+2)
			diffArgs = append(diffArgs, parts[1:]...)
			diffArgs = append(diffArgs, entry.Target, path)

			diffCmd := exec.CommandContext(cmd.Context(), parts[0], diffArgs...)
			diffCmd.Stdout = cmd.OutOrStdout()
			diffCmd.Stderr = cmd.ErrOrStderr()

			if err := diffCmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					code := exitErr.ExitCode()
					if code == 1 {
						return &ExitError{Code: 1}
					}
					return fmt.Errorf("diff tool exited with code %d", code)
				}
				return fmt.Errorf("running diff tool: %w", err)
			}
			return nil
		},
	}
}
