import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import {
  LANGUAGE_CONFIG_PATH,
  OUTPUT_STYLES_CONFIG_PATH,
  QUALITY_CONFIG_PATH,
  SOURCE_MAP_PATH,
  WORKFLOW_CONFIG_PATH,
} from "./constants.ts";

export interface MoaiCompatConfig {
  conversationLanguage: string;
  codeCommentsLanguage: string;
  qualityMode: string;
  sourceMapPath: string;
  workflowConfigPath: string;
  outputStyle: OutputStyleConfig;
}

export interface OutputStyleConfig {
  name: string;
  sourcePath: string;
  instruction: string;
  loaded: boolean;
  sanitized: boolean;
  enforcement: "prompt-level";
  error?: string;
}

function readIfExists(path: string): string {
  const abs = resolve(process.cwd(), path);
  return existsSync(abs) ? readFileSync(abs, "utf8") : "";
}

function pathExists(path: string): boolean {
  return existsSync(resolve(process.cwd(), path));
}

function findYamlScalar(text: string, keys: string[], fallback: string): string {
  for (const key of keys) {
    const match = text.match(new RegExp(`^\\s*${key}\\s*:\\s*[\"']?([^\"'\\n#]+)`, "m"));
    if (match?.[1]) return match[1].trim();
  }
  return fallback;
}

export function loadMoaiCompatConfig(): MoaiCompatConfig {
  const languageYaml = readIfExists(LANGUAGE_CONFIG_PATH);
  const qualityYaml = readIfExists(QUALITY_CONFIG_PATH);

  return {
    conversationLanguage: findYamlScalar(languageYaml, ["conversation_language", "user_responses", "user"], "ko"),
    codeCommentsLanguage: findYamlScalar(languageYaml, ["code_comments", "comments"], "ko"),
    qualityMode: findYamlScalar(qualityYaml, ["mode", "development_mode"], "tdd"),
    sourceMapPath: SOURCE_MAP_PATH,
    workflowConfigPath: WORKFLOW_CONFIG_PATH,
    outputStyle: loadOutputStyleConfig(),
  };
}

function loadOutputStyleConfig(path = OUTPUT_STYLES_CONFIG_PATH): OutputStyleConfig {
  const fallback: OutputStyleConfig = {
    name: "moai",
    sourcePath: "",
    instruction: "MoAI output style: use concise Markdown, user's conversation language, no user-facing XML tags.",
    loaded: false,
    sanitized: true,
    enforcement: "prompt-level",
    error: "output style config not loaded",
  };

  try {
    if (!pathExists(path)) return { ...fallback, error: `missing ${path}` };
    const parsed = JSON.parse(readIfExists(path)) as {
      default?: unknown;
      styles?: Record<string, { source?: unknown }>;
    };
    const name = typeof parsed.default === "string" ? parsed.default : "moai";
    const source = parsed.styles?.[name]?.source;
    if (typeof source !== "string" || !source.trim()) {
      return { ...fallback, name, error: `missing source for style ${name}` };
    }

    const sourcePath = resolveOutputStyleSource(path, source);
    if (!pathExists(sourcePath)) {
      return { ...fallback, name, sourcePath, error: `missing output style source ${sourcePath}` };
    }

    const raw = readIfExists(sourcePath);
    const sanitized = sanitizeOutputStyleMarkdown(raw);
    return {
      name,
      sourcePath,
      instruction: buildOutputStyleInstruction(name, sanitized),
      loaded: true,
      sanitized: raw !== sanitized,
      enforcement: "prompt-level",
    };
  } catch (error) {
    return { ...fallback, error: error instanceof Error ? error.message : String(error) };
  }
}

function resolveOutputStyleSource(configPath: string, source: string): string {
  const normalized = source.replace(/^\.\//, "");
  const piRelative = resolve(process.cwd(), ".pi", normalized);
  if (existsSync(piRelative)) return `.pi/${normalized}`;

  const configRelative = resolve(process.cwd(), dirname(configPath), source);
  if (existsSync(configRelative)) return resolve(process.cwd(), dirname(configPath), source);

  return `.pi/${normalized}`;
}

function sanitizeOutputStyleMarkdown(markdown: string): string {
  return markdown
    .replace(/<moai>\s*(DONE|COMPLETE)\s*<\/moai>/gi, "MoAI DONE")
    .replace(/<\/?[A-Za-z][^>\n]*>/g, (tag) => `\`${tag}\``)
    .replace(/\bXML tags are reserved for internal agent-to-agent data transfer only\./g, "XML-like tags are reserved for internal data transfer only; never show them in user-facing output.");
}

function buildOutputStyleInstruction(name: string, markdown: string): string {
  const body = stripFrontmatter(markdown).trim();
  return [
    `MoAI output style '${name}' is active at prompt level (not hard post-generation enforcement).`,
    "Follow the style guidance below when responding to users. If any style example conflicts with Pi policy, Pi policy wins: Markdown only, no user-facing XML tags.",
    body,
  ].join("\n\n");
}

function stripFrontmatter(text: string): string {
  return text.replace(/^---\n[\s\S]*?\n---\n?/, "");
}

export function outputStyleStatus(config: MoaiCompatConfig): string[] {
  return [
    config.outputStyle.loaded
      ? `ok: output style '${config.outputStyle.name}' loaded from ${config.outputStyle.sourcePath}`
      : `missing: output style '${config.outputStyle.name}' not loaded${config.outputStyle.error ? ` (${config.outputStyle.error})` : ""}`,
    config.outputStyle.sanitized
      ? "ok: output style XML-like user-facing markers sanitized"
      : "info: output style required no XML marker sanitization",
    `partial: output style enforcement is ${config.outputStyle.enforcement}; no hard post-generation output filter is installed`,
  ];
}

export function buildCoreInstruction(config: MoaiCompatConfig): string {
  return [
    "MoAI pi compatibility layer is active.",
    `User-facing responses must use conversation_language=${config.conversationLanguage}.`,
    "Use Markdown for user-facing output and do not display XML tags.",
    "Use pi-local source snapshots under .pi/generated/source/**.",
    "Prefer pi packages over custom implementation. Use moai-claude-compat only for schema conversion, glue, and MoAI-specific policy enforcement.",
    "Claude permissionMode is intentionally excluded from parity. Preserve it as metadata only; enforce allow/ask/deny guardrails separately.",
    "Subagents must not ask users directly; escalate decisions to the parent session.",
  ].join("\n");
}
