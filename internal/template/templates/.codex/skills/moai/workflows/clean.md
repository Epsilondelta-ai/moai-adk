# Workflow: `clean`

## When To Use

Use this workflow when the user wants dead code, stale artifacts, unused files, or other safe cleanup performed without expanding scope into unrelated refactors.

## Inspect First

1. `git status` and the requested cleanup scope
2. relevant `.moai/specs/**` or `.moai/plans/**` if cleanup is part of approved work
3. `.moai/project/structure.md` and `.moai/project/tech.md` for architectural context
4. candidate code, imports, generated files, and tests that prove whether the target is still used

## Expected Behavior

- Confirm that the candidate code or artifact is truly stale, unused, or superseded before removing it.
- Prefer the smallest safe cleanup that can be verified in the current turn.
- Update adjacent imports, references, docs, or `.moai/**` notes when cleanup changes the recorded project state.
- Skip uncertain removals unless the user explicitly accepts the risk.

## Primary Outputs

- focused cleanup changes in the repository
- verification results showing the cleanup did not break the targeted scope
- a short note listing anything intentionally left in place because usage was uncertain

## Codex Constraints

- Do not assume Claude batch agents, rollback helpers, or MX-tag automation exist.
- Never remove code only because it looks old; require evidence from references, tests, generated outputs, or spec history.
- Prefer a dry review of candidates over speculative deletion when confidence is low.

## Handoff

- Hand off to `sync.md` if cleanup changes docs, codemaps, or workflow state.
- Hand off to `loop.md` when cleanup is part of a larger repeated fix-and-verify cycle.
