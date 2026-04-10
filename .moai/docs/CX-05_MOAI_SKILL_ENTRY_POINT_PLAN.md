# CX-05 `$moai` Skill Entry Point Plan

## Purpose

This document is the execution plan for `CX-05 $moai Skill Entry Point` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to turn the reserved Codex scaffold at `internal/template/templates/.codex/skills/moai/SKILL.md` into the real `$moai` entry contract so Codex can discover the skill, route core MoAI subcommands, and read project state from `.moai/**` without assuming Claude-only runtime features.

## Working Branch

- `feature/cx-05-moai-skill-entry-point`

## Dependency Note

- The roadmap `Task Index` still lists `CX-04` as `PENDING`, but the `CX-04` task detail, execution note, and verification log record it as completed on 2026-04-10.
- This plan treats `CX-04` as complete and uses its reserved Codex path, because `CX-05` depends on that scaffold and should not introduce path churn.

## Task Outcome

This task produces the first real Codex-facing MoAI skill contract.

The task is complete only when:

- the scaffolded file at `internal/template/templates/.codex/skills/moai/SKILL.md` is replaced with a real `$moai` skill document
- the skill clearly explains how Codex should route `$moai`, `$moai plan`, `$moai run`, and `$moai sync`
- the skill points Codex to shared project state under `.moai/**`
- the implementation stays additive under Codex-owned paths and does not recreate Claude-only hooks, commands, or launcher behavior
- template tests verify the Codex skill content strongly enough to catch regressions in discoverability or handoff guidance

## Inputs

- roadmap decisions and handoff notes in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- reserved Codex skill scaffold:
  - `internal/template/templates/.codex/skills/moai/SKILL.md`
- current Claude-side command routing patterns:
  - `internal/template/templates/.claude/commands/moai/project.md.tmpl`
  - `internal/template/templates/.claude/commands/moai/plan.md.tmpl`
  - `internal/template/templates/.claude/commands/moai/run.md.tmpl`
  - `internal/template/templates/.claude/commands/moai/sync.md.tmpl`
- current template verification coverage:
  - `internal/template/embed_test.go`
  - `internal/template/deployer_test.go`
  - `internal/template/deployer_mode_test.go`
- shared MoAI project-state roots the skill should reference:
  - `.moai/project/**`
  - `.moai/plans/**`
  - `.moai/specs/**`
  - `.moai/state/**`
  - `.moai/docs/CODEX_COMPAT_ROADMAP.md`

## Working Assumptions

- `CX-05` should preserve `.codex/skills/moai/SKILL.md` as the generated entrypoint path exactly as reserved by `CX-04`.
- The first usable Codex entrypoint can be implemented as a single skill document. Separate Codex command wrappers are not required in this task.
- Codex parity target is workflow guidance parity, not Claude hook parity or Claude slash-command parity.
- The skill should route users toward shared `.moai/**` state and existing MoAI conventions instead of duplicating workflow state inside `.codex/**`.
- Provisioning changes in `moai init` and `moai update` belong to `CX-06`, so `CX-05` should not depend on new CLI wiring to define the skill contract.
- Detailed workflow prompt packs beyond the core entry routing belong to `CX-07`, so `CX-05` should establish the contract and handoff shape, not solve every downstream workflow in full depth.

## Skill Contract Target

`CX-05` should define one canonical Codex skill contract that covers:

- what `$moai` means in Codex
- when the skill should be invoked
- which `.moai/**` files Codex should inspect first
- how `$moai plan`, `$moai run`, and `$moai sync` differ in intent and expected inputs
- what constraints apply because Codex does not provide Claude-only hooks, slash commands, statusline assets, or launcher semantics

The contract should be concise enough to stay maintainable, but specific enough that a future Codex session can resume from the skill alone plus project state.

## Execution Steps

### Step 1. Reconfirm The Smallest Real `$moai` Contract

Use the roadmap handoff from `CX-04` and the existing Claude command routing pattern to define the minimum Codex skill content that is still operationally useful.

Check for:

- which subcommands must be documented now to satisfy the roadmap completion criteria
- whether the skill should describe bare `$moai` as a dispatcher, a help surface, or both
- which `.moai/**` locations must be referenced explicitly for project-state discovery

Expected output:

- a final checklist of required sections and command routes for the Codex skill

### Step 2. Map Core Subcommands To Shared MoAI State

Translate the existing MoAI workflow phases into Codex-readable entry guidance without depending on Claude-specific command wrappers.

Focus on:

- `$moai` as the top-level entry and re-entry surface
- `$moai plan` reading roadmap, specs, and planning context from `.moai/**`
- `$moai run` reading approved spec or plan context from `.moai/specs/**` and related state
- `$moai sync` reading documentation and drift context from `.moai/**`
- any minimal mention of `project` routing if needed to explain how Codex should bootstrap project understanding

Expected output:

- a concrete route map from each supported invocation to the corresponding `.moai/**` inputs and expected behavior

### Step 3. Replace The Scaffold With The Real Codex Skill

Rewrite `internal/template/templates/.codex/skills/moai/SKILL.md` from placeholder text into the actual Codex-facing contract.

Rules:

- keep the file at the reserved path
- prefer plain Markdown unless templating is clearly required
- explicitly reference shared `.moai/**` sources of truth
- describe supported subcommands in a way Codex can follow directly
- avoid claiming unavailable Claude-only features as if Codex supported them

Expected output:

- a real `$moai` skill entrypoint document under the existing Codex template path

### Step 4. Tighten Template Regression Coverage

Update template tests so they verify the Codex skill is not merely present, but materially useful.

Focus on:

- embedded template content containing the real `$moai` contract markers
- deploy/extract behavior for `.codex/skills/moai/SKILL.md`
- content assertions that catch accidental regression back to skeletal scaffold text

Likely files:

- `internal/template/embed_test.go`
- `internal/template/deployer_test.go`
- `internal/template/deployer_mode_test.go`

Expected output:

- tests that fail if the Codex skill path disappears or if its required routing guidance is removed

### Step 5. Verify Additive Safety And Record Handoff

Confirm the task stayed inside the Codex-owned surface and write the implementation handoff back into the roadmap.

Check for:

- no `.claude/**` rename, relocation, or deletion
- no new dependency on Claude hook infrastructure
- explicit notes for `CX-06` about provisioning
- explicit notes for `CX-07` about extending the workflow prompt pack from the new entry contract

Expected output:

- an additive diff centered on the Codex skill template, targeted tests, and roadmap execution notes

## Deliverable Shape

The implementation for `CX-05` should ideally leave:

- one real Codex skill entrypoint at `internal/template/templates/.codex/skills/moai/SKILL.md`
- targeted template tests that verify the skill contract content and deployment path
- roadmap notes recording what `$moai`, `$moai plan`, `$moai run`, and `$moai sync` now mean in Codex
- no Claude-adapter path changes

## Verification Focus

Primary verification should stay narrow and template-centric:

- `go test ./internal/template/...`

If the implementation touches roadmap status or other non-template files, verify those changes by direct file review. Broader CLI or provisioning tests should wait for `CX-06` unless `CX-05` unexpectedly requires more.

## Out Of Scope

- wiring `moai init` or `moai update` to provision `.codex/**`
- adding `internal/cli/codex.go`
- adding `internal/codex/**` runtime helpers
- recreating the full Codex workflow prompt pack for every MoAI workflow
- claiming Claude hook, slash-command, or statusline parity in Codex
