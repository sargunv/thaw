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

// Entry represents a single materialized file.
type Entry struct {
	Target         string    `json:"target"`
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
	return os.MkdirAll(s.dir, 0o755)
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
		return nil, fmt.Errorf("unsupported state version %d (expected %d)", sf.Version, stateVersion)
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
	if err := renameio.WriteFile(s.statePath(), data, 0o644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	return nil
}

// withLock runs fn while holding an exclusive lock on the state lockfile.
func (s *Store) withLock(fn func() error) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	fl := flock.New(s.lockPath())
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("acquiring state lock: %w", err)
	}
	defer func() { _ = fl.Unlock() }()

	return fn()
}

// withRLock runs fn while holding a shared read lock on the state lockfile.
func (s *Store) withRLock(fn func() error) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	fl := flock.New(s.lockPath())
	if err := fl.RLock(); err != nil {
		return fmt.Errorf("acquiring state lock: %w", err)
	}
	defer func() { _ = fl.Unlock() }()

	return fn()
}

// Get returns the entry for the given path, or false if not found.
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
func (s *Store) Add(path string, target string, materializedAt time.Time) error {
	return s.withLock(func() error {
		sf, err := s.load()
		if err != nil {
			return err
		}
		if _, exists := sf.Entries[path]; exists {
			return fmt.Errorf("path already materialized: %s", path)
		}
		sf.Entries[path] = Entry{
			Target:         target,
			MaterializedAt: materializedAt,
		}
		return s.save(sf)
	})
}

// Remove deletes the entry for the given path. Returns an error if not found.
func (s *Store) Remove(path string) error {
	return s.withLock(func() error {
		sf, err := s.load()
		if err != nil {
			return err
		}
		if _, exists := sf.Entries[path]; !exists {
			return fmt.Errorf("path not tracked: %s", path)
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
		entries = make(map[string]Entry, len(sf.Entries))
		for k, v := range sf.Entries {
			entries[k] = v
		}
		return nil
	})
	return entries, err
}
