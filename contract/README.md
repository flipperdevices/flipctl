# FlipCTL Contract

Formal, machine-readable artifacts that complement the prose in [`../frontend.md`](../frontend.md) and [`../backend.md`](../backend.md). Those documents used to describe the contract in words and illustrative JSON examples; here it's the same thing as real schemas that can be validated and used to generate types.

Status: PoC-stage draft (`0.1.0`), not final — see the open questions below and in `backend.md §12`/`frontend.md §11`.

## Files

| File | What it describes | Source of truth in prose |
|---|---|---|
| `openapi.yaml` | HTTP/SSE API between `flipctld` and the Web/TUI frontends | `backend.md §7`, `frontend.md §4.2-4.3` |
| `manifest.schema.json` | A plugin's `manifest.toml` (validated against its TOML→JSON representation) | `backend.md §4.1-4.4` |
| `ipc-messages.schema.json` | NDJSON messages between `flipctld` and a plugin process (stdin/stdout) | `backend.md §5` |

`GET /api/registry` in `openapi.yaml` (the `RegistryEntry` schema) and `manifest.schema.json` are intentionally not the same file: the former is what the frontend sees (a UI projection), the latter is what a plugin author writes (a full description, including button bindings and permissions). The `inputs`/`outputs`↔`status_fields` fields are semantically linked between them, but these are two different contracts at two different boundaries of the system.

## How to use this

- **Manifest validation at discovery time** (`backend.md §8`): run the parsed TOML through `manifest.schema.json` with any JSON Schema draft 2020-12 validator for Go — e.g. `github.com/santhosh-tekuri/jsonschema`. A manifest that fails validation is logged and skipped (it doesn't block the other plugins), as already described in `backend.md §8`.
- **Type codegen**:
  - Go (`flipctld` and the TUI, `frontend.md §6`/`backend.md §11.1`): **one** `oapi-codegen` run against `openapi.yaml` → the `contract/go/apitypes` package, imported directly by both Go binaries. Not two independent generators for two Go projects — one shared package.
  - TypeScript (Web, `frontend.md §5`): `openapi-typescript` against `openapi.yaml`, run separately — Web is inevitably a different language, so codegen stays a bridge here rather than a shared package.
  - For `ipc-messages.schema.json` (consumed only inside plugins and `flipctld`, not by the frontends) — `quicktype` can generate types for most plugin languages from a single JSON Schema if needed; for plugins written in Go, the same `oapi-codegen`/hand-written structs, to the plugin author's taste.
- This section used to describe a fallback plan for the case where the TUI stayed on Rust (separately comparing Go/Rust/TS codegen) — the TUI has since been rewritten in Go specifically so that the backend↔TUI pair doesn't need a per-language codegen step at all, just one shared package.

## Conscious simplifications (default choices, not agreed requirements)

Formalizing the contract required a few technical calls that weren't in the prose before — they're low-risk and reversible, but worth knowing they exist:

- **OpenAPI 3.1**, not 3.0 — JSON-Schema-compatible with `manifest.schema.json`/`ipc-messages.schema.json` (same schema grammar), but slightly less supported by older Swagger UI/codegen tool versions — check compatibility with specific tools during implementation.
- **A single error envelope** `{"error": {"code", "message"}}` for all non-2xx API responses — this was never fixed anywhere before.
- **SSE isn't typed in OpenAPI** (the spec can't express it) — `GET /api/jobs/{jobId}/events` in `openapi.yaml` only declares `text/event-stream`; the authoritative payload shape lives in `ipc-messages.schema.json`. If honest typing of SSE endpoints is ever needed, the next step is AsyncAPI on top of the same `$defs`.
- **`select` inputs with a static `options` list** — dynamic options (e.g. "a live-queried list of Wi-Fi interfaces") are mentioned as a future extension in `backend.md §4.1`, but not specified here.
- The versions of all three files are synced with `info.version: 0.1.0` in `openapi.yaml` by hand — there's no automatic version sync across files.

## What's still unresolved (not covered by this formalization)

- The soft-button slot panel as a standalone UI Kit primitive (`backend.md §11`, backlogged per the discussion).
- The Session identification mechanism (`backend.md §6`) — neither schema above describes how a client presents its `connection_id`.
- A formal schema for the `GET /api/plugins/{id}/status` response — currently described as "mirrors the manifest's `outputs[]`" (`additionalProperties: true` in `openapi.yaml`), without per-plugin strict typing.
