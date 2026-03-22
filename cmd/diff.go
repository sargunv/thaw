package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func (rc *rootCmd) newDiffCmd() *cobra.Command {
	var tool string

	diffCmd := &cobra.Command{
		Use:   "diff <path>",
		Short: "Diff a materialized file against its original symlink target",
		Example: `  thaw diff ~/.config/foo/config.toml
  thaw diff --tool delta ~/.config/foo/config.toml
  THAW_DIFF=delta thaw diff ~/.config/foo/config.toml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := absPath(args[0])
			if err != nil {
				return err
			}

			entry, err := rc.getTrackedEntry(path)
			if err != nil {
				return err
			}

			toolStr := cmp.Or(tool, os.Getenv("THAW_DIFF"), "diff -u")
			parts := strings.Fields(toolStr)
			if len(parts) == 0 {
				return fmt.Errorf("diff tool is set but empty")
			}

			diffArgs := append(parts[1:len(parts):len(parts)], entry.Target, path)

			diffCmd := exec.CommandContext(cmd.Context(), parts[0], diffArgs...)
			diffCmd.Stdout = cmd.OutOrStdout()
			diffCmd.Stderr = cmd.ErrOrStderr()

			if err := diffCmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					code := exitErr.ExitCode()
					switch code {
					case 1:
						return &ExitError{Code: 1}
					case -1:
						return fmt.Errorf("%s killed by signal", parts[0])
					default:
						return fmt.Errorf("%s exited with code %d", parts[0], code)
					}
				}
				return fmt.Errorf("running %s: %w", parts[0], err)
			}
			return nil
		},
	}

	diffCmd.Flags().StringVar(&tool, "tool", "", `diff program (default: $THAW_DIFF or "diff -u")`)

	return diffCmd
}
