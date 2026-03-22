package cmd_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sargunv/thaw/cmd"
	"github.com/sargunv/thaw/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCmd(t *testing.T, stateDir string, args ...string) error {
	t.Helper()
	root := cmd.New()
	root.SetArgs(append([]string{"--state-dir", stateDir}, args...))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root.Execute()
}

func runCmdWithOutput(t *testing.T, stateDir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := cmd.New()
	root.SetArgs(append([]string{"--state-dir", stateDir}, args...))
	root.SetOut(&buf)
	root.SetErr(io.Discard)
	err := root.Execute()
	return buf.String(), err
}

// setupSymlink creates a target file with content and a symlink pointing to it.
// Returns the absolute paths of the symlink and target.
func setupSymlink(t *testing.T, dir string, content string, mode os.FileMode) (link, target string) {
	t.Helper()

	target = filepath.Join(dir, "store", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte(content), mode))

	link = filepath.Join(dir, "home", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(target, link))

	return link, target
}

func TestMaterialize(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, target := setupSymlink(t, fsDir, "key = \"value\"\n", 0o644)

	err := runCmd(t, stateDir, "materialize", link)
	require.NoError(t, err)

	// Should be a regular file now, not a symlink
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.True(t, fi.Mode().IsRegular(), "expected regular file, got %s", fi.Mode())

	// Content should match
	content, err := os.ReadFile(link)
	require.NoError(t, err)
	assert.Equal(t, "key = \"value\"\n", string(content))

	// Permissions should match
	assert.Equal(t, os.FileMode(0o644), fi.Mode().Perm())

	// State should have an entry
	store := state.NewStore(stateDir)
	entry, found, err := store.Get(link)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, target, entry.Target)
}

func TestMaterializeAlreadyTracked(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, target := setupSymlink(t, fsDir, "content", 0o644)

	err := runCmd(t, stateDir, "materialize", link)
	require.NoError(t, err)

	// Replace the regular file with a symlink again to test the state check
	require.NoError(t, os.Remove(link))
	require.NoError(t, os.Symlink(target, link))

	err = runCmd(t, stateDir, "materialize", link)
	assert.ErrorContains(t, err, "already materialized")
}

func TestMaterializeNotASymlink(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	regularFile := filepath.Join(fsDir, "regular.toml")
	require.NoError(t, os.WriteFile(regularFile, []byte("content"), 0o644))

	err := runCmd(t, stateDir, "materialize", regularFile)
	assert.ErrorContains(t, err, "not a symlink")
}

func TestMaterializeSymlinkToDirectory(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()

	targetDir := filepath.Join(fsDir, "store", "configdir")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	link := filepath.Join(fsDir, "home", "configdir")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(targetDir, link))

	err := runCmd(t, stateDir, "materialize", link)
	assert.ErrorContains(t, err, "not a regular file")
}

func TestMaterializeRelativeSymlink(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()

	// Create target file
	target := filepath.Join(fsDir, "store", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("content"), 0o644))

	// Create symlink with a relative target
	link := filepath.Join(fsDir, "home", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	relTarget, err := filepath.Rel(filepath.Dir(link), target)
	require.NoError(t, err)
	require.NoError(t, os.Symlink(relTarget, link))

	err = runCmd(t, stateDir, "materialize", link)
	require.NoError(t, err)

	// State should store the absolute target path
	store := state.NewStore(stateDir)
	entry, found, err := store.Get(link)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, target, entry.Target)

	// Restore should work with the resolved absolute path
	err = runCmd(t, stateDir, "restore", link)
	require.NoError(t, err)

	got, err := os.Readlink(link)
	require.NoError(t, err)
	assert.Equal(t, target, got)
}

func TestMaterializePreservesPermissions(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, _ := setupSymlink(t, fsDir, "content", 0o755)

	err := runCmd(t, stateDir, "materialize", link)
	require.NoError(t, err)

	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), fi.Mode().Perm())
}

func TestRestore(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, target := setupSymlink(t, fsDir, "content", 0o644)

	// Materialize first
	err := runCmd(t, stateDir, "materialize", link)
	require.NoError(t, err)

	// Restore
	err = runCmd(t, stateDir, "restore", link)
	require.NoError(t, err)

	// Should be a symlink again
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.True(t, fi.Mode()&os.ModeSymlink != 0, "expected symlink")

	// Should point to original target
	got, err := os.Readlink(link)
	require.NoError(t, err)
	assert.Equal(t, target, got)

	// State should be cleared
	store := state.NewStore(stateDir)
	_, found, err := store.Get(link)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRestoreNotTracked(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()

	err := runCmd(t, stateDir, "restore", filepath.Join(fsDir, "nonexistent"))
	assert.ErrorContains(t, err, "not tracked")
}

func TestClear(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, _ := setupSymlink(t, fsDir, "content", 0o644)

	// Materialize first
	err := runCmd(t, stateDir, "materialize", link)
	require.NoError(t, err)

	// Clear
	err = runCmd(t, stateDir, "clear", link)
	require.NoError(t, err)

	// State should be cleared
	store := state.NewStore(stateDir)
	_, found, err := store.Get(link)
	require.NoError(t, err)
	assert.False(t, found)

	// File should still exist as a regular file (clear doesn't touch filesystem)
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.True(t, fi.Mode().IsRegular())
}

func TestClearNotTracked(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()

	err := runCmd(t, stateDir, "clear", filepath.Join(fsDir, "nonexistent"))
	assert.ErrorContains(t, err, "not tracked")
}

func TestDiffNoChanges(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, _ := setupSymlink(t, fsDir, "key = \"value\"\n", 0o644)

	require.NoError(t, runCmd(t, stateDir, "materialize", link))

	err := runCmd(t, stateDir, "diff", link)
	assert.NoError(t, err)
}

func TestDiffWithChanges(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, _ := setupSymlink(t, fsDir, "original\n", 0o644)

	require.NoError(t, runCmd(t, stateDir, "materialize", link))
	require.NoError(t, os.WriteFile(link, []byte("modified\n"), 0o644))

	err := runCmd(t, stateDir, "diff", link)
	var exitErr *cmd.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.Code)
}

func TestDiffNotTracked(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()

	err := runCmd(t, stateDir, "diff", filepath.Join(fsDir, "nonexistent"))
	assert.ErrorContains(t, err, "not tracked")
}

func TestDiffToolNotFound(t *testing.T) {
	t.Setenv("THAW_DIFF", "thaw-nonexistent-tool-xyz")
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, _ := setupSymlink(t, fsDir, "content\n", 0o644)

	require.NoError(t, runCmd(t, stateDir, "materialize", link))

	err := runCmd(t, stateDir, "diff", link)
	assert.ErrorContains(t, err, "running diff tool")
}

func TestDiffToolExitError(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, _ := setupSymlink(t, fsDir, "content\n", 0o644)

	require.NoError(t, runCmd(t, stateDir, "materialize", link))

	// Delete the original target so diff gets a nonexistent path, causing exit code 2
	require.NoError(t, os.Remove(filepath.Join(fsDir, "store", "config.toml")))

	err := runCmd(t, stateDir, "diff", link)
	assert.ErrorContains(t, err, "diff tool exited with code 2")
	var exitErr *cmd.ExitError
	assert.False(t, errors.As(err, &exitErr))
}

func TestStatusEmpty(t *testing.T) {
	stateDir := t.TempDir()

	output, err := runCmdWithOutput(t, stateDir, "status")
	require.NoError(t, err)
	assert.Equal(t, "No materialized files\n", output)
}

func TestStatus(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()
	link, target := setupSymlink(t, fsDir, "content", 0o644)

	require.NoError(t, runCmd(t, stateDir, "materialize", link))

	output, err := runCmdWithOutput(t, stateDir, "status")
	require.NoError(t, err)
	assert.Contains(t, output, fmt.Sprintf("%s -> %s", link, target))
}

func TestStatusSorted(t *testing.T) {
	stateDir := t.TempDir()
	fsDir := t.TempDir()

	// Create two symlinks in the same dir with predictable names
	aTarget := filepath.Join(fsDir, "store", "a.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(aTarget), 0o755))
	require.NoError(t, os.WriteFile(aTarget, []byte("a"), 0o644))
	aLink := filepath.Join(fsDir, "home", "a.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(aLink), 0o755))
	require.NoError(t, os.Symlink(aTarget, aLink))

	bTarget := filepath.Join(fsDir, "store", "b.toml")
	require.NoError(t, os.WriteFile(bTarget, []byte("b"), 0o644))
	bLink := filepath.Join(fsDir, "home", "b.toml")
	require.NoError(t, os.Symlink(bTarget, bLink))

	// Materialize b first, then a — output should still be sorted
	require.NoError(t, runCmd(t, stateDir, "materialize", bLink))
	require.NoError(t, runCmd(t, stateDir, "materialize", aLink))

	output, err := runCmdWithOutput(t, stateDir, "status")
	require.NoError(t, err)

	aIdx := strings.Index(output, aLink)
	bIdx := strings.Index(output, bLink)
	assert.Greater(t, aIdx, -1, "expected output to contain %s", aLink)
	assert.Greater(t, bIdx, -1, "expected output to contain %s", bLink)
	assert.Less(t, aIdx, bIdx, "expected %s to appear before %s", aLink, bLink)
}
