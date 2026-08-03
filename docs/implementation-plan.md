# OrangeCount v0.1 implementation plan

## Product contract

OrangeCount is an Apache-2.0, clean-room Go implementation of the Beancount v3 core language and accounting semantics. It is a single offline binary with an embedded, read-only, localhost-only web interface. Its source of truth is a user-maintained `.bean` include graph; it neither executes Python plugins nor edits source ledgers.

The first release ships Simplified Chinese (`zh-CN`) and English (`en`), uses BeanQuery semantics for its query workbench, and includes no bank/broker import, budget model, persistent database, public HTTP API, telemetry, automatic update checks, or Fava-plugin compatibility.

## Architecture and ownership boundaries

```
cmd/orangecount/          CLI commands: check, serve, query
internal/source/          source spans, file/include graph, safe document roots
internal/ledger/          AST, amounts, inventory, directives, evaluator, validators
internal/diagnostic/      accumulated, localized diagnostics and safe rendering
internal/snapshot/        atomic immutable snapshot build/publish and file watching
internal/query/           BeanQuery parser, planner, evaluator, CSV export
internal/web/             localhost server, embedded assets, internal API, safe attachments
web/                      TypeScript UI, translations, report and journal workflows
internal/reference/       optional uv-driven Beancount-v3 differential-test harness
testdata/                 only sanitized minimal fixtures; never the private reference ledger
```

The runtime dependency graph must point inward: web/query consume an immutable snapshot; only the snapshot builder reads source files; source files never depend on a database or the web layer.

## Phase 0 — repository and compliance foundation

1. Create the Go module, Apache-2.0 `LICENSE`, `NOTICE`, source-header policy, `.gitignore`, reproducible build targets, and dependency-license scan.
2. Add a Go version policy and CI jobs for formatting, unit tests, race tests, frontend build, and license validation.
3. Add a `uv`-based, development-only Beancount v3 environment. It must never be a release dependency.
4. Add a private local validation configuration (ignored by version control) for the owner-provided reference ledger. Logs and test reports may only contain redacted summaries.

Exit: a clean checkout builds one binary without Python/Node at runtime; the reference harness is opt-in and refuses to copy a ledger into the repository.

## Phase 1 — source model, parser, and diagnostics

1. Implement UTF-8 source files, precise byte/line/column spans, comments, metadata, includes, include-cycle detection, and source provenance.
2. Parse every Beancount v3 core directive, transactions, postings, tags/links, numbers, currencies, costs/lots, prices, flags, strings, dates, options, and custom values.
3. Recover at directive boundaries so a malformed file yields all possible diagnostics in one pass.
4. Implement stable diagnostic codes, related spans, localized messages, migration warnings for `plugin`, and CLI output in human and JSON forms.

Exit: sanitized grammar fixtures parse into a lossless-enough AST with accurate spans; malformed fixtures produce multiple deterministic diagnostics.

## Phase 2 — v3 evaluation and validation

1. Model exact decimal arithmetic, amounts, positions, inventories, booking, costs/lots, price maps, account state, and metadata.
2. Evaluate directives in source order with deterministic behavior and implement v3-compatible validation: open/close lifecycle, balancing, tolerances, balance assertions, pads, currencies, documents, and supported options.
3. Preserve v3 accounting results by default. Any intentional semantic correction must be explicit, documented, and hidden behind a compatibility/experimental setting until accepted.
4. Publish an immutable snapshot only after a fully valid build; retain the previous snapshot on a failed reload.

Exit: differential tests match Beancount v3 on sanitized entries, errors, balances, inventories, and prices; snapshot replacement is atomic.

## Phase 3 — local CLI, file watching, logging, and security

1. Implement `check`, `serve`, and `query` commands; all should accept an entry ledger path and explicit display-locale settings.
2. Add debounced include-graph watching and rebuilds with redacted status logs.
3. Bind `serve` to loopback only, choose/report a safe port, and never make network requests.
4. Implement structured local JSON logging with default redaction and a clearly marked, explicitly enabled sensitive diagnostics mode.
5. Restrict attachment serving to normalized, configured document roots; prohibit traversal and arbitrary filesystem browsing.

Exit: saving a valid file swaps snapshots without a server restart; saving an invalid file leaves the previous UI state usable and exposes actionable diagnostics.

## Phase 4 — BeanQuery and core reports

1. Implement current BeanQuery grammar, typed expressions, aggregate functions, grouping, ordering, date handling, parameterization, and CSV export.
2. Build deterministic account/journal, trial balance, balance sheet, income statement, holdings, cost basis, price, event, document, and error views from snapshots.
3. Establish explicit valuation, ordering, date, timezone, and empty-account rules shared by CLI and web reports.

Exit: query and report fixtures are deterministic, and their normalized results are differentially validated where the v3/BeanQuery reference exposes an equivalent result.

## Phase 5 — embedded multilingual web interface

1. Build a TypeScript UI whose compiled static assets are embedded into the Go binary; no runtime CDN, Node, or external service.
2. Implement complete `zh-CN` and `en` translation catalogs from the first UI slice; locale selection must never alter ledger semantics.
3. Deliver read-only overview, account tree, journal, search/filter by tags/links/metadata, core reports, holdings/prices, BeanQuery workbench, document links, diagnostics, and source navigation.
4. Keep the local HTTP API internal and versioned; enforce safe attachment behavior in the server rather than the client.

Exit: both shipped locales complete the agreed read-only workflows in an offline single-binary run.

## Phase 6 — compatibility, performance, and release hardening

1. Expand sanitized corpus with grammar edge cases, real-world bug regressions, include errors, lot/cost scenarios, Unicode, localization, report ordering, and reload failures.
2. Run differential testing against official Beancount v3 via `uv`; track approved differences in a machine-readable compatibility ledger.
3. Locally validate the private reference ledger without persisting its contents: it must fully parse, resolve its include graph, evaluate, and produce a valid OrangeCount snapshot; all differences must be explained.
4. Benchmark parse, evaluation, query, and reload paths; optimize only after recording baseline measurements.
5. Verify offline execution, redaction, dependency licenses, vulnerability checks, binary reproducibility, and cross-platform release artifacts.

## v0.1 release gate

Release only when all of the following hold:

1. Sanitized fixtures cover v3 core syntax and semantics, and differential results either match v3 or have an approved compatibility/experimental rationale.
2. The private reference ledger parses from its entry file, resolves its include graph, and evaluates to a valid OrangeCount snapshot; no balance, inventory, query, or report difference remains unexplained.
3. The local read-only web workflows work fully in `zh-CN` and `en`.
4. The released binary is self-contained at runtime, offline by default, localhost-only, atomically reloads snapshots, provides accumulated precise diagnostics, and writes redacted structured logs.
5. No private ledger data is in source control, CI output, fixtures, generated docs, logs, or release artifacts; dependency and license scans pass.

## Delivery sequence

Implement Phases 0–3 as the first vertical slice, then add reports/query before the full web UI. Do not treat the parser alone as a release milestone: syntax, semantics, diagnostics, reload safety, and the reference-ledger check advance together.
