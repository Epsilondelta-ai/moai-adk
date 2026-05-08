#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT/.moai/runtime/pi-smoke-transcript.txt}"
mkdir -p "$(dirname "$OUT")"
{
  echo "# Pi MoAI Transcript Smoke"
  echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo
  "$ROOT/scripts/pi-smoke.sh"
} | tee "$OUT"

if ! grep -q "MoAI Pi smoke passed" "$OUT"; then
  echo "transcript smoke failed: missing completion marker" >&2
  exit 1
fi

echo "transcript=$OUT"
