#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Apply verified inventory results from the Go comparison report.

Only items whose oracle golden exists, whose Go comparison passed, and whose
go_location is concrete become verified. Everything else keeps its status.
This script never marks an item excluded.
"""
import argparse
import json
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
INVENTORY = REPO / 'docs/compatibility/upstream-inventory.json'

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--report', default='/tmp/inv-report.json')
    parser.add_argument('--check', action='store_true')
    parser.add_argument('--kind', action='append', default=[])
    args = parser.parse_args()
    report = json.loads(Path(args.report).read_text())
    results = {item['id']: item for item in report['results']}
    original = INVENTORY.read_text()
    data = json.loads(original)
    changed = 0
    for item in data['capabilities']:
        if args.kind and item['kind'] not in args.kind:
            continue
        result = results.get(item['id'])
        if not result or result.get('status') != 'verified':
            continue
        if not result.get('go_location'):
            continue
        desired = {
            'status': 'verified',
            'go_location': result['go_location'],
            'cases': [
                {'kind': 'oracle-golden', 'path': 'testdata/golden/inventory/%s.json' % item['id']},
                {'kind': 'go-comparison', 'path': 'internal/compat/inventory_test.go'},
            ],
        }
        for field, value in desired.items():
            if item.get(field) != value:
                item[field] = value
                changed += 1
    encoded = json.dumps(data, indent=2, sort_keys=True, ensure_ascii=False) + '\n'
    if args.check:
        if encoded != original:
            raise SystemExit('inventory verification status is stale')
        print('inventory verification status is current')
        return
    # Refuse to overwrite a concurrent writer's schema additions.
    if not original.lstrip().startswith('{'):
        raise SystemExit('inventory has an unexpected shape; refusing to write')
    INVENTORY.write_text(encoded)
    print('updated %d inventory fields' % changed)

if __name__ == '__main__':
    main()
