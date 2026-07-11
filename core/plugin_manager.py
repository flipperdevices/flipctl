"""Plugin discovery, validation, and bounded subprocess execution."""
from __future__ import annotations

import copy
import json
import logging
import os
import re
import shlex
import signal
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any

import yaml

PLUGINS_DIR = Path(__file__).parent.parent /