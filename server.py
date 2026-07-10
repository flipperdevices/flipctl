"""FlipCTL backend server shared by the Web UI and TUI."""

from __future__ import annotations

import asyncio
import os
from pathlib import Path
from typing import Any

from fastapi import FastAPI, Request
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, ConfigDict, Field

from core.plugin_manager import PluginError, PluginManager

app = FastAPI(title="FlipCTL", version="0.1.0")

WEB_DIR = Path(__file__).parent / "web"
plugin_manager = PluginManager()
MAX_CONCURRENT_EXECUTIONS = int(os.getenv("FLIPCTL_MAX_CONCURRENCY", "4"))
execution_slots = asyncio.Semaphore(MAX_CONCURRENT_EXECUTIONS)


class ExecuteRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    plugin: str = Field(min_length=1, max_length=64)
    inputs: dict[str, Any]


@app.exception_handler(PluginError)
async def plugin_error_handler(_: Request, exc: PluginError) -> JSONResponse:
    return JSONResponse(
        status_code=400,
        content={"error": {"code": exc.code, "message": str(exc)}},
    )


@app.get("/api/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/plugins")
async def list_plugins() -> list[dict[str, Any]]:
    return plugin_manager.list_plugins()


@app.post("/api/execute")
async def execute_plugin(req: ExecuteRequest) -> dict[str, Any]:
    async with execution_slots:
        return await asyncio.to_thread(plugin_manager.execute, req.plugin, req.inputs)


@app.get("/")
async def serve_ui() -> FileResponse | JSONResponse:
    index = WEB_DIR / "index.html"
    if not index.exists():
        return JSONResponse(
            status_code=404,
            content={"error": {"code": "ui_not_found", "message": "Web UI not found"}},
        )
    return FileResponse(index)


if WEB_DIR.exists():
    app.mount("/static", StaticFiles(directory=WEB_DIR), name="static")
