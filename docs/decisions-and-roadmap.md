# Decisions, handoffs, and future iterations

This document records what the prototype chooses now, what that choice gives us, and which decisions are intentionally handed to later iterations.

The purpose is not to avoid decisions. It is to make the timing and ownership of each decision explicit.

## Current decisions

| Area | MVP choice | Benefit | Cost / handoff |
|---|---|---|---|
| Core service | Go daemon | Simple process control, concurrency, deployment, and TUI reuse | Rust remains an option if later requirements justify migration |
| Frontend contract | Semantic `ViewDocument` | One application model across Web, TUI, and constrained displays | Every public block must be supported by each renderer |
| Client state | Per-frontend sessions | Independent navigation and editing | Session persistence is deferred |
| Work state | Shared daemon-owned jobs | Jobs survive client disconnects and can be operated cross-frontend | Durable recovery after daemon restart is deferred |
| Live updates | SSE with snapshot refetch | Browser-native and easy to debug | A compact/bidirectional transport may be useful later |
| Initial plugins | Declarative command apps | Small, reviewable, ideal for Ping/Nmap | Complex services need a later plugin tier |
| TUI | Bubble Tea | Existing Go toolchain and working prototype | Ratatui remains a future alternative |
| Embedded UI | Canvas renderer preview | Proves the semantic model fits 256×144 | Physical display and input integration remain hardware work |
| Tests | Fake tools + Docker + ARM64 smoke | Reproducible without a device | Does not prove device-specific peripherals |

## Component handoffs

### Application author → FlipCTL core

The author supplies typed inputs, command intent, parsing, and capability requirements.

The core supplies process lifecycle, jobs, events, limits, and frontend projection.

**Reason:** application authors should not need to implement three frontends or duplicate process supervision.

### FlipCTL core → frontend author

The core supplies a current semantic view and available actions.

The frontend supplies platform-native layout, focus, scrolling, editing, and input mapping.

**Reason:** a 256×144 D-pad interface and an SSH terminal should share meaning, not widgets.

### Userspace prototype → hardware integration

The prototype proves sessions, jobs, plugins, events, and renderers in deterministic environments.

A hardware iteration must choose and validate:

- physical key event source;
- framebuffer/display sink;
- browser/display launcher;
- DRM/Wayland strategy;
- MCU protocol integration;
- service startup on the Flipper image.

**Reason:** these choices depend on real drivers, connectors, and boot behavior and should not be guessed from a desktop environment.

## Decisions to revisit

### 1. Uniform action response

**Current question:** should every session action return the updated `ViewDocument`, or may actions such as `job.start` return a domain resource?

**Recommended direction:** return the updated document for all interactive actions. Resource APIs may remain available for diagnostics and automation.

**Decision trigger:** before freezing the public frontend API.

### 2. Native plugin protocol

Candidates include gRPC subprocesses, JSON-RPC over stdio, and D-Bus services for selected system integrations.

**Decision trigger:** the first application that cannot be represented as a declarative command app.

Evaluation criteria:

- streaming and cancellation;
- version negotiation;
- supervision;
- language support;
- packaging;
- resource footprint.

### 3. WebAssembly plugin tier

Wasm Components could provide explicit capabilities and portable third-party logic.

It should remain future work until FlipCTL proves asynchronous job streaming, cancellation and resource ownership, host API ergonomics, ARM64 runtime footprint, and language toolchain quality.

### 4. Persistence

The MVP stores sessions and jobs in memory.

Potential evolution:

- persist settings first;
- persist terminal job metadata second;
- add restart recovery only if real workflows require it.

**Decision trigger:** product requirements for reboot/reconnect continuity.

### 5. Embedded display stack

Possible launchers include direct Cog DRM, Cage + Cog/Wayland, or another renderer.

The core should not depend on one choice.

**Decision trigger:** tests on the target Flipper image and internal SPI LCD.

### 6. System service integration

Simple diagnostics can remain command apps. NetworkManager, systemd, ModemManager, power, audio, and MCU operations should move behind typed host adapters when they become product features.

**Decision trigger:** when stateful system operations need richer error handling and lifecycle semantics than a CLI wrapper provides.

### 7. TUI toolkit

Bubble Tea is the MVP choice. Ratatui is a serious alternative if the project later adopts Rust broadly or needs its immediate-mode layout ecosystem.

Changing the toolkit should not change the public `ViewDocument` or daemon contracts.

## Suggested iteration plan

### Iteration 1 — architecture proof

- Web and TUI on one daemon;
- Ping and Nmap command apps;
- shared jobs and cancellation;
- strict semantic view contract;
- deterministic userspace tests.

This repository covers this iteration.

### Iteration 2 — Linux productization

- Debian package and systemd units;
- settings persistence;
- typed host capabilities;
- first NetworkManager/systemd applications;
- operational metrics and upgrade strategy.

### Iteration 3 — Flipper One integration

- MCU input adapter;
- framebuffer/display adapter;
- selected display launcher;
- pixel renderer refinement;
- validation on Flipper R&D Debian and target hardware.

### Iteration 4 — ecosystem

- supervised native plugin protocol;
- plugin SDK and compatibility policy;
- package/distribution model;
- optional Wasm component tier;
- external control-panel transport.

## Architectural invariants

Future implementations may change languages, protocols, and rendering technology, but should preserve these invariants:

1. The daemon owns authoritative system and job state.
2. Frontend navigation is independent.
3. Jobs are not owned by transient connections.
4. Applications do not embed frontend-specific UI implementations.
5. Renderers receive semantic, versioned data.
6. Hardware and operating-system integrations are replaceable adapters.
7. Tests clearly distinguish userspace proof from hardware proof.

These invariants are the long-term value of the proposal; the specific MVP libraries are replaceable.
