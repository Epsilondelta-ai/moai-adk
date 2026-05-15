import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { getCodexQuotaFooterText, hasActiveCodexQuotaContext, registerCodexQuota } from "./codex-quota.ts";
import { getGlmQuotaFooterText, hasActiveGlmQuotaContext, registerGlmQuota } from "./glm-quota.ts";
import { getKimiQuotaFooterText, hasActiveKimiQuotaContext, registerKimiQuota } from "./kimi-quota.ts";

export function registerQuotaFooter(pi: ExtensionAPI, onUpdate: (ctx: ExtensionContext) => void): void {
  registerCodexQuota(pi, onUpdate);
  registerGlmQuota(pi, onUpdate);
  registerKimiQuota(pi, onUpdate);
}

export function getQuotaFooterText(_ctx: ExtensionContext, width: number): string | undefined {
  if (hasActiveCodexQuotaContext()) return getCodexQuotaFooterText(width);
  if (hasActiveGlmQuotaContext()) return getGlmQuotaFooterText(width);
  if (hasActiveKimiQuotaContext()) return getKimiQuotaFooterText(width);
  return undefined;
}
