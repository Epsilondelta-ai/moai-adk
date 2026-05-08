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
}

let latestQuota: QuotaSnapshot = {};

export function getQuotaSnapshot(): QuotaSnapshot {
	return { ...latestQuota };
}

export function getQuotaDiagnostics(): string {
	const headerKeys = latestQuota.headerKeys ?? [];
	const lines = [
		"MoAI Pi quota diagnostics",
		`provider: ${latestQuota.provider ?? "unknown"}`,
		`source: ${latestQuota.source ?? "not-observed"}`,
		`ST: ${formatDiagnosticQuota(latestQuota.shortWindowPercent, latestQuota.shortWindowLabel)}`,
		`7D: ${formatDiagnosticQuota(latestQuota.weeklyWindowPercent, latestQuota.weeklyWindowLabel)}`,
		`headerKeys: ${headerKeys.length > 0 ? headerKeys.join(", ") : "none"}`,
	];

	if (!latestQuota.source) {
		lines.push("diagnosis: no provider response has been observed by the extension yet.");
	} else if (latestQuota.source === "headers-unavailable") {
		lines.push("diagnosis: Pi/provider did not expose response headers to the extension.");
		lines.push("action: ST/7D cannot be computed for this provider transport until Pi exposes quota metadata or response headers.");
	} else if (latestQuota.source === "headers-without-rate-limit") {
		lines.push("diagnosis: response headers were observed, but no supported rate-limit quota headers were present.");
		lines.push("action: add mappings for any provider-specific quota headers shown above, if available.");
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

	latestQuota = {
		...latestQuota,
		provider: ctx.model?.provider,
		source: shortQuota || weeklyQuota ? "provider-headers" : headerKeys.length > 0 ? "headers-without-rate-limit" : "headers-unavailable",
		shortWindowPercent: shortQuota?.percent,
		shortWindowLabel: shortQuota?.label,
		weeklyWindowPercent: weeklyQuota?.percent,
		weeklyWindowLabel: weeklyQuota?.label,
		resetAt: resetAt ?? latestQuota.resetAt,
		headerKeys,
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
