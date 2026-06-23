# ADR-0004: Canonical ViewDocument contract

Status: Accepted

FlipCTL uses exactly one public ViewDocument contract version:
`viewdoc.flipctl.dev/v1alpha1`.

The HTTP API keeps its `/api/v1` route version independently from the
ViewDocument `apiVersion`. API versioning describes transport endpoints;
ViewDocument versioning describes the renderer-neutral document payload shared
by Web, Canvas, and terminal adapters.

The canonical document root contains `apiVersion`, `sessionId`, `viewId`,
`revision`, `title`, and `blocks`, with optional generic `actions`. Blocks use a
small semantic vocabulary: `text`, `notice`, `list`, `form`, `key_value`,
`table`, `log`, and `progress`.

ViewDocument must not expose parser- or app-specific public result fields such
as Ping reply arrays or Nmap port-state model fields. Tool-specific data belongs
in backend job/parser models and is projected into generic semantic blocks.
