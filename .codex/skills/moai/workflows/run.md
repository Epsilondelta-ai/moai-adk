# Workflow: `run`

## When To Use

Use this workflow when the user asks to implement approved work from `.moai/specs/**`, references a `SPEC-...` item, or wants code changes aligned to an existing MoAI plan.

## Inspect First

1. target `.moai/specs/SPEC-*/spec.md`
2. companion spec files such as `plan.md`, `acceptance.md`, `research.md`, and `progress.md`
3. `.moai/project/product.md`
4. `.moai/project/structure.md`
5. `.moai/project/tech.md`
6. relevant `.moai/state/**`
7. code, tests, and configs touched by the target spec

## Expected Behavior

- Confirm the implementation target and the acceptance shape before editing code.
- Make the requested code changes in the repository, keeping behavior aligned with the active spec and existing project patterns.
- Run the most relevant verification commands available in the repo and report what was or was not verified.
- Update `.moai/specs/**` or `.moai/state/**` when implementation changes execution status, checkpoints, or recorded progress.

## Primary Outputs

- repository code changes aligned to the active spec
- updated execution notes such as `progress.md` when the shared MoAI state changed
- a concise verification summary tied to the implemented scope

## Codex Constraints

- Do not promise Claude DDD/TDD orchestration, hook checkpoints, or agent-team execution.
- Use the repo's real commands and tests instead of implying hidden runtime helpers.
- If the spec is incomplete or unapproved, stop improvising and hand back to `plan.md`.

## Handoff

- Hand off to `sync.md` after implementation when docs or shared state need reconciliation.
- Hand off to `loop.md` when the work requires repeated fix-and-verify passes rather than a single implementation pass.
- Hand off to `review.md` when the user explicitly asks for a risk review before or after landing the change.
