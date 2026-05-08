import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { callBridge } from "./bridge.js";
import { formatMoaiBridgeResponse } from "./output-style.js";

const KNOWN_SUBCOMMANDS = new Set([
	"plan",
	"run",
	"sync",
	"design",
	"db",
	"project",
	"fix",
	"loop",
	"mx",
	"feedback",
	"review",
	"clean",
	"codemaps",
	"coverage",
	"e2e",
	"gate",
	"brain",
]);

function parseMoaiArgs(args: string): { command: string; rest: string } {
	const trimmed = args.trim();
	if (!trimmed) return { command: "capabilities", rest: "" };
	const [first, ...rest] = trimmed.split(/\s+/);
	if (KNOWN_SUBCOMMANDS.has(first)) {
		return { command: first, rest: trimmed.slice(first.length).trim() };
	}
	return { command: "default", rest: trimmed };
}

export function registerCommands(pi: ExtensionAPI): void {
	pi.registerCommand("moai", {
		description: "MoAI workflow entrypoint: /moai <plan|run|sync|review|...>",
		handler: async (args, ctx) => {
			const parsed = parseMoaiArgs(args);
			const kind = parsed.command === "capabilities" ? "capabilities" : "command";
			const response = await callBridge(ctx, {
				kind,
				payload: kind === "command" ? { command: parsed.command, args: parsed.rest, raw: args } : {},
			});
			pi.sendMessage({
				customType: "moai-pi",
				content: formatMoaiBridgeResponse(response, ctx.cwd),
				display: true,
				details: response,
			});
			// Commands are executed by the MoAI Runtime Kernel through the bridge.
			// No prompt handoff is needed for normal command execution.
		},
	});

	pi.registerCommand("moai-doctor", {
		description: "Check MoAI Pi bridge readiness",
		handler: async (_args, ctx) => {
			const response = await callBridge(ctx, { kind: "doctor" });
			pi.sendMessage({
				customType: "moai-pi",
				content: formatMoaiBridgeResponse(response, ctx.cwd),
				display: true,
				details: response,
			});
		},
	});
}
