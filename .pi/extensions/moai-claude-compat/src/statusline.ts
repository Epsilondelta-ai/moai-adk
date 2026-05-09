import { existsSync } from "node:fs";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import type { ExtensionContext } from "@earendil-works/pi-coding-agent";
import { STATUS_ID, WIDGET_ID } from "./constants.ts";
import type { MoaiCompatConfig } from "./config.ts";

export interface MoaiStatusState {
  phase: "idle" | "plan" | "run" | "sync" | "review" | "gate";
  specId?: string;
  teamMode?: string;
  worktree?: string;
  quotaProvider?: string;
}

const state: MoaiStatusState = { phase: "idle" };

export function updateMoaiStatus(ctx: ExtensionContext, config: MoaiCompatConfig, patch: Partial<MoaiStatusState> = {}) {
  Object.assign(state, patch);
  const status = buildClaudeLikeStatus(ctx) ?? buildFallbackStatus(ctx, config);
  ctx.ui.setStatus(STATUS_ID, status);
}

function buildClaudeLikeStatus(ctx: ExtensionContext): string | undefined {
  const cwd = ctx.cwd || process.cwd();
  const payload = JSON.stringify({
    cwd,
    workspace: {
      current_dir: cwd,
      project_dir: cwd,
    },
    model: normalizeModel(ctx.model),
    output_style: { name: "MoAI" },
  });

  const script = join(cwd, ".moai", "status_line.sh");
  const candidates = existsSync(script)
    ? [{ command: script, args: [] as string[] }, { command: "moai", args: ["statusline"] }]
    : [{ command: "moai", args: ["statusline"] }];

  for (const candidate of candidates) {
    const result = spawnSync(candidate.command, candidate.args, {
      cwd,
      input: payload,
      encoding: "utf8",
      timeout: 1_500,
      env: {
        ...process.env,
        CLAUDE_PROJECT_DIR: cwd,
        DEBUG_STATUSLINE: process.env.DEBUG_STATUSLINE ?? "0",
        HOME: process.env.MOAI_PI_STATUSLINE_HOME ?? "/tmp/moai-pi-statusline-no-home",
        MOAI_NO_COLOR: "1",
        NO_COLOR: "1",
      },
    });
    if (result.error || result.status !== 0) continue;

    const text = normalizeClaudeStatus(result.stdout);
    if (text) return text;
  }

  return undefined;
}

function normalizeModel(model: unknown): { id: string; name: string; display_name: string } {
  const record = typeof model === "object" && model !== null ? model as Record<string, unknown> : {};
  const id = String(record.id ?? record.name ?? "pi");
  const displayName = String(record.displayName ?? record.display_name ?? record.name ?? record.id ?? "Pi");
  return { id, name: id, display_name: displayName };
}

function normalizeClaudeStatus(output: string): string | undefined {
  const lines = output
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .filter((line) => !isQuotaLine(line));

  if (lines.length === 0) return undefined;
  return lines.join(" │ ").replace(/[\r\n\t]/g, " ").replace(/ +/g, " ").trim();
}

function isQuotaLine(line: string): boolean {
  return /(^|\b)5H\s*:/i.test(line) || /(^|\b)7D\s*:/i.test(line);
}

function buildFallbackStatus(ctx: ExtensionContext, config: MoaiCompatConfig): string {
  const theme = ctx.ui.theme;
  const phase = theme.fg("accent", `MoAI:${state.phase}`);
  const spec = state.specId ? theme.fg("dim", ` ${state.specId}`) : "";
  const quality = theme.fg("dim", ` quality:${config.qualityMode}`);
  const team = state.teamMode ? theme.fg("dim", ` team:${state.teamMode}`) : "";
  const wt = state.worktree ? theme.fg("dim", ` wt:${state.worktree}`) : "";
  return `${phase}${spec}${quality}${team}${wt}`;
}

export function setMoaiWidget(ctx: ExtensionContext, lines?: string[]) {
  ctx.ui.setWidget(WIDGET_ID, lines && lines.length > 0 ? lines : undefined);
}

export function inferPhaseFromCommand(commandName: string): MoaiStatusState["phase"] {
  if (commandName.includes("plan")) return "plan";
  if (commandName.includes("run")) return "run";
  if (commandName.includes("sync")) return "sync";
  if (commandName.includes("review")) return "review";
  if (commandName.includes("gate") || commandName.includes("coverage") || commandName.includes("e2e")) return "gate";
  return "idle";
}
