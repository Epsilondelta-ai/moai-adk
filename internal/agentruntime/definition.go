// Package agentruntime provides runtime-neutral MoAI agent discovery and invocation contracts.
package agentruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Definition is a MoAI agent definition loaded from markdown frontmatter.
type Definition struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Tools          []string `json:"tools,omitempty"`
	Model          string   `json:"model,omitempty"`
	PermissionMode string   `json:"permissionMode,omitempty"`
	Memory         string   `json:"memory,omitempty"`
	Skills         []string `json:"skills,omitempty"`
	SystemPrompt   string   `json:"systemPrompt"`
	Path           string   `json:"path"`
}

type frontmatter struct {
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	Tools          any      `yaml:"tools"`
	Model          string   `yaml:"model"`
	PermissionMode string   `yaml:"permissionMode"`
	Memory         string   `yaml:"memory"`
	Skills         []string `yaml:"skills"`
}

// Discover loads MoAI agent definitions from .claude/agents/moai under cwd or an ancestor.
func Discover(cwd string) ([]Definition, error) {
	dir := findAgentsDir(cwd)
	if dir == "" {
		return []Definition{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var defs []Definition
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		def, err := LoadDefinition(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs, nil
}

// ResolveDefinition maps user-facing aliases such as "backend" or
// "expert backend" to concrete MoAI agent definition names.
func ResolveDefinition(defs []Definition, requested string) (*Definition, string) {
	alias := NormalizeAgentName(requested)
	for i := range defs {
		if NormalizeAgentName(defs[i].Name) == alias {
			return &defs[i], defs[i].Name
		}
	}
	for i := range defs {
		name := NormalizeAgentName(defs[i].Name)
		if strings.HasPrefix(name, "expert-") && strings.TrimPrefix(name, "expert-") == alias {
			return &defs[i], defs[i].Name
		}
		if strings.HasPrefix(name, "manager-") && strings.TrimPrefix(name, "manager-") == alias {
			return &defs[i], defs[i].Name
		}
	}
	return nil, requested
}

// NormalizeAgentName converts display names to stable agent lookup keys.
func NormalizeAgentName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.Join(strings.Fields(name), "-")
	return name
}

// LoadDefinition parses one markdown agent definition.
func LoadDefinition(path string) (Definition, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}
	fmText, body, err := splitFrontmatter(string(bytes))
	if err != nil {
		return Definition{}, fmt.Errorf("parse %s: %w", path, err)
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return Definition{}, fmt.Errorf("parse frontmatter %s: %w", path, err)
	}
	if fm.Name == "" {
		fm.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return Definition{
		Name:           fm.Name,
		Description:    strings.TrimSpace(fm.Description),
		Tools:          normalizeTools(fm.Tools),
		Model:          fm.Model,
		PermissionMode: fm.PermissionMode,
		Memory:         fm.Memory,
		Skills:         fm.Skills,
		SystemPrompt:   strings.TrimSpace(body),
		Path:           path,
	}, nil
}

func findAgentsDir(cwd string) string {
	current := cwd
	for {
		candidate := filepath.Join(current, ".claude", "agents", "moai")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func splitFrontmatter(content string) (string, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", content, nil
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return "", "", fmt.Errorf("closing frontmatter marker not found")
	}
	return rest[:idx], rest[idx+len("\n---\n"):], nil
}

func normalizeTools(value any) []string {
	switch v := value.(type) {
	case string:
		parts := strings.Split(v, ",")
		tools := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				tools = append(tools, trimmed)
			}
		}
		return tools
	case []any:
		tools := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				tools = append(tools, strings.TrimSpace(s))
			}
		}
		return tools
	default:
		return nil
	}
}
