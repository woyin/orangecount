# Financial Planning and Cycle Review: Design and Implementation Plan

This document captures the shared understanding reached in the
`$grill-with-docs` session after Quick Entry shipped. It defines a conservative
safe-to-spend planning model, its owner-confirmed inputs, its relationship to
the source ledger, and a phased delivery that establishes planning before
cycle review.

The canonical vocabulary is in [CONTEXT.md](../CONTEXT.md). The persistence
decision is recorded in
[ADR-0044](adr/0044-embed-planning-assumptions-in-the-ledger.md), and the
boundary from category budgeting is recorded in
[ADR-0045](adr/0045-build-liquidity-planning-before-category-budgets.md).

## Product outcome

OrangeCount is a complete personal ledger rather than a remembered-spending
log. Quick Entry reduces the cost of recording what happened. The next product
capability uses that ledger to support two connected owner decisions:

1. Review the financial cycle that just ended, focusing on meaningful
   differences between confirmed plans and actual ledger activity.
2. Decide how much currently held money remains safe to spend before the next
   stable income, and whether a proposed large purchase fits without violating
   known obligations or the owner's minimum reserve.

The resulting loop is:

```text
actual ledger activity
  -> explain plan variances
  -> propose evidence-backed next-cycle candidates
  -> owner confirms plans
  -> calculate safe-to-spend timeline
  -> evaluate temporary affordability scenarios
```

OrangeCount explains constraints and trade-offs. It does not approve a
purchase, give financial advice, or silently convert a historical pattern into
a future fact.

## Delivery order

### Phase A — Planning foundation and safe-to-spend

Deliver the planning profile, plan maintenance, deterministic daily timeline,
primary safe-to-spend amount, funding-shortfall state, and ephemeral
affordability scenarios.

### Phase B — Financial-cycle review

After at least one cycle can have confirmed plans as a baseline, add guided
plan-to-actual matching, material variances, overdue-plan resolution,
explainable recurring candidates, review completion, and stale-review
detection.

A review built before planning would have no confirmed baseline and would
collapse into category statistics already covered by existing reports.

## Phase A decisions

### Planning horizon

- The default safe-to-spend horizon ends immediately before the next expected
  occurrence of the owner's financial-cycle anchor.
- The anchor is one explicitly selected recurring income rule. Salary,
  reimbursements, and incidental income do not become anchors automatically.
- Without an anchor, the owner must choose an explicit end date.
- Dates are interpreted in an owner-confirmed IANA planning timezone. The
  machine timezone is only an initialization suggestion.

### Planning inputs

Before OrangeCount presents a safe-to-spend result, the owner must confirm:

- one planning currency;
- an explicit list of spendable-funds accounts;
- an explicit list of applicable short-term debt accounts;
- a minimum reserve;
- a financial-cycle anchor or explicit horizon date;
- current committed outflows; and
- the date through which the owner believes real financial activity has been
  recorded in the source ledger.

An incomplete setup produces a checklist, never a guessed amount. A result is
in current planning status only when the ledger-recorded-through date is today
in the planning timezone. Older results remain visible as stale estimates and
are not labelled safe.

### Funds and debt

- Only explicitly selected, same-planning-currency accounts enter the
  spendable-funds pool.
- Newly discovered asset accounts never enter automatically.
- Selected accounts are treated as transferable within the horizon. An
  account-level deficit creates a planned transfer need; it does not create an
  aggregate funding shortfall while the combined pool remains sufficient.
- Valuable but unavailable assets—term deposits, securities, restricted
  savings, and other non-selected accounts—are excluded and remain visible.
- The full current balance of each selected short-term liability is deducted
  once, including credit-card purchases that have not yet appeared on a
  statement. Paying that liability later is a transfer and does not reduce
  aggregate capacity again.
- Other currencies are visible but excluded. Phase A performs no implicit
  valuation or foreign-exchange assumption.

### Plans

A planned cash flow requires only:

- a name;
- one expected date;
- one amount in the planning currency;
- inflow or outflow direction; and
- for an outflow, committed or adjustable classification.

Expected source/destination account and notes are optional. They improve
transfer reminders and later match suggestions but do not turn the plan into
a future Beancount transaction.

Uncertain values remain deterministic:

- use one owner-confirmed conservative amount, normally the upper credible
  amount for an outflow;
- use the earliest credible date for an outflow;
- use the latest credible date for an inflow; and
- ranges and probabilities may appear in notes but do not enter the first
  version calculation.

Recurring cash-flow rules generate candidates only. Every occurrence must be
confirmed or edited before it becomes a plan and affects a result.

### Primary safe-to-spend calculation

For each dated step `d` in the horizon:

```text
primary_headroom(d)
  = current spendable-funds pool
  - full current selected short-term debt
  - confirmed committed outflows due on or before d
  - confirmed adjustable outflows due on or before d
  - minimum reserve
```

The primary safe-to-spend amount is the lowest `primary_headroom(d)` across
the whole horizon, not the ending balance and not an average.

Additional rules:

- Confirmed but unrealized future inflows never increase the primary amount.
- Outflows are applied before inflows on the same date.
- A debt-funded planned purchase occupies capacity once on its economic date;
  its later repayment affects account-level liquidity but is not another
  aggregate outflow.
- When the minimum headroom is non-negative, it is the displayed primary
  safe-to-spend amount.
- When it is negative, the interface displays safe-to-spend as zero and shows
  the absolute funding shortfall, its first occurrence date, and contributing
  assumptions.

The primary value is deliberately the most conservative of four useful
views:

| Funding basis | Preserve all plans | Release adjustable plans |
| --- | --- | --- |
| Current funds only | **Primary value** | Current-funds floor scenario |
| Include confirmed future inflows | Expected conservative scenario | Expected floor scenario |

Only the primary value is a headline. The other three are explicit “what if”
views and must identify each relaxed assumption.

### Affordability scenarios

An affordability scenario accepts a proposed amount and date and explains:

- the resulting planning liquidity low point and date;
- reserve headroom or funding shortfall;
- which adjustable outflows would need to be displaced;
- whether the result depends on unrealized income;
- account-level transfer needs; and
- excluded currencies and unavailable assets.

It may state that the current conservative scenario does not support the
purchase, but it never decides whether the owner should proceed.

A scenario is ephemeral by default. It changes no planning result and writes
nothing to the ledger. Only an explicit “Add to plan” action, followed by
committed/adjustable confirmation and reviewed-write preview, persists it.

## Phase B decisions

### Review period and focus

- The default review covers the completed period between consecutive actual
  occurrences of the financial-cycle anchor, not a calendar month.
- Calendar-month analysis remains available through normal reports.
- The review focuses on planning variance rather than repeating category
  charts: amount or date changes, missed occurrence, duplication, overdue
  items, and loss or gain of planned headroom.
- Every missed, duplicated, or overdue committed outflow enters the primary
  review flow regardless of amount.
- Ordinary amount and timing differences use an owner-confirmed materiality
  threshold. Below-threshold items remain inspectable.

### Guided completion flow

1. Confirm the actual transactions that delimit the cycle.
2. Review suggested plan-fulfillment matches.
3. Confirm matches and resolve overdue items by fulfilling, rescheduling,
   cancelling, or recording the missing event.
4. Review material variances and the actual liquidity low point.
5. Inspect explainable historical patterns and confirm or reject next-cycle
   candidates.
6. Complete the review and enter the next horizon with an updated primary
   safe-to-spend amount.

Similarity in date, amount, account, or normalized description may create a
match suggestion, but never fulfills a plan automatically. An overdue outflow
continues to reserve funds until explicitly resolved.

Historical-pattern detection is local, deterministic, and explainable. Each
candidate exposes its supporting occurrences, cadence, normalized
description, accounts, and amount range. The first version uses no external AI
or opaque score.

### Completion and staleness

A completed review persists only compact confirmation facts:

- cycle boundaries;
- the supporting ledger-snapshot fingerprint;
- confirmed plan revisions;
- confirmed fulfillment matches; and
- completion time.

Charts, variances, and low points are derived again from source data. If
historical ledger activity or supporting planning assumptions change, the
review becomes stale. The previous conclusion remains visible, the ledger is
not locked, and the owner must reopen and confirm the review; OrangeCount never
silently rewrites it.

## Source-ledger representation

Planning data uses versioned standard Beancount `custom` directives and does
not change balances. The exact field spelling must be frozen by contract tests
before release, but the v1 representation follows this shape:

```beancount
2026-08-12 custom "orangecount.planning-profile.v1" "primary"
  currency: "CNY"
  timezone: "Asia/Singapore"
  minimum_reserve: 20000 CNY
  spendable_account: Assets:CMB:Checking
  spendable_account: Assets:WeChat
  short_term_debt_account: Liabilities:CMB:CreditCard
  recorded_through: 2026-08-12

2026-08-12 custom "orangecount.recurring-flow.v1" "salary"
  name: "工资"
  direction: "inflow"
  cadence: "monthly"
  cycle_anchor: TRUE

2026-08-12 custom "orangecount.planned-flow.v1" "rent-2026-09"
  revision: 1
  name: "房租"
  direction: "outflow"
  expected_date: 2026-09-01
  amount: 5000 CNY
  commitment: "committed"
  status: "active"
```

Each plan has a stable ID. Editing, rescheduling, cancelling, or fulfilling it
appends a higher revision rather than overwriting or deleting its history. The
latest valid revision is effective. Unsupported schema versions and malformed
planning directives produce isolated planning diagnostics; they do not make
the accounting ledger invalid or change reports.

Planning initialization is a reviewed write:

1. Suggest `planning.bean` beside the entry ledger, or allow selection of an
   existing included target.
2. Preview the new file, any required `include "planning.bean"`, and initial
   directives.
3. Confirm through the existing atomic, backed-up, revalidated write path.
4. Reject collisions, unauthorized paths, or a stale include graph.

All subsequent plan/profile writes use the same preview, snapshot guard,
atomic replacement, backup, and revalidation guarantees as other reviewed
writes.

## Interface

Planning is a clearly labelled OrangeCount extension page. It does not replace
the default landing page or compete with the Fava standard navigation surface.
After planning is ready, the existing home page may show a dismissible summary
link but remains otherwise unchanged.

The Planning page contains:

- setup/readiness checklist or current-status banner;
- one primary safe-to-spend headline;
- dated funding-shortfall state when applicable;
- funds timeline with its low point;
- current, overdue, and upcoming plans;
- account-level transfer needs;
- explicit scenario controls; and
- planning profile/settings.

Every result expands into its source accounts, debt balances, plans, reserve,
horizon, recorded-through date, and ordering assumptions. Other currencies and
excluded accounts remain visible near the result.

The review page in Phase B is a completable workflow. A dashboard summary may
link to it but never implies completion merely because it was viewed.

Both English and Simplified Chinese ship together.

## Suggested implementation seams

Use a UI-independent planning core rather than placing calculation logic in
HTTP handlers or Svelte components:

```text
internal/planning/
  profile.go          parse and validate versioned custom directives
  effective.go        resolve append-only revisions and statuses
  timeline.go         deterministic dated cash-flow evaluation
  scenarios.go        ephemeral affordability evaluations
  patterns.go         Phase B explainable recurring candidates
  review.go           Phase B matching, variance, completion, staleness

internal/web/favaadapter/
  planning.go         read DTOs and reviewed-write preview envelopes

web/src/fava/reports/
  PlanningReport.svelte
  FinancialCycleReview.svelte   Phase B
```

The core consumes ledger balances and typed planning configuration and returns
explanation-rich results. It has no web dependency, does not mutate source
files, and does not obtain market or institution data. The existing web write
layer owns preview tokens, snapshot guards, backup, atomic publication, and
rollback.

## Implementation phases and acceptance

### A0 — Contract and deterministic core

- Freeze `.v1` directive names, typed fields, revision resolution, and
  planning-only diagnostic behavior.
- Parse readiness inputs and selected account balances.
- Implement the daily timeline, same-day ordering, debt single-counting,
  minimum reserve, low point, primary amount, and funding shortfall.
- Produce an explanation tree for every result.

Acceptance: table-driven tests cover current funds, full credit-card debt,
committed and adjustable plans, no future-income uplift, same-day ordering,
debt-funded purchase, account transfer need, negative headroom, stale
recorded-through date, and excluded currencies.

### A1 — Reviewed planning configuration

- Preview and initialize `planning.bean` and its include.
- List, add, revise, cancel, and reschedule plans through append-only
  directives.
- Manage profile, account lists, debt lists, reserve, timezone, currency,
  horizon, recurring rules, and recorded-through date.
- Reuse snapshot, atomic write, backup, revalidation, and rollback controls.

Acceptance: no configuration write changes accounting balances; failed or
stale writes publish nothing; unsupported planning schema affects planning
only; prior plan revisions remain available.

### A2 — Planning page and scenarios

- Add the OrangeCount Planning extension route.
- Show readiness instead of a number until setup is complete.
- Render the primary result, source explanation, dated low point, excluded
  resources, transfer needs, and shortfall state.
- Add ephemeral affordability scenario comparison and explicit conversion to
  a plan.
- Localize all released interface and diagnostics in English and Simplified
  Chinese.

Acceptance: a user can initialize planning from an existing ledger, obtain an
explainable primary result, test a large purchase without modifying the
ledger, and explicitly add it as a plan. Incomplete, stale, or not-recorded-
through-today state never appears as safe.

### B0 — Candidate and matching engine

- Generate explainable recurring candidates from deterministic evidence.
- Suggest plan-to-actual matches without auto-confirmation.
- Keep overdue plans reserving funds until explicit resolution.
- Apply owner-confirmed materiality thresholds.

Acceptance: every suggestion exposes its evidence; no candidate, recurrence,
or match affects planning before confirmation.

### B1 — Guided financial-cycle review

- Implement the six-step review flow.
- Persist compact review confirmations and supporting snapshot fingerprints.
- Recompute derived review results and mark confirmations stale after relevant
  source or plan changes.
- Feed confirmed next-cycle candidates into the next planning horizon.

Acceptance: a user can complete one anchored cycle, understand material
variances, confirm next-cycle plans, and receive a refreshed primary result;
later historical edits retain but invalidate the prior confirmation.

## Explicit non-goals

The first planning release does not include:

- bank or payment-platform connections, statement acquisition, or import
  adapters;
- category budgets or monthly spending allowances;
- portfolio advice, return forecasting, or assumed asset liquidation;
- probabilistic forecasts, external AI, or opaque scoring;
- automatic plan creation or automatic plan-to-actual matching;
- automatic foreign-exchange conversion;
- shared household ledgers or multi-person approval; or
- notifications, collection reminders, automatic transfers, or payments.

The earlier institution export research remains available in
[china-statement-export-research.md](china-statement-export-research.md), but
it is evidence for a separate possible direction and is not a dependency of
this plan.

## Release-level completion standard

Phase A is complete only when a user can initialize planning against an
existing source ledger, confirm every readiness input, maintain plans, obtain
an explanation-rich primary safe-to-spend amount and timeline low point, and
evaluate a large purchase without persisting it unless explicitly requested.

Additionally:

- every persistent change previews, writes atomically, creates a backup, and
  revalidates before publication;
- accounting balances are unchanged by planning configuration;
- every displayed result traces to accounts, debts, plans, reserve, horizon,
  currency, timezone, and ledger-recorded-through date;
- stale or incomplete inputs cannot produce a result labelled safe;
- English and Simplified Chinese are both complete; and
- runtime behavior remains local, offline, deterministic, and independent of
  a private database.
