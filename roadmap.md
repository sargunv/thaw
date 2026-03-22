# thaw: Implementation Roadmap

## PR 1: Project scaffold + state management

- Go module init, Cobra root command, basic project structure
- State file read/write/lock (`state.json` with `state.lock`)
- XDG path resolution (`$XDG_STATE_HOME/thaw/`)
- Unit tests for state operations

## PR 2: `materialize` + `restore` + `clear`

- `thaw materialize <path>` — validate symlink, record state, copy file
- `thaw restore <path>` — re-create symlink, remove state entry
- `thaw untrack <path>` — remove state entry only
- Integration tests (create temp symlinks, run commands, assert filesystem
  state)

## PR 3: `diff` + `status`

- `thaw diff <path>` — look up original target, shell out to diff tool, exit
  code convention (0 = no changes, 1 = changes found)
- `--tool` flag and `THAW_DIFF` env var for selecting diff program
- `thaw status` — list tracked files with targets and timestamps
- Tests

## PR 4: Polish

- Lipgloss styled output (success/error messages, status table)
- Error message UX (clear messages for common mistakes like "not a symlink",
  "already materialized")
- CLI help text and usage examples
