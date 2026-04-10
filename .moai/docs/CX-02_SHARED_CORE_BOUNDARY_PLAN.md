# CX-02 Shared/Core Boundary Definition Plan

## Purpose

This document is the execution plan for `CX-02 Shared/Core Boundary Definition` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to define a stable ownership boundary between the shared MoAI core, the existing Claude adapter, and the new Codex adapter before any Codex layout or template implementation begins.

## Task Outcome

This task produces architecture decisions, not runtime behavior.

The task is complete only when the roadmap contains:

- a written boundary decision under `Architecture Decisions`
- a concrete file ownership map
- explicit lists for shared/core, Claude adapter, Codex adapter, and protected paths

## Inputs

- `CX-01` audit results already recorded in the roadmap
- current project state under `.moai/` and `.claude/`
- template source layout under `internal/template/templates/`
- runtime and CLI surfaces under:
  - `internal/cli/`
  - `internal/profile/`
  - `internal/hook/`
  - `internal/config/`
  - `internal/manifest/`
  - `internal/loop/`
  - `internal/workflow/`
  - `internal/mcp/`

## Working Assumptions

- `.moai` remains the shared project-state authority.
- `.claude` remains an upstream-owned adapter surface.
- Codex support must be additive and merge-friendly.
- Claude runtime semantics are not a target for reuse unless they are genuinely adapter-neutral.
- If a path is ambiguous, prefer classifying it as protected until a narrower shared seam is proven.

## Boundary Rules

Use these rules when classifying each path:

### 1. Shared/Core

Classify a path as shared/core only if it meets both conditions:

- it carries engine-neutral project state, template plumbing, orchestration, or reusable helpers
- it does not encode Claude-specific path contracts, launch semantics, hook protocol details, or UX assumptions

### 2. Claude Adapter

Classify a path as Claude adapter owned when it does any of the following:

- reads or writes `.claude/...`
- depends on Claude launch UX, permission vocabulary, or environment naming
- implements Claude hook/event/statusline behavior
- exists mainly to provision or maintain Claude-facing generated assets

### 3. Codex Adapter

Classify a path as Codex adapter owned when it is a new additive surface for:

- `.codex/...` generated assets
- Codex-specific workflow prompt packs or skills
- Codex-specific launcher, helper, or update logic
- adapter glue that should not expand Claude-owned packages

### 4. Protected Paths

Mark a path as protected when changing it increases upstream merge risk and no immediate Codex requirement forces change.

## Repo Areas To Review

The classification pass should cover these concrete areas:

- Shared-state candidates:
  - `.moai/**`
  - `internal/config/**`
  - `internal/manifest/**`
  - `internal/template/**`
  - `internal/loop/**`
  - `internal/workflow/**`
  - `internal/mcp/**`
  - `internal/core/**`
  - `internal/foundation/**`
  - `internal/update/**`
- Claude adapter candidates:
  - `.claude/**`
  - `internal/template/templates/.claude/**`
  - `internal/hook/**`
  - `internal/profile/**`
  - Claude launcher and statusline surfaces in `internal/cli/`
- Codex target seams to reserve:
  - `.codex/**`
  - `internal/template/templates/.codex/**`
  - optional `internal/cli/codex.go`
  - `internal/codex/**` or `internal/runtime/codex/**`

## Required Decisions

`CX-02` should explicitly answer these questions:

1. Which existing paths are safely shared without refactor?
2. Which existing paths are Claude-owned and should be left untouched unless unavoidable?
3. Which paths should be considered protected even if they contain some reusable logic?
4. Where should future Codex-specific code live so `CX-03` can finalize layout without re-opening ownership debates?
5. Which mixed packages, if any, may accept additive extension points without becoming adapter dumping grounds?

## Execution Steps

### Step 1. Consolidate CX-01 Findings

Translate the `CX-01` audit into an initial ownership draft:

- copy reusable modules into a provisional shared/core list
- copy Claude-locked modules into a provisional adapter/protected list
- copy Codex integration seams into a provisional Codex-owned list

Expected output:

- a first-pass ownership table with open questions

### Step 2. Validate Shared/Core Candidates

Review the roadmap's current shared/core draft against the actual repo layout and tighten it where needed.

Focus on:

- whether `internal/template/**` should remain broadly shared while keeping adapter templates separately owned
- whether `internal/core/**`, `internal/foundation/**`, and `internal/update/**` belong in the shared set or need narrower classification
- whether any CLI orchestration code should stay shared without inheriting Claude semantics

Expected output:

- a narrowed shared/core path list with rationale

### Step 3. Lock Claude-Owned And Protected Paths

Review current Claude-facing surfaces and separate:

- paths that are adapter-owned
- paths that are technically reusable but should still be protected for merge safety

Priority focus:

- `.claude/**`
- `internal/template/templates/.claude/**`
- `internal/hook/**`
- `internal/profile/**`
- Claude-specific CLI entrypoints and statusline paths

Expected output:

- a protected-path list and a "do not modify unless necessary" list

### Step 4. Reserve Codex-Owned Surfaces

Define the ownership boundary for future additive Codex work without implementing it yet.

This step should settle:

- where generated project assets will live
- where source-side templates will live
- where Codex helper/runtime code should live
- whether CLI integration gets its own file or extends an existing command surface

Expected output:

- a Codex-owned path list that feeds directly into `CX-03`

### Step 5. Write Roadmap Decisions

Update the roadmap in `Architecture Decisions` with:

- the final boundary decision
- the file ownership map
- explicit protected paths

Append an `Execution Notes` entry recording:

- the decision summary
- files reviewed
- any deferred ambiguity for `CX-03`

### Step 6. Verify Task Closure

Before closing `CX-02`, confirm:

- the `Task Index` status is updated
- the task detail `Status` line is updated
- the ownership map is specific enough to guide template and runtime placement
- no decision relies on undocumented assumptions

## Decision Heuristics

Use these heuristics when a path looks mixed:

- Prefer narrow additive seams over broad refactors.
- Prefer adapter-owned wrappers over moving existing Claude code.
- Shared/core should describe stable responsibilities, not "everything reused today."
- If a package name is generic but its behavior is Claude-bound, classify by behavior, not by name.

## Deliverable Shape

The roadmap update for `CX-02` should contain:

- one concise boundary decision section
- one explicit ownership map grouped by responsibility
- one explicit protected-path list
- one execution note with any remaining handoff context for `CX-03`

## Out Of Scope

- creating `.codex` templates
- implementing a Codex CLI command
- changing init/update behavior
- rewriting Claude profile or hook systems
- making runtime helper decisions beyond what is required to reserve ownership

## Exit Criteria

`CX-02` is ready to close when:

- shared/core paths are listed explicitly
- Claude adapter paths are listed explicitly
- Codex adapter paths are listed explicitly
- protected paths are listed explicitly
- the result is concrete enough for `CX-03 Codex Adapter Layout` to proceed without re-litigating ownership
