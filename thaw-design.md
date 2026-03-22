# thaw: Design Document

A CLI tool for temporarily materializing immutable symlinked config files so
that applications can write to them, then diffing the result against the
original.

## Problem

Dotfile managers like Home Manager, GNU Stow, and others create symlinks from
locations in `~` to read-only sources (e.g. `/nix/store`, a git-tracked
dotfiles directory). When an application tries to modify one of these files,
it fails silently or with a confusing error. The current workaround is manual:

1. Identify which symlink the app wants to write to (sometimes non-obvious).
2. Replace the symlink with a mutable copy of the file.
3. Let the app make its changes.
4. Diff the modified file against the original to figure out what changed.
5. Port the diff back into the dotfile manager's source of truth.
6. Either re-apply the dotfile manager (which overwrites the file) or manually
   restore the symlink.

This tool automates steps 2, 4, and 6.

## Workflows

### Flow A: Discover a config change and feed it back

```
thaw materialize ~/.config/foo/config.toml
# app writes to the file
thaw diff ~/.config/foo/config.toml
# user updates their dotfile source based on the diff
# user re-applies their dotfile manager (overwrites the file with a new symlink)
thaw untrack ~/.config/foo/config.toml
```

### Flow B: Temporarily materialize, then restore

```
thaw materialize ~/.config/foo/config.toml
# user pokes around, app writes to the file
thaw restore ~/.config/foo/config.toml
# symlink is put back, state is cleared in one step
```

## Commands

### `thaw materialize <path>`

1. Validate that `<path>` is a symlink.
2. Record the symlink target in the state directory.
3. Replace the symlink with a copy of the target file.
4. Confirm to the user.

Errors if the path is already materialized (i.e. already tracked in state).

### `thaw diff <path>`

1. Look up the original symlink target from state.
2. Run a diff between the original target and the current file at `<path>`.
3. Print the diff to stdout.

Exit code follows diff conventions: 0 = no changes, 1 = changes found.

Supports `--tool` to select the diff program (default: `diff -u`; could also
use `delta`, `difft`, etc.). The `THAW_DIFF` environment variable serves as
a fallback when `--tool` is not specified.

### `thaw restore <path>`

1. Look up the original symlink target from state.
2. Remove the file at `<path>`.
3. Re-create the symlink: `<path>` -> original target.
4. Remove the state entry.

### `thaw untrack <path>`

Remove the state entry for `<path>` without touching the filesystem. Used after
the dotfile manager has already replaced the file with a fresh symlink.

### `thaw status`

List all currently tracked (materialized) files with their original symlink
targets and materialization timestamps.

## State

### Location

`$XDG_STATE_HOME/thaw/` (defaults to `~/.local/state/thaw/`).

### Format

A single JSON file: `state.json`.

```json
{
  "version": 1,
  "entries": {
    "/Users/alice/.config/foo/config.toml": {
      "target": "/nix/store/abc123-foo-config/config.toml",
      "materialized_at": "2026-03-18T10:30:00Z"
    }
  }
}
```

The key is the absolute path of the file. The value contains:

- **`target`**: The original symlink target. Used by `diff` and `restore`.
- **`materialized_at`**: ISO 8601 timestamp. For display in `status` and
  potential future staleness warnings.

A flat JSON file is sufficient — the expected number of concurrently
materialized files is very small (single digits). No need for SQLite or
per-file state directories.

### File locking

Use a lockfile (`state.lock`) with `flock` semantics to prevent concurrent
mutations. Cheap to implement and prevents corruption if two terminals run
commands simultaneously.

## Technology

- **Language**: Go
- **CLI framework**: Cobra
- **UX/output**: Charmbracelet (lipgloss for styled output, likely nothing
  heavier)
- **Diff**: Shell out to an external diff tool by default. The tool does not
  need to implement its own differ.

## Future ideas (out of scope for v1)

- **`thaw watch <path>`**: Materialize, then watch for filesystem changes
  and print diffs automatically.
- **`thaw trace <command>`**: Run a command under `fs_usage` (macOS) or
  `strace` (Linux) and detect which symlinks it failed to write to.
  Offer to materialize them.
- **Source mapping**: Given a diff, grep a dotfile repo to suggest which
  source file likely generated the managed config.
- **`thaw untrack --all`**: Clear all entries whose paths are already symlinks
  again (i.e. the dotfile manager has already taken them back).
