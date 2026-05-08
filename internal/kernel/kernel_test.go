package kernel

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePlanCreatesCanonicalSpec(t *testing.T) {
	cwd := t.TempDir()
	result, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "plan", Args: "build pi parity", CWD: cwd})
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if !result.OK || result.Data["specId"] == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
	specID := result.Data["specId"].(string)
	if specID != "SPEC-PI-001" {
		t.Fatalf("specID = %q, want SPEC-PI-001", specID)
	}
	for _, name := range []string{"spec.md", "plan.md", "acceptance.md", "workflow.json", "delegation.md", "clarifications.md", "status.json"} {
		if _, err := os.Stat(filepath.Join(cwd, ".moai", "specs", specID, name)); err != nil {
			t.Fatalf("%s not created: %v", name, err)
		}
	}
	bytes, err := os.ReadFile(filepath.Join(cwd, ".moai", "specs", specID, "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(bytes)
	for _, required := range []string{"id: SPEC-PI-001", "created_at:", "updated_at:", "issue_number: null", "## EARS REQUIREMENTS", "## Exclusions (What NOT to Build)"} {
		if !strings.Contains(content, required) {
			t.Fatalf("spec.md missing %q:\n%s", required, content)
		}
	}
}

func TestExecutePlanRequiresDescription(t *testing.T) {
	cwd := t.TempDir()
	result, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "plan", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Data["blocker"] != true {
		t.Fatalf("expected blocker result: %#v", result)
	}
}

func TestExecuteRunUsesLatestCanonicalSpec(t *testing.T) {
	cwd := t.TempDir()
	plan, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "plan", Args: "x", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	run, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "run", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if !run.OK || run.Data["specId"] != plan.Data["specId"] {
		t.Fatalf("run did not use latest spec: %#v", run)
	}
	specID := plan.Data["specId"].(string)
	for _, name := range []string{"run-plan.md", "progress.md", "workflow.json", "delegation.md"} {
		if _, err := os.Stat(filepath.Join(cwd, ".moai", "specs", specID, name)); err != nil {
			t.Fatalf("%s not created: %v", name, err)
		}
	}
}

func TestExecuteRunBlocksNonCanonicalSpec(t *testing.T) {
	cwd := t.TempDir()
	specDir := filepath.Join(cwd, ".moai", "specs", "SPEC-PI-001")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "run", Args: "SPEC-PI-001", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Data["blocker"] != true {
		t.Fatalf("expected blocker for non-canonical spec: %#v", result)
	}
}

func TestExecuteSyncRequiresRunArtifacts(t *testing.T) {
	cwd := t.TempDir()
	plan, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "plan", Args: "x", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	result, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "sync", Args: plan.Data["specId"].(string), CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Data["blocker"] != true {
		t.Fatalf("expected sync blocker before run: %#v", result)
	}
}

func TestExecuteSyncAfterRun(t *testing.T) {
	cwd := t.TempDir()
	plan, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "plan", Args: "x", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	specID := plan.Data["specId"].(string)
	if _, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "run", Args: specID, CWD: cwd}); err != nil {
		t.Fatal(err)
	}
	sync, err := New().ExecuteCommand(context.Background(), CommandRequest{Command: "sync", Args: specID, CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if !sync.OK || sync.Data["specId"] != specID {
		t.Fatalf("unexpected sync result: %#v", sync)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".moai", "specs", specID, "sync.md")); err != nil {
		t.Fatalf("sync.md not created: %v", err)
	}
}
