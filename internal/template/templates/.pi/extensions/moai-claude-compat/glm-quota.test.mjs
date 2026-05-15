import assert from 'node:assert/strict';
import {
  fetchLiveSnapshotForTest,
  formatGlmQuotaFooterTextForTest,
  glmQuotaURLsForTest,
  isGlmModelForTest,
  mapZaiQuotaPayloadForTest,
} from './src/glm-quota.ts';

const glmById = {
  hasUI: true,
  model: {
    provider: 'anthropic',
    id: 'glm-5.1',
  },
};
assert.equal(isGlmModelForTest(glmById), true, 'glm-5.1 model id should be detected');

const glmByProvider = {
  hasUI: true,
  model: {
    provider: 'z.ai',
    id: 'glm-5.1',
  },
};
assert.equal(isGlmModelForTest(glmByProvider), true, 'z.ai provider should be detected');

const glmByBaseUrl = {
  hasUI: true,
  model: {
    provider: 'custom-anthropic',
    id: 'glm-5.1',
    baseUrl: 'https://api.z.ai/api/anthropic',
  },
};
assert.equal(isGlmModelForTest(glmByBaseUrl), true, 'api.z.ai baseUrl should be detected');

const cnGlmByBaseUrl = {
  hasUI: true,
  model: {
    provider: 'custom-anthropic',
    id: 'glm-5.1',
    baseUrl: 'https://open.bigmodel.cn/api/anthropic',
  },
};
assert.equal(isGlmModelForTest(cnGlmByBaseUrl), true, 'open.bigmodel.cn baseUrl should be detected');
assert.equal(glmQuotaURLsForTest(cnGlmByBaseUrl)[0], 'https://open.bigmodel.cn/api/monitor/usage/quota/limit', 'CN baseUrl should try CN quota endpoint first');

const maliciousLookalike = {
  hasUI: true,
  model: {
    provider: 'custom-anthropic',
    id: 'glm-5.1',
    baseUrl: 'https://api.z.ai.evil.example/api/anthropic',
  },
};
assert.equal(isGlmModelForTest(maliciousLookalike), false, 'lookalike z.ai host must not be detected');

const noUi = {
  hasUI: false,
  model: {
    provider: 'z.ai',
    id: 'glm-5.1',
  },
};
assert.equal(isGlmModelForTest(noUi), false, 'quota footer should only activate with UI');

const reset = Date.UTC(2026, 0, 1, 5, 0, 0);
const payload = {
  data: {
    level: 'pro',
    limits: [
      {
        type: 'TOKENS_LIMIT',
        percentage: 5,
        nextResetTime: reset,
      },
      {
        type: 'TIME_LIMIT',
        percentage: 1,
      },
    ],
  },
};
const snapshot = mapZaiQuotaPayloadForTest(payload, Date.UTC(2026, 0, 1));
assert.equal(snapshot.primary?.label, '5H:', 'TOKENS_LIMIT should map to native 5H window');
assert.equal(snapshot.primary?.usedPercent, 5, 'TOKENS_LIMIT percentage should map directly to 5H used percent');
assert.equal(snapshot.primary?.resetsAtMs, reset, 'nextResetTime ms should parse');
assert.equal(snapshot.secondary?.label, '7D:', 'non-token quota should map to native 7D window');
assert.equal(snapshot.secondary?.usedPercent, 1, 'secondary quota percentage should map directly to 7D used percent even without reset time');

const footer = formatGlmQuotaFooterTextForTest(snapshot) ?? '';
assert(footer.includes('5H: 🔋'), 'GLM quota footer should use native 5H segment style');
assert(footer.includes('5%'), 'GLM quota footer should show direct 5H used percentage');
assert(footer.includes('7D: 🔋'), 'GLM quota footer should show native 7D segment when secondary quota exists');
assert(footer.includes('1%'), 'GLM quota footer should show direct secondary used percentage');
assert(!footer.includes('GLM:'), 'GLM quota footer should not add provider prefix inside native bar line');

let requestedUrl = '';
let requestedAuthorization = '';
const fetchCtx = {
  hasUI: true,
  model: {
    provider: 'anthropic',
    id: 'glm-5.1',
  },
  modelRegistry: {
    async getApiKeyAndHeaders() {
      return { ok: true, apiKey: 'registry-glm-token', headers: { Authorization: 'Bearer stale-token' } };
    },
  },
};
const fetchedSnapshot = await fetchLiveSnapshotForTest(fetchCtx, async (url, init) => {
  requestedUrl = String(url);
  requestedAuthorization = String(init?.headers?.Authorization ?? '');
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
});
assert.equal(requestedUrl, 'https://api.z.ai/api/monitor/usage/quota/limit', 'GLM quota fetch should use official Z.AI endpoint');
assert.equal(requestedAuthorization, 'Bearer registry-glm-token', 'model registry apiKey should win over stale Authorization header');
assert.equal(fetchedSnapshot.primary?.usedPercent, 5, 'fetch should parse Z.AI used percentage directly');
assert.equal(fetchedSnapshot.secondary?.usedPercent, 1, 'fetch should parse secondary quota percentage directly');

const oldGlmApiKey = process.env.GLM_API_KEY;
const oldZaiApiKey = process.env.ZAI_API_KEY;
const oldAnthropicAuthToken = process.env.ANTHROPIC_AUTH_TOKEN;
delete process.env.GLM_API_KEY;
delete process.env.ZAI_API_KEY;
process.env.ANTHROPIC_AUTH_TOKEN = 'anthropic-glm-token';
let fallbackAuthorization = '';
const fallbackCtx = {
  hasUI: true,
  model: {
    provider: 'anthropic',
    id: 'glm-5.1',
  },
  modelRegistry: {
    async getApiKeyAndHeaders() {
      return { ok: false, error: 'no provider key' };
    },
  },
};
await fetchLiveSnapshotForTest(fallbackCtx, async (_url, init) => {
  fallbackAuthorization = String(init?.headers?.Authorization ?? '');
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
});
if (oldGlmApiKey === undefined) delete process.env.GLM_API_KEY;
else process.env.GLM_API_KEY = oldGlmApiKey;
if (oldZaiApiKey === undefined) delete process.env.ZAI_API_KEY;
else process.env.ZAI_API_KEY = oldZaiApiKey;
if (oldAnthropicAuthToken === undefined) delete process.env.ANTHROPIC_AUTH_TOKEN;
else process.env.ANTHROPIC_AUTH_TOKEN = oldAnthropicAuthToken;
assert.equal(fallbackAuthorization, 'Bearer anthropic-glm-token', 'ANTHROPIC_AUTH_TOKEN fallback should support moai glm env');

const littleUsed = {
  data: {
    level: 'pro',
    limits: [
      {
        type: 'TOKENS_LIMIT',
        percentage: 5,
        nextResetTime: Date.now() + 5 * 3600 * 1000,
      },
      {
        type: 'TIME_LIMIT',
        percentage: 1,
      },
    ],
  },
};
const littleUsedSnap = mapZaiQuotaPayloadForTest(littleUsed, Date.now());
assert.equal(littleUsedSnap.primary?.usedPercent, 5, '5H percentage=5 → 5% used');
assert.equal(littleUsedSnap.secondary?.usedPercent, 1, 'secondary percentage=1 → 1% used');

console.log('glm quota regression ok');
