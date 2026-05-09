import { QUOTA_FOOTER_PRIORITY, TEAM_BACKEND_PRIORITY } from "./constants.ts";

export interface PackageConflictFinding {
  level: "ok" | "warn";
  message: string;
}

function normalizePackageName(spec: string): string {
  let s = spec.replace(/^npm:/, "").replace(/^git:/, "");
  s = s.split("#")[0].split("?")[0];
  if (s.startsWith("@")) {
    const parts = s.split("@");
    return parts.length > 2 ? `@${parts[1]}` : s;
  }
  return s.split("@")[0];
}

export function normalizePackageSpecs(specs: string[] = []): string[] {
  return specs.map(normalizePackageName).filter(Boolean);
}

export function analyzePackageConflicts(specs: string[] = []): PackageConflictFinding[] {
  const names = normalizePackageSpecs(specs);
  const findings: PackageConflictFinding[] = [];
  const has = (name: string) => names.includes(name);

  const team = TEAM_BACKEND_PRIORITY.filter(has);
  if (team.length === 0) findings.push({ level: "warn", message: "Agent Teams backend not active; will use schema/fallback planning only" });
  else if (team.length > 1) findings.push({ level: "warn", message: `Multiple Agent Teams backends active: ${team.join(", ")}` });
  else findings.push({ level: "ok", message: `Agent Teams backend candidate active: ${team[0]}` });

  const quota = QUOTA_FOOTER_PRIORITY.filter(has);
  if (quota.length === 0) findings.push({ level: "warn", message: "Codex/GPT quota footer package not active" });
  else if (quota.length > 1) findings.push({ level: "warn", message: `Multiple quota footer packages active: ${quota.join(", ")}` });
  else findings.push({ level: "ok", message: `Quota footer package active: ${quota[0]}` });

  if (has("pi-yaml-hooks") && has("@aliou/pi-guardrails")) {
    findings.push({ level: "warn", message: "pi-yaml-hooks and @aliou/pi-guardrails may both confirm/block dangerous commands; verify ordering" });
  }
  if (has("pi-yaml-hooks")) {
    findings.push({ level: "warn", message: "pi-yaml-hooks@2026.5.8 has known peer dependency risk; install in isolation first" });
  }
  if (specs.length === 0) {
    findings.push({ level: "ok", message: "Active packages empty by design for skeleton mode" });
  }

  return findings;
}

export function formatFindings(findings: PackageConflictFinding[]): string[] {
  return findings.map((f) => `${f.level === "ok" ? "ok" : "warn(non-blocking)"}: ${f.message}`);
}
