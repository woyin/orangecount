#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Classify every pinned upstream capability by how it can be verified.

The classification is a plan, not evidence. It records for each inventory item
whether a probe can evaluate it directly, whether it needs a ledger fixture, or
whether it is upstream-internal structure with no public Go behaviour. Writing
a category never changes the item's verification status.
"""
import argparse
import hashlib
import json
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
INVENTORY = REPO / 'docs/compatibility/upstream-inventory.json'
DEST = REPO / 'docs/compatibility/inventory-triage.json'

# Syntax nodes that only describe parser structure. They have no independent
# query-visible behaviour, so a probe cannot distinguish them from their use.
INTERNAL_SYNTAX = {
    'syntax.Node': 'base AST node class; no independent public behaviour',
    'syntax.Ordering': 'ORDER BY ordering enum; behaviour is exercised by syntax.Ordering through queries, not as a standalone surface',
}

def sha256(data):
    return hashlib.sha256(data).hexdigest()

def classify(item):
    kind = item['kind']
    ident = item['id']
    if ident in INTERNAL_SYNTAX:
        return {
            'category': 'internal',
            'rationale': INTERNAL_SYNTAX[ident],
            'probe': None,
        }
    if kind == 'column':
        _, table, column = ident.split('.')
        return {
            'category': 'probe',
            'probe': {
                'surface': 'sql',
                'query': 'SELECT "%s" FROM "%s"' % (column, table),
                'fixture': 'testdata/golden/inventory/fixture.bean',
                'observes': ['declared column type', 'typed row values'],
            },
        }
    if kind == 'function':
        name = ident.split('.')[1]
        requires_ledger = name in {
            'open_date', 'close_date', 'open_meta', 'close_meta', 'meta',
            'any_meta', 'entry_meta', 'commodity_meta', 'currency_meta',
            'has_account', 'getprice', 'convert', 'value', 'only',
        }
        return {
            'category': 'probe',
            'probe': {
                'surface': 'sql-function',
                'requires_ledger': requires_ledger,
                'fixture': 'testdata/golden/inventory/fixture.bean' if requires_ledger else None,
                'observes': ['output type', 'value for constructed arguments'],
                'unstable': ident == 'function.today.0',
                'note': 'today() depends on the wall clock, so only its declared type is stable' if ident == 'function.today.0' else None,
            },
        }
    if kind == 'operator':
        return {
            'category': 'probe',
            'probe': {
                'surface': 'sql-operator',
                'observes': ['output type', 'value for constructed operands'],
                'note': 'unary and binary forms are exercised through the SQL expression surface',
            },
        }
    if kind == 'syntax':
        return {
            'category': 'needs-fixture',
            'probe': {
                'surface': 'sql-parse',
                'fixture': 'testdata/golden/inventory/fixture.bean',
                'observes': ['parse acceptance', 'evaluated result of the syntax form'],
            },
        }
    if kind == 'accounting-option':
        return {
            'category': 'needs-fixture',
            'probe': {
                'surface': 'loader-option',
                'fixture': 'testdata/golden/inventory/options.bean',
                'observes': ['option value after load', 'entry shape affected by the option'],
            },
        }
    if kind == 'accounting-directive':
        return {
            'category': 'needs-fixture',
            'probe': {
                'surface': 'loader-entry',
                'fixture': 'testdata/golden/inventory/fixture.bean',
                'observes': ['directive fields', 'loader errors', 'table row'],
            },
        }
    raise SystemExit('unknown capability kind: ' + kind)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    data = INVENTORY.read_bytes()
    inventory = json.loads(data)
    items = []
    counts = {}
    for item in inventory['capabilities']:
        decision = classify(item)
        items.append({
            'id': item['id'],
            'kind': item['kind'],
            'category': decision['category'],
            'probe': decision.get('probe'),
            'rationale': decision.get('rationale'),
        })
        counts[decision['category']] = counts.get(decision['category'], 0) + 1
    output = {
        'schema_version': 1,
        'source_inventory': 'docs/compatibility/upstream-inventory.json',
        'source_inventory_sha256': sha256(data),
        'counts': counts,
        'items': items,
    }
    encoded = json.dumps(output, indent=2, sort_keys=True, ensure_ascii=False) + '\n'
    if args.check:
        if not DEST.exists() or DEST.read_text() != encoded:
            raise SystemExit('inventory triage is stale; rerun tools/reference/triage_inventory.py')
        print('inventory triage matches %d capabilities' % len(items))
        return
    DEST.write_text(encoded)
    print('wrote %s: %s' % (DEST, counts))

if __name__ == '__main__':
    main()
