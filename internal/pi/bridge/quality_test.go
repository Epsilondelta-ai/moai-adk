package bridge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunQualityChecksSkipsUnknownProject(t *testing.T) {
	result := runQualityChecks(t.TempDir())
	if result["passed"] != true || result["skipped"] != 1 {
		t.Fatalf("unexpected unknown project result: %#v", result)
	}
}

func TestRunQualityChecksReportsMissingRequiredTool(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "pytest.ini"), []byte("[pytest]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())
	result := runQualityChecks(cwd)
	t.Setenv("PATH", oldPath)
	if result["passed"] != false {
		t.Fatalf("expected failed result when required pytest is unavailable: %#v", result)
	}
	commands := result["commands"].([]qualityCommandResult)
	foundPytest := false
	for _, command := range commands {
		if command.Name == "pytest" {
			foundPytest = true
			if command.Status != "skipped" || !strings.Contains(command.Reason, "required tool") {
				t.Fatalf("unexpected pytest command result: %#v", command)
			}
		}
	}
	if !foundPytest {
		t.Fatalf("pytest command not detected: %#v", commands)
	}
}

func TestRunLSPCheckUsesFallbackDiagnostics(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runLSPCheck(cwd)
	if result["supported"] != true || result["filesChecked"] != 1 {
		t.Fatalf("unexpected lsp result: %#v", result)
	}
	if result["diagnostics"] == nil || result["unavailable"] == nil {
		t.Fatalf("lsp result missing structured diagnostics: %#v", result)
	}
}

func TestScanMXReportsMissingReasonAndMalformedTags(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "good.go"), []byte("// @MX:NOTE useful context\n// @MX:WARN risky\n// later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "bad.go"), []byte("// @MX:warn lowercase\n// @MX broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scanMX(cwd)
	if result["passed"] != false {
		t.Fatalf("expected mx scan failure: %#v", result)
	}
	counts := result["counts"].(map[string]int)
	if counts["NOTE"] != 1 || counts["WARN"] != 1 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
	missing := result["missingReasons"].([]textIssue)
	if len(missing) != 1 || missing[0].File != "good.go" {
		t.Fatalf("missing reason not reported: %#v", missing)
	}
	malformed := result["malformed"].([]textIssue)
	if len(malformed) != 2 {
		t.Fatalf("malformed tags not reported: %#v", malformed)
	}
}

func TestScanMXAcceptsNearbyWarnReasonAndFiltersPaths(t *testing.T) {
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "src", "a.go"), []byte("// @MX:WARN risky\n// @MX:REASON guarded by tests\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "b.go"), []byte("// @MX:TODO outside filter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scanMX(cwd, "src")
	if result["passed"] != true {
		t.Fatalf("expected filtered scan to pass: %#v", result)
	}
	counts := result["counts"].(map[string]int)
	if counts["WARN"] != 1 || counts["TODO"] != 0 {
		t.Fatalf("unexpected filtered counts: %#v", counts)
	}
}

func TestHandleToolSeparatesLSPFromQualityAndSupportsMXFilter(t *testing.T) {
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "src", "a.go"), []byte("// @MX:TODO scoped\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "outside.go"), []byte("// @MX:TODO outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler()
	lsp := handler.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_lsp_check"}})
	if !lsp.OK || lsp.Message != "LSP diagnostics completed" {
		t.Fatalf("unexpected lsp response: %#v", lsp)
	}
	mx := handler.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_mx_scan", "payload": map[string]any{"path": "src"}}})
	if !mx.OK {
		t.Fatalf("mx response failed: %#v", mx)
	}
	counts := mx.Data["counts"].(map[string]int)
	if counts["TODO"] != 1 {
		t.Fatalf("expected filtered TODO count 1, got %#v", counts)
	}
}
