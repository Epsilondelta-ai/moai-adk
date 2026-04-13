# Workflow: `loop`

## When To Use

Use this workflow when one pass is unlikely to finish the task cleanly and the user wants repeated diagnose-fix-verify cycles until the target issue set is stable or a stopping condition is reached.

## Inspect First

1. the current failure signal: failing command, error output, broken behavior, or review findings
2. relevant `.moai/specs/**` progress and acceptance context
3. relevant `.moai/state/**`
4. recent code and test changes related to the failure

## Expected Behavior

- Define the loop target clearly before iterating: which failures count, which verification command matters, and when to stop.
- Repeat a bounded cycle of diagnose, edit, verify, and reassess using the real repository commands.
- Record shared progress in `.moai/specs/**` or `.moai/state/**` when the loop materially changes execution status.
- Stop when the issue set is resolved, the remaining blocker needs user input, or further looping would become speculative.

## Primary Outputs

- iterative fixes tied to a concrete failure signal
- repeated verification results showing whether the issue count is shrinking
- a final stop reason: resolved, blocked, or deferred

## Codex Constraints

- Do not promise Claude snapshot stores, memory-pressure helpers, completion markers, or autonomous background loops.
- Keep the loop bounded to the current task and current turn rather than implying indefinite execution.
- If the workflow becomes a dead-code cleanup or a documentation reconciliation task, switch explicitly to `clean.md` or `sync.md` instead of stretching the loop contract.

## Handoff

- Hand off to `clean.md` after the fixes stabilize if they exposed dead code or stale files.
- Hand off to `sync.md` when the final state requires documentation or roadmap reconciliation.
