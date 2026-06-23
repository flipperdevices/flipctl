# Testing

Use the deterministic local check before committing implementation changes. This is the authoritative target used by CI:

```bash
make check
```

`make check` runs generated contract drift detection (`make check-generated`), the canonical renderer contract lane (`make renderer-contract`), terminal renderer goldens (`make test-tui-render`), Go tests, Web tests, local binary builds, and a local daemon smoke against readiness/API/schema endpoints. It intentionally does not require Docker, browser automation, real-tools, or public network targets so the default check stays deterministic and less dependent on external image/package caches.

CI then runs the explicit support lane:

```bash
make ci-support
```

`make ci-support` runs `make openapi-validate`, `make architecture-acceptance`, `make go-vet`, and `make race-stateful`. These cover executable OpenAPI/schema validation, the two-session architecture acceptance test, Go static checks, and race tests for stateful API/runner packages. GitHub Actions bootstraps Go, Node.js, and pinned `pnpm@9.15.0`, then invokes `make check VP="corepack pnpm"` and `make ci-support VP="corepack pnpm"`; local developer shells can continue using the default `VP=vp` when VitePlus is installed.

## Test lanes

| Lane | Command / job | What it proves | Limits |
| --- | --- | --- | --- |
| Local canonical gate | `make check` plus `make ci-support` in CI | Generated contract drift, renderer contracts, TUI renderer, Go/Web tests, builds, local daemon smoke, OpenAPI/two-session/race support checks. | No Docker, real tools, ARM64 userspace, or hardware. |
| Docker userspace | `make docker-fake-smoke` / CI `docker-fake-smoke` job | Builds and starts the default fake-command image, confirms the daemon runs non-root, serves Web assets, creates a canonical session, opens fake Ping through session actions, waits for completion, and validates initial/final canonical views. | Requires Docker and image/package cache or network access; uses deterministic fake commands only. |
| ARM64 userspace | `make arm64-userspace-smoke` / CI `arm64-userspace-smoke` job | Uses Buildx/QEMU to build the runtime image for `linux/arm64`, starts it, records platform/inspection artifacts, and runs the same canonical fake Ping API flow. | Userspace container proof only; this is not RK3576 hardware emulation and does not prove kernel, USB, GPIO, timing, or device peripheral behavior. Requires Buildx/QEMU support and enough runner time/cache. |
| Future hardware | To be defined when physical Flipper One/RK3576 access exists. | Would prove real hardware/kernel/peripheral integration. | Not part of MVP CI and no hardware validation is currently claimed. |

Use the Docker demo smoke separately when validating runtime packaging:

```bash
make demo-smoke
make docker-fake-smoke
make arm64-userspace-smoke
```

`make demo-smoke` builds the Docker image, starts Compose, and curls `/readyz`, `/api/v1/apps`, and `/` on port 8080. `make docker-fake-smoke` is the stronger deterministic Docker proof: it starts the non-root default container, verifies Web assets, creates a canonical session, runs fake Ping through revision-aware semantic session actions, observes the terminal job state, and validates canonical initial/final views against `schemas/view-document.json`. `make arm64-userspace-smoke` builds and runs the same default runtime image as `linux/arm64` with Buildx/QEMU and performs the same canonical API flow. Docker checks require Docker plus network/cache access for base images and pnpm packages if they are not already available.

Use the opt-in real-tools smoke only when Docker integration and real command evidence are needed:

```bash
make demo-real-tools-smoke
```

`make demo-real-tools-smoke` starts the `real-tools` Compose profile on port 18081 with Linux host networking, records host `nmap` as absent/non-required, proves real container `ping`/`nmap`, checks non-root runtime behavior, creates canonical sessions, opens apps from the canonical home list action, updates forms and starts jobs through revision-aware session actions, captures ordered SSE live events, asserts pre-terminal output/progress where available, validates the daemon's exact `/sessions/{id}/view` response against `schemas/view-document.json`, renders that exact response with the terminal semantic renderer, and cleans up the profile. This target is intentionally excluded from `make check` because it depends on Docker, image/package cache, DNS, and external network reachability.

Renderer parity checks remain deterministic and independent of Docker. They validate that Web DOM/Canvas helpers and the TUI semantic renderer consume the canonical `viewdoc.flipctl.dev/v1alpha1` fixture corpus:

```bash
make renderer-contract
make test-tui-render
cd web && ${VP:-vp} test -- --run
cd web && ${VP:-vp} build
```

No physical Flipper One hardware validation is claimed. Real host `nmap` is not required for default checks; default Ping/Nmap demo behavior uses deterministic fake commands. Real `nmap` acceptance evidence comes from the real-tools container, which uses host networking and passes normal user-provided targets to `/usr/bin/nmap` with bounded default flags.
