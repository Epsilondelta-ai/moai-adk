#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

bridge() {
  local json="$1"
  (cd "$ROOT" && go run ./cmd/moai pi bridge <<<"$json")
}

printf '== MoAI Pi smoke: doctor ==\n'
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"doctor\",\"cwd\":\"$TMP\"}" | python3 -m json.tool >/dev/null

printf '== MoAI Pi smoke: plan/run/sync ==\n'
PLAN=$(bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"command\",\"cwd\":\"$TMP\",\"payload\":{\"command\":\"plan\",\"args\":\"small smoke feature\"}}")
SPEC=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["commandResult"]["data"]["specId"])' <<<"$PLAN")
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"command\",\"cwd\":\"$TMP\",\"payload\":{\"command\":\"run\",\"args\":\"$SPEC\"}}" | python3 -m json.tool >/dev/null
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"command\",\"cwd\":\"$TMP\",\"payload\":{\"command\":\"sync\",\"args\":\"$SPEC\"}}" | python3 -m json.tool >/dev/null

printf '== MoAI Pi smoke: task tools ==\n'
TASK=$(bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"tool\",\"cwd\":\"$TMP\",\"payload\":{\"tool\":\"moai_task_create\",\"payload\":{\"subject\":\"smoke\",\"description\":\"verify task store\"}}}")
TASK_ID=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["task"]["id"])' <<<"$TASK")
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"tool\",\"cwd\":\"$TMP\",\"payload\":{\"tool\":\"moai_task_list\"}}" | python3 -m json.tool >/dev/null
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"tool\",\"cwd\":\"$TMP\",\"payload\":{\"tool\":\"moai_task_get\",\"payload\":{\"id\":\"$TASK_ID\"}}}" | python3 -m json.tool >/dev/null
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"tool\",\"cwd\":\"$TMP\",\"payload\":{\"tool\":\"moai_task_update\",\"payload\":{\"id\":\"$TASK_ID\",\"status\":\"completed\"}}}" | python3 -m json.tool >/dev/null

printf '== MoAI Pi smoke: gates ==\n'
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"tool\",\"cwd\":\"$TMP\",\"payload\":{\"tool\":\"moai_quality_gate\"}}" | python3 -m json.tool >/dev/null
printf '// @MX:NOTE smoke\n' > "$TMP/smoke.go"
bridge "{\"version\":\"moai.pi.bridge.v1\",\"kind\":\"tool\",\"cwd\":\"$TMP\",\"payload\":{\"tool\":\"moai_mx_scan\"}}" | python3 -m json.tool >/dev/null

printf 'MoAI Pi smoke passed: %s\n' "$SPEC"
