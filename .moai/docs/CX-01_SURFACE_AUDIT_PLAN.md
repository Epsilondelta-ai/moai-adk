# CX-01 Codex Surface Audit Plan

## Purpose

This document is the execution plan for `CX-01 Codex Surface Audit` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to identify the minimum viable Codex integration surface without changing Claude-only behavior.

## Scope

Audit only the current surface area that defines Claude-facing behavior:

- `internal/cli/`
- `internal/template/templates/.claude/`
- `internal/hook/`
- `internal/profile/`
- related `.moai` configuration only when needed for classification context

Do not implement Codex support in this task. This task produces classification and boundary inputs for `CX-02`.

## Working Assumptions

- `.moai` remains the shared source of truth.
- `.claude` remains the existing adapter and should not be redesigned here.
- Codex support should be additive, not a refactor of Claude paths.
- Workflow parity matters more than Claude runtime parity.

## Deliverables

This task is complete only when the roadmap file contains a short audit section under `Execution Notes` with:

- explicitly named reusable modules
- explicitly named Claude-locked modules
- explicitly named candidate Codex integration points

## Classification Rules

Use exactly these three buckets during the audit:

### 1. Reusable

Place code here when it is not inherently tied to Claude naming, Claude runtime semantics, or Claude-specific on-disk paths.

Examples:

- shared config loading
- generic template plumbing
- engine-neutral helper logic
- reusable profile CRUD patterns without Claude-specific storage assumptions

### 2. Claude-Locked

Place code here when it depends on Claude-specific contracts, naming, paths, or UX.

Examples:

- `.claude/...` asset paths
- `CLAUDE_CONFIG_DIR`
- Claude permission mode values
- Claude hook protocol and event semantics
- statusline behavior described specifically for Claude Code

### 3. Candidate Codex Integration Points

Place code here when it suggests a natural seam for additive Codex support.

Examples:

- a new `internal/cli/codex.go`
- a new `internal/template/templates/.codex/` tree
- a new Codex-specific runtime/helper package

## Execution Steps

### Step 1. Audit CLI Surface

Review launchers and entrypoints in `internal/cli/` and separate:

- generic orchestration from engine-specific launch logic
- shared config/setup behavior from Claude-specific paths and messaging
- possible additive Codex entrypoints from hard-coded Claude commands

Priority files:

- `internal/cli/launcher.go`
- `internal/cli/cc.go`
- `internal/cli/cg.go`
- `internal/cli/glm.go`
- `internal/cli/hook.go`
- `internal/cli/statusline.go`
- `internal/cli/profile.go`
- `internal/cli/init.go`
- `internal/cli/update.go`

Expected output from this step:

- a list of CLI files that are Claude-locked
- a list of CLI helpers that may be reused
- a short note on whether `internal/cli/codex.go` should exist

### Step 2. Audit Profile Surface

Review `internal/profile/` and distinguish between:

- reusable profile lifecycle logic
- Claude-specific storage and environment conventions
- preference fields that should not be reused as-is for Codex

Priority files:

- `internal/profile/profile.go`
- `internal/profile/preferences.go`
- `internal/profile/sync.go`

Expected output from this step:

- whether profile CRUD can be adapter-neutral
- which profile fields are Claude-only today
- whether Codex should have a separate profile store or a shared abstraction later

### Step 3. Audit Hook Surface

Review `internal/hook/` to determine which parts are:

- bound to Claude hook protocol and event names
- reusable internal quality/trace/registry components
- impossible to claim for Codex without a real equivalent

Expected output from this step:

- named Claude protocol-bound packages or files
- named engine-neutral helper areas worth preserving
- a list of hook behavior that must not be promised in Codex

### Step 4. Audit Claude Template Surface

Review `internal/template/templates/.claude/` at the subtree level and classify:

- reusable content patterns
- Claude-only generated assets
- prompt or workflow assets that should be reauthored for Codex instead of copied blindly

Focus on these subtree categories:

- `commands/`
- `skills/`
- `agents/`
- `output-styles/`
- `hooks/`
- settings-related assets

Expected output from this step:

- subtree-level keep/rewrite/additive-copy decisions
- a list of `.claude` assets that should stay untouched
- a list of assets that imply future `.codex` counterparts

### Step 5. Produce CX-01 Audit Note

Append a concise execution note to the roadmap file with this structure:

```md
### YYYY-MM-DD HH:MM TZ - CX-01 Codex Surface Audit

- Reusable modules:
  - ...
- Claude-locked modules:
  - ...
- Candidate Codex integration points:
  - ...
```

Keep the note short and file-oriented. Do not expand into design decisions reserved for `CX-02`.

## Current Preliminary Findings

These are starting hypotheses to verify during the audit:

- `internal/cli/launcher.go` is likely mixed: reusable orchestration plus Claude-specific launch semantics.
- `internal/cli/hook.go` is likely Claude-locked at the command protocol boundary.
- `internal/cli/statusline.go` is likely Claude-locked at the UX contract boundary.
- `internal/profile/profile.go` contains reusable CRUD patterns but is currently Claude-path-bound.
- `internal/profile/preferences.go` contains Claude-specific permission and display preferences.
- `internal/template/templates/.claude/` should be treated as adapter-owned by default, not shared by default.

## Out of Scope

- implementing `.codex` templates
- adding a Codex CLI command
- changing profile storage
- rewriting Claude hooks
- making architecture decisions beyond what is required to classify surfaces

## Exit Criteria

`CX-01` is ready to close when:

- the roadmap `Execution Notes` contains the audit entry
- reusable modules are named explicitly
- Claude-locked modules are named explicitly
- Codex integration seams are named explicitly
- the result is sufficient input for `CX-02 Shared/Core Boundary Definition`
