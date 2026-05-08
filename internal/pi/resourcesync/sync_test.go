package resourcesync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncGeneratesPiPrompts(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".claude", "commands", "moai")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "plan.md"), []byte("---\ndescription: plan\n---\nUse Skill(\"moai\") with arguments: plan $ARGUMENTS"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Sync(root)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(result.Prompts) != 1 {
		t.Fatalf("prompts = %#v", result.Prompts)
	}
	bytes, err := os.ReadFile(filepath.Join(root, ".pi", "prompts", "moai-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(bytes)
	if !strings.Contains(content, "Load and follow the moai skill") || strings.Contains(content, "Skill(\"moai\")") {
		t.Fatalf("unexpected generated content: %s", content)
	}
	check, err := Check(root)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(check.Stale) != 0 {
		t.Fatalf("expected no drift, got %#v", check.Stale)
	}
	if err := os.WriteFile(filepath.Join(root, ".pi", "prompts", "moai-plan.md"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	check, err = Check(root)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(check.Stale) != 1 {
		t.Fatalf("expected drift, got %#v", check.Stale)
	}
}
