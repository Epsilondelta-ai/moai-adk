# CX-04 Codex Template Scaffold Plan

## Purpose

This document is the execution plan for `CX-04 Codex Template Scaffold` from `.moai/docs/CODEX_COMPAT_ROADMAP.md`.

The goal is to add the first Codex-owned source template tree under `internal/template/templates/.codex/**` so later tasks can provision `.codex/**` assets without reopening layout or ownership decisions.

## Working Branch

- `feature/cx-04-codex-template-scaffold`

## Task Outcome

This task produces a minimal, additive Codex template scaffold.

The task is complete only when:

- a real source template subtree exists under `internal/template/templates/.codex/**`
- the scaffold reserves the future Codex skill entry path needed by `CX-05`
- template files are additive and do not rename, relocate, or remove any `.claude/**` asset
- template embedding and deployment tests cover the new `.codex` subtree well enough to catch regressions

## Inputs

- `CX-03` layout decisions already recorded in the roadmap
- current template embedding and deployment code in:
  - `internal/template/embed.go`
  - `internal/template/deployer.go`
  - `internal/template/renderer.go`
- current template verification in:
  - `internal/template/embed_test.go`
  - `internal/template/deployer_test.go`
  - `internal/template/deployer_mode_test.go`
- current source template trees:
  - `internal/template/templates/.claude/**`
  - `internal/template/templates/.moai/**`

## Working Assumptions

- `.codex/**` is already fixed as the generated project asset root and must not be debated again in this task.
- `internal/template/templates/.codex/**` is already fixed as the source template root and must be created as a sibling of `.claude/**` and `.moai/**`.
- `CX-04` should create real tracked files, not empty directories, because the embed/deploy system operates on files.
- The scaffold should be minimal. Do not recreate Claude hooks, statusline assets, launcher-specific files, or other Claude-only behavior under Codex names.
- Where later tasks clearly need a concrete file path, prefer a small stub file over `.gitkeep` so future work can extend the same managed asset.

## Minimum Scaffold Target

`CX-04` should reserve the smallest file set that makes the Codex tree real and gives `CX-05` a stable handoff path.

Default target:

- `internal/template/templates/.codex/skills/moai/SKILL.md` as the future `$moai` skill entrypoint path

If implementation shows that Codex asset discovery needs one more tracked file for clarity, add only the minimum additional scaffold file under `.codex/**` and document why. Do not introduce speculative parallel trees such as Claude-style hooks or commands unless a concrete Codex requirement forces it.

## Execution Steps

### Step 1. Reconfirm The Smallest Viable Codex Asset Shape

Use the roadmap handoff from `CX-03` and the upcoming `CX-05` requirement to fix the minimum scaffold file set.

Check for:

- which future Codex asset path must exist now to avoid churn in `CX-05`
- whether the scaffold should use plain `.md` content or `.tmpl` rendering
- whether any additional non-skill placeholder is actually required

Expected output:

- a short final list of source template files to add under `internal/template/templates/.codex/**`

### Step 2. Add The `.codex` Source Template Tree

Create the new template files under the fixed Codex-owned path.

Rules:

- keep content intentionally skeletal but valid
- include enough placeholder guidance that `CX-05` can evolve the same file into the real `$moai` entrypoint
- use `.tmpl` only if the file will clearly need template context during deployment; otherwise prefer plain content
- do not copy Claude-only command, hook, statusline, or launcher content into the Codex tree

Expected output:

- a new additive `internal/template/templates/.codex/**` subtree with at least the reserved `$moai` skill path represented by a real file

### Step 3. Extend Template Coverage Tests

Update template tests so the new dot-prefixed sibling tree is verified explicitly.

Focus on:

- embedded filesystem visibility for `.codex/**`
- readable/listable template paths for the new subtree
- deployer behavior for Codex-owned destination paths

Likely files:

- `internal/template/embed_test.go`
- `internal/template/deployer_test.go`
- `internal/template/deployer_mode_test.go`

Expected output:

- tests that would fail if `.codex/**` stops being embedded, listed, or deployed correctly

### Step 4. Verify Additive Safety

Confirm the scaffold stayed inside the reserved Codex-owned surface.

Check for:

- no `.claude/**` rename, relocation, or deletion
- no widening of Claude-owned runtime or CLI packages
- no requirement for new shared template plumbing beyond the already-approved seams

Expected output:

- a clean additive diff centered on `internal/template/templates/.codex/**` and focused template tests

### Step 5. Record Handoff Constraints For Follow-up Tasks

When the implementation is done, update the roadmap work log with the exact scaffold that was added and any constraints that `CX-05` and `CX-06` must preserve.

The handoff should capture:

- the reserved Codex skill path
- whether the scaffold file is plain content or rendered template
- any intentionally deferred content that belongs to `CX-05` or `CX-06`

## Deliverable Shape

The implementation for `CX-04` should ideally leave:

- new files under `internal/template/templates/.codex/**`
- targeted template tests covering the new subtree
- no changes to `.claude/**` source templates
- no provisioning changes in `internal/cli/init.go` or `internal/cli/update.go` yet

## Verification Focus

Primary verification should be targeted and template-centric:

- `go test ./internal/template/...`

If the resulting changes touch broader provisioning behavior unexpectedly, widen verification only as needed.

## Out Of Scope

- wiring `moai init` or `moai update` to provision `.codex/**`
- implementing the real `$moai` Codex workflow contract
- adding `internal/cli/codex.go`
- adding `internal/codex/**` runtime helpers
- recreating Claude hook parity or Claude command parity under Codex paths
