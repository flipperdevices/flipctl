# Model Service (modeld)

The Model is the central management component, implementing the Model‑View‑Presenter (MVP) pattern. In this design, it combines the roles of both Presenter and Model — it holds the application state while also orchestrating user interactions and coordinating business logic.

The service acts as the single datasource for all View implementations (HW, Web, TUI) and drives task execution through the AppWrapper.

## Key Concepts
1. Plugins
Plugins are defined as JSON files stored in the plugins directory. On startup, the Model scans this folder and loads every valid plugin it finds. Each plugin describes one application (console utility) and its corresponding screen.

2. Screens
- Main Screen (Menu)
Automatically generated from the set of loaded plugins. It displays a list of all available tasks (applications) that the user can launch.

- Task Screen
Created per plugin based on its JSON definition. This screen contains controls (fields, buttons, tables, etc.) that:
    - Show the current status and output of the running application.
    - Provide input elements for parameters and control actions (e.g., start/stop).

3. Controls
Controls are the UI elements that populate a task screen. They are defined inside the plugin's controls array.

Controls can be:
- Static – explicitly described once in the JSON.
- Auto‑generated (simplified) – derived automatically from the params section to avoid duplicating input definitions. This reduces boilerplate and keeps the plugin concise.

4. Params
Parameters are specified in the plugin as an array of objects. Each parameter defines a user‑editable value (e.g., a string, number, or choice).

When the user launches the task, these parameter values are substituted into the command‑line arguments via templates – the args field of the plugin uses placeholders like {paramId} that are replaced with the actual values at runtime.

5. Parsing Rules
Defined under the parsing.rules section of the plugin.

Whenever the AppWrapper delivers new output data (lines or chunks), the Model applies these rules sequentially to the incoming text. Each rule extracts structured information (e.g., matching a regex, parsing a JSON field) and updates the corresponding controls with the fresh data, keeping the UI in sync with the application's progress.

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


