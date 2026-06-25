# Model Service (modeld)

The Model is the central management component, implementing the Model‑View‑Presenter (MVP) pattern. In this design, it combines the roles of both Presenter and Model — it holds the application state while also orchestrating user interactions and coordinating business logic.

The service acts as the single datasource for all View implementations (HW, Web, TUI) and drives task execution through the AppWrapper.


