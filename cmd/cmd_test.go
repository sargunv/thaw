package cmd_test

import (
	"io"
	"os"
	"path/filepath"
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
