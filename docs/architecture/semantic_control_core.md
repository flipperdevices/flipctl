# Semantic Control Core Architecture Proposal

## 1. Purpose

FlipCTL should be organized around a semantic control core rather than a renderer-led UI stack.

This document proposes an upstream-facing architecture boundary for FlipCTL. It defines the semantic control core as the behavioral center of the system: the place where normalized events, stable state, bounded capabilities, power policy, plugin contracts, and observable transitions meet.

Replay is not the thesis of this proposal. Replay becomes possible because events, state, capabilities, and transitions are stable and observable. In this architecture, replay is an observability and regression tool for checking drift, not the premise for the product or implementation.

## 2. Problem

A renderer-led UI stack can make early prototypes easier to see, but it risks placing behavior in the wrong layer. If screens, widgets, terminal views, web views, or device-specific presentation paths become the source of truth, then FlipCTL behavior may become difficult to share, test, observe, and evolve across render targets.

The architectural problem is to define one stable control contract that can survive multiple renderers and later implementation choices. FlipCTL needs a model where UI rendering is an output of semantic state, not the owner of semantic behavior.

## 3. Core rule

```text
State is authority.
Renderers are targets.
Capabilities are bounded effects.
Power is state.
Observability records behavior.
Replay checks drift.
```

The core rule means:

- state transitions are owned by the semantic control core
- renderers receive projections of state and submit semantic events, but do not own behavior
- capabilities are explicit effect boundaries, not ad hoc host calls
- power mode and power intent are part of the state model
- observability records what happened at semantic boundaries
- replay compares recorded behavior with current behavior to identify regressions

## 4. Proposed architecture diagram

```text
Event Sources
  ↓
Input Grammar
  ↓
Semantic Control Core
  ├─→ Renderers
  ├─→ Capability Bus
  ├─→ Plugin System
  ├─→ Power Policy
  └─→ Observability / Replay
```

A renderer-facing projection, such as a ViewDocument-style model, can sit between the semantic control core and renderers. In that arrangement, the projection is not the full behavioral authority; it is the renderable view of semantic state.

## 5. Component definitions

### Event Sources

Event sources produce raw input or external signals. They may include buttons, keyboard input, pointer input, timers, backend notifications, network status changes, plugin signals, or renderer-originated input.

Event sources should not directly mutate application state. They should feed the input grammar.

### Input Grammar

The input grammar normalizes raw input and signals into named semantic events. It turns implementation-specific actions into stable intent, such as moving selection, opening a tool, starting a ping, cancelling an operation, or acknowledging a result.

The grammar is the first compatibility boundary between hardware, host environments, renderers, and the semantic control core.

### Semantic Control Core

The semantic control core owns application state and state transitions. It receives named semantic events, evaluates them against current state, and produces the next state plus any bounded effects requested through declared capability interfaces.

The core should not be renderer-specific. It should describe what FlipCTL is doing, not how a particular screen draws that behavior.

### Renderers

Renderers are targets. They consume state projections and render them for a specific environment, such as a web UI, TUI, hardware display, simulator, or debug view.

Renderers may emit user actions back into event sources or the input grammar, but they should not own semantic transitions. Renderer behavior should be replaceable without changing the core state machine.

### Capability Bus

The capability bus is the boundary for bounded effects. Capabilities represent host operations such as network checks, filesystem access, device communication, shell-adjacent operations, or other integration points.

The semantic control core requests capabilities through declared interfaces. Capability implementations perform the effect and return structured results or events. This keeps effects explicit, reviewable, observable, and mockable.

### Plugin System

The plugin system extends FlipCTL through declared contracts. Plugins should add or contribute semantic events, state extensions, renderer projections, or capabilities only through bounded extension points.

Plugins should not bypass the core to mutate state or call host effects directly. A plugin should be understandable as an extension of the control model rather than as an unbounded patch into the runtime.

### Power Policy

Power policy is explicit state. The system should represent power mode, power intent, and power-sensitive scheduling decisions as part of the semantic model rather than as hidden renderer or backend behavior.

This proposal does not claim specific battery, hardware, or measurement results. It only defines the architecture direction: power decisions should be observable state transitions and policy choices.

### Observability / Replay

Observability records behavior at semantic boundaries: events accepted by the grammar, state transitions produced by the core, capability requests and responses, power policy changes, renderer projections, and relevant errors.

Replay checks drift. It should be used to compare current behavior against previously recorded semantic traces for regression detection. Replay is not proposed as the central product premise and should not be framed as adding replay to FlipCTL. It is a consequence of stable semantic boundaries.

## 6. Runtime flow

1. An event source produces raw input or a signal.
2. The input grammar maps that input into a named semantic event.
3. The semantic control core receives the event with the current state.
4. The core decides whether the event is valid for the current state.
5. The core produces a new authoritative state.
6. If an effect is needed, the core requests it through the capability bus.
7. Capability results return as bounded results or new semantic events.
8. Renderers receive projections of the authoritative state.
9. Power policy updates are represented as state changes.
10. Observability records the accepted events, transitions, projections, capability boundaries, and errors needed for later inspection.
11. Replay can use those records to check whether current behavior has drifted.

## 7. Capability flow

Capability flow should be explicit and directional:

```text
Semantic Control Core
  ↓ request declared capability
Capability Bus
  ↓ invoke bounded implementation
Capability Implementation
  ↓ return result or error
Capability Bus
  ↓ normalize response
Semantic Control Core
  ↓ update state through transition
Renderers / Observability
```

The core should not depend on concrete host implementations. Host behavior should be reached through named capabilities with declared inputs, outputs, errors, and permission expectations.

## 8. Plugin contract direction

The plugin contract should move toward declared extension points rather than arbitrary runtime mutation. A plugin may propose or provide:

- semantic events it can emit
- state it needs to expose through the core
- renderer projections or view contributions derived from state
- capabilities it requires
- capabilities it provides
- power policy hints or constraints
- observability labels useful for inspection and regression

A plugin should not own the main state authority. The semantic control core remains responsible for accepting plugin events, applying transitions, and deciding what becomes authoritative state.

## 9. Power policy direction

Power policy should be modeled as state and transition rules. Examples of future policy state may include active, idle, suspended, network-waiting, user-waiting, or capability-running modes.

The policy direction is intentionally conservative for this proposal:

- define where power policy belongs in the architecture
- avoid hardware-specific claims
- avoid battery-life claims
- avoid measurement claims
- make power-related decisions visible to observability
- let renderers reflect power state without owning it

## 10. Observability/replay boundary

Observability should record semantic behavior without turning replay into the architecture premise. Useful records may include:

- normalized semantic events
- accepted and rejected transitions
- previous and next state identifiers or summaries
- capability requests, responses, and errors
- power policy transitions
- renderer projection identifiers or summaries
- plugin-originated events at declared boundaries

Replay should be limited to observability and regression. Its role is to check whether the same semantic input sequence still produces compatible transitions and outputs. It should not be used in this proposal to claim deterministic rendering, hardware timing, power measurement, or complete system simulation.

## 11. Minimal prototype target

The proposed first implementation path should be bounded to:

```text
Home → Network → Ping → Result
```

That path is a later implementation target, not part of this documentation PR. It is intentionally small because it can exercise the architecture without requiring broad product scope:

- event source: user chooses Network and Ping
- input grammar: selection and action events become semantic events
- semantic control core: owns navigation, ping request state, and result state
- capability bus: requests a ping-like network capability
- renderer: displays Home, Network, Ping, and Result states
- power policy: represents idle, running, and result states as explicit state where appropriate
- observability: records events, transitions, capability boundaries, and result handling
- replay: checks the path for regression drift

## 12. Contrast with upstream PR #4

This proposal is intended to complement upstream PR #4, not compete with it.

Upstream PR #4 appears to advance a daemon plus ViewDocument plus Web/TUI prototype direction. This architecture proposal separates one boundary question:

```text
Is ViewDocument the full semantic state model,
or is it the renderer-facing projection of a deeper semantic control core?
```

This document proposes the latter boundary:

```text
Event sources
  ↓
Input grammar
  ↓
Semantic control core
  ↓
ViewDocument projection
  ↓
Renderers
```

Under this model, a ViewDocument-style layer can remain valuable as a renderer-facing contract. The semantic control core simply sits upstream of that contract and owns behavior before it becomes a renderable projection.

## 13. Explicit out-of-scope list

This PR does not add or claim:

- implementation code
- prototype code
- schemas
- tests
- CI changes
- vendored assets
- screenshots
- benchmark claims
- hardware claims
- power measurement claims
- live backend integration
- deterministic rendering claims
- precision-replay-specific claims
- a full plugin API
- a complete capability schema
- a complete power state taxonomy
- a replacement for upstream PR #4

## 14. Acceptance criteria for the architecture proposal

This architecture proposal is acceptable when:

- `docs/architecture/semantic_control_core.md` exists
- the document stands alone without requiring precision-replay context
- FlipCTL is framed around a semantic control core rather than a renderer-led UI stack
- replay is described only as observability and regression
- the proposal is framed as complementary to upstream PR #4
- event sources, input grammar, renderers, capability bus, plugin system, power policy, and observability/replay are defined
- the proposed first implementation path is bounded to `Home → Network → Ping → Result`
- no code or prototype files are added in this PR
