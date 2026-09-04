# OrangeCount

OrangeCount is a personal, Go-native accounting system that can read and validate existing Beancount ledgers without a Python Beancount runtime.

## Language

### Ledger and compatibility

**Compatible ledger**:
A Beancount ledger whose accepted syntax and accounting meaning OrangeCount preserves, subject to explicitly documented compatibility boundaries.
_Avoid_: import format, legacy file

**Pivot report**:
An Excel-style cross-tab over the ledger — rows, columns, and a value (period totals or ending balance) chosen from pickers — served as a report surface on top of grouped query results. Not a query-language feature.
_Avoid_: PIVOT syntax, full SQL

**Ending balance**:
An account's or group's balance as of the end of a reporting interval, computed from interval balances rather than summed from postings, because inventories do not sum across currencies.
_Avoid_: final balance, sum of balance

**Independent implementation**:
The Go implementation that parses and evaluates a compatible ledger without delegating execution to Python Beancount.
_Avoid_: wrapper, binding, port

**Core ledger compatibility**:
Support for Beancount's standard directives, transactions, amounts, costs, lots, tolerances, includes, options, and query syntax with equivalent accounting meaning. Python plugin code is not executed; plugin declarations are instead preserved and diagnosed as migration work.
_Avoid_: partial parser, plugin compatibility

**Compatibility contract**:
The Beancount v3 language and accounting semantics that OrangeCount uses as its behavioral reference. Valid v2 constructs remain supported only where they are also valid v3 core syntax.
_Avoid_: v2 parity, best-effort compatibility

**Semantic change**:
A behavior difference that alters a ledger's accounting result, including balances, inventories, booking, or query values, rather than merely improving presentation or diagnostics.
_Avoid_: bug fix, formatting change

### Workbench and Fava parity

**Built-in web interface**:
The Go-served personal ledger interface that provides the exploration, reporting, and reviewed authoring workflows associated with Fava.
_Avoid_: external dashboard, Fava integration

**Workflow compatibility**:
The ability to complete the same personal ledger analysis workflows as Fava without reproducing its implementation, HTTP API, plugin ecosystem, or visual design.
_Avoid_: API compatibility, pixel parity

**Fava parity**:
The ability to complete Fava's local personal-ledger interface workflows with materially equivalent page structure, controls, interaction states, and outcomes. OrangeCount may use an independent implementation while preserving the user-visible workflow.
_Avoid_: superficial theme match, API compatibility

**Observed UX parity**:
High-fidelity reproduction of Fava's user-observable information architecture, visual hierarchy, interaction behavior, responsive states, keyboard behavior, and task outcomes, verified from black-box use of a running Fava instance and, where useful, selective adaptation of Fava's MIT-licensed frontend. It excludes Fava's Python/Beancount runtime and public HTTP API.
_Avoid_: unlicensed third-party reuse, backend port, theme-only parity

**Fava standard surface**:
The complete set of user-visible, built-in local-interface capabilities supplied by the pinned Fava 1.30.12 release, including its built-in pages, editor, importer, and options. Third-party plugin pages, user-defined extensions, and Fava's HTTP API are excluded.
_Avoid_: plugin ecosystem parity, unbounded feature scope, API compatibility

**Fava option compatibility**:
Support for every user-visible Fava 1.30.12 built-in option that does not depend on an explicitly excluded capability, with the same interface effect and precedence. Excluded option behavior remains visible as an approved Fava deviation rather than being silently ignored.
_Avoid_: supported subset, display-only option listing, silent fallback

**UX parity gate**:
The release criterion for an in-scope Fava capability: equivalent task outcomes, interaction and keyboard states, responsive behavior, and Fava rendering fidelity verified against a redacted deterministic fixture.
_Avoid_: workflow-only acceptance, manual-only visual review, approximate parity

**Fava rendering fidelity**:
The requirement that a fixed Fava version and OrangeCount produce the same observable visual composition under controlled browser, viewport, theme, locale, and fixture conditions, with every remaining difference explicitly documented and approved.
_Avoid_: approximate theme match, undocumented visual deviation, arbitrary screenshot similarity score

**Parity authority**:
The source that decides whether a parity concern is correct: Fava decides user-observable UX; Beancount v3 decides ledger, valuation, and BeanQuery semantics. When the two differ, OrangeCount retains the v3 result and presents it through the Fava-aligned UX.
_Avoid_: Fava-defined accounting semantics, UI-driven semantic drift

**Parity evidence discipline**:
The rule that permits private-ledger Fava observation only in transient local browser sessions. Repository evidence contains only abstracted behavior records and redacted deterministic-fixture artifacts; it never contains private screenshots, DOM, responses, source locations, account names, amounts, or task payloads derived from them.
_Avoid_: redacted-after-commit evidence, private golden image, raw browser dump

**Fava-derived frontend**:
A deliberately selected and adapted portion of Fava 1.30.12's MIT-licensed frontend code, styles, or assets, used to preserve UX fidelity while replacing its Python API and runtime assumptions with OrangeCount adapters. Every derived file retains required copyright and MIT-license attribution and is recorded in the third-party notice inventory.
_Avoid_: untracked copy, Python runtime dependency, whole-application fork

**Fava reference mirror**:
A read-only, repository-external checkout pinned to the Fava 1.30.12 source commit. It is available for complete source study and provenance mapping but is neither shipped nor versioned as part of OrangeCount; the repository records its revision and each adopted source unit.
_Avoid_: floating upstream checkout, vendored whole application, undocumented source reference

**Fava visual baseline**:
An approved rendering produced by the pinned Fava release under controlled conditions using only the deterministic synthetic ledger. It is the visual authority for regression comparison and never contains private-ledger material.
_Avoid_: private screenshot, developer recollection, self-referential OrangeCount snapshot

**Approved Fava deviation**:
An explicit, reviewed departure from the Fava visual baseline or behavior, made either to satisfy OrangeCount's semantic authority, security, data integrity, privacy, or accessibility obligations, or to serve a switchable ledger-owner-approved presentation preference. Each deviation is logged with its basis (obligation or owner preference).
_Avoid_: silent drift, unreviewed redesign, a presentation change that cannot be switched back to parity

**Modern chart layer**:
The standard time-series chart presentation for the income-statement, balance-sheet, and account routes (bar and line charts). It is d3-backed (scales, stack offsets, tick computation, HCL color interpolation) and consumes the same adapter data contract as the ledger's other views; it never changes valuation, currency aggregation, or any accounting semantics. Hierarchy charts (treemap/sunburst/icicle) are out of scope and still use the parity renderer. See ADR-0040/0041.
_Avoid_: switchable skin, parity fallback for time-series, dashboard mode

**Fava frontend transplant**:
The primary OrangeCount web-client migration that starts from Fava 1.30.12's frontend composition, components, styles, and interactions, then replaces Fava data access with Go-backed adapters. The legacy OrangeCount interface is a temporary per-page fallback and is deleted as each page family is accepted.
_Avoid_: incremental restyling of the legacy shell, Python backend migration, permanent dual UI

**Fava-shaped adapter**:
A loopback-only, internal Go HTTP contract that supplies exactly the request, response, and error shapes required by the transplanted Fava frontend. It is derived per implemented page, backed by OrangeCount's v3 semantic core, and has no public API compatibility or third-party-client support commitment.
_Avoid_: public Fava API, Python endpoint emulation, domain model leakage

**Fava source inventory**:
The mandatory, complete Fava 1.30.12 study performed before any migration: frontend files, routes, state, styles, server endpoints, tests, build tooling, and license dependencies are mapped to OrangeCount adoption, adaptation, exclusion, or replacement decisions.
_Avoid_: page-only reconnaissance, undocumented selective reading, implementation before mapping

**UI migration flag**:
A temporary, explicit local selection of the transplanted Fava frontend during migration. It permits complete per-route cutovers and fallback to the legacy UI while preserving one Go write and snapshot path; it is removed with the legacy UI only after every standard route passes the UX parity gate.
_Avoid_: mixed components on one route, parallel write paths, permanent compatibility switch

**English visual baseline**:
The English Fava standard surface used to define visual-regression expectations. OrangeCount's English interface must meet the full UX parity gate; its Simplified Chinese interface keeps the same information architecture, controls, and behavior while allowing language- and font-driven layout differences.
_Avoid_: English-only product, translated screenshot pixel match, locale-specific workflow

**Standard navigation surface**:
The Fava 1.30.12 main navigation, default landing behavior, URL routes, and page hierarchy. OrangeCount-exclusive tools may exist only as clearly labeled extensions that do not alter or compete with this standard surface.
_Avoid_: mixed primary navigation, OrangeCount default route, hidden route incompatibility

**Local web session**:
A built-in web-interface session initiated by the OrangeCount CLI and bound only to the loopback network interface for its owner.
_Avoid_: hosted account, shared instance

### Authoring and Quick Entry

**Source ledger**:
The user-maintained `.bean` file set, including its include graph, that remains OrangeCount's authoritative accounting record.
_Avoid_: application database, managed ledger

**Complete personal ledger**:
A personal accounting record intended to represent the owner's real assets, liabilities, income, and expenses completely enough to be checked against independent financial evidence.
_Avoid_: spending diary, remembered-transactions log, expense tracker

**Safe-to-spend amount**:
The portion of the owner's liquid funds that can be spent within a stated future period without compromising known obligations or an owner-defined financial reserve.
_Avoid_: account balance, disposable income, remaining budget

**Safe-to-spend horizon**:
The future period covered by a safe-to-spend decision, ending by default immediately before the owner's next expected stable income and otherwise on an owner-selected date.
_Avoid_: calendar month, report period, fixed 30-day window

**Planned cash flow**:
An owner-confirmed, named future inflow or outflow with an expected date and amount in the planning currency that participates in a safe-to-spend calculation; an outflow also declares whether it is committed or adjustable. Expected accounts and notes may enrich transfer and matching suggestions, but the plan remains an assumption rather than an occurred accounting event.
_Avoid_: scheduled transaction, forecasted posting, future ledger entry

**Conservative planning amount**:
The single owner-confirmed amount used for a planned cash flow whose actual value is uncertain, chosen to avoid overstating safe-to-spend capacity. Ranges and probabilities may be noted but do not enter the first-version calculation.
_Avoid_: expected-value estimate, probabilistic amount, hidden upper bound

**Conservative planning date**:
The single owner-confirmed date used when timing is uncertain: the earliest credible date for an outflow and the latest credible date for an inflow. Date ranges may be noted but do not enter the first-version timeline.
_Avoid_: average date, optimistic payday, hidden date range

**Cash-flow candidate**:
A non-binding suggestion derived from visible ledger-history evidence that may become a planned cash flow only after the owner confirms or edits its expected date, amount, and commitment. It never affects safe-to-spend results by itself.
_Avoid_: predicted transaction, inferred obligation, automatic plan

**Spendable-funds account**:
An explicitly owner-designated account whose liquid balance may fund ordinary spending and may be transferred within the planning pool during the safe-to-spend horizon. Valuable but intentionally unavailable assets and newly discovered accounts are excluded unless the owner explicitly changes their designation.
_Avoid_: every asset account, net worth account, automatically liquid asset

**Current short-term debt**:
The full ledger balance of an owner-designated short-term liability, including amounts not yet billed or due, that reduces safe-to-spend resources once. A later payment of that debt is a transfer and does not reduce the result again.
_Avoid_: statement balance, minimum payment, planned expense

**Minimum reserve**:
An owner-confirmed amount of liquid funds that a safe-to-spend decision must preserve throughout its horizon. Historical spending may inform a suggested value, but cannot change the effective reserve without confirmation.
_Avoid_: emergency-fund account, predicted reserve, unused balance

**Committed outflow**:
An owner-confirmed planned cash outflow that must be preserved within the safe-to-spend horizon and is always deducted from safe-to-spend resources.
_Avoid_: fixed expense, unavoidable transaction, recurring category

**Adjustable outflow**:
An owner-confirmed planned cash outflow that occupies funds under the current plan but may be reduced or abandoned to make room for another decision.
_Avoid_: discretionary category, optional posting, ignored expense

**Ledger-embedded planning profile**:
The versioned OrangeCount planning configuration recorded as standard Beancount `custom` directives, optionally in a dedicated included file. It stores confirmed planning assumptions and reserves without representing occurred transactions or changing accounting balances.
_Avoid_: future transaction, application preference, private sidecar database

**Planning-profile initialization**:
A reviewed write that previews and then atomically creates or selects the planning target, adds any required source-ledger include, writes the initial versioned directives, and revalidates the ledger. It never silently restructures the include graph.
_Avoid_: implicit setup, hidden file creation, unmanaged sidecar

**Plan revision**:
An append-only, owner-confirmed change to a stable plan identity that supersedes its prior effective state without deleting it. Changes, cancellation, fulfillment, and rescheduling remain historically explainable while only the latest valid revision is effective.
_Avoid_: in-place plan edit, deleted plan, mutable history

**Recurring cash-flow rule**:
An owner-confirmed pattern that proposes future cash-flow candidates on an expected cadence. A generated occurrence affects no planning result until the owner confirms or edits it as a planned cash flow.
_Avoid_: scheduled transaction, automatic obligation, recurring posting

**Current-funded safe-to-spend amount**:
A safe-to-spend result funded only by liquid resources already held; confirmed but unrealized future inflows cannot increase it.
_Avoid_: projected balance, expected income balance, available credit

**Expected safe-to-spend amount**:
A scenario result that may include owner-confirmed future inflows within the horizon and visibly identifies each unrealized inflow assumption.
_Avoid_: guaranteed amount, current balance, hidden income forecast

**Primary safe-to-spend amount**:
The default safe-to-spend result that uses only currently held funds and preserves both committed and adjustable planned outflows. Less conservative alternatives are explicit scenarios, never co-equal headline values.
_Avoid_: maximum affordable amount, expected balance, scenario average

**Planning currency**:
The single commodity in which a safe-to-spend result is calculated. Other-currency resources and obligations remain visible but are excluded rather than silently converted or assumed available for exchange.
_Avoid_: display currency, automatic base-currency conversion, net-worth currency

**Planning timezone**:
The owner-confirmed IANA timezone that determines planning dates, overdue status, financial-cycle boundaries, and whether a ledger-recorded-through date is current. The machine timezone may be suggested during setup but never silently becomes the authority.
_Avoid_: browser timezone, implicit local time, UTC planning day

**Planning liquidity low point**:
The lowest projected balance of spendable funds reached at any dated step within a safe-to-spend horizon. Safety is evaluated against the minimum reserve at this point, not merely against the ending balance.
_Avoid_: ending balance, average balance, current balance

**Planning funding shortfall**:
The amount by which a planning liquidity low point falls below the minimum reserve. The interface presents zero safe-to-spend capacity plus the shortfall amount, first occurrence date, and contributing assumptions rather than describing a negative amount as spendable.
_Avoid_: negative spending allowance, overdraft limit, hidden deficit

**Same-day planning order**:
The conservative daily timeline rule that applies planned outflows before planned inflows sharing a date. First-version planning uses dates rather than assumed intraday timing.
_Avoid_: netted daily change, income-first ordering, inferred transaction time

**Debt-funded planned outflow**:
A planned purchase expected to create short-term debt that reserves safe-to-spend resources once on its economic date. Its later repayment affects account-level liquidity and transfer needs but does not reduce aggregate capacity again.
_Avoid_: repayment-date expense, available credit, double-counted payment

**Planned transfer need**:
An account-level funding shortfall at a dated planning step when the combined spendable-funds pool remains sufficient. It prompts an explicit transfer action without reducing aggregate safe-to-spend capacity again.
_Avoid_: overall shortfall, planned expense, automatic transfer

**Affordability scenario**:
An explanatory what-if evaluation of a proposed amount and date against the planning timeline. It exposes reserve headroom, the low point, displaced adjustable outflows, unrealized-income dependencies, and excluded resources without deciding whether the owner should spend.
_Avoid_: purchase approval, financial advice, affordability score

**Scenario draft**:
An ephemeral affordability scenario that never changes a planning result or source-ledger configuration. It becomes a planned cash flow only through an explicit owner-confirmed conversion.
_Avoid_: draft plan, automatically saved purchase, pending transaction

**Financial-cycle review**:
A review of the completed period between consecutive stable-income occurrences that compares confirmed plans with actual ledger activity and prepares candidates for the next safe-to-spend horizon. Calendar-month analysis remains a report view rather than the default planning review.
_Avoid_: monthly report, category dashboard, transaction audit

**Plan-fulfillment match**:
An owner-confirmed link between a planned cash flow and an occurred source-ledger transaction. Similarity in date, amount, account, or description may produce a visible match suggestion but never fulfills the plan automatically.
_Avoid_: automatic reconciliation, inferred completion, transaction replacement

**Unresolved overdue plan**:
A planned outflow whose expected date has passed without a confirmed fulfillment, cancellation, or rescheduling. It continues to reserve funds and remains visible until the owner explicitly resolves it.
_Avoid_: expired plan, ignored payment, automatically cancelled obligation

**Financial-cycle anchor**:
The single owner-designated recurring income rule whose confirmed actual occurrence closes one financial-cycle review and anchors the next safe-to-spend horizon. Incidental inflows never become the anchor implicitly; without one, the owner selects a horizon date.
_Avoid_: any income, largest deposit, automatically detected payday

**Planning variance**:
An explainable difference between a confirmed plan and actual ledger activity within a financial cycle, including changed amount or timing, non-occurrence, duplication, and displacement of planned headroom. It is the primary subject of a financial-cycle review.
_Avoid_: category total, generic anomaly, accounting error

**Completed financial-cycle review**:
An owner-confirmed completion of the guided review workflow after plan matches, unresolved items, material variances, and next-cycle candidates have been addressed. A summary may present it, but a live dashboard alone does not complete it.
_Avoid_: viewed dashboard, generated report, calendar close

**Review confirmation record**:
A compact ledger-embedded record of a completed financial cycle, its supporting ledger-snapshot fingerprint, and the owner-confirmed plan revisions and fulfillment matches. Derived charts, variances, and low points are recomputed rather than persisted.
_Avoid_: saved report snapshot, duplicated analytics, locked accounting period

**Stale financial-cycle review**:
A previously completed review whose supporting historical ledger activity or confirmed planning assumptions have since changed. It remains visible as a prior conclusion but is no longer current until the owner reopens and confirms it.
_Avoid_: locked period, silently recomputed review, deleted review

**Variance materiality threshold**:
An owner-confirmed amount or timing tolerance that determines which ordinary planning variances enter the primary review flow. Missed, duplicated, or overdue committed outflows remain primary regardless of this threshold, and suppressed variances remain inspectable.
_Avoid_: hidden significance score, automatic threshold, ignored difference

**Explainable cash-flow pattern**:
A locally derived, deterministic historical pattern whose supporting occurrences, cadence, normalized description, accounts, and amount range are visible. It may generate a cash-flow candidate but never a confirmed plan.
_Avoid_: AI prediction, opaque recurrence score, automatic categorization

**Planning readiness**:
The condition in which the owner has confirmed the planning currency, spendable-funds accounts, applicable short-term debts, minimum reserve, horizon anchor or date, and current committed outflows. No safe-to-spend result is presented before this condition is met.
_Avoid_: inferred setup, partial estimate, account-balance fallback

**Ledger-recorded-through date**:
The owner-confirmed latest date through which real financial activity is represented in the source ledger for planning purposes. It states the calculation's completeness assumption without claiming external reconciliation.
_Avoid_: reconciliation date, latest transaction date, verified balance date

**Current planning status**:
A planning result whose ledger-recorded-through date is today in the owner's display timezone and whose planning-readiness requirements remain satisfied. Older results remain viewable only as explicitly stale estimates and are not presented as safe.
_Avoid_: recently updated estimate, latest transaction status, reconciled status

**Synthetic reference ledger**:
A deterministic, non-private ledger designed to reproduce the scale, density, currencies, long labels, unavailable valuations, and interaction states needed for Fava parity evidence.
_Avoid_: toy fixture, anonymized private ledger, random sample data

**Private reference ledger**:
A source ledger used only on its owner's machine to verify a release against real-world usage. A release must parse and evaluate it into a valid ledger snapshot without copying its content into the project.
_Avoid_: committed fixture, sample ledger

**Document root**:
An explicitly configured filesystem root from which OrangeCount may resolve attachments named by a `document` directive after path normalization and containment checks.
_Avoid_: arbitrary local path, file browser

**Reviewed write workflow**:
A source-ledger change that is previewed or validated against an expected ledger snapshot and published only through an atomic, recoverable, full-graph revalidation path. Any failure leaves both the source ledger and the published ledger snapshot unchanged; the workflow is independent of its Web or CLI delivery surface.
_Avoid_: autosave, direct file mutation, partial snapshot publication, web-only mutation

**Ledger proposal**:
A complete, validated description of one or more source-ledger directives and their visible target file, prepared for review but not yet published. It has no accounting effect until a reviewed write workflow succeeds.
_Avoid_: HTTP request model, serialized JSON, active ledger change, quick-entry draft

**Quick-entry notation**:
A transient, deterministically compiled shorthand for capturing accounting intent through explicit aliases, templates, and defaults. Suggestions may be adaptive, but compilation never makes a probabilistic accounting choice; the reviewed result is canonical Beancount source and the sole editable and auditable record.
_Avoid_: DSL source ledger, alternate ledger format, synchronized shorthand file

**Quick-entry profile**:
The dated, ledger-owned aliases, templates, and defaults declared with standard Beancount `custom` directives and used to compile quick-entry notation. It is portable and auditable source-ledger configuration that never changes balances by itself.
_Avoid_: hidden application preference, global alias database, sidecar DSL configuration

**Quick-entry confirmation**:
The mandatory two-stage keyboard flow in which the first confirmation compiles, previews, and validates canonical Beancount output and the second publishes it through the reviewed write workflow. Ambiguous mappings, invalid accounting output, or a stale ledger snapshot prevent publication.
_Avoid_: blind append, single-keystroke write, implicit confirmation

**Quick-entry transaction**:
The first-version quick-entry target: an expense, income, or transfer with one explicit amount and currency and exactly two postings, whose accounts may be supplied by the quick-entry profile. Split postings, costs, prices, balance assertions, and non-transaction directives remain full-authoring work.
_Avoid_: abbreviated Beancount language, universal entry DSL, hidden split transaction

**Quick-entry grammar**:
The strict two-form language for a quick-entry transaction: an exact template invocation for the shortest recurring cases and an explicitly delimited form for accounts, direction, narration, tags, and links. Completion may insert valid syntax, but free-form natural language is never interpreted as accounting intent.
_Avoid_: natural-language bookkeeping, heuristic token order, conversational parser

**Visible quick-entry default**:
A uniquely resolved date, currency, or transaction flag that may be omitted from quick-entry notation only while its effective value and source remain visible and overridable before preview. Ledger history never supplies a probabilistic default.
_Avoid_: hidden inference, historical guess, silent fallback

**Quick-entry target**:
The visible, overridable source-ledger file that receives a confirmed quick-entry transaction. It must already belong to the current include graph; the quick-entry profile may choose its default, otherwise the entry file is used, and first-version capture never creates files or edits includes.
_Avoid_: hidden append destination, automatic ledger restructuring, unincluded output file

**Quick-entry direction**:
The value-flow arrow from a source account to a destination account for a strictly positive quick-entry amount. Compilation renders the source decrease and destination increase with Beancount's required posting signs, including negative income balances; reversals use the opposite arrow rather than a negative input amount.
_Avoid_: debit-credit arrow, signed quick-entry amount, double-negative reversal

**Quick-entry template**:
A named quick-entry-profile rule that may prefill either or both posting roles, currency, payee, narration, tags, and links while leaving the transaction's amount and effective date explicit for every capture. Its complete expansion is visible before confirmation.
_Avoid_: fixed-amount macro, scheduled transaction, opaque expansion

**Quick-entry duplicate guard**:
The combination of a single-use preview token that prevents repeated publication and a non-blocking warning for an equivalent existing transaction. A ledger owner may explicitly confirm a legitimate business duplicate, and no OrangeCount-specific transaction identifier is written into the source ledger.
_Avoid_: retry duplication, duplicate rejection, private transaction ID

**Quick-entry batch**:
A set of independent, single-line quick-entry transactions compiled and previewed with line-specific results, then published atomically as one reviewed write. Any invalid or unresolved line prevents the entire batch from changing the source ledger.
_Avoid_: partial batch write, cross-line grammar, implicit continuation

**Effective quick-entry rule**:
The latest quick-entry-profile definition dated no later than the transaction it compiles. Future rules do not affect historical capture, same-day competing definitions are ambiguous, and dated retirement preserves rather than deletes configuration history.
_Avoid_: current-profile lookup, destructive alias edit, future-rule backfill

**Web-first quick entry**:
The first delivery surface for quick-entry notation: a keyboard-accessible capture interface in the local web session that shares its snapshot, preview, concurrency, rollback, and publication safeguards. The compiler remains UI-independent, while a mutating CLI is deferred until it can use the same reviewed write implementation.
_Avoid_: web-only compiler, separate CLI writer, divergent publication path

**Quick-entry profile manager**:
The required first-version reviewed interface for listing effective aliases and templates and creating, superseding, or retiring their dated Beancount `custom` directives with account completion and validation. Direct source editing remains an advanced path to the same representation.
_Avoid_: hand-authored-only profile, hidden preference editor, destructive profile mutation

**Versioned quick-entry profile schema**:
The public, ledger-embedded representation whose standard Beancount `custom` type names carry an explicit schema version and whose fields use typed values and directive metadata. Unsupported versions are diagnosed and ignored by compilation; incompatible evolution receives a new version rather than changing an existing meaning.
_Avoid_: unversioned custom contract, JSON payload, silent schema reinterpretation

**Quick-entry profile diagnostic**:
A non-accounting problem in syntactically valid quick-entry-profile configuration that disables only the affected rule or quick-entry publication. It never invalidates the ledger snapshot or core reports; ordinary Beancount syntax errors retain their normal blocking behavior.
_Avoid_: ledger-blocking profile error, silently ignored rule, accounting diagnostic

**Quick-entry append placement**:
The first-version rule that publishes a confirmed batch only at the end of its visible target file and warns when its date predates that file's latest transaction. It never guesses document sections or chronologically reorders source content.
_Avoid_: automatic chronological insertion, section inference, source reordering

**Single-target quick-entry batch**:
A quick-entry batch whose transactions all publish to one visible target file in one atomic write and one undo boundary. Templates cannot override the batch target; capture for another file requires another batch.
_Avoid_: cross-file quick batch, template-routed write, partial multi-file rollback

**Ephemeral quick-entry draft**:
Unconfirmed notation retained only in the current page's memory, with a loss warning on dismissal and immediate clearing after successful publication. It is never written to browser persistence, server files, logs, or the source ledger and does not survive reload or restart.
_Avoid_: recovered DSL draft, local-storage capture history, server-side shorthand queue

**Portable quick-entry syntax**:
The locale-independent lexical form that uses ungrouped decimal amounts, canonical commodities, and fixed ASCII structure symbols while permitting Unicode in aliases and descriptive text. Interface guidance is localized, but changing display locale never changes compilation.
_Avoid_: locale-sensitive amount, translated operator, display-driven parsing

**Explicit quick-entry output**:
The canonical Beancount transaction emitted with both source and destination posting amounts and commodities written explicitly in value-flow order. Generated source remains fully readable and auditable without relying on balance interpolation or OrangeCount-specific metadata.
_Avoid_: elided balancing amount, generated private metadata, opaque posting order

**Quick-entry undo**:
The reviewed restoration of the current web session's most recently published quick-entry batch, available only while its resulting ledger snapshot remains current. It previews the exact removal and uses the atomic, backed-up, revalidated write path; later ledger changes require manual correction instead.
_Avoid_: general history, automatic merge, unconditional rollback

### Diagnostics and repair guidance

**Core-derived report**:
A report whose result can be obtained solely from the v3 source ledger and its explicitly supported options, without executing a plugin or reading an OrangeCount-specific extension.
_Avoid_: plugin report, proprietary report

**Fava budget projection**:
A read-only interpretation of Fava's built-in `custom "budget"` directives for Journal and account-report presentation. It never changes balances, inventories, booking, or other ledger semantics.
_Avoid_: budget plugin, accounting balance, proprietary budget model

**Diagnostic**:
An actionable explanation of a compatibility or accounting problem, anchored to its source span and any related ledger locations.
_Avoid_: parser error, warning text

**Repair guidance**:
The user-facing, non-mutating explanation attached to a diagnostic that identifies the cause, relevant ledger context, and a safe next edit for the ledger owner to make. It never changes a source ledger or replaces accounting judgment.
_Avoid_: auto-fix, automatic repair, write-back

**Blocking ledger problem**:
A source-ledger problem that prevents OrangeCount from producing a valid ledger snapshot, including invalid syntax and failed core accounting validation. It is the first repair-guidance scope; compatibility migration and query/report problems follow later.
_Avoid_: common error, warning, migration issue

**Web-first repair experience**:
The delivery model in which the web interface provides the full repair guidance for a blocking ledger problem, while the CLI exposes its concise, script-friendly counterpart. Both representations derive from the same guidance content and diagnostic code.
_Avoid_: web-only help, separate CLI documentation, divergent advice

**Non-prescriptive repair guidance**:
Repair guidance that identifies the ledger construct to inspect and may show generic before-and-after examples, but never infers a ledger owner's account, amount, commodity, or accounting intent. It provides no one-click or directly applicable source-ledger change.
_Avoid_: auto-fix, generated patch, prescriptive bookkeeping

**Complete blocking-guidance coverage**:
The release condition that every currently emitted error-severity diagnostic code has repair guidance. A code may use a documented generic fallback only while it is not part of the released error catalogue.
_Avoid_: best-effort help, selected-error coverage, silent fallback

**Repair-guidance anatomy**:
The common content structure for repair guidance: what happened, why it blocks the ledger, where to inspect, how to check or modify it safely, and a generic example with a revalidation next step. The web interface initially shows the conclusion and source locations; the CLI shows a concise action and help topic.
_Avoid_: unstructured error article, prose-only error text, full help dump

**Localized repair guidance**:
Repair guidance whose complete anatomy, examples, and concise CLI action are shipped in every supported display locale. Beancount directives, account names, and diagnostic codes remain canonical rather than translated.
_Avoid_: English fallback in a released locale, translated directives, partial help translation

**Offline repair knowledge base**:
The versioned, bundled collection of repair-guidance topics keyed by stable diagnostic code. It is accessible through local web deep links and CLI topic identifiers without network, external AI, or third-party documentation dependencies.
_Avoid_: online-only manual, external-doc redirect, runtime knowledge lookup

**Repair order**:
The user-facing ordering that presents source-graph, encoding, and syntax problems as the first repair batch, then asks the ledger owner to revalidate before acting on account, transaction, assertion, inventory, or option problems. It retains every diagnostic and preserves source ordering within each batch; it is not a claim of exact causality.
_Avoid_: hidden diagnostics, causal-proof ordering, source-order-only triage

**Local repair context**:
The on-demand display of the diagnostic line, adjacent source lines, and related locations within the local web session. It is never added to diagnostics transport, URLs, logs, committed screenshots, or bundled help topics; the CLI provides locations without source excerpts.
_Avoid_: diagnostic payload excerpt, logged ledger context, remote source preview

**Static first-version guidance**:
The first release of repair guidance uses stable diagnostic codes, source locations, local repair context, and generic examples only. Ledger-specific computed explanations are deferred until each fact can be independently validated and privacy-bounded.
_Avoid_: dynamic repair facts, inferred adjustment, ledger-specific fix recommendation

**Repair-guidance release gate**:
The acceptance condition requiring complete bilingual guidance for every released error code, a triggering fixture and consistent web/CLI/deep-link behavior for each topic, local-only source context, no network requests, and representative successful revalidation after following the guidance.
_Avoid_: content-only review, untested help page, partial-code release

**Ledger snapshot**:
An immutable, fully evaluated view of a source ledger that the interface can safely query. A failed reload never replaces the most recent valid snapshot.
_Avoid_: live parse, partial ledger

**Deterministic report**:
A query or report whose ordering, temporal interpretation, and valuation rules are explicit and produce the same result for the same ledger snapshot.
_Avoid_: best-effort report, incidental order

**Query compatibility**:
The current BeanQuery language semantics used by Fava for ledger queries. Legacy query constructs may receive migration diagnostics but do not establish a second result-semantic contract.
_Avoid_: legacy query parity, dual query engine

**Fava filter expression (FQL)**:
The Fava 1.30.12 interface language for narrowing visible entries and report data through account, tag, link, payee, amount, and logical predicates. It is a UI filtering contract distinct from BeanQuery and does not define accounting semantics.
_Avoid_: query, BeanQuery filter, plain text search

**Fava time filter**:
The Fava 1.30.12 interface language for selecting calendar, fiscal, relative, and explicit date intervals in reports and journals. It determines visible scope without changing ledger semantics.
_Avoid_: date prefix, from/to fields, browser-local period

**Display locale**:
The explicitly selected language and regional formatting convention used to present OrangeCount's interface, diagnostics, dates, and numbers without changing ledger semantics.
_Avoid_: browser-default locale, account language

**Display amount**:
A localized, human-readable rendering of an exact ledger amount. It may round a non-terminating computed value for readability while the accounting engine and machine exports retain the exact value.
_Avoid_: rational implementation value, floating-point amount

**Supported locale**:
A display locale shipped with the product's complete interface and diagnostic translation catalog. The initial supported locales are Simplified Chinese (`zh-CN`) and English (`en`).
_Avoid_: partial translation, fallback-only language

**Average cost booking**:
OrangeCount's first implementation of Beancount's `Booking.AVERAGE` (an enum Beancount v3 defines but leaves disabled, returning "AVERAGE method is not supported"). It is per-account opt-in via the standard `booking "AVERAGE"` open directive; the default booking (FIFO) is unchanged. Lots are merged lazily at reduction time into a single weighted-average lot whose date is the earliest contributing lot's date; reductions with an explicit cost and cross-cost-currency merges are rejected with diagnostics. It never rewrites the source ledger and produces no intermediate files.
_Avoid_: eager merge, global booking change, display-only average

**Stat-only change detection**:
The watch loop's contract for deciding that a ledger changed: stat the entry, every file of the latest attempted include graph, and every path reported by an `E-INCLUDE-READ` diagnostic, and rebuild when any size or modification time differs. File contents are never re-read to detect change (ADR-0044).
_Avoid_: content-hash patrol, eager re-read, directory notifications

### Dialect language

**Dialect line**:
The one-line two-posting shorthand accepted inside ledger files as a Beancount v3 superset: `[DATE] [!] AMOUNT [CURRENCY] @source -> @destination ["payee"] [: narration] [#tag] [^link]`, compiled into an ordinary transaction at snapshot build (ADR-0045).
_Avoid_: quick-entry input, template invocation, natural language

**Block anchoring**:
The dialect date rule: a dialect line without a date inherits the date of the nearest preceding dialect line that has one, in the same file; never the wall clock and never a non-dialect directive.
_Avoid_: default today, file mtime, compile-time date

**Three-level endpoint resolution**:
How a dialect endpoint name becomes an account: exact full name, then date-effective declared alias, then a unique tail-segment match over opened accounts; ambiguity or no match is an error that lists candidates.
_Avoid_: fuzzy match, best guess, case-insensitive match

**Dialect export**:
`orangecount export`: a disposable, pure-Beancount-v3 snapshot emitted from a dialect source for external tools. Hand-editing an export is prohibited; changes go only to the dialect source (ADR-0045).
_Avoid_: synchronized copy, mirror file, second ledger

**Dialectize**:
`orangecount dialectize`: the reverse filter that rewrites a v3 ledger's trivially-representable transactions into dialect lines and preserves every other entry byte-for-byte, with round-trip balance and fixpoint guaranteed by property tests.
_Avoid_: full translation, lossy conversion, normalization
