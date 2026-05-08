package agentruntime

import (
	"context"
	"time"
)

// Status is the final state of an agent invocation.
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusBlocked Status = "blocked"
	StatusAborted Status = "aborted"
)

// Request describes one agent invocation.
type Request struct {
	Agent          string          `json:"agent"`
	Task           string          `json:"task"`
	CWD            string          `json:"cwd,omitempty"`
	Tools          []string        `json:"tools,omitempty"`
	Model          string          `json:"model,omitempty"`
	Timeout        time.Duration   `json:"-"`
	TimeoutSeconds int             `json:"timeoutSeconds,omitempty"`
	Worktree       WorktreeOptions `json:"worktree,omitempty"`
}

// Usage captures model usage emitted by Pi JSON mode.
type Usage struct {
	Input         int     `json:"input"`
	Output        int     `json:"output"`
	CacheRead     int     `json:"cacheRead"`
	CacheWrite    int     `json:"cacheWrite"`
	Cost          float64 `json:"cost"`
	ContextTokens int     `json:"contextTokens"`
	Turns         int     `json:"turns"`
}

// Result is the structured output of an agent invocation.
type Result struct {
	Agent         string           `json:"agent"`
	ResolvedAgent string           `json:"resolvedAgent,omitempty"`
	Status        Status           `json:"status"`
	Summary       string           `json:"summary,omitempty"`
	Output        string           `json:"output,omitempty"`
	Error         string           `json:"error,omitempty"`
	ErrorClass    string           `json:"errorClass,omitempty"`
	Recovery      string           `json:"recovery,omitempty"`
	ExitCode      int              `json:"exitCode"`
	Usage         Usage            `json:"usage"`
	Model         string           `json:"model,omitempty"`
	Messages      []map[string]any `json:"messages,omitempty"`
	StartedAt     time.Time        `json:"startedAt"`
	CompletedAt   time.Time        `json:"completedAt"`
	WorktreePath  string           `json:"worktreePath,omitempty"`
	Branch        string           `json:"branch,omitempty"`
	WorktreeKept  bool             `json:"worktreeKept,omitempty"`
}

// Runtime invokes agents.
type Runtime interface {
	Invoke(ctx context.Context, request Request) (*Result, error)
}
