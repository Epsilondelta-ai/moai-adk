# CX-10 Verification Matrix Plan

## Purpose

This document is the execution plan for `CX-10 Verification Matrix` from `.moai/docs/CODEX_COMPAT_ROADMAP.md`.

The goal is to verify that the Codex adapter can be added safely to an already initialized MoAI project, that the new Codex workflow surfaces actually work end to end, and that the additive Codex changes do not regress the existing Claude-facing paths.

## Working Branch

- `feature/cx-10-verification-matrix`

## Dependency Note

- `CX-05` introduced the real `$moai` Codex entry contract and established the skill surface that `CX-10` now has to prove is discoverable and usable.
- `CX-06` moved Codex asset provisioning into the shared `init` and `update` lifecycle, so `CX-10` must verify Codex deployment on an already initialized `.moai` project rather than inventing a separate install path.
- `CX-07` added the seven-file Codex workflow prompt pack, which means `CX-10` should verify both skill entrypoint discovery and workflow-pack completeness.
- `CX-08` defined the manifest and drift-state rules for `.codex/**`, so `CX-10` must verify repeated update behavior for untouched, drifted, deleted, and deprecated Codex-managed files against the recorded manifest outcomes.
- `CX-09` added `moai codex doctor` and a narrow `--json` mode backed by `internal/codex/**`, so `CX-10` must verify positive and negative readiness output, command discoverability, and expected behavior when Codex assets are missing or incomplete.
- `CX-10` should treat all of those decisions as fixed inputs. It should verify them, not reopen architecture, template ownership, or helper-scope decisions.

## Task Outcome

This task is about converting prior design and implementation work into a concrete verification matrix that can be rerun after future Codex changes.

The task is complete only when:

- the verification matrix covers both already-initialized project upgrades and fresh initialization behavior where the roadmap template requires it
- Codex asset deployment and update-state behavior are verified against the `CX-08` manifest policy
- `$moai` skill discovery plus core workflow-pack presence are verified in the deployed `.codex/**` tree
- `moai codex doctor` is verified across success and failure cases, including missing `.moai/**`, missing `.codex/**`, and incomplete workflow-pack scenarios
- Claude-facing behavior still passes targeted regression checks after the additive Codex work
- the roadmap `Verification Log` receives concrete pass/fail notes and commands rather than a vague summary

## Inputs

- roadmap requirements and follow-up notes in `.moai/docs/CODEX_COMPAT_ROADMAP.md`
- prior task plans:
  - `.moai/docs/CX-05_MOAI_SKILL_ENTRY_POINT_PLAN.md`
  - `.moai/docs/CX-06_INIT_AND_UPDATE_PROVISIONING_PLAN.md`
  - `.moai/docs/CX-07_CODEX_WORKFLOW_PROMPT_PACK_PLAN.md`
  - `.moai/docs/CX-08_MANIFEST_AND_DRIFT_POLICY_PLAN.md`
  - `.moai/docs/CX-09_CODEX_RUNTIME_HELPERS_PLAN.md`
- Codex CLI/runtime seams under test:
  - `internal/cli/root.go`
  - `internal/cli/codex.go`
  - `internal/cli/doctor.go`
  - `internal/codex/doctor.go`
- shared provisioning and update behavior under test:
  - `internal/cli/init.go`
  - `internal/cli/update.go`
  - `internal/update/**`
  - `internal/template/**`
  - `internal/manifest/**`
- deployed Codex asset surface under test:
  - `internal/template/templates/.codex/skills/moai/SKILL.md`
  - `internal/template/templates/.codex/skills/moai/workflows/project.md`
  - `internal/template/templates/.codex/skills/moai/workflows/plan.md`
  - `internal/template/templates/.codex/skills/moai/workflows/run.md`
  - `internal/template/templates/.codex/skills/moai/workflows/sync.md`
  - `internal/template/templates/.codex/skills/moai/workflows/review.md`
  - `internal/template/templates/.codex/skills/moai/workflows/clean.md`
  - `internal/template/templates/.codex/skills/moai/workflows/loop.md`
- likely regression test surfaces:
  - `internal/cli/root_test.go`
  - `internal/cli/codex_test.go`
  - `internal/cli/doctor_test.go`
  - `internal/cli/update_test.go`
  - `internal/cli/init_test.go`
  - `internal/codex/doctor_test.go`

## Current Code Observations

- The repository now has a real `codex` command tree in `internal/cli/codex.go`, so `CX-10` can verify command discoverability directly instead of treating the helper surface as hypothetical.
- `internal/codex/doctor.go` currently checks four readiness categories: `git`, shared `.moai/**` state, `.codex/skills/moai/SKILL.md`, and the seven expected workflow documents under `.codex/skills/moai/workflows/`.
- The roadmap verification template is still shorter than the real verification surface now required by `CX-08` and `CX-09`, so `CX-10` should expand the recorded results to cover drift-state transitions and negative readiness cases explicitly.
- The Codex workflow pack exists as embedded template assets, but the current roadmap log does not yet prove that repeated `update` cycles preserve the expected manifest provenance for local drift and upstream template removal.
- Claude behavior remains upstream-owned and mostly outside the Codex seams, which means `CX-10` should prefer narrow regression checks around shared init/update/root wiring rather than broad speculative Claude revalidation.

## Verification Matrix Dimensions

`CX-10` should organize verification around these axes:

1. Provisioning path
2. Asset state transition
3. Codex workflow readiness
4. Claude non-regression
5. Test-suite execution

For each verification row, record:

- scenario
- setup state
- command or test entrypoint
- expected result
- actual result
- pass/fail note

## Working Assumptions

- `CX-10` is primarily a verification and documentation task. It should add or adjust tests only where existing coverage does not prove the required behavior cleanly.
- The most important environment is an already initialized `.moai` project receiving Codex assets through `update`, because that is the roadmap's stated migration path.
- A small number of targeted negative fixtures is better than one large end-to-end script if the smaller fixtures make failure causes easier to understand.
- Claude parity does not mean feature parity. The regression goal is that additive Codex work does not break existing Claude-owned commands and shared lifecycle behavior.
- If a planned verification step exposes a real bug, the bug should be fixed in task scope because an unverified matrix is not a finished outcome.

## Questions This Task Must Resolve

`CX-10` should answer these questions explicitly before it is considered complete:

- Can an existing `.moai` project receive the Codex asset pack via the shared update lifecycle without requiring a separate Codex bootstrap command?
- Do repeated updates preserve the correct manifest state for untouched, drifted, deleted, and deprecated `.codex/**` files?
- Is `$moai` discoverable in the deployed Codex skill surface, and is the workflow pack complete after deployment?
- Does `moai codex doctor` report readiness correctly in both human-readable and `--json` output modes?
- Are the negative readiness cases clear and actionable when `.moai/**`, `.codex/**`, or workflow files are missing?
- Do shared CLI paths still behave correctly for Claude-oriented flows after the Codex additions?
- Which verification commands and tests are strong enough to record permanently in the roadmap `Verification Log`?

## Execution Steps

### Step 1. Build The Verification Matrix From The Roadmap Commitments

Translate the roadmap minimum checks plus `CX-08` and `CX-09` follow-up notes into an explicit table or checklist.

The matrix should include:

- update into an already initialized `.moai` project
- fresh init behavior if that path is still covered by the roadmap template
- manifest-state checks for untouched, drifted, deleted, and deprecated `.codex/**` files
- `moai codex doctor` success and failure cases
- `$moai` discovery and workflow-pack completeness
- targeted Claude non-regression checks

Expected output:

- one concrete verification matrix that maps every prior commitment to a rerunnable check

### Step 2. Verify Provisioning And Update-State Transitions

Exercise the shared init/update lifecycle against realistic project states.

Focus on:

- deploying Codex assets into a project that already has `.moai/**`
- rerunning `update` with untouched Codex files
- preserving local drift correctly after `update`
- restoring locally deleted active Codex assets
- preserving deprecated template paths when an embedded Codex file disappears upstream

Likely touch points:

- `internal/cli/update.go`
- `internal/update/**`
- `internal/manifest/**`
- `internal/template/**`

Expected output:

- verification evidence that the `CX-08` manifest policy holds across repeated update cycles

### Step 3. Verify Codex Readiness And Workflow Discoverability

Prove that the shipped Codex surface is visible and usable after deployment.

Focus on:

- `.codex/skills/moai/SKILL.md` presence and discoverability expectations
- completeness of the seven workflow documents
- `moai codex doctor` default output
- `moai codex doctor --json` structured output
- negative readiness cases for missing `.moai/**`, missing `.codex/**`, and incomplete workflow packs

Likely touch points:

- `internal/cli/codex.go`
- `internal/codex/doctor.go`
- `internal/cli/codex_test.go`
- `internal/codex/doctor_test.go`

Expected output:

- repeatable readiness checks with clear pass/fail outcomes for both positive and negative states

### Step 4. Run Shared CLI And Claude Non-Regression Coverage

Verify that Codex additions did not break shared behavior or Claude-facing flows.

Focus on:

- root command wiring and command grouping
- shared `init` and `update` behavior
- existing Claude-oriented diagnostics remaining intact
- any tests that cover shared template embedding or CLI registration paths touched by Codex work

Likely commands:

- `go test ./internal/cli/...`
- targeted package tests under `internal/template/...`, `internal/update/...`, and `internal/manifest/...`

Expected output:

- a narrow but defensible regression result showing additive Codex work did not break existing shared or Claude-owned surfaces

### Step 5. Record Final Results In The Roadmap

Once the matrix has been executed, write the results back to `.moai/docs/CODEX_COMPAT_ROADMAP.md`.

Update at least:

- `CX-10` task status and task index entry
- `Verification Log` with the actual commands and outcomes
- execution notes summarizing what passed, what failed, and any follow-up handed to `CX-11`

Expected output:

- a resumable roadmap record that turns `CX-10` from a pending checklist into a concrete verification baseline

## Deliverable Shape

The final `CX-10` outcome should ideally leave:

- a committed verification plan and execution record for Codex compatibility
- any minimal missing regression tests needed to prove roadmap commitments
- roadmap log entries that name the exact verification commands and scenarios run
- a clear statement of residual risk if any scenario remains intentionally unverified
