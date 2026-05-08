// Package policy contains runtime-neutral MoAI event policy decisions.
package policy

// EventRequest is a runtime-neutral event payload.
type EventRequest struct {
	Event     string         `json:"event"`
	ToolName  string         `json:"toolName,omitempty"`
	ToolInput map[string]any `json:"toolInput,omitempty"`
	CWD       string         `json:"cwd,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// Decision is returned to runtime adapters.
type Decision struct {
	Block   bool   `json:"block,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// Result is the outcome of policy evaluation.
type Result struct {
	Event    string         `json:"event"`
	Decision Decision       `json:"decision"`
	Data     map[string]any `json:"data,omitempty"`
}
