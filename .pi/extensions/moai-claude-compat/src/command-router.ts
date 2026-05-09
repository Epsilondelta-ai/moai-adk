import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import type { ExtensionAPI, ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { buildAuditReport, buildDoctorReport } from "./doctor.ts";
import { inferPhaseFromCommand, setMoaiWidget, updateMoaiStatus } from "./statusline.ts";
import type { MoaiCompatConfig } from "./config.ts";
import { writeConvertedAgents } from "./agent-converter.ts";
const PI_PROMPTS_PATH = ".pi/prompts";

const MOAI_SUBCOMMANDS = [
  "plan",
  "run",
  "sync",
  "project",
  "fix",
  "loop",
  "mx",
  "review",
  "clean",
  "codemaps",
  "coverage",
  "e2e",
  "feedback",
  "context",
  "gate",
  "security",
] as const;

const MOAI_SUBCOMMAND_ALIASES: Record<string, MoaiSubcommand> = {
  spec: "plan",
  impl: "run",
  docs: "sync",
  pr: "sync",
  fb: "feedback",
  bug: "feedback",
  issue: "feedback",
  "code-review": "review",
  "dead-code": "clean",
  cov: "coverage",
  "e2e-test": "e2e",
  ctx: "context",
  memory: "context",
  check: "gate",
  "pre-commit": "gate",
  audit: "security",
  sec: "security",
};

const SPEC_ID_PATTERN = /SPEC-[A-Z0-9]+(?:-[A-Z0-9]+)*/i;

type MoaiSubcommand = typeof MOAI_SUBCOMMANDS[number];

const MOAI_COMMAND_PROMPT_BY_SUBCOMMAND: Partial<Record<MoaiSubcommand, string>> = {
  plan: `${PI_PROMPTS_PATH}/moai-plan.md`,
  run: `${PI_PROMPTS_PATH}/moai-run.md`,
  sync: `${PI_PROMPTS_PATH}/moai-sync.md`,
  project: `${PI_PROMPTS_PATH}/moai-project.md`,
  fix: `${PI_PROMPTS_PATH}/moai-fix.md`,
  loop: `${PI_PROMPTS_PATH}/moai-loop.md`,
  mx: `${PI_PROMPTS_PATH}/moai-mx.md`,
  review: `${PI_PROMPTS_PATH}/moai-review.md`,
  clean: `${PI_PROMPTS_PATH}/moai-clean.md`,
  codemaps: `${PI_PROMPTS_PATH}/moai-codemaps.md`,
  coverage: `${PI_PROMPTS_PATH}/moai-coverage.md`,
  e2e: `${PI_PROMPTS_PATH}/moai-e2e.md`,
  feedback: `${PI_PROMPTS_PATH}/moai-feedback.md`,
  context: `${PI_PROMPTS_PATH}/moai-context.md`,
  gate: `${PI_PROMPTS_PATH}/moai-gate.md`,
  security: `${PI_PROMPTS_PATH}/moai-security.md`,
};

function normalizeSubcommand(value: string): MoaiSubcommand | undefined {
  if ((MOAI_SUBCOMMANDS as readonly string[]).includes(value)) return value as MoaiSubcommand;
  return MOAI_SUBCOMMAND_ALIASES[value];
}

function stripFrontmatter(text: string): string {
  return text.replace(/^---\n[\s\S]*?\n---\n?/, "").trim();
}

function readPromptBody(path: string): string | undefined {
  const abs = resolve(process.cwd(), path);
  if (!existsSync(abs)) return undefined;
  return stripFrontmatter(readFileSync(abs, "utf8"));
}

function applyArguments(prompt: string, args: string): string {
  return prompt
    .replaceAll("$ARGUMENTS", args)
    .replaceAll("$@", args)
    .trim();
}

function buildMoaiSkillInvocation(args: string): string {
  return `Use Skill("moai") with arguments: ${args}`.trim();
}

function buildMoaiPrompt(subcommand: string, args: string): string {
  const normalized = subcommand || "auto";
  const skillArgs = normalized === "auto" ? args.trim() : `${normalized}${args ? ` ${args}` : ""}`.trim();
  const promptPath = normalized === "auto" ? undefined : MOAI_COMMAND_PROMPT_BY_SUBCOMMAND[normalized as MoaiSubcommand];
  const promptBody = promptPath ? readPromptBody(promptPath) : undefined;
  return applyArguments(promptBody ?? buildMoaiSkillInvocation(skillArgs), args);
}

function buildCommandPrompt(path: string, args: string): string {
  return applyArguments(readPromptBody(path) ?? `Arguments provided: $ARGUMENTS`, args);
}

async function dispatchMoai(pi: ExtensionAPI, args: string, ctx: ExtensionCommandContext, config: MoaiCompatConfig) {
  const [first = "", ...rest] = args.trim().split(/\s+/).filter(Boolean);
  const canonical = normalizeSubcommand(first);
  const subcommand = canonical ?? "auto";
  const remaining = canonical ? rest.join(" ") : args.trim();
  updateMoaiStatus(ctx, config, { phase: inferPhaseFromCommand(subcommand), specId: remaining.match(SPEC_ID_PATTERN)?.[0] });
  pi.sendUserMessage(buildMoaiPrompt(subcommand, remaining));
}

export function registerCommands(pi: ExtensionAPI, config: MoaiCompatConfig) {
  pi.registerCommand("moai", {
    description: "Route to the MoAI Claude harness through pi compatibility layer",
    getArgumentCompletions: (prefix) => {
      const candidates = [...MOAI_SUBCOMMANDS, ...Object.keys(MOAI_SUBCOMMAND_ALIASES)].sort();
      const matches = candidates.filter((s) => s.startsWith(prefix));
      return matches.length ? matches.map((value) => ({ value, label: value })) : null;
    },
    handler: async (args, ctx) => dispatchMoai(pi, args, ctx, config),
  });

  for (const subcommand of MOAI_SUBCOMMANDS) {
    pi.registerCommand(`moai-${subcommand}`, {
      description: `Shortcut for /moai ${subcommand}`,
      handler: async (args, ctx) => dispatchMoai(pi, `${subcommand} ${args}`.trim(), ctx, config),
    });
  }

  pi.registerCommand("github", {
    description: "Run MoAI GitHub workflow from pi-local command snapshot",
    handler: async (args, ctx) => {
      updateMoaiStatus(ctx, config, { phase: "review" });
      pi.sendUserMessage(buildCommandPrompt(`${PI_PROMPTS_PATH}/github.md`, args));
    },
  });

  pi.registerCommand("release", {
    description: "Run MoAI release workflow from pi-local command snapshot",
    handler: async (args, ctx) => {
      updateMoaiStatus(ctx, config, { phase: "gate" });
      pi.sendUserMessage(buildCommandPrompt(`${PI_PROMPTS_PATH}/release.md`, args));
    },
  });

  pi.registerCommand("moai-pi-doctor", {
    description: "Check MoAI pi overlay/package status",
    handler: async (_args, ctx) => {
      const lines = buildDoctorReport();
      setMoaiWidget(ctx, lines);
      ctx.ui.notify(lines.join("\n"), "info");
    },
  });

  pi.registerCommand("moai-pi-audit", {
    description: "Show MoAI pi parity audit checklist",
    handler: async (_args, ctx) => {
      const lines = buildAuditReport();
      setMoaiWidget(ctx, lines);
      ctx.ui.notify(lines.join("\n"), "info");
    },
  });

  pi.registerCommand("moai-pi-generate-agents", {
    description: "Generate normalized pi-side agent metadata from pi-local agent snapshot without modifying upstream sources",
    handler: async (_args, ctx) => {
      const agents = writeConvertedAgents();
      const lines = [
        "MoAI pi agent generation",
        `generated: ${agents.length} files under .pi/generated/agents`,
        "permissionMode: preserved as metadata-only",
      ];
      setMoaiWidget(ctx, lines);
      ctx.ui.notify(lines.join("\n"), "info");
    },
  });

  pi.registerCommand("moai-pi-validate", {
    description: "Run MoAI pi validation report",
    handler: async (_args, ctx) => {
      const lines = ["MoAI pi validation", ...buildAuditReport().slice(1)];
      setMoaiWidget(ctx, lines);
      ctx.ui.notify(lines.join("\n"), "info");
    },
  });
}
