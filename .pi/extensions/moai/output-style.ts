import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { BridgeResponse } from "./types.js";

const MOAI_STYLE_MARKER = "MoAI Pi Output Style v1";

export function shouldInjectMoaiOutputStyle(cwd: string, prompt: string): boolean {
	const normalized = prompt.toLowerCase();
	if (normalized.includes("/moai") || normalized.includes("moai") || normalized.includes("phase ")) {
		return true;
	}
	return existsSync(join(cwd, ".moai"));
}

export function moaiOutputStylePrompt(cwd: string): string {
	const language = conversationLanguage(cwd);
	const korean = language === "ko";
	const complete = korean ? "완료" : "Complete";
	const taskStart = korean ? "작업 시작" : "Task Start";
	const gate = korean ? "검증" : "Gate";
	const error = korean ? "오류" : "Error";

	return `

# ${MOAI_STYLE_MARKER}

This repository is a MoAI-enabled project. Every user-facing assistant response in this project MUST start with a concise MoAI-branded frame, even when the user does not type /moai.

For trivial informational answers, use the compact completion template. For multi-step work, use task start, gate, and completion templates as appropriate.

Never use XML tags in user-facing output. Use the user's conversation language for labels and summaries. Structural icons may remain as icons.

Task start template:
\`\`\`
🤖 MoAI ★ ${taskStart} ─────────────────────────
📋 [intent]
🎯 [success criterion]
⏳ [current stage]
──────────────────────────────────────────────
\`\`\`

Gate/checkpoint template:
\`\`\`
🤖 MoAI ★ ${gate} ─────────────────────────
✅ [verified items]
📊 [evidence summary]
──────────────────────────────────────────────
\`\`\`

Completion template:
\`\`\`
🤖 MoAI ★ ${complete} ─────────────────────────
✅ [result summary]
📊 [files/tests/state summary]
──────────────────────────────────────────────
\`\`\`

Error template:
\`\`\`
🤖 MoAI ★ ${error} ─────────────────────────
❌ [what failed]
🔧 [next recovery action]
──────────────────────────────────────────────
\`\`\`
`.trim();
}

export function formatMoaiBridgeResponse(response: BridgeResponse, cwd: string): string {
	const korean = conversationLanguage(cwd) === "ko";
	const title = response.ok ? (korean ? "완료" : "Complete") : (korean ? "오류" : "Error");
	const statusLine = response.ok
		? `✅ ${response.message ?? (korean ? "요청 완료" : "Request completed")}`
		: `❌ ${response.error?.message ?? (korean ? "알 수 없는 오류" : "Unknown error")}`;
	const detailLine = response.ok
		? bridgeSummary(response, korean)
		: `🔧 ${response.error?.code ?? "unknown"}`;

	return [
		`🤖 MoAI ★ ${title} ─────────────────────────`,
		statusLine,
		detailLine,
		"────────────────────────────────────────────",
	].join("\n");
}

function bridgeSummary(response: BridgeResponse, korean: boolean): string {
	const data = response.data ?? {};
	const commandResult = data.commandResult as { command?: string; ok?: boolean; artifacts?: Array<{ path?: string; type?: string }>; messages?: Array<{ text?: string }> } | undefined;
	if (commandResult) {
		const artifacts = Array.isArray(commandResult.artifacts) ? commandResult.artifacts.length : 0;
		const command = commandResult.command ?? "command";
		return `📊 ${korean ? "명령" : "Command"}: ${command} │ ${korean ? "아티팩트" : "Artifacts"}: ${artifacts}`;
	}
	if (data.protocol) {
		return `📊 protocol: ${String(data.protocol)}`;
	}
	if (data.cwd) {
		return `📊 cwd: ${String(data.cwd)}`;
	}
	return `📊 ${korean ? "상태" : "Status"}: ok`;
}

function conversationLanguage(cwd: string): string {
	const path = join(cwd, ".moai", "config", "sections", "language.yaml");
	try {
		const content = readFileSync(path, "utf8");
		const match = content.match(/conversation_language(?:_name)?:\s*['\"]?([^'\"\s#]+)/);
		if (match?.[1]) return match[1].toLowerCase();
	} catch {
		// default below
	}
	return "en";
}
