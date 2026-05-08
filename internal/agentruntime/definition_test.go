package agentruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefinition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "expert-test.md")
	content := `---
name: expert-test
description: |
  Test expert
tools: Read, Write, Bash
model: sonnet
permissionMode: bypassPermissions
skills:
  - moai-foundation-core
---

# Test Expert

System prompt.
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	def, err := LoadDefinition(path)
	if err != nil {
		t.Fatalf("LoadDefinition() error = %v", err)
	}
	if def.Name != "expert-test" || def.Model != "sonnet" || len(def.Tools) != 3 || len(def.Skills) != 1 {
		t.Fatalf("unexpected definition: %#v", def)
	}
	if def.SystemPrompt == "" {
		t.Fatal("SystemPrompt is empty")
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	agentsDir := filepath.Join(root, ".claude", "agents", "moai")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "a.md"), []byte("---\nname: a\ndescription: A\n---\nA"), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	defs, err := Discover(child)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(defs) != 1 || defs[0].Name != "a" {
		t.Fatalf("Discover() = %#v", defs)
	}
}
