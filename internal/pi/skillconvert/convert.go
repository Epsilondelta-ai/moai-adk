// Package skillconvert converts Claude Code MoAI skills into Pi-compatible skills.
package skillconvert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Result summarizes generated Pi skills.
type Result struct {
	Skills  []string `json:"skills"`
	Stale   []string `json:"stale,omitempty"`
	Checked bool     `json:"checked,omitempty"`
}

// Convert converts .claude/skills into .pi/skills.
func Convert(cwd string) (*Result, error) {
	return convertSkills(cwd, false)
}

// Check reports skill drift without writing files.
func Check(cwd string) (*Result, error) {
	return convertSkills(cwd, true)
}

func convertSkills(cwd string, check bool) (*Result, error) {
	sourceRoot := filepath.Join(cwd, ".claude", "skills")
	targetRoot := filepath.Join(cwd, ".pi", "skills")
	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("read skills: %w", err)
	}
	if !check {
		if err := os.MkdirAll(targetRoot, 0o755); err != nil {
			return nil, err
		}
	}
	result := &Result{Checked: check}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		sourceDir := filepath.Join(sourceRoot, skillName)
		if _, err := os.Stat(filepath.Join(sourceDir, "SKILL.md")); err != nil {
			continue
		}
		targetDir := filepath.Join(targetRoot, skillName)
		if check {
			stale, err := skillDirDiffers(sourceDir, targetDir)
			if err != nil || stale {
				result.Stale = append(result.Stale, targetDir)
			}
		} else if err := copySkillDir(sourceDir, targetDir); err != nil {
			return nil, err
		}
		result.Skills = append(result.Skills, targetDir)
	}
	return result, nil
}

func skillDirDiffers(sourceDir, targetDir string) (bool, error) {
	stale := false
	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || stale {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetDir, rel)
		if d.IsDir() {
			if info, err := os.Stat(target); err != nil || !info.IsDir() {
				stale = true
			}
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(bytes)
		if filepath.Base(path) == "SKILL.md" || strings.HasSuffix(path, ".md") {
			content = ConvertSkillText(content)
			if filepath.Base(path) == "SKILL.md" {
				content = NormalizeSkillFrontmatter(content)
			}
		}
		existing, err := os.ReadFile(target)
		if err != nil || string(existing) != content {
			stale = true
		}
		return nil
	})
	return stale, err
}

func copySkillDir(sourceDir, targetDir string) error {
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	return filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(bytes)
		if filepath.Base(path) == "SKILL.md" || strings.HasSuffix(path, ".md") {
			content = ConvertSkillText(content)
			if filepath.Base(path) == "SKILL.md" {
				content = NormalizeSkillFrontmatter(content)
			}
		}
		return os.WriteFile(target, []byte(content), 0o644)
	})
}

// NormalizeSkillFrontmatter ensures SKILL.md starts with YAML frontmatter.
// Pi follows the Agent Skills convention and ignores descriptions if comments
// appear before the opening --- marker.
func NormalizeSkillFrontmatter(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if strings.HasPrefix(content, "---\n") {
		return content
	}
	start := strings.Index(content, "\n---\n")
	if start < 0 {
		return content
	}
	prefix := strings.TrimSpace(content[:start])
	rest := content[start+1:]
	end := strings.Index(rest[len("---\n"):], "\n---\n")
	if end < 0 {
		return content
	}
	end += len("---\n")
	frontmatter := rest[:end+len("\n---\n")]
	body := rest[end+len("\n---\n"):]
	if prefix == "" {
		return frontmatter + body
	}
	return frontmatter + "\n" + prefix + "\n" + body
}

// ConvertSkillText rewrites Claude Code-specific runtime instructions for Pi.
func ConvertSkillText(content string) string {
	replacements := []struct{ old, new string }{
		{"AskUserQuestion", "moai_ask_user"},
		{"TaskCreate", "moai_task_create"},
		{"TaskUpdate", "moai_task_update"},
		{"TaskList", "moai_task_list"},
		{"TaskGet", "moai_task_get"},
		{"TodoWrite", "moai_task_create/moai_task_update"},
		{"Agent(", "moai_agent_invoke("},
		{"TeamCreate", "moai_team_run"},
		{"SendMessage", "moai_team_run message dispatch"},
		{"Claude Code", "Pi/MoAI runtime"},
		{"CLAUDE.md", "AGENTS.md or MoAI loaded context"},
		{".claude/", ".pi/ or .moai runtime resources converted from .claude/"},
		{"WebSearch", "configured Pi web/search skill"},
		{"WebFetch", "configured Pi web/fetch skill"},
		{"Skill(\"moai\")", "/skill:moai"},
	}
	for _, replacement := range replacements {
		content = strings.ReplaceAll(content, replacement.old, replacement.new)
	}
	return content
}
