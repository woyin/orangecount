# Quick Entry: Design and Implementation Plan

This plan captures the consensus from the `$grill-with-docs` session on a
fast capture path for the document-based bookkeeping workflow. It records the
decisions, the grammar, the source-ledger contract, and the phased build. All
terminology aligns with the terms now registered in [CONTEXT.md](../CONTEXT.md)
under "Quick-entry …"; the public profile schema is fixed by
[ADR-0043](adr/0043-version-the-ledger-embedded-quick-entry-profile.md).

## Problem

Beancount's document model is rigorous but verbose for the most common daily
capture ("lunch 28 CNY from WeChat"). Today the only reviewed write paths are
the Editor and the Add Entry modal. Both require the ledger owner to fill every
field by hand, even for a recurring two-posting transaction.

The goal is a Quick Entry surface that makes high-frequency capture fast
without weakening the reviewed-write safety model, the single canonical ledger,
or compatibility with Beancount v3.

## Non-goals (first version)

- No second persisted ledger representation. Quick-entry text is ephemeral.
- No negative amounts, cost lots, prices, balance assertions, or non-transaction
  directives in the shorthand. Those remain in Add Entry / Editor.
- No natural-language bookkeeping. The grammar is strict and deterministic.
- No mutating CLI writer in this version. The compiler stays UI-independent so
  a future CLI can reuse the same reviewed-write module.
- No cross-file batches, automatic chronological insertion, scheduled
  transactions, or general history/undo beyond the last batch.

## Decisions (from grilling)

| Decision | Choice | Rationale |
|---|---|---|
| Persistence | Shorthand is transient; canonical Beancount is the only record | Avoids two-source-of-truth, sync, and audit gaps |
| Determinism | Same context + text compiles to the same preview; no probabilistic account choice | Suggestions may be adaptive; publication must be predictable |
| Profile storage | Standard Beancount `custom` directives in the source ledger | Portable, dated, auditable, parseable by compatible implementations |
| Safety | Two-stage confirmation; preview then publish through reviewed write | Keeps the existing atomic/recoverable/revalidated write model |
| Scope | Two-posting expense/income/transfer only | Covers the high-frequency case without recreating Beancount |
| Grammar | Exact template invocation + explicit delimited form; no free NLU | Speed without interpretive ambiguity |
| Defaults | Omission allowed only when the value is unique and visible | "Fast" must not mean "hidden inference" |
| Target file | Visible, overridable, must already be in the include graph | No surprise appends or ledger restructuring |
| Direction | Positive amount + arrow determines flow; compiler emits signs | Users express intent; Beancount signs stay correct |
| Templates | May prefill accounts/currency/payee/narration/tags/links; never amount/date | Reduces repetition without fixing facts of the event |
| Duplicates | Single-use token blocks retry; equivalent-txn is a non-blocking warning | Technical retry is prevented; real business duplicates are allowed |
| Batches | Each line independent; whole batch previews and publishes atomically | Matches existing serializer; one failure aborts the batch |
| Profile dating | Rule effective as of the transaction date; future rules do not backfill | Historical capture uses historically correct mappings |
| Delivery | Web-first; compiler is UI-independent | Reuses the safe local write path; CLI deferred |
| Profile manager | Required in v1, not an enhancement | Hand-writing `custom` directives cannot be the only way to configure |
| Syntax portability | Operators/numbers/commodities fixed; Unicode allowed in content | Same text compiles the same under any display locale |
| Output | Both posting amounts always written explicitly | Auditable without OrangeCount |
| Undo | Restore last batch only, if snapshot still current | Safety net for quick-entry mistakes; not a history system |
| Schema versioning | `.v1` in the public `custom` type; incompatible change = `.v2` | Persisted user ledgers make later changes expensive |
| Fault isolation | Profile problems never invalidate the accounting ledger | Bookkeeping validity and capture config are separate concerns |
| Placement | Append-only; warn on historical date | Respects user's document organization |
| Batch target | One file per batch | Keeps single-file atomic write and single undo |
| Drafts | In-memory only; no localStorage/server/log/ledger recovery | Preserves "no second representation" absolutely |
| Entry surface | Same modal as Add Entry; Quick is a labeled extension tab; `a q` opens it | Fava standard entry surface stays primary; Quick is additive |

## Source-ledger contract (ADR-0043)

Profile configuration is standard Beancount, dated, and versioned:

```beancount
2026-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat

2026-01-01 custom "orangecount.quick-template.v1" "午餐"
  destination: "Expenses:Food"
  currency: "CNY"
  narration: "午餐"
  tags: "报销"
```

- Each directive is one alias or template at one version.
- Fields use typed values (string/account/currency/date/bool) and directive
  metadata. No embedded JSON payload.
- `.v1` meaning is frozen; incompatible evolution gets `.v2`.
- Unsupported versions are diagnosed and excluded from compilation; they never
  invalidate the accounting snapshot.
- Same-day competing definitions of the same alias are ambiguous and block
  that rule only.
- Retirement is a dated directive (`retired: TRUE` metadata, or a superseding
  definition); historical rules remain effective for their period.

## Grammar

Two strict forms. No free-form natural language is interpreted.

### Template invocation (compact)

```text
午餐 28 @微信
工资 10000
```

The first token must exactly match a defined template name; the remaining
tokens supply amount, currency override, and an optional counterparty alias.
Expansion is fully visible in preview.

### Explicit form

```text
28 CNY @微信 -> @餐饮 : 工作午餐 #报销
```

Tokens:

- `AMOUNT [CURRENCY]` — positive decimal; currency optional if unique/templated.
- `@name` — account alias (must resolve to one account for the txn date).
- `->` — value-flow arrow. Left side loses the amount; right side gains it.
- `: narration` — description.
- `#tag`, `^link` — repeatable; same validity as existing Add Entry.

### Generated output

```beancount
2026-08-12 * "工作午餐"
  Assets:WeChat   -28 CNY
  Expenses:Food    28 CNY
```

- Both postings always carry explicit amount and currency.
- Signs follow value-flow: source decrease, destination increase. Income
  accounts still emit the conventional negative balance.
- No OrangeCount-private metadata is written.

## Defaults (visible and overridable)

- **Date**: the date picker next to the capture box, initialized to today.
  Editable per batch; applies to every line.
- **Currency**: template-configured value, else the ledger's single
  `operating_currency`. Ambiguous → must be supplied per line.
- **Flag**: `*`; shown in preview; overridable to `!`.
- Any non-unique resolution is a compile error for that line, never a guess.

## Capture and publication flow

1. User opens the modal via `+` → Quick tab, or `a q`.
2. Types one or more lines; a date and target-file selector stay visible.
3. First Enter (or Preview button):
   - Each line is compiled independently with date-effective profile rules.
   - Errors are reported per line (unresolved alias, ambiguous mapping,
     invalid amount, non-`E-` blocking issue).
   - Duplicate warnings are attached per line but do not block.
   - A single-use preview token is issued; the batch is not written.
4. Second Enter (or Commit):
   - Server validates the preview token (rejects replay/retry).
   - Validates the target file still belongs to the current include graph and
     the snapshot id is still current.
   - Appends all generated blocks to the end of the target file in one write.
   - Re-evaluates; on failure rolls back from backup and returns diagnostics.
   - On success: publishes new snapshot, clears the draft, records the batch as
     the session's undoable last action.

## Undo

- Available only for the session's most recent published Quick batch.
- Available only while the snapshot after that batch is still current.
- Previews the exact lines to remove, then runs the same atomic/backup/
  revalidate path.
- Later ledger edits disable undo and surface a diagnostic directing the user
  to manual correction.

## Architecture

```
internal/quickentry/        NEW — UI-independent compiler core (Go)
  compiler.go               parse + compile text → []NewEntry | errors
  profile.go                read dated custom directives → effective rules
  profile_test.go
  compiler_test.go
  grammar_tests/            golden text → expected beancount

internal/web/favaadapter/   EXTEND
  quickentry.go             preview/commit/undo envelopes; reuses SerializeNewEntries
  registry.go               routes: quick-preview, quick-commit, quick-undo,
                            quick-profile (list/upsert/retire)

internal/web/server.go      EXTEND
  handlers for the new routes; reuse replaceGraphFile + snapshot guards
  single-use preview token store (like importPreviewStore)

web/src/fava/
  modals/AddEntryModal.svelte   EXTEND — add Quick tab; keep Transaction/Balance/Note
  modals/QuickEntryPanel.svelte NEW — textarea, per-line preview, duplicate badges,
                                target/date controls, undo button
  reports/                      EXTEND — profile manager page (list/upsert/retire)
  keyboard-shortcuts.ts         EXTEND — `a q` opens Quick

CONTEXT.md                     UPDATED — Quick-entry terms (already done)
docs/adr/0043-…                NEW — public schema (already done)
```

The compiler depends only on `internal/ledger` types and the profile reader;
it has no web dependency and is called from tests directly. The web layer
treats its output the same as Add Entry's `NewEntry` and reuses
`SerializeNewEntries`, `replaceGraphFile`, backup, revalidation, and snapshot
publication.

## Phased plan

### Phase 0 — Compiler core and profile reader

- `internal/quickentry/profile.go`: parse `orangecount.quick-account.v1` and
  `orangecount.quick-template.v1` directives from an `Evaluation`; choose the
  effective rule for a given transaction date; detect ambiguity; ignore
  unsupported versions with structured diagnostics.
- `internal/quickentry/compiler.go`: lex and parse both grammar forms; resolve
  aliases/templates; emit `[]favaadapter.NewEntry` plus per-line errors.
- Golden tests for: compact template, explicit form, both-ends alias,
  single-end template, income sign convention, batch with mixed lines,
  duplicate detection, ambiguity, historical date profile selection.

Exit: the compiler is fully covered by tests and has no UI coupling.

### Phase 1 — Preview/commit server routes

- `POST /api/v1/quick/preview`: accepts text + date + target file + snapshot id;
  returns per-line compiled entries, generated Beancount, duplicate warnings,
  diagnostics, and a single-use token.
- `POST /api/v1/quick/commit`: validates token + snapshot id + target file;
  appends serialized entries via `replaceGraphFile`; returns published snapshot
  id + backup path or diagnostics.
- Single-use token store modeled on `importPreviewStore`; bounded and TTL'd.
- Reuse existing `requireSameOrigin`, `decodeJSONBody`, and diagnostics payload.

Exit: round-trip server tests pass for success, replay rejection, stale
snapshot rejection, target-file validation, and invalid-line rejection.

### Phase 2 — Undo route

- `POST /api/v1/quick/undo`: available only for the session's last committed
  batch; checks snapshot id still current; previews removal; on confirm
  rewrites the file minus that batch's appended lines and revalidates.

Exit: tests cover undo success, undo after later edit (disabled), and undo
after replay.

### Phase 3 — Web UI

- Add Quick tab to `AddEntryModal`; keep Transaction/Balance/Note unchanged.
- `QuickEntryPanel`: textarea, visible date + target-file selectors, per-line
  preview rows, duplicate badges, error rows, two-button (Preview / Commit)
  flow, post-commit "Undo last batch" affordance.
- Bind `a q` to open Quick directly; leave `+` on its current Add Entry default.
- Profile manager page: list effective rules as of today; create/supersede/
  retire via a small reviewed-write form that emits the exact `custom`
  directives; surface profile diagnostics.

Exit: black-box interaction tests; `a q` opens Quick; a two-line batch previews,
commits, and appears in the journal; undo removes it; profile manager creates a
template usable from Quick.

### Phase 4 — Localization, help, and docs

- UI strings in `web/src/translations.ts` for `en` and `zh-CN`.
- Help topics for Quick grammar, profile directives, and duplicate warnings
  surfaced under `/help`.
- README section and a short user-facing Quick Entry guide.

Exit: both locales render; help search finds the topic; no English fallback in
a released locale.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Users treat Quick as the only entry path and lose fluency with full Beancount | Quick stays a labeled extension tab; Add Entry remains the `+` default |
| Profile drift between machines | Profile lives in the ledger, so it travels with the include graph |
| Replay/retry double-writes a batch | Single-use preview token validated on commit |
| Historical capture uses today's mapping | Effective-rule lookup is dated; tested explicitly |
| Profile misconfiguration blocks the ledger | Profile diagnostics never touch accounting validity |
| Public schema needs to evolve | `.v1` frozen; `.v2` is additive with its own diagnostics |

## Open items (deferred, not blocking v1)

- Cost-lot / price shorthand, if a clear grammar emerges.
- CLI write command sharing the reviewed-write module.
- Automatic chronological insertion as an explicit, opt-in placement strategy.
- Cross-file batches once the write layer supports multi-file transactions.
