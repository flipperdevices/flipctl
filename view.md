
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


