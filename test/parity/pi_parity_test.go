package parity_test

import (
	"context"
	"testing"

	"github.com/modu-ai/moai-adk/internal/kernel"
	"github.com/modu-ai/moai-adk/internal/pi/bridge"
	"github.com/modu-ai/moai-adk/internal/policy"
)

func TestPiCommandParityUsesKernel(t *testing.T) {
	cwd := t.TempDir()
	kernelResult, err := kernel.New().ExecuteCommand(context.Background(), kernel.CommandRequest{Command: "plan", Args: "parity", CWD: cwd})
	if err != nil {
		t.Fatalf("kernel command error = %v", err)
	}
	bridgeResult := bridge.NewHandler().Handle(context.Background(), bridge.Request{Kind: "command", CWD: cwd, Payload: map[string]any{"command": "plan", "args": "parity"}})
	if !bridgeResult.OK {
		t.Fatalf("bridge command failed: %#v", bridgeResult.Error)
	}
	if !kernelResult.OK {
		t.Fatalf("kernel result failed: %#v", kernelResult)
	}
}

func TestPiEventPolicyParity(t *testing.T) {
	piResult := bridge.NewHandler().Handle(context.Background(), bridge.Request{Kind: "event", CWD: t.TempDir(), Payload: map[string]any{"event": "tool_call", "toolName": "bash", "input": map[string]any{"command": "rm -rf /"}}})
	if !piResult.OK {
		t.Fatalf("bridge event failed: %#v", piResult.Error)
	}
	coreResult := policy.NewEvaluator().Evaluate(policy.EventRequest{Event: "tool_call", ToolName: "bash", ToolInput: map[string]any{"command": "rm -rf /"}})
	decision, _ := piResult.Data["decision"].(policy.Decision)
	if !coreResult.Decision.Block || !decision.Block {
		t.Fatalf("expected both policies to block: bridge=%#v core=%#v", piResult.Data["decision"], coreResult.Decision)
	}
}
