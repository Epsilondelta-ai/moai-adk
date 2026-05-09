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
  "notification",
] as const;

type HookRuntimeState = "extension-connected" | "package-backed-adapter-available" | "package-backed-bridge-missing" | "adapter-needed" | "intentionally-excluded";

export interface HookRuntimeClassification {
  state: HookRuntimeState;
  detail: string;
}

export const HOOK_RUNTIME_CLASSIFICATION: Record<string, HookRuntimeClassification> = {
  "session-start": { state: "extension-connected", detail: "mapped from Pi session_start" },
  compact: { state: "extension-connected", detail: "mapped from Pi session_before_compact" },
  "session-end": { state: "extension-connected", detail: "mapped from Pi session_shutdown" },
  "pre-tool": { state: "extension-connected", detail: "mapped from Pi tool_call" },
  "post-tool": { state: "extension-connected", detail: "mapped from Pi tool_result" },
  stop: { state: "extension-connected", detail: "mapped from Pi agent_end for the main session" },
  "post-tool-failure": { state: "extension-connected", detail: "derived from Pi tool_result when isError is true" },
  "user-prompt-submit": { state: "extension-connected", detail: "mapped from Pi input" },
  "agent-hook": { state: "adapter-needed", detail: "Claude agent frontmatter hooks need a Pi agent metadata execution adapter" },
  "subagent-start": { state: "package-backed-bridge-missing", detail: "Pi subagent/team packages own worker lifecycle; no compat extension event is exposed for safe direct mapping" },
  "subagent-stop": { state: "package-backed-bridge-missing", detail: "Pi subagent/team packages own worker lifecycle; main-session agent_end is not a safe SubagentStop equivalent" },
  notification: { state: "extension-connected", detail: "mapped for MoAI compat internal notifyMoai calls; Pi-global notification interception is not available" },
  "permission-request": { state: "intentionally-excluded", detail: "Claude permissionMode parity is excluded by design; pi-yaml-hooks guardrails replace it" },
  "teammate-idle": { state: "package-backed-adapter-available", detail: "pi-agent-teams optional idle hooks can call project-local MoAI adapter scripts when explicitly enabled" },
  "task-completed": { state: "package-backed-adapter-available", detail: "pi-agent-teams optional task_completed hooks can call project-local MoAI adapter scripts when explicitly enabled" },
  "worktree-create": { state: "package-backed-bridge-missing", detail: "pi-agent-teams supports worktree creation internally, but exposes no compat extension hook for this lifecycle" },
  "worktree-remove": { state: "package-backed-bridge-missing", detail: "pi-agent-teams cleans worktrees internally, but exposes no compat extension hook for this lifecycle" },
};

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

export function hookRuntimeClassificationStatus(): string[] {
  const mapped = Object.keys(HOOK_SCRIPT_BY_EVENT);
  const byState = (state: HookRuntimeState) => mapped.filter((event) => HOOK_RUNTIME_CLASSIFICATION[event]?.state === state);
  const connected = byState("extension-connected");
  const packageBackedAvailable = byState("package-backed-adapter-available");
  const packageBackedMissing = byState("package-backed-bridge-missing");
  const adapterNeeded = byState("adapter-needed");
  const excluded = byState("intentionally-excluded");
  const notConnected = mapped.length - connected.length;
  return [
    `partial: extension-connected hook events ${connected.length}/${mapped.length}; ${notConnected} are package-backed, adapter-needed, or intentionally excluded`,
    packageBackedAvailable.length
      ? `partial: package-backed adapter-available hook events ${packageBackedAvailable.join(", ")} (requires explicit Agent Teams hook env activation)`
      : "ok: no package-backed hook adapters are waiting for activation",
    packageBackedMissing.length
      ? `partial: package-backed bridge-missing hook events ${packageBackedMissing.join(", ")} (Pi packages provide the lifecycle, but no safe compat extension event/adapter is installed yet)`
      : "ok: no package-backed hook bridges are missing",
    adapterNeeded.length
      ? `partial: adapter-needed hook events ${adapterNeeded.join(", ")} (compat extension needs explicit runtime adapters)`
      : "ok: no adapter-needed hook events",
    excluded.length
      ? `info: intentionally excluded hook events ${excluded.join(", ")} (replaced by Pi policy/guardrail mechanisms)`
      : "ok: no intentionally excluded hook events",
  ];
}

export function hookRuntimeClassificationDetailStatus(): string[] {
  return Object.entries(HOOK_RUNTIME_CLASSIFICATION)
    .filter(([, classification]) => classification.state !== "extension-connected")
    .map(([event, classification]) => `info: hook ${event} ${classification.state} - ${classification.detail}`);
}

export function hookBridgeParityStatus(): string[] {
  const mapped = Object.keys(HOOK_SCRIPT_BY_EVENT);
  const present = mapped.filter(hasHookScript).length;
  return [
    hookBridgeStatus(),
    ...hookRuntimeClassificationStatus(),
    ...hookRuntimeClassificationDetailStatus(),
    `info: ${NON_BLOCKING_HOOK_BRIDGE_POLICY}`,
    present === mapped.length
      ? "ok: all mapped hook scripts exist"
      : `missing: hook scripts present ${present}/${mapped.length}`,
  ];
}
