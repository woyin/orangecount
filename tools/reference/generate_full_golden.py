#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Pinned, public-only oracle for the schema-2 full accounting comparison.

Preserves ordered postings, exact decimal strings, typed user metadata, flags,
cost dates/labels, and entry multiplicities. Never reads a private ledger.
"""
from __future__ import annotations

import json
from collections import defaultdict
from datetime import date
from decimal import Decimal
from pathlib import Path

import beancount
from beancount import loader
from beancount.core import data
from beancount.core.inventory import Inventory
from generate_golden import norm_entry, norm_num

REPO = Path(__file__).resolve().parents[2]
DEST = REPO / "testdata/golden/v3-full"
EXTENSIONS = {"average-explicit.bean", "global-average.bean"}
RESERVED_META = {"filename", "lineno", "__automatic__", "__tolerances__"}


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def amount(value):
    return None if value is None else {"number": norm_num(value.number), "currency": value.currency}


def cost(value):
    if value is None:
        return None
    return {"number": norm_num(value.number), "currency": value.currency,
            "date": value.date.isoformat() if value.date else None, "label": value.label}


def typed(value, dtype=None):
    if value is None:
        kind, result = "null", None
    elif isinstance(value, bool):
        kind, result = "bool", value
    elif isinstance(value, Decimal):
        kind, result = "number", norm_num(value)
    elif isinstance(value, date):
        kind, result = "date", value.isoformat()
    elif isinstance(value, data.Amount):
        kind, result = "amount", amount(value)
    elif dtype == "<AccountDummy>":
        kind, result = "account", value
    elif dtype is not None and getattr(dtype, "__name__", "") in {"Account", "Currency"}:
        kind, result = dtype.__name__.lower(), value
    elif isinstance(value, str):
        kind, result = "string", value
    else:
        raise TypeError(f"Unsupported oracle metadata type: {type(value).__name__}")
    return {"type": kind, "value": result}


def metadata(meta):
    return {k: typed(v) for k, v in (meta or {}).items() if k not in RESERVED_META}


def entry(value):
    # norm_entry's Open booking enum is deliberately replaced before JSON.
    result = norm_entry(value)
    if isinstance(value, data.Open):
        result["booking"] = value.booking.name if value.booking else None
    if isinstance(value, data.Transaction):
        result["postings"] = [
            {"account": p.account, "units": amount(p.units), "cost": cost(p.cost),
             "price": amount(p.price), "flag": p.flag, "metadata": metadata(p.meta)}
            for p in value.postings
        ]
    if isinstance(value, data.Custom):
        result["values"] = [typed(v.value, v.dtype) for v in value.values]
    result["metadata"] = metadata(value.meta)
    result["source"] = {"file": Path(value.meta["filename"]).name, "line": value.meta["lineno"]}
    return result


def inventory(value):
    currencies = defaultdict(Decimal)
    lots = []
    for p in value:
        if p.cost is None:
            currencies[p.units.currency] += p.units.number
        else:
            lots.append(encode({"currency": p.units.currency, "number": norm_num(p.units.number), "cost": cost(p.cost)}))
    return {"currencies": {k: norm_num(v) for k, v in sorted(currencies.items()) if v}, "lots": sorted(lots)}


def generate(path):
    entries, errors, _ = loader.load_file(str(path))
    balances = defaultdict(Inventory)
    if not errors:
        for value in entries:
            if isinstance(value, data.Transaction):
                for p in value.postings:
                    balances[p.account].add_position(p)
    return {"schema_version": 2, "fixture": path.name,
            "oracle_engine": "beancount-python-" + beancount.__version__, "valid": not errors,
            "errors": [{"type": type(e).__name__, "line": e.source.get("lineno")} for e in errors],
            "state": {"entries": [] if errors else sorted(encode(entry(e)) for e in entries),
                      "balances": {} if errors else {k: inventory(v) for k, v in sorted(balances.items())}}}


def main():
    if beancount.__version__ != "3.2.3":
        raise RuntimeError("Expected pinned Beancount 3.2.3")
    DEST.mkdir(parents=True, exist_ok=True)
    for directory in ["v3-parity", "semantics"]:
        for path in sorted((REPO / "testdata/fixtures" / directory).glob("*.bean")):
            if path.name in EXTENSIONS:
                continue
            result = generate(path)
            (DEST / (path.stem + ".golden.json")).write_text(json.dumps(result, sort_keys=True, indent=2, ensure_ascii=False) + "\n")
            print(f"{path.name}: {'valid' if result['valid'] else 'invalid'}")


if __name__ == "__main__":
    main()
