# FlipCTL — Architecture (condensed version)

> Full versions: [`frontend.md`](./frontend.md), [`backend.md`](./backend.md). This file is a condensed summary for onboarding.

**Goal:** convenient access to Linux tools (networking, systemd, processes) through a pixel-perfect interface on a small screen with a limited set of physical buttons.

## Overall architecture
[![FlipCTL arch preview](/assets/260711-FlipCTL-arch-preview.png)](/260711-FlipCTL-arch.pdf)

> [PDF version](/260711-FlipCTL-arch.pdf)

Key ideas:
- **A shared contract instead of shared code.** Frontend (Web and TUI) and backend are three independent stacks, each idiomatic to its own platform, connected by an API/schemas (`contract/`) rather than a shared runtime.
- **A system of user and system plugins in a thin Go wrapper.** Plugins can be written in any suitable language, or built on existing Linux mechanisms like D-Bus or libraries (libcurl).
- **The GUI is built on web technologies.** Easy to develop, cheap to maintain.
- **Web and TUI share a common backend API, but don't share UI logic or assets between themselves.** Each has its own rendering engine.
- **Rendering to the device screen is done from a headless browser via Linux DRM.** No fuss with drivers, a small memory footprint. A full-fledged web interface as a bonus.

## Frontend

- **Web** — React + `react-dom`, rendering into a single `<canvas>` (`PixelSurface`/`CanvasSurface`) to preserve the prototype's pixel-perfect 1-bit style. We don't use Yoga to try to unify the UI with the TUI — instead we use fixed pixel constants, as in `fake-FlipCTL`.
- **TUI** — Go + `bubbletea`/`lipgloss`/`bubbles`. Same language as the backend — real synergy on shared types (`contract/go/apitypes`). Not pixel-perfect, a regular text UI. A compiled binary, saving resources on the portable SBC. SSH — via a forced-command on the system `sshd` (or `wish` from the same `bubbletea` stack).
- Shared between Web and TUI:
  - input semantics (`InputAction`: Up/Down/Ok/Back/SoftKey1/...);
  - the navigation model (a screen stack + a separate overlay stack);
  - the app registry — comes from the backend's registry manager, not stored in the frontend.
- Example of **a single element** of the array returned by `GET /api/registry` — this is how the backend describes one specific plugin (`ping`) to the frontend; the full registry is an array of such objects, one per installed plugin:
  ```jsonc
  { "id": "ping", "kind": "generic",
    "inputs": [{ "id": "target", "type": "text" }],
    "actions": [{ "id": "run", "endpoint": "/api/plugins/ping/run", "bind": "slot:2" }] }
  ```
  - `id` — the plugin's unique identifier; all other paths are built from it (`/api/plugins/ping/...`).
  - `kind: "generic"` — the frontend doesn't need to look for a hand-written screen; the plugin is drawn by the shared `GenericActionScreen.tsx`/`generic_action.go` component. For comparison: `kind: "custom"` → a hand-written screen (Wi-Fi, Ethernet, Power) with its own logic.
  - `inputs` — fields the user fills in before launch. Here there's one text field, `target` — a text input will appear on screen; whatever's typed there is passed to the plugin under the same `target` key (see the IPC example below — it's the same `ping`, just at the next step).
  - `actions` — the screen's soft buttons.

## Backend

- **`flipctld` (Go)** — a thin supervisor. Not a store of business logic. Replaces `server.js` from `fake-FlipCTL`. It implements the API, while a separate `Caddy` handles the Web build's static files and external requests, sitting in front of it (reverse proxy `/api/*`), not `flipctld` itself.
- **Plugins** — separate OS processes. Any language can be used (the only requirement is being able to read/write JSON). They're wired in through a small Go wrapper. Packaged together with a manifest, binaries, icons, and a declarative UI description. They can be split and classified along several axes.

  - By access level (`tier`):
    - `system` (Wi-Fi, power, cron, ...) — a mandatory baseline. Written and vetted by the Flipper team.
    - `community` (ping, nmap, curl, ...) — user-contributed.
  - By execution type (`execution`):
    - `one-shot` — starts, does its work, exits. `whoami`.
    - `stream` — stays alive, sends data and events. `ping`
    - `daemon` — lives independently of clients, mostly idle, waiting for something. E.g. a notification system.
  - By UI type (`ui_type`):
    - `custom` — its own unique UI, backed by real assets in `frontend/UI`.
    - `generic` — uses primitives from the default UI kit: list, grid, MenuBar, softKeys, dropDown, ...

- **IPC** — a separate protocol between `flipctld` and a specific running plugin process; kicked off right after the user presses the button from the example above. NDJSON = one JSON object per line — the simplest possible format, needs no library in any language:
  ```json
  {"type":"start","inputs":{"target":"8.8.8.8"}}
  {"type":"output","fields":{"status":"reachable","rtt_ms":13.2}}
  {"type":"done","exit_code":0}
  ```
  - Line 2 — the plugin writes to its own **stdout**: it called the system `ping`, parsed its output itself, and handed back an already-structured result (`status`, `rtt_ms` — fields declared in the plugin's manifest as `outputs`). `flipctld` picks up this line and broadcasts it to every client subscribed to this job over SSE — including clients that didn't start the job themselves.

- **Job / Session / Event Bus** — a job is shared across all clients from the start: anyone can subscribe to an already-running job over SSE (Web and TUI see the same live stream at the same time). There are more states than just "running/not running" — it matters exactly at which stage something went wrong:

  ```mermaid
  stateDiagram-v2
      [*] --> Pending: POST /api/plugins/{id}/{action}
      Pending --> Running: process spawned, got "ready"
      Pending --> Failed: failed to start the process
      Running --> Completed: "done", exit_code=0
      Running --> Failed: "error" or exit_code != 0
      Running --> Stopped: stop_job (SIGTERM)
      Completed --> [*]
      Failed --> [*]
      Stopped --> [*]
  ```

- **Button input in plugins**: an action in the manifest has a `bind` (`slot:0..4` — a position on the screen's button panel, or `input:back` — a universal action) and separate `on_press`/`on_release` handlers with an effect of `start_job | stop_job | send_event`. `send_event` sends a named event to the stdin of an already-running process instead of spawning a new one.
- **System plugins can (and should) use D-Bus** (talking to NetworkManager/systemd directly) instead of just parsing CLI output with regex.
- **Privileges (deferred for now)**: all plugins are currently equally trusted. A future candidate is D-Bus policy + **polkit**, the same stack NetworkManager itself uses; it only works if each plugin has its own D-Bus identity (the plugin holds its own client, rather than `flipctld` acting on its behalf).

## Contract
- `contract/openapi.yaml` — the HTTP/SSE API (`/api/registry`, `/api/plugins/{id}/{action}`, `/api/jobs/*`).
- `contract/manifest.schema.json` — the JSON Schema for a plugin manifest.
- `contract/ipc-messages.schema.json` — the schema for NDJSON messages between `flipctld` and a plugin.
- Formalized so the risk of manual type drift is closed by codegen: the backend and TUI (both in Go) use one generated `contract/go/apitypes` package directly; Web (TypeScript) separately generates TS types from the same schema.

## Notes
- There was an option to draw the TUI first and generate the web UI from it — dropped because you can't get a pixel-perfect screen that way (in a TUI we're working with a tall terminal cell, not a square pixel), and it also drags in Node.js and WASM.
- I'd sketch out the **critical path** and build a quick prototype to test this architecture's assumptions. **It wouldn't have a TUI branch of the architecture diagram at all**.
- **COG/WPE needs to be replaced**, improved, or at least researched more deeply — right now there are real problems getting it running in pure DRM mode on actual hardware.
- **Backend** = Golang partly for the synergy (TUI frontend is `bubbletea`) — the backend and TUI use a shared generated type package directly. Bubbletea and its whole ecosystem is an old, well-proven project.
- **Plugin sandboxing** — deferred.
- **Plugin permissions system** — deferred.
- **Caddy** as a separate process was chosen deliberately for a low barrier to entry and an independent release cycle for the Web frontend — the cost: a second resident process on the SBC.
