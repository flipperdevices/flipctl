# Demo

The default demo uses Docker Compose and the deterministic fake command harness. It does not require host `nmap`, does not run real network scans, and does not claim physical Flipper One hardware validation.

```bash
make demo
```

`make demo` builds the image, including the Preact/Vite Web frontend with Corepack/pnpm, starts `flipctld` on <http://localhost:8080>, loads `plugins/`, and serves the built Web UI from `web/dist` inside the container.

For a bounded non-interactive check, run:

```bash
make demo-smoke
```

`make demo-smoke` validates the Compose-built fake-command runtime by starting the service, probing `/readyz`, `/api/v1/apps`, and `/`, then stopping the Compose stack. It needs a working local Docker daemon and network/cache access for image package downloads when the image is not already cached.

For cross-frontend canonical checks without Docker, use `make check` or the narrower `make renderer-contract` / `make test-tui-render` lanes. These prove that API sessions emit schema-valid canonical `ViewDocument` data and that Web/Canvas/TUI renderer paths consume the same generic blocks without PTY/xterm or Web-hosted TUI shims.

## Opt-in real-tools profile

The real-tools demo is intentionally separate from the default fake path:

```bash
make demo-real-tools-smoke
```

This target starts `docker compose --profile real-tools` with `flipctld-real-tools` on <http://localhost:18081> using Linux host networking. It records host `nmap` as absent or non-required, verifies real `ping` and `nmap` inside the container, checks the daemon/container command user is non-root, creates canonical sessions, opens Ping and Nmap from the canonical home list action, updates form values and starts jobs through revision-aware session actions, captures ordered SSE live events, polls final parsed results, validates the daemon's exact `/sessions/{id}/view` response against the checked-in schema, and renders that exact response through `cmd/flipctl-tui --render-viewdoc`.

Security and scope limits:

- The real-tools profile remains opt-in; `make check`, `make demo`, and `make demo-smoke` keep using fake commands.
- Real `nmap` proof is containerized. Host `nmap` must not be installed or used for acceptance.
- The real-tools profile intentionally does not impose a demo-only target allowlist. `plugins-real/nmap.yaml` passes the requested target to real `/usr/bin/nmap` with bounded default arguments (`-sT`, `-Pn`, `--stats-every 1s`, `-p 80`), and real command errors/output are surfaced to clients.
- The Compose profile does not use `privileged: true`; the daemon image runs as `nobody`.
- No physical Flipper hardware validation is claimed.

## Canvas renderer preview scope

The Canvas panel in the Web demo is an embedded renderer preview, not a fully interactive embedded frontend. It consumes the same current canonical `ViewDocument` as the Web DOM renderer and draws deterministic generic block/action semantics into an exact 256×144 canvas. Buttons below the canvas are simulator/browser controls outside the canvas surface.

What Canvas proves:

- Web and Canvas consume the same current `ViewDocument`;
- generic home, form, running, result, and error states produce deterministic 256×144 renderer snapshots;
- the renderer has no Ping/Nmap-specific branch.

What Canvas does not yet prove:

- physical Flipper display or hardware button input;
- a fully interactive embedded UI loop;
- DRM/Cage/Cog/MCU/I²C/SPI integration;
- Playwright PNG goldens. Current evidence uses deterministic draw-log/snapshot tests as the practical substitute. Future interactive Canvas and hardware work is tracked separately from the MVP renderer-preview claim.
