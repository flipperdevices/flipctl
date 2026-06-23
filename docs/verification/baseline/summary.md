# Baseline verification summary

- Date: 2026-06-22
- Branch: `chore/flipctl-finalization-baseline`
- Commit: `ee00de045caf8c6258f856e84389da154cc44e6d`
- Worktree: `/home/kcnc/code/flipctl/.worktrees/chore-flipctl-finalization-baseline`

## Results

- `make check`: PASS
  - Go tests, Web tests, builds, and local API smoke passed.
  - Excerpt: `Test Files 3 passed (3)`, `Tests 10 passed (10)`, `✓ built`, `/readyz` returned `{"apps":2,"ready":true}`.
- `make demo-smoke`: PASS
  - Docker Compose fake-command demo built/started, `/readyz`, `/api/v1/apps`, and `/` were probed, then cleanup ran.
  - Excerpt: container started; `/readyz` returned `{"apps":2,"ready":true}`; container/network removed.

Detailed logs are stored in `docs/plans/2026-06-22-flipctl-finalization/verification/baseline/logs/`.

## Blockers

None observed.
