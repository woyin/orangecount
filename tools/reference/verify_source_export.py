#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Public-only, pinned independent source-export roundtrip acceptance."""
import ctypes
import json
import os
import sys
import subprocess
import tempfile
from collections import defaultdict
from pathlib import Path
import beancount
from beancount import loader
from beancount.core import data
from beancount.core.inventory import Inventory
from generate_full_golden import entry, inventory

REPO = Path(__file__).resolve().parents[2]
# These source fixtures intentionally depend on an include outside their entry
# directory. F04 refuses remapping; invalid and AVERAGE cases are separate gates.
EXTERNAL = {"metadata-pad.bean", "metadata-stack.bean"}

def state(path):
    entries, errors, _ = loader.load_file(str(path))
    if errors:
        raise AssertionError([type(error).__name__ for error in errors])
    encoded = []
    balances = defaultdict(Inventory)
    checkpoints = []
    for value in entries:
        normal = entry(value)
        normal.pop("source", None)
        encoded.append(normal)
        if isinstance(value, data.Transaction):
            for posting in value.postings:
                balances[posting.account].add_position(posting)
            # Every transaction prefix, rather than only final account totals.
            checkpoints.append({key: inventory(v) for key, v in sorted(balances.items())})
    return {"entries": encoded, "checkpoints": checkpoints,
            "final": {key: inventory(v) for key, v in sorted(balances.items())}}

def main():
    assert beancount.__version__ == "3.2.3"
    accepted = rejected = checkpoints = 0
    with tempfile.TemporaryDirectory(prefix="orangecount-public-export-") as directory:
        work = Path(directory)
        binary = work / "orangecount"
        subprocess.run(["go", "build", "-o", str(binary), "./cmd/orangecount"], cwd=REPO, check=True)
        for folder in ["v3-parity", "semantics"]:
            for path in sorted((REPO / "testdata/fixtures" / folder).glob("*.bean")):
                golden = REPO / "testdata/golden/v3-full" / (path.stem + ".golden.json")
                if not golden.exists() or not json.loads(golden.read_text())["valid"]:
                    continue
                out_parent = work / f"{folder}-{path.stem}"
                out_parent.mkdir()
                multi = 'include "' in path.read_text()
                target = out_parent / ("tree" if multi else "export.bean")
                result = subprocess.run([str(binary), "export", "-semantics", "v3", "-out", str(target), str(path)], cwd=REPO, text=True, capture_output=True)
                if path.name == "booking-average.bean":
                    assert result.returncode != 0 and "AVERAGE" in result.stderr
                    assert not target.exists()
                    rejected += 1
                    continue
                if path.name in EXTERNAL:
                    assert result.returncode != 0 and "E-EXPORT-INCLUDE" in result.stderr, (path.name, result.stderr)
                    assert not target.exists()
                    rejected += 1
                    continue
                assert result.returncode == 0, (path.name, result.stderr)
                original = state(path)
                actual = state(target / path.name if multi else target)
                assert original == actual, path.name
                accepted += 1
                checkpoints += len(original["checkpoints"])
        for path, expected in [
            (REPO / "testdata/fixtures/query/metadata-pad.bean", None),
            (REPO / "testdata/fixtures/query/metadata-stack.bean", None),
            (REPO / "testdata/fixtures/source-export/dialect.bean", REPO / "testdata/fixtures/source-export/dialect.expected.bean"),
        ]:
            target = work / ("extra-" + path.stem)
            result = subprocess.run([str(binary), "export", "-semantics", "v3", "-out", str(target), str(path)], cwd=REPO, text=True, capture_output=True)
            assert result.returncode == 0, (path.name, result.stderr)
            output = target / path.name if 'include "' in path.read_text() else target
            original, actual = state(expected or path), state(output)
            assert original == actual, path.name
            accepted += 1
            checkpoints += len(original["checkpoints"])
        if sys.platform == "darwin":
            target = work / "finder-alias.bean"
            target.write_text("alias target stays unchanged")
            finder_info = bytearray(32)
            finder_info[8] = 0x80
            libc = ctypes.CDLL(None, use_errno=True)
            libc.setxattr.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_void_p, ctypes.c_size_t, ctypes.c_uint32, ctypes.c_int]
            buffer = ctypes.create_string_buffer(bytes(finder_info))
            assert libc.setxattr(os.fsencode(target), b"com.apple.FinderInfo", buffer, 32, 0, 0) == 0
            result = subprocess.run([str(binary), "export", "-semantics", "v3", "-out", str(target), str(REPO / "testdata/fixtures/v3-parity/txn-basic.bean")], cwd=REPO, text=True, capture_output=True)
            assert result.returncode != 0 and "Finder alias" in result.stderr
            assert target.read_text() == "alias target stays unchanged"
            rejected += 1
        for name in ["global-average.bean", "average-explicit.bean"]:
            path = REPO / "testdata/fixtures/semantics" / name
            target = work / f"reject-{name}"
            target.write_text("old target")
            result = subprocess.run([str(binary), "export", "-semantics", "v3", "-out", str(target), str(path)], cwd=REPO, text=True, capture_output=True)
            assert result.returncode != 0 and "AVERAGE" in result.stderr
            assert target.read_text() == "old target"
            rejected += 1
    print(json.dumps({"oracle": "beancount-3.2.3", "accepted_roundtrips": accepted,
                      "transaction_inventory_checkpoints": checkpoints, "explicit_refusals": rejected}, sort_keys=True))

if __name__ == "__main__":
    main()
