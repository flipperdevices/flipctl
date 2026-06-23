#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

cat <<'MSG'
Starting interactive FlipCTL TUI against real-tools daemon.

The real-tools daemon uses Linux host networking; Ping and Nmap targets default to real external hosts and may be changed in the app fields.

Keys:
  j/down      move down
  k/up        move up
  enter or s  start selected app
  r           refresh jobs
  c           cancel first running job
  q           quit

MSG

go run ./cmd/flipctl-tui -addr http://localhost:18081
