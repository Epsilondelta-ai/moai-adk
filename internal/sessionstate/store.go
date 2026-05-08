// Package sessionstate stores runtime session state across Pi resume/tree/compact flows.
package sessionstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State is persisted under .moai/runtime/pi-session.json.
type State struct {
	SessionFile  string         `json:"sessionFile,omitempty"`
	LeafID       string         `json:"leafId,omitempty"`
	Mode         string         `json:"mode,omitempty"`
	LastEvent    string         `json:"lastEvent,omitempty"`
	LastUpdated  string         `json:"lastUpdated"`
	ActiveSpecID string         `json:"activeSpecId,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
}

// Store persists state.
type Store struct{ path string }

// New creates a store rooted at cwd.
func New(cwd string) *Store {
	return &Store{path: filepath.Join(cwd, ".moai", "runtime", "pi-session.json")}
}

// Load reads state if present.
func (s *Store) Load() (*State, error) {
	bytes, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &State{LastUpdated: time.Now().UTC().Format(time.RFC3339), Data: map[string]any{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(bytes, &state); err != nil {
		return nil, err
	}
	if state.Data == nil {
		state.Data = map[string]any{}
	}
	return &state, nil
}

// Save writes state atomically.
func (s *Store) Save(state *State) error {
	if state.Data == nil {
		state.Data = map[string]any{}
	}
	state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(bytes, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
