# CX-03 Codex Adapter Layout Plan

## Purpose

This document is the execution plan for `CX-03 Codex Adapter Layout` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to lock the Codex adapter's on-disk layout before template scaffolding or provisioning work begins, so later tasks can add assets and code without reopening ownership or merge-risk decisions.

## Task Outcome

This task produces a layout decision, not end-user behavior.

The task is complete only when the roadmap contains:

- a final Codex path plan under `Architecture Decisions`
- explicit answers for project asset root, source template root, CLI entrypoint placement, and Codex runtime/helper package placement
- a layout that keeps all existing Claude-owned paths in place

## Inputs

- `CX-02` boundary decisions already recorded in the roadmap
- current source template layout under `internal/template/templates/`
- current CLI structure under `internal/cli/`
- shared provisioning seams in:
  - `internal/cli/init.go`
  - `internal/cli/update.go`
  - `internal/template/**`
  - `internal/profile/sync.go`

## Working Assumptions

- `.moai/**` remains the adapter-neutral project-state authority.
- `.claude/**` remains an upstream-owned adapter surface and should not move.
- Codex support should be additive, merge-friendly, and path-stable.
- The default Codex asset root should be `.codex/**` unless a concrete constraint disproves it.
- The default source template root should be `internal/template/templates/.codex/**` unless template plumbing makes that impractical.
- Codex-specific helper code should live in a dedicated package instead of expanding Claude-owned packages.

## Required Decisions

`CX-03` should explicitly answer these questions:

1. Do generated project assets live under `.codex/**`?
2. Do source-side Codex templates live under `internal/template/templates/.codex/**`?
3. Does Codex CLI integration require a new `internal/cli/codex.go` entrypoint?
4. Should Codex runtime/helper code live under `internal/codex/**` or `internal/runtime/codex/**`?
5. Which shared packages may reference the new layout, and which protected paths must remain untouched?

## Decision Heuristics

Use these rules when evaluating candidate layouts:

- Prefer new Codex-owned paths over widening Claude-owned packages.
- Prefer file placement that makes future `CX-04` template scaffolding straightforward.
- Prefer path names that clearly separate generated assets from source templates and runtime helpers.
- If two layouts are technically viable, choose the one that minimizes upstream merge conflict risk.
- Do not move or rename existing Claude paths to make the Codex layout look symmetrical.

## Execution Steps

### Step 1. Reconfirm Reserved Ownership From CX-02

Translate the `CX-02` boundary decision into concrete layout candidates:

- `.codex/**` as generated project assets
- `internal/template/templates/.codex/**` as source templates
- `internal/cli/codex.go` as an additive CLI seam
- `internal/codex/**` or `internal/runtime/codex/**` as Codex helper/runtime code

Expected output:

- a shortlist of candidate layouts with any unresolved coupling concerns

### Step 2. Validate Generated Asset Root

Review existing project-level generated asset patterns and confirm whether `.codex/**` is the stable target.

Check for:

- consistency with current `.claude/**` and `.moai/**` ownership
- whether any existing manifest or template code assumes a fixed set of top-level generated roots
- whether future `moai init` and `moai update` flows can provision `.codex/**` additively

Expected output:

- a final decision on the generated asset root and any constraints that later tasks must respect

### Step 3. Validate Source Template Placement

Review the template source tree and confirm where Codex template files should live.

Check for:

- how `internal/template/templates/` is embedded and rendered
- whether `.codex` can be added as a sibling to `.claude` without special handling
- whether any naming or layout convention should be preserved for future template discovery

Expected output:

- a final decision on source template placement and any template-tree conventions to preserve

### Step 4. Validate CLI Integration Shape

Review the current CLI package layout and decide whether Codex should get a dedicated entry file.

Check for:

- how existing adapter-specific commands are separated
- whether `internal/cli/codex.go` reduces coupling versus extending existing Claude entrypoints
- whether root command wiring can stay additive

Expected output:

- a final decision on whether `internal/cli/codex.go` is required, optional, or deferred

### Step 5. Validate Runtime/Helper Package Location

Compare `internal/codex/**` against `internal/runtime/codex/**` using boundary clarity and future scope.

Check for:

- whether the package will hold only Codex-specific helpers or broader runtime abstractions
- whether `internal/runtime/` already exists or would be introduced solely for Codex
- whether the chosen path keeps future helper growth coherent without implying premature generalization

Expected output:

- a final package-home decision with rationale

### Step 6. Write Roadmap Decisions

Update the roadmap in `Architecture Decisions` with:

- the final Codex path plan
- the accepted top-level and source-side paths
- the CLI and runtime/helper placement decision
- any constraints that `CX-04`, `CX-05`, and `CX-06` must honor

Append an `Execution Notes` entry recording:

- the chosen layout
- alternatives considered and rejected
- files reviewed
- any deferred implementation details for later tasks

### Step 7. Verify Task Closure

Before closing `CX-03`, confirm:

- the `Task Index` status is updated
- the task detail `Status` line is updated
- all required decisions are answered explicitly
- no Claude-owned path needs to move or be renamed
- the decision is specific enough for `CX-04` to scaffold templates without further layout debate

## Deliverable Shape

The roadmap update for `CX-03` should contain:

- one concise layout decision section under `Architecture Decisions`
- one explicit list of fixed Codex-owned paths
- one explicit note on shared seams that may reference those paths
- one execution note with rationale and handoff guidance for `CX-04`

## Verification Focus

`CX-03` is an architecture task. Verification should therefore confirm:

- path decisions are internally consistent with `CX-02`
- the chosen layout is additive
- no protected Claude path is moved
- later tasks have a stable destination for generated assets, templates, and helper code

## Out Of Scope

- creating actual `.codex` template files
- implementing a Codex CLI command
- wiring `moai init` or `moai update` to provision Codex assets
- defining Codex workflow prompt contents
- changing Claude-owned runtime, hook, profile, or statusline behavior
