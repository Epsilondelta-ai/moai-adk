# CX-09 Codex Runtime Helpers Plan

## Purpose

This document is the execution plan for `CX-09 Codex Runtime Helpers` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to add only the Codex-specific CLI/runtime helpers that are actually needed to make the current Codex workflow pack operational, while keeping the implementation additive under the reserved Codex-owned seams and avoiding speculative Claude-parity work.

## Working Branch

- `feature/cx-09-codex-runtime-helpers`

## Dependency Note

- `CX-06` already proved that Codex assets are provisioned through the shared init/update lifecycle, so `CX-09` does not need to invent another provisioning path.
- `CX-07` already established the Codex `$moai` skill and workflow pack, and those docs explicitly say to acknowledge missing helpers instead of pretending parity exists.
- `CX-08` already tightened manifest/update state for `.codex/**`, so `CX-09` can focus on runtime/helper gaps rather than drift policy or file ownership.
- `CX-09` should build directly on those decisions. It should not reopen provisioning, template layout, manifest policy, or Claude-owned runtime boundaries.

## Task Outcome

This task is about proving which helper surface is necessary for Codex and implementing only that minimum surface.

The task is complete only when:

- the current Codex workflow pack has been audited for helper gaps that cannot be handled cleanly with prompt-only guidance
- the helper scope is frozen to a narrow additive Codex surface before implementation expands
- any approved helper entrypoints live under the reserved Codex-owned seams such as `internal/cli/codex.go` and `internal/codex/**`
- helper output is usable by Codex workflows without claiming Claude hook, launcher, or statusline parity
- regression tests cover the chosen helper behavior strongly enough to keep the new surface stable
- the roadmap records which helpers were added, which candidate helpers were rejected, and what `CX-10` must verify

## Inputs

- roadmap decisions and handoff notes in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- current Codex prompt surface:
  - `internal/template/templates/.codex/skills/moai/SKILL.md`
  - `internal/template/templates/.codex/skills/moai/workflows/project.md`
  - `internal/template/templates/.codex/skills/moai/workflows/plan.md`
  - `internal/template/templates/.codex/skills/moai/workflows/run.md`
  - `internal/template/templates/.codex/skills/moai/workflows/sync.md`
  - `internal/template/templates/.codex/skills/moai/workflows/review.md`
  - `internal/template/templates/.codex/skills/moai/workflows/clean.md`
  - `internal/template/templates/.codex/skills/moai/workflows/loop.md`
- current CLI surfaces most likely to be extended:
  - `internal/cli/root.go`
  - `internal/cli/doctor.go`
  - `internal/cli/update.go`
  - `internal/cli/init.go`
- currently reserved but still-unused Codex runtime seams:
  - `internal/cli/codex.go`
  - `internal/codex/**`
- shared path definitions that may need a small additive constant if Codex path literals start spreading:
  - `internal/defs/dirs.go`
- likely verification surfaces:
  - `internal/cli/root_test.go`
  - `internal/cli/doctor_test.go`
  - `internal/cli/update_test.go`
  - any new `internal/codex/**/*_test.go`

## Current Code Observations

- The roadmap already fixed `internal/cli/codex.go` and `internal/codex/**` as the Codex-owned helper seams, but neither surface exists yet in the repository.
- The root CLI currently groups commands under launch, project, and tools, and there is no Codex-specific subcommand tree wired into `internal/cli/root.go`.
- The existing `moai doctor` command is the closest reusable helper surface, but its diagnostics are Claude-oriented today. It checks `.moai/` and `.claude/` presence, not `.codex/` readiness or Codex skill availability.
- The current Codex `$moai` skill contract explicitly says to state helper limitations directly when Codex does not yet have the needed support.
- The Codex workflow pack already covers `project`, `plan`, `run`, `sync`, `review`, `clean`, and `loop`, so `CX-09` should focus on the concrete places where those workflows need structured local help rather than rewriting prompt docs from scratch.
- The roadmap's possible scope mentions `moai codex doctor`, `moai codex sync`, and helper output for skill consumption, but those are candidates rather than pre-approved deliverables.
- Shared update logic in `internal/cli/update.go` already handles template synchronization, so a new `moai codex sync` helper is only justified if Codex needs a narrower machine-friendly sync/report surface that `moai update` and current workflow guidance cannot provide cleanly.

## Working Assumptions

- Prompt-only behavior remains the default. A helper should be added only when it removes a real repeatability or discoverability gap in Codex workflows.
- The first useful helper, if any, is more likely diagnostic or machine-readable than a broad new orchestration command.
- If an existing shared command can satisfy the need with a narrow additive extension, prefer that over introducing a large Codex-only command tree.
- Codex helper output should support skill consumption through stable text or JSON-like output, but should not require Claude-only hooks, slash commands, or statusline integration.
- If the minimum viable outcome is "only one helper plus explicit non-goals for the rest," that is acceptable. `CX-09` is specifically constrained to avoid speculative parity work.

## Questions This Task Must Resolve

`CX-09` should answer these questions explicitly before implementation is considered complete:

- Which current Codex workflows fail or degrade materially because a helper does not exist today?
- Is the biggest gap a human-facing diagnostic command, a machine-friendly helper output mode, or a narrow sync/status report?
- Can the gap be closed by extending existing shared diagnostics, or does it require a new Codex-owned package under `internal/codex/**`?
- Should `moai codex doctor` be a distinct subcommand, or should the existing `doctor` surface become adapter-aware with Codex checks layered in additively?
- Does `moai codex sync` provide unique value beyond `moai update`, current workflow docs, and direct repository operations, or is it just duplicate naming?
- What concrete verification will prove the helper is useful for Codex without coupling it to Claude runtime behavior?

## Candidate Helper Decision Order

The task should evaluate helper candidates in this order:

1. `moai codex doctor`
2. helper output for skill consumption
3. `moai codex sync`

Rationale:

- diagnostics are the most obvious current gap because existing checks are still Claude-biased
- machine-friendly output is a narrow extension that could make Codex workflows more deterministic without expanding command count aggressively
- a new sync command is the highest-risk candidate for overlap with existing `moai update` and prompt-level workflow guidance, so it should be justified last

## Execution Steps

### Step 1. Audit Real Helper Gaps Across The Codex Workflow Pack

Read the Codex workflow docs and map where the agent currently has to rely on vague manual inspection because no helper exists.

Check for:

- repeated repository checks that the workflows would benefit from turning into one command
- readiness questions around `.moai/**`, `.codex/**`, git, or generated assets that are awkward to answer consistently by hand
- places where the workflow pack currently warns about missing helpers
- overlap between proposed helper ideas and existing shared commands like `moai doctor` or `moai update`

Expected output:

- a short prioritized gap list that names the exact workflow pain points worth solving

### Step 2. Freeze The Minimum Helper Surface Before Writing Code

Choose the smallest Codex helper shape that closes the highest-value gap.

Focus on:

- whether the helper belongs in a new `moai codex ...` subtree or as an additive extension of an existing command
- whether any reusable logic should live in `internal/codex/**` instead of bloating CLI files directly
- whether helper output needs a stable structured mode for skill consumption
- which candidate helpers are explicitly rejected for now and why

Expected output:

- one frozen helper surface with an explicit non-goal list for the rejected candidates

### Step 3. Implement The Chosen Helper At Codex-Owned Seams

Patch only the narrow seams that actually own the new behavior.

Likely touch points:

- `internal/cli/root.go`
- `internal/cli/codex.go`
- `internal/cli/doctor.go`
- `internal/codex/**`
- `internal/defs/dirs.go`

Rules:

- keep the change additive and Codex-scoped
- do not widen Claude launchers, hooks, statusline code, or profile storage
- do not add a broad framework for hypothetical future adapters
- prefer a small reusable helper package over large command functions if multiple commands need the same Codex readiness logic

Expected output:

- one concrete Codex helper path with narrowly owned implementation

### Step 4. Lock Behavior With Focused CLI And Runtime Tests

Add regression coverage only where the helper contract can realistically break.

Priority coverage:

- root command wiring if a new `codex` subtree is added
- command output and exit behavior for the chosen helper
- Codex-specific readiness checks such as `.codex/**` presence, skill asset expectations, or machine-friendly output markers
- negative cases proving the helper reports missing Codex setup clearly without confusing it with Claude-only failures

Likely files:

- `internal/cli/root_test.go`
- `internal/cli/doctor_test.go`
- new `internal/cli/codex*_test.go`
- new `internal/codex/**/*_test.go`

Expected output:

- tests that fail if the helper disappears, stops reporting Codex readiness correctly, or regresses into Claude-specific assumptions

### Step 5. Update Prompt Contracts Only Where The Helper Changes Reality

Adjust the Codex skill docs only after the helper surface is real.

Focus on:

- replacing any "helper missing" limitation text that is no longer true
- referencing the helper only in the workflows that genuinely benefit from it
- keeping the prompt pack honest about what still remains manual

Likely files:

- `internal/template/templates/.codex/skills/moai/SKILL.md`
- selected files under `internal/template/templates/.codex/skills/moai/workflows/`

Expected output:

- Codex workflow docs that reference the helper accurately and narrowly

### Step 6. Write The Final Scope Decision Back Into The Roadmap

Close the task by recording both what shipped and what was intentionally omitted.

Check for:

- `CX-09` status and task index updates in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- execution notes that explain the chosen helper surface and rejected alternatives
- verification notes for `CX-10`, especially around helper discoverability and CLI behavior
- handoff notes for `CX-11` if a new Codex command surface changes merge-maintenance expectations

Expected output:

- a resumable roadmap record with explicit helper-scope decisions

## Deliverable Shape

The implementation for `CX-09` should ideally leave:

- either one narrow Codex helper command or one narrow shared command extension with Codex-specific behavior
- any reusable Codex readiness logic isolated under `internal/codex/**` if that reduces CLI duplication cleanly
- focused tests for helper wiring and output
- prompt-pack updates only where the new helper materially changes workflow guidance
- roadmap notes that explain why some candidate helpers were rejected or deferred
- no Claude-adapter ownership changes

## Verification Focus

Primary verification should stay narrow and behavior-driven:

- `go test ./internal/cli/...`

If shared helper logic lands under the reserved runtime seam, add:

- `go test ./internal/codex/...`

If the helper changes generated workflow guidance materially, add:

- `go test ./internal/template/...`

## Out Of Scope

- redesigning the Codex workflow prompt pack beyond helper-related wording updates
- reopening `.codex/**` template layout, provisioning flow, or manifest/drift policy
- porting Claude hook, slash-command, statusline, or launcher semantics into Codex
- broad adapter abstraction work for runtimes other than Codex
- adding multiple Codex helper commands without proving each one is necessary
- claiming full verification coverage for `CX-10`
