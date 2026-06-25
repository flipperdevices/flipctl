# Architecture Overview

The UI system is built upon a microservice architecture. It comprises three primary logical layers, each implemented as standalone daemons (services) that communicate via D‑Bus.

## Core Components

1. Model (modeld)
The central orchestration service. Acts as the brain of the system, responsible for managing core business logic, state, and coordinating requests between the view layer and the system layer.

2. View (hw_viewd, web_viewd, tui_viewd)
The user interface services. Each daemon handles a specific presentation medium:

- hw_viewd – Hardware/embedded UI rendering.
- web_viewd – Web-based interface rendering.
- tui_viewd – Terminal-based (TUI) interface rendering.

3. AppWrapper (app_wrapperd)

The execution service. This daemon is responsible for managing, spawning, and interfacing with underlying console utilities.

## Design Pattern

The system employs the Model-View-Presenter (MVP) architectural pattern.

To reduce complexity and avoid unnecessary overhead, the Model and Presenter roles are combined (merged) within the modeld service, streamlining the communication flow between the central logic and the views.

## Technology Stack

- Runtime: All services are written entirely in Node.js.
- Concurrency Model: The system leverages an asynchronous, event-driven architecture, utilizing non-blocking I/O to ensure high responsiveness and efficient resource utilization across all daemons.

![Syatem diagram](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/silart/flipctl/ui_arch/diagrams/main.puml)


