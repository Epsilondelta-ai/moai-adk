package template

import (
	"encoding/json"
	"fmt"
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
		Extensions []string `json:"extensions"`
		Skills     []string `json:"skills"`
		Prompts    []string `json:"prompts"`
		Packages   []any    `json:"packages"`
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
		if !isPiDefaultComponentActiveForTest(name, configured, settings.Extensions) {
			t.Fatalf("default Pi component %q is not active in packages or local extension", name)
		}
	}

	for _, requiredPath := range []struct {
		name   string
		values []string
		want   string
	}{
		{name: "extensions", values: settings.Extensions, want: "./extensions/moai-claude-compat"},
		{name: "skills", values: settings.Skills, want: "./generated/source/skills"},
		{name: "prompts", values: settings.Prompts, want: "./prompts"},
	} {
		if !containsStringForTest(requiredPath.values, requiredPath.want) {
			t.Fatalf(".pi/settings.json %s must reference %q, got %#v", requiredPath.name, requiredPath.want, requiredPath.values)
		}
	}
	if !containsPackageSpecForTest(settings.Packages, "./packages/pi-provider-kimi-code") {
		t.Fatalf(".pi/settings.json packages must include local Kimi provider package, got %#v", settings.Packages)
	}

	contextMode := findPiPackageSpecForTest(settings.Packages, "context-mode")
	contextModeObject, ok := contextMode.(map[string]any)
	if !ok {
		t.Fatal("context-mode package must use object filter form")
	}
	if extensions, ok := contextModeObject["extensions"].([]any); !ok || len(extensions) != 0 {
		t.Fatalf("context-mode extensions must be disabled, got %#v", contextModeObject["extensions"])
	}
	if skills, ok := contextModeObject["skills"].([]any); !ok || len(skills) != 1 || skills[0] != "./skills" {
		t.Fatalf("context-mode skills filter must preserve ./skills, got %#v", contextModeObject["skills"])
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

func findPiPackageSpecForTest(specs []any, packageName string) any {
	for _, spec := range specs {
		if normalizePiPackageSpecForTest(spec) == packageName {
			return spec
		}
	}
	return nil
}

func normalizePiPackageSpecForTest(spec any) string {
	value := fmt.Sprint(spec)
	if object, ok := spec.(map[string]any); ok {
		value = fmt.Sprint(object["source"])
	}
	value = strings.TrimPrefix(strings.TrimPrefix(value, "npm:"), "git:")
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

func isPiDefaultComponentActiveForTest(name string, configured map[string]bool, extensions []string) bool {
	if configured[name] {
		return true
	}
	switch name {
	case "moai-claude-compat", "pi-notify-glass.ts":
		return containsStringForTest(extensions, "./extensions/moai-claude-compat")
	default:
		return false
	}
}

func containsStringForTest(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPackageSpecForTest(specs []any, want string) bool {
	for _, spec := range specs {
		if value, ok := spec.(string); ok && value == want {
			return true
		}
	}
	return false
}
