import { closeSync, existsSync, openSync, readdirSync, readSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { basename, join } from "node:path";
import type { ExtensionContext } from "@earendil-works/pi-coding-agent";

export interface QuotaSnapshot {
	shortWindowPercent?: number;
	weeklyWindowPercent?: number;
	shortWindowLabel?: string;
	weeklyWindowLabel?: string;
	resetAt?: number;
	provider?: string;
	source?: string;
	headerKeys?: string[];
	observedAt?: number;
	codexSessionFile?: string;
	planType?: string;
}

const CODEX_RATE_LIMIT_MAX_AGE_MS = 60 * 60 * 1000;

let latestQuota: QuotaSnapshot = {};
let codexCacheLastChecked = 0;

export function getQuotaSnapshot(): QuotaSnapshot {
	refreshQuotaFromCodexSessionCache();
	return { ...latestQuota };
}

export function getQuotaDiagnostics(): string {
	refreshQuotaFromCodexSessionCache();
	const headerKeys = latestQuota.headerKeys ?? [];
	const lines = [
		"MoAI Pi quota diagnostics",
		`provider: ${latestQuota.provider ?? "unknown"}`,
		`source: ${latestQuota.source ?? "not-observed"}`,
		`ST: ${formatDiagnosticQuota(latestQuota.shortWindowPercent, latestQuota.shortWindowLabel)}`,
		`7D: ${formatDiagnosticQuota(latestQuota.weeklyWindowPercent, latestQuota.weeklyWindowLabel)}`,
		`headerKeys: ${headerKeys.length > 0 ? headerKeys.join(", ") : "none"}`,
		`observedAt: ${latestQuota.observedAt ? new Date(latestQuota.observedAt).toISOString() : "unknown"}`,
		`codexSession: ${latestQuota.codexSessionFile ? basename(latestQuota.codexSessionFile) : "none"}`,
		`planType: ${latestQuota.planType ?? "unknown"}`,
	];

	if (!latestQuota.source) {
		lines.push("diagnosis: no provider response has been observed by the extension yet.");
	} else if (latestQuota.source === "headers-unavailable") {
		lines.push("diagnosis: Pi/provider did not expose response headers to the extension.");
		lines.push("action: ST/7D cannot be computed for this provider transport until Pi exposes quota metadata or response headers.");
	} else if (latestQuota.source === "headers-without-rate-limit") {
		lines.push("diagnosis: response headers were observed, but no supported rate-limit quota headers were present.");
		lines.push("action: add mappings for any provider-specific quota headers shown above, if available.");
	} else if (latestQuota.source?.includes("codex-session-cache")) {
		lines.push("diagnosis: using the latest Codex session rate_limits cache, matching the data source behind Codex/oh-my-codex quota display.");
	} else if (latestQuota.shortWindowPercent === undefined) {
		lines.push("diagnosis: provider headers were observed, but short-window quota was not present.");
	} else if (latestQuota.weeklyWindowPercent === undefined) {
		lines.push("diagnosis: short-window quota is available; weekly quota was not exposed by the provider.");
	}
	return lines.join("\n");
}

export function updateQuotaFromHeaders(headers: unknown, ctx: ExtensionContext): void {
	const normalized = normalizeHeaders(headers);
	const tokenQuota = quotaFromLimitRemaining(normalized, [
		["anthropic-ratelimit-tokens-limit", "anthropic-ratelimit-tokens-remaining"],
		["x-ratelimit-limit-tokens", "x-ratelimit-remaining-tokens"],
		["x-ratelimit-limit", "x-ratelimit-remaining"],
		["x-ms-ratelimit-limit-tokens", "x-ms-ratelimit-remaining-tokens"],
	]);
	const requestQuota = quotaFromLimitRemaining(normalized, [
		["anthropic-ratelimit-requests-limit", "anthropic-ratelimit-requests-remaining"],
		["x-ratelimit-limit-requests", "x-ratelimit-remaining-requests"],
		["x-ms-ratelimit-limit-requests", "x-ms-ratelimit-remaining-requests"],
	]);
	const weeklyQuota = quotaFromLimitRemaining(normalized, [
		["x-ratelimit-limit-tokens-7d", "x-ratelimit-remaining-tokens-7d"],
		["x-ratelimit-limit-7d", "x-ratelimit-remaining-7d"],
		["x-weekly-ratelimit-limit", "x-weekly-ratelimit-remaining"],
	]);
	const shortQuota = tokenQuota ?? requestQuota;
	const resetAt = resetFromHeaders(normalized);
	const headerKeys = Object.keys(normalized).sort();
	const source = shortQuota || weeklyQuota
		? "provider-headers"
		: latestQuota.source?.includes("codex-session-cache")
			? `codex-session-cache+${headerKeys.length > 0 ? "headers-without-rate-limit" : "headers-unavailable"}`
			: headerKeys.length > 0 ? "headers-without-rate-limit" : "headers-unavailable";

	latestQuota = {
		...latestQuota,
		provider: ctx.model?.provider,
		source,
		shortWindowPercent: shortQuota?.percent ?? latestQuota.shortWindowPercent,
		shortWindowLabel: shortQuota?.label ?? latestQuota.shortWindowLabel,
		weeklyWindowPercent: weeklyQuota?.percent ?? latestQuota.weeklyWindowPercent,
		weeklyWindowLabel: weeklyQuota?.label ?? latestQuota.weeklyWindowLabel,
		resetAt: resetAt ?? latestQuota.resetAt,
		headerKeys,
		observedAt: Date.now(),
	};
}

export function updateQuotaFromRetryAfter(headers: unknown, ctx: ExtensionContext): void {
	const normalized = normalizeHeaders(headers);
	const retryAfter = numberHeader(normalized, "retry-after");
	if (retryAfter === undefined) return;
	latestQuota = {
		...latestQuota,
		provider: ctx.model?.provider,
		source: "retry-after",
		shortWindowPercent: 100,
		resetAt: Date.now() + retryAfter * 1000,
		observedAt: Date.now(),
	};
}

export function refreshQuotaFromCodexSessionCache(options: { force?: boolean } = {}): void {
	const now = Date.now();
	if (!options.force && now - codexCacheLastChecked < 30_000) return;
	codexCacheLastChecked = now;

	const observed = readLatestCodexRateLimits();
	if (!observed) {
		if (latestQuota.source?.includes("codex-session-cache") && !latestQuota.source.startsWith("provider-headers")) {
			latestQuota = {
				...latestQuota,
				source: "codex-session-cache-stale",
				shortWindowPercent: undefined,
				weeklyWindowPercent: undefined,
				shortWindowLabel: undefined,
				weeklyWindowLabel: undefined,
			};
		}
		return;
	}

	const preserveProviderQuota = latestQuota.source?.startsWith("provider-headers") ?? false;
	const shortWindowPercent = preserveProviderQuota ? latestQuota.shortWindowPercent ?? observed.shortWindowPercent : observed.shortWindowPercent;
	const weeklyWindowPercent = preserveProviderQuota ? latestQuota.weeklyWindowPercent ?? observed.weeklyWindowPercent : observed.weeklyWindowPercent;
	if (shortWindowPercent === undefined && weeklyWindowPercent === undefined) return;

	latestQuota = {
		...latestQuota,
		provider: latestQuota.provider ?? "codex",
		source: latestQuota.source === "provider-headers" ? "provider-headers+codex-session-cache" : "codex-session-cache",
		shortWindowPercent,
		weeklyWindowPercent,
		resetAt: latestQuota.resetAt ?? observed.resetAt,
		observedAt: observed.observedAt,
		codexSessionFile: observed.file,
		planType: observed.planType,
	};
}

function normalizeHeaders(headers: unknown): Record<string, string> {
	const normalized: Record<string, string> = {};
	if (!headers) return normalized;

	const setHeader = (key: unknown, value: unknown) => {
		if (typeof key !== "string" || value === undefined || value === null) return;
		normalized[key.toLowerCase()] = Array.isArray(value) ? value.join(",") : String(value);
	};

	if (typeof (headers as { forEach?: unknown }).forEach === "function") {
		(headers as { forEach: (callback: (value: unknown, key: unknown) => void) => void }).forEach((value, key) => setHeader(key, value));
		return normalized;
	}
	if (typeof (headers as { entries?: unknown }).entries === "function") {
		for (const [key, value] of headers as Iterable<[unknown, unknown]>) setHeader(key, value);
		return normalized;
	}
	for (const [key, value] of Object.entries(headers as Record<string, unknown>)) setHeader(key, value);
	return normalized;
}

function quotaFromLimitRemaining(headers: Record<string, string>, pairs: Array<[string, string]>): { percent: number; label: string } | undefined {
	for (const [limitKey, remainingKey] of pairs) {
		const limit = numberHeader(headers, limitKey);
		const remaining = numberHeader(headers, remainingKey);
		if (limit !== undefined && remaining !== undefined && limit > 0) {
			return {
				percent: clamp(Math.round(((limit - remaining) / limit) * 100)),
				label: `${formatQuotaNumber(remaining)}/${formatQuotaNumber(limit)}`,
			};
		}
	}
	return undefined;
}

interface CodexRateLimitObservation {
	shortWindowPercent?: number;
	weeklyWindowPercent?: number;
	resetAt?: number;
	observedAt: number;
	file: string;
	planType?: string;
}

function readLatestCodexRateLimits(): CodexRateLimitObservation | undefined {
	const sessionsRoot = join(homedir(), ".codex", "sessions");
	if (!existsSync(sessionsRoot)) return undefined;
	for (const file of recentJsonlFiles(sessionsRoot, 24)) {
		const observed = readRateLimitsFromJsonlTail(file);
		if (observed) return observed;
	}
	return undefined;
}

function recentJsonlFiles(root: string, limit: number): string[] {
	const files: Array<{ path: string; mtimeMs: number }> = [];
	const visit = (dir: string, depth: number) => {
		if (depth > 6) return;
		let entries: string[];
		try {
			entries = readdirSync(dir);
		} catch {
			return;
		}
		for (const entry of entries) {
			const path = join(dir, entry);
			let stat;
			try {
				stat = statSync(path);
			} catch {
				continue;
			}
			if (stat.isDirectory()) visit(path, depth + 1);
			else if (entry.endsWith(".jsonl")) files.push({ path, mtimeMs: stat.mtimeMs });
		}
	};
	visit(root, 0);
	return files.sort((a, b) => b.mtimeMs - a.mtimeMs).slice(0, limit).map((file) => file.path);
}

function readRateLimitsFromJsonlTail(file: string): CodexRateLimitObservation | undefined {
	let fd: number | undefined;
	try {
		const stat = statSync(file);
		const length = Math.min(stat.size, 512 * 1024);
		const buffer = Buffer.alloc(length);
		fd = openSync(file, "r");
		readSync(fd, buffer, 0, length, stat.size - length);
		const lines = buffer.toString("utf8").split(/\r?\n/);
		for (let i = lines.length - 1; i >= 0; i--) {
			const line = lines[i]?.trim();
			if (!line || !line.includes("rate_limits")) continue;
			const observation = codexRateLimitObservationFromLine(line, file);
			if (observation) return observation;
		}
	} catch {
		return undefined;
	} finally {
		if (fd !== undefined) {
			try {
				closeSync(fd);
			} catch {
				// Ignore close errors for best-effort cache reads.
			}
		}
	}
	return undefined;
}

function codexRateLimitObservationFromLine(line: string, file: string): CodexRateLimitObservation | undefined {
	let entry: any;
	try {
		entry = JSON.parse(line);
	} catch {
		return undefined;
	}
	const payload = entry?.payload ?? entry;
	const rateLimits = payload?.rate_limits ?? entry?.rate_limits;
	if (!rateLimits || typeof rateLimits !== "object") return undefined;

	const primary = rateLimits.primary;
	const secondary = rateLimits.secondary;
	const observedAt = Date.parse(entry?.timestamp) || Date.now();
	const shortWindowPercent = isFreshCodexLimit(primary, observedAt) ? percentFromCodexLimit(primary) : undefined;
	const weeklyWindowPercent = isFreshCodexLimit(secondary, observedAt) ? percentFromCodexLimit(secondary) : undefined;
	if (shortWindowPercent === undefined && weeklyWindowPercent === undefined) return undefined;

	return {
		shortWindowPercent,
		weeklyWindowPercent,
		resetAt: resetAtFromCodexLimit(primary),
		observedAt,
		file,
		planType: typeof rateLimits.plan_type === "string" ? rateLimits.plan_type : undefined,
	};
}

function isFreshCodexLimit(limit: unknown, observedAt: number): boolean {
	if (!limit || typeof limit !== "object") return false;
	const now = Date.now();
	if (now - observedAt > CODEX_RATE_LIMIT_MAX_AGE_MS) return false;
	const resetAt = resetAtFromCodexLimit(limit);
	return resetAt === undefined || resetAt > now;
}

function percentFromCodexLimit(limit: unknown): number | undefined {
	if (!limit || typeof limit !== "object") return undefined;
	const used = numericField(limit, "used_percent", "used_percentage");
	if (used !== undefined) return clamp(Math.round(used));
	const remaining = numericField(limit, "remaining_percent", "remaining_percentage");
	return remaining !== undefined ? clamp(Math.round(100 - remaining)) : undefined;
}

function numericField(source: unknown, ...keys: string[]): number | undefined {
	if (!source || typeof source !== "object") return undefined;
	for (const key of keys) {
		const raw = (source as Record<string, unknown>)[key];
		const value = typeof raw === "number" ? raw : typeof raw === "string" ? Number(raw) : undefined;
		if (value !== undefined && Number.isFinite(value)) return value;
	}
	return undefined;
}

function resetAtFromCodexLimit(limit: unknown): number | undefined {
	if (!limit || typeof limit !== "object") return undefined;
	const resetsAt = (limit as { resets_at?: unknown }).resets_at;
	const value = typeof resetsAt === "number" ? resetsAt : typeof resetsAt === "string" ? Number(resetsAt) : undefined;
	if (value === undefined || !Number.isFinite(value)) return undefined;
	return value > 10_000_000_000 ? value : value * 1000;
}

function resetFromHeaders(headers: Record<string, string>): number | undefined {
	for (const key of ["anthropic-ratelimit-tokens-reset", "anthropic-ratelimit-requests-reset", "x-ratelimit-reset", "x-ratelimit-reset-tokens"]) {
		const raw = headers[key];
		if (!raw) continue;
		const numeric = Number(raw);
		if (Number.isFinite(numeric)) {
			return numeric > 10_000_000_000 ? numeric : numeric * 1000;
		}
		const parsed = Date.parse(raw);
		if (!Number.isNaN(parsed)) return parsed;
	}
	return undefined;
}

function formatDiagnosticQuota(percent: number | undefined, label: string | undefined): string {
	if (percent === undefined) return "unknown";
	return label ? `${percent}% used (${label} remaining/limit)` : `${percent}% used`;
}

function formatQuotaNumber(value: number): string {
	if (value >= 1_000_000) return `${Math.round(value / 1_000_000)}M`;
	if (value >= 1_000) return `${Math.round(value / 1_000)}K`;
	return String(value);
}

function numberHeader(headers: Record<string, string>, key: string): number | undefined {
	const raw = headers[key];
	if (!raw) return undefined;
	const first = raw.split(",")[0]?.trim();
	const value = Number(first);
	return Number.isFinite(value) ? value : undefined;
}

function clamp(value: number): number {
	return Math.max(0, Math.min(100, value));
}
