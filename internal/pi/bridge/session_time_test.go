package bridge

import (
	"testing"
	"time"

	"github.com/modu-ai/moai-adk/internal/sessionstate"
)

func TestSessionStartedAtFromSessionFile(t *testing.T) {
	got := sessionStartedAtFromSessionFile("/tmp/2026-05-08T07-08-29-762Z_019e066a.jsonl")
	want := time.Date(2026, 5, 8, 7, 8, 29, 762*int(time.Millisecond), time.UTC).UnixMilli()
	if got != want {
		t.Fatalf("sessionStartedAtFromSessionFile() = %d, want %d", got, want)
	}
}

func TestEnsureSessionStartedAtDoesNotOverwrite(t *testing.T) {
	state := &sessionstate.State{SessionFile: "/tmp/2026-05-08T07-08-29-762Z_019e066a.jsonl", Data: map[string]any{"sessionStartedAt": float64(1234)}}
	ensureSessionStartedAt(state, "tool_call")
	if got := int64FromAny(state.Data["sessionStartedAt"]); got != 1234 {
		t.Fatalf("sessionStartedAt overwritten: got %d", got)
	}
}

func TestEnsureSessionStartedAtFromSessionFile(t *testing.T) {
	state := &sessionstate.State{SessionFile: "/tmp/2026-05-08T07-08-29-762Z_019e066a.jsonl", Data: map[string]any{}}
	ensureSessionStartedAt(state, "tool_call")
	want := time.Date(2026, 5, 8, 7, 8, 29, 762*int(time.Millisecond), time.UTC).UnixMilli()
	if got := int64FromAny(state.Data["sessionStartedAt"]); got != want {
		t.Fatalf("sessionStartedAt = %d, want %d", got, want)
	}
}
