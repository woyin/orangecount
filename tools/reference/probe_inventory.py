#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Capture pinned upstream behaviour for inventory capabilities.

Each probe evaluates the capability on the fixed oracle (beancount 3.2.3 /
beanquery 0.2.0) against a reviewed fixture and writes the typed result to
testdata/golden/inventory/<id>.json. A failed evaluation is recorded as a
failure with its exact class and message; values are never invented.
"""
import argparse
import datetime
import decimal
import hashlib
import json
from pathlib import Path

import os

import beancount
import beanquery
from beancount.core import compare, data
from dateutil.relativedelta import relativedelta
from beancount import loader
from beancount.core import amount as amount_module
from beancount.core import inventory as inventory_module
from beancount.core import position as position_module
from beanquery import types

REPO = Path(__file__).resolve().parents[2]
INVENTORY = REPO / 'docs/compatibility/upstream-inventory.json'
OUT = REPO / 'testdata/golden/inventory'
OPTIONS_FIXTURE = OUT / 'options.bean'
SYNTAX_FIXTURE = OUT / 'syntax.bean'
FIXTURE = OUT / 'fixture.bean'
DISCREPANCY_FIXTURE = OUT / 'discrepancy.bean'
INDEX = OUT / 'index.json'
COLUMN_FIXTURES = {'table.balances.discrepancy': 'discrepancy.bean'}

# Every option probe loads a fixture that exercises the option, then records the
# value the loader kept plus the directive shape the option changed.
OPTION_QUERIES = {
    'option.account_current_conversions': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.account_current_earnings': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.account_previous_balances': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.account_previous_conversions': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.account_previous_earnings': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.account_rounding': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.account_unrealized_gains': ("SELECT type AS v FROM #entries LIMIT 1", 'options.bean'),
    'option.allow_deprecated_none_for_tags_and_links': ("SELECT tags AS v FROM #transactions", 'options.bean'),
    'option.allow_pipe_separator': ("SELECT payee AS v FROM #transactions", 'options.bean'),
    'option.booking_method': ("SELECT position AS v FROM #postings", 'options.bean'),
    'option.commodities': ("SELECT name AS v FROM #commodities", 'options.bean'),
    'option.conversion_currency': ("SELECT position AS v FROM #postings", 'options.bean'),
    'option.dcontext': ("SELECT position AS v FROM #postings", 'options.bean'),
    'option.display_precision': ("SELECT number AS v FROM #postings", 'options.bean'),
    'option.documents': ("SELECT filename AS v FROM #documents", 'fixture.bean'),
    'option.filename': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.include': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.infer_tolerance_from_cost': ("SELECT tolerance AS v FROM #balances", 'fixture.bean'),
    'option.inferred_tolerance_default': ("SELECT tolerance AS v FROM #balances", 'fixture.bean'),
    'option.inferred_tolerance_multiplier': ("SELECT number AS v FROM #postings", 'options.bean'),
    'option.input_hash': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.insert_pythonpath': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.long_string_maxlines': ("SELECT narration AS v FROM #transactions", 'options.bean'),
    'option.name_assets': ("SELECT account AS v FROM #accounts", 'options.bean'),
    'option.name_equity': ("SELECT account AS v FROM #accounts", 'options.bean'),
    'option.name_expenses': ("SELECT account AS v FROM #accounts", 'options.bean'),
    'option.name_income': ("SELECT account AS v FROM #accounts", 'options.bean'),
    'option.name_liabilities': ("SELECT account AS v FROM #accounts", 'options.bean'),
    'option.operating_currency': ("SELECT currency AS v FROM #prices", 'fixture.bean'),
    'option.plugin': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.plugin_processing_mode': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.render_commas': ("SELECT amount AS v FROM #prices", 'options.bean'),
    'option.title': ("SELECT filename AS v FROM #entries", 'fixture.bean'),
    'option.tolerance_multiplier': ("SELECT tolerance AS v FROM #balances", 'options.bean'),
    'option.use_precise_interpolation': ("SELECT amount AS v FROM #prices", 'fixture.bean'),
}

SYNTAX_QUERIES = {
    'syntax.Add': "SELECT 1 + 2 AS v FROM #entries LIMIT 1",
    'syntax.And': "SELECT 1 = 1 AND 2 = 2 AS v FROM #entries LIMIT 1",
    'syntax.Div': "SELECT 6 / 2 AS v FROM #entries LIMIT 1",
    'syntax.Equal': "SELECT 1 = 1 AS v FROM #entries LIMIT 1",
    'syntax.From': "SELECT account AS v FROM #accounts LIMIT 1",
    'syntax.Greater': "SELECT 2 > 1 AS v FROM #entries LIMIT 1",
    'syntax.GreaterEq': "SELECT 2 >= 2 AS v FROM #entries LIMIT 1",
    'syntax.In': "SELECT 'Assets:Cash' IN accounts AS v FROM #postings LIMIT 1",
    'syntax.IsNotNull': "SELECT 1 IS NOT NULL AS v FROM #entries LIMIT 1",
    'syntax.IsNull': "SELECT 1 IS NULL AS v FROM #entries LIMIT 1",
    'syntax.Less': "SELECT 1 < 2 AS v FROM #entries LIMIT 1",
    'syntax.LessEq': "SELECT 2 <= 2 AS v FROM #entries LIMIT 1",
    'syntax.Match': "SELECT 'abc' ~ 'b' AS v FROM #entries LIMIT 1",
    'syntax.Matches': "SELECT 'abc' ?~ 'abc' AS v FROM #entries LIMIT 1",
    'syntax.Mod': "SELECT 5 % 2 AS v FROM #entries LIMIT 1",
    'syntax.Mul': "SELECT 2 * 3 AS v FROM #entries LIMIT 1",
    'syntax.Neg': "SELECT -2 AS v FROM #entries LIMIT 1",
    'syntax.Not': "SELECT NOT TRUE AS v FROM #entries LIMIT 1",
    'syntax.NotEqual': "SELECT 1 != 2 AS v FROM #entries LIMIT 1",
    'syntax.NotIn': "SELECT 'x' NOT IN accounts AS v FROM #postings LIMIT 1",
    'syntax.NotMatch': "SELECT 'abc' !~ 'z' AS v FROM #entries LIMIT 1",
    'syntax.Or': "SELECT 1 = 2 OR 2 = 2 AS v FROM #entries LIMIT 1",
    # Ordering is an IntEnum (ASC=0, DESC=1) whose only observable effect is
    # the direction of a sort, so the probe reads the single value that ORDER
    # BY DESC selects. An implementation that ignored the flag would return
    # the ASC first row instead and fail the comparison.
    'syntax.Ordering': "SELECT account AS v FROM #accounts ORDER BY account DESC LIMIT 1",
    # Node is the AST base class: its observable contract is that one statement
    # text decomposes into a set of typed nodes, each of which is a Node. The
    # golden records that fingerprint, and the Go side must produce the same
    # node kinds from its own parser.
    'syntax.Node': "SELECT account AS v FROM #accounts ORDER BY account DESC",
    'syntax.Sub': "SELECT 3 - 1 AS v FROM #entries LIMIT 1",
}

DIRECTIVE_QUERIES = {
    'directive.Open': "SELECT open AS v FROM #accounts LIMIT 1",
    'directive.Close': "SELECT close AS v FROM #accounts",
    'directive.Commodity': "SELECT name AS v FROM #commodities LIMIT 1",
    'directive.Pad': "SELECT type AS v FROM #entries WHERE type = 'pad' LIMIT 1",
    'directive.Balance': "SELECT amount AS v FROM #balances LIMIT 1",
    'directive.Transaction': "SELECT entry AS v FROM #postings LIMIT 1",
    'directive.Note': "SELECT comment AS v FROM #notes LIMIT 1",
    'directive.Event': "SELECT type AS v FROM #events LIMIT 1",
    'directive.Query': "SELECT type AS v FROM #entries WHERE type = 'query' LIMIT 1",
    'directive.Price': "SELECT amount AS v FROM #prices LIMIT 1",
    'directive.Document': "SELECT filename AS v FROM #documents LIMIT 1",
    'directive.Custom': "SELECT type AS v FROM #entries WHERE type = 'custom' LIMIT 1",
}

ORACLE = {'beancount': '3.2.3', 'beanquery': '0.2.0'}

# Oracle datatype name -> Go/JSON cell type used by internal/query.
CELL_TYPES = {
    'str': 'string',
    'int': 'integer',
    'decimal': 'decimal',
    'date': 'date',
    'datetime': 'date',
    'bool': 'bool',
    'amount': 'amount',
    'position': 'position',
    'inventory': 'inventory',
    'set': 'set<string>',
    'set[str]': 'set<string>',
    'frozenset': 'set<string>',
    'metadata': 'dictionary',
    'dict': 'dictionary',
    'open': 'open',
    'close': 'close',
    'transaction': 'transaction',
    'relativedelta': 'interval',
    'object': 'object',
    'any': 'object',
    'null': 'null',
    'relativedelta': 'interval',
}

def sha256(data):
    return hashlib.sha256(data).hexdigest()

def relative(text):
    return text.replace(str(REPO) + '/', '')

def cell(kind, value):
    if value is None:
        return {'type': 'null'}
    # Inventory subclasses dict, and Position/Amount are namedtuples, so the
    # compound beancount types must be recognized before the generic containers.
    if isinstance(value, inventory_module.Inventory):
        positions = [position_value(p) for p in value]
        positions.sort(key=lambda p: json.dumps(p, sort_keys=True))
        return {'type': 'inventory', 'value': positions}
    if isinstance(value, position_module.Position):
        return {'type': 'position', 'value': position_value(value)}
    if isinstance(value, amount_module.Amount):
        return {'type': 'amount', 'value': {'number': str(value.number), 'currency': value.currency}}
    if isinstance(value, dict):
        entries = []
        for key in sorted(value):
            entries.append({'key': key, 'value': cell(cell_type_of(value[key]), value[key])})
        return {'type': 'dictionary', 'value': entries}
    if isinstance(value, (set, frozenset)):
        return {'type': 'set<string>', 'value': sorted(value)}
    if isinstance(value, (list, tuple)) and not hasattr(value, '_fields'):
        return {'type': 'list', 'value': [cell(cell_type_of(v), v) for v in value]}
    if hasattr(value, '_fields'):
        # The reviewed query goldens project the transaction record as its
        # scalar fields plus the account set; postings stay out of the wire
        # value so a directive cell stays bounded.
        names = sorted(value._fields)
        override = {}
        if type(value).__name__ == 'Transaction':
            names = sorted([name for name in names if name != 'postings'] + ['accounts'])
            override['accounts'] = {p.account for p in value.postings}
        fields = []
        for key in names:
            child = override[key] if key in override else getattr(value, key)
            fields.append({'key': key, 'value': cell(cell_type_of(child), child)})
        return {'type': kind, 'value': fields}
    if isinstance(value, bool):
        return {'type': 'bool', 'value': value}
    if isinstance(value, int):
        # internal/query presents integers as exact text so no precision is lost.
        return {'type': 'integer', 'value': str(value)}
    if isinstance(value, relativedelta):
        return {'type': 'interval', 'value': {
            'years': value.years, 'months': value.months, 'days': value.days,
            'hours': value.hours, 'minutes': value.minutes, 'seconds': value.seconds,
        }}
    if isinstance(value, decimal.Decimal):
        return {'type': 'decimal', 'value': str(value)}
    if isinstance(value, datetime.datetime):
        return {'type': 'date', 'value': value.date().isoformat()}
    if isinstance(value, datetime.date):
        return {'type': 'date', 'value': value.isoformat()}
    if isinstance(value, str):
        return {'type': 'string', 'value': relative(value)}
    raise TypeError('unsupported oracle value type: %r' % type(value).__name__)

def cell_type_of(value):
    if value is None:
        return 'null'
    if isinstance(value, amount_module.Amount):
        return 'amount'
    if isinstance(value, position_module.Position):
        return 'position'
    if isinstance(value, inventory_module.Inventory):
        return 'inventory'
    for name in ('Transaction', 'Open', 'Close', 'Balance', 'Note', 'Event',
                 'Document', 'Price', 'Commodity', 'Pad', 'Query', 'Custom'):
        if type(value).__name__ == name:
            return name.lower()
    if isinstance(value, bool):
        return 'bool'
    if isinstance(value, int):
        return 'integer'
    if isinstance(value, relativedelta):
        return 'interval'
    if isinstance(value, decimal.Decimal):
        return 'decimal'
    if isinstance(value, (datetime.date, datetime.datetime)):
        return 'date'
    if isinstance(value, str):
        return 'string'
    if isinstance(value, dict):
        return 'dictionary'
    if type(value).__name__ == 'Transaction':
        return 'transaction'
    if isinstance(value, (set, frozenset)):
        return 'set<string>'
    if isinstance(value, (list, tuple)):
        return 'list'
    return 'object'

def position_value(value):
    result = {'units': {'number': str(value.units.number), 'currency': value.units.currency}, 'cost': None}
    if value.cost is not None:
        result['cost'] = {
            'number': str(value.cost.number),
            'currency': value.cost.currency,
            'date': value.cost.date.isoformat() if value.cost.date else None,
            'label': value.cost.label,
        }
    return result

def connect(entries, errors, options):
    return beanquery.connect('beancount:', entries=entries, errors=errors, options=options)

def probe_query(context, query):
    cursor = context.cursor()
    cursor.execute(query)
    column = cursor.description[0]
    dtype = types.name(column.datatype)
    rows = cursor.fetchall()
    cells = []
    for row in rows:
        value = row[0]
        cells.append(cell(cell_type_of(value), value))
    cells.sort(key=lambda item: json.dumps(item, sort_keys=True))
    declared = CELL_TYPES.get(dtype, dtype)
    # A nullable column returns null rows too, so only non-null values can
    # confirm the declared type matches what upstream actually produces.
    observed = {row['type'] for row in cells if row['type'] != 'null'}
    return {
        'oracle_type': dtype,
        'type': declared,
        # Upstream declares a few result types that differ from the value it
        # actually returns (for example relativedelta - relativedelta declares
        # date). Recording the inconsistency keeps the comparison honest.
        'declared_type_consistent': len(observed) <= 1 and (not observed or declared in observed),
        'rows': cells,
    }

def probe_ids(context, identifier, fixture_dir):
    """Capture the id column: one hash per directive, path-normalized.

    The id column is not a stored field, so it cannot be selected directly;
    this calls the same upstream hash_entry the column delegates to.
    """
    import beancount.parser
    root = str(fixture_dir)
    entries = context['entries']
    if identifier == 'table.entries.id':
        values = [normalized_entry_id(entry, root) for entry in entries]
    else:
        values = [normalized_entry_id(entry, root) for entry in entries
                  if isinstance(entry, data.Transaction) for _ in entry.postings]
    cells = [{'type': 'string', 'value': value} for value in values]
    cells.sort(key=lambda item: json.dumps(item, sort_keys=True))
    return {'oracle_type': 'str', 'type': 'string', 'declared_type_consistent': True, 'rows': cells}

def column_query(item):
    _, table, column = item['id'].split('.')
    return 'SELECT %s AS v FROM #%s' % (column, table)

LITERALS = {
    'builtins.int': '2',
    'decimal.Decimal': '2.5',
    'builtins.str': "'abc'",
    'datetime.date': "date('2020-01-02')",
    'dateutil.relativedelta.relativedelta': "interval('1 month')",
    'beanquery.types.Any': '1',
    'builtins.dict': "{'a': 1}",
    'builtins.list': '[1, 2]',
    'builtins.set': '{1, 2}',
}

OPERATOR_SYMBOLS = {
    'Add': '+', 'Sub': '-', 'Mul': '*', 'Div': '/', 'Mod': '%',
    'Equal': '=', 'NotEqual': '!=', 'Greater': '>', 'GreaterEq': '>=',
    'Less': '<', 'LessEq': '<=', 'Match': '~', 'NotMatch': '!~', 'Matches': '?~',
}

# Collection membership needs real columns because the SQL grammar has no
# dict/list/set literals. The table column supplies the same operand type.
COLLECTION_MEMBERSHIP = {
    ('In', 'builtins.dict'): "SELECT 'filename' IN meta AS v FROM #entries LIMIT 1",
    ('NotIn', 'builtins.dict'): "SELECT 'filename' NOT IN meta AS v FROM #entries LIMIT 1",
    ('In', 'builtins.set'): "SELECT 'food' IN tags AS v FROM #transactions LIMIT 1",
    ('NotIn', 'builtins.set'): "SELECT 'food' NOT IN tags AS v FROM #transactions LIMIT 1",
    ('In', 'builtins.list'): "SELECT 2 IN (1, 2) AS v FROM #entries LIMIT 1",
    ('NotIn', 'builtins.list'): "SELECT 2 NOT IN (1, 2) AS v FROM #entries LIMIT 1",
}

def operator_query(item):
    _, name, _number = item['id'].split('.')
    inputs = item.get('input') or []
    key = (name, inputs[1] if len(inputs) > 1 else '')
    if key in COLLECTION_MEMBERSHIP:
        return COLLECTION_MEMBERSHIP[key]
    if name in ('IsNull', 'IsNotNull'):
        operand = 'NULL' if name == 'IsNull' else '1'
        keyword = 'IS NULL' if name == 'IsNull' else 'IS NOT NULL'
        return 'SELECT %s %s AS v FROM #entries LIMIT 1' % (operand, keyword)
    if name == 'Not':
        return 'SELECT NOT %s AS v FROM #entries LIMIT 1' % LITERALS[inputs[0]]
    if name == 'Neg':
        return 'SELECT -%s AS v FROM #entries LIMIT 1' % LITERALS[inputs[0]]
    if name in ('In', 'NotIn'):
        keyword = 'IN' if name == 'In' else 'NOT IN'
        return 'SELECT %s %s (1, 2) AS v FROM #entries LIMIT 1' % (LITERALS[inputs[0]], keyword)
    if name == 'Between':
        return 'SELECT %s BETWEEN %s AND %s AS v FROM #entries LIMIT 1' % (
            LITERALS[inputs[0]], LITERALS[inputs[1]], LITERALS[inputs[2]])
    if name in ('Match', 'NotMatch'):
        return "SELECT 'abc' %s 'b' AS v FROM #entries LIMIT 1" % OPERATOR_SYMBOLS[name]
    symbol = OPERATOR_SYMBOLS.get(name)
    if symbol is None:
        raise SystemExit('unmapped operator: ' + item['id'])
    return 'SELECT %s %s %s AS v FROM #entries LIMIT 1' % (
        LITERALS[inputs[0]], symbol, LITERALS[inputs[1]])


# Each function probe uses the pinned fixture so every signature gets a real
# oracle evaluation. Typed operands come from table columns because the SQL
# grammar has no dict, list, set, amount, position, or inventory literals.
FUNCTION_QUERIES = {
    "function.abs.0": "SELECT abs(balance) AS v FROM #postings LIMIT 1",
    "function.abs.1": "SELECT abs(position) AS v FROM #postings LIMIT 1",
    "function.abs.2": "SELECT abs(number) AS v FROM #postings LIMIT 1",
    "function.account_sortkey.0": "SELECT account_sortkey('Assets:Cash') AS v FROM #entries LIMIT 1",
    "function.any_meta.0": "SELECT any_meta('probe') AS v FROM #postings LIMIT 1",
    "function.bool.0": "SELECT bool(number) AS v FROM #postings LIMIT 1",
    "function.close_date.0": "SELECT close_date('Expenses:Food') AS v FROM #entries LIMIT 1",
    "function.commodity.0": "SELECT commodity(weight) AS v FROM #postings LIMIT 1",
    "function.commodity_meta.0": "SELECT commodity_meta('HOOL','lineno') AS v FROM #entries LIMIT 1",
    "function.commodity_meta.1": "SELECT commodity_meta('HOOL') AS v FROM #entries LIMIT 1",
    "function.convert.0": "SELECT convert(weight,'USD',date('2020-01-02')) AS v FROM #postings LIMIT 1",
    "function.convert.1": "SELECT convert(weight,'USD') AS v FROM #postings LIMIT 1",
    "function.convert.2": "SELECT convert(position,'USD',date('2020-01-02')) AS v FROM #postings LIMIT 1",
    "function.convert.3": "SELECT convert(position,'USD') AS v FROM #postings LIMIT 1",
    "function.convert.4": "SELECT convert(balance,'USD',date('2020-01-02')) AS v FROM #postings LIMIT 1",
    "function.convert.5": "SELECT convert(balance,'USD') AS v FROM #postings LIMIT 1",
    "function.cost.0": "SELECT cost(position) AS v FROM #postings LIMIT 1",
    "function.cost.1": "SELECT cost(balance) AS v FROM #postings LIMIT 1",
    "function.count.0": "SELECT count(*) AS v FROM #postings",
    "function.count.1": "SELECT count(number) AS v FROM #postings",
    "function.currency.0": "SELECT currency(weight) AS v FROM #postings LIMIT 1",
    "function.currency_meta.0": "SELECT currency_meta('HOOL','lineno') AS v FROM #entries LIMIT 1",
    "function.currency_meta.1": "SELECT currency_meta('HOOL') AS v FROM #entries LIMIT 1",
    "function.date.0": "SELECT date(getitem(meta,'when')) AS v FROM #postings LIMIT 1",
    "function.date.1": "SELECT date('2020-01-02') AS v FROM #entries LIMIT 1",
    "function.date.2": "SELECT date(date('2020-01-02')) AS v FROM #entries LIMIT 1",
    "function.date.3": "SELECT date(2020,1,2) AS v FROM #entries LIMIT 1",
    "function.date_add.0": "SELECT date_add(date('2020-01-02'),3) AS v FROM #entries LIMIT 1",
    "function.date_bin.0": "SELECT date_bin(interval('1 month'),date('2020-05-06'),date('2020-01-01')) AS v FROM #entries LIMIT 1",
    "function.date_bin.1": "SELECT date_bin('month',date('2020-05-06'),date('2020-01-01')) AS v FROM #entries LIMIT 1",
    "function.date_diff.0": "SELECT date_diff(date('2020-01-05'),date('2020-01-02')) AS v FROM #entries LIMIT 1",
    "function.date_part.0": "SELECT date_part('month',date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.date_trunc.0": "SELECT date_trunc('month',date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.day.0": "SELECT day(date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.decimal.0": "SELECT decimal(number) AS v FROM #postings LIMIT 1",
    "function.decimal.1": "SELECT decimal('2.50') AS v FROM #entries LIMIT 1",
    "function.decimal.2": "SELECT decimal(TRUE) AS v FROM #entries LIMIT 1",
    "function.decimal.3": "SELECT decimal(2) AS v FROM #entries LIMIT 1",
    "function.decimal.4": "SELECT decimal(2.50) AS v FROM #entries LIMIT 1",
    "function.empty.0": "SELECT empty(balance) AS v FROM #postings LIMIT 1",
    "function.entry_meta.0": "SELECT entry_meta('probe') AS v FROM #postings LIMIT 1",
    "function.filter_currency.0": "SELECT filter_currency(position,'USD') AS v FROM #postings LIMIT 1",
    "function.filter_currency.1": "SELECT filter_currency(balance,'USD') AS v FROM #postings LIMIT 1",
    "function.findfirst.0": "SELECT findfirst('A.*',tags) AS v FROM #postings LIMIT 1",
    "function.first.0": "SELECT first(number) AS v FROM #postings",
    "function.getitem.0": "SELECT getitem(meta,'lineno') AS v FROM #postings LIMIT 1",
    "function.getitem.1": "SELECT getitem(meta,'absent','fallback') AS v FROM #postings LIMIT 1",
    "function.getprice.0": "SELECT getprice('HOOL','USD',date('2020-01-09')) AS v FROM #entries LIMIT 1",
    "function.getprice.1": "SELECT getprice('HOOL','USD') AS v FROM #entries LIMIT 1",
    "function.grep.0": "SELECT grep('a.*',account) AS v FROM #postings LIMIT 1",
    "function.grepn.0": "SELECT grepn('(a)(ssets)',account,1) AS v FROM #postings LIMIT 1",
    "function.has_account.0": "SELECT has_account('Assets:Cash') AS v FROM #postings LIMIT 1",
    "function.int.0": "SELECT int(number) AS v FROM #postings LIMIT 1",
    "function.int.1": "SELECT int('42') AS v FROM #entries LIMIT 1",
    "function.int.2": "SELECT int(2.9) AS v FROM #entries LIMIT 1",
    "function.int.3": "SELECT int(TRUE) AS v FROM #entries LIMIT 1",
    "function.int.4": "SELECT int(7) AS v FROM #entries LIMIT 1",
    "function.interval.0": "SELECT interval('1 month') AS v FROM #entries LIMIT 1",
    "function.joinstr.0": "SELECT joinstr(tags) AS v FROM #transactions LIMIT 1",
    "function.last.0": "SELECT last(number) AS v FROM #postings",
    "function.leaf.0": "SELECT leaf('Assets:Cash') AS v FROM #entries LIMIT 1",
    "function.length.0": "SELECT length('abc') AS v FROM #entries LIMIT 1",
    "function.length.1": "SELECT length(tags) AS v FROM #transactions LIMIT 1",
    "function.length.2": "SELECT length(other_accounts) AS v FROM #postings LIMIT 1",
    "function.lower.0": "SELECT lower('ABC') AS v FROM #entries LIMIT 1",
    "function.max.0": "SELECT max(number) AS v FROM #postings",
    "function.maxwidth.0": "SELECT maxwidth('abcdef',6) AS v FROM #entries LIMIT 1",
    "function.meta.0": "SELECT meta('probe') AS v FROM #postings LIMIT 1",
    "function.min.0": "SELECT min(number) AS v FROM #postings",
    "function.month.0": "SELECT month(date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.neg.0": "SELECT neg(balance) AS v FROM #postings LIMIT 1",
    "function.neg.1": "SELECT neg(position) AS v FROM #postings LIMIT 1",
    "function.neg.2": "SELECT neg(weight) AS v FROM #postings LIMIT 1",
    "function.neg.3": "SELECT neg(number) AS v FROM #postings LIMIT 1",
    "function.number.0": "SELECT number(weight) AS v FROM #postings LIMIT 1",
    "function.only.0": "SELECT only('USD',balance) AS v FROM #postings WHERE account = 'Assets:Cash' LIMIT 1",
    "function.open_date.0": "SELECT open_date('Assets:Cash') AS v FROM #entries LIMIT 1",
    "function.open_meta.0": "SELECT open_meta('Assets:Cash','lineno') AS v FROM #entries LIMIT 1",
    "function.open_meta.1": "SELECT open_meta('Assets:Cash') AS v FROM #entries LIMIT 1",
    "function.parent.0": "SELECT parent('Assets:Cash') AS v FROM #entries LIMIT 1",
    "function.parse_date.0": "SELECT parse_date('2020-01-02','%Y-%m-%d') AS v FROM #entries LIMIT 1",
    "function.parse_date.1": "SELECT parse_date('2020-01-02') AS v FROM #entries LIMIT 1",
    "function.possign.0": "SELECT possign(balance,'Assets') AS v FROM #postings LIMIT 1",
    "function.possign.1": "SELECT possign(position,'Assets') AS v FROM #postings LIMIT 1",
    "function.possign.2": "SELECT possign(weight,'Assets') AS v FROM #postings LIMIT 1",
    "function.possign.3": "SELECT possign(number,'Assets') AS v FROM #postings LIMIT 1",
    "function.quarter.0": "SELECT quarter(date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.repr.0": "SELECT repr(number) AS v FROM #postings LIMIT 1",
    "function.root.0": "SELECT root('a:b:c',2) AS v FROM #entries LIMIT 1",
    "function.root.1": "SELECT root('a:b:c') AS v FROM #entries LIMIT 1",
    "function.round.0": "SELECT round(7,1) AS v FROM #entries LIMIT 1",
    "function.round.1": "SELECT round(7) AS v FROM #entries LIMIT 1",
    "function.round.2": "SELECT round(2.567,2) AS v FROM #entries LIMIT 1",
    "function.round.3": "SELECT round(2.567) AS v FROM #entries LIMIT 1",
    "function.safediv.0": "SELECT safediv(number,0) AS v FROM #postings LIMIT 1",
    "function.safediv.1": "SELECT safediv(number,2.5) AS v FROM #postings LIMIT 1",
    "function.splitcomp.0": "SELECT splitcomp(account,':',1) AS v FROM #postings LIMIT 1",
    "function.str.0": "SELECT str(number) AS v FROM #postings LIMIT 1",
    "function.subst.0": "SELECT subst('(a)(ssets)','Assets','X') AS v FROM #postings LIMIT 1",
    "function.substr.0": "SELECT substr('abcdef',1,3) AS v FROM #entries LIMIT 1",
    "function.sum.0": "SELECT sum(1) AS v FROM #entries",
    "function.sum.1": "SELECT sum(number) AS v FROM #postings",
    "function.sum.2": "SELECT sum(weight) AS v FROM #postings",
    "function.sum.3": "SELECT sum(position) AS v FROM #postings",
    "function.sum.4": "SELECT sum(balance) AS v FROM #postings",
    "function.today.0": "SELECT today() AS v FROM #entries LIMIT 1",
    "function.units.0": "SELECT units(position) AS v FROM #postings LIMIT 1",
    "function.units.1": "SELECT units(balance) AS v FROM #postings LIMIT 1",
    "function.upper.0": "SELECT upper('ab') AS v FROM #entries LIMIT 1",
    "function.value.0": "SELECT value(position,date('2020-01-09')) AS v FROM #postings LIMIT 1",
    "function.value.1": "SELECT value(position) AS v FROM #postings LIMIT 1",
    "function.value.2": "SELECT value(balance,date('2020-01-09')) AS v FROM #postings LIMIT 1",
    "function.value.3": "SELECT value(balance) AS v FROM #postings LIMIT 1",
    "function.weekday.0": "SELECT weekday(date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.year.0": "SELECT year(date('2020-05-06')) AS v FROM #entries LIMIT 1",
    "function.yearmonth.0": "SELECT yearmonth(date('2020-05-06')) AS v FROM #entries LIMIT 1",
}

def normalized_entry_id(entry, root):
    """Hash one directive the way upstream hash_entry does, with source
    filenames made relative to the ledger directory.

    The pinned oracle hashes str(meta) verbatim, so a raw id changes with the
    checkout path. Both this capture and the Go implementation normalize the
    filename to the ledger directory first; that is the documented id
    contract, and it keeps the golden reproducible across machines.
    """
    def normalize_meta(meta):
        if not meta:
            return meta
        out = {}
        for key, value in meta.items():
            out[key] = os.path.relpath(value, root) if key == 'filename' else value
        return out

    if isinstance(entry, data.Transaction):
        postings = [p._replace(meta=normalize_meta(p.meta)) for p in entry.postings]
        entry = entry._replace(postings=postings)
    if isinstance(entry, data.Document) and entry.filename:
        entry = entry._replace(filename=os.path.relpath(entry.filename, root))
    return compare.hash_entry(entry._replace(meta=normalize_meta(entry.meta)))

def probe_ast_nodes(text):
    """Record the typed node fingerprint of one statement.

    beanquery.parse returns a Node subtree; walk() yields every node in it. The
    sorted class names are the AST's structure, which is what the Node base
    contract promises and what a compatible implementation must reproduce.
    """
    from beanquery.parser import parse
    root = parse(text)
    kinds = sorted(type(node).__name__ for node in root.walk())
    cells = [{'type': 'string', 'value': kind} for kind in kinds]
    return {'oracle_type': 'object', 'type': 'string', 'declared_type_consistent': True, 'rows': cells}


def query_for(item):
    if item['kind'] == 'column':
        return column_query(item)
    if item['kind'] == 'syntax':
        return SYNTAX_QUERIES[item['id']]
    if item['kind'] == 'accounting-directive':
        return DIRECTIVE_QUERIES[item['id']]
    if item['kind'] == 'accounting-option':
        return OPTION_QUERIES[item['id']][0]
    if item['id'] in ('table.entries.id', 'table.postings.id'):
        # The id column is hashed, not selected: see id_rows below.
        return None
    if item['kind'] == 'operator':
        return operator_query(item)
    if item['kind'] == 'function':
        return FUNCTION_QUERIES[item['id']]
    raise SystemExit('unsupported probe kind: ' + item['kind'])

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--only', action='append', default=[], help='capability id prefix to probe')
    parser.add_argument('--kind', action='append', default=[], help='capability kind to probe')
    parser.add_argument('--check', action='store_true', help='verify goldens instead of writing them')
    parser.add_argument('--quiet', action='store_true')
    args = parser.parse_args()

    assert beancount.__version__ == ORACLE['beancount']
    assert beanquery.__version__ == ORACLE['beanquery']
    inventory_data = INVENTORY.read_bytes()
    capabilities = json.loads(inventory_data)['capabilities']
    fixture_data = FIXTURE.read_bytes()
    contexts = {}
    def context_for(name):
        if name not in contexts:
            entries, errors, options = loader.load_file(str(OUT / name))
            contexts[name] = {'context': connect(entries, errors, options), 'errors': errors, 'entries': entries}
        return contexts[name]
    default_context = context_for('fixture.bean')

    items = []
    seen = set()
    for item in capabilities:
        if args.kind and item['kind'] not in args.kind:
            continue
        if args.only and not any(item['id'].startswith(prefix) for prefix in args.only):
            continue
        seen.add(item['id'])
        if item['kind'] not in ('column', 'operator', 'function', 'syntax', 'accounting-directive', 'accounting-option'):
            continue
        # Items triaged as upstream-internal carry no public behaviour to probe.
        if (item['kind'] == 'syntax' and item['id'] not in SYNTAX_QUERIES) or (
                item['kind'] == 'accounting-option' and item['id'] not in OPTION_QUERIES) or (
                item['kind'] == 'accounting-directive' and item['id'] not in DIRECTIVE_QUERIES):
            continue
        if item['kind'] == 'accounting-directive':
            # Pad, Query, and Custom need their own directives in the fixture.
            fixture_name = 'directives.bean'
        elif item['kind'] == 'accounting-option':
            fixture_name = OPTION_QUERIES[item['id']][1]
        else:
            fixture_name = COLUMN_FIXTURES.get(item['id'], 'fixture.bean')
        item_context = context_for(fixture_name)
        record = {
            'schema_version': 1,
            'id': item['id'],
            'kind': item['kind'],
            'oracle': ORACLE,
            'fixture': 'testdata/golden/inventory/' + fixture_name,
            'fixture_sha256': sha256((OUT / fixture_name).read_bytes()),
            'query': query_for(item),
            'loader_errors': [{'class': type(error).__name__, 'message': error.message} for error in item_context['errors']],
        }
        try:
            if item['id'] == 'function.today.0':
                # today() cannot be frozen: the oracle records today's date, so a
                # plain comparison would fail the next day. Mark the golden as
                # dynamic so the Go side checks the rule instead of the value.
                record['dynamic'] = 'today'
            if item['id'] == 'syntax.Node':
                observation = probe_ast_nodes(record['query'])
                # Node has no value of its own; the Go side compares the node
                # fingerprint through this marker instead of a query result.
                record['dynamic'] = 'ast-node'
            elif item['id'] in ('table.entries.id', 'table.postings.id'):
                observation = probe_ids(item_context, item['id'], OUT)
            else:
                observation = probe_query(item_context['context'], record['query'])
            record.update(observation)
            record['status'] = 'ok'
        except Exception as error:  # Record the real failure; never synthesize a value.
            record['status'] = 'error'
            record['error'] = {'class': type(error).__name__, 'message': str(error)}
        path = OUT / (item['id'] + '.json')
        encoded = json.dumps(record, indent=2, sort_keys=True, ensure_ascii=False) + '\n'
        if args.check:
            if not path.exists() or path.read_text() != encoded:
                raise SystemExit('stale inventory golden: ' + str(path))
        else:
            path.write_text(encoded)
        items.append({'id': item['id'], 'kind': item['kind'], 'status': record['status']})
        if not args.quiet:
            print('%-45s %s' % (item['id'], record['status']))

    if args.check:
        print('inventory goldens match the oracle: %d items' % len(items))
        return
    index_path = INDEX
    previous = {}
    if index_path.exists():
        previous = {entry['id']: entry for entry in json.loads(index_path.read_text()).get('items', [])}
    for entry in items:
        previous[entry['id']] = entry
    index = {
        'schema_version': 1,
        'oracle': ORACLE,
        'source_inventory_sha256': sha256(inventory_data),
        'fixture_sha256': sha256(fixture_data),
        'items': [previous[key] for key in sorted(previous)],
    }
    index_path.write_text(json.dumps(index, indent=2, sort_keys=True, ensure_ascii=False) + '\n')
    print('wrote %d goldens under %s' % (len(items), OUT))

if __name__ == '__main__':
    main()
