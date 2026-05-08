package bridge

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modu-ai/moai-adk/internal/policy"
	"github.com/modu-ai/moai-adk/internal/sessionstate"
)

func TestHandleCommandExecutesKernel(t *testing.T) {
	root := t.TempDir()
	res := NewHandler().Handle(context.Background(), Request{
		Kind: "command",
		CWD:  root,
		Payload: map[string]any{
			"command": "plan",
			"args":    "build pi adapter",
		},
	})
	if !res.OK {
		t.Fatalf("Handle() error = %#v", res.Error)
	}
	if res.Data["commandResult"] == nil || res.Data["uiState"] == nil {
		t.Fatalf("kernel result missing: %#v", res.Data)
	}
}

func TestHandleCapabilitiesReportsNativeAndUnsupportedEvents(t *testing.T) {
	res := NewHandler().Handle(context.Background(), Request{Kind: "capabilities", CWD: t.TempDir()})
	if !res.OK {
		t.Fatalf("capabilities failed: %#v", res)
	}
	events := stringsFromAnyInterface(res.Data["events"])
	for _, want := range []string{"agent_start", "turn_start", "session_before_compact", "after_provider_response"} {
		if !containsString(events, want) {
			t.Fatalf("events missing %q: %#v", want, events)
		}
	}
	support := res.Data["eventSupport"].(map[string]any)
	unsupported := stringsFromAnyInterface(support["unsupported"])
	if len(unsupported) != 0 {
		t.Fatalf("unexpected unsupported hook list: %#v", unsupported)
	}
	synthetic := stringsFromAnyInterface(support["synthetic"])
	if !containsString(synthetic, "FileChanged") || !containsString(synthetic, "TeammateIdle") {
		t.Fatalf("synthetic hook list incomplete: %#v", synthetic)
	}
}

func TestHandleEventPersistsLifecycleState(t *testing.T) {
	cwd := t.TempDir()
	handler := NewHandler()
	for _, event := range []Request{
		{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "agent_start"}},
		{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "turn_start", "turnIndex": float64(2)}},
		{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "tool_result", "toolName": "bash", "isError": true}},
		{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "session_before_compact"}},
		{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "session_compact"}},
		{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "agent_end"}},
	} {
		res := handler.Handle(context.Background(), event)
		if !res.OK {
			t.Fatalf("event failed: %#v", res)
		}
	}
	state, err := sessionstate.New(cwd).Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.LastEvent != "agent_end" {
		t.Fatalf("unexpected last event: %#v", state)
	}
	if state.Data["agentActive"] != false || state.Data["lastToolName"] != "bash" || state.Data["lastToolError"] != true {
		t.Fatalf("lifecycle state not updated: %#v", state.Data)
	}
	if state.Data["compactionPending"] != false || state.Data["lastCompactedAt"] == nil {
		t.Fatalf("compaction state not updated: %#v", state.Data)
	}
	counts := state.Data["eventCounts"].(map[string]any)
	if counts["agent_start"].(float64) != 1 || counts["agent_end"].(float64) != 1 {
		t.Fatalf("event counts not persisted: %#v", counts)
	}
	recent := state.Data["recentEvents"].([]any)
	if len(recent) != 6 {
		t.Fatalf("recent events not persisted: %#v", recent)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".moai", "runtime", "pi-session.json")); err != nil {
		t.Fatalf("session state file missing: %v", err)
	}
}

func TestHandleSyntheticEventsPersistParityState(t *testing.T) {
	cwd := t.TempDir()
	res := NewHandler().Handle(context.Background(), Request{Kind: "event", CWD: cwd, Payload: map[string]any{"event": "FileChanged", "path": "x.go"}})
	if !res.OK {
		t.Fatalf("event failed: %#v", res)
	}
	state, err := sessionstate.New(cwd).Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Data["lastFileChange"] == nil {
		t.Fatalf("synthetic file event not persisted: %#v", state.Data)
	}
}

func TestHandleEventBlocksProtectedToolCall(t *testing.T) {
	res := NewHandler().Handle(context.Background(), Request{
		Kind: "event",
		CWD:  t.TempDir(),
		Payload: map[string]any{
			"event":    "tool_call",
			"toolName": "write",
			"input":    map[string]any{"path": ".env.local"},
		},
	})
	if !res.OK {
		t.Fatalf("event failed: %#v", res)
	}
	decision := res.Data["decision"].(policy.Decision)
	if !decision.Block {
		t.Fatalf("expected policy block: %#v", res.Data)
	}
}

func stringsFromAnyInterface(value any) []string {
	items, ok := value.([]string)
	if ok {
		return items
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
