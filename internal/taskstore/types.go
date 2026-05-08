// Package taskstore provides durable MoAI task state for runtimes that do not
// provide Claude Code TaskCreate/TaskUpdate/TaskList/TaskGet tools.
package taskstore

import "time"

// TaskStatus is the lifecycle state of a MoAI task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Task is a durable project-level task record.
type Task struct {
	ID          string         `json:"id"`
	Subject     string         `json:"subject"`
	Description string         `json:"description,omitempty"`
	Status      TaskStatus     `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	Version     int            `json:"version"`
}

// Patch updates selected fields on an existing task.
type Patch struct {
	Subject     *string        `json:"subject,omitempty"`
	Description *string        `json:"description,omitempty"`
	Status      *TaskStatus    `json:"status,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// FileData is the on-disk task store format.
type FileData struct {
	Version int     `json:"version"`
	Tasks   []*Task `json:"tasks"`
}
