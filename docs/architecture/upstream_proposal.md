# Upstream Semantic Control Proposal

## 1. Purpose

This document packages the semantic control core architecture proposal for upstream discussion.

It is intended to be a concise companion to the fuller architecture note in this repository and to stay complementary to upstream PR #4. It does not propose code, schemas, tests, prototype files, or a replacement implementation path.

This proposal is not a request to replace the daemon or ViewDocument direction.

It asks whether FlipCTL should make the semantic control core explicit as the behavioral authority, with ViewDocument-style models treated as renderer-facing projections.

## 2. Short thesis

FlipCTL should have one explicit behavioral authority for semantic state and transitions.

Renderers should consume projections of that authority. Capability wrappers should return bounded semantic results before UI state consumes them. Observability can then record accepted events, transitions, capability boundaries, and projection outputs for inspection and regression.

## 3. Proposed boundary

```text
Semantic core
  ↓
ViewDocument describes what to show
  ↓
Web / TUI / panel render it
  ↓
User input returns
  ↓
FlipCTL gives that input a clear name
  ↓
Semantic core decides what changes
```

The semantic control core owns accepted events, state transitions, capability requests, capability results, and result handling. ViewDocument-style models describe what a renderer needs to show, not necessarily the full semantic authority for the product.

Under this boundary, renderers sit in a feedback loop rather than a one-way chain. The core publishes semantic state as a ViewDocument-style projection. Renderers display that projection and return user input. The input grammar normalizes that input before the core decides any semantic transition.

## 4. Relationship to daemon / ViewDocument / Web / TUI direction

This proposal is meant to clarify the boundary between FlipCTL behavior, ViewDocument-style projections, and renderer implementations.

The daemon can remain the runtime process and integration host. The semantic control core can define the behavioral rules inside or alongside that runtime. ViewDocument can remain the renderer-facing contract consumed by Web, TUI, and panel renderers.

The boundary question is narrower:

```text
Is ViewDocument the full semantic state model,
or is it the renderer-facing projection of a deeper semantic control core?
```

This document proposes treating ViewDocument as a projection.

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

## 6. Settled proposal positions

This proposal takes the following positions:

- FlipCTL should make the semantic control core an explicit architecture boundary so behavior is owned in one place and shared across renderers.
- ViewDocument should describe what renderers need to show, not own the full behavioral state of the product.
- Capability wrappers should return bounded semantic results before those results enter UI state.
- `Home → Network → Ping → Result` is the right first proof path because it exercises navigation, capability request, bounded result handling, projection, and observability without broad product scope.
- The proposal does not conflict with the daemon, ViewDocument, Web, or TUI direction unless ViewDocument is intended to be the complete behavioral authority.

## 7. Out of scope

This proposal is a discussion package only. It does not add, request, or claim:

- implementation code
- prototype files
- schemas
- tests
- CI changes
- hardware behavior
- battery-life results
- power measurement results
