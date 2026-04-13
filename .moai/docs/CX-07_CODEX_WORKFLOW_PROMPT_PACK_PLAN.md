# CX-07 Codex Workflow Prompt Pack Plan

## Purpose

This document is the execution plan for `CX-07 Codex Workflow Prompt Pack` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to extend the current Codex `$moai` entry contract into a real workflow prompt pack that gives Codex explicit, reusable prompt guidance for the key MoAI workflows while continuing to route all shared state through `.moai/**` and avoiding Claude-only runtime assumptions.

## Working Branch

- `feature/cx-07-codex-workflow-prompt-pack`

## Dependency Note

- `CX-05` already established `internal/template/templates/.codex/skills/moai/SKILL.md` as the canonical Codex-side `$moai` entry contract.
- `CX-06` already proved that Codex assets under `internal/template/templates/.codex/**` can be provisioned through normal init/update lifecycle flows.
- `CX-07` should build directly on those two decisions. It should not reopen provisioning behavior, path ownership, or drift-policy questions reserved for `CX-08`.

## Task Outcome

This task turns the single Codex entrypoint into a fuller workflow pack that Codex can follow without Claude slash commands, hooks, or launcher semantics.

The task is complete only when:

- Codex-owned workflow prompt docs exist for `project`, `plan`, `run`, `sync`, `review`, `clean`, and `loop`
- each workflow doc defines a Codex-side prompt contract with primary inputs, expected behavior, and Codex-specific constraints
- the top-level Codex `$moai` skill points to the workflow pack clearly enough for re-entry and continuation
- the prompt pack consistently references `.moai/**` as shared state instead of Claude runtime features
- template tests verify the new Codex workflow pack strongly enough to catch missing files or contract regressions

## Inputs

- roadmap decisions and handoff notes in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- current Codex entry contract:
  - `internal/template/templates/.codex/skills/moai/SKILL.md`
- Claude-side command wrappers that define the user-facing workflow surface:
  - `.claude/commands/moai/project.md`
  - `.claude/commands/moai/plan.md`
  - `.claude/commands/moai/run.md`
  - `.claude/commands/moai/sync.md`
  - `.claude/commands/moai/review.md`
  - `.claude/commands/moai/clean.md`
  - `.claude/commands/moai/loop.md`
- Claude-side workflow reference material to translate carefully rather than copy blindly:
  - `internal/template/templates/.claude/skills/moai/SKILL.md`
  - `internal/template/templates/.claude/skills/moai/workflows/project.md`
  - `internal/template/templates/.claude/skills/moai/workflows/plan.md`
  - `internal/template/templates/.claude/skills/moai/workflows/run.md`
  - `internal/template/templates/.claude/skills/moai/workflows/sync.md`
  - `internal/template/templates/.claude/skills/moai/workflows/review.md`
  - `internal/template/templates/.claude/skills/moai/workflows/clean.md`
  - `internal/template/templates/.claude/skills/moai/workflows/loop.md`
- existing Codex template verification surfaces:
  - `internal/template/embed_test.go`
  - `internal/template/deployer_test.go`
  - `internal/template/deployer_mode_test.go`
- shared MoAI state roots each Codex workflow should use:
  - `.moai/project/**`
  - `.moai/plans/**`
  - `.moai/specs/**`
  - `.moai/state/**`
  - `.moai/docs/CODEX_COMPAT_ROADMAP.md`

## Current Asset Observations

- The Codex adapter currently has one real skill entrypoint at `internal/template/templates/.codex/skills/moai/SKILL.md`, but it does not yet have per-workflow prompt docs comparable to the Claude-side `workflows/*.md` pack.
- The Claude surface splits user invocation into thin command wrappers and deeper workflow documents. Codex does not need command wrappers, but it still benefits from having deeper workflow contracts available as additive Markdown assets.
- The Claude workflow docs contain many instructions that are explicitly Claude-specific, including slash-command routing, hook behavior, AskUserQuestion flows, Task APIs, and agent catalogs. `CX-07` must translate intent and sequence, not copy those runtime assumptions.
- The roadmap completion criteria are about prompt-level workflow parity. They do not require recreating Claude's runtime machinery in Codex.

## Working Assumptions

- The Codex workflow pack should remain under the existing skill tree so the top-level `$moai` contract and the deeper workflow docs stay co-located.
- The first useful structure is likely additive Markdown such as `internal/template/templates/.codex/skills/moai/workflows/{project,plan,run,sync,review,clean,loop}.md`, plus selective updates to the top-level `SKILL.md`.
- Codex workflow docs should describe what to read, how to interpret `.moai/**` state, what the expected outcome is, and what not to assume from Claude. They do not need Claude YAML frontmatter or Claude tool directives unless Codex will actually consume them.
- The prompt pack should stay narrower than the Claude originals. The goal is a reliable Codex contract, not a full port of every Claude sub-phase, flag, or agent policy.
- If a workflow is not yet backed by Codex-specific helper commands, the prompt doc should state the limitation directly and route through current repo/context state rather than pretending parity exists.

## Prompt Pack Target Shape

`CX-07` should ideally leave this Codex-owned structure in place:

- `internal/template/templates/.codex/skills/moai/SKILL.md`
- `internal/template/templates/.codex/skills/moai/workflows/project.md`
- `internal/template/templates/.codex/skills/moai/workflows/plan.md`
- `internal/template/templates/.codex/skills/moai/workflows/run.md`
- `internal/template/templates/.codex/skills/moai/workflows/sync.md`
- `internal/template/templates/.codex/skills/moai/workflows/review.md`
- `internal/template/templates/.codex/skills/moai/workflows/clean.md`
- `internal/template/templates/.codex/skills/moai/workflows/loop.md`

Each workflow contract should answer, at minimum:

- when the workflow should be used in Codex
- which `.moai/**` files or directories must be inspected first
- what the workflow should produce or update
- what Codex-native limitations apply compared with Claude
- when to hand off to other MoAI workflow documents instead of improvising

## Execution Steps

### Step 1. Audit The Real Workflow Delta Between Claude And Codex

Read the existing Claude wrappers and workflow docs side by side with the current Codex `$moai` skill.

Check for:

- which workflow surfaces are already covered by the current Codex entry contract
- which workflow behaviors are essential to preserve at the prompt level
- which Claude instructions are unusable in Codex and must be translated or dropped
- which shared `.moai/**` inputs each workflow genuinely depends on

Expected output:

- a concrete per-workflow gap list for `project`, `plan`, `run`, `sync`, `review`, `clean`, and `loop`

### Step 2. Freeze The Codex Workflow Pack Information Architecture

Decide the final file layout and document responsibilities before writing content.

Focus on:

- whether all seven workflow docs live under `internal/template/templates/.codex/skills/moai/workflows/`
- what remains in the top-level `SKILL.md` versus what moves into per-workflow docs
- how `$moai` should reference those deeper docs for re-entry and continuation
- keeping the structure additive under Codex-owned paths only

Expected output:

- one stable Codex workflow-pack layout with clear ownership of entrypoint versus workflow docs

### Step 3. Write Codex-Native Contracts For Each Core Workflow

Create the workflow docs with Codex-native instructions instead of Claude runtime assumptions.

Per-workflow focus:

- `project`: project understanding, documentation generation, and missing-context handling
- `plan`: turning requests into plan/spec artifacts grounded in current `.moai/**` state
- `run`: implementing approved work from `.moai/specs/**` and updating execution state
- `sync`: reconciling docs, roadmap notes, and workflow state with the codebase
- `review`: risk-focused code review workflow using current changes and `.moai/**` context where relevant
- `clean`: safe dead-code and stale-artifact cleanup with verification expectations
- `loop`: iterative fix/verify cycles without promising Claude-only loop helpers

Rules:

- reference shared `.moai/**` state explicitly
- be precise about outputs and follow-up artifacts
- remove or rewrite Claude-only tool assumptions
- keep the contracts operationally useful without porting every Claude sub-phase

Expected output:

- seven Codex workflow documents that are direct, resumable, and additive

### Step 4. Update The Top-Level `$moai` Skill To Point At The Prompt Pack

Refine the existing entry contract so it acts as the dispatcher into the deeper workflow pack.

Focus on:

- keeping bare `$moai` as the top-level router and re-entry surface
- extending the supported invocation guidance to include `project`, `review`, `clean`, and `loop`
- pointing each route at the new workflow docs instead of keeping all detail inside one file
- preserving the existing `.moai/**` first-read behavior and Codex constraint language

Expected output:

- a leaner but more complete `internal/template/templates/.codex/skills/moai/SKILL.md` that delegates depth to the workflow pack

### Step 5. Lock The Workflow Pack With Template Regression Tests

Add or extend tests so the new prompt-pack assets are embedded, deployed, and meaningfully validated.

Priority coverage:

- all seven workflow docs are embedded under `.codex/**`
- deploy/extract behavior includes the new workflow docs
- key contract markers exist so tests fail if a workflow doc becomes empty, skeletal, or loses `.moai/**` guidance
- the top-level `SKILL.md` continues to route into the workflow pack correctly

Likely files:

- `internal/template/embed_test.go`
- `internal/template/deployer_test.go`
- `internal/template/deployer_mode_test.go`

Expected output:

- template tests that treat the Codex workflow pack as a real supported asset set

### Step 6. Record Additive Safety And Handoff

Before closing the task, verify the implementation stayed inside the intended ownership boundary.

Check for:

- no `.claude/**` rename, relocation, or behavior changes
- no new dependence on Claude hooks, slash commands, or statusline assets
- clear notes for `CX-08` if the prompt pack exposes update/drift-policy questions
- clear notes for `CX-09` if any workflow needs a helper that prompt-only guidance cannot cover well
- clear notes for `CX-10` about how to verify Codex workflow discoverability and invocation coverage

Expected output:

- an additive diff centered on Codex prompt assets, template tests, and roadmap execution notes

## Deliverable Shape

The implementation for `CX-07` should ideally leave:

- a Codex workflow prompt pack for the seven roadmap-required workflows
- an updated top-level `$moai` skill that dispatches into those workflow docs
- targeted template regression coverage for the expanded `.codex/**` asset tree
- roadmap notes recording which Claude semantics were translated, which were intentionally omitted, and what follow-up remains
- no Claude-adapter ownership changes

## Verification Focus

Primary verification should stay asset- and template-centric:

- `go test ./internal/template/...`

Secondary verification by direct file review should confirm:

- each required workflow doc exists under the Codex template tree
- each doc references shared `.moai/**` state and avoids false Claude parity claims
- the top-level Codex skill routes to the new workflow docs consistently

## Out Of Scope

- redesigning Codex provisioning or manifest overwrite policy
- adding Codex CLI/runtime helpers under `internal/cli/codex.go` or `internal/codex/**`
- recreating Claude slash-command wrappers inside `.codex/**`
- porting the entire Claude agent catalog, hook protocol, or AskUserQuestion flows to Codex
- changing protected Claude-owned paths to make the Codex workflow pack work
