// Package teamruntime coordinates MoAI team executions on runtimes without native Agent Teams.
package teamruntime

import "github.com/modu-ai/moai-adk/internal/agentruntime"

// RoleProfile describes a MoAI teammate role.
type RoleProfile struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Model       string `json:"model"`
	Isolation   string `json:"isolation"`
	Description string `json:"description"`
}

// Task is a unit assigned to a teammate role.
type Task struct {
	Role          string   `json:"role"`
	Agent         string   `json:"agent"`
	Task          string   `json:"task"`
	OwnedPatterns []string `json:"ownedPatterns,omitempty"`
}

// Result is the aggregate outcome of a team run.
type Result struct {
	TeamID  string                 `json:"teamId,omitempty"`
	Tasks   []Task                 `json:"tasks"`
	Results []*agentruntime.Result `json:"results"`
}
