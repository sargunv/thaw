// Package state manages thaw's persistent state file, stored as JSON
// at $XDG_STATE_HOME/thaw/state.json with a lockfile for concurrent access.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/gofrs/flock"
	"github.com/google/renameio/v2"
)

const (
	stateVersion  = 1
	stateFileName = "state.json"
	lockFileName  = "state.lock"
)

var (
	// ErrAlreadyMaterialized is returned when trying to materialize a path that is already tracked.
	ErrAlreadyMaterialized = errors.New("path already materialized")
	// ErrNotTracked is returned when operating on a path that is not tracked.
	ErrNotTracked = errors.New("path not tracked")
)

// Entry represents a single materialized file.
type Entry struct {
	Target         string    `json:"target"`
	RawTarget      string    `json:"raw_target,omitempty"`
	MaterializedAt time.Time `json:"materialized_at"`
}

// stateFile is the on-disk JSON format.
type stateFile struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
}

// Store manages thaw's persistent state.
type Store struct {
	dir string
}

// DefaultDir returns the default state directory using XDG conventions.
func DefaultDir() string {
	return filepath.Join(xdg.StateHome, "thaw")
}

// NewStore creates a Store backed by the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// ensureDir creates the state directory if it doesn't exist.
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.dir, 0o700)
}

func (s *Store) statePath() string {
	return filepath.Join(s.dir, stateFileName)
}

func (s *Store) lockPath() string {
	return filepath.Join(s.dir, lockFileName)
}

// load reads the state file. Returns an empty state if the file doesn't exist.
func (s *Store) load() (*stateFile, error) {
	data, err := os.ReadFile(s.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return &stateFile{Version: stateVersion, Entries: make(map[string]Entry)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	if sf.Version != stateVersion {
		return nil, fmt.Errorf("unsupported state version %d (expected %d); the state file may have been written by a newer version of thaw", sf.Version, stateVersion)
	}
	if sf.Entries == nil {
		sf.Entries = make(map[string]Entry)
	}
	return &sf, nil
}

// save writes the state file atomically.
func (s *Store) save(sf *stateFile) error {
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	data = append(data, '\n')
	if err := renameio.WriteFile(s.statePath(), data, 0o600); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	return nil
}

// withLock runs fn while holding an exclusive lock on the lockfile.
func (s *Store) withLock(fn func() error) error {
	return s.withFlock(false, fn)
}

// withRLock runs fn while holding a shared lock on the lockfile.
func (s *Store) withRLock(fn func() error) error {
	return s.withFlock(true, fn)
}

func (s *Store) withFlock(shared bool, fn func() error) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	fl := flock.New(s.lockPath())
	if shared {
		if err := fl.RLock(); err != nil {
			return fmt.Errorf("acquiring state lock: %w", err)
		}
	} else {
		if err := fl.Lock(); err != nil {
			return fmt.Errorf("acquiring state lock: %w", err)
		}
	}
	defer func() { _ = fl.Unlock() }()

	return fn()
}

// Get returns the entry for the given path. The bool is false if the path is not tracked.
func (s *Store) Get(path string) (Entry, bool, error) {
	var entry Entry
	var found bool
	err := s.withRLock(func() error {
		sf, err := s.load()
		if err != nil {
			return err
		}
		entry, found = sf.Entries[path]
		return nil
	})
	return entry, found, err
}

// Add records a new entry. Returns an error if the path is already tracked.
func (s *Store) Add(path string, target string, rawTarget string, materializedAt time.Time) error {
	return s.withLock(func() error {
		sf, err := s.load()
		if err != nil {
			return err
		}
		if _, exists := sf.Entries[path]; exists {
			return fmt.Errorf("%s: %w", path, ErrAlreadyMaterialized)
		}
		sf.Entries[path] = Entry{
			Target:         target,
			RawTarget:      rawTarget,
			MaterializedAt: materializedAt,
		}
		return s.save(sf)
	})
}

// Remove deletes the entry for the given path. Returns an error if the path is not tracked.
func (s *Store) Remove(path string) error {
	return s.withLock(func() error {
		sf, err := s.load()
		if err != nil {
			return err
		}
		if _, exists := sf.Entries[path]; !exists {
			return fmt.Errorf("%s: %w", path, ErrNotTracked)
		}
		delete(sf.Entries, path)
		return s.save(sf)
	})
}

// List returns all tracked entries.
func (s *Store) List() (map[string]Entry, error) {
	var entries map[string]Entry
	err := s.withRLock(func() error {
		sf, err := s.load()
		if err != nil {
			return err
		}
		entries = sf.Entries
		return nil
	})
	return entries, err
}
