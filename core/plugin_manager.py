"""FlipCTL plugin discovery, validation, and subprocess execution."""

from __future__ import annotations

import json
import os
import shlex
import signal
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any

import yaml

PLUGINS_DIR = Path(__file__).parent.parent / "plugins"
SUPPORTED_INPUT_TYPES = {"string", "integer", "boolean", "enum"}
DEFAULT_MAX_OUTPUT_BYTES = 1_000_000


class PluginError(Exception):
    """Raised when a plugin definition or execution is invalid."""

    def __init__(self, message: str, code: str = "plugin_error") -> None:
        super().__init__(message)
        self.code = code


class PluginManager:
    def __init__(
        self,
        plugins_dir: Path = PLUGINS_DIR,
        max_output_bytes: int = DEFAULT_MAX_OUTPUT_BYTES,
    ) -> None:
        self.plugins_dir = plugins_dir
        self.max_output_bytes = max_output_bytes
        self._plugins: dict[str, dict[str, Any]] = {}
        self._load_all()

    def _load_all(self) -> None:
        if not self.plugins_dir.exists():
            return
        for entry in sorted(self.plugins_dir.iterdir()):
            spec_path = entry / "plugin.yaml"
            if entry.is_dir() and spec_path.exists():
                try:
                    self._plugins[entry.name] = self._load_spec(entry, spec_path)
                except Exception as exc:
                    print(f"[plugin_manager] skipping {entry.name}: {exc}", file=sys.stderr)

    def _load_spec(self, plugin_dir: Path, spec_path: Path) -> dict[str, Any]:
        with spec_path.open(encoding="utf-8") as handle:
            spec = yaml.safe_load(handle)

        if not isinstance(spec, dict):
            raise ValueError("plugin.yaml must contain a mapping")

        required_keys = {"name", "description", "version", "inputs", "command", "timeout"}
        missing = required_keys - spec.keys()
        if missing:
            raise ValueError(f"plugin.yaml missing keys: {sorted(missing)}")
        if spec["name"] != plugin_dir.name:
            raise ValueError("plugin name must match its directory name")
        if not isinstance(spec["command"], str) or not spec["command"].strip():
            raise ValueError("command must be a non-empty string")
        if not isinstance(spec["timeout"], int) or not 1 <= spec["timeout"] <= 300:
            raise ValueError("timeout must be an integer between 1 and 300 seconds")
        if not isinstance(spec["inputs"], list):
            raise ValueError("inputs must be a list")

        seen: set[str] = set()
        for field in spec["inputs"]:
            if not isinstance(field, dict):
                raise ValueError("each input must be a mapping")
            name = field.get("name")
            field_type = field.get("type")
            if not isinstance(name, str) or not name:
                raise ValueError("each input requires a non-empty name")
            if name in seen:
                raise ValueError(f"duplicate input name: {name}")
            seen.add(name)
            if field_type not in SUPPORTED_INPUT_TYPES:
                raise ValueError(f"unsupported input type for {name}: {field_type}")
            if field_type == "enum":
                values = field.get("values")
                if not isinstance(values, list) or not values:
                    raise ValueError(f"enum input {name} requires non-empty values")

        spec["_dir"] = plugin_dir
        return spec

    def list_plugins(self) -> list[dict[str, Any]]:
        return [
            {
                "name": spec["name"],
                "description": spec["description"],
                "version": spec["version"],
                "inputs": spec["inputs"],
            }
            for spec in self._plugins.values()
        ]

    def execute(self, plugin_name: str, inputs: dict[str, Any]) -> dict[str, Any]:
        spec = self._plugins.get(plugin_name)
        if spec is None:
            raise PluginError(f"Unknown plugin: {plugin_name!r}", "unknown_plugin")

        validated_inputs = self._validate_inputs(spec, inputs)
        plugin_dir: Path = spec["_dir"]
        parts = shlex.split(spec["command"])
        if not parts:
            raise PluginError("Plugin command is empty", "invalid_plugin")
        if parts[0] in ("python", "python3"):
            parts[0] = sys.executable

        with tempfile.TemporaryFile() as stdout_file, tempfile.TemporaryFile() as stderr_file:
            try:
                process = subprocess.Popen(
                    parts,
                    stdin=subprocess.PIPE,
                    stdout=stdout_file,
                    stderr=stderr_file,
                    text=True,
                    cwd=plugin_dir,
                    start_new_session=(os.name != "nt"),
                    env=self._clean_environment(),
                )
                process.communicate(json.dumps(validated_inputs), timeout=spec["timeout"])
            except subprocess.TimeoutExpired:
                self._terminate_process_tree(process)
                raise PluginError(
                    f"Plugin {plugin_name!r} timed out after {spec['timeout']}s",
                    "timeout",
                )
            except FileNotFoundError as exc:
                raise PluginError(f"Plugin command not found: {exc}", "command_not_found")

            stdout = self._read_bounded(stdout_file, "stdout")
            stderr = self._read_bounded(stderr_file, "stderr")

        if process.returncode != 0:
            raise PluginError(
                f"Plugin {plugin_name!r} exited {process.returncode}: {stderr.strip()}",
                "execution_failed",
            )
        if not stdout.strip():
            raise PluginError(f"Plugin {plugin_name!r} returned empty output", "invalid_output")

        try:
            result = json.loads(stdout)
        except json.JSONDecodeError as exc:
            raise PluginError(
                f"Plugin {plugin_name!r} returned invalid JSON: {exc}",
                "invalid_output",
            )
        if not isinstance(result, dict):
            raise PluginError("Plugin output must be a JSON object", "invalid_output")
        return result

    def _validate_inputs(
        self, spec: dict[str, Any], inputs: dict[str, Any]
    ) -> dict[str, Any]:
        if not isinstance(inputs, dict):
            raise PluginError("inputs must be a JSON object", "invalid_inputs")

        fields = {field["name"]: field for field in spec["inputs"]}
        unknown = set(inputs) - set(fields)
        if unknown:
            raise PluginError(
                f"Unknown input fields: {sorted(unknown)}", "invalid_inputs"
            )

        validated: dict[str, Any] = {}
        for name, field in fields.items():
            if name in inputs:
                value = inputs[name]
            elif "default" in field:
                value = field["default"]
            elif field.get("required"):
                raise PluginError(f"Missing required input: {name!r}", "invalid_inputs")
            else:
                continue

            expected = field["type"]
            if expected == "string" and not isinstance(value, str):
                raise PluginError(f"Input {name!r} must be a string", "invalid_inputs")
            if expected == "integer" and (not isinstance(value, int) or isinstance(value, bool)):
                raise PluginError(f"Input {name!r} must be an integer", "invalid_inputs")
            if expected == "boolean" and not isinstance(value, bool):
                raise PluginError(f"Input {name!r} must be a boolean", "invalid_inputs")
            if expected == "enum" and value not in field["values"]:
                raise PluginError(
                    f"Input {name!r} must be one of {field['values']}", "invalid_inputs"
                )

            if isinstance(value, str) and len(value) > int(field.get("max_length", 255)):
                raise PluginError(f"Input {name!r} is too long", "invalid_inputs")
            if isinstance(value, int):
                if "minimum" in field and value < field["minimum"]:
                    raise PluginError(f"Input {name!r} is below minimum", "invalid_inputs")
                if "maximum" in field and value > field["maximum"]:
                    raise PluginError(f"Input {name!r} exceeds maximum", "invalid_inputs")
            validated[name] = value
        return validated

    def _read_bounded(self, file_obj: Any, stream_name: str) -> str:
        size = file_obj.tell()
        if size > self.max_output_bytes:
            raise PluginError(
                f"Plugin {stream_name} exceeded {self.max_output_bytes} bytes",
                "output_too_large",
            )
        file_obj.seek(0)
        return file_obj.read().decode("utf-8", errors="replace")

    @staticmethod
    def _clean_environment() -> dict[str, str]:
        allowed = {"PATH", "LANG", "LC_ALL", "HOME", "TMPDIR", "SYSTEMROOT", "WINDIR"}
        return {key: value for key, value in os.environ.items() if key in allowed}

    @staticmethod
    def _terminate_process_tree(process: subprocess.Popen[str]) -> None:
        if process.poll() is not None:
            return
        if os.name == "nt":
            process.kill()
        else:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
