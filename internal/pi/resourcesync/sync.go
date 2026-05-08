// Package resourcesync generates Pi-compatible resources from MoAI Claude assets.
package resourcesync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Result summarizes a resource sync run.
type Result struct {
	Prompts []string `json:"prompts"`
	Stale   []string `json:"stale,omitempty"`
	Checked bool     `json:"checked,omitempty"`
}

// Sync generates Pi prompt templates from .claude/commands/moai.
func Sync(cwd string) (*Result, error) {
	return syncPrompts(cwd, false)
}

// Check reports prompt drift without writing files.
func Check(cwd string) (*Result, error) {
	return syncPrompts(cwd, true)
}

func syncPrompts(cwd string, check bool) (*Result, error) {
	sourceDir := filepath.Join(cwd, ".claude", "commands", "moai")
	targetDir := filepath.Join(cwd, ".pi", "prompts")
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("read command templates: %w", err)
	}
	if !check {
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return nil, err
		}
	}
	result := &Result{Checked: check}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		bytes, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		content := toPiPrompt(name, string(bytes))
		target := filepath.Join(targetDir, "moai-"+name+".md")
		if check {
			if stale, err := fileContentDiffers(target, content); err != nil || stale {
				result.Stale = append(result.Stale, target)
			}
		} else if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return nil, err
		}
		result.Prompts = append(result.Prompts, target)
	}
	if err := syncOutputStyle(cwd, check, result); err != nil {
		return nil, err
	}
	return result, nil
}

func syncOutputStyle(cwd string, check bool, result *Result) error {
	source := filepath.Join(cwd, ".claude", "output-styles", "moai", "moai.md")
	bytes, err := os.ReadFile(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	content := toPiOutputStyle(string(bytes))
	target := filepath.Join(cwd, ".pi", "prompts", "moai-output-style.md")
	if check {
		if stale, err := fileContentDiffers(target, content); err != nil || stale {
			result.Stale = append(result.Stale, target)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return err
		}
	}
	result.Prompts = append(result.Prompts, target)
	return nil
}

func fileContentDiffers(path, expected string) (bool, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return true, err
	}
	return string(bytes) != expected, nil
}

func toPiPrompt(name, content string) string {
	body := stripFrontmatter(content)
	body = strings.ReplaceAll(body, "Use Skill(\"moai\") with arguments:", "Load and follow the moai skill with arguments:")
	return fmt.Sprintf("---\ndescription: MoAI %s workflow prompt for Pi\ngenerated_from: .claude/commands/moai/%s.md\n---\n\n%s\n", name, name, strings.TrimSpace(body))
}

func toPiOutputStyle(content string) string {
	body := stripFrontmatter(content)
	body = strings.ReplaceAll(body, "Claude Code", "Pi/MoAI runtime")
	return "---\ndescription: MoAI output style prompt for Pi\ngenerated_from: .claude/output-styles/moai/moai.md\n---\n\n" + strings.TrimSpace(body) + "\n"
}

func stripFrontmatter(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return content
	}
	return rest[idx+len("\n---\n"):]
}
