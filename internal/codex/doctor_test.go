package codex

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectorInspect_AllChecksPass(t *testing.T) {
	root := t.TempDir()
	writeCodexFixture(t, root)

	inspector := Inspector{
		LookPath: func(string) (string, error) {
			return "/usr/bin/git", nil
		},
	}

	report := inspector.Inspect(root, true)

	if !report.Ready {
		t.Fatal("report.Ready = false, want true")
	}
	if report.Summary.Fail != 0 {
		t.Fatalf("report.Summary.Fail = %d, want 0", report.Summary.Fail)
	}
	if len(report.Checks) != 4 {
		t.Fatalf("len(report.Checks) = %d, want 4", len(report.Checks))
	}

	skill := findCheck(t, report, "Codex Skill")
	if skill.Status != StatusOK {
		t.Fatalf("Codex Skill status = %q, want %q", skill.Status, StatusOK)
	}
	if skill.Detail == "" {
		t.Fatal("Codex Skill detail should be populated in verbose mode")
	}
}

func TestInspectorInspect_MissingCodexAssetsFailsReadiness(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".moai", "config", "sections"), 0o755); err != nil {
		t.Fatal(err)
	}

	inspector := Inspector{
		LookPath: func(string) (string, error) {
			return "/usr/bin/git", nil
		},
	}

	report := inspector.Inspect(root, false)

	if report.Ready {
		t.Fatal("report.Ready = true, want false")
	}
	if report.Summary.Fail != 2 {
		t.Fatalf("report.Summary.Fail = %d, want 2", report.Summary.Fail)
	}

	skill := findCheck(t, report, "Codex Skill")
	if skill.Status != StatusFail {
		t.Fatalf("Codex Skill status = %q, want %q", skill.Status, StatusFail)
	}
	workflows := findCheck(t, report, "Codex Workflows")
	if workflows.Status != StatusFail {
		t.Fatalf("Codex Workflows status = %q, want %q", workflows.Status, StatusFail)
	}
}

func TestInspectorInspect_MissingWorkflowDocsReportsNames(t *testing.T) {
	root := t.TempDir()
	writeCodexFixture(t, root)

	if err := os.Remove(filepath.Join(root, workflowsDir, "sync.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, workflowsDir, "loop.md")); err != nil {
		t.Fatal(err)
	}

	inspector := NewInspector()
	report := inspector.Inspect(root, false)

	check := findCheck(t, report, "Codex Workflows")
	if check.Status != StatusFail {
		t.Fatalf("Codex Workflows status = %q, want %q", check.Status, StatusFail)
	}
	if check.Detail != "Missing: sync.md, loop.md" && check.Detail != "Missing: loop.md, sync.md" {
		t.Fatalf("Codex Workflows detail = %q, want missing file list", check.Detail)
	}
}

func TestInspectorInspect_MissingGitWarnsWithoutBlockingReadiness(t *testing.T) {
	root := t.TempDir()
	writeCodexFixture(t, root)

	inspector := Inspector{
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
	}

	report := inspector.Inspect(root, false)

	if !report.Ready {
		t.Fatal("report.Ready = false, want true when only git is missing")
	}
	if report.Summary.Warn != 1 {
		t.Fatalf("report.Summary.Warn = %d, want 1", report.Summary.Warn)
	}

	check := findCheck(t, report, "Git")
	if check.Status != StatusWarn {
		t.Fatalf("Git status = %q, want %q", check.Status, StatusWarn)
	}
}

func findCheck(t *testing.T, report Report, name string) Check {
	t.Helper()

	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("check %q not found", name)
	return Check{}
}

func writeCodexFixture(t *testing.T, root string) {
	t.Helper()

	for _, dir := range []string{
		filepath.Join(root, ".moai", "config", "sections"),
		filepath.Join(root, ".codex", "skills", "moai", "workflows"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, skillPath), []byte("# `$moai` Codex Entry Point\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, name := range expectedWorkflowDocs {
		path := filepath.Join(root, workflowsDir, name)
		if err := os.WriteFile(path, []byte("# Workflow\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
