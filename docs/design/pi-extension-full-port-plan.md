# MoAI-ADK Pi Extension Full Port Plan

Status: Reviewed and revised  
Date: 2026-05-08  
Scope: Feature-complete Pi runtime support for MoAI-ADK, not MVP-only compatibility

## 1. Review Verdict

The previous plan is directionally sound: MoAI-ADK can become a Pi package/extension while keeping Claude Code support. The plan needs one important refinement: the Pi port should not duplicate MoAI logic in TypeScript. The correct architecture is a shared MoAI Core with runtime adapters.

Final architectural decision:

```text
Pi TUI / Pi Agent Runtime
        ↓
MoAI Pi Extension, TypeScript adapter
        ↓
MoAI Bridge Protocol, JSON over process/stdout or future RPC
        ↓
MoAI Go Core
        ↓
.moai state, SPECs, quality gates, MX, memory, worktrees, git
```

This keeps the Go codebase as the source of truth and makes Pi an additional runtime surface, not a forked implementation.

## 2. The Three Critical Problems Are Solvable

### 2.1 Claude Code Sub-Agent Replacement

Solvable by implementing a MoAI Agent Runtime.

Recommended progression:

1. Use Pi subprocess workers for compatibility.
2. Add a typed MoAI Agent Runtime abstraction in Go.
3. Later support direct provider calls if Pi subprocess orchestration becomes too expensive or hard to control.

Pi already has a working subagent example that spawns isolated `pi --mode json -p --no-session` processes with custom system prompts, tool selection, streaming JSON event parsing, parallel mode, chain mode, and custom rendering. MoAI should adapt this pattern rather than invent a completely unrelated model.

Key design requirement: subagents must not ask the user directly. If a subagent needs clarification, it returns a structured blocker report to the orchestrator, and the Pi adapter asks the user via `ctx.ui`.

### 2.2 Claude Code Hooks Replacement

Solvable by mapping Claude Code hooks to Pi extension events and adding shims for events Pi does not natively expose.

Most hook behavior maps cleanly:

| Claude Code Hook | Pi Replacement |
|---|---|
| `SessionStart` | `session_start` |
| `SessionEnd` | `session_shutdown` |
| `UserPromptSubmit` | `input`, `before_agent_start` |
| `PreToolUse` | `tool_call` |
| `PostToolUse` | `tool_result` |
| `PostToolUseFailure` | `tool_result` with `isError` |
| `PreCompact` | `session_before_compact` |
| `PostCompact` | `session_compact` |
| `Notification` | `ctx.ui.notify()` |
| `PermissionRequest` | `tool_call` + `ctx.ui.confirm()` |
| `Stop` | `agent_end` |
| `StopFailure` | `after_provider_response`, `message_end`, `agent_end` synthesis |
| `SubagentStart` / `SubagentStop` | MoAI Agent Runtime events |
| `TeammateIdle` / `TaskCompleted` | MoAI Team Coordinator events |
| `WorktreeCreate` / `WorktreeRemove` | MoAI worktree lifecycle events |
| `ConfigChange` | file watcher + reload hook |
| `CwdChanged` | session/input cwd tracking + watcher |
| `FileChanged` | watcher or tool-result based detection |
| `Elicitation` / `ElicitationResult` | Pi UI interaction port |
| `PermissionDenied` | policy engine denial event |

Important refinement: hook logic should be moved into runtime-neutral Go policy modules where possible. The Pi extension should translate Pi events into MoAI event payloads and call the shared policy engine.

### 2.3 AskUserQuestion and Task Tool Replacement

Solvable by creating runtime-neutral ports plus Pi implementations.

Interaction:

```text
UserInteractionPort
├── select
├── confirm
├── input
├── editor
└── notify
```

Pi implementation:

```text
ctx.ui.select
ctx.ui.confirm
ctx.ui.input
ctx.ui.editor
ctx.ui.notify
ctx.ui.custom, when richer UI is needed
```

Task state:

```text
TaskPort
├── create
├── update
├── list
└── get
```

Persistence:

```text
.moai/tasks/*.json       project-level durable state
Pi custom session entries branch-aware session state
Tool result details       replayable branch reconstruction
```

Pi custom tools:

```text
moai_task_create
moai_task_update
moai_task_list
moai_task_get
moai_ask_user
```

## 3. Corrected Target Architecture

```text
MoAI-ADK Repository
├── Go Core
│   ├── config
│   ├── spec
│   ├── workflow
│   ├── hook policy engine
│   ├── quality gates
│   ├── LSP gates
│   ├── MX engine
│   ├── task store
│   ├── interaction ports
│   ├── agent runtime
│   └── team coordinator
│
├── Claude Code Adapter
│   ├── .claude/commands
│   ├── .claude/hooks
│   ├── .claude/agents
│   └── .claude/skills
│
└── Pi Adapter
    ├── .pi/extensions/moai/index.ts
    ├── .pi/extensions/moai/commands.ts
    ├── .pi/extensions/moai/events.ts
    ├── .pi/extensions/moai/tools.ts
    ├── .pi/extensions/moai/agents.ts
    ├── .pi/extensions/moai/tasks.ts
    ├── .pi/extensions/moai/ui.ts
    ├── .pi/extensions/moai/bridge.ts
    ├── .pi/skills
    ├── .pi/prompts
    └── .pi/themes
```

## 4. Non-Negotiable Design Principles

1. **Single source of truth:** business logic stays in Go Core, not duplicated in TypeScript.
2. **Runtime adapters only translate:** Claude Code and Pi adapters translate events, commands, UI, and tools.
3. **Feature parity over name parity:** if Pi lacks a Claude Code primitive, MoAI provides an equivalent behavior through its own runtime.
4. **Branch-aware persistence:** Pi sessions can branch; state must be reconstructable from `.moai` and Pi session entries.
5. **No direct user prompts from subagents:** all questions flow through the orchestrator UI port.
6. **Generated resources:** Pi skills/prompts/agents should be generated or synchronized from existing MoAI assets to avoid drift.
7. **Tool mutation safety:** custom file-mutating Pi tools must use Pi's file mutation queue or avoid direct file writes.
8. **Security parity:** dangerous command blocking, protected path rules, permission gates, and worktree isolation must match or exceed Claude Code behavior.

## 5. Workstream A — Runtime-Neutral Go Core

Add or formalize the following interfaces:

```go
type RuntimeAdapter interface {
    Name() string
    EmitEvent(ctx context.Context, event MoAIEvent) (*MoAIEventResult, error)
    Ask(ctx context.Context, question UserQuestion) (UserAnswer, error)
    RunAgent(ctx context.Context, request AgentRequest) (AgentResult, error)
    RunTask(ctx context.Context, request TaskRequest) (TaskResult, error)
}

type UserInteractionPort interface {
    Select(ctx context.Context, q SelectQuestion) (SelectAnswer, error)
    Confirm(ctx context.Context, q ConfirmQuestion) (bool, error)
    Input(ctx context.Context, q InputQuestion) (string, error)
    Editor(ctx context.Context, q EditorQuestion) (string, error)
    Notify(ctx context.Context, n Notification) error
}

type TaskPort interface {
    Create(ctx context.Context, task Task) (TaskID, error)
    Update(ctx context.Context, id TaskID, patch TaskPatch) error
    List(ctx context.Context, filter TaskFilter) ([]Task, error)
    Get(ctx context.Context, id TaskID) (Task, error)
}

type AgentRuntime interface {
    Invoke(ctx context.Context, request AgentRequest) (AgentResult, error)
    InvokeParallel(ctx context.Context, requests []AgentRequest) ([]AgentResult, error)
    InvokeChain(ctx context.Context, chain []AgentRequest) ([]AgentResult, error)
}
```

Required Go additions:

```text
internal/runtimeadapter
internal/interaction
internal/taskstore
internal/agentruntime
internal/teamruntime
internal/pi
```

Add a bridge command group:

```bash
moai pi bridge event
moai pi bridge command
moai pi bridge tool
moai pi bridge agent
moai pi bridge task
moai pi doctor
moai pi package
moai pi sync-resources
```

The bridge should use stable JSON request/response schemas. This lets the TypeScript Pi extension stay thin and testable.

## 6. Workstream B — Pi Package Layout

Add a Pi package manifest.

```json
{
  "name": "moai-adk-pi",
  "keywords": ["pi-package"],
  "pi": {
    "extensions": ["./.pi/extensions"],
    "skills": ["./.pi/skills"],
    "prompts": ["./.pi/prompts"],
    "themes": ["./.pi/themes"]
  },
  "peerDependencies": {
    "@earendil-works/pi-coding-agent": "*",
    "@earendil-works/pi-ai": "*",
    "@earendil-works/pi-tui": "*",
    "typebox": "*"
  }
}
```

Extension layout:

```text
.pi/extensions/moai/
├── index.ts
├── bridge.ts
├── commands.ts
├── events.ts
├── tools.ts
├── ui.ts
├── agents.ts
├── tasks.ts
├── renderers.ts
├── policy.ts
└── types.ts
```

## 7. Workstream C — Slash Command Parity

The user-facing command should remain:

```bash
/moai <subcommand> <args>
```

Pi command implementation:

```ts
pi.registerCommand("moai", {
  description: "MoAI workflow entrypoint",
  handler: async (args, ctx) => {
    await dispatchMoaiCommand(args, ctx);
  }
});
```

Required subcommands:

```text
plan
run
sync
design
db
project
fix
loop
mx
feedback
review
clean
codemaps
coverage
e2e
gate
brain
```

Each command path should call `moai pi bridge command` with:

```json
{
  "cwd": "...",
  "command": "plan",
  "args": "...",
  "session": { "file": "...", "branch": "...", "leafId": "..." },
  "uiMode": "interactive|rpc|json|print"
}
```

## 8. Workstream D — Pi Event Layer and Hook Policy Port

Pi extension event handlers should translate events to shared MoAI event payloads.

Required handlers:

```ts
pi.on("session_start", ...)
pi.on("session_shutdown", ...)
pi.on("input", ...)
pi.on("before_agent_start", ...)
pi.on("tool_call", ...)
pi.on("tool_result", ...)
pi.on("turn_start", ...)
pi.on("turn_end", ...)
pi.on("agent_start", ...)
pi.on("agent_end", ...)
pi.on("session_before_compact", ...)
pi.on("session_compact", ...)
pi.on("before_provider_request", ...)
pi.on("after_provider_response", ...)
```

Missing Claude Code events must be synthesized:

```text
FileChanged       file watcher + write/edit/tool_result observation
ConfigChange      watcher over .moai, .pi, .claude config paths
CwdChanged        compare ctx.cwd across event boundaries
PermissionDenied  emitted when MoAI policy blocks a tool
StopFailure       provider response + message stopReason/error synthesis
TaskCreated       emitted by TaskPort
SubagentStart     emitted by AgentRuntime
SubagentStop      emitted by AgentRuntime
WorktreeCreate    emitted by TeamCoordinator
WorktreeRemove    emitted by TeamCoordinator
```

## 9. Workstream E — Interaction and Task Replacement

Pi UI implementation details:

- Use `ctx.ui.select()` for structured option questions.
- Use `ctx.ui.confirm()` for permission gates.
- Use `ctx.ui.input()` for short text.
- Use `ctx.ui.editor()` for multi-line clarification or plan refinement.
- Use `ctx.ui.custom()` for multi-question Socratic interviews and progress dashboards.
- Check `ctx.hasUI` before interactive calls. In print/json modes, return a blocker requiring interactive execution or use provided defaults.

Task tools:

```ts
pi.registerTool({ name: "moai_task_create", ... })
pi.registerTool({ name: "moai_task_update", ... })
pi.registerTool({ name: "moai_task_list", ... })
pi.registerTool({ name: "moai_task_get", ... })
```

State reconstruction rule:

1. Load durable state from `.moai/tasks`.
2. Replay Pi session custom entries on the active branch.
3. Reconcile conflicts by timestamp and task version.
4. Persist every mutation to both durable state and session entry.

## 10. Workstream F — MoAI Agent Runtime

Agent definition sources:

```text
.claude/agents/moai/*.md       source of truth initially
.pi/agents/moai/*.md           generated/synchronized Pi copy, optional
```

Agent runtime capabilities:

```text
single agent invocation
parallel invocation
chain invocation
agent-specific tools
agent-specific model
token budget tracking
worktree isolation
structured result contract
blocker report contract
streamed progress updates
retry limit and loop prevention
```

Pi subprocess invocation baseline:

```bash
pi --mode json -p --no-session \
  --append-system-prompt /tmp/moai-agent-prompt.md \
  --tools read,bash,grep,find,ls,edit,write \
  "Task: ..."
```

For write-capable agents:

1. Create or assign an isolated worktree before invocation.
2. Set subprocess cwd to the worktree.
3. Enforce file ownership before merge.
4. Run quality gates before integration.
5. Merge through the MoAI worktree orchestrator.

Subagent result contract:

```json
{
  "agent": "expert-backend",
  "status": "success|failed|blocked",
  "summary": "...",
  "changedFiles": [],
  "blockers": [],
  "quality": {},
  "usage": {},
  "artifacts": []
}
```

## 11. Workstream G — Team Runtime

Team mode cannot depend on Claude Code Agent Teams. It must be implemented by MoAI.

Coordinator responsibilities:

```text
role profile resolution from .moai/config/sections/workflow.yaml
task graph construction
file ownership allocation
parallel worker spawning
worktree lifecycle
progress tracking
teammate idle detection
task completion validation
merge conflict detection
quality gate enforcement
review synthesis
```

Role policy:

| Role | Pi execution mode | Isolation |
|---|---|---|
| researcher | read-only subprocess | none |
| analyst | read-only subprocess | none |
| architect | read-only subprocess | none |
| reviewer | read-only subprocess | none |
| implementer | write-capable subprocess | worktree |
| tester | write-capable subprocess | worktree |
| designer | write-capable subprocess | worktree |

## 12. Workstream H — Skills, Prompts, Themes, and Renderers

Skills:

- Validate `.claude/skills/moai-*` against Pi's Agent Skills rules.
- Generate `.pi/skills` from source assets.
- Replace Claude Code-only tool references with Pi equivalents.
- Keep names lowercase and directory-matching.

Prompts:

- Convert `.claude/commands/moai/*.md` into `.pi/prompts/*.md` where useful.
- Keep `/moai` command as the official workflow entrypoint.

Themes/renderers:

- Convert output styles into prompt guidelines where they affect assistant behavior.
- Use Pi theme files for colors only.
- Use `pi.registerMessageRenderer()` and custom tool renderers for MoAI status, task lists, quality reports, and agent/team progress.

## 13. Workstream I — Tooling and Quality Gates

Required Pi custom tools:

```text
moai_config_get
moai_config_set
moai_spec_create
moai_spec_update
moai_spec_list
moai_spec_status
moai_quality_gate
moai_lsp_check
moai_mx_scan
moai_mx_update
moai_agent_invoke
moai_team_run
moai_task_create
moai_task_update
moai_task_list
moai_task_get
moai_memory_read
moai_memory_write
moai_context_search
moai_worktree_create
moai_worktree_merge
moai_worktree_cleanup
```

Guidelines:

- Use TypeBox schemas for all tools.
- Use `StringEnum` from `@earendil-works/pi-ai` for enum compatibility.
- Truncate outputs to Pi-safe limits.
- Return rich `details` for rendering and state reconstruction.
- Throw errors from tool execution to signal failed tool calls.
- Use file mutation queues for file-writing custom tools.

## 14. Workstream J — Full Workflow Parity

### `/moai plan`

```text
context discovery
Socratic interview through Pi UI
manager-spec invocation
plan-auditor invocation
SPEC document generation
MX target proposal
harness level decision
plan approval gate
```

### `/moai run`

```text
SPEC load
development mode selection: TDD or DDD
agent/team execution plan
worktree setup
reproduction-first bug fixing when applicable
implementation agents
test agents
quality gates
post-implementation review
merge or blocker report
```

### `/moai sync`

```text
documentation synchronization
MX validation
README/docs updates
SPEC status update
session summary persistence
```

### `/moai review`

```text
changed-file detection
parallel expert reviews
TRUST 5 scoring
security/performance/testing pass
evaluator-active synthesis
report rendering
```

All other MoAI subcommands must call the same Go Core flows as Claude Code mode.

## 15. Testing Strategy

### Go tests

```text
RuntimeAdapter interface tests
bridge JSON schema tests
hook policy parity tests
task store tests
agent runtime contract tests
team coordinator tests
worktree lifecycle tests
```

### TypeScript/Pi tests

```text
command parser tests
bridge invocation tests
tool schema tests
event mapping tests
renderer smoke tests
session persistence tests
```

### End-to-end tests

```bash
pi -e ./.pi/extensions/moai/index.ts -p "/moai plan test feature"
pi -e ./.pi/extensions/moai/index.ts --mode json -p "/moai review"
```

### Parity tests

Use fixtures for existing Claude Code hook payloads and expected policy decisions, then feed equivalent Pi event payloads and verify identical MoAI decisions.

## 16. Implementation Phases

These are dependency phases, not MVP boundaries. The target remains full parity.

### Phase 0 — Inventory and Compatibility Matrix

Deliverables:

- Command inventory
- Hook inventory
- Agent inventory
- Skill validation report
- Claude-only dependency list
- Pi parity matrix

### Phase 1 — Bridge Protocol

Deliverables:

- `moai pi bridge` command group
- JSON schemas
- request/response tests
- error contract

### Phase 2 — Pi Extension Shell

Deliverables:

- package manifest
- extension entrypoint
- `/moai` command parser
- bridge client
- session status indicator
- custom renderers scaffold

### Phase 3 — Hook/Event Parity

Deliverables:

- Pi event handlers
- synthesized missing events
- policy engine integration
- dangerous command/protected path parity

### Phase 4 — Interaction and Task System

Deliverables:

- `UserInteractionPort`
- Pi UI implementation
- `TaskPort`
- task custom tools
- `.moai/tasks` persistence
- session replay support

### Phase 5 — Agent Runtime

Deliverables:

- agent definition loader
- Pi subprocess worker
- single/parallel/chain execution
- streaming updates
- result contract
- blocker handling
- worktree-aware write agents

### Phase 6 — Team Runtime

Deliverables:

- role profile resolver
- task graph coordinator
- worktree isolation
- file ownership enforcement
- merge and conflict policy
- evaluator synthesis

### Phase 7 — Workflow Parity

Deliverables:

- full `/moai plan`
- full `/moai run`
- full `/moai sync`
- full `/moai review`
- all utility and quality subcommands

### Phase 8 — Resource Conversion

Deliverables:

- skill converter
- prompt converter
- theme/renderer integration
- resource drift checks

### Phase 9 — Stabilization and Release

Deliverables:

- `moai pi doctor`
- CI matrix with Pi installed
- E2E tests
- documentation
- install/update flow
- release artifacts

## 17. Major Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| Subprocess-based agents are slower/costlier | High | Add concurrency limits, token budgets, result compression, future provider-native runtime |
| Pi print/json modes cannot ask users | Medium | Return structured blocker or require interactive mode for interview workflows |
| State divergence between `.moai` and Pi session | High | Durable store + branch replay + versioned task records |
| Hook mapping gaps | Medium | Synthetic events and parity fixtures |
| Worktree merge conflicts | High | file ownership, small task shards, pre-merge quality gate, conflict report |
| Skill drift between Claude and Pi | Medium | generated Pi resources and drift CI |
| Custom tool write races | High | Pi file mutation queue and Go-side file locks |
| Project-local agent security | High | confirmation gate and trusted-repo policy |
| Model/tool mismatch in subagents | Medium | agent config validation and explicit tool lists |

## 18. Definition of Full Pi Support

MoAI-ADK is considered fully supported on Pi only when all criteria pass:

1. `pi install git:...moai-adk` loads extension, skills, prompts, and themes.
2. `/moai` and all documented subcommands work from Pi.
3. Existing `.moai` projects run without migration loss.
4. Claude Code hook policies have Pi event parity tests.
5. AskUserQuestion behavior is replaced by Pi UI with no free-form prompt waiting.
6. TaskCreate/Update/List/Get behavior is replaced by MoAI task tools and persistent state.
7. All static agents can be invoked through MoAI Agent Runtime.
8. Team workflows run through MoAI Team Coordinator with worktree isolation.
9. Plan/run/sync/review workflows produce equivalent SPEC and quality artifacts.
10. Quality gates, LSP gates, MX validation, and memory systems operate in Pi.
11. Session resume and branch navigation preserve MoAI state.
12. `moai pi doctor` detects missing Pi, extension, package, binary, and config issues.
13. CI runs Go tests, TypeScript checks, and Pi E2E tests.

## 19. Final Recommendation

Proceed with the plan, but implement it as a **multi-runtime refactor** rather than a TypeScript-only Pi rewrite.

The three hard problems are solvable:

1. Sub-agents become MoAI Agent Runtime workers.
2. Claude Code hooks become Pi event translations plus synthetic events.
3. AskUserQuestion and Task tools become runtime-neutral ports backed by Pi UI and MoAI task storage.

The most important early decision is to build the bridge and runtime ports first. If that foundation is skipped, the Pi extension will become a second implementation of MoAI and will diverge from Claude Code support.
