# Workflow: `review`

## When To Use

Use this workflow when the user asks for code review, risk analysis, or a quality/security assessment of current changes, a specific file, or a spec-linked implementation.

## Inspect First

1. requested review scope such as `git diff`, staged changes, a branch diff, or named files
2. relevant `.moai/specs/**` or `.moai/plans/**` artifacts for intent and acceptance context
3. `.moai/project/**` when architecture or product context affects the review
4. tests or fixtures that define expected behavior for the changed code

## Expected Behavior

- Review with a bug-and-risk mindset first: correctness, regressions, security, performance, and missing tests.
- Report findings before summaries, ordered by severity, with precise file and line references when possible.
- State clearly when no findings were identified and mention residual verification gaps.
- Keep the review grounded in the actual diff and relevant `.moai/**` context instead of generic style advice.

## Primary Outputs

- a prioritized finding list
- explicit open questions or assumptions when the review scope is ambiguous
- an optional brief summary only after the findings

## Codex Constraints

- Do not assume Claude task APIs, review teams, AskUserQuestion menus, or auto-fix workflows exist.
- Do not silently edit code during review unless the user asked for fixes rather than review.
- If the requested review needs missing spec context, say so and review the observable code with that limitation noted.

## Handoff

- Hand off to `run.md` or `loop.md` when the user wants the findings fixed.
- Hand off to `sync.md` when the review shows the docs or recorded state no longer match the implementation.
