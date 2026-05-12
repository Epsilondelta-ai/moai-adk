package template

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/modu-ai/moai-adk/internal/manifest"
)

func TestEmbeddedPiTemplatesIncludeRuntimeArtifacts(t *testing.T) {
	fsys, err := EmbeddedTemplates()
	if err != nil {
		t.Fatalf("EmbeddedTemplates() error = %v", err)
	}

	for _, rel := range []string{
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
		if _, err := fs.Stat(fsys, rel); err != nil {
			t.Fatalf("expected embedded Pi runtime artifact %q: %v", rel, err)
		}
	}

	for _, rel := range []string{
		".pi/npm",
		".pi/state/sessions",
	} {
		if _, err := fs.Stat(fsys, rel); !os.IsNotExist(err) {
			t.Fatalf("embedded Pi templates must not include transient path %q; stat err = %v", rel, err)
		}
	}
}

func TestPiOnlyDeployerFiltersTemplates(t *testing.T) {
	root, mgr := setupDeployProject(t)
	fsys := fstest.MapFS{
		".moai/config/sections/user.yaml": &fstest.MapFile{Data: []byte("user: {}\n")},
		".pi/settings.json":               &fstest.MapFile{Data: []byte("{}\n")},
		".claude/settings.json":           &fstest.MapFile{Data: []byte("{}\n")},
		"CLAUDE.md":                       &fstest.MapFile{Data: []byte("# Claude\n")},
		".github/workflows/ci.yml":        &fstest.MapFile{Data: []byte("name: ci\n")},
		".gitignore":                      &fstest.MapFile{Data: []byte("dist/\n")},
		"Makefile":                        &fstest.MapFile{Data: []byte("test:\n")},
		"scripts/setup.sh":                &fstest.MapFile{Data: []byte("#!/bin/sh\n")},
	}

	deployer := NewPiOnlyDeployer(fsys)
	if err := deployer.Deploy(context.Background(), root, mgr, nil); err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	for _, rel := range []string{
		".moai/config/sections/user.yaml",
		".pi/settings.json",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected Pi-mode artifact %q to exist: %v", rel, err)
		}
		if entry, ok := mgr.GetEntry(rel); !ok {
			t.Fatalf("expected manifest entry for %q", rel)
		} else if entry.Provenance != manifest.TemplateManaged {
			t.Fatalf("manifest provenance for %q = %s, want %s", rel, entry.Provenance, manifest.TemplateManaged)
		}
	}

	for _, rel := range []string{
		".claude/settings.json",
		"CLAUDE.md",
		".github/workflows/ci.yml",
		".gitignore",
		"Makefile",
		"scripts/setup.sh",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("Pi-mode deployer must not create %q; stat err = %v", rel, err)
		}
		if _, ok := mgr.GetEntry(rel); ok {
			t.Fatalf("Pi-mode deployer must not track non-Pi manifest entry %q", rel)
		}
	}

	for _, rel := range deployer.ListTemplates() {
		if !strings.HasPrefix(rel, ".moai/") && !strings.HasPrefix(rel, ".pi/") {
			t.Fatalf("Pi-mode ListTemplates returned non-Pi path %q", rel)
		}
	}
}
