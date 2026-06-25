# Model Service (modeld)

The Model is the central management component, implementing the Model‑View‑Presenter (MVP) pattern. In this design, it combines the roles of both Presenter and Model — it holds the application state while also orchestrating user interactions and coordinating business logic.

The service acts as the single datasource for all View implementations (HW, Web, TUI) and drives task execution through the AppWrapper.

## Key Responsibilities

1. State Management
Stores and maintains the complete application state, including:

- List of available tasks
- Current active screen
- Task parameters and configuration
- Execution status (running, completed, failed)
- Accumulated output from running applications

2. Command Handling
Receives user commands from the View layer, such as:
- Task selection
- Parameter changes
- Start / stop actions
- User data input
Translates these commands into actionable operations and updates the state accordingly.

3. Coordination with AppWrapper
- Launches applications (console utilities) with the appropriate parameter substitution.
- Subscribes to output and completion signals emitted by AppWrapper.
- Updates the internal state based on real‑time data received from running processes.

4. Output Parsing
Applies parsing rules (e.g., regex patterns, JSON extraction) to the raw output of applications, extracting structured data that is then used to refresh specific UI controls (e.g., progress bars, text fields, graphs).

5. State Broadcasting
After every state change, the Model emits a StateChanged D‑Bus signal. All subscribed View services receive this signal, ensuring that every UI instance stays synchronised with the current state.

6. Navigation Management
Maintains a screen stack to support hierarchical navigation flows — for example:
- Main menu → Task selection screen → Task detail screen → Back to previous screen.
This stack enables seamless forward/backward navigation across the user interface.


