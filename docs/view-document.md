# ViewDocument contract

## Purpose

`ViewDocument` is FlipCTL's renderer-neutral UI contract. The daemon projects session, app, form, and job state into one semantic document so Web, TUI, and the 256×144 Canvas preview render the same state without sharing frontend widgets.

## Canonical version

The only public payload version is `viewdoc.flipctl.dev/v1alpha1` in the root `apiVersion` field. The HTTP API version (`/api/v1`) is separate from this document contract. The machine-readable source of truth is `schemas/view-document.json` (JSON Schema Draft 2020-12).

## Root fields

Required fields:

- `apiVersion` — constant `viewdoc.flipctl.dev/v1alpha1`.
- `sessionId` — daemon session that owns this projected view.
- `viewId` — stable identity for the current view, for example `home-app-list`, `ping-form`, or `ping-running`.
- `revision` — monotonically increasing integer for this session view.
- `title` — renderer-neutral page/title text.
- `blocks` — ordered array of semantic blocks.

Optional fields:

- `actions` — top-level semantic actions available for the current view.

Unknown root fields are invalid.

## Block union

Every block has `id` and `kind`. The schema discriminates by `kind`, requires the payload for that kind, and rejects unknown or cross-kind fields.

### `text`

Fields: `id`, `kind: "text"`, `text`; optional `title`.

<!-- viewdocument-json: skip block fragment -->
```json
{ "id": "intro", "kind": "text", "title": "Welcome", "text": "Choose an app." }
```

### `notice`

Fields: `id`, `kind: "notice"`, `severity`, `text`; optional `title`. `severity` is `info`, `success`, `warning`, or `error`.

<!-- viewdocument-json: skip block fragment -->
```json
{ "id": "status", "kind": "notice", "severity": "info", "text": "Job is running." }
```

### `list`

Fields: `id`, `kind: "list"`, non-empty `items`; optional `title`. Each item has `label` and optional `value`, `description`, and `action`.

<!-- viewdocument-json: skip block fragment -->
```json
{
  "id": "apps",
  "kind": "list",
  "title": "Apps",
  "items": [
    {
      "label": "Ping",
      "value": "network.ping",
      "description": "Check host reachability",
      "action": { "id": "app.open", "label": "Open Ping", "enabled": true, "params": { "appId": "network.ping" } }
    }
  ]
}
```

### `form`

Fields: `id`, `kind: "form"`, non-empty `fields`; optional `title`. Fields use `id`, `label`, `type`, optional `required`, `default`, `options`, and `validation`.

<!-- viewdocument-json: skip block fragment -->
```json
{
  "id": "ping-form",
  "kind": "form",
  "title": "Ping options",
  "fields": [
    { "id": "target", "label": "Target", "type": "string", "required": true, "default": "127.0.0.1", "validation": "^[A-Za-z0-9_.:-]+$" },
    { "id": "count", "label": "Count", "type": "select", "default": "3", "options": ["1", "2", "3", "4", "5"] }
  ]
}
```

### `key_value`

Fields: `id`, `kind: "key_value"`, non-empty `pairs`; optional `title`. Each pair has `key` and `value`.

<!-- viewdocument-json: skip block fragment -->
```json
{ "id": "job", "kind": "key_value", "title": "Job", "pairs": [{ "key": "State", "value": "running" }] }
```

### `table`

Fields: `id`, `kind: "table"`, non-empty `columns`, `rows`; optional `title`. Columns have `id` and `label`; each row has a `cells` object keyed by column id.

<!-- viewdocument-json: skip block fragment -->
```json
{
  "id": "hosts",
  "kind": "table",
  "title": "Hosts",
  "columns": [{ "id": "host", "label": "Host" }, { "id": "state", "label": "State" }],
  "rows": [{ "cells": { "host": "127.0.0.1", "state": "up" } }]
}
```

### `log`

Fields: `id`, `kind: "log"`, `lines`; optional `title`. Lines are ordered objects with `text` and optional `id`, `source`, and `severity`.

<!-- viewdocument-json: skip block fragment -->
```json
{ "id": "output", "kind": "log", "title": "Output", "lines": [{ "id": "1", "source": "stdout", "text": "PING 127.0.0.1" }] }
```

### `progress`

Fields: `id`, `kind: "progress"`, `progress`; optional `title`. Progress has required `label` and optional numeric `value`, numeric `max`, and display `text`.

<!-- viewdocument-json: skip block fragment -->
```json
{ "id": "progress", "kind": "progress", "progress": { "label": "Ping", "value": 1, "max": 3, "text": "1 of 3 replies received" } }
```

## Action semantics

A `ViewAction` describes intent, not transport. Required fields are:

- `id` — semantic action identifier such as `app.open`, `form.update`, `job.start`, `job.select`, `job.cancel`, or `navigation.back`.
- `label` — user-visible label.
- `enabled` — whether the action can currently be invoked.

Optional fields:

- `role` — `primary`, `secondary`, or `destructive` presentation hint.
- `params` — semantic string parameters, such as `appId` or `jobId`.

Submit actions to the canonical session endpoint with the current view identity:

<!-- viewdocument-json: skip action request payload -->
```json
{ "viewId": "ping-form", "revision": 3, "actionId": "job.start", "appId": "network.ping" }
```

`method`, `path`, inverted `disabled`, URLs, HTML snippets, terminal escape sequences, and Canvas drawing commands do not belong in actions.

## Revision semantics

`revision` is optimistic concurrency for one session's view. Clients must include the current `viewId`, `revision`, and `actionId` when posting an action. Missing fields return structured `400`. Stale fields return `409` and do not mutate session or job state.

Example stale response (an HTTP error envelope, not a `ViewDocument`):

<!-- viewdocument-json: skip HTTP error envelope -->
```json
{
  "error": {
    "code": "stale_view",
    "message": "viewId does not match current view",
    "field": "viewId",
    "viewId": "ping-running",
    "revision": 5
  }
}
```

After `409`, clients refetch `/api/v1/sessions/{sessionId}/view` and decide whether to retry.

## Snapshot versus events

A session view is a snapshot: `GET /api/v1/sessions/{sessionId}/view` returns the current `ViewDocument`. SSE events are notifications and replay records: clients use `/api/v1/events` to learn that jobs or views changed, then refetch the canonical view. Events do not replace schema validation of the snapshot.

## Renderer responsibilities

Renderers must:

- render generic block kinds consistently with their platform constraints;
- invoke semantic actions through the session action endpoint;
- include current `viewId` and `revision` for every action;
- refetch after stale revision responses or relevant SSE notifications;
- treat Canvas as a deterministic 256×144 preview of the same document, not as a separate app model.

## What does not belong

Do not put these in `ViewDocument`:

- app-specific public result fields for Ping, Nmap, or future tools;
- HTTP methods, paths, or frontend routes in actions;
- HTML, CSS, terminal escape sequences, Canvas drawing instructions, or component names;
- privileged hardware claims or host-specific paths/secrets.

Parsers may produce tool-specific internal structures, but the daemon must project them into generic blocks before they reach renderers.

## Versioning rules

The schema is strict. Additive changes require schema, fixture, renderer contract, and documentation updates. Breaking changes require a new `apiVersion`. Unknown fields are invalid so incompatible extensions fail early in tests and clients.

## Complete Ping example

<!-- viewdocument-json: validate full Ping ViewDocument -->
```json
{
  "apiVersion": "viewdoc.flipctl.dev/v1alpha1",
  "sessionId": "s-web",
  "viewId": "ping-running",
  "revision": 5,
  "title": "Ping",
  "blocks": [
    { "id": "summary", "kind": "key_value", "title": "Job", "pairs": [{ "key": "Job ID", "value": "j-1" }, { "key": "State", "value": "running" }] },
    { "id": "progress", "kind": "progress", "progress": { "label": "Ping", "value": 1, "max": 3, "text": "1 of 3 replies received" } },
    { "id": "output", "kind": "log", "title": "Output", "lines": [{ "source": "stdout", "text": "PING 127.0.0.1" }, { "source": "stdout", "text": "64 bytes from 127.0.0.1" }] }
  ],
  "actions": [{ "id": "job.cancel", "label": "Cancel", "enabled": true, "role": "destructive", "params": { "jobId": "j-1" } }]
}
```

## Complete Nmap example

<!-- viewdocument-json: validate full Nmap ViewDocument -->
```json
{
  "apiVersion": "viewdoc.flipctl.dev/v1alpha1",
  "sessionId": "s-tui",
  "viewId": "nmap-complete",
  "revision": 7,
  "title": "Nmap",
  "blocks": [
    { "id": "status", "kind": "notice", "severity": "success", "text": "Scan complete." },
    { "id": "hosts", "kind": "table", "title": "Hosts", "columns": [{ "id": "host", "label": "Host" }, { "id": "state", "label": "State" }], "rows": [{ "cells": { "host": "127.0.0.1", "state": "up" } }] },
    { "id": "output", "kind": "log", "title": "Output", "lines": [{ "id": "1", "source": "stdout", "text": "Starting Nmap" }, { "id": "2", "source": "stdout", "text": "Nmap scan report for 127.0.0.1" }] }
  ],
  "actions": [{ "id": "navigation.back", "label": "Back", "enabled": true, "role": "secondary" }]
}
```
