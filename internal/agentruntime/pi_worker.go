package agentruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const DefaultInvocationTimeout = 10 * time.Minute

// PiWorker invokes MoAI agents by spawning isolated Pi JSON-mode subprocesses.
type PiWorker struct {
	PiBinary string
	Now      func() time.Time
}

// NewPiWorker creates a Pi subprocess worker.
func NewPiWorker() *PiWorker {
	bin := os.Getenv("PI_BIN")
	if bin == "" {
		bin = "pi"
	}
	return &PiWorker{PiBinary: bin, Now: func() time.Time { return time.Now().UTC() }}
}

// Invoke runs a single agent in an isolated Pi process.
func (w *PiWorker) Invoke(ctx context.Context, req Request) (*Result, error) {
	if strings.TrimSpace(req.Agent) == "" {
		return nil, fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(req.Task) == "" {
		return nil, fmt.Errorf("task is required")
	}
	cwd := req.CWD
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	worktreePath, branch, cleanupWorktree, err := PrepareWorktree(ctx, cwd, req.Agent, req.Worktree)
	if err != nil {
		return nil, err
	}
	defer cleanupWorktree()
	if worktreePath != "" {
		cwd = worktreePath
	}
	defs, err := Discover(cwd)
	if err != nil {
		return nil, err
	}
	def, resolvedAgent := ResolveDefinition(defs, req.Agent)
	if def == nil {
		return &Result{Agent: req.Agent, ResolvedAgent: resolvedAgent, Status: StatusBlocked, Error: "agent definition not found", ErrorClass: "missing_agent", Recovery: "Run moai_agent_invoke action=list and use one of the returned agent names.", ExitCode: 1, StartedAt: w.now(), CompletedAt: w.now()}, nil
	}

	timeout := req.Timeout
	if timeout == 0 && req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = DefaultInvocationTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := w.now()
	promptPath, cleanup, err := writeAgentPrompt(def)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	args := []string{"--mode", "json", "-p", "--no-session", "--append-system-prompt", promptPath}
	model := MapModelAlias(firstNonEmpty(req.Model, def.Model))
	if model != "" {
		args = append(args, "--model", model)
	}
	tools := req.Tools
	if len(tools) == 0 {
		tools = normalizePiTools(def.Tools)
	}
	if len(tools) > 0 {
		args = append(args, "--tools", strings.Join(tools, ","))
	}
	args = append(args, "Task: "+req.Task)

	cmd := exec.CommandContext(ctx, w.PiBinary, args...)
	cmd.Dir = cwd
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	result := &Result{Agent: req.Agent, ResolvedAgent: def.Name, Status: StatusSuccess, ExitCode: 0, StartedAt: startedAt, Model: model, WorktreePath: worktreePath, Branch: branch, WorktreeKept: req.Worktree.Enabled && req.Worktree.Keep}
	if err := cmd.Start(); err != nil {
		classifyAgentError(result, err.Error())
		result.ExitCode = 1
		result.CompletedAt = w.now()
		return result, nil
	}

	scanErr := scanPiJSON(stdout, result)
	waitErr := cmd.Wait()
	result.CompletedAt = w.now()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
		result.Status = StatusAborted
		result.Error = ctx.Err().Error()
		result.ExitCode = 1
		return result, nil
	}
	if scanErr != nil && result.Error == "" {
		result.Error = scanErr.Error()
	}
	if waitErr != nil {
		classifyAgentError(result, strings.TrimSpace(firstNonEmpty(result.Error, stderr.String(), waitErr.Error())))
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, nil
	}
	if result.Output == "" {
		result.Output = result.Summary
	}
	if report, ok := ParseBlockerReport(result.Output); ok {
		result.Status = StatusBlocked
		result.Error = firstNonEmpty(report.Reason, "agent returned blocker report")
		result.ErrorClass = "blocker"
		result.Recovery = report.Recovery
	} else if blocked, marker := DetectBlocker(result.Output); blocked {
		result.Status = StatusBlocked
		result.Error = "agent returned blocker report: " + marker
		result.ErrorClass = "blocker"
	}
	return result, nil
}

func (w *PiWorker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now().UTC()
}

func writeAgentPrompt(def *Definition) (string, func(), error) {
	dir, err := os.MkdirTemp("", "moai-agent-*")
	if err != nil {
		return "", func() {}, err
	}
	path := filepath.Join(dir, def.Name+".md")
	content := agentExecutionContract(def)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", func() {}, err
	}
	return path, func() { _ = os.RemoveAll(dir) }, nil
}

func agentExecutionContract(def *Definition) string {
	body := def.SystemPrompt
	if body == "" {
		body = def.Description
	}
	return strings.TrimSpace(body) + "\n\n## MoAI Subagent Execution Contract\n\n- Do not ask the user questions directly. Return a blocker report instead.\n- Use only allowed tools from the invocation.\n- Preserve SPEC, task, memory, and MX context provided by the orchestrator.\n- On blocker, return JSON: {\"blocker\":true,\"reason\":\"...\",\"recovery\":\"...\"}.\n- On success, summarize changed files, tests, risks, and follow-up validation.\n"
}

func scanPiJSON(stdout io.Reader, result *Result) error {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		consumePiEvent(event, result)
	}
	return scanner.Err()
}

func consumePiEvent(event map[string]any, result *Result) {
	msg, ok := event["message"].(map[string]any)
	if !ok {
		return
	}
	result.Messages = append(result.Messages, msg)
	role, _ := msg["role"].(string)
	if role != "assistant" {
		return
	}
	result.Usage.Turns++
	if model, _ := msg["model"].(string); model != "" {
		result.Model = model
	}
	if usage, ok := msg["usage"].(map[string]any); ok {
		result.Usage.Input += intNumber(usage["input"])
		result.Usage.Output += intNumber(usage["output"])
		result.Usage.CacheRead += intNumber(usage["cacheRead"])
		result.Usage.CacheWrite += intNumber(usage["cacheWrite"])
		result.Usage.ContextTokens = intNumber(usage["totalTokens"])
		if cost, ok := usage["cost"].(map[string]any); ok {
			result.Usage.Cost += floatNumber(cost["total"])
		}
	}
	text := assistantText(msg)
	if text != "" {
		result.Output = text
		result.Summary = firstLine(text)
	}
	if stopReason, _ := msg["stopReason"].(string); stopReason == "error" {
		result.Status = StatusFailed
		if errorMessage, _ := msg["errorMessage"].(string); errorMessage != "" {
			result.Error = errorMessage
		}
	}
}

func assistantText(msg map[string]any) string {
	content, ok := msg["content"].([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := block["type"].(string); typ != "text" {
			continue
		}
		if text, _ := block["text"].(string); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func normalizePiTools(tools []string) []string {
	mapping := map[string]string{
		"Read": "read", "Write": "write", "Edit": "edit", "MultiEdit": "edit", "Bash": "bash",
		"Grep": "grep", "Glob": "find", "LS": "ls", "TodoWrite": "moai_task_update",
		"TaskCreate": "moai_task_create", "TaskUpdate": "moai_task_update", "TaskList": "moai_task_list", "TaskGet": "moai_task_get",
		"AskUserQuestion": "moai_ask_user",
	}
	seen := map[string]bool{}
	var result []string
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if mapped, ok := mapping[tool]; ok {
			tool = mapped
		} else {
			tool = strings.ToLower(tool)
		}
		if strings.HasPrefix(tool, "mcp__") || tool == "skill" || tool == "webfetch" || tool == "websearch" {
			continue
		}
		if tool != "" && !seen[tool] {
			seen[tool] = true
			result = append(result, tool)
		}
	}
	return result
}

func classifyAgentError(result *Result, message string) {
	result.Error = strings.TrimSpace(message)
	lower := strings.ToLower(result.Error)
	if strings.Contains(lower, "api key") || strings.Contains(lower, "auth") || strings.Contains(lower, "login") || strings.Contains(lower, "provider") || strings.Contains(lower, "not found") && strings.Contains(lower, "executable") {
		result.Status = StatusBlocked
		result.ErrorClass = "configuration"
		result.Recovery = "Configure Pi provider credentials or run pi login/configuration, then retry the agent invocation."
		return
	}
	result.Status = StatusFailed
	result.ErrorClass = "runtime"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstLine(text string) string {
	line := strings.TrimSpace(strings.Split(text, "\n")[0])
	if len(line) > 240 {
		return line[:240]
	}
	return line
}

func intNumber(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func floatNumber(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}
