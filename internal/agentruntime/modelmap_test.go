package agentruntime

import "testing"

func TestMapModelAlias(t *testing.T) {
	if MapModelAlias("sonnet") != "claude-sonnet-4-5" {
		t.Fatalf("sonnet alias not mapped")
	}
	if MapModelAlias("custom-model") != "custom-model" {
		t.Fatalf("custom model should pass through")
	}
}

func TestDetectBlocker(t *testing.T) {
	blocked, _ := DetectBlocker("BLOCKER: need user approval")
	if !blocked {
		t.Fatal("expected blocker")
	}
}
