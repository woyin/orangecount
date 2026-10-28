---
name: orangecount-authoring
description: >-
  Change an OrangeCount ledger safely: add or edit entries, use quick entry,
  import a file, or hand-edit a .bean source. Use whenever a write to a ledger
  is requested.
  Triggers: 记账, 添加条目, 修改账本, 编辑账本, 快速录入, 导入, 删除条目, 修正错误.
---

# OrangeCount: writing to a ledger

There is no CLI writer. Writes happen through the web workbench or by editing
the `.bean` source directly. Both must preserve the same invariants.

## Invariants

1. The source `.bean` include graph is the only source of truth. There is no
   side database.
2. Every write is followed by a full re-evaluation. If it fails, the write is
   rolled back from the backup and the diagnostics are returned.
3. The previous content is kept as `<file>.orangecount.bak`.
4. Never change an amount, date, or account the user did not mention.
5. Never delete history to make a diagnostic disappear.

## Path A: the reviewed web workflows

Start the workbench first (see `orangecount-workbench`), then use:

| Task | Surface |
| --- | --- |
| Add one structured directive | `+` then Add Entry |
| Fast capture of several two-posting transactions | `+` then the Quick tab |
| Edit raw source with syntax awareness | `/editor` |
| Import a file and review the diff | `/import` |
| Attach or move a receipt | `/documents` |

Quick entry grammar is strict, two forms, no natural language:

```text
# explicit form: AMOUNT [CURRENCY] alias -> alias : narration #tag ^link
28 CNY @微信 -> @餐饮 : 工作午餐 #报销

# template form: template-name AMOUNT [CURRENCY] [counterparty]
午餐 28 @微信
```

Both postings always carry an explicit amount and currency, signs follow value
flow, and the generated block carries no private metadata. A non-unique alias is
a compile error, never a guess. Preview first, then commit: the commit is one
atomic append and is undoable as the session's last action.

## Path B: hand-editing the source

Use this for content the modal forms cannot express: metadata, `custom`
directives, reordering, bulk edits.

```sh
bin/orangecount check --semantics v3 "$LEDGER"    # 1. baseline must be clean
cp "$LEDGER" "$LEDGER.manual.bak"                 # 2. your own escape hatch
# 3. edit with the edit tool, preserving surrounding formatting
bin/orangecount check --semantics v3 "$LEDGER"    # 4. must still be clean
```

Rules for this path:

- Keep the file's existing style: same date format, same two-space indentation,
  same metadata key order.
- Put a new transaction next to its date neighbours. Evaluation sorts by date,
  so position never changes accounting, but human review depends on it.
- One change per edit. Do not reformat untouched lines; an unrelated hunk is a
  review liability.
- Do not commit `.orangecount.bak` or your own backups. Check `.gitignore`
  before creating new file patterns in a tracked ledger directory.

## Verify the write

```sh
bin/orangecount check --semantics v3 "$LEDGER"
bin/orangecount query --format typed-csv --semantics v3 "$LEDGER" \
  "SELECT date, narration, account, number FROM postings WHERE date >= '2026-01-01' ORDER BY date"
```

Then confirm the specific entry the user asked for. A valid ledger can still
contain the wrong amount, so validity is not acceptance.

## Rollback

```sh
cp "$LEDGER.orangecount.bak" "$LEDGER"   # after a web write
cp "$LEDGER.manual.bak" "$LEDGER"        # after a hand edit
```

Report the diagnostic code and its guide (`orangecount help diagnostics/<CODE>`)
rather than guessing a repair.

## Never do this

- Reformat or sort the whole file.
- Change an `option` value to make an account error disappear. The account is
  usually the wrong part.
- Delete a transaction because a balance assertion fails. Fix the assertion or
  the posting, and ask the user which one is wrong.
- Write while the workbench holds an unsaved editor buffer; reload it first or
  the buffer will overwrite your change.
- Write to a ledger you have not `check`ed first; you will not be able to tell
  your breakage from pre-existing breakage.
