package agentruntime

import (
	"encoding/json"
	"strings"
)

// BlockerReport is the normalized subagent blocker contract.
type BlockerReport struct {
	Blocker  bool   `json:"blocker"`
	Reason   string `json:"reason,omitempty"`
	Recovery string `json:"recovery,omitempty"`
}

// DetectBlocker marks structured subagent blocker reports.
func DetectBlocker(output string) (bool, string) {
	if report, ok := ParseBlockerReport(output); ok {
		return true, firstNonEmpty(report.Reason, "structured blocker report")
	}
	upper := strings.ToUpper(output)
	markers := []string{"BLOCKER:", "BLOCKED:", "NEEDS CLARIFICATION", "USER INPUT REQUIRED"}
	for _, marker := range markers {
		if strings.Contains(upper, marker) {
			return true, marker
		}
	}
	return false, ""
}

// ParseBlockerReport extracts a JSON blocker report from agent output.
func ParseBlockerReport(output string) (BlockerReport, bool) {
	for _, candidate := range []string{strings.TrimSpace(output), extractJSONBlock(output)} {
		if candidate == "" {
			continue
		}
		var report BlockerReport
		if err := json.Unmarshal([]byte(candidate), &report); err == nil && report.Blocker {
			return report, true
		}
	}
	return BlockerReport{}, false
}

func extractJSONBlock(output string) string {
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start < 0 || end <= start {
		return ""
	}
	return output[start : end+1]
}
