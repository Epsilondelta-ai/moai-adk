# `$moai` Codex Entry Point

Use this skill when the user invokes `$moai`, asks to continue a MoAI workflow, or needs shared MoAI state interpreted from `.moai/**` inside Codex.

This file is the canonical Codex-side router for MoAI. It keeps the entry contract lean, then delegates workflow depth to the Codex-owned prompt pack under `workflows/`.

## Read First

Inspect shared project state in this order before choosing a workflow:

1. `.moai/docs/CODEX_COMPAT_ROADMAP.md`
2. `.moai/project/product.md`
3. `.moai/project/structure.md`
4. `.moai/project/tech.md`
5. `.moai/specs/**`
6. `.moai/plans/**`
7. `.moai/state/**`

If a referenced file or directory does not exist, continue with the remaining sources and state the missing context briefly.

## Workflow Pack

Use these Codex workflow contracts for the detailed behavior:

- `workflows/project.md`
- `workflows/plan.md`
- `workflows/run.md`
- `workflows/sync.md`
- `workflows/review.md`
- `workflows/clean.md`
- `workflows/loop.md`

## Supported Invocations

### `$moai`

Treat bare `$moai` as the top-level dispatcher and re-entry surface.

- Read the shared `.moai/**` context first.
- If the user names a `SPEC-...` item or asks to implement approved work, route to `$moai run` and open `workflows/run.md`.
- If the user asks to understand the project, regenerate project docs, or recover missing baseline context, route to `$moai project` and open `workflows/project.md`.
- If the user asks to define work, scope a feature, or create a plan/spec, route to `$moai plan` and open `workflows/plan.md`.
- If the user asks to reconcile docs, roadmap notes, codemaps, or project drift, route to `$moai sync` and open `workflows/sync.md`.
- If the user asks for a code review or risk scan, route to `$moai review` and open `workflows/review.md`.
- If the user asks to remove dead code or stale artifacts safely, route to `$moai clean` and open `workflows/clean.md`.
- If the user asks to keep fixing, retry, or iterate until stable, route to `$moai loop` and open `workflows/loop.md`.
- If intent is still ambiguous after reading `.moai/**`, list the most relevant routes and ask one concise clarifying question.

### `$moai project`

Use `workflows/project.md` for project understanding, project documentation generation, and missing-context recovery grounded in `.moai/project/**` plus the current repository state.

### `$moai plan`

Use `workflows/plan.md` for plan/spec creation or re-planning grounded in `.moai/project/**`, `.moai/plans/**`, and `.moai/specs/**`.

### `$moai run`

Use `workflows/run.md` for implementation against an approved spec, plan, or explicit `SPEC-...` target under `.moai/specs/**`.

### `$moai sync`

Use `workflows/sync.md` for documentation synchronization, roadmap alignment, and `.moai/**` state reconciliation after planning or implementation.

### `$moai review`

Use `workflows/review.md` for risk-focused review of working tree changes, staged changes, or a requested target, with findings reported before summaries.

### `$moai clean`

Use `workflows/clean.md` for safe dead-code and stale-artifact cleanup tied back to current `.moai/**` context and verification results.

### `$moai loop`

Use `workflows/loop.md` for iterative diagnose-fix-verify cycles when one pass is unlikely to finish the task cleanly.

## Shared-State Rules

- `.moai/**` is the source of truth for shared MoAI project state.
- Keep Codex-specific prompt assets additive under `.codex/**`; do not duplicate workflow state there.
- Prefer reusing existing `.moai/specs/**`, `.moai/plans/**`, `.moai/project/**`, and `.moai/state/**` artifacts over inventing parallel Codex-only notes.
- When execution changes shared state, update the relevant `.moai/**` files instead of storing hidden progress in the prompt pack.

## Codex Constraints

- Do not claim Claude slash-command parity.
- Do not assume Claude hook execution, launcher wrappers, statusline assets, AskUserQuestion flows, or Claude agent catalogs exist.
- Route work through this skill document, the workflow pack, and the shared `.moai/**` state instead of Claude-only runtime features.
- If Codex readiness is unclear, prefer `moai codex doctor` before assuming `.moai/**` or `.codex/**` assets are missing.
- If a workflow would benefit from a helper that Codex does not yet have, state the limitation directly and continue with the best repo-local/manual path instead of pretending parity exists.
