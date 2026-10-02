#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
"""Same-filesystem glob oracle using only public generated source names."""
import glob
import json
import subprocess
import tempfile
from pathlib import Path
from beancount import loader
from beancount.core import data
REPO = Path(__file__).resolve().parents[2]
PATTERNS = ['parts/*.bean', 'parts/**/*.bean', 'parts/**', 'parts/**/', 'parts/?.bean',
            'parts/[ab].bean', 'parts/[!ab].bean', 'parts/[^a].bean', 'parts/[a-cz-a].bean',
            'parts/[a--z].bean', 'parts/[[]x.bean', 'parts/[]a].bean', 'parts/.hidden.bean',
            'parts/.*/*.bean', 'parts/[.]hidden.bean', 'parts/子目录/*.bean',
            'parts/no-match*.bean', 'parts/[x.bean', 'parts/back\\slash.bean']
def main():
    with tempfile.TemporaryDirectory(prefix='orangecount-public-loading-') as directory:
        root = Path(directory)
        binary = root / 'probe'
        subprocess.run(['go','build','-o',str(binary),'./tools/sourceprobe'],cwd=REPO,check=True)
        names = ['parts/z.bean','parts/a.bean','parts/b.bean','parts/c.bean','parts/^.bean',
                 'parts/[x.bean','parts/].bean','parts/back\\slash.bean','parts/.hidden.bean',
                 'parts/nested/d.bean','parts/子目录/账本.bean','parts/.secret/e.bean']
        for name in names:
            file=root/name;file.parent.mkdir(parents=True,exist_ok=True);file.write_text('; public fixture\n')
        entry=root/'main.bean'
        for pattern in PATTERNS + [str(root/'parts/*.bean')]:
            entry.write_text('include '+json.dumps(pattern,ensure_ascii=False)+'\n')
            observed=json.loads(subprocess.check_output([str(binary),'-entry',str(entry)],cwd=REPO))
            expected=[str(Path(p).resolve().relative_to(root.resolve())) for p in glob.glob(str(root/pattern),recursive=True)]
            actual=[e['Path'] for e in observed['edges']]
            # Directory matches are load errors, so they have no graph edge.
            expected_files=[p for p in expected if (root/p).is_file()]
            assert actual==expected_files,(pattern,actual,expected_files)
            if not expected or len(expected_files)!=len(expected):
                assert 'E-INCLUDE-READ' in observed['errors'],pattern
        def txn(name, line=None):
            meta='' if line is None else f'  lineno: {line}\n'
            return f'2000-01-02 * "{name}"\n'+meta+'  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n'
        opens='2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n'
        cases=[
            {'main.bean':opens+txn('parent')+'include "child.bean"\n','child.bean':txn('child')},
            {'main.bean':opens+txn('parent',0)+'include "child.bean"\n','child.bean':txn('child')},
            {'main.bean':'include "a.bean"\ninclude "b.bean"\n'+opens,'a.bean':txn('a')+'include "c.bean"\n','b.bean':txn('b'),'c.bean':txn('c')},
            {'main.bean':opens+'include "child.bean"\ninclude "child.bean"\n','child.bean':txn('duplicate')},
            {'main.bean':opens+'include "child.bean"\n','child.bean':'include "main.bean"\n'},
            {'main.bean':opens+'2000-01-02 * "metadata"\n  Assets:Cash 1 USD\n    filename: "reserved"\n  Equity:Opening -1 USD\n'},
        ]
        for number, files in enumerate(cases):
            folder=root/f'case-{number}';folder.mkdir()
            for name,text in files.items():(folder/name).write_text(text)
            case_entry=folder/'main.bean'
            entries,errors,_=loader.load_file(str(case_entry))
            observed=json.loads(subprocess.check_output([str(binary),'-entry',str(case_entry),'-evaluate'],cwd=REPO))
            assert bool(errors)==bool(observed['errors']),(number,errors,observed)
            if not errors:
                expected=[e.narration for e in entries if isinstance(e,data.Transaction)]
                assert observed['narrations']==expected,(number,expected,observed)
        print(json.dumps({'glob_patterns':len(PATTERNS)+1,'load_semantic_cases':len(cases),'match_order_equal':True,'public_only':True}))
if __name__=='__main__':main()
