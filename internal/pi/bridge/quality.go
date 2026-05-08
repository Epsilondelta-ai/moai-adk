package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	slsp "github.com/modu-ai/moai-adk/internal/lsp/hook"
)

const qualityOutputLimit = 12000

type qualityCommand struct {
	Name     string   `json:"name"`
	Language string   `json:"language"`
	Command  []string `json:"command"`
	Required bool     `json:"required"`
	Reason   string   `json:"reason,omitempty"`
}

type qualityCommandResult struct {
	Name            string   `json:"name"`
	Language        string   `json:"language"`
	Command         []string `json:"command"`
	Status          string   `json:"status"`
	Required        bool     `json:"required"`
	ExitCode        int      `json:"exitCode,omitempty"`
	DurationMS      int64    `json:"durationMs,omitempty"`
	Output          string   `json:"output,omitempty"`
	OutputTruncated bool     `json:"outputTruncated,omitempty"`
	Reason          string   `json:"reason,omitempty"`
}

type textIssue struct {
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

// runQualityChecks executes language-aware quality gates and returns a stable,
// structured payload for Pi tools. Missing optional tools are skipped rather
// than treated as failures; missing required tools are reported as blockers.
func runQualityChecks(cwd string) map[string]any {
	checks := detectQualityChecks(cwd)
	results := make([]qualityCommandResult, 0, len(checks))
	languages := map[string]bool{}
	passed := true
	errors := 0
	warnings := 0
	skipped := 0

	if len(checks) == 0 {
		return map[string]any{
			"passed":    true,
			"warnings":  0,
			"errors":    0,
			"skipped":   1,
			"commands":  []qualityCommandResult{{Name: "quality detection", Command: []string{}, Status: "skipped", Reason: "no recognized project marker found"}},
			"languages": []string{},
		}
	}

	for _, check := range checks {
		if check.Language != "" {
			languages[check.Language] = true
		}
		result := executeQualityCommand(cwd, check)
		switch result.Status {
		case "passed":
		case "skipped":
			skipped++
			warnings++
			if check.Required {
				passed = false
				errors++
			}
		default:
			passed = false
			errors++
		}
		results = append(results, result)
	}

	return map[string]any{
		"passed":    passed,
		"warnings":  warnings,
		"errors":    errors,
		"skipped":   skipped,
		"languages": sortedMapKeys(languages),
		"commands":  results,
		"summary": map[string]any{
			"total":   len(results),
			"passed":  countCommandStatus(results, "passed"),
			"failed":  countCommandStatus(results, "failed"),
			"skipped": skipped,
		},
	}
}

func executeQualityCommand(cwd string, check qualityCommand) qualityCommandResult {
	result := qualityCommandResult{
		Name:     check.Name,
		Language: check.Language,
		Command:  check.Command,
		Required: check.Required,
		Reason:   check.Reason,
	}
	if len(check.Command) == 0 {
		result.Status = "skipped"
		result.Reason = "empty command"
		return result
	}
	if _, err := exec.LookPath(check.Command[0]); err != nil {
		result.Status = "skipped"
		if check.Required {
			result.Reason = fmt.Sprintf("required tool %q not found in PATH", check.Command[0])
		} else {
			result.Reason = fmt.Sprintf("optional tool %q not found in PATH", check.Command[0])
		}
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, check.Command[0], check.Command[1:]...)
	cmd.Dir = cwd
	start := time.Now()
	out, err := cmd.CombinedOutput()
	result.DurationMS = time.Since(start).Milliseconds()
	result.Output, result.OutputTruncated = truncateWithFlag(string(out), qualityOutputLimit)
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "failed"
		result.Reason = "command timed out after 2m"
		result.ExitCode = -1
		return result
	}
	if err != nil {
		result.Status = "failed"
		result.ExitCode = exitCode(err)
		return result
	}
	result.Status = "passed"
	return result
}

func detectQualityChecks(cwd string) []qualityCommand {
	var checks []qualityCommand
	if pathExists(filepath.Join(cwd, "go.mod")) {
		checks = append(checks,
			qualityCommand{Name: "go vet", Language: "go", Command: []string{"go", "vet", "./..."}, Required: true},
			qualityCommand{Name: "golangci-lint", Language: "go", Command: []string{"golangci-lint", "run"}, Required: false, Reason: "optional Go lint gate"},
			qualityCommand{Name: "go test", Language: "go", Command: []string{"go", "test", "./..."}, Required: true},
		)
	}
	if pathExists(filepath.Join(cwd, "package.json")) {
		scripts := packageScripts(filepath.Join(cwd, "package.json"))
		if scripts["lint"] {
			checks = append(checks, qualityCommand{Name: "npm run lint", Language: "node", Command: []string{"npm", "run", "lint"}, Required: true})
		}
		if scripts["typecheck"] {
			checks = append(checks, qualityCommand{Name: "npm run typecheck", Language: "node", Command: []string{"npm", "run", "typecheck"}, Required: true})
		}
		if scripts["test"] {
			checks = append(checks, qualityCommand{Name: "npm test", Language: "node", Command: []string{"npm", "test"}, Required: true})
		} else {
			checks = append(checks, qualityCommand{Name: "npm test", Language: "node", Command: []string{"npm", "test"}, Required: false, Reason: "package.json has no test script"})
		}
	}
	if pathExists(filepath.Join(cwd, "pyproject.toml")) || pathExists(filepath.Join(cwd, "pytest.ini")) {
		checks = append(checks,
			qualityCommand{Name: "ruff", Language: "python", Command: []string{"ruff", "check", "."}, Required: false, Reason: "optional Python lint gate"},
			qualityCommand{Name: "pytest", Language: "python", Command: []string{"pytest"}, Required: true},
		)
	}
	if pathExists(filepath.Join(cwd, "Cargo.toml")) {
		checks = append(checks,
			qualityCommand{Name: "cargo clippy", Language: "rust", Command: []string{"cargo", "clippy", "--all-targets", "--", "-D", "warnings"}, Required: true},
			qualityCommand{Name: "cargo test", Language: "rust", Command: []string{"cargo", "test"}, Required: true},
		)
	}
	return checks
}

func runLSPCheck(cwd string) map[string]any {
	provider := slsp.NewFallbackDiagnostics()
	files := diagnosticTargetFiles(cwd, 80)
	diagnostics := []map[string]any{}
	unavailable := []map[string]any{}
	counts := map[string]int{"errors": 0, "warnings": 0, "information": 0, "hints": 0}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, file := range files {
		diags, err := provider.RunFallback(ctx, file)
		if err != nil {
			unavailable = append(unavailable, map[string]any{"file": relativePath(cwd, file), "reason": err.Error()})
			continue
		}
		for _, diag := range diags {
			severity := string(diag.Severity)
			switch severity {
			case "error":
				counts["errors"]++
			case "warning":
				counts["warnings"]++
			case "information":
				counts["information"]++
			case "hint":
				counts["hints"]++
			}
			diagnostics = append(diagnostics, map[string]any{
				"file":     relativePath(cwd, file),
				"line":     diag.Range.Start.Line + 1,
				"column":   diag.Range.Start.Character + 1,
				"severity": severity,
				"code":     diag.Code,
				"source":   diag.Source,
				"message":  diag.Message,
				"range":    diag.Range,
			})
		}
	}
	return map[string]any{
		"passed":       counts["errors"] == 0,
		"supported":    len(files) > 0,
		"errors":       counts["errors"],
		"warnings":     counts["warnings"],
		"information":  counts["information"],
		"hints":        counts["hints"],
		"skipped":      len(unavailable),
		"diagnostics":  diagnostics,
		"unavailable":  unavailable,
		"filesChecked": len(files),
		"cwd":          cwd,
	}
}

func scanMX(cwd string, filters ...string) map[string]any {
	counts := map[string]int{"NOTE": 0, "WARN": 0, "ANCHOR": 0, "TODO": 0}
	validTags := map[string]bool{"NOTE": true, "WARN": true, "ANCHOR": true, "TODO": true, "REASON": true}
	malformed := []textIssue{}
	missingReasons := []textIssue{}
	filesWithTags := []string{}
	filesScanned := 0
	tagRE := regexp.MustCompile(`@MX:([A-Za-z_:-]+)`)
	malformedMXRE := regexp.MustCompile(`@MX(\b|[^:])`)

	_ = filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipMXDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !pathMatchesFilters(cwd, path, filters) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 1024*1024 {
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil || looksBinary(bytes) {
			return nil
		}
		filesScanned++
		rel := relativePath(cwd, path)
		text := string(bytes)
		lines := splitLines(text)
		foundInFile := false
		for idx, line := range lines {
			for _, match := range tagRE.FindAllStringSubmatchIndex(line, -1) {
				tag := line[match[2]:match[3]]
				upper := strings.ToUpper(tag)
				column := match[0] + 1
				if tag != upper || !validTags[upper] {
					malformed = append(malformed, textIssue{File: rel, Line: idx + 1, Column: column, Kind: "malformed", Message: "malformed or unknown MX tag", Tag: tag})
					continue
				}
				if _, ok := counts[upper]; ok {
					counts[upper]++
					foundInFile = true
				}
				if upper == "WARN" && !warnHasReason(lines, idx) {
					missingReasons = append(missingReasons, textIssue{File: rel, Line: idx + 1, Column: column, Kind: "missing_reason", Message: "@MX:WARN requires @MX:REASON on the same line or nearby following lines", Tag: "WARN"})
				}
			}
			if malformedMXRE.MatchString(line) && !strings.Contains(line, "@MX:") {
				malformed = append(malformed, textIssue{File: rel, Line: idx + 1, Column: strings.Index(line, "@MX") + 1, Kind: "malformed", Message: "MX marker must use @MX:TAG form"})
			}
		}
		if foundInFile {
			filesWithTags = append(filesWithTags, rel)
		}
		return nil
	})

	sort.Strings(filesWithTags)
	warnings := len(malformed) + len(missingReasons)
	return map[string]any{
		"passed":         warnings == 0,
		"files":          len(filesWithTags),
		"filesScanned":   filesScanned,
		"counts":         counts,
		"warnings":       warnings,
		"malformed":      malformed,
		"missingReasons": missingReasons,
		"filesWithTags":  filesWithTags,
		"filters":        filters,
	}
}

func diagnosticTargetFiles(cwd string, limit int) []string {
	var files []string
	_ = filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(files) >= limit {
			return nil
		}
		if d.IsDir() {
			if shouldSkipMXDir(d.Name()) || d.Name() == ".moai" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !isDiagnosticExtension(path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files
}

func isDiagnosticExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".pyi", ".ts", ".tsx", ".js", ".jsx", ".rs":
		return true
	default:
		return false
	}
}

func packageScripts(path string) map[string]bool {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return map[string]bool{}
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(bytes, &pkg); err != nil {
		return map[string]bool{}
	}
	result := map[string]bool{}
	for name, script := range pkg.Scripts {
		if strings.TrimSpace(script) != "" {
			result[name] = true
		}
	}
	return result
}

func mxFiltersFromPayload(payload map[string]any) []string {
	filters := []string{}
	if path, _ := payload["path"].(string); strings.TrimSpace(path) != "" {
		filters = append(filters, strings.TrimSpace(path))
	}
	if path, _ := payload["file"].(string); strings.TrimSpace(path) != "" {
		filters = append(filters, strings.TrimSpace(path))
	}
	if raw, ok := payload["paths"].([]any); ok {
		for _, item := range raw {
			if path, ok := item.(string); ok && strings.TrimSpace(path) != "" {
				filters = append(filters, strings.TrimSpace(path))
			}
		}
	}
	return filters
}

func shouldSkipMXDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".next", "dist", "build", "target", "vendor":
		return true
	default:
		return false
	}
}

func pathMatchesFilters(cwd, path string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	rel := relativePath(cwd, path)
	for _, filter := range filters {
		filter = filepath.Clean(filter)
		if filepath.IsAbs(filter) {
			if path == filter || strings.HasPrefix(path, filter+string(os.PathSeparator)) {
				return true
			}
			continue
		}
		if rel == filter || strings.HasPrefix(rel, filter+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func warnHasReason(lines []string, warnLine int) bool {
	for i := warnLine; i < len(lines) && i <= warnLine+2; i++ {
		if strings.Contains(lines[i], "@MX:REASON") {
			return true
		}
	}
	return false
}

func splitLines(text string) []string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 && text != "" {
		return []string{text}
	}
	return lines
}

func looksBinary(bytes []byte) bool {
	limit := len(bytes)
	if limit > 8000 {
		limit = 8000
	}
	for i := 0; i < limit; i++ {
		if bytes[i] == 0 {
			return true
		}
	}
	return false
}

func relativePath(cwd, path string) string {
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

func exitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func countCommandStatus(results []qualityCommandResult, status string) int {
	count := 0
	for _, result := range results {
		if result.Status == status {
			count++
		}
	}
	return count
}

func sortedMapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func truncateString(text string, limit int) string {
	truncated, _ := truncateWithFlag(text, limit)
	return truncated
}

func truncateWithFlag(text string, limit int) (string, bool) {
	if len(text) <= limit {
		return text, false
	}
	return text[len(text)-limit:], true
}
