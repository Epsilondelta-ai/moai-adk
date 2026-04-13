# Workflow: `sync`

## When To Use

Use this workflow when code, docs, roadmap notes, or recorded workflow state have drifted and the user wants `.moai/**` plus project documentation brought back in line with the current repository.

## Inspect First

1. `git status` and relevant diffs
2. `.moai/docs/CODEX_COMPAT_ROADMAP.md`
3. `.moai/project/**`
4. `.moai/specs/**`
5. `.moai/plans/**`
6. `.moai/state/**`
7. `README.md` and other user-facing docs affected by the change

## Expected Behavior

- Identify which documents or state files are now stale relative to the codebase.
- Update only the artifacts that changed meaningfully, keeping `.moai/**` as the shared source of truth.
- Record roadmap, progress, or project-doc updates when Codex work changed the documented truth.
- Surface remaining drift explicitly if some files cannot be reconciled safely in the current turn.

## Primary Outputs

- synchronized `.moai/**` documentation and state
- updated README or adjacent docs when they are part of the changed truth
- a short drift summary listing what was synced and what remains open

## Codex Constraints

- Do not assume Claude PR wrappers, auto-merge flows, hook-based quality gates, or statusline state.
- Run relevant repo-local verification when sync work changes executable code or generated docs.
- If the user did not ask for GitHub or PR actions, do not imply they happened.

## Handoff

- Hand off to `review.md` when the synchronized result still needs a risk review.
- Hand off to `clean.md` when sync work reveals stale generated artifacts or dead files.
- Hand off to `project.md` when project-level documentation is missing rather than merely stale.
