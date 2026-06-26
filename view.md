
## View Layer

The **View** is the user interface layer responsible for rendering the system state and handling user interaction. The architecture supports multiple View implementations (e.g., hardware display, web interface, terminal‑based TUI), all of which follow a unified communication protocol with the `Model` (`modeld`) via D‑Bus.

## View Responsibilities

- **State Rendering** – Displaying the current system state received from the `Model`. This includes the active screen, the list of controls (fields, buttons, tables) with their values, the status of running tasks, and navigation hints.

- **User Input Forwarding** – Transmitting user interactions (button presses, menu selections, field value changes, task start/stop actions) to the `Model`.

- **Navigation Support** – Showing the current screen and providing mechanisms to switch between screens (main menu, task screens) and navigate back to the previous screen.

- **Real‑time Event Handling** – Subscribing to the `StateChanged` signal and redrawing the interface on every state update, ensuring the UI stays synchronised with the `Model`.

## Interaction with the Model

All View implementations use the same set of D‑Bus methods and signals exposed by the `Model`:

- **Subscribing to `StateChanged`** – On every model update, the View receives a JSON payload containing the current screen state.
- **Invoking model methods** to forward user actions:
  - `SelectScreen()`
  - `SetControlValue(screenId, controlId, value)`
  - `SendInput(taskId, data)`
  - `GoBack()`

In this design, the View contains no business logic and has no knowledge of the Model's internal workings. It solely renders data and relays commands to the Model.

## Abstract View Workflow

### 1. Initialization
- The View connects to the system D‑Bus bus.
- It acquires a proxy object for `dev.flipper.model` and subscribes to the `StateChanged` signal.
- It calls `GetState()` to retrieve the initial full state.

### 2. Rendering
- Upon receiving a `StateChanged` signal, the View extracts the current screen.
- Based on the `controls` array, it builds UI elements corresponding to each control (field, button, table, etc.).
- It respects the current `value`, `enabled` state, labels, and attributes.
- It displays the screen title, task status (if applicable), and navigation hints (e.g., a "Back" button).

### 3. Handling User Input
When a user interacts with the UI (click, text input, button press), the View identifies the affected control and invokes the appropriate model method:

- Menu item selection → `SelectScreen`
- Changing a field value → `SetControlValue`
- "Start" button → `Execute`
- "Stop" button → `Stop`
- Sending a command to an interactive task → `SendInput`
- Going back → `GoBack`

After calling a method, the model updates its state and sends a new `StateChanged` signal, which the View processes to refresh the UI.

### 4. Update Loop
- The View continuously listens for model signals.
- On each new signal, the View redraws the screen.
- This ensures real‑time synchronisation of the interface.

### Implementation‑Specific Notes

While all Views follow the same protocol, each implementation has its own specifics:

- **HW View** – Runs on a microcontroller with a hardware display and physical buttons. It renders graphical controls, polls buttons, and communicates with a separate Node.js service via SPI or USB interfaces.

- **Web View** – Implemented as a separate Node.js service using HTTP and WebSockets to serve a web‑based interface.

- **TUI View** – Implemented as a separate Node.js service using terminal UI libraries such as Ink or OpenTUI.

Despite different technical stacks, all Views adhere to the unified protocol and contain no business logic—they only transform data into UI elements and forward user commands to the Model.


