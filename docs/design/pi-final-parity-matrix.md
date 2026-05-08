# Pi MoAI Final Parity Matrix

Status: Active verification matrix

| Area | Pi status | Verification |
|---|---|---|
| Output framing | Implemented via output-style injection | Manual Pi TUI QA |
| Structured user interaction | Implemented via `moai_ask_user` and `ctx.ui` | Manual Pi TUI QA |
| Command workflow | Canonical SPEC + workflow/delegation artifacts | `go test ./internal/kernel/...` |
| Quality gate | Language-aware executable checks | `moai_quality_gate`, tests |
| LSP diagnostics | Fallback diagnostics via internal LSP hook tools | `moai_lsp_check`, tests |
| MX scan/update | Validation + line-aware update foundation | `moai_mx_scan`, `moai_mx_update`, tests |
| Subagent runtime | Normalized Pi subprocess contract + blocker parser | `go test ./internal/agentruntime/...` |
| Agent Teams | Durable state + synthetic lifecycle events | `go test ./internal/teamruntime/...` |
| Hook/event parity | Native + synthetic + partial support advertised | `moai_bridge_capabilities` |
| Tool surface | Registered tools match bridge capabilities | bridge tests |
| Resource drift | `moai pi sync-resources --check` | drift check |
| E2E smoke | Scripted bridge smoke | `./scripts/pi-smoke.sh` |

## Manual parity gates still required on real TUI

1. Normal chat starts with MoAI branded frame in a MoAI-enabled project.
2. `/moai-doctor` renders framed completion.
3. `moai_ask_user` shows select/confirm/input/editor UI and returns tool output.
4. Footer widget reflects phase, task, quality, LSP, and MX state.

These gates require an interactive Pi process and cannot be fully proven by non-interactive Go tests.
