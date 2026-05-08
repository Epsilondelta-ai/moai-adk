import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { isToolCallEventType } from "@earendil-works/pi-coding-agent";
import { callBridge } from "./bridge.js";
import { updateQuotaFromHeaders, updateQuotaFromRetryAfter } from "./quota.js";
import { shouldInjectMoaiOutputStyle, moaiOutputStylePrompt } from "./output-style.js";
import { refreshMoaiUI } from "./ui.js";

async function forwardEvent(
	ctx: ExtensionContext,
	event: string,
	payload: Record<string, unknown> = {},
	options: { timeoutMs?: number; signal?: AbortSignal } = {},
) {
	return callBridge(
		ctx,
		{ kind: "event", payload: { event, ...payload } },
		{ timeoutMs: options.timeoutMs ?? 5_000, signal: options.signal },
	);
}

export function registerEvents(pi: ExtensionAPI): void {
	pi.on("session_start", async (event, ctx) => {
		const response = await callBridge(ctx, { kind: "doctor", payload: { reason: event.reason } }, { timeoutMs: 10_000 });
		if (!response.ok) {
			ctx.ui.setStatus("moai", ctx.ui.theme.fg("warning", "MoAI bridge unavailable"));
		}
		await forwardEvent(ctx, "session_start", { reason: event.reason, previousSessionFile: event.previousSessionFile });
		await refreshMoaiUI(ctx, { phase: "session_start" });
	});

	pi.on("session_shutdown", async (event, ctx) => {
		await forwardEvent(ctx, "session_shutdown", {
			reason: event.reason,
			targetSessionFile: event.targetSessionFile,
		});
	});

	pi.on("session_before_compact", async (event, ctx) => {
		await forwardEvent(ctx, "session_before_compact", {
			tokensBefore: event.preparation?.tokensBefore,
			firstKeptEntryId: event.preparation?.firstKeptEntryId,
			customInstructions: event.customInstructions,
		}, { signal: event.signal });
		return undefined;
	});

	pi.on("session_compact", async (event, ctx) => {
		await forwardEvent(ctx, "session_compact", {
			fromExtension: event.fromExtension,
			compactionEntryId: event.compactionEntry?.id,
		});
		await refreshMoaiUI(ctx, { phase: "session_compact" });
	});

	pi.on("input", async (event, ctx) => {
		await forwardEvent(ctx, "input", { source: event.source, text: event.text });
		await refreshMoaiUI(ctx, { phase: "input" });
		return { action: "continue" };
	});

	pi.on("before_agent_start", async (event, ctx) => {
		await forwardEvent(
			ctx,
			"before_agent_start",
			{
				prompt: event.prompt,
				imageCount: event.images?.length ?? 0,
				selectedTools: event.systemPromptOptions?.selectedTools,
			},
			{ timeoutMs: 5_000, signal: ctx.signal },
		);
		if (!shouldInjectMoaiOutputStyle(ctx.cwd, event.prompt)) {
			return;
		}
		const stylePrompt = moaiOutputStylePrompt(ctx.cwd);
		if (event.systemPrompt.includes("MoAI Pi Output Style v1")) {
			return;
		}
		return { systemPrompt: `${event.systemPrompt}\n\n${stylePrompt}` };
	});

	pi.on("agent_start", async (_event, ctx) => {
		await forwardEvent(ctx, "agent_start");
		await refreshMoaiUI(ctx, { phase: "agent_start" });
	});

	pi.on("turn_start", async (event, ctx) => {
		await forwardEvent(ctx, "turn_start", { turnIndex: event.turnIndex, timestamp: event.timestamp }, { signal: ctx.signal });
		await refreshMoaiUI(ctx, { phase: "turn_start" });
	});

	pi.on("tool_call", async (event, ctx) => {
		const response = await forwardEvent(
			ctx,
			"tool_call",
			{
				toolName: event.toolName,
				toolCallId: event.toolCallId,
				input: event.input,
			},
			{ timeoutMs: 10_000, signal: ctx.signal },
		);
		const decision = response.data?.decision as { block?: boolean; reason?: string } | undefined;
		if (decision?.block) {
			return { block: true, reason: decision.reason ?? "Blocked by MoAI policy" };
		}

		// UI fallback for destructive commands when policy bridge is present but the
		// command falls outside the strict runtime-neutral blocker rules.
		if (isToolCallEventType("bash", event)) {
			const command = event.input.command ?? "";
			if (/\brm\s+-rf\s+(\/|~|\*)/.test(command) && ctx.hasUI) {
				const ok = await ctx.ui.confirm("MoAI safety gate", `Allow destructive command?\n\n${command}`);
				if (!ok) return { block: true, reason: "Blocked by MoAI safety gate" };
			}
		}
	});

	pi.on("tool_result", async (event, ctx) => {
		await forwardEvent(
			ctx,
			"tool_result",
			{
				toolName: event.toolName,
				toolCallId: event.toolCallId,
				isError: event.isError,
				details: event.details,
			},
			{ timeoutMs: 10_000, signal: ctx.signal },
		);
		await refreshMoaiUI(ctx, { phase: "tool_result", qualityStatus: event.isError ? "failed" : "unknown" });
	});

	pi.on("turn_end", async (event, ctx) => {
		await forwardEvent(ctx, "turn_end", {
			turnIndex: event.turnIndex,
			toolResultCount: event.toolResults?.length ?? 0,
			messageRole: event.message?.role,
		}, { signal: ctx.signal });
		await refreshMoaiUI(ctx, { phase: "turn_end" });
	});

	pi.on("after_provider_response", async (event, ctx) => {
		updateQuotaFromHeaders(event.headers, ctx);
		if (event.status === 429) {
			updateQuotaFromRetryAfter(event.headers, ctx);
		}
		await forwardEvent(ctx, "after_provider_response", { status: event.status }, { signal: ctx.signal });
		await refreshMoaiUI(ctx, { phase: event.status === 429 ? "rate_limited" : "provider_response" });
	});

	pi.on("agent_end", async (event, ctx) => {
		await forwardEvent(ctx, "agent_end", { messageCount: event.messages?.length ?? 0 }, { signal: ctx.signal });
		await refreshMoaiUI(ctx, { phase: "agent_end" });
	});
}
