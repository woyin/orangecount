#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Generate or check the pinned public query corpus, never a private ledger.

Generation is an explicit developer operation; Go gates only read reviewed
goldens. No ledger path argument is accepted.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from datetime import date
from decimal import Decimal
from pathlib import Path

import beancount
import beanquery
from beancount import loader
from beancount.core.amount import Amount
from beancount.core.position import Position
from beancount.core.inventory import Inventory
from beanquery import query, types

ROOT = Path(__file__).resolve().parents[2]
CORPORA = ("default-source", "compound-values", "price-weight", "inventory-order", "metadata-source", "metadata-stack", "metadata-pad", "dynamic-values", "casts", "compound-casts", "aggregates", "presets", "balance-stream", "balance-cost", "source-order", "aggregate-longtail", "standard-tables")
TYPE_NAMES = {"str": "string", "int": "integer", "decimal": "decimal", "bool": "bool", "date": "date", "object": "object", "amount": "amount", "position": "position", "inventory": "inventory", "NULL": "null"}


def decimal_text(value):
    if not value:
        return "0"
    text = format(value, "f")
    return text.rstrip("0").rstrip(".") if "." in text else text


def cell(value):
    if value is None:
        return {"type": "null"}
    if isinstance(value, bool):
        return {"type": "bool", "value": value}
    if isinstance(value, Decimal):
        return {"type": "decimal", "value": decimal_text(value)}
    if isinstance(value, int):
        return {"type": "integer", "value": str(value)}
    if isinstance(value, str):
        return {"type": "string", "value": value}
    if isinstance(value, date):
        return {"type": "date", "value": value.isoformat()}
    if isinstance(value, Amount):
        return {"type": "amount", "value": amount_value(value)}
    if isinstance(value, Position):
        return {"type": "position", "value": position_value(value)}
    if isinstance(value, Inventory):
        # Inventory iteration is explicitly unordered. Normalize only its
        # financial identity order; result row order and duplicates stay intact.
        return {"type": "inventory", "value": [position_value(p) for p in sorted(value, key=inventory_key)]}
    raise TypeError(f"Unregistered oracle result type: {type(value).__name__}")


def amount_value(value):
    return {"number": decimal_text(value.number), "currency": value.currency}


def position_value(value):
    cost = value.cost
    return {"units": amount_value(value.units), "cost": None if cost is None else {
        "number": decimal_text(cost.number), "currency": cost.currency,
        "date": cost.date.isoformat() if cost.date else None, "label": cost.label,
    }}


def inventory_key(value):
    cost = value.cost
    return (value.units.currency, cost is not None,
            cost.number if cost else Decimal(0), cost.currency if cost else "",
            bool(cost and cost.date), cost.date if cost and cost.date else date.min,
            bool(cost and cost.label is not None), cost.label if cost and cost.label is not None else "")


def generate(name):
    fixture = ROOT / f"testdata/fixtures/query/{name}.bean"
    queries = fixture.with_suffix(".queries.json")
    if (beancount.__version__, beanquery.__version__) != ("3.2.3", "0.2.0"):
        raise RuntimeError("Expected Beancount 3.2.3 / BeanQuery 0.2.0")
    entries, errors, options = loader.load_file(str(fixture))
    if errors:
        raise RuntimeError("Public query fixture must evaluate without errors")
    cases = []
    for item in json.loads(queries.read_text()):
        try:
            columns, rows = query.run_query(entries, options, item["query"])
        except Exception as error:
            expected = item.get("error")
            if not expected or type(error).__name__ != expected["class"]:
                raise RuntimeError(f"Unexpected oracle failure for {item['id']}") from error
            cases.append(item)
            continue
        if "error" in item:
            raise RuntimeError(f"Expected oracle error did not occur for {item['id']}")
        normalized = [[cell(value) for value in row] for row in rows]
        for row in normalized:
            for index in item.get("source_columns", []):
                if row[index]["type"] != "null":
                    row[index]["value"] = Path(row[index]["value"]).relative_to(fixture.parent).as_posix()
        cases.append({**item, "result": {
            "columns": [{"name": c.name, "type": TYPE_NAMES[types.name(c.datatype)]} for c in columns],
            "rows": normalized,
        }})
    sources = {}
    allowed = (ROOT / "testdata/fixtures/query").resolve()
    for source_name in options["include"]:
        path = Path(source_name).resolve()
        path.relative_to(allowed)  # Reject arbitrary/private include graphs.
        sources[path.relative_to(ROOT).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    return {"schema_version": 2, "oracle": {"beancount": beancount.__version__, "beanquery": beanquery.__version__},
            "fixture": fixture.relative_to(ROOT).as_posix(),
            "input_sha256": hashlib.sha256(fixture.read_bytes()).hexdigest(),
            "queries_sha256": hashlib.sha256(queries.read_bytes()).hexdigest(),
            "sources_sha256": dict(sorted(sources.items())), "cases": cases}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", choices=CORPORA, help="one registered public corpus; defaults to all")
    parser.add_argument("--output", type=Path, help="single-corpus output (defaults to default-source when --corpus is omitted)")
    parser.add_argument("--check", action="store_true", help="compare without changing any golden")
    args = parser.parse_args()
    names = (args.corpus,) if args.corpus else (("default-source",) if args.output else CORPORA)
    for name in names:
        result = generate(name)
        output = args.output or ROOT / f"testdata/golden/query-v1/{name}.json"
        if args.check:
            if not output.exists() or json.loads(output.read_text()) != result:
                raise SystemExit(f"Pinned query golden mismatch for {name}; explicit regeneration/review required")
        else:
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
        print(f"query oracle: {name}, {len(result['cases'])} cases, {'checked' if args.check else 'generated'}")


if __name__ == "__main__":
    main()
