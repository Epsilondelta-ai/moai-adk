---
name: moai-pi-compat
description: Compatibility rules for running the Claude-oriented MoAI harness inside pi. Use whenever working on MoAI pi porting, command routing, package parity, Agent Teams parity, hooks, footer quota, or source-map/generated/overrides synchronization.
---

# MoAI pi compatibility

This project treats `.pi/generated/source/**` as the pi-local snapshot of the Claude harness. Pi-specific files live under `.pi/**`.

## Core rules

- User-facing responses use the configured conversation language.
- Use Markdown. Do not display XML tags to the user.
- Prefer pi packages over custom implementation.
- `moai-claude-compat` should only provide schema conversion, package glue, and MoAI-specific policy enforcement.
- Claude `permissionMode` is excluded from parity by design. Preserve it as metadata only.
- Preserve allow/ask/deny safety guardrails via package-backed policy.

## Package priorities

- Agent Teams: `@tmustier/pi-agent-teams` → `pi-teams` → `pi-crew` → `pi-subagents`.
- Hooks: `pi-yaml-hooks` → `pi-autohooks` → custom bridge.
- Guardrails: `@gotgenes/pi-permission-system`, `@aliou/pi-guardrails`, `pi-yaml-hooks`.
- Trigger preload: `pi-prompt-template-model` plus custom trigger indexer.
- Codex/GPT quota footer: `@kmiyh/pi-codex-plan-limits`; do not call internal ChatGPT endpoints directly from MoAI compat.

## Source maps

Read `.pi/claude-compat/source-map.json` before implementing compatibility changes. Use its `.pi/generated/source/**` paths during runtime.

Generated files go under `.pi/generated/**`; manual overrides go under `.pi/overrides/**`.
