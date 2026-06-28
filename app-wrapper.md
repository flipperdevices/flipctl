# AppWrapper Service (`app_wrapperd`)

`AppWrapper` is a service responsible for launching, managing, and monitoring arbitrary console utilities and applications. It acts as the execution engine for the `Model` (`modeld`), providing a unified interface for handling child processes regardless of their nature—whether they are one‑off tasks, long‑running daemons, or interactive sessions. The service is **fully asynchronous** and built upon Node.js' event‑driven model.

---

## Key Concepts

### Process
- Each running instance of an application is identified by a **unique D‑Bus object path** (e.g., `/appwrap/process/<id>`).
- For each process, a separate D‑Bus object is created that implements the `dev.flipper.Process` interface. Through this interface, the client (Model) can receive signals about output and termination.
- Processes can be of **two types**:
  - **Regular** – launched via `child_process.spawn`, with I/O through standard streams.
  - **Interactive** – uses `node-pty` to provide a full‑featured terminal.

### Client Interface
- The **Manager** (at `/appwrap/manager`) provides methods for process management.
- The **Process object** (at `/appwrap/process/<id>`) provides signals for receiving data.

---

## Key Responsibilities

- **Process Launch** – Spawning applications with specified command‑line arguments, working directory, and environment variables.
- **Lifecycle Management** – Controlling the process lifecycle: stopping via signals, forced termination, and enforcing timeouts.
- **Interactive I/O** – Handling interactive input/output for utilities that require a pseudo‑terminal (PTY) via `node-pty`.
- **Stream Redirection** – Capturing `stdout` and `stderr` streams and forwarding them as D‑Bus signals.
- **Status Notifications** – Emitting events for process start, termination (with exit code or signal), and errors.
- **Monitoring** – Maintaining a registry of all active processes and providing the ability to list them.

---

## AppWrapper D‑Bus Interface

### Service Identification
- **Service Name:** `dev.flipper.appwrap`
- **Manager Object Path:** `/appwrap/manager`
- **Process Object Path:** `/appwrap/process/<id>`

### Manager Interface (`dev.flipper.AppWrapper.Manager`)

The Manager provides methods for controlling processes.

| Method | Description |
|--------|-------------|
| `Start(appId: string, args: array<string>, env: dict<string,string>, options: dict<variant>)` | Launches a new process. Returns the object path of the created process. |
| `Stop(path: object path)` | Stops the process identified by the given object path. |
| `SendInput(path: object path, data: string)` | Sends data to the `STDIN` of the process (works for both regular and PTY‑based interactive processes). |
| `List()` | Returns an array of object paths for all currently active processes. |

The `options` dictionary may contain the following fields:
| Field | Type | Description |
|-------|------|-------------|
| `cwd` | string | Working directory |
| `env` | object | Environment variables (key‑value) |
| `timeout` | number | Maximum execution time (seconds) |
| `interactive` | boolean | Use PTY (default: false) |
| `cols` | number | Terminal columns (for PTY) |
| `rows` | number | Terminal rows (for PTY) |
| `stopSignal` | string | Signal to send on stop (default: "SIGTERM") |

### Process Object Interface (`dev.flipper.AppWrapper.Process`)

Each running process exposes a D‑Bus object at `/appwrap/process/<id>` that emits the following signals:

| Signal | Description |
|--------|-------------|
| `Started(pid: uint32)` | Emitted immediately after the process has been successfully started. Provides the system PID. |
| `Output(stream: string, data: string)` | Emitted when new output is available. `stream` indicates `"stdout"` or `"stderr"`; `data` is the output chunk (typically line‑buffered). |
| `Exited(code: int32, signal: int32)` | Emitted when the process terminates. For normal exit, `code` holds the exit status and `signal` is `0`; when killed by a signal, `signal` holds the signal number (for PTY‑based processes, only `code` is used). |
| `Error(message: string)` | Emitted when an error occurs during process startup or execution (e.g., executable not found). |

### Client Interaction Model

- Clients (typically the `Model`) interact with the **Manager** to start, stop, and send input to processes.
- Each started process gets a unique object path, and the client can subscribe to signals on that object to receive output, termination, and error notifications.
- The `AppWrapper` maintains a registry of all active processes, which can be retrieved via the `List` method.

---

## Process Lifecycle (from AppWrapper Perspective)

1. **Receive Start Request**
   - Validate arguments.
   - Create a process object and generate a unique object path.

2. **Launch Child Process**
   - **Regular:** `spawn(cmd, args, { cwd, env, stdio: ['pipe', 'pipe', 'pipe'] })`
   - **Interactive:** `pty.spawn(cmd, args, { cwd, env, cols, rows })`

3. **Set Up Event Handlers**
   - For regular processes:
     - `stdout.on('data')`, `stderr.on('data')`
     - `child.on('close')`, `child.on('error')`
   - For PTY‑based processes:
     - `pty.on('data')`, `pty.on('exit')`

4. **Emit `Started` Signal**
   - Send the system PID to the client.

5. **Stream Redirection**
   - All data from `stdout` and `stderr` is converted into `Output` signals and forwarded to the client (Model).

6. **Handle Commands**
   - **`Stop`** – Sends a signal (default `SIGTERM`) to the process; can force‑terminate if needed.
   - **`SendInput`** – Writes data to `STDIN` (via `child.stdin.write()` for regular processes, or `pty.write()` for interactive ones).

7. **Process Termination**
   - Upon receiving `close` or `exit` events, the service emits an `Exited` signal with the exit code and/or signal number.
   - The process object remains alive for a short grace period (to ensure delivery of final signals), then is removed from memory.

8. **Error Handling**
   - On launch or runtime errors, an `Error` signal is emitted.

---

### Integration with the Model

- The `Model` calls `Start` with parameters derived from plugin definitions.
- The `Model` subscribes to `Output`, `Exited`, and `Error` signals for each process.
- The `Model` may use `SendInput` for interactive utilities (e.g., sending commands to a debugger).
- The `Model` is responsible for storing object paths and managing the overall lifecycle of processes.

![AppWrapper diagram](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/silart/flipctl/ui_arch/diagrams/app-wrapper.puml)
