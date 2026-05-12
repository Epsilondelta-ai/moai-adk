package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitCmd_HasPiFlag(t *testing.T) {
	if initCmd.Flags().Lookup("pi") == nil {
		t.Fatal("init command must expose --pi flag")
	}
}

func TestRunInit_PiModeCreatesOnlyMoAIAndPiArtifacts(t *testing.T) {
	t.Setenv("MOAI_SKIP_BINARY_UPDATE", "1")
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	cmd := newRunInitPiTestCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	mustSetFlag(t, cmd, "root", root)
	mustSetFlag(t, cmd, "name", "pi-only")
	mustSetFlag(t, cmd, "language", "Go")
	mustSetFlag(t, cmd, "mode", "tdd")
	mustSetFlag(t, cmd, "non-interactive", "true")
	mustSetFlag(t, cmd, "no-hooks", "true")
	mustSetFlag(t, cmd, "pi", "true")

	if err := runInit(cmd, nil); err != nil {
		t.Fatalf("runInit() error = %v\nOutput:\n%s", err, out.String())
	}

	for _, rel := range []string{
		".moai/config/sections",
		".moai/manifest.json",
		".pi/settings.json",
		".pi/hooks.yaml",
		".pi/package.json",
		".pi/agents/moai/expert-debug.md",
		".pi/claude-compat/tool-aliases.json",
		".pi/extensions/moai-claude-compat/package.json",
		".pi/generated/source/CLAUDE.md",
		".pi/generated/source/skills/moai/SKILL.md",
		".pi/packages/pi-provider-kimi-code/package.json",
		".pi/prompts/moai.md",
		".pi/state/.gitkeep",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected Pi init artifact %q to exist: %v", rel, err)
		}
	}

	for _, rel := range []string{
		".pi/npm",
		".pi/state/sessions",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("moai init --pi must not create transient Pi path %q; stat err = %v", rel, err)
		}
	}

	for _, rel := range []string{
		".claude",
		"CLAUDE.md",
		".github",
		".gitignore",
		"Makefile",
		".mcp.json",
		"scripts",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("moai init --pi must not create %q; stat err = %v", rel, err)
		}
	}

	manifestData, err := os.ReadFile(filepath.Join(root, ".moai", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Files map[string]any `json:"files"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(manifest.Files) == 0 {
		t.Fatal("expected manifest to track Pi-mode template artifacts")
	}
	for path := range manifest.Files {
		if !strings.HasPrefix(path, ".moai/") && !strings.HasPrefix(path, ".pi/") {
			t.Fatalf("manifest contains non-Pi entry %q", path)
		}
	}
}

func newRunInitPiTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "init-test"}
	cmd.Flags().String("root", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("language", "", "")
	cmd.Flags().String("framework", "", "")
	cmd.Flags().String("mode", "", "")
	cmd.Flags().String("git-mode", "", "")
	cmd.Flags().String("git-provider", "", "")
	cmd.Flags().String("github-username", "", "")
	cmd.Flags().String("gitlab-instance-url", "", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("no-hooks", false, "")
	cmd.Flags().Bool("pi", false, "")
	return cmd
}

func mustSetFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("set flag %s=%s: %v", name, value, err)
	}
}
