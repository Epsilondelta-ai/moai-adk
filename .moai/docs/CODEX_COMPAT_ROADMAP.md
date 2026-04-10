# Codex Compatibility Roadmap

## Purpose

This document is the persistent execution roadmap for making MoAI-ADK usable in Codex while preserving upstream mergeability with the Claude-first upstream repository.

This file is written for AI execution first, not human discussion first.

After context is cleared, an AI should be able to continue by being given:

- this file path
- one task ID or task name

Example re-entry prompts:

- `Continue task CX-01 Codex Surface Audit using .moai/docs/CODEX_COMPAT_ROADMAP.md`
- `Work on CX-04 Codex Template Scaffold from .moai/docs/CODEX_COMPAT_ROADMAP.md`
- `Read .moai/docs/CODEX_COMPAT_ROADMAP.md and execute next pending task`

## Operating Rules

1. Treat `.moai` as the shared source of truth.
2. Treat `.claude` as the existing upstream adapter. Do not redesign it unless unavoidable.
3. Add Codex support as a new adapter layer, primarily under new Codex-specific paths.
4. Prefer additive changes over invasive refactors.
5. Preserve upstream-friendly file ownership boundaries.
6. When possible, reuse existing `internal/config`, `internal/template`, `internal/manifest`, `internal/loop`, `internal/workflow`, and `internal/mcp`.
7. Do not claim Claude hook parity in Codex unless a real equivalent exists.
8. Favor workflow parity over runtime parity.

## End State

When this roadmap is complete:

- A project already initialized with `moai init` can also be used from Codex.
- Codex can invoke MoAI via `$moai`.
- `.moai` data is reused by both Claude and Codex.
- Codex-specific generated assets live in Codex-specific paths.
- `moai init` and `moai update` can provision and refresh Codex assets.
- Future upstream syncs minimize conflicts because Claude-specific files remain mostly untouched.

## Status Legend

- `PENDING`: not started
- `IN_PROGRESS`: currently active
- `BLOCKED`: cannot proceed without a decision or prerequisite
- `DONE`: implemented and verified
- `SKIPPED`: intentionally omitted

## Task Index

| ID | Name | Status | Depends On |
| --- | --- | --- | --- |
| CX-01 | Codex Surface Audit | DONE | - |
| CX-02 | Shared/Core Boundary Definition | PENDING | CX-01 |
| CX-03 | Codex Adapter Layout | PENDING | CX-02 |
| CX-04 | Codex Template Scaffold | PENDING | CX-03 |
| CX-05 | `$moai` Skill Entry Point | PENDING | CX-04 |
| CX-06 | Init and Update Provisioning | PENDING | CX-04 |
| CX-07 | Codex Workflow Prompt Pack | PENDING | CX-05 |
| CX-08 | Manifest and Drift Policy | PENDING | CX-06 |
| CX-09 | Codex Runtime Helpers | PENDING | CX-06 |
| CX-10 | Verification Matrix | PENDING | CX-05, CX-06, CX-07, CX-08, CX-09 |
| CX-11 | Upstream Merge Playbook | PENDING | CX-10 |

## Task Details

### CX-01 Codex Surface Audit

- Status: `DONE`
- Goal: Identify the minimum viable Codex integration surface without touching Claude-only behavior.
- Why:
  Claude-facing behavior is spread across CLI launchers, hooks, templates, profiles, and statusline logic. This task maps what is Claude-only, what is reusable, and what must be replaced for Codex.
- Inputs:
  - `.moai/docs/CODEX_COMPAT_ROADMAP.md`
  - `internal/cli/`
  - `internal/template/templates/.claude/`
  - `internal/hook/`
  - `internal/profile/`
- Outputs:
  - A short audit section appended under `Execution Notes`
  - A concrete list of reusable modules and Claude-locked modules
- Completion Criteria:
  - Reusable modules are named explicitly
  - Claude-only modules are named explicitly
  - Candidate Codex integration points are named explicitly

### CX-02 Shared/Core Boundary Definition

- Status: `PENDING`
- Goal: Define what belongs to the shared MoAI core versus the Claude adapter versus the new Codex adapter.
- Why:
  Without this boundary, later work drifts into broad refactors and merge conflicts.
- Inputs:
  - CX-01 results
  - `.moai`, `.claude`, current template layout
- Outputs:
  - A written boundary decision under `Architecture Decisions`
  - A file ownership map
- Completion Criteria:
  - Shared core paths are listed
  - Claude adapter paths are listed
  - Codex adapter paths are listed
  - "Do not modify unless necessary" paths are listed

### CX-03 Codex Adapter Layout

- Status: `PENDING`
- Goal: Choose stable on-disk paths for Codex-generated assets and Codex-specific source code.
- Why:
  Layout decisions determine future merge cost.
- Inputs:
  - CX-02
- Outputs:
  - Final path plan under `Architecture Decisions`
- Required Decisions:
  - Whether project assets live under `.codex/`
  - Where Codex templates live in repo source
  - Whether a new `internal/cli/codex.go` is needed
  - Whether Codex runtime code belongs under `internal/codex/` or `internal/runtime/codex/`
- Completion Criteria:
  - All new top-level paths are fixed
  - No existing Claude path needs to move

### CX-04 Codex Template Scaffold

- Status: `PENDING`
- Goal: Add Codex template source files without disturbing existing Claude templates.
- Why:
  Codex support should be provisioned as generated assets, like Claude support.
- Inputs:
  - CX-03
- Outputs:
  - New template tree for Codex assets
- Candidate Paths:
  - `internal/template/templates/.codex/...`
- Completion Criteria:
  - Template tree exists
  - Templates are additive
  - No `.claude` template rename or relocation happened

### CX-05 `$moai` Skill Entry Point

- Status: `PENDING`
- Goal: Make Codex able to invoke MoAI workflows through `$moai`.
- Why:
  This is the primary UX requirement.
- Inputs:
  - CX-04
- Outputs:
  - Codex skill entrypoint
  - Re-entry instructions for `$moai`, `$moai plan`, `$moai run`, `$moai sync`
- Completion Criteria:
  - Codex can discover a MoAI skill
  - The skill explains how to route common subcommands
  - The skill reads project state from `.moai`

### CX-06 Init and Update Provisioning

- Status: `PENDING`
- Goal: Ensure `moai init` and `moai update` create and refresh Codex assets.
- Why:
  Manual setup is too fragile for AI-first operation.
- Inputs:
  - CX-04
- Outputs:
  - Init/update code changes
  - Manifest tracking for Codex-generated files where appropriate
- Completion Criteria:
  - New projects receive Codex assets
  - Existing projects can receive Codex assets via update
  - Existing Claude flow still works

### CX-07 Codex Workflow Prompt Pack

- Status: `PENDING`
- Goal: Recreate MoAI workflow behavior in Codex at the prompt/skill level.
- Why:
  Claude hook semantics cannot simply be assumed in Codex.
- Inputs:
  - CX-05
  - Existing `.claude/commands/moai/`
  - Existing `.claude/skills/moai*`
- Outputs:
  - Codex workflow skill docs or references for:
    - project
    - plan
    - run
    - sync
    - review
    - clean
    - loop
- Completion Criteria:
  - Each key workflow has a Codex-side prompt contract
  - The prompt pack references `.moai` state instead of Claude-only runtime features

### CX-08 Manifest and Drift Policy

- Status: `PENDING`
- Goal: Decide how Codex-generated files are tracked and updated safely.
- Why:
  Asset drift will happen. The system needs deterministic update behavior.
- Inputs:
  - CX-06
- Outputs:
  - Tracking policy in this document
  - Any required manifest integration changes
- Completion Criteria:
  - Template-managed versus user-modified policy is defined for Codex assets
  - Update overwrite rules are documented
  - Safe regeneration behavior is defined

### CX-09 Codex Runtime Helpers

- Status: `PENDING`
- Goal: Add any helper CLI/runtime support needed specifically for Codex workflows.
- Why:
  Some capabilities may need local helpers even if the main interface is a skill.
- Inputs:
  - CX-06
- Possible Scope:
  - `moai codex doctor`
  - `moai codex sync`
  - helper output for skill consumption
- Completion Criteria:
  - Only necessary helpers are added
  - Helper scope remains Codex-specific and additive

### CX-10 Verification Matrix

- Status: `PENDING`
- Goal: Verify the Codex adapter works on an already initialized MoAI project and does not regress Claude support.
- Why:
  This change spans templates, provisioning, prompt assets, and possibly CLI.
- Inputs:
  - CX-05 through CX-09
- Outputs:
  - Verification checklist under `Verification Log`
- Minimum Checks:
  - Existing `.moai` project can receive Codex assets
  - `$moai` skill is discoverable in Codex
  - Core workflows can be invoked from Codex
  - `go test ./...` passes
- Completion Criteria:
  - Results are recorded with pass/fail notes

### CX-11 Upstream Merge Playbook

- Status: `PENDING`
- Goal: Document how to keep rebasing or merging upstream with minimal conflict.
- Why:
  Long-term maintainability is a stated requirement.
- Inputs:
  - All prior tasks
- Outputs:
  - Merge playbook under `Upstream Strategy`
- Completion Criteria:
  - Lists protected paths
  - Lists Codex-owned paths
  - Lists expected conflict hotspots
  - Defines recommended merge order and review checkpoints

## Architecture Decisions

This section is intentionally mutable. Update it as decisions are made.

### Current Working Direction

- Shared source of truth: `.moai`
- Existing adapter: `.claude`
- New adapter: `.codex`
- Preferred implementation style: additive templates and additive Codex-specific runtime/helpers
- Preferred compatibility target: workflow parity in Codex, not Claude hook parity

### File Ownership Draft

- Shared/core:
  - `.moai/**`
  - `internal/config/**`
  - `internal/manifest/**`
  - `internal/template/**`
  - `internal/loop/**`
  - `internal/workflow/**`
  - `internal/mcp/**`
- Upstream Claude-owned unless necessary:
  - `internal/template/templates/.claude/**`
  - `internal/hook/**`
  - `internal/cli/cc.go`
  - `internal/cli/cg.go`
  - `internal/cli/glm.go`
  - `internal/cli/launcher.go`
- Codex-owned planned additions:
  - `internal/template/templates/.codex/**`
  - `internal/codex/**` or `internal/runtime/codex/**`
  - optional `internal/cli/codex.go`

## Execution Notes

Use this section as an append-only work log.

Template:

```md
### YYYY-MM-DD HH:MM TZ - TASK-ID Task Name

- Status: `IN_PROGRESS` -> `DONE`
- Summary:
  - ...
- Files touched:
  - ...
- Verification:
  - ...
- Follow-up:
  - ...
```

### 2026-04-10 14:32 KST - CX-01 Codex Surface Audit

- Status: `IN_PROGRESS` -> `DONE`
- Reusable modules:
  - `internal/cli/init.go`, `internal/cli/update.go`: reusable project/template orchestration, but currently provision Claude assets.
  - `internal/profile/sync.go`: `.moai/config/sections/` sync path is adapter-neutral aside from statusline field naming.
  - `internal/hook/registry.go`, `internal/hook/trace/`, `internal/hook/quality/`, `internal/hook/security/`, `internal/hook/memo/`, `internal/hook/lifecycle/`, `internal/hook/worktree_registry.go`: internal registry, trace, quality, and state helpers worth preserving behind any future adapter boundary.
- Claude-locked modules:
  - `internal/cli/cc.go`, `internal/cli/cg.go`, `internal/cli/glm.go`, `internal/cli/launcher.go`, `internal/cli/hook.go`, `internal/cli/statusline.go`, `internal/cli/profile.go`: hard-coded Claude launch UX, `.claude` paths, `CLAUDE_CONFIG_DIR`, Claude permission modes, and Claude hook/statusline contracts.
  - `internal/profile/profile.go`, `internal/profile/preferences.go`: storage rooted at `~/.moai/claude-profiles` with Claude-specific env and permission semantics.
  - `internal/hook/protocol.go`, `internal/hook/types.go`, event handlers under `internal/hook/`: Claude Code JSON protocol, event names, exit semantics, and teammate/worktree behavior are Claude-bound.
  - `internal/template/templates/.claude/{commands,skills,agents,output-styles,hooks,rules,settings.json.tmpl}`: adapter-owned Claude assets that should stay untouched.
- Candidate Codex integration points:
  - `internal/cli/codex.go`: additive Codex launcher rather than widening `cc/cg/glm` semantics.
  - `internal/cli/init.go` and `internal/cli/update.go`: additive provisioning seam for a future `internal/template/templates/.codex/` tree.
  - `internal/profile/`: shared profile CRUD could move behind an adapter-aware store later, but current Claude store should not be reused as-is.
  - new Codex-specific runtime/helper package alongside, not inside, `internal/hook/`.
- Verification:
  - Audited priority surfaces in `internal/cli/`, `internal/profile/`, `internal/hook/`, and `internal/template/templates/.claude/`; updated roadmap classifications only.
- Follow-up:
  - `CX-02` should define the shared/core boundary before any Codex template or launcher implementation.

## Verification Log

Use this section to record concrete verification results.

Template:

```md
- [ ] Existing `.moai` project receives Codex assets via update
- [ ] Fresh init creates Codex assets
- [ ] `$moai` is discoverable in Codex
- [ ] `$moai plan` works
- [ ] `$moai run` works
- [ ] `$moai sync` works
- [ ] Claude flow still works
- [ ] `go test ./...` passes
```

- 2026-04-10 14:32 KST: `CX-01` completed as a source audit and roadmap update only; no code-path tests were required or run.

## Upstream Strategy

Target merge policy:

- Keep upstream as authority for Claude adapter behavior.
- Avoid moving or renaming upstream Claude files.
- Add Codex support in clearly owned additive paths.
- Refactor shared code only when the refactor reduces future additive diff size.
- If a broad refactor is tempting, first record why additive extension is insufficient.

## Update Instructions

When a task starts:

1. Change that task's `Status` line to `IN_PROGRESS`.
2. Add an entry in `Execution Notes`.

When a task finishes:

1. Change that task's `Status` line to `DONE`.
2. Update the `Task Index` row.
3. Record verification in `Verification Log`.
4. Add follow-up notes if another task must absorb deferred work.

When blocked:

1. Change that task's `Status` line to `BLOCKED`.
2. Add a short blocker note in `Execution Notes`.
3. Do not silently skip unresolved architectural decisions.

## Next Task Selection Rule

Default execution order:

1. CX-01
2. CX-02
3. CX-03
4. CX-04
5. CX-05
6. CX-06
7. CX-07
8. CX-08
9. CX-09
10. CX-10
11. CX-11

If resuming after context loss:

- If the user names a task ID or task name, execute that task.
- If the user says `next task`, execute the first task in `Task Index` whose status is `PENDING` and whose dependencies are `DONE`.
- If multiple tasks are possible, prefer the lowest-numbered task.
