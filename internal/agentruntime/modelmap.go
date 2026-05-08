package agentruntime

import "strings"

// MapModelAlias converts MoAI/Claude model aliases into Pi model selectors.
func MapModelAlias(model string) string {
	normalized := strings.TrimSpace(strings.ToLower(model))
	switch normalized {
	case "", "inherit":
		return ""
	case "haiku":
		return "claude-haiku-4-5"
	case "sonnet":
		return "claude-sonnet-4-5"
	case "opus":
		return "claude-opus-4-7"
	default:
		return model
	}
}
