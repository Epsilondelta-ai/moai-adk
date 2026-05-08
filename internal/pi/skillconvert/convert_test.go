package skillconvert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertSkillText(t *testing.T) {
	got := ConvertSkillText("Use AskUserQuestion then TaskCreate then Agent( expert ) in Claude Code")
	for _, forbidden := range []string{"AskUserQuestion", "TaskCreate", "Claude Code"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("ConvertSkillText left %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "moai_task_create") || !strings.Contains(got, "moai_agent_invoke") {
		t.Fatalf("ConvertSkillText missing Pi replacements: %q", got)
	}
}

func TestNormalizeSkillFrontmatterMovesLeadingComments(t *testing.T) {
	input := "<!-- comment -->\n---\nname: x\ndescription: y\n---\nBody"
	got := NormalizeSkillFrontmatter(input)
	if !strings.HasPrefix(got, "---\nname: x") {
		t.Fatalf("frontmatter was not moved to top: %q", got)
	}
	if !strings.Contains(got, "<!-- comment -->") {
		t.Fatalf("leading comment was not preserved: %q", got)
	}
}

func TestConvertGeneratesPiSkills(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".claude", "skills", "moai-test")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: moai-test\ndescription: test\n---\nUse AskUserQuestion."), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Convert(root)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("skills = %#v", result.Skills)
	}
	bytes, err := os.ReadFile(filepath.Join(root, ".pi", "skills", "moai-test", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytes), "AskUserQuestion") {
		t.Fatalf("skill was not converted: %s", string(bytes))
	}
}
