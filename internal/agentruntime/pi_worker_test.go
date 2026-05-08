package agentruntime

import (
	"strings"
	"testing"
)

func TestNormalizePiTools(t *testing.T) {
	got := normalizePiTools([]string{"Read", "Write", "Edit", "MultiEdit", "Grep", "Glob", "AskUserQuestion", "TaskCreate", "TodoWrite", "WebSearch", "Skill", "mcp__x"})
	joined := strings.Join(got, ",")
	if joined != "read,write,edit,grep,find,moai_ask_user,moai_task_create,moai_task_update" {
		t.Fatalf("normalizePiTools() = %q", joined)
	}
}

func TestClassifyAgentErrorMarksProviderFailuresAsBlockers(t *testing.T) {
	result := &Result{}
	classifyAgentError(result, "API key not found for provider")
	if result.Status != StatusBlocked || result.ErrorClass != "configuration" || result.Recovery == "" {
		t.Fatalf("unexpected classification: %#v", result)
	}
}

func TestConsumePiEventExtractsAssistantOutput(t *testing.T) {
	result := &Result{}
	consumePiEvent(map[string]any{
		"message": map[string]any{
			"role":  "assistant",
			"model": "test-model",
			"content": []any{
				map[string]any{"type": "text", "text": "hello\nworld"},
			},
			"usage": map[string]any{
				"input":       float64(10),
				"output":      float64(5),
				"cacheRead":   float64(2),
				"cacheWrite":  float64(1),
				"totalTokens": float64(15),
				"cost":        map[string]any{"total": float64(0.01)},
			},
		},
	}, result)
	if result.Output != "hello\nworld" || result.Summary != "hello" || result.Model != "test-model" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Usage.Input != 10 || result.Usage.Cost != 0.01 || result.Usage.Turns != 1 {
		t.Fatalf("unexpected usage: %#v", result.Usage)
	}
}
