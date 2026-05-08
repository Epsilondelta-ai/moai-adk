# Pi MoAI QA Checklist

Status: Active

## Automated smoke

```bash
./scripts/pi-smoke.sh
./scripts/pi-record-smoke.sh /tmp/pi-smoke-transcript.txt
```

The script validates:

- bridge doctor
- `/moai plan` → `/moai run` → `/moai sync`
- task create/list/get/update
- quality gate structured skip/pass behavior
- MX scan

## Manual interactive QA

1. Start Pi in this repository.
2. Run `/reload`.
3. Run `/moai-doctor`.
4. Confirm the response starts with the MoAI frame.
5. Run `/moai plan "small test feature"`.
6. Run `/moai run <SPEC-ID>`.
7. Run `/moai sync <SPEC-ID>`.
8. Trigger a workflow that calls `moai_ask_user` and confirm Pi shows structured UI.
9. Confirm footer/state updates for phase, quality, tasks, and MX data.

## Known limitations

- `moai_lsp_check` uses executable fallback diagnostics through MoAI's LSP hook package; it is not editor-native Pi LSP telemetry.
- Worktree merge supports a guarded `execute=true` path but defaults to a safe merge handoff.
- Team idle/completion hooks are synthesized through durable bridge state where Pi does not emit native teammate hooks.
