#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Same-input private comparison; only redacted counts leave this process.

Source captures and oracle inputs live outside the repository in a temporary
directory which is removed even on failure. Never prints exception messages.
"""
from __future__ import annotations

import base64
import json
import os
import subprocess
import tempfile
from collections import Counter
from decimal import Decimal, localcontext
from fractions import Fraction
from pathlib import Path

from beanquery import query
from beancount import loader
from beancount.core import data, inventory as bean_inventory
from generate_full_golden import generate, norm_num

REPO = Path(__file__).resolve().parents[2]


def compare():
    path = Path(os.environ["ORANGECOUNT_PRIVATE_LEDGER"])
    if not path.is_absolute() or path.resolve().is_relative_to(REPO):
        raise ValueError("input must be external")
    with tempfile.TemporaryDirectory(prefix="orangecount-private-full-") as directory:
        temporary = Path(directory)
        capture_path = temporary / "capture.json"
        captured = subprocess.run(["go", "run", "./tools/statecapture", "-ledger", str(path), "-out", str(capture_path)],
                                  cwd=REPO, capture_output=True, check=False)
        if captured.returncode:
            raise RuntimeError("capture failed")
        capture = json.loads(capture_path.read_text())
        files = {str(f["ID"]): f for f in capture["files"]}
        common = Path(os.path.commonpath([str(Path(f["Path"]).parent) for f in files.values()]))
        mirror = temporary / "input"
        locations = {key: mirror / Path(f["Path"]).relative_to(common) for key, f in files.items()}
        for key, file in files.items():
            target = locations[key]
            target.parent.mkdir(parents=True, exist_ok=True)
            content = base64.b64decode(file["Data"] or "")
            # Rewrite only include literals so the pinned loader reads exactly
            # the captured graph, never a later version of a user's source.
            replacements = {}
            for edge in capture["edges"].get(key, []):
                replacements.setdefault((edge["Span"]["start"], edge["Span"]["end"]), []).append(edge)
            for (start, end), edges in sorted(replacements.items(), reverse=True):
                literal = edges[0]["Literal"]
                if any(c in literal for c in "*?["):
                    # Glob captures keep their original relative structure;
                    # no noncaptured files are present in this input tree.
                    replacement = str(mirror / Path(literal).relative_to(common)) if Path(literal).is_absolute() else literal
                else:
                    replacement = str(locations[str(edges[0]["To"])])
                content = content[:start] + json.dumps(replacement).encode() + content[end:]
            target.write_bytes(content)
        entry_path = mirror / Path(capture["entry"]).relative_to(common)
        want = generate(entry_path)
        have = capture["side"]
        desired, actual = want["state"], have["state"]
        missing = Counter(desired["entries"]) - Counter(actual["entries"])
        extra = Counter(actual["entries"]) - Counter(desired["entries"])
        accounts = set(desired["balances"]) | set(actual["balances"])
        changed_accounts = sum(desired["balances"].get(a) != actual["balances"].get(a) for a in accounts)
        categories = Counter()
        price_policy = Counter()
        index = {}
        for raw in actual["entries"]:
            value = json.loads(raw)
            index[(value["kind"], value["source"]["file"], value["source"]["line"])] = value
        for raw in missing:
            before = json.loads(raw)
            after = index.get((before["kind"], before["source"]["file"], before["source"]["line"]))
            if after is None:
                categories["missing_source_entry"] += 1
                continue
            for field in set(before) | set(after):
                if before.get(field) == after.get(field):
                    continue
                if field == "postings":
                    encode = lambda p: json.dumps(p, sort_keys=True)
                    if Counter(map(encode, before[field])) == Counter(map(encode, after[field])):
                        categories["posting_order"] += 1
                    else:
                        changed = set()
                        for left, right in zip(before[field], after[field]):
                            for part in set(left) | set(right):
                                if left.get(part) != right.get(part): changed.add(part)
                            if left.get("price") and right.get("price") and left["price"] != right["price"]:
                                exact = Fraction(right["price"]["number"])
                                with localcontext() as context:
                                    context.prec = 28
                                    rounded = Decimal(exact.numerator) / Decimal(exact.denominator)
                                price_policy["pinned_decimal_context_equal" if rounded == Decimal(left["price"]["number"]) else "other"] += 1
                        for part in changed: categories["posting_" + part] += 1
                else:
                    categories["entry_" + field] += 1
        entries, errors, options = loader.load_file(str(entry_path))
        period_counts = {}
        for account in ["Expenses", "Income"]:
            sql = ("SELECT year(date) AS year, month(date) AS month, currency, sum(number) AS total FROM postings "
                   f"WHERE account ~ '^{account}' GROUP BY year(date), month(date), currency ORDER BY year(date), month(date), currency")
            _, rows = query.run_query(entries, options, sql)
            oracle = [[str(y), str(m), cur, norm_num(total)] for y, m, cur, total in rows]
            orange = [[str(row["year"]), str(row["month"]), row["currency"], row["total"]] for row in (capture["periods"][account]["rows"] or [])]
            period_counts[account] = {"rows": len(oracle), "equal": oracle == orange}
        summary = {"schema_version": 2, "oracle": want["oracle_engine"], "semantics_profile": "v3",
                   "same_input_capture": True, "captured_files": len(files),
                   "oracle_valid": want["valid"], "orangecount_valid": have["valid"],
                   "oracle_errors": len(errors), "orangecount_errors": len(have["errors"]),
                   "oracle_entries": len(desired["entries"]), "orangecount_entries": len(actual["entries"]),
                   "missing_entries": sum(missing.values()), "extra_entries": sum(extra.values()),
                   "difference_categories": dict(categories),
                   "price_policy_diagnostics": dict(price_policy),
                   "changed_inventory_accounts": changed_accounts, "periods": period_counts}
        native_probe = os.environ.get("ORANGECOUNT_NATIVE_PROBE")
        native_ok = True
        summary["native_checked"] = bool(native_probe)
        if native_probe:
            roots = {}
            for name, inventory in desired["balances"].items():
                root = name.split(":", 1)[0]
                if root not in {"Assets", "Liabilities", "Equity", "Income", "Expenses"}:
                    continue
                totals = roots.setdefault(root, {})
                for cur, number in inventory["currencies"].items():
                    totals[cur] = totals.get(cur, Fraction(0)) + Fraction(number)
                for encoded in inventory["lots"]:
                    lot = json.loads(encoded)
                    cur = lot["cost"]["currency"]
                    totals[cur] = totals.get(cur, Fraction(0)) + Fraction(lot["number"]) * Fraction(lot["cost"]["number"])
            def exact(number):
                denominator = number.denominator
                twos = fives = 0
                while denominator % 2 == 0:
                    denominator //= 2; twos += 1
                while denominator % 5 == 0:
                    denominator //= 5; fives += 1
                if denominator != 1:
                    return str(number)
                scale = max(twos, fives)
                if not scale:
                    return str(number.numerator)
                magnitude = abs(number.numerator) * (10**scale // number.denominator)
                digits = str(magnitude).zfill(scale + 1)
                value = digits[:-scale] + "." + digits[-scale:]
                value = value.rstrip("0").rstrip(".")
                return ("-" if number < 0 else "") + value
            expected = {"input_identity": capture["input_identity"], "entry_count": len(entries),
                        "transaction_count": sum(type(e).__name__ == "Transaction" for e in entries),
                        "account_count": len({e.account for e in entries if type(e).__name__ == "Open"}),
                        "roots": {r: {c: exact(n) for c, n in totals.items()} for r, totals in roots.items()}}
            # Independent historical inventory: fold pinned Python's already
            # booked postings at distinct cutoffs. Never filter surviving lots.
            dates = sorted({entry.date for entry in entries if entry.date is not None})
            cutoffs = sorted({dates[0], dates[len(dates) // 2], dates[-1]}) if dates else []
            holding_checks = []
            for cutoff in cutoffs:
                accounts = {}
                for entry in entries:
                    if not isinstance(entry, data.Transaction) or entry.date > cutoff:
                        continue
                    for posting in entry.postings:
                        if posting.account.split(":", 1)[0] not in {"Assets", "Liabilities"}:
                            continue
                        accounts.setdefault(posting.account, bean_inventory.Inventory()).add_position(posting)
                groups = {}
                for account, positions in accounts.items():
                    for position in positions:
                        quantity = Fraction(position.units.number)
                        if not quantity:
                            continue
                        cost = position.cost
                        key = (account, position.units.currency, cost.currency if cost is not None else None)
                        group = groups.setdefault(key, {"account": account, "commodity": position.units.currency,
                                                       "cost_currency": key[2], "quantity": Fraction(0),
                                                       "cost_basis": Fraction(0) if cost is not None else None, "lots": []})
                        group["quantity"] += quantity
                        if cost is not None:
                            group["cost_basis"] += quantity * Fraction(cost.number)
                        group["lots"].append({"quantity": exact(quantity),
                                              "cost_number": exact(Fraction(cost.number)) if cost is not None else None,
                                              "cost_date": str(cost.date) if cost is not None and cost.date is not None else None,
                                              "cost_label": cost.label if cost is not None else None})
                for group in groups.values():
                    quantity, basis = group["quantity"], group["cost_basis"]
                    group["average_cost"] = exact(basis / quantity) if basis is not None and quantity else None
                    group["quantity"] = exact(quantity)
                    group["cost_basis"] = exact(basis) if basis is not None else None
                holding_checks.append({"date_to": str(cutoff), "groups": list(groups.values())})
            expected["holding_checks"] = holding_checks
            # Price oracle is independent of the Go projection. Preserve every
            # source price, including duplicates, and select at each cutoff.
            catalog = {}
            def add_commodity(currency, account=None):
                item = catalog.setdefault(currency, {"currency": currency, "accounts": set(),
                    "quote_currencies": set(), "declaration_count": 0, "price_count": 0})
                if account is not None:
                    item["accounts"].add(account)
                return item
            source_prices = []
            for order, entry in enumerate(entries):
                if isinstance(entry, data.Commodity):
                    add_commodity(entry.currency)["declaration_count"] += 1
                elif isinstance(entry, data.Open):
                    for currency in entry.currencies or []:
                        add_commodity(currency, entry.account)
                elif isinstance(entry, data.Balance):
                    add_commodity(entry.amount.currency, entry.account)
                elif isinstance(entry, data.Transaction):
                    for posting in entry.postings:
                        add_commodity(posting.units.currency, posting.account)
                        if posting.cost is not None:
                            add_commodity(posting.cost.currency, posting.account)
                        if posting.price is not None:
                            add_commodity(posting.price.currency, posting.account)
                elif isinstance(entry, data.Price):
                    left, right = add_commodity(entry.currency), add_commodity(entry.amount.currency)
                    left["quote_currencies"].add(entry.amount.currency)
                    right["quote_currencies"].add(entry.currency)
                    left["price_count"] += 1
                    if left is not right:
                        right["price_count"] += 1
                    source_prices.append((entry.date, order, entry.currency, entry.amount.currency, Fraction(entry.amount.number)))
            for item in catalog.values():
                item["accounts"] = sorted(item["accounts"])
                item["quote_currencies"] = sorted(item["quote_currencies"])
            expected["price_catalog"] = [catalog[c] for c in sorted(catalog)]
            pairs = {(p[2], p[3]) for p in source_prices}
            pairs |= {(quote, base) for base, quote in list(pairs)}
            # Also check genuinely missing pairs when a ledger has no prices.
            # The target must exist in the catalog, not just an option string.
            target = next((c for c in options.get("operating_currency", []) if c in catalog),
                          next(iter(sorted(catalog)), None))
            if target is not None:
                pairs |= {(currency, target) for currency in catalog}
            price_checks = []
            for cutoff in cutoffs:
                for base, quote in sorted(pairs):
                    direct = [p for p in source_prices if p[2:4] == (base, quote) and p[0] <= cutoff]
                    reverse = [p for p in source_prices if p[2:4] == (quote, base) and p[0] <= cutoff]
                    chosen = max(direct, default=None, key=lambda p: p[:2])
                    inverse = chosen is None
                    if inverse:
                        chosen = max(reverse, default=None, key=lambda p: p[:2])
                        if chosen is not None and not chosen[4]:
                            chosen = None
                    selected = None if chosen is None else {"date": str(chosen[0]),
                        "rate": exact(1 / chosen[4] if inverse else chosen[4]), "inverse": inverse,
                        "source_base": chosen[2], "source_quote": chosen[3], "age_days": (cutoff - chosen[0]).days}
                    price_checks.append({"base": base, "quote": quote, "date_to": str(cutoff),
                        "records": [{"date": str(p[0]), "rate": exact(p[4])} for p in direct], "selected": selected})
            expected["price_checks"] = price_checks
            expected_path = temporary / "native-expected.json"
            expected_path.write_text(json.dumps(expected))
            result = subprocess.run([native_probe, str(path), str(expected_path)], capture_output=True, check=False)
            native_ok = result.returncode == 0
            summary["native_contract"] = native_ok
            summary["native_holding_cutoffs"] = len(holding_checks) if native_ok else 0
            summary["native_price_checks"] = len(price_checks) if native_ok else 0
            # The probe has a counts-only output contract; never forward raw
            # stdout/stderr in case an unhandled decoder message contains data.
        print(json.dumps(summary, sort_keys=True))
        return int(not (want["valid"] and have["valid"]) or bool(missing or extra or changed_accounts)
                   or not all(p["equal"] for p in period_counts.values()) or not native_ok)


if __name__ == "__main__":
    try:
        status = compare()
    except Exception as error:
        print(json.dumps({"verification_error": type(error).__name__}))
        status = 2
    raise SystemExit(status)
