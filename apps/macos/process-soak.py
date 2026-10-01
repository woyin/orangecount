#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.

"""Observe app RSS during repeated edits to a disposable single-file ledger.

This records process evidence, not GUI correctness, build counts, or frame time.
The leaks report is diagnostic output; exit code 0 does not mean leak-free.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import tempfile
import time


def main():
    root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("ledger", type=Path, help="single-file synthetic ledger; the original is never edited")
    parser.add_argument("--cycles", type=int, default=60, help="one edit every 10 seconds (default: 60)")
    args = parser.parse_args()
    if args.cycles < 1:
        parser.error("--cycles must be positive")
    app = root / "build/OrangeCount-QA.app/Contents/MacOS/OrangeCount"
    if not app.is_file():
        parser.error("run make app and sh apps/macos/prepare-qa.sh first")
    source = args.ledger.read_text()
    if re.search(r"^\s*include\s", source, re.MULTILINE):
        parser.error("use a single-file fixture without includes")
    output_root = root / "build/macos-baseline"
    output_root.mkdir(parents=True, exist_ok=True)
    folder = Path(tempfile.mkdtemp(prefix="app-soak.", dir=output_root))
    ledger = folder / "main.bean"
    ledger.write_text(source)
    report = folder / "process.jsonl"
    metadata = {
        "app_sha256": hashlib.sha256(app.read_bytes()).hexdigest(),
        "input_sha256": hashlib.sha256(source.encode()).hexdigest(),
        "system": platform.platform(),
        "cycles": args.cycles,
        "interval_seconds": 10,
        "fixture": str(ledger),
        "boundary": "process RSS during external file changes; no GUI assertions",
    }
    with (folder / "app.log").open("w") as log, report.open("w") as output:
        child = subprocess.Popen([str(app), str(ledger)],
                                 env=dict(os.environ, ORANGECOUNT_TEST_MODE="1"),
                                 stdout=log, stderr=log)
        started = time.monotonic()

        def record(item, show=False):
            output.write(json.dumps(item) + "\n")
            output.flush()
            if show:
                print(json.dumps(item), flush=True)

        record(dict(metadata, pid=child.pid), show=True)
        try:
            for cycle in range(args.cycles + 1):
                if child.poll() is not None:
                    raise RuntimeError(f"app exited before observation finished: {child.returncode}")
                rss = subprocess.check_output(["ps", "-p", str(child.pid), "-o", "rss="], text=True)
                record({"cycle": cycle, "elapsed_seconds": round(time.monotonic() - started, 3),
                        "rss_kib": int(rss.strip())}, show=cycle % 5 == 0 or cycle == args.cycles)
                if cycle == args.cycles:
                    break
                ledger.write_text(f"; process soak cycle {cycle + 1}\n" + source)
                deadline = started + (cycle + 1) * 10
                time.sleep(max(0, deadline - time.monotonic()))
            with (folder / "leaks.txt").open("w") as leak_output:
                result = subprocess.run(["leaks", "--noContent", "--nostacks", str(child.pid)],
                                        stdout=leak_output, stderr=log)
            record({"leaks_exit_code": result.returncode,
                    "note": "interpret against a framework baseline; not a pass/fail verdict"})
        finally:
            child.terminate()
            try:
                child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
            record({"app_exit_code": child.returncode, "termination": "probe cleanup, not UI Quit"})
    print(f"Process observation saved: {report}", flush=True)


if __name__ == "__main__":
    main()
