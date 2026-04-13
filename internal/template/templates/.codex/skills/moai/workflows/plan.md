# Workflow: `plan`

## When To Use

Use this workflow when the user wants a new plan, a new spec, or a revision to existing planned work inside `.moai/plans/**` or `.moai/specs/**`.

## Inspect First

1. `.moai/docs/CODEX_COMPAT_ROADMAP.md`
2. `.moai/project/product.md`
3. `.moai/project/structure.md`
4. `.moai/project/tech.md`
5. relevant `.moai/plans/**`
6. relevant `.moai/specs/**`
7. current repository files related to the requested feature

## Expected Behavior

- Translate the user request into a concrete planning target grounded in current `.moai/**` state.
- Reuse or extend an existing `SPEC-...` directory when the work clearly belongs to it; otherwise create the smallest new planning artifact that matches the request.
- Make scope, dependencies, acceptance shape, and known risks explicit enough for `run.md`.
- Record assumptions and unresolved questions directly in the planning artifact instead of keeping them implicit in conversation.

## Primary Outputs

- updated `.moai/plans/**` note or `.moai/specs/SPEC-*/{spec.md,plan.md,acceptance.md,research.md}` as needed
- a brief execution handoff describing what is approved, missing, or blocked

## Codex Constraints

- Do not assume Claude interview loops, worktree creation, GitHub issue creation, or task APIs exist.
- Ask concise follow-up questions only when the missing detail materially changes scope or acceptance.
- Keep the artifact operational. Avoid porting Claude-only phase names or flags unless the repository already uses them in `.moai/**`.

## Handoff

- Hand off to `run.md` when implementation can begin from an approved plan or spec.
- Hand off to `project.md` first if `.moai/project/**` is too incomplete to plan responsibly.
- Hand off to `sync.md` if planning exposed roadmap or documentation drift that should be recorded immediately.
