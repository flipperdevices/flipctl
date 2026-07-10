from pathlib import Path

import pytest

from core.plugin_manager import PluginError, PluginManager


@pytest.fixture()
def manager() -> PluginManager:
    return PluginManager(Path(__file__).parents[1] / "plugins")


def test_lists_builtin_plugins(manager: PluginManager) -> None:
    names = {plugin["name"] for plugin in manager.list_plugins()}
    assert names == {"ping", "nmap"}


def test_applies_defaults(manager: PluginManager) -> None:
    spec = manager._plugins["ping"]
    validated = manager._validate_inputs(spec, {"target": "127.0.0.1"})
    assert validated == {"target": "127.0.0.1", "count": 4}


def test_rejects_unknown_inputs(manager: PluginManager) -> None:
    spec = manager._plugins["ping"]
    with pytest.raises(PluginError, match="Unknown input fields"):
        manager._validate_inputs(spec, {"target": "127.0.0.1", "extra": True})


def test_rejects_wrong_type(manager: PluginManager) -> None:
    spec = manager._plugins["ping"]
    with pytest.raises(PluginError, match="must be an integer"):
        manager._validate_inputs(spec, {"target": "127.0.0.1", "count": "4"})


def test_rejects_out_of_range_integer(manager: PluginManager) -> None:
    spec = manager._plugins["ping"]
    with pytest.raises(PluginError, match="exceeds maximum"):
        manager._validate_inputs(spec, {"target": "127.0.0.1", "count": 100})


def test_rejects_unsupported_nmap_profile(manager: PluginManager) -> None:
    spec = manager._plugins["nmap"]
    with pytest.raises(PluginError, match="must be one of"):
        manager._validate_inputs(
            spec,
            {"target": "127.0.0.1", "scan_type": "arbitrary-flags"},
        )
