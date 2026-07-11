#!/usr/bin/env python3
"""
FlipCTL ping plugin.
Reads inputs from stdin as JSON, writes result to stdout as JSON.
"""
import json
import platform
import re
import subprocess
import sys


def run(target: str, count: int = 4) -> dict:
    flag = "-n" if platform.system().lower() == "windows" else "-c"
    cmd = ["ping", flag, str(count), target]

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30,
        )
        raw = result.stdout + result.stderr
        success = result.returncode == 0

        packets_sent = count
        packets_received = 0
        avg_ms = None

        if success:
            recv_match = re.search(r"(\d+) (?:packets )?received", raw)
            if recv_match:
                packets_received = int(recv_match.group(1))

            avg_match = re.search(r"(?:avg|rtt)[^=]+=\s*[\d.]+/([\d.]+)", raw)
            if not avg_match:
                avg_match = re.search(r"Average = ([\d.]+)ms", raw)
            if avg_match:
                avg_ms = float(avg_match.group(1))

        return {
            "success": success,
            "target": target,
            "packets_sent": packets_sent,
            "packets_received": packets_received,
            "avg_ms": avg_ms,
            "raw_output": raw.strip(),
        }

    except subprocess.TimeoutExpired:
        return {"success": False, "target": target, "error": "timeout", "raw_output": ""}
    except FileNotFoundError:
        return {"success": False, "target": target, "error": "ping not found", "raw_output": ""}


if __name__ == "__main__":
    inputs = json.loads(sys.stdin.read())
    output = run(
        target=inputs["target"],
        count=int(inputs.get("count", 4)),
    )
    print(json.dumps(output))
