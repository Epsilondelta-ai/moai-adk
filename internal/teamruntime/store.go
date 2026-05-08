package teamruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TeamState is durable team runtime state stored under .moai/state/teams.
type TeamState struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
	Tasks     []Task          `json:"tasks"`
	Members   []TeammateState `json:"members"`
	Messages  []TeamMessage   `json:"messages,omitempty"`
	Events    []TeamEvent     `json:"events,omitempty"`
	Results   []*AgentSummary `json:"results,omitempty"`
}

// TeammateState captures assigned role state and file ownership.
type TeammateState struct {
	Role          string   `json:"role"`
	Agent         string   `json:"agent"`
	Status        string   `json:"status"`
	Isolation     string   `json:"isolation"`
	OwnedPatterns []string `json:"ownedPatterns,omitempty"`
	WorktreePath  string   `json:"worktreePath,omitempty"`
	Branch        string   `json:"branch,omitempty"`
}

// TeamMessage is a persisted orchestration message.
type TeamMessage struct {
	From string `json:"from"`
	To   string `json:"to,omitempty"`
	Text string `json:"text"`
	At   string `json:"at"`
}

// TeamEvent records synthetic Agent Teams lifecycle events.
type TeamEvent struct {
	Type  string         `json:"type"`
	Role  string         `json:"role,omitempty"`
	Agent string         `json:"agent,omitempty"`
	At    string         `json:"at"`
	Data  map[string]any `json:"data,omitempty"`
}

// AgentSummary avoids importing agentruntime into the store schema tests.
type AgentSummary struct {
	Agent         string `json:"agent"`
	ResolvedAgent string `json:"resolvedAgent,omitempty"`
	Status        string `json:"status"`
	Summary       string `json:"summary,omitempty"`
	Error         string `json:"error,omitempty"`
	WorktreePath  string `json:"worktreePath,omitempty"`
	Branch        string `json:"branch,omitempty"`
}

// Store persists team state.
type Store struct{ dir string }

// NewStore creates a team store rooted at cwd.
func NewStore(cwd string) *Store { return &Store{dir: filepath.Join(cwd, ".moai", "state", "teams")} }

// Create initializes durable team state.
func (s *Store) Create(tasks []Task, profiles map[string]RoleProfile) (*TeamState, error) {
	if err := validateFileOwnership(tasks); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	state := &TeamState{ID: fmt.Sprintf("team-%d", time.Now().UTC().UnixNano()), Status: "created", CreatedAt: now, UpdatedAt: now, Tasks: tasks}
	for _, task := range tasks {
		profile := profiles[task.Role]
		state.Members = append(state.Members, TeammateState{Role: task.Role, Agent: task.Agent, Status: "assigned", Isolation: normalizedIsolation(task.Role, profile), OwnedPatterns: task.OwnedPatterns})
		state.Events = append(state.Events, TeamEvent{Type: "SubagentStart", Role: task.Role, Agent: task.Agent, At: now})
		if normalizedIsolation(task.Role, profile) == "worktree" {
			state.Events = append(state.Events, TeamEvent{Type: "WorktreeCreate", Role: task.Role, Agent: task.Agent, At: now})
		}
	}
	return state, s.Save(state)
}

// Save writes a team state.
func (s *Store) Save(state *TeamState) error {
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, state.ID+".json"), append(bytes, '\n'), 0o600)
}

// Get loads one team state.
func (s *Store) Get(id string) (*TeamState, error) {
	bytes, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return nil, err
	}
	var state TeamState
	if err := json.Unmarshal(bytes, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Latest returns the newest team state.
func (s *Store) Latest() (*TeamState, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var newest os.DirEntry
	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if newest == nil || info.ModTime().After(newestTime) {
			newest = entry
			newestTime = info.ModTime()
		}
	}
	if newest == nil {
		return nil, os.ErrNotExist
	}
	return s.Get(strings.TrimSuffix(newest.Name(), ".json"))
}

// RecordEvent appends a lifecycle event.
func (s *Store) RecordEvent(state *TeamState, event TeamEvent) error {
	if event.At == "" {
		event.At = time.Now().UTC().Format(time.RFC3339)
	}
	state.Events = append(state.Events, event)
	return s.Save(state)
}

// ReviewCompletion accepts or rejects teammate completion.
func (s *Store) ReviewCompletion(state *TeamState, role string, accepted bool, reason string) error {
	status := "rejected"
	eventType := "TaskCompletedRejected"
	if accepted {
		status = "accepted"
		eventType = "TaskCompleted"
	}
	for i := range state.Members {
		if state.Members[i].Role == role || state.Members[i].Agent == role {
			state.Members[i].Status = status
		}
	}
	state.Events = append(state.Events, TeamEvent{Type: eventType, Role: role, At: time.Now().UTC().Format(time.RFC3339), Data: map[string]any{"reason": reason}})
	return s.Save(state)
}

// Delete removes one team state.
func (s *Store) Delete(id string) error { return os.Remove(filepath.Join(s.dir, id+".json")) }

func normalizedIsolation(role string, profile RoleProfile) string {
	if profile.Isolation == "worktree" || role == "implementer" || role == "tester" || role == "designer" {
		return "worktree"
	}
	return "none"
}

func validateFileOwnership(tasks []Task) error {
	owners := map[string]string{}
	for _, task := range tasks {
		for _, pattern := range task.OwnedPatterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if owner, ok := owners[pattern]; ok && owner != task.Role {
				return fmt.Errorf("file ownership conflict for %q between %s and %s", pattern, owner, task.Role)
			}
			owners[pattern] = task.Role
		}
	}
	return nil
}
