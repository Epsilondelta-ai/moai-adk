import type { ExtensionContext } from "@earendil-works/pi-coding-agent";

export interface QuotaSnapshot {
	shortWindowPercent?: number;
	weeklyWindowPercent?: number;
	resetAt?: number;
	provider?: string;
	source?: string;
}

let latestQuota: QuotaSnapshot = {};

export function getQuotaSnapshot(): QuotaSnapshot {
	return { ...latestQuota };
}

export function updateQuotaFromHeaders(headers: Record<string, string | string[] | undefined>, ctx: ExtensionContext): void {
	const normalized = normalizeHeaders(headers);
	const tokenPercent = percentFromLimitRemaining(normalized, [
		["anthropic-ratelimit-tokens-limit", "anthropic-ratelimit-tokens-remaining"],
		["x-ratelimit-limit-tokens", "x-ratelimit-remaining-tokens"],
		["x-ratelimit-limit", "x-ratelimit-remaining"],
	]);
	const requestPercent = percentFromLimitRemaining(normalized, [
		["anthropic-ratelimit-requests-limit", "anthropic-ratelimit-requests-remaining"],
		["x-ratelimit-limit-requests", "x-ratelimit-remaining-requests"],
	]);
	const shortWindowPercent = tokenPercent ?? requestPercent;
	const resetAt = resetFromHeaders(normalized);

	latestQuota = {
		...latestQuota,
		provider: ctx.model?.provider,
		source: shortWindowPercent === undefined ? latestQuota.source : "provider-headers",
		shortWindowPercent: shortWindowPercent ?? latestQuota.shortWindowPercent,
		weeklyWindowPercent: latestQuota.weeklyWindowPercent,
		resetAt: resetAt ?? latestQuota.resetAt,
	};
}

export function updateQuotaFromRetryAfter(headers: Record<string, string | string[] | undefined>, ctx: ExtensionContext): void {
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

function normalizeHeaders(headers: Record<string, string | string[] | undefined>): Record<string, string> {
	const normalized: Record<string, string> = {};
	for (const [key, value] of Object.entries(headers ?? {})) {
		if (value === undefined) continue;
		normalized[key.toLowerCase()] = Array.isArray(value) ? value.join(",") : value;
	}
	return normalized;
}

function percentFromLimitRemaining(headers: Record<string, string>, pairs: Array<[string, string]>): number | undefined {
	for (const [limitKey, remainingKey] of pairs) {
		const limit = numberHeader(headers, limitKey);
		const remaining = numberHeader(headers, remainingKey);
		if (limit !== undefined && remaining !== undefined && limit > 0) {
			return clamp(Math.round(((limit - remaining) / limit) * 100));
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
