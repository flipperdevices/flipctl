#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PING_DOC="docs/plans/2026-06-21-real-tools-docker-demo/artifacts/ping-live-viewdocument.json"
NMAP_DOC="docs/plans/2026-06-21-real-tools-docker-demo/artifacts/nmap-live-viewdocument.json"

if [[ ! -f "$PING_DOC" || ! -f "$NMAP_DOC" ]]; then
  echo "Missing live artifacts. Run ./tmp/run-real-tools-smoke-and-show.sh first." >&2
  exit 1
fi

printf '\n== Ping ViewDocument rendered by flipctl-tui ==\n'
go run ./cmd/flipctl-tui --render-viewdoc "$PING_DOC" --render-width 100 --render-height 40 --render-color plain

printf '\n== Nmap ViewDocument rendered by flipctl-tui ==\n'
go run ./cmd/flipctl-tui --render-viewdoc "$NMAP_DOC" --render-width 100 --render-height 40 --render-color plain
