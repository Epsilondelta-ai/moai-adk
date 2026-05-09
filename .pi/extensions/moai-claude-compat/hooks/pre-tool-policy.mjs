#!/usr/bin/env node

const payload = JSON.parse(await readStdin())
const toolName = normalizeToolName(String(payload.tool_name ?? payload.toolName ?? ""))
const toolArgs = isRecord(payload.tool_args) ? payload.tool_args : isRecord(payload.input) ? payload.input : {}

if (["read", "write", "edit", "grep", "glob"].includes(toolName)) {
  const filePath = getToolPath(toolArgs)
  if (isSensitivePath(filePath)) deny(`MoAI pi guard denied ${toolName} to sensitive path: ${filePath}`)
  if (toolName === "read" && isEnvPath(filePath)) ask(`MoAI pi guard requires approval for reading env file: ${filePath}`)
}

if (toolName === "bash") {
  const command = String(toolArgs.command ?? "")
  const compact = command.replace(/\s+/g, " ").trim()
  const denyReason = findDenyBashReason(compact)
  if (denyReason) deny(denyReason)
  const askReason = findAskBashReason(compact)
  if (askReason) ask(askReason)
}

function normalizeToolName(name) {
  const lower = name.toLowerCase()
  if (lower === "multi_edit" || lower === "multiedit") return "edit"
  return lower
}

function getToolPath(toolArgs) {
  return String(
    toolArgs.path
      ?? toolArgs.filePath
      ?? toolArgs.file_path
      ?? toolArgs.file
      ?? toolArgs.pattern
      ?? ""
  )
}

function findDenyBashReason(command) {
  const dangerous = [
    [/(^|\s)git\s+push\s+(-f|--force|--force-with-lease)(\s|$)/, "Denied force git push"],
    [/(^|\s)git\s+reset\s+--hard(\s|$)/, "Denied git reset --hard"],
    [/(^|\s)git\s+clean\s+-[^\n;]*f[^\n;]*d/, "Denied destructive git clean"],
    [/(^|\s)git\s+rebase\s+-i(\s|$)/, "Denied interactive git rebase"],
    [/(^|\s)rm\s+-[^\n;]*r[^\n;]*f\s+(\/|\/\*|\$HOME|~|~\/)(\s|$)/, "Denied broad rm -rf target"],
    [/(^|\s)(del\s+\/S\s+\/Q|rmdir\s+\/S\s+\/Q|Remove-Item\s+-Recurse\s+-Force)\s+C[:\\/]/i, "Denied destructive Windows filesystem command"],
    [/(^|\s)(Clear-Disk|Format-Volume|format|dd|mkfs|fdisk|reboot|shutdown|init|systemctl|killall)(\s|$)/i, "Denied destructive system command"],
    [/(^|\s)kill\s+-9(\s|$)/, "Denied kill -9"],
    [/(^|\s)chmod\s+(-R\s+)?777(\s|$)/, "Denied chmod 777"],
    [/(^|\s)(DROP\s+DATABASE|DROP\s+TABLE|TRUNCATE|DELETE\s+FROM)(\s|$)/i, "Denied destructive database statement"],
    [/(^|\s)(mongo|mongosh)(\s|$)/, "Denied Mongo shell command"],
    [/(^|\s)redis-cli\s+(FLUSHALL|FLUSHDB)(\s|$)/i, "Denied Redis destructive command"],
    [/(^|\s)psql\s+[^\n;]*-c\s+['\"]?\s*DROP\b/i, "Denied psql DROP command"],
    [/(^|\s)mysql\s+[^\n;]*-e\s+['\"]?\s*DROP\b/i, "Denied mysql DROP command"],
    [/(curl|wget)[^|;&]*\|\s*(sh|bash)(\s|$)/, "Denied pipe-to-shell command"],
  ]
  return dangerous.find(([pattern]) => pattern.test(command))?.[1]
}

function findAskBashReason(command) {
  const ask = [
    [/(^|\s)rm(\s|$)/, "Approval required for rm"],
    [/(^|\s)sudo(\s|$)/, "Approval required for sudo"],
    [/(^|\s)chmod(\s|$)/, "Approval required for chmod"],
    [/(^|\s)chown(\s|$)/, "Approval required for chown"],
  ]
  return ask.find(([pattern]) => pattern.test(command))?.[1]
}

function isSensitivePath(filePath) {
  const normalized = normalizePath(filePath)
  return normalized === "secrets"
    || normalized.startsWith("secrets/")
    || normalized.includes("/secrets/")
    || normalized.startsWith("~/.ssh/")
    || normalized.startsWith("~/.aws/")
    || normalized.startsWith("~/.config/gcloud/")
    || normalized.startsWith(".ssh/")
    || normalized.startsWith(".aws/")
    || normalized.startsWith(".config/gcloud/")
    || normalized.includes("/.ssh/")
    || normalized.includes("/.aws/")
    || normalized.includes("/.config/gcloud/")
}

function isEnvPath(filePath) {
  const normalized = normalizePath(filePath)
  const base = normalized.split("/").pop() ?? ""
  return base === ".env" || base.startsWith(".env.")
}

function normalizePath(filePath) {
  return String(filePath ?? "").replaceAll("\\", "/").replace(/^\.\//, "")
}

function deny(message) {
  console.error(message)
  process.exit(2)
}

function ask(message) {
  // pi-yaml-hooks headless/non-interactive policy: request-like guardrails fail closed.
  console.error(message)
  process.exit(2)
}

function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

async function readStdin() {
  const chunks = []
  for await (const chunk of process.stdin) chunks.push(chunk)
  return Buffer.concat(chunks).toString("utf8")
}
