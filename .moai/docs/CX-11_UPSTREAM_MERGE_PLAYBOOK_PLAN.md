# CX-11 Upstream Merge Playbook Plan

## Purpose

This document is the execution plan for `CX-11 Upstream Merge Playbook` from `.moai/docs/CODEX_COMPAT_ROADMAP.md`.

The goal is to turn the current ownership and verification decisions into a concrete upstream merge playbook that keeps future rebases or merges low-conflict, preserves additive Codex support, and avoids accidental regressions in Claude-owned surfaces.

## Working Branch

- `codex/cx-11-upstream-merge-playbook`

## Dependency Note

- `CX-02` defined the shared/core versus adapter-owned boundary, which means `CX-11` should treat ownership classification as fixed input rather than reopening path ownership.
- `CX-03` fixed the additive Codex layout under `.codex/**`, `internal/template/templates/.codex/**`, `internal/cli/codex.go`, and `internal/codex/**`, so the merge playbook must keep those paths isolated from upstream Claude-owned changes whenever possible.
- `CX-06` and `CX-08` moved Codex provisioning into shared init/update flows and defined manifest drift behavior, which means merge guidance must call out shared template and manifest code as intentional conflict hotspots.
- `CX-10` established the compatibility verification baseline, so `CX-11` should reuse that baseline as the required post-merge review and test checkpoint set.
- The roadmap already notes that deprecated Codex-managed assets are intentionally preserved when upstream removes a workflow file permanently. `CX-11` must document that this state may require manual cleanup instead of treating it as an automatic conflict-resolution failure.

## Task Outcome

This task is about writing the operating playbook for future upstream syncs, not about changing the underlying adapter architecture.

The task is complete only when:

- the roadmap `Upstream Strategy` section becomes a concrete merge/rebase playbook rather than a short policy stub
- the playbook lists protected Claude-owned paths explicitly
- the playbook lists Codex-owned paths explicitly
- the playbook identifies expected conflict hotspots in shared packages
- the playbook defines a recommended merge order with review checkpoints
- the playbook explains how to handle preserved deprecated Codex assets after upstream removes a managed file

## Inputs

- roadmap requirements and follow-up notes in `.moai/docs/CODEX_COMPAT_ROADMAP.md`
- prior task plans:
  - `.moai/docs/CX-02_SHARED_CORE_BOUNDARY_PLAN.md`
  - `.moai/docs/CX-03_CODEX_ADAPTER_LAYOUT_PLAN.md`
  - `.moai/docs/CX-06_INIT_AND_UPDATE_PROVISIONING_PLAN.md`
  - `.moai/docs/CX-08_MANIFEST_AND_DRIFT_POLICY_PLAN.md`
  - `.moai/docs/CX-10_VERIFICATION_MATRIX_PLAN.md`
- roadmap ownership and merge-governance sections:
  - `File Ownership Map`
  - `Protected Paths`
  - `Verification Log`
  - `Upstream Strategy`
- likely shared conflict surfaces:
  - `internal/cli/init.go`
  - `internal/cli/update.go`
  - `internal/profile/sync.go`
  - `internal/template/embed.go`
  - `internal/template/deployer.go`
  - `internal/template/renderer.go`
  - `internal/template/templates/**`
  - `internal/manifest/**`
- Codex-owned additive surfaces that should usually stay local-first during conflict review:
  - `.codex/**`
  - `internal/template/templates/.codex/**`
  - `internal/cli/codex.go`
  - `internal/codex/**`

## Current Code Observations

- The roadmap already contains the ownership map, protected paths, and a minimal upstream strategy, so `CX-11` should consolidate existing decisions into an operator-facing playbook instead of inventing new architecture.
- Most likely merge conflicts will not come from `.codex/**` itself, but from shared provisioning and template sync seams that now know about Codex assets.
- Claude-facing command files and hook/profile paths remain upstream-owned and should be treated as protected during conflict resolution unless a Codex requirement cannot be met additively.
- `CX-10` already recorded a strong verification baseline around `init`, `update`, manifest drift handling, Codex readiness, and full test-suite non-regression. That baseline is the natural post-merge checkpoint set.
- Deprecated managed files are a deliberate manifest state, not an accidental leftover. The playbook must distinguish between intentional preservation and stale local junk.

## Working Assumptions

- `CX-11` is primarily a documentation and maintenance-guidance task. It should not broaden adapter seams unless the current roadmap cannot express the merge policy clearly.
- The most useful playbook is specific to this repository's ownership map and verification baseline, not a generic Git tutorial.
- Merge guidance should prefer additive resolution strategies that keep upstream as authority for Claude-owned behavior while preserving Codex-owned paths and narrow shared seams.
- Review checkpoints should be short enough to run regularly after upstream syncs, with a clear escalation path to the full `go test ./...` sweep when shared surfaces changed.

## Questions This Task Must Resolve

`CX-11` should answer these questions explicitly before it is considered complete:

- Which path groups should default to upstream authority, local authority, or case-by-case review during merge conflicts?
- Which shared packages are expected conflict hotspots because they host additive Codex seams inside otherwise upstream-driven flows?
- In what order should an upstream merge or rebase be reviewed so ownership mistakes are caught before tests run?
- Which targeted checks should run after resolving shared-surface conflicts, and when is the full repository test sweep required?
- How should engineers treat manifest entries or deployed files that remain in a `deprecated` state after upstream removes a Codex-managed asset?
- What review notes should be captured when a merge requires non-additive changes in currently protected Claude-owned paths?

## Execution Steps

### Step 1. Convert Ownership Rules Into Merge Authority Rules

Translate the roadmap ownership map into explicit merge guidance.

This should classify at least:

- shared/core paths that require careful manual review
- Claude-owned protected paths that default to upstream authority
- Codex-owned additive paths that default to preserving local Codex work
- extension seams where upstream and Codex changes can legitimately meet

Expected output:

- a clear authority model that tells reviewers which side should usually win for each path class

### Step 2. Identify Expected Conflict Hotspots

Document the files and directories most likely to conflict during future upstream syncs.

Focus on:

- shared CLI lifecycle wiring in `internal/cli/init.go` and `internal/cli/update.go`
- shared template deployment and rendering seams in `internal/template/**`
- manifest state handling in `internal/manifest/**`
- profile/config sync seams such as `internal/profile/sync.go`
- any roadmap or template files where both upstream and Codex maintenance commonly land

Expected output:

- a hotspot list with short reasoning for why each area is conflict-prone

### Step 3. Define The Recommended Merge Order And Review Checkpoints

Write the actual playbook sequence for rebasing or merging upstream.

The sequence should cover:

- pre-merge inventory of changed files against the ownership map
- resolving protected Claude-owned paths conservatively
- resolving shared seams before validating Codex-owned additive paths
- reviewing deprecated managed assets for intentional preservation versus manual cleanup
- running targeted verification first and the full regression sweep when required

Expected output:

- a step-by-step merge order with concrete review checkpoints tied to repository-specific risks

### Step 4. Specify Post-Merge Verification Gates

Bind the playbook to the `CX-10` verification baseline instead of leaving verification vague.

At minimum, document when to run:

- targeted CLI and template tests for shared-surface conflicts
- Codex readiness checks or their underlying test coverage
- manifest drift-policy coverage when template or update behavior changes
- `go test ./...` as the final high-confidence sweep for meaningful shared-surface merges

Expected output:

- a concise verification ladder that scales with the size and location of the merge diff

### Step 5. Write The Final Playbook Into The Roadmap

Once the merge authority rules, hotspots, order, and checkpoints are defined, update `.moai/docs/CODEX_COMPAT_ROADMAP.md`.

Update at least:

- `CX-11` task status and task index entry when execution is complete
- `Upstream Strategy` with the full merge playbook
- `Verification Log` with any concrete validation run for the documentation update
- execution notes summarizing the merge-policy decisions and any residual manual steps

Expected output:

- a roadmap section that future Codex work can use as the default upstream sync procedure

## Deliverable Shape

The final `CX-11` outcome should ideally leave:

- a committed roadmap update that turns `Upstream Strategy` into a repository-specific merge playbook
- explicit lists of protected Claude-owned paths, Codex-owned additive paths, and shared conflict hotspots
- a recommended merge order with review checkpoints that reference the `CX-10` verification baseline
- a clear note that deprecated Codex-managed assets may remain intentionally preserved and can require manual cleanup after upstream removals
