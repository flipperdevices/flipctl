#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

printf '\n== real ping/nmap Docker smoke ==\n'
make demo-real-tools-smoke

printf '\n== rendered Ping terminal artifact ==\n'
cat docs/plans/2026-06-21-real-tools-docker-demo/artifacts/ping-terminal-render.txt

printf '\n== rendered Nmap terminal artifact ==\n'
cat docs/plans/2026-06-21-real-tools-docker-demo/artifacts/nmap-terminal-render.txt

printf '\n== parsed real-tool job results ==\n'
jq '.ping.result.parsed, .nmap.result.parsed' \
  docs/plans/2026-06-21-real-tools-docker-demo/artifacts/real-tools-jobs.json
