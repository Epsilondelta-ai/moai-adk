# `$moai` Codex Entry Point

Use this skill when the user invokes `$moai`, asks to continue a MoAI workflow, or needs MoAI project state interpreted from `.moai/**` inside Codex.

This file is the canonical Codex-side entry contract for MoAI. It defines how Codex should route the core MoAI workflow surfaces without assuming Claude-only slash commands, hooks, launchers, or statusline behavior.

## Read First

Inspect shared project state in this order before deciding how to proceed:

1. `.moai/docs/CODEX_COMPAT_ROADMAP.md`
2. `.moai/project/product.md`
3. `.moai/project/structure.md`
4. `.moai/project/tech.md`
5. `.moai/specs/**`
6. `.moai/plans/**`
7. `.moai/state/**`

If a referenced file or directory does not exist, continue with the remaining sources and state the missing context briefly.

## Supported Invocations

### `$moai`

Treat bare `$moai` as the top-level dispatcher and re-entry surface.

- Read the shared `.moai/**` context first.
- If the user names a `SPEC-...` item or asks to implement approved work, route to `$moai run`.
- If the user asks to define work, scope a feature, or create a plan/spec, route to `$moai plan`.
- If the user asks to reconcile docs, codemaps, roadmap notes, or project drift, route to `$moai sync`.
- If intent is still ambiguous after reading `.moai/**`, respond with the available routes and ask one concise clarifying question.

### `$moai plan`

Use for planning, specification, and re-planning work.

Primary inputs:

- `.moai/docs/CODEX_COMPAT_ROADMAP.md`
- `.moai/project/**`
- `.moai/plans/**`
- relevant `.moai/specs/**` directories when resuming or extending existing work

Expected behavior:

- turn the user request into a concrete plan or spec direction grounded in current `.moai/**` state
- create or update the appropriate planning artifact under `.moai/plans/**` or `.moai/specs/SPEC-*/`
- make dependencies, assumptions, and acceptance shape explicit enough for `$moai run`

### `$moai run`

Use for implementation against an approved spec, plan, or explicit `SPEC-...` target.

Primary inputs:

- target `.moai/specs/SPEC-*/spec.md`
- optional companion files such as `plan.md`, `acceptance.md`, `research.md`, and `progress.md`
- relevant `.moai/state/**` entries and project docs under `.moai/project/**`

Expected behavior:

- implement the requested work in the codebase using the approved MoAI context
- keep code changes aligned with the active spec and acceptance material
- update related progress or state artifacts under `.moai/**` when the implementation changes execution state

### `$moai sync`

Use for documentation synchronization, roadmap/state alignment, and drift cleanup after planning or implementation.

Primary inputs:

- `.moai/project/**`
- `.moai/specs/**`
- `.moai/plans/**`
- `.moai/state/**`
- `.moai/docs/CODEX_COMPAT_ROADMAP.md`

Expected behavior:

- reconcile project documentation and recorded workflow state with the current codebase
- update codemaps or roadmap notes when Codex-side work changes the documented truth
- surface remaining drift explicitly if full synchronization cannot be completed in the current turn

## Shared-State Rules

- `.moai/**` is the source of truth for shared MoAI project state.
- Keep Codex-specific assets additive under `.codex/**`; do not duplicate workflow state there.
- Prefer reusing existing `.moai/specs/**`, `.moai/plans/**`, and `.moai/project/**` artifacts over inventing parallel Codex-only notes.

## Codex Constraints

- Do not claim Claude slash-command parity.
- Do not assume Claude hook execution, launcher wrappers, or statusline assets exist.
- Route work through this skill document and the shared `.moai/**` state instead of Claude-only runtime features.
- Treat future Codex provisioning and richer workflow prompt packs as follow-up work outside this entry contract.
