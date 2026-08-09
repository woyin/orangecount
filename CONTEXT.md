# OrangeCount

OrangeCount is a personal, Go-native accounting system that can read and validate existing Beancount ledgers without a Python Beancount runtime.

## Language

**Compatible ledger**:
A Beancount ledger whose accepted syntax and accounting meaning OrangeCount preserves, subject to explicitly documented compatibility boundaries.
_Avoid_: import format, legacy file

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

**Source ledger**:
The user-maintained `.bean` file set, including its include graph, that remains OrangeCount's authoritative accounting record.
_Avoid_: application database, managed ledger

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
A local web operation that previews or validates an explicit source-ledger or document change and publishes it only through an atomic, recoverable, revalidated path. Uncommitted editing never changes the source ledger or active snapshot.
_Avoid_: autosave, direct file mutation, partial snapshot publication

**Core-derived report**:
A report whose result can be obtained solely from the v3 source ledger and its explicitly supported options, without executing a plugin or reading an OrangeCount-specific extension.
_Avoid_: plugin report, proprietary report

**Fava budget projection**:
A read-only interpretation of Fava's built-in `custom "budget"` directives for Journal and account-report presentation. It never changes balances, inventories, booking, or other ledger semantics.
_Avoid_: budget plugin, accounting balance, proprietary budget model

**Diagnostic**:
An actionable explanation of a compatibility or accounting problem, anchored to its source span and any related ledger locations.
_Avoid_: parser error, warning text

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
