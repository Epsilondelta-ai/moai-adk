import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { truncateToWidth } from "@earendil-works/pi-tui";
import { callBridge } from "./bridge.js";
import { getQuotaSnapshot } from "./quota.js";
import { normalizeState, type MoaiUIState } from "./state.js";

let currentState: MoaiUIState = {};
let sessionStartedAt = Date.now();
let piVersionCache: string | undefined;

export async function refreshMoaiUI(ctx: ExtensionContext, patch: Record<string, unknown> = {}): Promise<void> {
	const response = await callBridge(ctx, { kind: "state", payload: patch }, { timeoutMs: 10_000, signal: ctx.signal });
	if (response.ok) {
		currentState = normalizeState(response.data);
		if (typeof currentState.sessionStartedAt === "number" && currentState.sessionStartedAt > 0) {
			sessionStartedAt = currentState.sessionStartedAt;
		}
	} else {
		currentState = { phase: "bridge-error", qualityStatus: response.error?.code ?? "error" };
	}
	renderMoaiUI(ctx);
}

export function renderMoaiUI(ctx: ExtensionContext): void {
	ctx.ui.setStatus("moai", ctx.ui.theme.fg("accent", "💬 MoAI"));
	ctx.ui.setWidget("moai-dashboard", undefined);
	ctx.ui.setFooter((tui, footerTheme, footerData) => ({
		dispose: footerData.onBranchChange(() => tui.requestRender()),
		invalidate() {},
		render(width: number) {
			const lines = buildClaudeStyleStatusLines(ctx, footerData.getGitBranch() || undefined, width);
			return lines.map((line) => footerTheme.fg("accent", truncateToWidth(line, width)));
		},
	}));
}

function buildClaudeStyleStatusLines(ctx: ExtensionContext, fallbackBranch?: string, width = 120): string[] {
	const model = modelLabel(ctx);
	const piVersion = currentState.piVersion || getPiVersion();
	const moaiVersion = versionLabel();
	const elapsed = elapsedLabel(Date.now() - sessionStartedAt);
	const outputStyle = "MoAI";
	const contextPct = contextPercent(ctx);
	const quota = getQuotaSnapshot();
	const shortPct = percentValue(quota.shortWindowPercent);
	const weeklyPct = percentValue(quota.weeklyWindowPercent);
	const branch = currentState.gitBranch || fallbackBranch || "no-branch";
	const projectName = currentState.projectName || basename(ctx.cwd);
	const added = currentState.gitAdded ?? 0;
	const modified = currentState.gitModified ?? 0;
	const untracked = currentState.gitUntracked ?? 0;

	return [
		...buildHeaderLines({ model, piVersion, moaiVersion, elapsed, outputStyle }, width),
		...buildQuotaLines({ contextPct, shortPct, weeklyPct, shortLabel: quota.shortWindowLabel, weeklyLabel: quota.weeklyWindowLabel }, width),
		`📁 ${projectName} │ 🔀 ${branch} │ 📊 +${added} M${modified} ?${untracked}`,
	];
}

interface HeaderParts {
	model: string;
	piVersion: string;
	moaiVersion: string;
	elapsed: string;
	outputStyle: string;
}

interface QuotaParts {
	contextPct: number;
	shortPct?: number;
	weeklyPct?: number;
	shortLabel?: string;
	weeklyLabel?: string;
}

function buildHeaderLines(parts: HeaderParts, width: number): string[] {
	const full = `🤖 ${parts.model} │ 🔅 v${stripV(parts.piVersion)} │ 🗿 ${parts.moaiVersion} │ ⏳ ${parts.elapsed} │ 💬 ${parts.outputStyle}`;
	if (displayWidth(full) <= width) return [full];

	const compactModel = compactModelLabel(parts.model, Math.max(10, Math.min(18, width - 20)));
	const primary = `⏳ ${parts.elapsed} │ 🤖 ${compactModel} │ 💬 ${parts.outputStyle}`;
	const version = `🔅 v${stripV(parts.piVersion)} │ 🗿 ${parts.moaiVersion}`;
	return displayWidth(primary) <= width
		? [primary, version]
		: [`⏳ ${parts.elapsed} │ 💬 ${parts.outputStyle}`, `🤖 ${compactModel} │ ${version}`];
}

function buildQuotaLines(parts: QuotaParts, width: number): string[] {
	const full = `${usageSegment("CW:", parts.contextPct, 10)} │ ${usageSegment("ST:", parts.shortPct, 10, parts.shortLabel)} │ ${usageSegment("7D:", parts.weeklyPct, 10, parts.weeklyLabel)}`;
	if (displayWidth(full) <= width) return [full];

	const barWidth = width < 24 ? 0 : Math.max(4, Math.min(10, Math.floor((width - 14) / 2)));
	return [
		usageSegment("CW:", parts.contextPct, barWidth),
		usageSegment("ST:", parts.shortPct, barWidth, parts.shortLabel),
		usageSegment("7D:", parts.weeklyPct, barWidth, parts.weeklyLabel),
	];
}

function usageSegment(label: string, percent: number | undefined, barWidth: number, quotaLabel?: string): string {
	if (percent === undefined) return `${label} ?`;
	const usageBar = barWidth > 0 ? ` ${bar(percent, barWidth)}` : "";
	const labelSuffix = quotaLabel ? ` ${quotaLabel}` : ` ${percent}%`;
	return `${label} ${batteryIcon(percent)}${usageBar}${labelSuffix}`;
}

function compactModelLabel(model: string, limit: number): string {
	const compact = model.replace(/\s*\(1M context\)/i, "").trim();
	if (displayWidth(compact) <= limit) return compact;
	return `${compact.slice(0, Math.max(1, limit - 1)).trim()}…`;
}

function displayWidth(value: string): number {
	return Array.from(value).length;
}

export function __testBuildFooterLines(input: {
	model?: string;
	piVersion?: string;
	moaiVersion?: string;
	elapsed?: string;
	contextPct?: number;
	shortPct?: number;
	weeklyPct?: number;
}, width: number): string[] {
	return [
		...buildHeaderLines(
			{
				model: input.model ?? "Opus 4.7 (1M context)",
				piVersion: input.piVersion ?? "2.1.50",
				moaiVersion: input.moaiVersion ?? "v2.8.0",
				elapsed: input.elapsed ?? "1h 23m",
				outputStyle: "MoAI",
			},
			width,
		),
		...buildQuotaLines(
			{ contextPct: input.contextPct ?? 30, shortPct: input.shortPct, weeklyPct: input.weeklyPct },
			width,
		),
	];
}

function modelLabel(ctx: ExtensionContext): string {
	const model = ctx.model;
	const raw = model?.name || model?.id || "Model";
	let label = raw
		.replace(/^claude-/i, "")
		.replace(/-20\d{6}$/i, "")
		.replace(/-/g, " ")
		.replace(/\b\w/g, (c) => c.toUpperCase());
	if (/opus/i.test(label) && !/1M context/i.test(label)) label += " (1M context)";
	return label;
}

function versionLabel(): string {
	const current = currentState.moaiVersion ? `v${stripV(currentState.moaiVersion)}` : "v?";
	if (currentState.moaiLatestVersion && stripV(currentState.moaiLatestVersion) !== stripV(currentState.moaiVersion || "")) {
		return `${current} ⬆️ v${stripV(currentState.moaiLatestVersion)}`;
	}
	return current;
}

function getPiVersion(): string {
	if (piVersionCache) return piVersionCache;
	const candidates = [
		join(dirname(fileURLToPath(import.meta.url)), "..", "..", "..", "package.json"),
		"/opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent/package.json",
	];
	for (const candidate of candidates) {
		try {
			const pkg = JSON.parse(readFileSync(candidate, "utf8")) as { version?: string };
			if (pkg.version) {
				piVersionCache = pkg.version;
				return piVersionCache;
			}
		} catch {
			// try next
		}
	}
	return "?";
}

function contextPercent(ctx: ExtensionContext): number {
	try {
		const usage = ctx.getContextUsage?.();
		const tokens = usage?.tokens ?? 0;
		const window = (ctx.model as { contextWindow?: number } | undefined)?.contextWindow ?? 200000;
		if (tokens > 0 && window > 0) return clamp(Math.round((tokens / window) * 100));
	} catch {
		// ignore
	}
	return 0;
}

function bar(percent: number, width: number): string {
	const filled = Math.max(0, Math.min(width, Math.round((percent / 100) * width)));
	return `${"█".repeat(filled)}${"░".repeat(width - filled)}`;
}

function percentValue(value: unknown): number | undefined {
	return typeof value === "number" ? clamp(Math.round(value)) : undefined;
}

function batteryIcon(percent: number): string {
	return percent >= 80 ? "🪫" : "🔋";
}

function elapsedLabel(ms: number): string {
	const minutes = Math.max(0, Math.floor(ms / 60000));
	const hours = Math.floor(minutes / 60);
	const mins = minutes % 60;
	if (hours > 0) return `${hours}h ${mins}m`;
	return `${mins}m`;
}

function stripV(version: string): string {
	return version.replace(/^v/i, "");
}

function clamp(value: number): number {
	return Math.max(0, Math.min(100, value));
}

function basename(path: string): string {
	const parts = path.split(/[\\/]/).filter(Boolean);
	return parts[parts.length - 1] || path;
}

export function registerMoaiUI(pi: ExtensionAPI): void {
	pi.registerCommand("moai-ui-refresh", {
		description: "Refresh MoAI footer/widget UI state",
		handler: async (_args, ctx) => refreshMoaiUI(ctx, { phase: "manual-refresh" }),
	});
}
