
## AppWrapper Service (`app_wrapperd`)

`AppWrapper` is a service responsible for launching, managing, and monitoring arbitrary console utilities and applications. It acts as the execution engine for the `Model` (`modeld`), providing a unified interface for handling child processes regardless of their nature—whether they are one‑off tasks, long‑running daemons, or interactive sessions. The service is fully asynchronous and built upon Node.js' event‑driven model.

## Key Concepts

### Process
- Each running instance of an application is identified by a unique D‑Bus object path (e.g., `/appwrap/process/<id>`).
- For each process, a separate D‑Bus object is created that implements the `dev.flipper.Process` interface. Through this interface, the client (Model) can receive signals about output and termination.
- Processes can be of two types:
  - **Regular**: launched via `child_process.spawn`, with I/O through standard streams.
  - **Interactive**: uses `node-pty` to provide a full‑featured terminal.

### Client Interface
- The Manager (at `/appwrap/manager`) provides methods for process management.
- The Process object (at `/appwrap/process/<id>`) provides signals for receiving data.

