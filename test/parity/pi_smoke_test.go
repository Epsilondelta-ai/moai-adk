package parity_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modu-ai/moai-adk/internal/kernel"
	"github.com/modu-ai/moai-adk/internal/pi/bridge"
)

func TestPiEndToEndSmokeFlow(t *testing.T) {
	cwd := t.TempDir()
	h := bridge.NewHandler()
	doctor := h.Handle(context.Background(), bridge.Request{Kind: "doctor", CWD: cwd})
	if !doctor.OK {
		t.Fatalf("doctor failed: %#v", doctor)
	}
	plan := h.Handle(context.Background(), bridge.Request{Kind: "command", CWD: cwd, Payload: map[string]any{"command": "plan", "args": "small smoke feature"}})
	if !plan.OK {
		t.Fatalf("plan failed: %#v", plan)
	}
	commandResult := plan.Data["commandResult"].(*kernel.CommandResult)
	specID := commandResult.Data["specId"].(string)
	for _, command := range []string{"run", "sync"} {
		res := h.Handle(context.Background(), bridge.Request{Kind: "command", CWD: cwd, Payload: map[string]any{"command": command, "args": specID}})
		if !res.OK {
			t.Fatalf("%s failed: %#v", command, res)
		}
	}
	task := h.Handle(context.Background(), bridge.Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_task_create", "payload": map[string]any{"subject": "smoke", "description": "task"}}})
	if !task.OK {
		t.Fatalf("task create failed: %#v", task)
	}
	quality := h.Handle(context.Background(), bridge.Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_quality_gate"}})
	if !quality.OK {
		t.Fatalf("quality failed: %#v", quality)
	}
	if err := os.WriteFile(filepath.Join(cwd, "smoke.go"), []byte("// @MX:NOTE smoke\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mx := h.Handle(context.Background(), bridge.Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_mx_scan"}})
	if !mx.OK || mx.Data["passed"] != true {
		t.Fatalf("mx failed: %#v", mx)
	}
}
