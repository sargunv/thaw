# thaw

Materialize immutable symlinked config files as mutable copies, then diff the changes or restore the
original symlink.

## Problem

Dotfile managers like Home Manager manage files as read-only symlinks. When an application tries to
modify one of these files, it fails. `thaw` temporarily replaces the symlink with a mutable copy so
the app can write to it. Once it has, you diff against the original, apply the changes back to your
dotfile source, and restore the symlink.

## Install

### Pre-built binaries

Download from [GitHub Releases](https://github.com/sargunv/thaw/releases) (Linux and macOS).

### Go

```
go install github.com/sargunv/thaw@latest
```

### Nix

```
nix profile install github:sargunv/thaw
```

## Usage

### Capture and review a config change

```
thaw materialize ~/.config/foo/config.toml
# app writes to the file
thaw diff ~/.config/foo/config.toml
# port the changes into your dotfile source, then re-apply your dotfile manager
thaw untrack ~/.config/foo/config.toml
```

### Materialize, then restore

```
thaw materialize ~/.config/foo/config.toml
# app writes to the file
thaw restore ~/.config/foo/config.toml
```

State is stored at `$XDG_STATE_HOME/thaw/` (defaults to `~/.local/state/thaw/`).
