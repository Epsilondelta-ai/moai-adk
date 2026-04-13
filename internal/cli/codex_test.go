package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	codexruntime "github.com/modu-ai/moai-adk/internal/codex"
)

func TestCodexCmd_IsSubcommandOfRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "codex" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("codex should be registered as a subcommand of root")
	}
}

func TestCodexCmd_HelpListsDoctor(t *testing.T) {
	usage := codexCmd.UsageString()
	if !strings.Contains(usage, "doctor") {
		t.Fatalf("codex usage should list doctor subcommand, got %q", usage)
	}
}

func TestWriteCodexDoctorReport_TextOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	inspector := codexruntime.Inspector{
		LookPath: func(string) (string, error) {
			return "", errors.New("missing git")
		},
	}

	if err := writeCodexDoctorReport(buf, t.TempDir(), false, false, inspector); err != nil {
		t.Fatalf("writeCodexDoctorReport error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Codex Readiness") {
		t.Fatalf("output should contain title, got %q", output)
	}
	if !strings.Contains(output, "Suggested Next Steps") {
		t.Fatalf("output should contain next steps, got %q", output)
	}
	if !strings.Contains(output, ".moai/ directory not found") {
		t.Fatalf("output should report missing MoAI config, got %q", output)
	}
}

func TestWriteCodexDoctorReport_JSONOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	root := t.TempDir()
	writeCodexFixtureForCLI(t, root)

	inspector := codexruntime.Inspector{
		LookPath: func(string) (string, error) {
			return "/usr/bin/git", nil
		},
	}

	if err := writeCodexDoctorReport(buf, root, true, true, inspector); err != nil {
		t.Fatalf("writeCodexDoctorReport error: %v", err)
	}

	var report codexruntime.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if !report.Ready {
		t.Fatal("report.Ready = false, want true")
	}
	if len(report.Checks) != 4 {
		t.Fatalf("len(report.Checks) = %d, want 4", len(report.Checks))
	}
}

func writeCodexFixtureForCLI(t *testing.T, root string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(root, ".moai", "config", "sections"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".codex", "skills", "moai", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".codex", "skills", "moai", "SKILL.md"), []byte("# `$moai` Codex Entry Point\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"project.md", "plan.md", "run.md", "sync.md", "review.md", "clean.md", "loop.md"} {
		if err := os.WriteFile(filepath.Join(root, ".codex", "skills", "moai", "workflows", name), []byte("# Workflow\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
