package policy

import "testing"

func TestEvaluatorBlocksDestructiveRM(t *testing.T) {
	result := NewEvaluator().Evaluate(EventRequest{Event: "tool_call", ToolName: "bash", ToolInput: map[string]any{"command": "rm -rf /"}})
	if !result.Decision.Block {
		t.Fatalf("expected block, got %#v", result)
	}
}

func TestEvaluatorBlocksProtectedPath(t *testing.T) {
	result := NewEvaluator().Evaluate(EventRequest{Event: "tool_call", ToolName: "write", ToolInput: map[string]any{"path": ".env"}})
	if !result.Decision.Block {
		t.Fatalf("expected block, got %#v", result)
	}
}

func TestEvaluatorAllowsReadOnlyCommand(t *testing.T) {
	result := NewEvaluator().Evaluate(EventRequest{Event: "tool_call", ToolName: "bash", ToolInput: map[string]any{"command": "git status"}})
	if result.Decision.Block {
		t.Fatalf("expected allow, got %#v", result)
	}
}
