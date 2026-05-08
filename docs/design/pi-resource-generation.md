# Pi Resource Generation

Status: Active

Pi runtime resources are generated from Claude/MoAI source resources where possible.

## Generated resources

- `.pi/prompts/moai-*.md` from `.claude/commands/moai/*.md`
- `.pi/prompts/moai-output-style.md` from `.claude/output-styles/moai/moai.md`
- `.pi/skills/*` from `.claude/skills/*`

## Hand-authored resources

- `.pi/extensions/moai/**`
- `.pi/settings.json`

## Workflow

Regenerate resources:

```bash
moai pi sync-resources
```

Check drift without writing:

```bash
moai pi sync-resources --check
```

The drift check fails when generated `.pi` resources do not match converted `.claude` sources.
