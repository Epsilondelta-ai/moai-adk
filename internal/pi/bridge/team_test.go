package bridge

import (
	"context"
	"testing"

	"github.com/modu-ai/moai-adk/internal/teamruntime"
)

func TestHandleTeamRunCreateStatusMessageDelete(t *testing.T) {
	cwd := t.TempDir()
	h := NewHandler()
	create := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_team_run", "action": "create", "payload": map[string]any{"tasks": []any{map[string]any{"role": "implementer", "agent": "expert-backend", "task": "build", "ownedPatterns": []any{"internal/**"}}}}}})
	if !create.OK {
		t.Fatalf("create failed: %#v", create)
	}
	team := create.Data["team"].(*teamruntime.TeamState)
	teamID := team.ID
	status := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_team_run", "action": "status", "payload": map[string]any{"teamId": teamID}}})
	if !status.OK {
		t.Fatalf("status failed: %#v", status)
	}
	msg := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_team_run", "action": "message", "payload": map[string]any{"teamId": teamID, "message": "continue"}}})
	if !msg.OK {
		t.Fatalf("message failed: %#v", msg)
	}
	idle := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_team_run", "action": "idle", "payload": map[string]any{"teamId": teamID, "role": "implementer"}}})
	if !idle.OK {
		t.Fatalf("idle failed: %#v", idle)
	}
	review := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_team_run", "action": "review", "payload": map[string]any{"teamId": teamID, "role": "implementer", "accepted": true}}})
	if !review.OK {
		t.Fatalf("review failed: %#v", review)
	}
	deleted := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_team_run", "action": "delete", "payload": map[string]any{"teamId": teamID}}})
	if !deleted.OK {
		t.Fatalf("delete failed: %#v", deleted)
	}
}
