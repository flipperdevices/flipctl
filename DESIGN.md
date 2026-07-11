# FlipCTL Architecture Design

## Overview

FlipCTL is a lightweight, multi-frontend plugin runner for embedded Linux systems (Flipper One and compatible headless devices). This prototype demonstrates a clean separation between a language-agnostic plugin system, a shared REST API backend, a Web UI, and a terminal TUI — all runnable without hardware.

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Control Interfaces                        │
│                                                                 │
│   ┌──────────────────┐          ┌──────────────────────────┐   │
│   │    Web Browser   │          │    Terminal / SSH         │   │
│   │   (index.html)   │          │    (tui/app.py)           │   │
│   └────────┬─────────┘          └──────────────┬────────────┘   │
│            │  HTTP fetch()                      │  httpx async   │
└────────────┼───────────────────────────────────┼───────────────┘
             │                                   │
             ▼                                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                    FastAPI Backend (server.py)                   │
│                                                                 │
│   GET  /api/plugins   →  list all plugins + input schemas       │
│   POST /api/execute   →  run plugin, return JSON result         │
│   GET  /              →  serve web/index.html                   │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Plugin Manager (core/)                         │
│                                                                 │
│   • Scans plugins/ on startup                                   │
│   • Loads & validates plugin.yaml for each plugin              │
│   • execute(name, inputs) → subprocess → JSON result           │
└──────────────┬────────────────────────────────────┬────────────┘
               │                                    │
               ▼                                    ▼
┌──────────────────────────┐          ┌──────────────────────────┐
│   plugins/ping/          │          │   plugins/nmap/          │
│   ├── plugin.yaml        │          │   ├── plugin.yaml        │
│   └── main.py            │          │   └── main.py            │
│   (wraps system ping)    │          │   (wraps nmap -F)        │
└──────────────────────────┘          └──────────────────────────┘
```

### Data flow for a plugin execution

```
User clicks "Run" in Web UI / TUI
        │
        ▼
POST /api/execute  { plugin: "ping", inputs: { target: "1.1.1.1" } }
        │
        ▼
PluginManager.execute("ping", inputs)
        │
  validates required inputs
        │
  subprocess: python plugins/ping/main.py
  stdin ← json.dumps(inputs)
        │
  stdout → json.loads(result)
        │
        ▼
{ success: true, target: "1.1.1.1", avg_ms: 12.3, ... }
        │
        ▼
HTTP 200 JSON → rendered in Web UI output panel / TUI RichLog
```

---

## Plugin System Design

### `plugin.yaml` Schema

```yaml
name: ping                         # unique plugin identifier (matches directory name)
description: "Send ICMP echo…"    # shown in UI
version: "1.0"
inputs:
  - name: target                   # field name passed to the plugin process
    type: string                   # string | integer | boolean
    required: true
    description: IP or hostname
  - name: count
    type: integer
    required: false
    default: 4
    description: Number of packets
command: python main.py            # entry-point (relative to plugin dir)
timeout: 30                        # seconds before the process is killed
```

### Execution Contract

- **Inputs:** passed as a JSON object on stdin  
- **Outputs:** a JSON object on stdout; non-zero exit with empty stdout is treated as an error  
- **Language-agnostic:** `command` can be `python main.py`, `bash main.sh`, `./main` (Rust/Go binary), or `node main.js`  
- **Isolation:** each execution is a fresh subprocess; plugins cannot share state or crash the server

### Adding a New Plugin

1. Create `plugins/<name>/plugin.yaml` following the schema above.
2. Create `plugins/<name>/main.py` (or any other entry point matching `command`).
3. Read inputs from `json.loads(sys.stdin.read())`.
4. Write a single JSON object to stdout.
5. Restart the server — the plugin manager rescans on startup.

---

## Frontend Architecture

Both frontends use **identical API calls**. The UI logic (plugin selection, input collection, result rendering) is intentionally kept symmetrical so the same behaviour is verifiable in both environments.

| Concern | Web UI | TUI |
|---------|--------|-----|
| Plugin list | `GET /api/plugins` via `fetch()` | `GET /api/plugins` via `httpx` |
| Execution | `POST /api/execute` via `fetch()` | `POST /api/execute` via `httpx` |
| Input rendering | Dynamic DOM from plugin schema | Textual Input widgets, dynamically mounted |
| Output rendering | `<pre>` formatted JSON | `RichLog` widget |
| Dependencies | None (single HTML file) | `textual`, `httpx` |

---

## Future-Proofing

### systemd / NetworkManager Integration
The `PluginManager` subprocess model is the right abstraction for OS-level integrations. A future `systemctl` plugin or `nmcli` plugin would follow the exact same `plugin.yaml` + `main.py` pattern; the server and frontends need no changes.

### WebSocket Streaming
Long-running plugins (e.g., a continuous `tcpdump` capture) can be supported by adding a `GET /api/stream/{plugin}` WebSocket endpoint. The TUI already uses async workers; the Web UI can switch from `fetch()` to a `WebSocket` for streaming output.

### Plugin Sandboxing
Each plugin runs as a subprocess. Future hardening options:
- Run plugins as a dedicated low-privilege user
- Use `bubblewrap` or Linux namespaces for filesystem isolation
- Set resource limits via `ulimit` or cgroup wrappers

### Async Task Queues
For queuing long-running scans, replace direct subprocess calls with an async task queue (e.g., `asyncio.Queue` backed by a worker pool), returning a `task_id` immediately and polling via `GET /api/tasks/{id}`.

### Renderer Abstraction
The Web UI is served as a static single-page app and can be rendered by a headless WebKit (DRM/KMS on Flipper One hardware) without modification. The TUI is already a native terminal renderer. Both use the same API — adding a third renderer (e.g., a Qt or GTK native app) requires only connecting to the existing API.

---

## Running the Prototype

### Prerequisites

```bash
python3 -m pip install -r requirements.txt
# optional: install nmap for real port scans
# macOS: brew install nmap
# Linux: sudo apt install nmap
```

### Start the backend server

```bash
uvicorn server:app --reload --host 0.0.0.0 --port 8000
```

### Open the Web UI

Navigate to `http://localhost:8000` in any browser.

### Run the TUI

In a separate terminal:

```bash
python tui/app.py
```

> The TUI connects to `http://localhost:8000` — the server must be running first.

---

## File Structure

```
flipctl/
├── core/
│   ├── __init__.py
│   └── plugin_manager.py      # Plugin discovery, validation, subprocess execution
├── plugins/
│   ├── ping/
│   │   ├── plugin.yaml
│   │   └── main.py
│   └── nmap/
│       ├── plugin.yaml
│       └── main.py
├── web/
│   └── index.html             # Zero-dependency single-page Web UI
├── tui/
│   └── app.py                 # Textual TUI
├── server.py                  # FastAPI: /api/plugins, /api/execute, /
├── requirements.txt
└── DESIGN.md                  # This document
```
