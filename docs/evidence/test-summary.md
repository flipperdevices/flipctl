# Test summary evidence

## Targeted checks

- Web assets built with `cd web && vp install --frozen-lockfile && vp build`: PASS.
- Screenshot capture with headless Chromium: PASS (`docs/evidence/web.png`, `docs/evidence/canvas.png`).
- TUI render evidence: PASS (`go run ./cmd/flipctl-tui -render-viewdoc schemas/view-document-fixtures/ping-running.json -render-width 80 -render-height 24`).
- Architecture proof test: PASS (`go test ./internal/api -run TestArchitecture_OneDaemonTwoSessionsOneSharedJob -count=1`).
- Plugin load/validation package check: PASS (`go test ./internal/plugin -count=1`; package has no test files but compiles validation code).
- Plugin guide manifest validation: PASS (`go run ./tmp/f-plugin-validate <tempdir>` printed `network.example ./fakecmds/fake-ping 2`; temporary helper was removed).
- Required docs/source scan: PASS (Python assertions checked required examples, plugin UI exclusion, and removal of speculative architecture header).

## Full check

- `make check VP=vp`: PASS.
- `make check VP="corepack pnpm"`: not run to completion. A direct prerequisite attempt, `corepack pnpm --dir web install --frozen-lockfile`, failed because this environment invoked pnpm 11.8.0 while the project pins pnpm 9.15.0. `VP=vp` is the repo README local default and passed.
