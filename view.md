
## View Layer

The **View** is the user interface layer responsible for rendering the system state and handling user interaction. The architecture supports multiple View implementations (e.g., hardware display, web interface, terminal‑based TUI), all of which follow a unified communication protocol with the `Model` (`modeld`) via D‑Bus.


