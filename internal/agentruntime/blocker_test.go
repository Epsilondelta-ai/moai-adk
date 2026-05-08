package agentruntime

import (
	"strings"
	"testing"
)

func TestParseBlockerReport(t *testing.T) {
	report, ok := ParseBlockerReport(`prefix {"blocker":true,"reason":"need spec","recovery":"run plan"} suffix`)
	if !ok || report.Reason != "need spec" || report.Recovery != "run plan" {
		t.Fatalf("unexpected report: %#v ok=%v", report, ok)
	}
	blocked, marker := DetectBlocker(`{"blocker":true,"reason":"need spec"}`)
	if !blocked || marker != "need spec" {
		t.Fatalf("unexpected blocker detection: %v %q", blocked, marker)
	}
}

func TestAgentExecutionContractIncludesNoPromptRule(t *testing.T) {
	content := agentExecutionContract(&Definition{Name: "expert-test", SystemPrompt: "base"})
	if !containsAll(content, []string{"Do not ask the user", "blocker", "allowed tools"}) {
		t.Fatalf("contract missing expected clauses: %s", content)
	}
}

func containsAll(text string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}
