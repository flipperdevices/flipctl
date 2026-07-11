#!/usr/bin/env python3
"""
FlipCTL TUI — powered by Textual.
Connects to the FlipCTL backend at http://localhost:8000 and exposes
the same plugin execution capability as the Web UI.
"""
import json
from typing import Any

import httpx
from textual import on, work
from textual.app import App, ComposeResult
from textual.containers import Horizontal, Vertical
from textual.widgets import Button, Footer, Header, Input, Label, RichLog, Select, Static

BASE_URL = "http://localhost:8000"


class FlipCTLApp(App):
    CSS = """
    Screen { background: #0d0d0d; }

    #sidebar {
        width: 30;
        border-right: solid #333;
        padding: 1 2;
    }

    #main-panel { padding: 1 2; }

    Label { color: #888; margin-bottom: 1; }

    Select { margin-bottom: 1; }

    #plugin-desc {
        color: #555;
        margin-bottom: 1;
        text-style: italic;
    }

    #inputs-container { margin-bottom: 1; }
    #inputs-container Input { margin-bottom: 1; }

    Button {
        background: #ff8c00;
        color: #000;
        border: none;
    }
    Button:hover { background: #ffa533; }
    Button:disabled { opacity: 0.4; }

    #output-log {
        border: solid #222;
        height: 1fr;
        margin-top: 1;
        background: #111;
    }
    """

    TITLE = "FlipCTL"
    SUB_TITLE = "Plugin Runner — TUI"

    def __init__(self):
        super().__init__()
        self._plugins: list[dict] = []
        self._current_plugin: dict | None = None
        self._input_widgets: dict[str, Input] = {}

    def compose(self) -> ComposeResult:
        yield Header()
        with Horizontal():
            with Vertical(id="sidebar"):
                yield Label("Plugin")
                yield Select([], id="plugin-select", prompt="Loading…")
                yield Static("", id="plugin-desc")
                yield Label("Inputs", id="inputs-label")
                yield Vertical(id="inputs-container")
                yield Button("Run", id="run-btn", disabled=True)
            with Vertical(id="main-panel"):
                yield Label("Output")
                yield RichLog(id="output-log", highlight=True, markup=True)
        yield Footer()

    def on_mount(self) -> None:
        self.load_plugins()

    @work(exclusive=True)
    async def load_plugins(self) -> None:
        log = self.query_one("#output-log", RichLog)
        try:
            async with httpx.AsyncClient() as client:
                resp = await client.get(f"{BASE_URL}/api/plugins", timeout=5)
                resp.raise_for_status()
                self._plugins = resp.json()
        except Exception as exc:
            log.write(f"[red]Failed to load plugins: {exc}[/red]")
            return

        sel = self.query_one("#plugin-select", Select)
        sel.set_options([(p["name"], p["name"]) for p in self._plugins])
        if self._plugins:
            sel.value = self._plugins[0]["name"]

    @on(Select.Changed, "#plugin-select")
    def on_plugin_selected(self, event: Select.Changed) -> None:
        plugin = next((p for p in self._plugins if p["name"] == event.value), None)
        if not plugin:
            return
        self._current_plugin = plugin
        self.query_one("#plugin-desc", Static).update(plugin.get("description", ""))
        self._render_inputs(plugin)
        self.query_one("#run-btn", Button).disabled = False

    def _render_inputs(self, plugin: dict) -> None:
        container = self.query_one("#inputs-container", Vertical)
        container.remove_children()
        self._input_widgets = {}
        for field in plugin.get("inputs", []):
            label = Label(
                f"{field['name']}{'*' if field.get('required') else ''} — {field.get('description', '')}"
            )
            inp = Input(
                placeholder=str(field.get("default", "")),
                id=f"inp-{field['name']}",
            )
            if field.get("default") is not None:
                inp.value = str(field["default"])
            self._input_widgets[field["name"]] = inp
            container.mount(label, inp)

    def _collect_inputs(self) -> dict[str, Any]:
        if not self._current_plugin:
            return {}
        result = {}
        for field in self._current_plugin.get("inputs", []):
            widget = self._input_widgets.get(field["name"])
            val = widget.value.strip() if widget else ""
            result[field["name"]] = int(val) if field.get("type") == "integer" and val else val
        return result

    @on(Button.Pressed, "#run-btn")
    def on_run(self, _event: Button.Pressed) -> None:
        self.execute_plugin()

    @work(exclusive=True)
    async def execute_plugin(self) -> None:
        if not self._current_plugin:
            return
        btn = self.query_one("#run-btn", Button)
        log = self.query_one("#output-log", RichLog)
        btn.disabled = True
        log.clear()
        log.write(f"[yellow]Running {self._current_plugin['name']}…[/yellow]")

        payload = {"plugin": self._current_plugin["name"], "inputs": self._collect_inputs()}
        try:
            async with httpx.AsyncClient() as client:
                resp = await client.post(
                    f"{BASE_URL}/api/execute",
                    json=payload,
                    timeout=self._current_plugin.get("timeout", 30) + 5,
                )
                data = resp.json()
                log.write(json.dumps(data, indent=2))
        except Exception as exc:
            log.write(f"[red]Error: {exc}[/red]")
        finally:
            btn.disabled = False


if __name__ == "__main__":
    FlipCTLApp().run()
