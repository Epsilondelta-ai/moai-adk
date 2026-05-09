package template

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

func TestPiSettingsTemplateDefaultPackages(t *testing.T) {
	fsys, err := EmbeddedTemplates()
	if err != nil {
		t.Fatalf("EmbeddedTemplates() error: %v", err)
	}

	data, err := fs.ReadFile(fsys, ".pi/settings.json")
	if err != nil {
		t.Fatalf("ReadFile(.pi/settings.json) error: %v", err)
	}

	var settings struct {
		Packages   []string `json:"packages"`
		MoAICompat struct {
			DefaultPackages []string `json:"defaultPackages"`
		} `json:"moaiCompat"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf(".pi/settings.json template is invalid JSON: %v", err)
	}

	configured := make(map[string]bool, len(settings.Packages))
	for _, spec := range settings.Packages {
		configured[normalizePiPackageSpecForTest(spec)] = true
	}

	if len(settings.MoAICompat.DefaultPackages) == 0 {
		t.Fatal("moaiCompat.defaultPackages must not be empty")
	}
	for _, spec := range settings.MoAICompat.DefaultPackages {
		name := normalizePiPackageSpecForTest(spec)
		if !configured[name] {
			t.Fatalf("default package %q is not active in packages", name)
		}
	}

	for _, required := range []string{
		"@juicesharp/rpiv-ask-user-question",
		"@tmustier/pi-agent-teams",
		"@zenobius/pi-worktrees",
		"context-mode",
		"pi-docparser",
		"pi-markdown-preview",
		"pi-mcp-adapter",
		"pi-subagents",
		"pi-web-access",
		"pi-yaml-hooks",
	} {
		if !configured[required] {
			t.Fatalf("required Pi package %q missing from template", required)
		}
	}
}

func normalizePiPackageSpecForTest(spec string) string {
	value := strings.TrimPrefix(strings.TrimPrefix(spec, "npm:"), "git:")
	if before, _, ok := strings.Cut(value, "#"); ok {
		value = before
	}
	if before, _, ok := strings.Cut(value, "?"); ok {
		value = before
	}
	if strings.HasPrefix(value, "@") {
		parts := strings.Split(value, "@")
		if len(parts) > 2 {
			return "@" + parts[1]
		}
		return value
	}
	if before, _, ok := strings.Cut(value, "@"); ok {
		return before
	}
	return value
}
