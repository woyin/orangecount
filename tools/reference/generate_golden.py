#!/usr/bin/env python3
"""Development-only Beancount v3 oracle golden-snapshot generator.

Runs the official Beancount v3 loader over the sanitized v3-parity fixture
corpus and writes one normalized golden JSON per fixture to
testdata/golden/v3-parity/. The fixtures contain no personal data; golden
files therefore never carry private ledger content.

Usage:
  uv run --project tools/reference python tools/reference/generate_golden.py
"""
from __future__ import annotations

import json
import sys
from collections import defaultdict
from datetime import date, datetime
from decimal import Decimal
from pathlib import Path

import beancount
from beancount import loader
from beancount.core import data
from beancount.core.inventory import Inventory

REPO = Path(__file__).resolve().parent.parent.parent
FIXTURES = REPO / "testdata" / "fixtures" / "v3-parity"
GOLDEN = REPO / "testdata" / "golden" / "v3-parity"


def norm_num(d: Decimal) -> str:
    s = format(d, "f")
    if "." in s:
        s = s.rstrip("0").rstrip(".")
    if s in ("-0", ""):
        s = "0"
    return s


def norm_amount(a) -> str | None:
    if a is None:
        return None
    return f"{norm_num(a.number)} {a.currency}"


def norm_cost(c) -> str | None:
    if c is None or c.number is None or c.currency is None:
        return None
    return f"{norm_num(c.number)} {c.currency}"


def norm_value(v):
    if isinstance(v, Decimal):
        return norm_num(v)
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, (datetime, date)):
        return v.isoformat()
    if isinstance(v, str):
        return v
    return str(v)


def norm_custom_value(v) -> str:
    """Beancount v3 custom values are ValueType(value=..., dtype=...) pairs."""
    value = getattr(v, "value", v)
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, Decimal):
        return norm_num(value)
    if hasattr(value, "number") and hasattr(value, "currency"):  # Amount
        return f"{norm_num(value.number)} {value.currency}"
    if isinstance(value, (datetime, date)):
        return value.isoformat()
    return str(value)


def norm_entry(e) -> dict:
    base = {"kind": type(e).__name__.lower(), "date": e.date.isoformat()}
    if isinstance(e, data.Transaction):
        base.update(
            kind="transaction",
            flag=e.flag,
            payee=e.payee,
            narration=e.narration,
            tags=sorted(e.tags),
            links=sorted(e.links),
            postings=[
                {
                    "account": p.account,
                    "units": norm_amount(p.units),
                    "cost": norm_cost(p.cost),
                    "price": norm_amount(p.price),
                }
                for p in e.postings
            ],
        )
    elif isinstance(e, data.Open):
        base.update(
            account=e.account,
            currencies=sorted(e.currencies or []),
            booking=e.booking,
        )
    elif isinstance(e, data.Close):
        base.update(account=e.account)
    elif isinstance(e, data.Balance):
        base.update(
            account=e.account,
            amount=norm_amount(e.amount),
            tolerance=norm_num(e.tolerance) if e.tolerance is not None else None,
        )
    elif isinstance(e, data.Commodity):
        base.update(currency=e.currency)
    elif isinstance(e, data.Pad):
        base.update(account=e.account, source_account=e.source_account)
    elif isinstance(e, data.Event):
        description = getattr(e, "description", None)
        if description is None:
            description = getattr(e, "value", "")
        base.update(event_type=e.type, value=description)
    elif isinstance(e, data.Query):
        base.update(name=e.name, query=e.query_string)
    elif isinstance(e, data.Price):
        base.update(currency=e.currency, amount=norm_amount(e.amount))
    elif isinstance(e, data.Document):
        base.update(
            account=e.account,
            filename=Path(e.filename).name,
            tags=sorted(e.tags),
            links=sorted(e.links),
        )
    elif isinstance(e, data.Note):
        base.update(account=e.account, comment=e.comment)
    elif isinstance(e, data.Custom):
        base.update(custom_type=e.type, values=[norm_custom_value(v) for v in e.values])
    else:
        base.update(unnormalized=type(e).__name__)
    return base


def norm_inventory(inv: Inventory) -> dict:
    currencies: dict[str, str] = defaultdict(Decimal)
    lots: list[str] = []
    for pos in inv:
        units = norm_amount(pos.units)
        if pos.cost is None:
            cur = pos.units.currency
            currencies[cur] += pos.units.number
        else:
            lots.append(f"{units} {{{norm_cost(pos.cost)}}}")
    return {
        "currencies": {c: norm_num(n) for c, n in sorted(currencies.items())},
        "lots": sorted(lots),
    }


def compute_balances(entries) -> dict[str, dict]:
    invs: dict[str, Inventory] = defaultdict(Inventory)
    for e in entries:
        if isinstance(e, data.Transaction):
            for p in e.postings:
                if p.units is not None:
                    invs[p.account].add_position(p)
    return {a: norm_inventory(i) for a, i in sorted(invs.items())}


def norm_cell(cell):
    if isinstance(cell, Inventory):
        return norm_inventory(cell)
    if isinstance(cell, Decimal):
        return norm_num(cell)
    if isinstance(cell, (datetime, date)):
        return cell.isoformat()
    if isinstance(cell, set):
        return sorted(cell)
    return cell


QUERIES = {
    "account_totals": "SELECT account, sum(position) AS total GROUP BY account ORDER BY account",
    "cash_entries": "SELECT date, narration WHERE account ~ 'Cash' ORDER BY date",
}


def run_queries(entries, options_map) -> dict:
    from beanquery import query

    out = {}
    for name, sql in QUERIES.items():
        try:
            _rtype, rows = query.run_query(entries, options_map, sql)
            out[name] = [[norm_cell(c) for c in row] for row in rows]
        except Exception as exc:  # noqa: BLE001 - oracle failure is evidence
            out[name] = {"query_error": type(exc).__name__}
    return out


def generate(fixture: Path) -> dict:
    entries, errors, options_map = loader.load_file(str(fixture))
    has_errors = bool(errors)
    golden = {
        "fixture": fixture.name,
        "oracle_engine": f"beancount-python-{beancount.__version__}",
        "valid": not has_errors,
        "errors": [
            {"type": type(e).__name__, "line": e.source.get("lineno")} for e in errors
        ],
        "entries": [] if has_errors else sorted(
            (json.dumps(norm_entry(e), sort_keys=True, separators=(",", ":"), ensure_ascii=False) for e in entries)
        ),
        "balances": {} if has_errors else compute_balances(entries),
        "queries": {} if has_errors else run_queries(entries, options_map),
    }
    return golden


def main() -> int:
    GOLDEN.mkdir(parents=True, exist_ok=True)
    fixtures = sorted(FIXTURES.glob("*.bean"))
    if not fixtures:
        print("no fixtures found", file=sys.stderr)
        return 2
    failures = 0
    for fixture in fixtures:
        try:
            golden = generate(fixture)
        except Exception as exc:  # noqa: BLE001
            print(f"ORACLE-FAILURE {fixture.name}: {type(exc).__name__}", file=sys.stderr)
            failures += 1
            continue
        out = GOLDEN / f"{fixture.stem}.golden.json"
        out.write_text(json.dumps(golden, indent=1, sort_keys=True) + "\n", encoding="utf-8")
        status = "invalid" if not golden["valid"] else f"valid entries={len(golden['entries'])}"
        print(f"{fixture.name}: {status}")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
