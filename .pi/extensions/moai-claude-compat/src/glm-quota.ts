import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";

const POLL_INTERVAL_MS = 60_000;
const MIN_EVENT_REFRESH_MS = 15_000;
const STALE_THRESHOLD_MS = 15 * 60_000;
const DEFAULT_ZAI_QUOTA_INTL_URL = "https://api.z.ai/api/monitor/usage/quota/limit";
const DEFAULT_ZAI_QUOTA_CN_URL = "https://open.bigmodel.cn/api/monitor/usage/quota/limit";

type GlmQuotaWindow = {
  label: "5H:" | "7D:";
  usedPercent: number;
  resetsAtMs?: number;
};

export type GlmQuotaSnapshot = {
  source: "live" | "cached";
  capturedAtMs: number;
  stale: boolean;
  primary?: GlmQuotaWindow;
  secondary?: GlmQuotaWindow;
  error?: string;
};

type ZaiQuotaLimit = {
  type?: string;
  percentage?: number | string;
  used_percent?: number | string;
  usedPercentage?: number | string;
  nextResetTime?: number | string;
  resetTime?: number | string;
  resetsAt?: number | string;
};

type ZaiQuotaPayload = {
  data?: {
    limits?: ZaiQuotaLimit[] | null;
  } | null;
  limits?: ZaiQuotaLimit[] | null;
};

type GlmResolvedAuth = {
  token: string;
  headers: Record<string, string>;
};

type GlmFetchDeps = {
  fetchFn: typeof fetch;
};

let latestSnapshot: GlmQuotaSnapshot | undefined;
let refreshInFlight: Promise<void> | undefined;
let refreshInFlightKey = "";
let refreshGeneration = 0;
let pollTimer: ReturnType<typeof setInterval> | undefined;
let activeCtx: ExtensionContext | undefined;
let lastRefreshStartedAt = 0;
let shutdownRequested = false;

export function registerGlmQuota(pi: ExtensionAPI, onUpdate: (ctx: ExtensionContext) => void): void {
  async function refresh(ctx: ExtensionContext, options?: { force?: boolean }): Promise<void> {
    activeCtx = ctx;
    if (!isGlmModel(ctx)) {
      latestSnapshot = undefined;
      refreshGeneration++;
      onUpdate(ctx);
      return;
    }

    const now = Date.now();
    const requestKey = modelContextKey(ctx);
    if (!options?.force && now - lastRefreshStartedAt < 2_000) return refreshInFlight;
    if (!options?.force && refreshInFlight && refreshInFlightKey === requestKey) return refreshInFlight;

    const generation = ++refreshGeneration;
    lastRefreshStartedAt = now;
    refreshInFlightKey = requestKey;
    const currentRefresh = (async () => {
      let nextSnapshot: GlmQuotaSnapshot | undefined;
      try {
        nextSnapshot = await loadBestSnapshot(ctx, latestSnapshot);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        nextSnapshot = latestSnapshot
          ? { ...latestSnapshot, source: "cached", stale: true, error: message }
          : { source: "cached", capturedAtMs: Date.now(), stale: true, error: message };
      } finally {
        if (refreshInFlight === currentRefresh) refreshInFlight = undefined;
        const stillCurrent = !shutdownRequested
          && generation === refreshGeneration
          && activeCtx === ctx
          && modelContextKey(ctx) === requestKey
          && isGlmModel(ctx);
        if (stillCurrent) {
          latestSnapshot = nextSnapshot;
          onUpdate(ctx);
        }
      }
    })();
    refreshInFlight = currentRefresh;
    return currentRefresh;
  }

  function refreshInBackground(ctx: ExtensionContext, options?: { force?: boolean }): void {
    void refresh(ctx, options);
  }

  function startPolling(ctx: ExtensionContext): void {
    activeCtx = ctx;
    stopPolling();
    pollTimer = setInterval(() => {
      if (activeCtx) refreshInBackground(activeCtx);
    }, POLL_INTERVAL_MS);
  }

  function stopPolling(): void {
    if (!pollTimer) return;
    clearInterval(pollTimer);
    pollTimer = undefined;
  }

  function refreshIfDue(ctx: ExtensionContext): void {
    activeCtx = ctx;
    if (!isGlmModel(ctx)) {
      latestSnapshot = undefined;
      onUpdate(ctx);
      return;
    }
    if (Date.now() - lastRefreshStartedAt < MIN_EVENT_REFRESH_MS) {
      onUpdate(ctx);
      return;
    }
    refreshInBackground(ctx);
  }

  pi.on("session_start", async (_event, ctx) => {
    shutdownRequested = false;
    activeCtx = ctx;
    startPolling(ctx);
    refreshInBackground(ctx, { force: true });
  });

  pi.on("model_select", async (_event, ctx) => {
    activeCtx = ctx;
    refreshInBackground(ctx, { force: true });
  });

  pi.on("turn_end", async (_event, ctx) => {
    refreshIfDue(ctx);
  });

  pi.on("session_shutdown", async () => {
    shutdownRequested = true;
    refreshGeneration++;
    stopPolling();
    activeCtx = undefined;
  });
}

export function getGlmQuotaFooterText(_width: number): string | undefined {
  if (!latestSnapshot || (!latestSnapshot.primary && !latestSnapshot.secondary)) return undefined;
  const text = formatGlmQuotaFooterText(latestSnapshot);
  if (!text) return undefined;
  return latestSnapshot.stale || latestSnapshot.source === "cached" ? `${text} (cached)` : text;
}

export function hasActiveGlmQuotaContext(): boolean {
  return Boolean(activeCtx && isGlmModel(activeCtx));
}

async function loadBestSnapshot(ctx: ExtensionContext, previousSnapshot: GlmQuotaSnapshot | undefined): Promise<GlmQuotaSnapshot> {
  try {
    return await fetchLiveSnapshot(ctx);
  } catch (error) {
    if (!previousSnapshot) throw error;
    const message = error instanceof Error ? error.message : String(error);
    return {
      ...previousSnapshot,
      source: "cached",
      stale: Date.now() - previousSnapshot.capturedAtMs > STALE_THRESHOLD_MS,
      error: message,
    };
  }
}

async function fetchLiveSnapshot(ctx: ExtensionContext): Promise<GlmQuotaSnapshot> {
  return fetchLiveSnapshotWithDeps(ctx, { fetchFn: fetch });
}

async function fetchLiveSnapshotWithDeps(ctx: ExtensionContext, deps: GlmFetchDeps): Promise<GlmQuotaSnapshot> {
  if (!isGlmModel(ctx)) throw new Error("Active Pi model is not a GLM/Z.AI model");

  const auth = await resolveGlmAuth(ctx);
  if (!auth?.token) throw new Error("Missing GLM/Z.AI API key. Set GLM_API_KEY, ZAI_API_KEY, ANTHROPIC_AUTH_TOKEN, or configure the active Pi model provider.");

  const urls = quotaURLsForContext(ctx);
  let lastError: Error | undefined;
  for (const url of urls) {
    const response = await deps.fetchFn(url, {
      method: "GET",
      headers: buildGlmRequestHeaders(auth),
      signal: AbortSignal.timeout(15_000),
    });

    if (!response.ok) {
      const body = await safeReadText(response);
      lastError = new Error(`GLM quota request failed (${response.status}): ${truncateInline(body, 200)}`);
      continue;
    }

    const payload = await response.json();
    return mapZaiQuotaPayload(payload as ZaiQuotaPayload, Date.now());
  }

  throw lastError ?? new Error("GLM quota request failed");
}

async function resolveGlmAuth(ctx: ExtensionContext): Promise<GlmResolvedAuth | undefined> {
  const model = ctx.model;
  if (model && ctx.modelRegistry?.getApiKeyAndHeaders) {
    const result = await ctx.modelRegistry.getApiKeyAndHeaders(model);
    if (result.ok) {
      const headers = sanitizeHeaderRecord(result.headers);
      const apiKey = sanitizeSecret(result.apiKey);
      const bearer = extractBearerToken(headers.Authorization ?? headers.authorization);
      const token = apiKey ?? bearer;
      if (token) return { token, headers };
    }
  }

  const fallback = sanitizeSecret(process.env.GLM_API_KEY)
    ?? sanitizeSecret(process.env.ZAI_API_KEY)
    ?? sanitizeSecret(process.env.ANTHROPIC_AUTH_TOKEN);
  return fallback ? { token: fallback, headers: {} } : undefined;
}

function buildGlmRequestHeaders(auth: GlmResolvedAuth): Record<string, string> {
  const headers = { ...auth.headers };
  for (const key of Object.keys(headers)) {
    if (key.toLowerCase() === "authorization") delete headers[key];
  }
  if (!Object.keys(headers).some((key) => key.toLowerCase() === "content-type")) {
    headers["Content-Type"] = "application/json";
  }
  headers.Authorization = `Bearer ${auth.token}`;
  return headers;
}

function quotaURLsForContext(ctx: Pick<ExtensionContext, "model">): string[] {
  const baseUrl = String((ctx.model as Record<string, unknown> | undefined)?.baseUrl ?? "");
  if (isAllowedBigModelBaseURL(baseUrl)) return [DEFAULT_ZAI_QUOTA_CN_URL, DEFAULT_ZAI_QUOTA_INTL_URL];
  return [DEFAULT_ZAI_QUOTA_INTL_URL, DEFAULT_ZAI_QUOTA_CN_URL];
}

function mapZaiQuotaPayload(payload: ZaiQuotaPayload, capturedAtMs: number): GlmQuotaSnapshot {
  const limits = payload.data?.limits?.length ? payload.data.limits : (payload.limits ?? []);
  const tokenLimit = limits.find((limit) => String(limit.type ?? "").toUpperCase() === "TOKENS_LIMIT");
  const primary = buildUsageWindow(tokenLimit ?? limits[0], "5H:");
  const secondarySource = limits.find((limit) => limit !== (tokenLimit ?? limits[0]) && hasUsablePercent(limit) && hasResetTime(limit));
  const secondary = buildUsageWindow(secondarySource, "7D:");

  if (!primary && !secondary) throw new Error("GLM quota response did not contain quota limits");
  return {
    source: "live",
    capturedAtMs,
    stale: false,
    primary,
    secondary,
  };
}

function buildUsageWindow(limit: ZaiQuotaLimit | undefined, label: GlmQuotaWindow["label"]): GlmQuotaWindow | undefined {
  if (!limit) return undefined;
  const usedPercent = resolveUsedPercent(limit);
  if (usedPercent === undefined) return undefined;
  return {
    label,
    usedPercent: clamp(usedPercent, 0, 100),
    resetsAtMs: parseResetTime(limit.nextResetTime ?? limit.resetTime ?? limit.resetsAt),
  };
}

function hasUsablePercent(limit: ZaiQuotaLimit | undefined): boolean {
  return resolveUsedPercent(limit) !== undefined;
}

function hasResetTime(limit: ZaiQuotaLimit | undefined): boolean {
  if (!limit) return false;
  return parseResetTime(limit.nextResetTime ?? limit.resetTime ?? limit.resetsAt) !== undefined;
}

function resolveUsedPercent(limit: ZaiQuotaLimit | undefined): number | undefined {
  if (!limit) return undefined;
  const raw = sanitizeNumber(limit.percentage) ?? sanitizeNumber(limit.used_percent) ?? sanitizeNumber(limit.usedPercentage);
  if (raw === undefined) return undefined;
  // Z.AI API `percentage` = remaining percentage (100 = nothing used, 0 = fully consumed)
  const remaining = raw > 1 ? raw : raw * 100;
  return clamp(100 - remaining, 0, 100);
}

function isGlmModel(ctx: Pick<ExtensionContext, "hasUI" | "model">): boolean {
  if (!ctx.hasUI) return false;
  const model = ctx.model as Record<string, unknown> | undefined;
  if (!model) return false;

  const baseUrl = String(model.baseUrl ?? "");
  if (baseUrl.trim() !== "") return isAllowedZaiBaseURL(baseUrl) || isAllowedBigModelBaseURL(baseUrl);

  const provider = String(model.provider ?? "").toLowerCase();
  const id = String(model.id ?? "").toLowerCase();
  const name = String(model.name ?? "").toLowerCase();
  const displayName = String(model.displayName ?? model.display_name ?? "").toLowerCase();
  return [provider, id, name, displayName].some((value) => value === "glm" || value === "zai" || value === "z.ai" || value === "zhipu" || value.includes("glm-"));
}

function isAllowedZaiBaseURL(rawURL: string): boolean {
  const url = parseURL(rawURL);
  if (!url) return false;
  return hostMatchesDomain(url.hostname.toLowerCase(), "z.ai");
}

function isAllowedBigModelBaseURL(rawURL: string): boolean {
  const url = parseURL(rawURL);
  if (!url) return false;
  return hostMatchesDomain(url.hostname.toLowerCase(), "bigmodel.cn");
}

function parseURL(rawURL: string): URL | undefined {
  try {
    return new URL(rawURL);
  } catch {
    return undefined;
  }
}

function hostMatchesDomain(hostname: string, domain: string): boolean {
  return hostname === domain || hostname.endsWith(`.${domain}`);
}

function modelContextKey(ctx: ExtensionContext): string {
  const model = ctx.model as Record<string, unknown> | undefined;
  return `${String(model?.provider ?? "none")}:${String(model?.id ?? "none")}:${String(model?.baseUrl ?? "")}`;
}

function formatGlmQuotaFooterText(snapshot: GlmQuotaSnapshot): string | undefined {
  const windows = [snapshot.primary, snapshot.secondary]
    .filter((window): window is GlmQuotaWindow => Boolean(window))
    .map((window) => formatNativeWindow(window));
  if (windows.length === 0) return undefined;
  return windows.join(" │ ");
}

function formatNativeWindow(window: GlmQuotaWindow): string {
  const pct = Math.round(clamp(window.usedPercent, 0, 100));
  const resetText = window.resetsAtMs ? formatResetDuration(window.resetsAtMs) : "";
  const base = `${window.label} ${batteryIcon(pct)} ${renderNativeBar(pct, 10)} ${pct}%`;
  return resetText ? `${base} (${resetText})` : base;
}

function renderNativeBar(usedPercent: number, width: number): string {
  const filled = Math.max(0, Math.min(width, Math.round((clamp(usedPercent, 0, 100) / 100) * width)));
  return `${"█".repeat(filled)}${"░".repeat(width - filled)}`;
}

function batteryIcon(usedPercent: number): string {
  return usedPercent > 70 ? "🪫" : "🔋";
}

function formatResetDuration(timestampMs: number, nowMs = Date.now()): string {
  const remaining = timestampMs - nowMs;
  if (remaining <= 0) return "";
  const totalMinutes = Math.max(1, Math.round(remaining / 60_000));
  const days = Math.floor(totalMinutes / 1_440);
  const hours = Math.floor((totalMinutes % 1_440) / 60);
  const minutes = totalMinutes % 60;
  if (days > 0) return `${days}d ${hours}h ${minutes}m`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function parseResetTime(value: number | string | undefined): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value > 1_000_000_000_000 ? value : value * 1000;
  if (typeof value === "string" && value.trim() !== "") {
    const numeric = Number(value);
    if (Number.isFinite(numeric)) return numeric > 1_000_000_000_000 ? numeric : numeric * 1000;
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function sanitizeHeaderRecord(value: unknown): Record<string, string> {
  if (!value || typeof value !== "object") return {};
  const headers: Record<string, string> = {};
  for (const [key, raw] of Object.entries(value as Record<string, unknown>)) {
    if (typeof raw === "string") headers[key] = raw;
  }
  return headers;
}

function sanitizeNumber(value: number | string | undefined): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function sanitizeSecret(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

function extractBearerToken(value: string | undefined): string | undefined {
  const match = value?.match(/^Bearer\s+(.+)$/i);
  return sanitizeSecret(match?.[1]);
}

function truncateInline(value: string, limit: number): string {
  const normalized = value.replace(/\s+/g, " ").trim();
  return normalized.length <= limit ? normalized : `${normalized.slice(0, limit - 1)}…`;
}

async function safeReadText(response: Response): Promise<string> {
  try {
    return await response.text();
  } catch {
    return "";
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export function isGlmModelForTest(ctx: Pick<ExtensionContext, "hasUI" | "model">): boolean {
  return isGlmModel(ctx);
}

export function mapZaiQuotaPayloadForTest(payload: ZaiQuotaPayload, capturedAtMs: number): GlmQuotaSnapshot {
  return mapZaiQuotaPayload(payload, capturedAtMs);
}

export function formatGlmQuotaFooterTextForTest(snapshot: GlmQuotaSnapshot): string | undefined {
  return formatGlmQuotaFooterText(snapshot);
}

export async function fetchLiveSnapshotForTest(ctx: ExtensionContext, fetchFn: typeof fetch): Promise<GlmQuotaSnapshot> {
  return fetchLiveSnapshotWithDeps(ctx, { fetchFn });
}

export function glmQuotaURLsForTest(ctx: Pick<ExtensionContext, "model">): string[] {
  return quotaURLsForContext(ctx);
}
