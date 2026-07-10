#!/usr/bin/env python3
"""Constrained nmap wrapper for FlipCTL."""

from __future__ import annotations

import ipaddress
import json
import re
import shutil
import subprocess
import sys

SCAN_PROFILES = {
    "fast": ["-F"],
    "service": ["-sV", "--version-light", "-F"],
}
HOSTNAME_RE = re.compile(
    r"^(?=.{1,253}$)(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)*"
    r"[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$"
)


def validate_target(target: str) -> str:
    try:
        return str(ipaddress.ip_address(target))
    except ValueError:
        if HOSTNAME_RE.fullmatch(target):
            return target
        raise ValueError("target must be a valid IP address or hostname")


def parse_ports(raw: str) -> list[dict[str, object]]:
    ports: list[dict[str, object]] = []
    for line in raw.splitlines():
        match = re.match(r"(\d+)/(tcp|udp)\s+open\s+(\S+)", line.strip())
        if match:
            ports.append(
                {
                    "port": int(match.group(1)),
                    "protocol": match.group(2),
                    "service": match.group(3),
                }
            )
    return ports


def run(target: str, scan_type: str = "fast") -> dict[str, object]:
    target = validate_target(target)
    flags = SCAN_PROFILES.get(scan_type)
    if flags is None:
        raise ValueError(f"unsupported scan type: {scan_type}")

    if not shutil.which("nmap"):
        return {
            "success": True,
            "target": target,
            "stubbed": True,
            "open_ports": [],
            "raw_output": "[nmap not installed; scan skipped]",
        }

    command = ["nmap", *flags, "--", target]
    try:
        result = subprocess.run(
            command,
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return {
            "success": False,
            "target": target,
            "error": "timeout",
            "open_ports": [],
            "raw_output": "",
        }

    raw = result.stdout + result.stderr
    return {
        "success": result.returncode == 0,
        "target": target,
        "stubbed": False,
        "open_ports": parse_ports(raw),
        "raw_output": raw.strip(),
    }


if __name__ == "__main__":
    try:
        inputs = json.loads(sys.stdin.read())
        print(json.dumps(run(inputs["target"], inputs.get("scan_type", "fast"))))
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as exc:
        print(json.dumps({"success": False, "error": str(exc)}))
        raise SystemExit(2)
