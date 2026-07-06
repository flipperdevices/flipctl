# Upstream Semantic Control Proposal

## 1. Purpose

This document packages the semantic control core architecture proposal for upstream discussion.

It is intended to be a concise companion to the fuller architecture note in this repository and to stay complementary to upstream PR #4. It does not propose code, schemas, tests, prototype files, or a replacement implementation path.

This proposal is not a request to replace the daemon or ViewDocument direction.

It asks whether FlipCTL should make the semantic control core explicit as the behavioral authority, with ViewDocument-style models treated as renderer-facing projections.

## 2. Short thesis

FlipCTL should have one explicit behavioral authority for semantic state and transitions.

Renderers should consume projections of that authority. Capability wrappers should return bounded semantic results before UI state consumes them. Observability can then record accepted events, transitions, capability boundaries, and projection outputs for inspection and regression.

Replay is only an observability and regression aid in this proposal. It is not the architecture premise and does not imply hardware timing, power measurement, deterministic rendering, or complete system simulation.

## 3. Proposed boundary

```text
Input happens
  ↓
FlipCTL gives it a clear name
  ↓
The core decides what it means
  ↓
ViewDocument describes what to show
  ↓
Web / TUI / panel render it
```

The semantic control core owns accepted events, state transitions, capability requests, capability results, and result handling. ViewDocument-style models describe what a renderer needs to show, not necessarily the full semantic authority for the product.

Under this boundary, renderers may emit user input, but that input is normalized through the input grammar before it affects semantic state.

## 4. Relationship to daemon / ViewDocument / Web / TUI direction

This proposal is meant to fit beside the daemon, ViewDocument, Web, and TUI direction from upstream PR #4.

The daemon can remain the runtime process and integration host. The semantic control core can define the behavioral rules inside or alongside that runtime. ViewDocument can remain the renderer-facing contract consumed by Web, TUI, and panel renderers.

The boundary question is narrower:

```text
Is ViewDocument the full semantic state model,
or is it the renderer-facing projection of a deeper semantic control core?
```

This document proposes treating ViewDocument as a projection unless upstream maintainers prefer it to be the full semantic model.

## 5. Minimal proof path

The smallest useful proof path is:

```text
Home → Network → Ping → Result
```

That path is small enough to exercise the architecture without requiring broad product scope:

- Home and Network test navigation state.
- Ping tests a bounded capability request.
- Result tests bounded capability output and semantic result handling.
- Web, TUI, and panel renderers can consume the same ViewDocument-style projection.
- Observability can record the accepted event sequence, state transitions, capability request, capability result, and projection changes for regression checks.

## 6. What feedback is requested

- Should FlipCTL make the semantic control core an explicit architecture boundary?
- Should ViewDocument be treated as the full semantic state model or as a renderer-facing projection?
- Should capability wrappers return bounded semantic results before UI state consumes them?
- Is Home → Network → Ping → Result the right minimal proof path?
- Which part of this boundary conflicts with the current daemon / ViewDocument direction?

## 7. Out of scope

This proposal does not add or request:

- code
- prototype files
- schemas
- tests
- CI changes
- hardware claims
- battery or power measurement claims
- upstream PR creation
- precision-replay-specific framing
- replacing the daemon direction
- replacing upstream PR #4
