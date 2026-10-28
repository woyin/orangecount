---
name: orangecount-workbench
description: >-
  Start and drive the OrangeCount web workbench: serve a ledger locally, reach
  the standard report pages, read the HTTP API, export query results to
  XLSX/ODS, or export a pure Beancount v3 snapshot.
  Triggers: 打开工作台, 启动服务, 看报表, 导出, 导出xlsx, 导出v3, dialectize.
---

# OrangeCount: workbench, reports, export

## Start the server

```sh
bin/orangecount serve --semantics v3 --addr 127.0.0.1:5100 "$LEDGER"
```

- `--semantics v3` for a v3 ledger. The default is `legacy`.
- `--addr` must be loopback; the server refuses a non-loopback bind.
- `--locale zh-CN` switches the UI and diagnostics language.
- `--document-root DIR` points receipts at a directory outside the ledger.
- `--sensitive-logs` only while debugging locally; it can print ledger values.

The default port is 5000, which macOS AirPlay Receiver occupies. Use a free high
port and confirm it first:

```sh
lsof -iTCP:5100 -sTCP:LISTEN -n -P     # no output means the port is free
```

## Standard pages

```text
/                          income statement (landing page)
/income_statement          /balance_sheet        /trial_balance
/journal                   /query                /holdings
/holdings/by_account       /holdings/by_currency /holdings/by_cost_currency
/holdings/by_root_account  /holdings/by_commodity
/commodities               /documents            /events
/statistics                /editor               /import
/options                   /help                 /source
/diagnostics               /errors               /quick-profile
/planning
```

## HTTP API

Read-only unless stated:

```text
GET /api/v1/status
GET /api/v1/diagnostics
GET /api/v1/reports/balance-sheet?format=json
GET /api/v1/reports/income-statement?format=json
GET /api/v1/reports/trial-balance?format=json
GET /api/v1/query?q=<BQL>&format=json|csv|xlsx|ods
GET /api/v1/source
GET /api/v1/options
```

`/api/v1/query` is the scripting path for spreadsheets. `format=xlsx` or `ods`
streams a workbook download:

```sh
curl -sG 'http://127.0.0.1:5100/api/v1/query' \
  --data-urlencode "q=SELECT account, sum(number) AS total FROM postings WHERE account ~ '^Expenses' GROUP BY account" \
  --data-urlencode format=xlsx -o expenses.xlsx
```

The write endpoints (`/api/v1/editor`, `/api/v1/import`, add-entries, document
upload and move) are the same reviewed path the UI uses. Prefer the UI unless
the user explicitly asked for automation; see `orangecount-authoring`.

## Export a pure Beancount v3 snapshot

For handing the ledger to another Beancount tool (Fava, the `beancount` CLI):

```sh
mkdir -p /path/outside/the/ledger
bin/orangecount export --semantics v3 -out /path/outside/the/ledger/book.bean "$LEDGER"
```

- The output must live **outside** the ledger's entry directory. A single
  existing file may be replaced; an include graph needs a brand-new directory.
- It refuses what it cannot represent in v3: AVERAGE booking, Python plugins,
  unregistered options, list or map metadata, and a non-global
  `account_root_scheme` vocabulary.
- `option "account_root_scheme" "global"` is dropped from the snapshot, because
  upstream rejects an unknown option key.
- It reloads the staged output and compares accounting before publishing, so a
  refused export leaves the target untouched.

`dialectize` rewrites the OrangeCount dialect shorthand into standard v3
syntax. It also requires `-out` and uses the same protections.

```sh
bin/orangecount dialectize -out /path/outside/the/ledger/book.bean "$LEDGER"
```

## Pitfalls

- Starting `serve` without `--semantics v3` on a v3 ledger shows different
  numbers than `check --semantics v3`.
- A file changed outside the workbench is not visible until the workbench
  reloads. Check `/api/v1/status` before trusting a report.
- The server is loopback-only; `--addr 0.0.0.0` is rejected. Forward a port to
  reach it from another machine.
- Do not claim a report reflects an edit unless the snapshot was reloaded after
  that edit.
