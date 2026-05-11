package template

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/modu-ai/moai-adk/internal/manifest"
)

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
