# MoAI Codex Skill Scaffold

This file reserves the future `$moai` Codex skill entrypoint at
`.codex/skills/moai/SKILL.md`.

`CX-05` must extend this same file path into the real skill contract.

Until then, preserve these constraints:

- keep the skill rooted at `.codex/skills/moai/SKILL.md`
- treat `.moai/**` as the shared project-state source of truth
- do not assume Claude-only hooks, commands, launchers, or statusline assets exist in Codex
