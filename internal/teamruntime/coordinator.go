package teamruntime

import (
	"context"
	"fmt"

	"github.com/modu-ai/moai-adk/internal/agentruntime"
)

// Coordinator maps team tasks to agent runtime requests.
type Coordinator struct {
	AgentCoordinator *agentruntime.Coordinator
	Profiles         map[string]RoleProfile
}

// NewCoordinator creates a team coordinator.
func NewCoordinator(agentCoordinator *agentruntime.Coordinator, profiles map[string]RoleProfile) *Coordinator {
	if agentCoordinator == nil {
		agentCoordinator = agentruntime.NewCoordinator(agentruntime.NewPiWorker())
	}
	return &Coordinator{AgentCoordinator: agentCoordinator, Profiles: profiles}
}

// Run executes team tasks in parallel, applying role profile isolation policy.
func (c *Coordinator) Run(ctx context.Context, tasks []Task, cwd string) (*Result, error) {
	requests := make([]agentruntime.Request, 0, len(tasks))
	for _, task := range tasks {
		if task.Agent == "" {
			return nil, fmt.Errorf("task for role %q missing agent", task.Role)
		}
		profile := c.Profiles[task.Role]
		request := agentruntime.Request{Agent: task.Agent, Task: task.Task, CWD: cwd, Model: profile.Model}
		if normalizedIsolation(task.Role, profile) == "worktree" {
			request.Worktree.Enabled = true
			request.Worktree.Keep = true
		}
		requests = append(requests, request)
	}
	results, err := c.AgentCoordinator.InvokeParallel(ctx, requests)
	if err != nil {
		return nil, err
	}
	return &Result{Tasks: tasks, Results: results}, nil
}
