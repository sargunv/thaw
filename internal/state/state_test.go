package state_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sargunv/thaw/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *state.Store {
	t.Helper()
	return state.NewStore(t.TempDir())
}

var testTime = time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC)

func TestAddAndGet(t *testing.T) {
	s := newTestStore(t)

	err := s.Add("/home/alice/.config/foo.toml", "/nix/store/abc123/foo.toml", testTime)
	require.NoError(t, err)

	entry, found, err := s.Get("/home/alice/.config/foo.toml")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "/nix/store/abc123/foo.toml", entry.Target)
	assert.Equal(t, testTime, entry.MaterializedAt)
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)

	_, found, err := s.Get("/nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestAddDuplicate(t *testing.T) {
	s := newTestStore(t)

	err := s.Add("/home/alice/.config/foo.toml", "/nix/store/abc123/foo.toml", testTime)
	require.NoError(t, err)

	err = s.Add("/home/alice/.config/foo.toml", "/nix/store/other/foo.toml", testTime)
	assert.ErrorContains(t, err, "already materialized")
}

func TestRemove(t *testing.T) {
	s := newTestStore(t)

	err := s.Add("/home/alice/.config/foo.toml", "/nix/store/abc123/foo.toml", testTime)
	require.NoError(t, err)

	err = s.Remove("/home/alice/.config/foo.toml")
	require.NoError(t, err)

	_, found, err := s.Get("/home/alice/.config/foo.toml")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRemoveNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.Remove("/nonexistent")
	assert.ErrorContains(t, err, "not tracked")
}

func TestList(t *testing.T) {
	s := newTestStore(t)

	entries, err := s.List()
	require.NoError(t, err)
	assert.Empty(t, entries)

	err = s.Add("/home/alice/.config/foo.toml", "/nix/store/abc/foo.toml", testTime)
	require.NoError(t, err)
	err = s.Add("/home/alice/.config/bar.toml", "/nix/store/def/bar.toml", testTime)
	require.NoError(t, err)

	entries, err = s.List()
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "/nix/store/abc/foo.toml", entries["/home/alice/.config/foo.toml"].Target)
	assert.Equal(t, "/nix/store/def/bar.toml", entries["/home/alice/.config/bar.toml"].Target)
}

func TestStateFileFormat(t *testing.T) {
	dir := t.TempDir()
	s := state.NewStore(dir)

	err := s.Add("/home/alice/.config/foo.toml", "/nix/store/abc123/foo.toml", testTime)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	var version int
	require.NoError(t, json.Unmarshal(raw["version"], &version))
	assert.Equal(t, 1, version)

	var entries map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["entries"], &entries))
	assert.Contains(t, entries, "/home/alice/.config/foo.toml")
}

func TestCorruptedStateFile(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("not json"), 0o644)
	require.NoError(t, err)

	s := state.NewStore(dir)

	_, _, err = s.Get("/anything")
	assert.ErrorContains(t, err, "parsing state file")

	err = s.Add("/foo", "/bar", testTime)
	assert.ErrorContains(t, err, "parsing state file")
}

func TestUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"version": 99, "entries": {}}`)
	err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o644)
	require.NoError(t, err)

	s := state.NewStore(dir)

	_, _, err = s.Get("/anything")
	assert.ErrorContains(t, err, "unsupported state version")
}

func TestAutoCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state", "dir")
	s := state.NewStore(dir)

	err := s.Add("/home/alice/.config/foo.toml", "/nix/store/abc/foo.toml", testTime)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "state.json"))
	assert.NoError(t, err)
}
