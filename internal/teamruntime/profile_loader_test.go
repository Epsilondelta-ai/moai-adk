package teamruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".moai", "config", "sections")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "workflow:\n  team:\n    role_profiles:\n      implementer:\n        mode: acceptEdits\n        model: sonnet\n        isolation: worktree\n        description: impl\n"
	if err := os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	profiles, err := LoadProfiles(root)
	if err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}
	if profiles["implementer"].Isolation != "worktree" {
		t.Fatalf("profile not loaded: %#v", profiles["implementer"])
	}
}
