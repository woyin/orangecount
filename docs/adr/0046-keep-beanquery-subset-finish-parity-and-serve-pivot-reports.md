# Keep the BeanQuery subset; finish its column parity honestly and serve pivot reports separately

2026-10-02范围修订：用户要求完整查询生态；固定上游实际支持子查询与PIVOT BY。本文相关排除理由已被[ADR-0050](0050-require-complete-plugin-free-beancount-and-bql-compatibility.md)修订，下文保留历史，不作为当前能力上限。

Users hit "cross-month merge and comparison feel impossible" and asked whether the
query language should become full SQL. Investigation showed the pain was three
distinct layers: (A) silent failures — unknown columns like `amount` evaluated to
0 and bare `year`/`month` group keys collapsed all rows into one — which violated
the honest-filtering principle; (B) unimplemented BeanQuery standard features —
the `~` regex operator, the derived `year`/`month`/`day` posting columns, the
`balance` running-balance and `weight` columns, and the `root(account, n)`
function that upstream beanquery ships and ADR-0016 promised; and (C) full SQL
(JOIN, subqueries, CTEs, window functions) which upstream beanquery deliberately
does not provide. We decided to fix A, complete B, and reject C: the report needs
(an Excel-style pivot: rows × columns × values, including period-end balances)
are served by a dedicated report surface over the same engine, not by growing the
query language into a SQL engine. Unknown columns now fail loudly with the list
of valid columns instead of returning zeroes.

## Considered Options

- **Full SQL engine** — rejected: no zero-dependency path; Fava's user base
  orders of magnitude larger has not needed it; the actual reporting gap is
  presentation, not query power.
- **PIVOT syntax inside BQL** — rejected: diverges from upstream beanquery
  semantics ADR-0016 committed to; pivoting is a view over grouped results.

## Consequences

- `sum(amount)`-style queries now error ("unknown column") instead of returning
  plausible-looking zeroes; existing silent-zero users must switch to real
  column names (`number`, `units`).
- The pivot report computes period-end balances from the interval machinery in
  `internal/report`, not from per-posting `balance` aggregation, because
  inventories are multi-currency and do not sum.
