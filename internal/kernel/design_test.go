package kernel

import (
	"context"
	"os"
	"testing"
)

func TestExecuteDesignCreatesSession(t *testing.T) {
	cwd := t.TempDir()
	result, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "design", Args: "landing page", CWD: cwd})
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	path, _ := result.Data["designSession"].(string)
	if path == "" {
		t.Fatalf("missing designSession: %#v", result)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("design session not created: %v", err)
	}
}
