package policy

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var destructiveRM = regexp.MustCompile(`\brm\s+-rf\s+(?:/|~|\*)`)

// Evaluator evaluates MoAI safety and workflow policies for any runtime.
type Evaluator struct{}

// NewEvaluator creates a policy evaluator.
func NewEvaluator() *Evaluator { return &Evaluator{} }

// Evaluate evaluates one event.
func (e *Evaluator) Evaluate(req EventRequest) Result {
	result := Result{Event: req.Event, Data: map[string]any{"policy": "moai-runtime"}}
	if req.Event == "tool_call" || req.Event == "PreToolUse" {
		return e.evaluateToolCall(req, result)
	}
	return result
}

func (e *Evaluator) evaluateToolCall(req EventRequest, result Result) Result {
	switch strings.ToLower(req.ToolName) {
	case "bash":
		command, _ := req.ToolInput["command"].(string)
		if destructiveRM.MatchString(command) {
			result.Decision = Decision{Block: true, Reason: "MoAI policy blocked destructive rm -rf command"}
			return result
		}
	case "write", "edit":
		path := pathFromInput(req.ToolInput)
		if protectedPath(path) {
			result.Decision = Decision{Block: true, Reason: fmt.Sprintf("MoAI policy blocked write to protected path: %s", path)}
			return result
		}
	}
	return result
}

func pathFromInput(input map[string]any) string {
	for _, key := range []string{"path", "file_path"} {
		if value, _ := input[key].(string); value != "" {
			return filepath.Clean(value)
		}
	}
	return ""
}

func protectedPath(path string) bool {
	if path == "" {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	protected := []string{".env", ".env.local", "node_modules", ".git", ".moai/config/sections/user.yaml"}
	for _, p := range protected {
		if clean == p || strings.HasPrefix(clean, p+"/") || strings.Contains(clean, "/"+p+"/") {
			return true
		}
	}
	return false
}
