package bridge

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/modu-ai/moai-adk/internal/agentruntime"
	"github.com/modu-ai/moai-adk/internal/kernel"
	"github.com/modu-ai/moai-adk/internal/policy"
	"github.com/modu-ai/moai-adk/internal/sessionstate"
	"github.com/modu-ai/moai-adk/internal/taskstore"
	"github.com/modu-ai/moai-adk/internal/teamruntime"
)

const ProtocolVersion = "moai.pi.bridge.v1"

// Handler processes bridge requests from Pi. It is intentionally small at first:
// deeper workflow implementations should be added behind this stable protocol.
type Handler struct{}

// NewHandler creates a bridge handler.
func NewHandler() *Handler { return &Handler{} }

// Handle dispatches a bridge request.
func (h *Handler) Handle(ctx context.Context, req Request) Response {
	if req.Kind == "" {
		return Failure("unknown", "missing_kind", "bridge request kind is required")
	}
	if ctx.Err() != nil {
		return Failure(req.Kind, "context_cancelled", ctx.Err().Error())
	}

	switch req.Kind {
	case "doctor":
		return h.handleDoctor(req)
	case "capabilities":
		return h.handleCapabilities(req)
	case "state":
		return Success(req.Kind, "MoAI UI state loaded", loadUIState(req.CWD, req.Payload))
	case "command":
		return h.handleCommand(req)
	case "event":
		return h.handleEvent(req)
	case "tool":
		return h.handleTool(req)
	default:
		return Failure(req.Kind, "unsupported_kind", fmt.Sprintf("unsupported bridge request kind %q", req.Kind))
	}
}

func (h *Handler) handleDoctor(req Request) Response {
	cwd := req.CWD
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	checks := map[string]any{
		"cwd":       cwd,
		"moaiDir":   pathExists(filepath.Join(cwd, ".moai")),
		"claudeDir": pathExists(filepath.Join(cwd, ".claude")),
		"piDir":     pathExists(filepath.Join(cwd, ".pi")),
		"protocol":  ProtocolVersion,
	}
	return Success("doctor", "Pi bridge doctor completed", checks)
}

func (h *Handler) handleCapabilities(req Request) Response {
	return Success(req.Kind, "MoAI Pi bridge capabilities", map[string]any{
		"protocol": ProtocolVersion,
		"kinds":    []string{"doctor", "capabilities", "command", "event", "tool"},
		"commands": []string{"plan", "run", "sync", "design", "db", "project", "fix", "loop", "mx", "feedback", "review", "clean", "codemaps", "coverage", "e2e", "gate", "brain"},
		"events":   supportedPiEvents(),
		"eventSupport": map[string]any{
			"native":      nativePiEvents(),
			"synthetic":   syntheticPiEvents(),
			"partial":     []string{"after_provider_response", "PermissionDenied", "Elicitation", "ElicitationResult"},
			"unsupported": unsupportedClaudeHookEvents(),
		},
		"tools": []string{"moai_ask_user", "moai_config_get", "moai_config_set", "moai_spec_status", "moai_spec_create", "moai_spec_update", "moai_spec_list", "moai_quality_gate", "moai_lsp_check", "moai_mx_scan", "moai_mx_update", "moai_memory_read", "moai_memory_write", "moai_context_search", "moai_worktree_create", "moai_worktree_merge", "moai_worktree_cleanup", "moai_task_create", "moai_task_update", "moai_task_list", "moai_task_get", "moai_agent_invoke", "moai_team_run"},
	})
}

func (h *Handler) handleCommand(req Request) Response {
	name, _ := req.Payload["command"].(string)
	if name == "" {
		return Failure(req.Kind, "missing_command", "payload.command is required")
	}
	args, _ := req.Payload["args"].(string)
	result, err := kernel.New().ExecuteCommand(context.Background(), kernel.CommandRequest{Command: name, Args: args, CWD: req.CWD, Session: mapFromAny(req.Session)})
	if err != nil {
		return Failure(req.Kind, "command_failed", err.Error())
	}
	return Success(req.Kind, "Command executed by MoAI Runtime Kernel", map[string]any{
		"commandResult": result,
		"uiState":       result.UIState,
	})
}

func (h *Handler) handleEvent(req Request) Response {
	name, _ := req.Payload["event"].(string)
	if name == "" {
		return Failure(req.Kind, "missing_event", "payload.event is required")
	}
	toolName, _ := req.Payload["toolName"].(string)
	input, _ := req.Payload["input"].(map[string]any)
	_ = persistSessionEvent(req, name)
	result := policy.NewEvaluator().Evaluate(policy.EventRequest{
		Event:     name,
		ToolName:  toolName,
		ToolInput: input,
		CWD:       req.CWD,
		Payload:   req.Payload,
	})
	return Success(req.Kind, "Event policy evaluated", map[string]any{
		"event":    name,
		"decision": result.Decision,
		"data":     result.Data,
	})
}

func (h *Handler) handleTool(req Request) Response {
	name, _ := req.Payload["tool"].(string)
	if name == "" {
		return Failure(req.Kind, "missing_tool", "payload.tool is required")
	}

	switch name {
	case "moai_config_get":
		return h.handleConfigGet(req)
	case "moai_config_set":
		return h.handleConfigSet(req)
	case "moai_spec_status", "moai_spec_list":
		return h.handleSpecStatus(req)
	case "moai_spec_create":
		return h.handleSpecCreate(req)
	case "moai_spec_update":
		return h.handleSpecUpdate(req)
	case "moai_quality_gate":
		return Success(req.Kind, "Quality checks completed", runQualityChecks(req.CWD))
	case "moai_lsp_check":
		return Success(req.Kind, "LSP diagnostics completed", runLSPCheck(req.CWD))
	case "moai_mx_scan":
		return Success(req.Kind, "MX scan completed", scanMX(req.CWD, mxFiltersFromPayload(nestedPayload(req))...))
	case "moai_mx_update":
		return h.handleMXUpdate(req)
	case "moai_memory_read":
		return h.handleMemoryRead(req)
	case "moai_memory_write":
		return h.handleMemoryWrite(req)
	case "moai_context_search":
		return h.handleContextSearch(req)
	case "moai_worktree_create":
		return h.handleWorktreeCreate(req)
	case "moai_worktree_merge":
		return h.handleWorktreeMerge(req)
	case "moai_worktree_cleanup":
		return h.handleWorktreeCleanup(req)
	case "moai_task_create":
		return h.handleTaskCreate(req)
	case "moai_task_update":
		return h.handleTaskUpdate(req)
	case "moai_task_list":
		return h.handleTaskList(req)
	case "moai_task_get":
		return h.handleTaskGet(req)
	case "moai_agent_invoke":
		return h.handleAgentInvoke(req)
	case "moai_team_run":
		return h.handleTeamRun(req)
	}

	return Success(req.Kind, "Tool accepted by Pi bridge foundation", map[string]any{
		"tool":   name,
		"status": "bridge-foundation-ready",
	})
}

func (h *Handler) handleConfigGet(req Request) Response {
	payload := nestedPayload(req)
	section, _ := payload["section"].(string)
	configDir := filepath.Join(req.CWD, ".moai", "config", "sections")
	if section == "" {
		entries, err := os.ReadDir(configDir)
		if err != nil {
			return Failure(req.Kind, "config_list_failed", err.Error())
		}
		sections := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			sections = append(sections, strings.TrimSuffix(entry.Name(), ".yaml"))
		}
		return Success(req.Kind, "Config sections loaded", map[string]any{"sections": sections})
	}
	path := filepath.Join(configDir, section+".yaml")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Failure(req.Kind, "config_read_failed", err.Error())
	}
	return Success(req.Kind, "Config section loaded", map[string]any{
		"section": section,
		"path":    path,
		"content": string(bytes),
	})
}

func (h *Handler) handleConfigSet(req Request) Response {
	payload := nestedPayload(req)
	section, _ := payload["section"].(string)
	content, _ := payload["content"].(string)
	if section == "" || content == "" {
		return Failure(req.Kind, "missing_config_input", "payload.section and payload.content are required")
	}
	path := filepath.Join(req.CWD, ".moai", "config", "sections", section+".yaml")
	if isProtectedProjectPath(req.CWD, path) {
		return Failure(req.Kind, "protected_path", "refusing to write protected config path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Failure(req.Kind, "config_write_failed", err.Error())
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return Failure(req.Kind, "config_write_failed", err.Error())
	}
	return Success(req.Kind, "Config section updated", map[string]any{"section": section, "path": path})
}

func (h *Handler) handleSpecStatus(req Request) Response {
	payload := nestedPayload(req)
	specID, _ := payload["specId"].(string)
	specDir := filepath.Join(req.CWD, ".moai", "specs")
	if specID == "" {
		entries, err := os.ReadDir(specDir)
		if err != nil {
			return Failure(req.Kind, "spec_list_failed", err.Error())
		}
		specs := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "SPEC-") {
				continue
			}
			specs = append(specs, map[string]any{"id": entry.Name(), "path": filepath.Join(specDir, entry.Name())})
		}
		return Success(req.Kind, "SPEC list loaded", map[string]any{"specs": specs})
	}
	path := filepath.Join(specDir, specID)
	entries, err := os.ReadDir(path)
	if err != nil {
		return Failure(req.Kind, "spec_read_failed", err.Error())
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return Success(req.Kind, "SPEC status loaded", map[string]any{"specId": specID, "path": path, "files": files})
}

func (h *Handler) handleSpecCreate(req Request) Response {
	payload := nestedPayload(req)
	args, _ := payload["description"].(string)
	if args == "" {
		args, _ = payload["args"].(string)
	}
	result, err := kernel.New().ExecuteCommand(context.Background(), kernel.CommandRequest{Command: "plan", Args: args, CWD: req.CWD, Session: mapFromAny(req.Session)})
	if err != nil {
		return Failure(req.Kind, "spec_create_failed", err.Error())
	}
	return Success(req.Kind, "SPEC created", map[string]any{"commandResult": result, "specId": result.Data["specId"]})
}

func (h *Handler) handleSpecUpdate(req Request) Response {
	payload := nestedPayload(req)
	specID, _ := payload["specId"].(string)
	file, _ := payload["file"].(string)
	content, _ := payload["content"].(string)
	if specID == "" || file == "" || content == "" {
		return Failure(req.Kind, "missing_spec_update_input", "payload.specId, payload.file, and payload.content are required")
	}
	if strings.Contains(file, "..") || filepath.IsAbs(file) {
		return Failure(req.Kind, "invalid_spec_file", "payload.file must be a relative file name")
	}
	if file == "spec.md" && !strings.Contains(content, "## HISTORY") {
		return Failure(req.Kind, "spec_lint_failed", "spec.md must contain ## HISTORY")
	}
	path := filepath.Join(req.CWD, ".moai", "specs", specID, file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Failure(req.Kind, "spec_update_failed", err.Error())
	}
	if file == "status.json" && !strings.Contains(content, "\"phase\"") {
		return Failure(req.Kind, "spec_status_invalid", "status.json must include phase")
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return Failure(req.Kind, "spec_update_failed", err.Error())
	}
	appendSpecHistory(req.CWD, specID, "Updated "+file)
	return Success(req.Kind, "SPEC updated", map[string]any{"specId": specID, "path": path, "validated": true})
}

func (h *Handler) handleMXUpdate(req Request) Response {
	payload := nestedPayload(req)
	pathRaw, _ := payload["path"].(string)
	annotation, _ := payload["annotation"].(string)
	if pathRaw == "" || annotation == "" {
		return Failure(req.Kind, "missing_mx_update_input", "payload.path and payload.annotation are required")
	}
	path := filepath.Join(req.CWD, filepath.Clean(pathRaw))
	if isProtectedProjectPath(req.CWD, path) {
		return Failure(req.Kind, "protected_path", "refusing to update protected path")
	}
	if !strings.Contains(annotation, "@MX:") {
		return Failure(req.Kind, "invalid_mx_annotation", "annotation must contain @MX: tag")
	}
	line := intFromAny(payload["line"])
	mode, _ := payload["mode"].(string)
	if mode == "" {
		mode = "append"
	}
	if err := writeMXAnnotation(path, annotation, line, mode); err != nil {
		return Failure(req.Kind, "mx_update_failed", err.Error())
	}
	return Success(req.Kind, "MX annotation updated", map[string]any{"path": path, "line": line, "mode": mode})
}

func (h *Handler) handleMemoryRead(req Request) Response {
	payload := nestedPayload(req)
	name, _ := payload["name"].(string)
	scope, _ := payload["scope"].(string)
	if scope == "" {
		scope = "project"
	}
	if name == "" {
		name = "project.md"
	}
	path := memoryPath(req.CWD, scope, name)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Failure(req.Kind, "memory_read_failed", err.Error())
	}
	return Success(req.Kind, "Memory loaded", map[string]any{"name": name, "scope": scope, "path": path, "content": string(bytes)})
}

func (h *Handler) handleMemoryWrite(req Request) Response {
	payload := nestedPayload(req)
	name, _ := payload["name"].(string)
	scope, _ := payload["scope"].(string)
	content, _ := payload["content"].(string)
	if scope == "" {
		scope = "project"
	}
	if name == "" || content == "" {
		return Failure(req.Kind, "missing_memory_input", "payload.name and payload.content are required")
	}
	path := memoryPath(req.CWD, scope, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Failure(req.Kind, "memory_write_failed", err.Error())
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return Failure(req.Kind, "memory_write_failed", err.Error())
	}
	return Success(req.Kind, "Memory written", map[string]any{"name": name, "scope": scope, "path": path})
}

func (h *Handler) handleContextSearch(req Request) Response {
	payload := nestedPayload(req)
	query, _ := payload["query"].(string)
	if strings.TrimSpace(query) == "" {
		return Failure(req.Kind, "missing_query", "payload.query is required")
	}
	matches := searchTextFiles(req.CWD, query, 20)
	return Success(req.Kind, "Context search completed", map[string]any{"query": query, "matches": matches})
}

func (h *Handler) handleWorktreeCreate(req Request) Response {
	payload := nestedPayload(req)
	agent, _ := payload["agent"].(string)
	if agent == "" {
		agent = "moai"
	}
	opts := worktreeOptionsFromAny(payload)
	opts.Enabled = true
	opts.Keep = true
	path, branch, cleanup, err := agentruntime.PrepareWorktree(context.Background(), req.CWD, agent, opts)
	if err != nil {
		return Failure(req.Kind, "worktree_create_failed", err.Error())
	}
	_ = cleanup
	return Success(req.Kind, "Worktree created", map[string]any{"path": path, "branch": branch})
}

func (h *Handler) handleWorktreeMerge(req Request) Response {
	payload := nestedPayload(req)
	branch, _ := payload["branch"].(string)
	path, _ := payload["path"].(string)
	execute, _ := payload["execute"].(bool)
	plan := []string{"git status --short", "git diff --stat", "git merge --no-ff " + branch}
	if !execute {
		return Success(req.Kind, "Worktree merge handoff prepared", map[string]any{"branch": branch, "path": path, "manual": true, "mergePlan": plan, "rollback": "git merge --abort or git reset --hard HEAD"})
	}
	if branch == "" {
		return Failure(req.Kind, "missing_branch", "payload.branch is required when execute=true")
	}
	cmd := exec.CommandContext(context.Background(), "git", "merge", "--no-ff", branch)
	cmd.Dir = req.CWD
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Failure(req.Kind, "worktree_merge_failed", string(out)+err.Error())
	}
	return Success(req.Kind, "Worktree merged", map[string]any{"branch": branch, "output": truncateString(string(out), 4000), "rollback": "git reset --hard HEAD~1 if merge commit was created"})
}

func (h *Handler) handleWorktreeCleanup(req Request) Response {
	payload := nestedPayload(req)
	path, _ := payload["path"].(string)
	if path == "" {
		return Failure(req.Kind, "missing_worktree_path", "payload.path is required")
	}
	if err := os.RemoveAll(path); err != nil {
		return Failure(req.Kind, "worktree_cleanup_failed", err.Error())
	}
	return Success(req.Kind, "Worktree cleaned up", map[string]any{"path": path})
}

func (h *Handler) handleTaskCreate(req Request) Response {
	payload := nestedPayload(req)
	subject, _ := payload["subject"].(string)
	description, _ := payload["description"].(string)
	metadata, _ := payload["metadata"].(map[string]any)
	task, err := taskstore.New(req.CWD).Create(subject, description, metadata)
	if err != nil {
		return Failure(req.Kind, "task_create_failed", err.Error())
	}
	return Success(req.Kind, "Task created", map[string]any{"task": task})
}

func (h *Handler) handleTaskUpdate(req Request) Response {
	payload := nestedPayload(req)
	id, _ := payload["id"].(string)
	if id == "" {
		return Failure(req.Kind, "missing_task_id", "payload.id is required")
	}
	patch := taskstore.Patch{}
	if subject, ok := payload["subject"].(string); ok {
		patch.Subject = &subject
	}
	if description, ok := payload["description"].(string); ok {
		patch.Description = &description
	}
	if statusRaw, ok := payload["status"].(string); ok {
		status := taskstore.TaskStatus(statusRaw)
		patch.Status = &status
	}
	if metadata, ok := payload["metadata"].(map[string]any); ok {
		patch.Metadata = metadata
	}
	task, err := taskstore.New(req.CWD).Update(id, patch)
	if err != nil {
		return Failure(req.Kind, "task_update_failed", err.Error())
	}
	return Success(req.Kind, "Task updated", map[string]any{"task": task})
}

func (h *Handler) handleTaskList(req Request) Response {
	tasks, err := taskstore.New(req.CWD).List()
	if err != nil {
		return Failure(req.Kind, "task_list_failed", err.Error())
	}
	return Success(req.Kind, "Tasks loaded", map[string]any{"tasks": tasks})
}

func (h *Handler) handleTaskGet(req Request) Response {
	payload := nestedPayload(req)
	id, _ := payload["id"].(string)
	if id == "" {
		return Failure(req.Kind, "missing_task_id", "payload.id is required")
	}
	task, err := taskstore.New(req.CWD).Get(id)
	if err != nil {
		return Failure(req.Kind, "task_get_failed", err.Error())
	}
	return Success(req.Kind, "Task loaded", map[string]any{"task": task})
}

func (h *Handler) handleTeamRun(req Request) Response {
	payload := nestedPayload(req)
	action, _ := req.Payload["action"].(string)
	if action == "" {
		action, _ = payload["action"].(string)
	}
	if action == "" {
		action = "run"
	}
	profiles, err := teamruntime.LoadProfiles(req.CWD)
	if err != nil {
		return Failure(req.Kind, "team_profile_load_failed", err.Error())
	}
	store := teamruntime.NewStore(req.CWD)
	switch action {
	case "create":
		state, err := store.Create(teamTasksFromAny(payload["tasks"]), profiles)
		if err != nil {
			return Failure(req.Kind, "team_create_failed", err.Error())
		}
		return Success(req.Kind, "Team created", map[string]any{"team": state})
	case "status":
		state, err := teamStateFromPayload(store, payload)
		if err != nil {
			return Failure(req.Kind, "team_status_failed", err.Error())
		}
		return Success(req.Kind, "Team status loaded", map[string]any{"team": state})
	case "message":
		state, err := teamStateFromPayload(store, payload)
		if err != nil {
			return Failure(req.Kind, "team_message_failed", err.Error())
		}
		text, _ := payload["message"].(string)
		from, _ := payload["from"].(string)
		to, _ := payload["to"].(string)
		state.Messages = append(state.Messages, teamruntime.TeamMessage{From: firstNonEmptyString(from, "orchestrator"), To: to, Text: text, At: nowRFC3339()})
		state.Events = append(state.Events, teamruntime.TeamEvent{Type: "SendMessage", Role: to, At: nowRFC3339(), Data: map[string]any{"from": firstNonEmptyString(from, "orchestrator"), "message": text}})
		if err := store.Save(state); err != nil {
			return Failure(req.Kind, "team_message_save_failed", err.Error())
		}
		return Success(req.Kind, "Team message recorded", map[string]any{"team": state})
	case "review":
		state, err := teamStateFromPayload(store, payload)
		if err != nil {
			return Failure(req.Kind, "team_review_failed", err.Error())
		}
		role, _ := payload["role"].(string)
		accepted, _ := payload["accepted"].(bool)
		reason, _ := payload["reason"].(string)
		if err := store.ReviewCompletion(state, role, accepted, reason); err != nil {
			return Failure(req.Kind, "team_review_save_failed", err.Error())
		}
		return Success(req.Kind, "Team completion reviewed", map[string]any{"team": state})
	case "idle":
		state, err := teamStateFromPayload(store, payload)
		if err != nil {
			return Failure(req.Kind, "team_idle_failed", err.Error())
		}
		role, _ := payload["role"].(string)
		for i := range state.Members {
			if state.Members[i].Role == role || state.Members[i].Agent == role {
				state.Members[i].Status = "idle"
			}
		}
		if err := store.RecordEvent(state, teamruntime.TeamEvent{Type: "TeammateIdle", Role: role}); err != nil {
			return Failure(req.Kind, "team_idle_save_failed", err.Error())
		}
		return Success(req.Kind, "Team idle recorded", map[string]any{"team": state})
	case "delete":
		id, _ := payload["teamId"].(string)
		if id == "" {
			state, err := store.Latest()
			if err != nil {
				return Failure(req.Kind, "team_delete_failed", err.Error())
			}
			id = state.ID
		}
		if err := store.Delete(id); err != nil {
			return Failure(req.Kind, "team_delete_failed", err.Error())
		}
		return Success(req.Kind, "Team deleted", map[string]any{"teamId": id})
	case "run":
		state, err := store.Create(teamTasksFromAny(payload["tasks"]), profiles)
		if err != nil {
			return Failure(req.Kind, "team_create_failed", err.Error())
		}
		result, err := teamruntime.NewCoordinator(nil, profiles).Run(context.Background(), state.Tasks, req.CWD)
		if err != nil {
			state.Status = "failed"
			_ = store.Save(state)
			return Failure(req.Kind, "team_run_failed", err.Error())
		}
		result.TeamID = state.ID
		state.Status = "completed"
		for _, r := range result.Results {
			state.Results = append(state.Results, &teamruntime.AgentSummary{Agent: r.Agent, ResolvedAgent: r.ResolvedAgent, Status: string(r.Status), Summary: r.Summary, Error: r.Error, WorktreePath: r.WorktreePath, Branch: r.Branch})
			for i := range state.Members {
				if state.Members[i].Agent == r.Agent {
					state.Members[i].Status = string(r.Status)
					state.Members[i].WorktreePath = r.WorktreePath
					state.Members[i].Branch = r.Branch
					state.Events = append(state.Events, teamruntime.TeamEvent{Type: "SubagentStop", Role: state.Members[i].Role, Agent: r.Agent, At: nowRFC3339(), Data: map[string]any{"status": string(r.Status)}})
					if r.Status == agentruntime.StatusSuccess {
						state.Events = append(state.Events, teamruntime.TeamEvent{Type: "TaskCompleted", Role: state.Members[i].Role, Agent: r.Agent, At: nowRFC3339()})
					}
				}
			}
		}
		if err := store.Save(state); err != nil {
			return Failure(req.Kind, "team_save_failed", err.Error())
		}
		return Success(req.Kind, "Team run completed", map[string]any{"result": result, "team": state})
	default:
		return Failure(req.Kind, "unsupported_team_action", "unsupported moai_team_run action: "+action)
	}
}

func (h *Handler) handleAgentInvoke(req Request) Response {
	payload := nestedPayload(req)
	action, _ := req.Payload["action"].(string)
	if action == "" {
		action, _ = payload["action"].(string)
	}
	if action == "" || action == "list" {
		defs, err := agentruntime.Discover(req.CWD)
		if err != nil {
			return Failure(req.Kind, "agent_discovery_failed", err.Error())
		}
		return Success(req.Kind, "Agents loaded", map[string]any{"agents": defs})
	}
	if action == "parallel" {
		requests := agentRequestsFromAny(payload["tasks"], req.CWD)
		results, err := agentruntime.NewCoordinator(agentruntime.NewPiWorker()).InvokeParallel(context.Background(), requests)
		if err != nil {
			return Failure(req.Kind, "agent_parallel_failed", err.Error())
		}
		return Success(req.Kind, "Parallel agent invocation completed", map[string]any{"results": results})
	}
	if action == "chain" {
		requests := agentRequestsFromAny(payload["chain"], req.CWD)
		results, err := agentruntime.NewCoordinator(agentruntime.NewPiWorker()).InvokeChain(context.Background(), requests)
		if err != nil {
			return Failure(req.Kind, "agent_chain_failed", err.Error())
		}
		return Success(req.Kind, "Chain agent invocation completed", map[string]any{"results": results})
	}

	agentName, _ := payload["agent"].(string)
	if agentName == "" {
		return Failure(req.Kind, "missing_agent", "payload.agent is required for agent invocation")
	}
	if action != "run" {
		return Success(req.Kind, "Agent invocation accepted by Pi bridge foundation", map[string]any{
			"agent":  agentName,
			"status": "agent-runtime-foundation-ready",
			"next":   "use action=run to execute the agent",
		})
	}
	task, _ := payload["task"].(string)
	model, _ := payload["model"].(string)
	timeoutSeconds := intFromAny(payload["timeoutSeconds"])
	tools := stringsFromAny(payload["tools"])
	result, err := agentruntime.NewPiWorker().Invoke(context.Background(), agentruntime.Request{
		Agent:          agentName,
		Task:           task,
		CWD:            req.CWD,
		Model:          model,
		Tools:          tools,
		TimeoutSeconds: timeoutSeconds,
		Worktree:       worktreeOptionsFromAny(payload["worktree"]),
	})
	if err != nil {
		return Failure(req.Kind, "agent_invoke_failed", err.Error())
	}
	return Success(req.Kind, "Agent invocation completed", map[string]any{"result": result})
}

func persistSessionEvent(req Request, eventName string) error {
	store := sessionstate.New(req.CWD)
	state, err := store.Load()
	if err != nil {
		return err
	}
	if req.Session != nil {
		state.SessionFile = req.Session.File
		state.LeafID = req.Session.LeafID
		state.Mode = req.Session.Mode
	}
	ensureSessionStartedAt(state, eventName)
	state.LastEvent = eventName
	if specID, _ := req.Payload["activeSpecId"].(string); specID != "" {
		state.ActiveSpecID = specID
	}
	state.Data["lastPayload"] = req.Payload
	state.Data["lastEventAt"] = nowRFC3339()
	incrementEventCount(state.Data, eventName)
	appendRecentEvent(state.Data, map[string]any{"event": eventName, "payload": req.Payload, "at": state.Data["lastEventAt"]})
	applyLifecycleState(state.Data, eventName, req.Payload)
	return store.Save(state)
}

func nativePiEvents() []string {
	return []string{
		"session_start",
		"session_shutdown",
		"session_before_compact",
		"session_compact",
		"input",
		"before_agent_start",
		"agent_start",
		"agent_end",
		"turn_start",
		"turn_end",
		"tool_call",
		"tool_result",
		"after_provider_response",
	}
}

func syntheticPiEvents() []string {
	return []string{
		"StopFailure",
		"ConfigChange",
		"CwdChanged",
		"FileChanged",
		"SubagentStart",
		"SubagentStop",
		"TaskCompleted",
		"TeammateIdle",
		"WorktreeCreate",
		"WorktreeRemove",
	}
}

func supportedPiEvents() []string {
	return append(append([]string{}, nativePiEvents()...), syntheticPiEvents()...)
}

func unsupportedClaudeHookEvents() []string {
	return []string{}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func ensureSessionStartedAt(state *sessionstate.State, eventName string) {
	if state.Data == nil {
		state.Data = map[string]any{}
	}
	if existing := int64FromAny(state.Data["sessionStartedAt"]); existing > 0 {
		return
	}
	if startedAt := sessionStartedAtFromSessionFile(state.SessionFile); startedAt > 0 {
		state.Data["sessionStartedAt"] = startedAt
		return
	}
	if eventName == "session_start" {
		state.Data["sessionStartedAt"] = time.Now().UTC().UnixMilli()
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func incrementEventCount(data map[string]any, eventName string) {
	counts, _ := data["eventCounts"].(map[string]any)
	if counts == nil {
		counts = map[string]any{}
		data["eventCounts"] = counts
	}
	current := 0
	switch value := counts[eventName].(type) {
	case float64:
		current = int(value)
	case int:
		current = value
	}
	counts[eventName] = current + 1
}

func appendRecentEvent(data map[string]any, event map[string]any) {
	var recent []any
	if raw, ok := data["recentEvents"].([]any); ok {
		recent = append(recent, raw...)
	}
	recent = append(recent, event)
	if len(recent) > 20 {
		recent = recent[len(recent)-20:]
	}
	data["recentEvents"] = recent
}

func applyLifecycleState(data map[string]any, eventName string, payload map[string]any) {
	switch eventName {
	case "agent_start":
		data["agentActive"] = true
	case "agent_end":
		data["agentActive"] = false
	case "turn_start", "turn_end":
		if turnIndex, ok := payload["turnIndex"]; ok {
			data["turnIndex"] = turnIndex
		}
	case "tool_call", "tool_result":
		if toolName, _ := payload["toolName"].(string); toolName != "" {
			data["lastToolName"] = toolName
		}
		if isError, ok := payload["isError"].(bool); ok {
			data["lastToolError"] = isError
		}
	case "session_before_compact":
		data["compactionPending"] = true
	case "session_compact":
		data["compactionPending"] = false
		data["lastCompactedAt"] = nowRFC3339()
	case "StopFailure":
		data["lastStopFailure"] = payload
	case "ConfigChange":
		data["lastConfigChange"] = payload
	case "CwdChanged":
		data["lastCwdChange"] = payload
	case "FileChanged":
		data["lastFileChange"] = payload
	case "PermissionDenied":
		data["lastPermissionDenied"] = payload
	case "Elicitation":
		data["elicitationPending"] = true
		data["lastElicitation"] = payload
	case "ElicitationResult":
		data["elicitationPending"] = false
		data["lastElicitationResult"] = payload
	case "SubagentStart", "SubagentStop", "TaskCompleted", "TeammateIdle", "WorktreeCreate", "WorktreeRemove":
		data["lastSyntheticLifecycle"] = payload
	}
}

type contextMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func searchTextFiles(cwd, query string, limit int) []contextMatch {
	var matches []contextMatch
	_ = filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(matches) >= limit {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "target" || d.Name() == ".moai-worktrees" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil || looksBinary(bytes) {
			return nil
		}
		for i, line := range strings.Split(string(bytes), "\n") {
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				matches = append(matches, contextMatch{Path: relativePath(cwd, path), Line: i + 1, Text: truncateString(strings.TrimSpace(line), 240)})
				if len(matches) >= limit {
					break
				}
			}
		}
		return nil
	})
	return matches
}

func writeMXAnnotation(path, annotation string, line int, mode string) error {
	if mode == "insert" && line > 0 {
		bytes, _ := os.ReadFile(path)
		lines := strings.Split(string(bytes), "\n")
		idx := line - 1
		if idx < 0 {
			idx = 0
		}
		if idx > len(lines) {
			idx = len(lines)
		}
		lines = append(lines[:idx], append([]string{annotation}, lines[idx:]...)...)
		return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n" + annotation + "\n")
	return err
}

func memoryPath(cwd, scope, name string) string {
	scope = filepath.Clean(scope)
	name = filepath.Clean(name)
	if scope == "." || strings.Contains(scope, "..") {
		scope = "project"
	}
	return filepath.Join(cwd, ".moai", "memory", scope, name)
}

func appendSpecHistory(cwd, specID, entry string) {
	path := filepath.Join(cwd, ".moai", "specs", specID, "history.log")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(nowRFC3339()+" "+entry+"\n"), 0o644)
}

func isProtectedProjectPath(cwd, path string) bool {
	rel := relativePath(cwd, path)
	clean := filepath.ToSlash(filepath.Clean(rel))
	return clean == ".env" || clean == ".env.local" || strings.HasPrefix(clean, ".git/") || strings.HasPrefix(clean, "node_modules/") || clean == ".moai/config/sections/user.yaml"
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return nil
}

func nestedPayload(req Request) map[string]any {
	if payload, ok := req.Payload["payload"].(map[string]any); ok && payload != nil {
		return payload
	}
	return req.Payload
}

func loadPiCommandPrompt(cwd, name, args string) (string, string, error) {
	path := findCommandPath(cwd, name)
	if path == "" {
		if name == "default" {
			return "Load and follow the moai skill for this request:\n\n" + args, "", nil
		}
		return "", "", fmt.Errorf("MoAI command template not found for %q", name)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", path, err
	}
	body := stripMarkdownFrontmatter(string(bytes))
	body = strings.ReplaceAll(body, "$ARGUMENTS", args)
	body = strings.ReplaceAll(body, "Use Skill(\"moai\") with arguments:", "Load and follow the moai skill with arguments:")
	return body, path, nil
}

func findCommandPath(cwd, name string) string {
	current := cwd
	for {
		candidate := filepath.Join(current, ".claude", "commands", "moai", name+".md")
		if pathExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func stripMarkdownFrontmatter(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return strings.TrimSpace(content)
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return strings.TrimSpace(content)
	}
	return strings.TrimSpace(rest[idx+len("\n---\n"):])
}

func teamStateFromPayload(store *teamruntime.Store, payload map[string]any) (*teamruntime.TeamState, error) {
	id, _ := payload["teamId"].(string)
	if id != "" {
		return store.Get(id)
	}
	return store.Latest()
}

func teamTasksFromAny(value any) []teamruntime.Task {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]teamruntime.Task, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		agent, _ := m["agent"].(string)
		task, _ := m["task"].(string)
		result = append(result, teamruntime.Task{Role: role, Agent: agent, Task: task, OwnedPatterns: stringsFromAny(m["ownedPatterns"])})
	}
	return result
}

func agentRequestsFromAny(value any, cwd string) []agentruntime.Request {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	requests := make([]agentruntime.Request, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		agent, _ := m["agent"].(string)
		task, _ := m["task"].(string)
		model, _ := m["model"].(string)
		requests = append(requests, agentruntime.Request{
			Agent:          agent,
			Task:           task,
			CWD:            cwd,
			Model:          model,
			Tools:          stringsFromAny(m["tools"]),
			TimeoutSeconds: intFromAny(m["timeoutSeconds"]),
			Worktree:       worktreeOptionsFromAny(m["worktree"]),
		})
	}
	return requests
}

func worktreeOptionsFromAny(value any) agentruntime.WorktreeOptions {
	m, ok := value.(map[string]any)
	if !ok {
		return agentruntime.WorktreeOptions{}
	}
	enabled, _ := m["enabled"].(bool)
	baseDir, _ := m["baseDir"].(string)
	branch, _ := m["branch"].(string)
	keep, _ := m["keep"].(bool)
	return agentruntime.WorktreeOptions{Enabled: enabled, BaseDir: baseDir, Branch: branch, Keep: keep}
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func stringsFromAny(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}
	return result
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
