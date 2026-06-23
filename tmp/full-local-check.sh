#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

make renderer-contract
make test-tui-render
make check
make demo-smoke
make demo-real-tools-smoke
