#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Extract pinned upstream capabilities; never mark presence as Go verification."""
import argparse
import hashlib
import inspect
import json
from pathlib import Path
import beancount
import beanquery
from beancount import loader
from beancount.core import data
from beancount.parser import options as parser_options
from beanquery import query_compile, types
from beanquery.parser import ast
from beanquery.sources import beancount as source

REPO = Path(__file__).resolve().parents[2]
DEST = REPO / 'docs/compatibility/upstream-inventory.json'

def typename(dtype):
    if dtype is types.Any:
        return "beanquery.types.Any"
    if isinstance(dtype, type):
        return dtype.__module__ + '.' + dtype.__qualname__
    return str(dtype)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    assert beancount.__version__ == '3.2.3' and beanquery.__version__ == '0.2.0'
    _, _, options = loader.load_string('')
    context = beanquery.connect('beancount:', entries=[], errors=[], options=options)
    root = Path(beanquery.__file__).parent
    hashes = {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest()
              for p in sorted(root.rglob('*.py'))}
    items = []
    for name, table in sorted(context.tables.items(), key=lambda item: str(item[0])):
        if name is None or name == '':
            continue
        for column, value in sorted(table.columns.items()):
            items.append({'id': f'table.{name}.{column}', 'upstream_symbol': type(table).__name__+'.'+column,
                          'kind': 'column', 'output': typename(value.dtype), 'status': 'pending',
                          'go_location': None, 'cases': []})
    for name, overloads in sorted(query_compile.FUNCTIONS.items()):
        for number, function in enumerate(overloads):
            inputs = list(function.__intypes__)
            values = [query_compile.EvalConstant(None, dtype) for dtype in inputs]
            try:
                result = typename(function(context, values).dtype)
            except Exception as error:
                result = {'audit_required': type(error).__name__}
            items.append({'id': f'function.{name}.{number}', 'upstream_symbol': function.__module__+'.'+function.__qualname__,
                          'kind': 'function', 'input': [typename(t) for t in inputs], 'output': result,
                          'status': 'pending', 'go_location': None, 'cases': []})
    for operator, overloads in sorted(query_compile.OPERATORS.items(), key=lambda item:item[0].__name__):
        for number, implementation in enumerate(overloads):
            inputs = list(implementation.__intypes__)
            values = [query_compile.EvalConstant(None, dtype) for dtype in inputs]
            result = typename(implementation(*values).dtype)
            items.append({'id':f'operator.{operator.__name__}.{number}', 'upstream_symbol':implementation.__module__+'.'+implementation.__name__,
                          'kind':'operator','input':[typename(t) for t in inputs],'output':result,
                          'status':'pending','go_location':None,'cases':[]})
    for name in sorted(parser_options.OPTIONS):
        items.append({'id':'option.'+name,'upstream_symbol':'beancount.parser.options.OPTIONS.'+name,
                      'kind':'accounting-option','status':'pending','go_location':None,'cases':[]})
    for cls in data.ALL_DIRECTIVES:
        items.append({'id':'directive.'+cls.__name__,'upstream_symbol':'beancount.core.data.'+cls.__name__,
                      'kind':'accounting-directive','fields':list(cls._fields),'status':'pending','go_location':None,'cases':[]})
    for name, cls in sorted(vars(ast).items()):
        if inspect.isclass(cls) and cls.__module__ == ast.__name__:
            items.append({'id': 'syntax.'+name, 'upstream_symbol': 'beanquery.parser.ast.'+name,
                          'kind': 'syntax', 'fields': list(getattr(cls, '_fields', [])),
                          'status': 'pending', 'go_location': None, 'cases': []})
    result = {'schema_version': 1, 'reference': {'beancount': '3.2.3', 'beanquery': '0.2.0'},
              'beanquery_source_sha256': hashes,
              'beancount_loader_sha256': hashlib.sha256(Path(loader.__file__).read_bytes()).hexdigest(),
              'note': 'Extracted upstream presence only; diagnostic rules, grammar combinations and Go evidence require separate audit.',
              'capabilities': items}
    text = json.dumps(result, indent=2, sort_keys=True)+'\n'
    if args.check:
        assert DEST.read_text() == text, 'upstream inventory changed'
    else:
        DEST.parent.mkdir(parents=True, exist_ok=True)
        DEST.write_text(text)
    print(json.dumps({'capabilities': len(items), 'columns': sum(i['kind']=='column' for i in items),
                      'overloads': sum(i['kind']=='function' for i in items),
                      'syntax_nodes': sum(i['kind']=='syntax' for i in items),
                      'operators': sum(i['kind']=='operator' for i in items),
                      'options': sum(i['kind']=='accounting-option' for i in items),
                      'directives': sum(i['kind']=='accounting-directive' for i in items),
                      'unresolved_outputs': sum(isinstance(i.get('output'),dict) for i in items)}))

if __name__ == '__main__':
    main()
