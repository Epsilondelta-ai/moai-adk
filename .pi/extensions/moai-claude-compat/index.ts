import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { buildCoreInstruction, loadMoaiCompatConfig, type OutputStyleConfig } from "./src/config.ts";
import { registerCommands } from "./src/command-router.ts";
import { EXTENSION_ID, PI_RULES_SOURCE_PATH } from "./src/constants.ts";
import { NON_BLOCKING_HOOK_BRIDGE_POLICY, runMoaiHook } from "./src/hook-bridge.ts";
import { notifyMoai, type MoaiNotificationContext } from "./src/notification-adapter.ts";
import { updateMoaiStatus } from "./src/statusline.ts";
import { buildSkillTriggerHints } from "./src/trigger-indexer.ts";

function buildCompactOutputStyleInstruction(outputStyle: OutputStyleConfig): string {
  return [
    `MoAI output style '${outputStyle.name}' is active at prompt level.`,
    "Follow concise, professional, transparent, language-aware Markdown.",
    "Use MoAI status/summary formatting only when it improves clarity.",
    "Never display internal completion markers or XML tags in user-facing responses.",
    outputStyle.loaded ? `Style source: ${outputStyle.sourcePath}` : `Style source unavailable: ${outputStyle.error ?? "unknown"}`,
  ].join("\n");
}

export default function moaiClaudeCompat(pi: ExtensionAPI) {
  const config = loadMoaiCompatConfig();
  const coreInstruction = buildCoreInstruction(config);
  const outputStyleInstruction = buildCompactOutputStyleInstruction(config.outputStyle);
  const baseAdditionalContext = [coreInstruction, outputStyleInstruction]
    .filter(Boolean)
    .join("\n\n");

  registerCommands(pi, config);

  async function invokeHook(eventName: string, payload: unknown, ctx?: MoaiNotificationContext) {
    // Compatibility hooks intentionally do not block Pi tool execution here.
    // Security-sensitive blocking is handled by pi-yaml-hooks tool.before.* guardrails.
    const result = await runMoaiHook(eventName, payload);
    if (!result.ok) {
      const detail = (result.stderr || result.stdout || `exit ${result.exitCode ?? "unknown"}`).trim();
      await notifyMoai(ctx, `MoAI hook '${eventName}' failed non-blocking: ${detail}`, "warning", {
        source: "hook-bridge",
        failedHookEvent: eventName,
      });
    }
    return result;
  }

  pi.on("session_start", async (event, ctx) => {
    await invokeHook("session-start", { hook_event_name: "SessionStart", event, cwd: ctx.cwd }, ctx);
    updateMoaiStatus(ctx, config, { phase: "idle" });
    await notifyMoai(ctx, "MoAI pi compatibility layer loaded", "info", {
      source: "moai-claude-compat",
      reason: event.reason,
    });
    pi.appendEntry(`${EXTENSION_ID}:loaded`, {
      conversationLanguage: config.conversationLanguage,
      qualityMode: config.qualityMode,
      permissionMode: "excluded-by-design",
      hookBridgePolicy: NON_BLOCKING_HOOK_BRIDGE_POLICY,
      outputStyle: {
        name: config.outputStyle.name,
        loaded: config.outputStyle.loaded,
        sanitized: config.outputStyle.sanitized,
        enforcement: config.outputStyle.enforcement,
      },
    });
  });

  pi.on("session_shutdown", async (event, ctx) => {
    await invokeHook("session-end", { hook_event_name: "SessionEnd", event, cwd: ctx.cwd }, ctx);
  });

  pi.on("session_before_compact", async (event, ctx) => {
    await invokeHook("compact", { hook_event_name: "PreCompact", event, cwd: ctx.cwd }, ctx);
  });

  pi.on("turn_start", async (_event, ctx) => {
    updateMoaiStatus(ctx, config);
  });

  pi.on("turn_end", async (_event, ctx) => {
    updateMoaiStatus(ctx, config);
  });

  pi.on("input", async (event, ctx) => {
    const text = typeof event.text === "string" ? event.text : typeof event.input === "string" ? event.input : "";
    if (!text.trim()) return;

    const hookResult = await invokeHook("user-prompt-submit", { hook_event_name: "UserPromptSubmit", prompt: text, event, cwd: ctx.cwd }, ctx);
    const lower = text.toLowerCase();
    const hints: string[] = [];
    if (lower.includes("--deepthink")) hints.push("Deepthink requested: prefer sequential-thinking MCP when available.");
    if (lower.includes("spec-") || lower.includes("/moai run")) hints.push(`SPEC workflow context likely required: read .moai/specs and pi-local MoAI workflow rules at ${PI_RULES_SOURCE_PATH}.`);
    if (lower.includes("permissionmode")) hints.push("Reminder: Claude permissionMode is excluded by design in pi parity.");
    hints.push(...buildSkillTriggerHints(text));
    if (hookResult.stdout.trim()) hints.push(`MoAI user-prompt hook output: ${hookResult.stdout.trim()}`);

    return {
      additionalContext: [baseAdditionalContext, ...hints].filter(Boolean).join("\n"),
    };
  });

  pi.on("tool_call", async (event, ctx) => {
    await invokeHook("pre-tool", { hook_event_name: "PreToolUse", tool_name: event.toolName, tool_input: event.input, event, cwd: ctx.cwd }, ctx);
  });

  pi.on("tool_result", async (event, ctx) => {
    const payload = {
      hook_event_name: "PostToolUse",
      tool_name: event.toolName,
      tool_input: event.input,
      tool_response: {
        content: event.content,
        details: event.details,
        isError: event.isError,
      },
      event,
      cwd: ctx.cwd,
    };
    await invokeHook("post-tool", payload, ctx);
    if (event.isError) {
      await invokeHook("post-tool-failure", { ...payload, hook_event_name: "PostToolUseFailure" }, ctx);
    }
  });

  pi.on("agent_end", async (event, ctx) => {
    await invokeHook("stop", { hook_event_name: "Stop", event, cwd: ctx.cwd }, ctx);
  });

}
