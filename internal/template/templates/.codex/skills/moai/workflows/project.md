# Workflow: `project`

## When To Use

Use this workflow when the user asks to understand the codebase, initialize or refresh `.moai/project/**`, or recover missing project context needed by other MoAI workflows.

## Inspect First

1. `.moai/docs/CODEX_COMPAT_ROADMAP.md`
2. `.moai/project/product.md`
3. `.moai/project/structure.md`
4. `.moai/project/tech.md`
5. `.moai/project/codemaps/**`
6. relevant top-level source, config, and test files in the repository

If project docs are missing, treat the repository itself as the fallback source and record which `.moai/project/**` files had to be regenerated.

## Expected Behavior

- Determine whether the request is project discovery, project-doc refresh, or missing-context recovery.
- When missing-context recovery may really be an unprovisioned Codex setup, run `moai codex doctor` first so `.moai/**` and `.codex/**` readiness is explicit.
- Read enough of the codebase to describe product purpose, structure, and technology choices accurately.
- Create or update `.moai/project/product.md`, `.moai/project/structure.md`, and `.moai/project/tech.md` when they are missing or stale.
- Update `.moai/project/codemaps/**` only when the repository already uses codemaps or the user explicitly asks for architecture mapping.
- State unresolved gaps briefly when the codebase alone cannot answer them.

## Primary Outputs

- refreshed project baseline under `.moai/project/**`
- a short note about what was regenerated, updated, or still unknown

## Codex Constraints

- Do not assume Claude interview flows, slash commands, or AskUserQuestion menus exist.
- Ask at most one concise clarifying question when missing product intent cannot be inferred safely from code or existing docs.
- Do not invent architecture details that are not supported by code, config, tests, or `.moai/**`.

## Handoff

- Hand off to `plan.md` once the project baseline is good enough to scope new work.
- Hand off to `sync.md` if the main task is documentation drift reconciliation after code changes.
