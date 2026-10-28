#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Pinned-oracle side of the generative differential (tools/genprediff).

For every generated *.bean fixture this emits the schema-2 full golden that
generate_full_golden.py already defines, plus the extra dimensions the
generative differential requires and the schema-2 golden cannot express:

  ordered_entries     the normalized entries in source order, so reordering is
                      visible (state.entries is a sorted multiset);
  prefix_inventories  the full account inventory after every transaction, not
                      only the final balances;
  prices              every price directive as (date, base, quote, rate);
  period_income       income/expense posting totals per month and currency.

Both sides of the differential are computed the same way so a difference points
at the engine, never at the harness. This script never reads a private ledger.
"""
from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict
from datetime import date
from decimal import Decimal
from pathlib import Path

import beancount
from beancount import loader
from beancount.core import data
from beancount.core.inventory import Inventory
from generate_full_golden import cost, encode, generate, inventory, norm_num

INCOME_ROOTS = ("Income", "Expenses")


def ordered_entries(entries):
    """Normalized entries in source-line order.

    Beancount returns the loaded entry stream already in evaluation order; each
    entry keeps its own source line, and a single-file fixture gives every entry
    a distinct line, so ordering by line is the unambiguous source order.
    """
    from generate_full_golden import entry as normalize_entry

    pairs = sorted(
        (value.meta.get("lineno", 0), encode(normalize_entry(value)))
        for value in entries
    )
    return [encoded for _, encoded in pairs]


def prefix_inventories(entries):
    """Net units per account and currency after each transaction, in entry order.

    Lot detail is deliberately left to the final-balances dimension, which
    compares it exactly; here the point is the *timeline*: a booking error that
    nets out later still shows up in the prefix it happened in.
    """
    totals = defaultdict(lambda: defaultdict(Decimal))
    prefixes = []
    for value in entries:
        if not isinstance(value, data.Transaction):
            continue
        for posting in value.postings:
            if posting.units is None or not posting.units.currency:
                continue
            totals[posting.account][posting.units.currency] += posting.units.number
        snapshot = {}
        for account in sorted(totals):
            recorded = {
                currency: norm_num(amount)
                for currency, amount in sorted(totals[account].items())
                if amount
            }
            if recorded:
                snapshot[account] = recorded
        prefixes.append(snapshot)
    return prefixes


def price_series(entries):
    """Every price directive as a normalized (date, base, quote, rate) row."""
    rows = []
    for value in entries:
        if not isinstance(value, data.Price):
            continue
        rows.append(encode({
            "date": value.date.isoformat(),
            "base": value.currency,
            "quote": value.amount.currency,
            "rate": norm_num(value.amount.number),
        }))
    return sorted(rows)


def period_income(entries):
    """Income/expense posting totals keyed by "<YYYY-MM>|<currency>"."""
    totals = defaultdict(Decimal)
    for value in entries:
        if not isinstance(value, data.Transaction):
            continue
        period = value.date.strftime("%Y-%m")
        for posting in value.postings:
            root = posting.account.split(":")[0]
            if root not in INCOME_ROOTS or posting.units is None:
                continue
            totals[(period, posting.units.currency)] += posting.units.number
    return {
        f"{period}|{currency}": norm_num(total)
        for (period, currency), total in sorted(totals.items())
        if total
    }


def extend(base, path):
    """Attach the extra dimensions to the schema-2 golden for one fixture."""
    entries, errors, _ = loader.load_file(str(path))
    if errors:
        # An invalid ledger has no accepted state, so every state dimension is
        # empty on both sides; carrying entries here would compare fiction.
        return dict(base, ordered_entries=[], prefix_inventories=[], prices=[], period_income={})
    return dict(
        base,
        ordered_entries=ordered_entries(entries),
        prefix_inventories=prefix_inventories(entries),
        prices=price_series(entries),
        period_income=period_income(entries),
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", help="directory of *.bean fixtures")
    parser.add_argument("output", help="destination JSON file")
    args = parser.parse_args()

    if beancount.__version__ != "3.2.3":
        raise SystemExit("Expected pinned Beancount 3.2.3")

    root = Path(args.directory)
    paths = sorted(root.glob("*.bean"))
    if not paths:
        raise SystemExit(f"no fixtures in {root}")

    results = [extend(generate(path), path) for path in paths]
    Path(args.output).write_text(json.dumps(results, indent=2, sort_keys=True, default=str) + "\n")
    print(json.dumps({"fixtures": len(results), "output": args.output}, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
