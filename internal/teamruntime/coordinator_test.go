package teamruntime

import (
	"context"
	"testing"

	"github.com/modu-ai/moai-adk/internal/agentruntime"
)

type fakeAgentRuntime struct{}

func (fakeAgentRuntime) Invoke(_ context.Context, request agentruntime.Request) (*agentruntime.Result, error) {
	status := agentruntime.StatusSuccess
	branch := ""
	if request.Worktree.Enabled {
		branch = "worktree-enabled"
	}
	return &agentruntime.Result{Agent: request.Agent, Status: status, Branch: branch}, nil
}

func TestCoordinatorRunAppliesWorktreeIsolation(t *testing.T) {
	coord := NewCoordinator(agentruntime.NewCoordinator(fakeAgentRuntime{}), map[string]RoleProfile{
		"implementer": {Name: "implementer", Isolation: "worktree", Model: "sonnet"},
	})
	result, err := coord.Run(context.Background(), []Task{{Role: "implementer", Agent: "expert-backend", Task: "do it"}}, t.TempDir())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].Branch != "worktree-enabled" {
		t.Fatalf("worktree isolation not applied: %#v", result.Results)
	}
}
