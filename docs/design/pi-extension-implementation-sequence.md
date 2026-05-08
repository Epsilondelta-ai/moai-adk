# Pi Extension Implementation Sequence

Status: Draft implementation roadmap  
Date: 2026-05-08  
Scope: Step-by-step plan to bring MoAI Pi support from foundation to Claude MoAI parity

## 0. Purpose

This document converts the Pi parity gap review into an executable sequence. Follow the phases in order. Each phase is intentionally small enough to implement, test, and commit independently.

The existing full-port architecture remains defined in `docs/design/pi-extension-full-port-plan.md`. This document is the tactical execution order.

## 1. Current Baseline

Implemented:

- `.pi/extensions`, `.pi/skills`, `.pi/prompts` project-local layout
- Pi bridge protocol foundation: `doctor`, `capabilities`, `command`, `event`, `tool`, `state`
- `/moai` and `/moai-doctor` commands
- Basic tools: config read, spec status, quality check, MX scan, task store, agent invoke, team run
- Basic Pi footer/status UI
- Basic safety policy for destructive `rm -rf` and protected path writes
- Basic Pi subprocess agent runtime
- Basic team runtime as parallel agent wrapper

Known limitation:

- The Pi port is currently a functional foundation, not full workflow parity with Claude MoAI.

## 2. Implementation Principles

1. Keep Go Core as the source of truth; TypeScript remains an adapter.
2. Commit each phase independently.
3. Add or update tests in the same phase as the implementation.
4. Prefer explicit structured bridge payloads over generic `action + payload` when the tool is user-facing.
5. Preserve Claude and Pi resources together when the same instruction exists in both trees.
6. Avoid adding features to later phases while implementing an earlier phase.

## 3. Phase 1 — Pi MoAI Output Style Parity

### Goal

Make Pi responses use MoAI-style start/gate/completion framing comparable to Claude's `.claude/output-styles/moai/moai.md`.

### Scope

- Add a Pi-specific output style prompt module.
- Inject MoAI response template guidance into Pi sessions.
- Ensure completion reports can render the familiar banner:

```text
🤖 MoAI ★ 완료 ─────────────────────────
✅ ...
📊 ...
────────────────────────────────────────
```

### Recommended Implementation

1. Add `.pi/prompts` or extension-level prompt injection support for MoAI style guidance.
2. Source template text from `.claude/output-styles/moai/moai.md` or create a generated Pi-safe subset.
3. Hook injection at one of:
   - `before_provider_request`, if Pi exposes provider request mutation in the current SDK
   - `before_agent_start` / session context injection, if provider mutation is not available
   - prompt template fallback for `/moai` commands
4. Keep formatting language-aware:
   - Korean users: `완료`, `작업 시작`, `검증`
   - English users: `Complete`, `Task Start`, `Gate`
5. Do not force banners for every trivial assistant response unless the active command is MoAI-related.

### Files Likely Touched

- `.pi/extensions/moai/events.ts`
- `.pi/extensions/moai/commands.ts`
- `.pi/extensions/moai/renderers.ts`
- optional: `.pi/extensions/moai/output-style.ts`
- optional: `.pi/prompts/moai-style.md`
- tests under `internal/pi/bridge` or future extension tests

### Acceptance Criteria

- `/moai-doctor` or `/moai <command>` responses display MoAI-branded framing in Pi.
- In a MoAI-enabled project, normal chat and MoAI commands both start with a compact MoAI-branded frame even when the user does not type `/moai`.
- The implementation does not duplicate large Claude-only instructions unnecessarily.
- No hardcoded Claude model/provider attribution is introduced.

### Validation

```bash
go test ./internal/pi/...
PI_OFFLINE=1 pi --no-tools -p "/moai-doctor"
```

Manual Pi check:

- Start Pi
- Run `/reload`
- Run `/moai-doctor`
- Confirm MoAI-style visible response framing

## 4. Phase 2 — Structured User Interaction Port

### Goal

Replace Claude `AskUserQuestion` semantics with a Pi-native interaction port backed by `ctx.ui`.

### Scope

- Add `moai_ask_user` custom tool or command-level interaction bridge.
- Support select, confirm, input, editor, and notify.
- Persist interaction decisions in `.moai` and/or Pi session entries when needed.

### Recommended Implementation

1. Define a Go interaction request/response schema.
2. Add bridge kind or tool handling for `moai_ask_user`.
3. In TypeScript, implement the actual UI call because only the Pi extension has `ctx.ui`.
4. Route structured options through:
   - `ctx.ui.select` for single-choice
   - `ctx.ui.confirm` for boolean gates
   - `ctx.ui.input` for short text
   - `ctx.ui.editor` for multi-line text
   - `ctx.ui.notify` for non-blocking notices
5. Enforce MoAI question rules:
   - max 4 questions per round
   - max 4 options per question
   - first option recommended
   - user conversation language
6. Update converted Pi skills so they refer to `moai_ask_user` instead of prose placeholders.

### Files Likely Touched

- `.pi/extensions/moai/tools.ts`
- `.pi/extensions/moai/types.ts`
- `.pi/extensions/moai/bridge.ts`
- new: `.pi/extensions/moai/interaction.ts`
- `internal/pi/bridge/handler.go`
- new or existing interaction package under `internal/interaction` or `internal/pi/bridge`
- `internal/pi/skillconvert/convert.go`
- `.pi/skills/moai/**`

### Acceptance Criteria

- The model can call `moai_ask_user` in Pi.
- Pi displays structured UI instead of requiring prose questions.
- Interaction result is returned to the model as a normal tool result.
- Converted skills no longer contain ambiguous placeholder text as the only interaction mechanism.

### Validation

```bash
go test ./internal/pi/... ./internal/taskstore/... ./internal/sessionstate/...
rg "Pi UI interaction via ctx.ui or the moai_ask_user runtime port" .pi/skills
```

Manual Pi check:

- Trigger a workflow that needs clarification.
- Confirm Pi shows a structured UI control.
- Confirm the assistant receives the selected answer.

## 5. Phase 3 — Command Workflow Parity Foundation

### Goal

Replace placeholder `/moai` command behavior with real workflow dispatch that follows existing MoAI workflow semantics.

### Scope

Start with the core three:

1. `/moai plan`
2. `/moai run`
3. `/moai sync`

Then expand to utility commands.

### Recommended Implementation

1. Replace `internal/kernel.executePlan` placeholder output with canonical SPEC directory structure:
   - `spec.md`
   - `plan.md`
   - `acceptance.md`
2. Ensure SPEC frontmatter follows the canonical schema used by MoAI.
3. Implement `/moai run` as orchestration preparation first, then agent/TDD-DDD invocation in a later sub-step.
4. Implement `/moai sync` as documentation/MX/status synchronization preparation first, then full docs agent flow in a later sub-step.
5. Keep all command results structured:
   - messages
   - artifacts
   - UI state
   - next actions
6. Add blocker reports when required context is missing.

### Files Likely Touched

- `internal/kernel/kernel.go`
- `internal/kernel/types.go`
- `internal/pi/bridge/handler.go`
- `.pi/extensions/moai/commands.ts`
- `.pi/prompts/moai-*.md`
- tests under `internal/kernel` and `internal/pi/bridge`

### Acceptance Criteria

- `/moai plan "..."` creates a valid MoAI SPEC folder, not a Pi-only placeholder format.
- `/moai run SPEC-...` validates SPEC existence and returns an implementation execution plan or blocker.
- `/moai sync SPEC-...` validates run artifacts and returns sync actions or blocker.
- Command results can be rendered by Pi without losing artifact paths.

### Validation

```bash
go test ./internal/kernel/... ./internal/pi/bridge/...
moai pi bridge <<EOF
{"version":"moai.pi.bridge.v1","kind":"command","cwd":"$PWD","payload":{"command":"plan","args":"test pi workflow parity"}}
EOF
```

Manual Pi check:

- Run `/moai plan test feature`
- Confirm SPEC folder shape matches Claude MoAI expectations.

## 6. Phase 4 — Quality, LSP, and MX Gate Parity

### Goal

Replace basic test runners and MX counters with MoAI-grade gates.

### Scope

- Quality gate routing by project language
- LSP diagnostic checks
- MX tag validation
- TRUST 5 report foundation

### Recommended Implementation

1. Expand quality checks:
   - Go: `go vet`, optional `golangci-lint`, `go test`
   - Node: eslint, typecheck if configured, test
   - Python: ruff, pytest
   - Rust: cargo clippy, cargo test
2. Add graceful skip for missing tools.
3. Add structured result model:
   - passed
   - warnings
   - errors
   - skipped
   - commands
   - output truncation
4. Implement `moai_lsp_check` as a real diagnostic adapter or clear unsupported result.
5. Expand MX scan:
   - count tags
   - validate `@MX:WARN` has `@MX:REASON`
   - detect malformed tags
   - support file path filtering
6. Add `moai_mx_update` only after validation is reliable.

### Files Likely Touched

- `internal/pi/bridge/quality.go`
- `internal/pi/bridge/handler.go`
- new package if needed: `internal/lsp`, `internal/mx`, or `internal/quality`
- `.pi/extensions/moai/tools.ts`
- tests under `internal/pi/bridge`

### Acceptance Criteria

- `moai_quality_gate` returns structured pass/fail/skip results.
- `moai_lsp_check` no longer aliases generic quality checks without explanation.
- `moai_mx_scan` reports malformed tags and missing reasons.
- Large output is safely truncated.

### Validation

```bash
go test ./internal/pi/... ./internal/quality/... ./internal/lsp/... ./internal/mx/...
```

Manual Pi check:

- Call `moai_quality_gate`
- Call `moai_mx_scan`
- Confirm results are specific enough for the assistant to act on.

## 7. Phase 5 — Event and Hook Policy Parity

### Goal

Map Claude MoAI hook behavior to Pi events as fully as Pi allows.

### Scope

Implement or synthesize these missing events:

- `turn_start`
- `turn_end`
- `agent_start`
- `session_before_compact`
- `session_compact`
- `StopFailure`
- `ConfigChange`
- `CwdChanged`
- `FileChanged`
- `PermissionDenied`
- `Elicitation`
- `ElicitationResult`
- `SubagentStart`
- `SubagentStop`
- `TaskCompleted`
- `TeammateIdle`
- `WorktreeCreate`
- `WorktreeRemove`

### Recommended Implementation

1. Audit Pi extension event APIs and determine which events are native.
2. Implement native mappings first.
3. For missing events, synthesize from existing events and state transitions.
4. Move policy decisions into Go where possible.
5. Add parity fixtures comparing Claude hook payloads and Pi event payloads.
6. Update `moai_bridge_capabilities` to reflect actual implemented events, not target events.

### Files Likely Touched

- `.pi/extensions/moai/events.ts`
- `.pi/extensions/moai/state.ts`
- `internal/policy/evaluator.go`
- `internal/pi/bridge/handler.go`
- `internal/sessionstate/store.go`
- tests under `internal/policy`, `internal/pi/bridge`, `internal/sessionstate`

### Acceptance Criteria

- Capabilities only list supported events or clearly mark synthetic/partial events.
- Dangerous command and protected path policy matches Claude behavior for core cases.
- Stop/agent/task lifecycle state is persisted enough for recovery.
- Event handling does not block normal Pi operation on bridge failures.

### Validation

```bash
go test ./internal/policy/... ./internal/pi/bridge/... ./internal/sessionstate/...
```

Manual Pi check:

- Run tool calls that should be blocked.
- Trigger tool failure and provider failure if possible.
- Confirm MoAI state records the event.

## 8. Phase 6 — Agent Runtime Hardening

### Goal

Make Pi subprocess agents reliable enough for real MoAI workflows.

### Scope

- Agent discovery and model mapping
- Tool permission mapping
- Worktree isolation
- Blocker report propagation
- Usage/cost capture
- Failure handling

### Recommended Implementation

1. Improve agent name aliasing so user-facing names map consistently.
2. Add clear errors for missing provider/API key and suggest Pi login/config action.
3. Preserve subagent no-user-prompt rule through blocker reports.
4. Expand tool mapping beyond basic read/write/edit/bash/find/grep/ls where Pi supports it.
5. Add lifecycle events around agent start/end.
6. Improve worktree cleanup and merge handoff.
7. Add tests using fake Pi subprocess output.

### Files Likely Touched

- `internal/agentruntime/pi_worker.go`
- `internal/agentruntime/definition.go`
- `internal/agentruntime/modelmap.go`
- `internal/agentruntime/worktree.go`
- `internal/pi/bridge/handler.go`
- `.pi/extensions/moai/tools.ts`

### Acceptance Criteria

- `moai_agent_invoke action=list` returns usable agent names.
- Missing API key/provider errors are classified as configuration blockers.
- Worktree-enabled invocations create isolated worktrees and report their paths.
- Blocker reports are surfaced to the orchestrator, not silently treated as generic failures.

### Validation

```bash
go test ./internal/agentruntime/... ./internal/pi/bridge/...
```

Manual Pi check:

- Run `moai_agent_invoke` list.
- Run a simple read-only agent with a short timeout.
- Run an implementation-style agent in worktree mode.

## 9. Phase 7 — Team Runtime Parity

### Goal

Move beyond parallel wrapper behavior toward MoAI Agent Teams semantics.

### Scope

- Role profiles
- File ownership strategy
- Task state integration
- Teammate idle handling
- Completion acceptance/rejection
- Team cleanup lifecycle

### Recommended Implementation

1. Add a durable team state store under `.moai/state/teams`.
2. Represent teammates, assigned tasks, status, and owned file patterns.
3. Implement `moai_team_run` actions:
   - `create`
   - `run`
   - `status`
   - `message`
   - `delete`
4. Enforce worktree isolation for write-capable roles.
5. Add conflict detection for overlapping file ownership.
6. Add task completion review hook.
7. Add graceful cleanup on failure.

### Files Likely Touched

- `internal/teamruntime/**`
- `internal/taskstore/**`
- `internal/agentruntime/**`
- `internal/pi/bridge/handler.go`
- `.pi/extensions/moai/tools.ts`

### Acceptance Criteria

- Team state survives bridge process boundaries.
- `moai_team_run status` returns current team/task status.
- Write-capable roles use worktree isolation by default.
- Completion can be accepted or rejected by policy.
- Cleanup removes transient team resources when requested.

### Validation

```bash
go test ./internal/teamruntime/... ./internal/taskstore/... ./internal/agentruntime/...
```

Manual Pi check:

- Start a small two-agent team.
- Confirm status updates.
- Confirm cleanup.

## 10. Phase 8 — Tool Surface Completion

### Goal

Expose the complete Pi tool set expected by MoAI workflows.

### Missing Tools To Add

- `moai_config_set`
- `moai_spec_create`
- `moai_spec_update`
- `moai_spec_list`
- `moai_mx_update`
- `moai_memory_read`
- `moai_memory_write`
- `moai_context_search`
- `moai_worktree_create`
- `moai_worktree_merge`
- `moai_worktree_cleanup`
- `moai_ask_user` if not completed in Phase 2

### Recommended Implementation

1. Replace overly generic schemas with tool-specific TypeBox schemas where practical.
2. Add Go handlers one tool at a time.
3. Add tests for each tool handler.
4. Update `moai_bridge_capabilities` after each tool is real.
5. Mark destructive/mutating tools with explicit policy checks.

### Files Likely Touched

- `.pi/extensions/moai/tools.ts`
- `.pi/extensions/moai/types.ts`
- `internal/pi/bridge/handler.go`
- domain packages for spec, memory, context, worktree as needed

### Acceptance Criteria

- Capabilities list matches actual registered tools.
- Each listed tool has a working bridge handler.
- Mutating tools validate inputs and protected paths.
- Tool result details are rich enough for rendering and replay.

### Validation

```bash
go test ./internal/pi/bridge/... ./internal/taskstore/... ./internal/sessionstate/...
```

Manual Pi check:

- Call every MoAI tool once with a safe input.
- Confirm error messages are actionable for invalid input.

## 11. Phase 9 — Resource Conversion and Drift Control

### Goal

Keep Claude and Pi skills/prompts aligned without manual drift.

### Scope

- Skill conversion
- Prompt conversion
- Output style conversion
- Drift detection CI or local check

### Recommended Implementation

1. Extend `internal/pi/skillconvert` to handle interaction/tool naming cleanly.
2. Extend `internal/pi/resourcesync` to generate prompt templates into `.pi/prompts`.
3. Add output style conversion for Pi style prompt subset.
4. Add a check command that reports drift between `.claude` sources and `.pi` generated resources.
5. Document which `.pi` files are generated and which are hand-authored.

### Files Likely Touched

- `internal/pi/skillconvert/**`
- `internal/pi/resourcesync/**`
- `.pi/skills/**`
- `.pi/prompts/**`
- optional: new drift test under `internal/pi`

### Acceptance Criteria

- Regeneration produces stable output.
- Drift check fails when generated resources are stale.
- Pi resources do not contain Claude-only tool names except in historical references.
- Documentation states the regeneration workflow.

### Validation

```bash
go test ./internal/pi/...
moai pi sync-resources --check
```

If `moai pi sync-resources --check` does not exist yet, implement it in this phase.

## 12. Phase 10 — End-to-End Stabilization

### Goal

Verify Pi MoAI is usable end-to-end for a small project workflow.

### Scope

- E2E smoke tests
- Manual QA script
- Docs update
- Release readiness checklist

### Recommended Implementation

1. Add a scripted smoke test for:
   - Pi extension load
   - `/moai-doctor`
   - `/moai plan`
   - `moai_task_create/list/get/update`
   - `moai_quality_gate`
   - `moai_mx_scan`
2. Add manual QA instructions for interactive UI pieces.
3. Update README or Pi-specific docs with current capabilities and limitations.
4. Ensure failure modes are clear when API keys/providers are missing.

### Files Likely Touched

- `test/` or `scripts/`
- `docs/` or `README.md`
- `docs/design/pi-extension-full-port-plan.md`
- `docs/design/pi-extension-implementation-sequence.md`

### Acceptance Criteria

- A new user can enable Pi MoAI and run the smoke flow.
- Known limitations are documented.
- Core commands do not create invalid MoAI state.
- All package/resource paths use `.pi`, not legacy `pi`.

### Validation

```bash
go test ./...
pi --offline --no-tools -p "/moai-doctor"
```

Manual Pi QA:

1. Start Pi in the repository.
2. Run `/reload`.
3. Run `/moai-doctor`.
4. Run `/moai plan "small test feature"`.
5. Run `/moai run <SPEC-ID>`.
6. Run `/moai sync <SPEC-ID>`.
7. Verify MoAI framing, footer, artifacts, and state.

## 13. Commit Order

Use this commit sequence:

1. `feat(pi): add moai output style guidance`
2. `feat(pi): add structured user interaction port`
3. `feat(pi): implement core command workflow parity`
4. `feat(pi): expand quality lsp and mx gates`
5. `feat(pi): improve event policy parity`
6. `feat(pi): harden subprocess agent runtime`
7. `feat(pi): add durable team runtime state`
8. `feat(pi): complete moai bridge tool surface`
9. `chore(pi): add resource drift controls`
10. `test(pi): add end-to-end smoke coverage`

## 14. Stop Conditions

Pause and re-plan if any of these occur:

- Pi SDK lacks an event or system prompt injection API required by the current phase.
- A phase requires changing more than 10 unrelated files.
- A bridge schema change would break already registered Pi tools.
- A workflow starts duplicating Claude-only logic in TypeScript instead of Go Core.
- Tests require live provider credentials for basic validation.

## 15. Definition of Done for Pi Full Parity

Pi MoAI can be considered parity-complete when:

- MoAI-branded responses and footer are visible in Pi.
- Structured user interaction works without prose fallback.
- `/moai plan/run/sync` operate on canonical MoAI SPEC artifacts.
- Quality, LSP, and MX gates produce actionable structured reports.
- Hook/event policy behavior matches Claude core safety cases.
- Agent and team runtimes support blocker propagation and worktree isolation.
- Capabilities accurately reflect implemented tools/events.
- Resource conversion prevents drift between Claude and Pi assets.
- E2E smoke flow passes on a fresh checkout.
