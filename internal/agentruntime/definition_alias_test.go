package agentruntime

import "testing"

func TestResolveDefinitionSupportsUserFacingAliases(t *testing.T) {
	defs := []Definition{{Name: "expert-backend"}, {Name: "manager-spec"}}
	def, resolved := ResolveDefinition(defs, "backend")
	if def == nil || resolved != "expert-backend" {
		t.Fatalf("backend alias not resolved: def=%#v resolved=%q", def, resolved)
	}
	def, resolved = ResolveDefinition(defs, "Manager Spec")
	if def == nil || resolved != "manager-spec" {
		t.Fatalf("manager alias not resolved: def=%#v resolved=%q", def, resolved)
	}
}
