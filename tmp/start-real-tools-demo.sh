#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-flipctl-real-tools-smoke}"

docker compose --profile real-tools up --build -d flipctld-real-tools

printf 'Waiting for real-tools daemon on http://localhost:18081/readyz'
for _ in {1..60}; do
  if curl -fsS http://localhost:18081/readyz >/dev/null 2>&1; then
    printf '\nReady.\n\n'
    break
  fi
  printf '.'
  sleep 1
done
curl -fsS http://localhost:18081/readyz

cat <<'MSG'

Real-tools demo is running with Linux host networking.

Open the Web UI and run real ping/nmap jobs against real targets such as ya.ru or example.com:
  http://localhost:18081

Run the actual interactive Bubble Tea TUI in another terminal:
  ./tmp/run-tui-real-tools.sh

Stop everything:
  ./tmp/stop-real-tools-demo.sh
MSG
