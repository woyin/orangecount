---
name: orangecount-ledger
description: >-
  Validate and query an OrangeCount ledger (a Beancount v3 superset). Use when
  asked to check a ledger, explain or fix a diagnostic code, run a BQL query,
  compute balances or spending totals, or compare legacy and v3 accounting.
  Triggers: 校验账本, 检查账本, 账本报错, 查余额, 查支出, 统计, 查询, 对账, 语义对比.
---

# OrangeCount: validate and query (read-only)

This skill only reads. Every step that modifies a ledger belongs to
`orangecount-authoring`.

## 0. Locate the binary and the entry file

```sh
make build                                    # -> bin/orangecount
LEDGER=~/Documents/FinanceBook/main.bean      # the entry file
```

Pass the **entry** file, the one holding `include` lines. The tool walks the
include graph itself; never point it at a random leaf `.bean`.

## 1. Validate before anything else

```sh
bin/orangecount check --semantics v3 "$LEDGER"
```

- Exit 0 means no error-severity diagnostic, and prints nothing.
- Exit 1 means at least one error; each line is `path:line:column: error CODE`.
- `--locale zh-CN` prints Chinese messages. `--json` gives machine output.

Run this even when the user only asked a question. A query against an invalid
ledger returns numbers you cannot defend.

Pass `--semantics v3` for a v3 ledger. **The default is `legacy`** and the two
profiles can differ. See section 4.

## 2. Explain and repair a diagnostic

```sh
bin/orangecount help diagnostics/E-EVAL-POSTING
```

Each code has a guide with the phase, what it means, what to inspect, safe
steps, and an example. Read the guide before proposing a fix; do not invent a
repair from the one-line message.

| Code | Meaning |
| --- | --- |
| `E-EVAL-ACCOUNT` | root is not a configured account type, or the name is malformed |
| `E-EVAL-POSTING` | posting falls outside the account open/close lifecycle |
| `E-EVAL-UNBALANCED` | transaction does not balance |
| `E-EVAL-INFER` | an omitted posting amount cannot be inferred |
| `E-EVAL-BALANCE` | balance assertion failed |
| `E-EVAL-CURRENCY` | posting currency is not allowed for the account |
| `E-EVAL-INVENTORY` | posting consumes more inventory than exists |
| `E-EVAL-PAD` | pad source account is unavailable |
| `E-EVAL-PAD-UNUSED` | a pad was never consumed by a balance assertion |
| `E-EVAL-OPTION` | unsupported option, bad value, or colliding root names |
| `E-PARSE-DIRECTIVE` | unknown or malformed directive |
| `E-SOURCE-UTF8` | source file is not valid UTF-8 |

## 3. Query

There are **two engines with different schemas**. Pick deliberately.

| Format flag | Engine | `accounts` columns |
| --- | --- | --- |
| `--format json` / `csv` | legacy query engine | `account`, `currency`, `balance`, `opened` |
| `--format typed-json` / `typed-csv` | typed BeanQuery subset | `account` only |

The typed `postings` table exposes `account`, `currency`, `number`, `balance`,
`date`, `year`, `month`, `day`, `payee`, `narration`, `flag`, `tags`, `links`,
`cost`, `weight`, `price`, `id`, `type`.

Verified queries:

```sql
-- top balances per account (typed)
SELECT account, sum(number) AS total FROM postings GROUP BY account ORDER BY total DESC

-- spending per account, regex filter (typed; LIKE does not exist)
SELECT account, sum(number) AS total FROM postings WHERE account ~ '^Expenses' GROUP BY account ORDER BY total DESC

-- monthly spending (typed)
SELECT year, month, sum(number) AS total FROM postings WHERE account ~ '^Expenses' GROUP BY year, month ORDER BY year, month

-- per-account per-currency balances (legacy engine, needs --format csv)
SELECT account, currency, sum(balance) AS balance FROM accounts GROUP BY account, currency ORDER BY account

-- journal slice (legacy engine)
SELECT date, narration, account, number, currency FROM postings WHERE date >= '2026-01-01' ORDER BY date

-- counts
SELECT count(*) AS n FROM postings
```

Rules that change the answer:

- Inventories are not summed across currencies. A cross-currency total is
  meaningless; filter by `currency` or query one account.
- The typed engine rejects `accounts.currency`. Use the legacy `csv` format for
  that, or group `postings` by `currency` instead.
- Decimals are exact. Do not reformat them as floats.
- `--json` and `--csv` are aliases for the legacy formats, not for typed.

## 4. Compare accounting semantics

```sh
bin/orangecount compare-semantics "$LEDGER"
bin/orangecount compare-semantics --details --fail-on-change "$LEDGER"
```

Use before recommending a migration from `legacy` to `v3`. It reports every
accounting difference instead of guessing.

## 5. When the answer is a report, not a query

Balance sheet, income statement, and trial balance are rendered by the web
workbench; there is no CLI report subcommand. Either start the workbench and
read the JSON API (see `orangecount-workbench`):

```sh
curl -s 'http://127.0.0.1:5100/api/v1/reports/balance-sheet?format=json'
curl -s 'http://127.0.0.1:5100/api/v1/reports/income-statement?format=json'
curl -s 'http://127.0.0.1:5100/api/v1/reports/trial-balance?format=json'
```

or reproduce the shape with the queries in section 3. Snake_case report names
(`balance_sheet`) work too.

## 6. Pitfalls

- Forgetting `--semantics v3` silently uses `legacy`.
- `check` prints nothing on success; read the exit code, do not grep for
  "0 errors".
- A ledger declaring `option "account_root_scheme"` with `zh-Hans`, `zh-Hant`,
  `ja`, or `ko` roots its accounts in localized names, so
  `WHERE account ~ '^Assets'` matches nothing. Read the scheme first, then
  filter with the matching root (资产, 資產, 資産, 자산).
- A path diagnostic may be redacted in JSON output; `--json` is safe to paste
  into a report, but it will omit absolute paths by design.
