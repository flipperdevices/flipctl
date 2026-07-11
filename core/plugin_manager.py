"""FlipCTL plugin discovery, validation, and bounded subprocess execution."""

from __future__ import annotations

import json
import logging
import os
import re
import shlex
import signal
import subprocess
import sys
import threading
from pathlib import Path
from typing import Any, BinaryIO

import yaml

PLUGINS_DIR = Path(__file__).parent.parent / "plugins"
SUPPORTED_INPUT_TYPES = {"string", "