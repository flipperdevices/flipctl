#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

printf '\n== renderer contract ==\n'
make renderer-contract

printf '\n== terminal renderer goldens ==\n'
make test-tui-render

printf '\nRenderer checks passed.\n'
