package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInit_PiModeCreatesMoAIAndPiWithoutClaude(t *testing.T) {
	root := t.TempDir()
	init := NewInitializer(nil, nil, nil)

	_, err := init.Init(context.Background(), InitOptions{
		ProjectRoot:     root,
		ProjectName:     "pi-only",
		Language:        "Go",
		Framework:       "none",
		UserName:        "test",
		ConvLang:        "en",
		DevelopmentMode: "tdd",
		PiMode:          true,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for _, rel := range []string{
		".moai",
		".moai/config/sections",
		".pi",
	} {
		if info, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %q to exist: %v", rel, err)
		} else if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", rel)
		}
	}

	for _, rel := range []string{
		".claude",
		"CLAUDE.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("PiMode must not create %q; stat err = %v", rel, err)
		}
	}
}
