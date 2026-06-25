
## AppWrapper Service (`app_wrapperd`)

`AppWrapper` is a service responsible for launching, managing, and monitoring arbitrary console utilities and applications. It acts as the execution engine for the `Model` (`modeld`), providing a unified interface for handling child processes regardless of their nature—whether they are one‑off tasks, long‑running daemons, or interactive sessions. The service is fully asynchronous and built upon Node.js' event‑driven model.


