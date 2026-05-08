package agentruntime

import "testing"

func TestSanitizeBranchPart(t *testing.T) {
	got := sanitizeBranchPart(" expert backend!? ")
	if got != "expert-backend" {
		t.Fatalf("sanitizeBranchPart() = %q", got)
	}
	if sanitizeBranchPart("///") != "agent" {
		t.Fatal("empty sanitized branch should fall back to agent")
	}
}
