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
The Go-served personal ledger interface that provides the essential exploration and reporting workflows associated with Fava.
_Avoid_: external dashboard, Fava integration

**Workflow compatibility**:
The ability to complete the same personal ledger analysis workflows as Fava without reproducing its implementation, HTTP API, plugin ecosystem, or visual design.
_Avoid_: API compatibility, pixel parity

**Fava parity**:
The ability to complete Fava's local personal-ledger interface workflows with materially equivalent page structure, controls, interaction states, and outcomes. OrangeCount may use an independent implementation while preserving the user-visible workflow.
_Avoid_: superficial theme match, API compatibility

**Local web session**:
A built-in web-interface session initiated by the OrangeCount CLI and bound only to the loopback network interface for its owner.
_Avoid_: hosted account, shared instance

**Source ledger**:
The user-maintained `.bean` file set, including its include graph, that remains OrangeCount's authoritative accounting record.
_Avoid_: application database, managed ledger

**Private reference ledger**:
A source ledger used only on its owner's machine to verify a release against real-world usage. A release must parse and evaluate it into a valid ledger snapshot without copying its content into the project.
_Avoid_: committed fixture, sample ledger

**Document root**:
An explicitly configured filesystem root from which OrangeCount may resolve attachments named by a `document` directive after path normalization and containment checks.
_Avoid_: arbitrary local path, file browser

**Read-only interface**:
A web interface that explores, validates, and links to source-ledger locations but does not modify ledger files.
_Avoid_: embedded editor, ledger authoring

**Core-derived report**:
A report whose result can be obtained solely from the v3 source ledger and its explicitly supported options, without executing a plugin or reading an OrangeCount-specific extension.
_Avoid_: plugin report, proprietary report

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

**Display locale**:
The explicitly selected language and regional formatting convention used to present OrangeCount's interface, diagnostics, dates, and numbers without changing ledger semantics.
_Avoid_: browser-default locale, account language

**Display amount**:
A localized, human-readable rendering of an exact ledger amount. It may round a non-terminating computed value for readability while the accounting engine and machine exports retain the exact value.
_Avoid_: rational implementation value, floating-point amount

**Supported locale**:
A display locale shipped with the product's complete interface and diagnostic translation catalog. The initial supported locales are Simplified Chinese (`zh-CN`) and English (`en`).
_Avoid_: partial translation, fallback-only language
