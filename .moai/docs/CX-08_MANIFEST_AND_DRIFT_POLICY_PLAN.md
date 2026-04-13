# CX-08 Manifest and Drift Policy Plan

## Purpose

This document is the execution plan for `CX-08 Manifest and Drift Policy` from [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md).

The goal is to make Codex-generated assets under `.codex/**` participate in a deterministic manifest/update policy so `moai update` can refresh safe files, preserve local edits, and record post-merge state without collapsing meaningful drift information.

## Working Branch

- `feature/cx-08-manifest-and-drift-policy`

## Dependency Note

- `CX-06` already proved that `.codex/**` assets are provisioned through the shared template deploy/update lifecycle and are written into `.moai/manifest.json`.
- `CX-07` expanded the Codex asset surface from one skill file to a multi-file workflow pack, which makes drift behavior a real ongoing concern rather than a one-file edge case.
- `CX-08` should resolve tracking and overwrite policy on top of those decisions. It should not reopen path ownership, provisioning layout, or workflow-pack content questions.

## Task Outcome

This task defines the supported lifecycle for Codex-managed generated files after initialization.

The task is complete only when:

- the policy clearly distinguishes overwrite-safe Codex assets from locally drifted Codex assets
- update behavior for untouched, user-modified, merged/restored, and removed-upstream template files is documented
- manifest data written after `moai update` preserves enough state to drive the next update safely
- shared update logic implements the policy without pretending merged Codex files are pristine template output
- regression tests prove the final manifest state for representative `.codex/**` scenarios
- the roadmap records the final policy and follow-up notes for verification work

## Inputs

- roadmap decisions and handoff notes in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- current shared manifest model:
  - `internal/manifest/types.go`
  - `internal/manifest/manifest.go`
- current shared deployment/update flow:
  - `internal/template/deployer.go`
  - `internal/template/deployer_mode.go`
  - `internal/cli/update.go`
  - `internal/core/project/initializer.go`
- current Codex asset set in scope:
  - `internal/template/templates/.codex/skills/moai/SKILL.md`
  - `internal/template/templates/.codex/skills/moai/workflows/project.md`
  - `internal/template/templates/.codex/skills/moai/workflows/plan.md`
  - `internal/template/templates/.codex/skills/moai/workflows/run.md`
  - `internal/template/templates/.codex/skills/moai/workflows/sync.md`
  - `internal/template/templates/.codex/skills/moai/workflows/review.md`
  - `internal/template/templates/.codex/skills/moai/workflows/clean.md`
  - `internal/template/templates/.codex/skills/moai/workflows/loop.md`
- current test surfaces most likely to lock behavior:
  - `internal/cli/update_test.go`
  - `internal/cli/update_merge_test.go`
  - `internal/cli/update_fileops_test.go`
  - `internal/core/project/initializer_test.go`
  - `internal/manifest/manifest_test.go`
  - `internal/template/deployer_test.go`
  - `internal/template/deployer_mode_test.go`

## Current Code Observations

- `moai init` deploys embedded templates through the shared deployer and tracks newly written Codex files as `template_managed`.
- `moai update` deploys templates in force-update mode, so existing tracked files are overwritten and immediately re-tracked as `template_managed` before restore and merge steps run.
- `restoreMoaiConfig(...)`, `.gitignore` preservation, and `mergeUserFiles(...)` can change on-disk content after deployment, but `finalizeTemplateSyncManifest(...)` currently rewrites `deployed_hash` and `current_hash` to the final bytes for every tracked file without changing provenance.
- That means a file whose content was restored or merged can end an update cycle looking indistinguishable from a pristine template deployment, even though the next update should treat it more carefully.
- The non-force deploy path already has a useful distinction for pre-existing untracked files: it records them as `user_created` and skips overwriting them.
- The roadmap handoff from `CX-06` and `CX-07` already points at this exact gap: Codex assets now have a real multi-file drift surface, and the current manifest model does not yet express that safely enough.

## Working Assumptions

- The policy should remain shared-first. If a rule is correct for `.codex/**`, it should usually live in shared manifest/update plumbing rather than Codex-only branches.
- `CX-08` should prefer the smallest state model that still makes next-update decisions deterministic. New provenance values are acceptable only if the existing model cannot describe the required behavior clearly.
- `template_hash` should continue to represent template lineage, not merely "whatever bytes are on disk after update", unless implementation proves that assumption impossible to preserve.
- Codex-generated skill and workflow docs are managed assets, but local projects may still customize them. The policy must preserve those edits explicitly instead of relying on accidental merge side effects.
- Documentation and tests should focus on concrete state transitions rather than informal language like "mostly managed" or "usually safe".

## Policy Questions To Resolve

`CX-08` needs explicit answers to these questions before implementation is complete:

- After a user-modified Codex file is merged or restored during update, should its provenance remain `template_managed`, move to `user_modified`, or move to a new drift-specific classification?
- Should `deployed_hash` continue to mean "last template bytes written" while `current_hash` means "actual current file bytes", instead of collapsing both to the same post-merge content?
- How should the next update behave when a tracked Codex file was deleted locally: restore it, mark it drifted, or preserve deletion intent?
- When a Codex template disappears upstream, should the manifest move that entry to `deprecated` and preserve the file in place, or should a stronger Codex-specific rule exist?
- Can the policy be expressed with the existing manifest fields plus better state transitions, or is additional metadata required?

## Execution Steps

### Step 1. Reproduce The Current Drift Ambiguities In Tests

Before changing behavior, pin down the exact states the current code produces for Codex assets.

Check for:

- untouched `.codex/**` files after update
- locally edited `.codex/**` files that are overwritten then merged back
- files preserved because the new template no longer contains the path
- final manifest entries after `finalizeTemplateSyncManifest(...)`

Expected output:

- failing or characterization tests that show where current manifest state is too lossy for the next update cycle

### Step 2. Decide The Minimal Durable State Model

Freeze the policy before broad code edits.

Focus on:

- the allowed provenance/state transitions from init through repeated updates
- whether merged/restored output should remain overwrite-safe or become drifted
- how `template_hash`, `deployed_hash`, and `current_hash` should evolve across deploy, merge, restore, and deprecation paths
- how much of the policy should be generic for all managed templates versus merely verified through Codex-owned files

Expected output:

- one explicit state model and overwrite matrix that can be written back into the roadmap

### Step 3. Implement Manifest And Update-State Handling At Narrow Shared Seams

Patch only the components that truly own file-state transitions.

Likely touch points:

- `internal/manifest/types.go`
- `internal/manifest/manifest.go`
- `internal/cli/update.go`
- `internal/template/deployer.go`
- `internal/template/deployer_mode.go`

Rules:

- preserve additive ownership boundaries
- avoid Codex-only branching unless shared behavior cannot safely express the policy
- ensure the final saved manifest reflects both template lineage and current drift state
- do not silently relabel merged user content as pristine template output

Expected output:

- deterministic manifest persistence and update routing for Codex-managed files

### Step 4. Lock The Policy With Codex-Focused Regression Coverage

Use `.codex/**` assets as the concrete verification surface for the shared policy.

Priority coverage:

- untouched Codex files remain overwrite-safe
- edited Codex files are not mistaken for pristine template output after update
- merged/restored Codex files preserve local content and record the correct follow-up state
- removed-upstream behavior is explicit if the task uses `deprecated` or another equivalent state

Likely files:

- `internal/cli/update_test.go`
- `internal/cli/update_merge_test.go`
- `internal/cli/update_fileops_test.go`
- `internal/core/project/initializer_test.go`
- `internal/manifest/manifest_test.go`

Expected output:

- tests that fail if Codex drift state regresses back to ambiguous `template_managed` behavior

### Step 5. Write The Policy Back Into The Roadmap And Handoff

Close the task by documenting the chosen policy where the roadmap expects it.

Check for:

- a written policy in [`.moai/docs/CODEX_COMPAT_ROADMAP.md`](/Users/juunini/Desktop/code/moai-adk/.moai/docs/CODEX_COMPAT_ROADMAP.md)
- execution-note updates that record what changed in manifest/update semantics
- explicit verification notes for `CX-10`
- explicit merge-maintenance notes for `CX-11` if the new policy affects upstream sync expectations

Expected output:

- a code-and-doc change set that leaves `CX-08` executable, reviewable, and resumable

## Deliverable Shape

The implementation for `CX-08` should ideally leave:

- a documented manifest/drift policy for Codex-managed generated assets
- shared manifest/update logic that persists the chosen post-merge state correctly
- targeted regression tests centered on real `.codex/**` files
- roadmap notes that explain the final overwrite and regeneration rules
- no Claude-adapter ownership changes

## Verification Focus

Primary verification should stay centered on the shared state machine:

- `go test ./internal/manifest/...`
- `go test ./internal/cli/...`
- `go test ./internal/core/project/...`

If deployer behavior changes materially, add:

- `go test ./internal/template/...`

## Out Of Scope

- redesigning Codex workflow content or provisioning layout
- adding new Codex CLI/runtime helpers under `internal/cli/codex.go` or `internal/codex/**`
- broad update UX redesign unrelated to manifest/drift decisions
- changing protected Claude-owned paths to make Codex policy work
- claiming verification completion for `CX-10`
