# CX-06 Init and Update Provisioning Plan

## Purpose

This document is the execution plan for `CX-06 Init and Update Provisioning` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to ensure `moai init` and `moai update` reliably provision and refresh Codex assets from `internal/template/templates/.codex/**`, while preserving the existing Claude flow and keeping manifest tracking aligned with the current shared template deployment model.

## Working Branch

- `feature/cx-06-init-update-provisioning`

## Dependency Note

- `CX-04` fixed the Codex template source root at `internal/template/templates/.codex/**`.
- `CX-05` turned `internal/template/templates/.codex/skills/moai/SKILL.md` into the first real Codex asset that provisioning must deliver.
- `CX-06` should consume those decisions and assets as-is. It should not reopen path ownership, entrypoint contract, or broader drift-policy questions reserved for `CX-08`.

## Task Outcome

This task makes Codex asset provisioning an explicit, verified part of project lifecycle commands.

The task is complete only when:

- fresh `moai init` creates the current Codex asset set under `.codex/**`
- `moai update` can add or refresh Codex assets in an existing initialized project
- Codex-generated files are tracked in `.moai/manifest.json` under the existing template-managed model where applicable
- the implementation remains additive and does not weaken the current Claude provisioning flow
- verification covers both init-time provisioning and update-time refresh behavior

## Inputs

- roadmap decisions and handoff notes in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- Codex template source currently in scope:
  - `internal/template/templates/.codex/skills/moai/SKILL.md`
- init and project initialization flow:
  - `internal/cli/init.go`
  - `internal/core/project/initializer.go`
- update and template sync flow:
  - `internal/cli/update.go`
- shared template deployment and manifest plumbing:
  - `internal/template/deployer.go`
  - `internal/template/deployer_mode.go`
  - `internal/template/embed.go`
  - `internal/manifest/**`
- current test surfaces most likely to absorb provisioning coverage:
  - `internal/core/project/initializer_test.go`
  - `internal/cli/update_test.go`
  - `internal/cli/target_coverage_test.go`
  - `internal/cli/coverage_improvement_test.go`
  - `internal/template/deployer_test.go`
  - `internal/template/deployer_mode_test.go`

## Current Code Observations

- `moai init` already wires `template.EmbeddedTemplates()` into `template.NewDeployerWithRenderer(...)`, so any embedded `.codex/**` template is a candidate to be provisioned during normal initialization without special-case Codex code.
- `projectInitializer.deployTemplates(...)` already loads the manifest and tracks deployed files through the shared deployer, which means Codex assets should inherit the same template-managed tracking model if they are deployed through that path.
- `moai update` already uses `template.NewDeployerWithRendererAndForceUpdate(...)` over the full embedded template tree, so Codex refresh may already work transitively, but the behavior is not yet locked by task-specific verification.
- `cleanMoaiManagedPaths(...)` is currently Claude- and `.moai/config`-focused. `CX-06` should only extend cleanup or update orchestration if verification shows the current generic deploy path is insufficient for `.codex/**`.
- Merge/overwrite policy for user-modified Codex assets should stay minimal in this task. Broader Codex drift semantics belong to `CX-08`.

## Working Assumptions

- The first `CX-06` implementation should prefer proving and tightening existing generic provisioning behavior before adding new Codex-specific branches.
- `.codex/**` remains adapter-owned generated output, but deployment should continue flowing through shared template/manifest plumbing rather than a separate Codex-only installer.
- The current Codex asset set is small, so targeted integration-style tests are a better fit than introducing a new provisioning abstraction.
- If repeated `.codex` path literals begin to spread across init/update logic, a small shared constant in `internal/defs` is acceptable, but only if it reduces duplication cleanly.
- `CX-06` should not claim or implement advanced overwrite policy beyond what the current manifest/deployer model already supports.

## Execution Steps

### Step 1. Verify The Real Baseline Before Changing Logic

Confirm what already works transitively today for both commands.

Check for:

- whether `moai init` already deploys `.codex/skills/moai/SKILL.md`
- whether the deployed Codex asset is written into `.moai/manifest.json`
- whether `moai update` already introduces the Codex asset into an existing project when it is absent
- whether update refreshes the Codex asset content when the embedded template changes

Expected output:

- a precise list of behaviors that are already present versus the gaps that need code changes

### Step 2. Add The Smallest Necessary Provisioning Changes

If baseline verification shows gaps, patch only the narrow seam responsible for them.

Likely touch points:

- `internal/cli/init.go`
- `internal/core/project/initializer.go`
- `internal/cli/update.go`
- `internal/defs/**` only if a shared Codex path constant becomes justified

Rules:

- keep provisioning additive through the shared template deployer and manifest manager
- do not widen Claude-owned launcher, hook, or template ownership
- avoid introducing Codex-specific branching unless the current generic deploy path genuinely misses a required lifecycle step

Expected output:

- minimal code changes that make init/update Codex provisioning explicit and reliable

### Step 3. Lock Manifest Behavior For Codex-Generated Files

Ensure the task completion criteria around tracking are covered concretely.

Focus on:

- confirming `.codex/skills/moai/SKILL.md` is tracked as `template_managed`
- confirming update-time refresh preserves manifest consistency after overwrite or first install
- avoiding silent reliance on template tests alone for lifecycle behavior

Expected output:

- lifecycle-oriented assertions that prove Codex assets are not only deployed, but tracked

### Step 4. Add Targeted Init And Update Regression Tests

Write narrow tests around the real lifecycle entrypoints instead of relying only on unit deployer coverage.

Priority coverage:

- init creates `.codex/skills/moai/SKILL.md` in a fresh project
- init writes a manifest entry for the Codex skill
- update installs the Codex skill into an already initialized project missing `.codex/**`
- update refreshes the Codex skill content from embedded templates without regressing existing Claude-oriented flow

Likely files:

- `internal/core/project/initializer_test.go`
- `internal/cli/update_test.go`
- `internal/cli/target_coverage_test.go`
- `internal/cli/coverage_improvement_test.go`

Expected output:

- tests that fail if Codex provisioning disappears from either lifecycle command

### Step 5. Reconfirm Additive Safety And Record Handoff

Before closing the task, verify that the work stayed inside the allowed shared seams and Codex-owned output paths.

Check for:

- no `.claude/**` renames, relocations, or Codex-specific behavior leakage
- no new dependence on Claude hooks, launcher semantics, or statusline runtime
- clear handoff notes for `CX-08` about any remaining Codex drift-policy decisions
- clear handoff notes for `CX-09` if provisioning exposes a helper gap better solved by Codex runtime support later

Expected output:

- an additive diff centered on lifecycle provisioning, lifecycle tests, and roadmap execution notes

## Deliverable Shape

The implementation for `CX-06` should ideally leave:

- verified init/update provisioning for the current Codex asset set
- manifest-backed tracking for provisioned Codex files using the shared model
- targeted lifecycle tests for init and update
- roadmap notes recording what was already generic, what needed adjustment, and what remains deferred to `CX-08`
- no Claude-adapter ownership changes

## Verification Focus

Primary verification should cover the actual lifecycle entrypoints:

- `go test ./internal/core/project/...`
- `go test ./internal/cli/...`
- `go test ./internal/template/...`

If the implementation remains mostly test-only because generic provisioning already worked, that is acceptable, but the verification must still prove the roadmap completion criteria explicitly.

## Out Of Scope

- redesigning the shared deployer or manifest model for future Codex-specific policy
- adding deep Codex drift-resolution semantics for user-modified `.codex/**` files
- implementing richer Codex runtime helpers under `internal/codex/**`
- expanding the Codex workflow prompt pack beyond the existing `$moai` entry contract
- changing protected Claude-owned paths to make Codex provisioning work
