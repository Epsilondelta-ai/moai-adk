import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { PI_HOOKS_SOURCE_PATH } from "./constants.ts";

export interface HookBridgeResult {
  ok: boolean;
  exitCode: number | null;
  stdout: string;
  stderr: string;
  skipped?: string;
}

export const HOOK_SCRIPT_BY_EVENT: Record<string, string> = {
  "session-start": `${PI_HOOKS_SOURCE_PATH}/handle-session-start.sh`,
  compact: `${PI_HOOKS_SOURCE_PATH}/handle-compact.sh`,
  "session-end": `${PI_HOOKS_SOURCE_PATH}/handle-session-end.sh`,
  "pre-tool": `${PI_HOOKS_SOURCE_PATH}/handle-pre-tool.sh`,
  "post-tool": `${PI_HOOKS_SOURCE_PATH}/handle-post-tool.sh`,
  stop: `${PI_HOOKS_SOURCE_PATH}/handle-stop.sh`,
  "agent-hook": `${PI_HOOKS_SOURCE_PATH}/handle-agent-hook.sh`,
  "subagent-start": `${PI_HOOKS_SOURCE_PATH}/handle-subagent-start.sh`,
  "subagent-stop": `${PI_HOOKS_SOURCE_PATH}/handle-subagent-stop.sh`,
  "post-tool-failure": `${PI_HOOKS_SOURCE_PATH}/handle-post-tool-failure.sh`,
  notification: `${PI_HOOKS_SOURCE_PATH}/handle-notification.sh`,
  "user-prompt-submit": `${PI_HOOKS_SOURCE_PATH}/handle-user-prompt-submit.sh`,
  "permission-request": `${PI_HOOKS_SOURCE_PATH}/handle-permission-request.sh`,
  "teammate-idle": `${PI_HOOKS_SOURCE_PATH}/handle-teammate-idle.sh`,
  "task-completed": `${PI_HOOKS_SOURCE_PATH}/handle-task-completed.sh`,
  "worktree-create": `${PI_HOOKS_SOURCE_PATH}/handle-worktree-create.sh`,
  "worktree-remove": `${PI_HOOKS_SOURCE_PATH}/handle-worktree-remove.sh`,
};

export const CONNECTED_PI_HOOK_EVENTS = [
  "session-start",
  "compact",
  "session-end",
  "user-prompt-submit",
  "pre-tool",
  "post-tool",
  "post-tool-failure",
  "stop",
] as const;

export const NON_BLOCKING_HOOK_BRIDGE_POLICY =
  "extension hook bridge is non-blocking compatibility telemetry; blocking guardrails are enforced by pi-yaml-hooks tool.before.* policies";

export function unsupportedHookEvents(): string[] {
  const connected = new Set<string>(CONNECTED_PI_HOOK_EVENTS);
  return Object.keys(HOOK_SCRIPT_BY_EVENT).filter((event) => !connected.has(event));
}

export function hasHookScript(eventName: string): boolean {
  const script = HOOK_SCRIPT_BY_EVENT[eventName];
  return Boolean(script && existsSync(resolve(process.cwd(), script)));
}

export async function runMoaiHook(eventName: string, payload: unknown, timeoutMs = 10000): Promise<HookBridgeResult> {
  const script = HOOK_SCRIPT_BY_EVENT[eventName];
  if (!script) return { ok: true, exitCode: 0, stdout: "", stderr: "", skipped: `no hook script mapped for ${eventName}` };
  const abs = resolve(process.cwd(), script);
  if (!existsSync(abs)) return { ok: true, exitCode: 0, stdout: "", stderr: "", skipped: `hook script missing: ${script}` };

  return new Promise((resolveResult) => {
    const child = spawn("bash", [abs], {
      cwd: process.cwd(),
      env: { ...process.env, CLAUDE_PROJECT_DIR: process.cwd() },
      stdio: ["pipe", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    const timer = setTimeout(() => child.kill("SIGTERM"), timeoutMs);
    child.stdout.on("data", (chunk) => (stdout += String(chunk)));
    child.stderr.on("data", (chunk) => (stderr += String(chunk)));
    child.on("close", (exitCode) => {
      clearTimeout(timer);
      resolveResult({ ok: exitCode === 0, exitCode, stdout, stderr });
    });
    child.stdin.end(`${JSON.stringify(payload)}\n`);
  });
}

export function hookBridgeStatus(): string {
  const mapped = Object.keys(HOOK_SCRIPT_BY_EVENT);
  const present = mapped.filter(hasHookScript).length;
  if (present !== mapped.length) {
    return `missing: hook bridge script files present ${present}/${mapped.length}`;
  }
  return `info: hook bridge script files present ${present}/${mapped.length} (existence only; runtime wiring is partial)`;
}

export function hookBridgeParityStatus(): string[] {
  const mapped = Object.keys(HOOK_SCRIPT_BY_EVENT);
  const present = mapped.filter(hasHookScript).length;
  const unsupported = unsupportedHookEvents();
  return [
    hookBridgeStatus(),
    `partial: extension-connected hook events ${CONNECTED_PI_HOOK_EVENTS.length}/${mapped.length}`,
    `info: ${NON_BLOCKING_HOOK_BRIDGE_POLICY}`,
    unsupported.length
      ? `partial: unsupported/unwired hook events ${unsupported.join(", ")}`
      : "ok: all mapped hook events are wired to pi extension events",
    present === mapped.length
      ? "ok: all mapped hook scripts exist"
      : `missing: hook scripts present ${present}/${mapped.length}`,
  ];
}
