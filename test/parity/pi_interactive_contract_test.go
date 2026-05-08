package parity_test

import (
	"os"
	"strings"
	"testing"
)

func TestPiInteractiveParityDocsCoverManualGates(t *testing.T) {
	bytes, err := os.ReadFile("../../docs/design/pi-final-parity-matrix.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(bytes)
	for _, want := range []string{"Normal chat", "moai_ask_user", "Footer widget", "Manual parity gates"} {
		if !strings.Contains(content, want) {
			t.Fatalf("parity matrix missing %q", want)
		}
	}
}
