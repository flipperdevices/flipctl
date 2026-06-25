# Model Service (modeld)

The Model is the central management component, implementing the Model‑View‑Presenter (MVP) pattern. In this design, it combines the roles of both Presenter and Model — it holds the application state while also orchestrating user interactions and coordinating business logic.

The service acts as the single datasource for all View implementations (HW, Web, TUI) and drives task execution through the AppWrapper.

## Key Concepts
1. **Plugins**

Plugins are defined as JSON files stored in the plugins directory. On startup, the Model scans this folder and loads every valid plugin it finds. Each plugin describes one application (console utility) and its corresponding screen.

[Example of plugin](model-pugin.md)

2. **Screens**

- Main Screen (Menu)

  Automatically generated from the set of loaded plugins. It displays a list of all available tasks (applications) that the user can launch.

- Task Screen

  Created per plugin based on its JSON definition. This screen contains controls (fields, buttons, tables, etc.) that:

    - Show the current status and output of the running application.
    - Provide input elements for parameters and control actions (e.g., start/stop).

3. **Controls**

Controls are the UI elements that populate a task screen. They are defined inside the plugin's controls array.

Controls can be:

- Static – explicitly described once in the JSON.
- Auto‑generated (simplified) – derived automatically from the params section to avoid duplicating input definitions. This reduces boilerplate and keeps the plugin concise.

4. **Params**

Parameters are specified in the plugin as an array of objects. Each parameter defines a user‑editable value (e.g., a string, number, or choice).

When the user launches the task, these parameter values are substituted into the command‑line arguments via templates – the args field of the plugin uses placeholders like {paramId} that are replaced with the actual values at runtime.

5. **Parsing Rules**

Defined under the parsing.rules section of the plugin.

Whenever the AppWrapper delivers new output data (lines or chunks), the Model applies these rules sequentially to the incoming text. Each rule extracts structured information (e.g., matching a regex, parsing a JSON field) and updates the corresponding controls with the fresh data, keeping the UI in sync with the application's progress.

## Key Responsibilities

1. **State Management**

Stores and maintains the complete application state, including:

- List of available tasks
- Current active screen
- Task parameters and configuration
- Execution status (running, completed, failed)
- Accumulated output from running applications

2. **Command Handling**

Receives user commands from the View layer, such as:

- Task selection
- Parameter changes
- Start / stop actions
- User data input
Translates these commands into actionable operations and updates the state accordingly.

3. **Coordination with AppWrapper**

- Launches applications (console utilities) with the appropriate parameter substitution.
- Subscribes to output and completion signals emitted by AppWrapper.
- Updates the internal state based on real‑time data received from running processes.

4. **Output Parsing**

Applies parsing rules (e.g., regex patterns, JSON extraction) to the raw output of applications, extracting structured data that is then used to refresh specific UI controls (e.g., progress bars, text fields, graphs).

5. **State Broadcasting**

After every state change, the Model emits a StateChanged D‑Bus signal. All subscribed View services receive this signal, ensuring that every UI instance stays synchronised with the current state.

6. **Navigation Management**

Maintains a screen stack to support hierarchical navigation flows — for example:
- Main menu → Task selection screen → Task detail screen → Back to previous screen.
This stack enables seamless forward/backward navigation across the user interface.

## D‑Bus Interface

### Service Identification

- **Service Name:** `dev.flipper.model`
- **Object Path:** `/model`
- **Interface:** `dev.flipper.Model`

### Methods

| Method | Description |
|--------|-------------|
| `GetState()` | Returns the current full application state as a serialized JSON string. |
| `SelectScreen(screenId: string)` | Switches the UI to the specified screen (identified by its `screenId`). |
| `SetParameter(taskId: string, paramId: string, value: variant)` | Sets the value of a specific parameter for the given task. |
| `Execute(taskId: string)` | Launches the specified task with the currently configured parameters. |
| `Stop(taskId: string)` | Stops the currently running task. |
| `SendInput(taskId: string, data: string)` | Sends arbitrary input data to the `STDIN` of an interactive (running) task. |
| `GoBack()` | Navigates back to the previous screen in the navigation stack. |

### Signals

| Signal | Description |
|--------|-------------|
| `StateChanged(newState: string)` | Emitted whenever any part of the application state changes. Provides the complete updated state as a JSON string, allowing all subscribed Views to stay synchronized. |

[State example](model-state.md)


## Lifecycle of Interaction

### 1. Initialization
- The `Model` loads all JSON plugins from the `plugins` directory and generates the main menu screen plus individual task screens for each plugin.
- The initial **State** contains all screens with their controls and default values.
- The first `StateChanged` signal is emitted to notify all Views.

### 2. Navigation
- The user (via a View) selects a task from the menu.
- The `Model` switches `currentScreenId` to the corresponding task screen and pushes it onto the navigation stack.
- When the user presses **Back**, the `Model` pops the previous screen from the stack and restores it.

### 3. Parameter Changes
- A View calls `SetParameter(taskId, paramId, value)`.
- The `Model` updates the stored value in `appParams[taskId][paramId]` and broadcasts the updated state.

### 4. Task Execution
- A View calls `Execute(taskId)`.
- The `Model` builds the command‑line arguments by substituting the current parameter values into the `args` template.
- It invokes `AppWrapper.Start(appId, args, env, options)` and stores the returned process identifier.
- The task status is set to `"running"`, and controls are updated (e.g., enabling a **Stop** button).
- A `StateChanged` signal is emitted.

### 5. Output Processing
- Upon receiving an `Output` signal from the `AppWrapper`, the `Model` applies the parsing rules sequentially:
  - For each line (or chunk), it checks for matches against defined regular expressions.
  - When a match is found, it updates the `value` of the specified target control (or appends a row to a table if `multiple` is set).
- The `Model` may also accumulate the full output in a dedicated field (e.g., for logs).
- A `StateChanged` signal is emitted after every update.

### 6. Termination / Error
- When an `Exited` or `Error` signal is received from `AppWrapper`, the `Model`:
  - Updates the task status.
  - Stops any running timers.
  - Refreshes relevant controls (e.g., result fields, buttons).
- The final state is broadcast via `StateChanged`.

### 7. Interactive Input
- For tasks marked with `interactive: true`, a View may call `SendInput(taskId, data)`.
- The `Model` forwards this data to the `AppWrapper`, which writes it to the `STDIN` of the running process.

![Model diagram](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/silart/flipctl/ui_arch/diagrams/model.puml)

