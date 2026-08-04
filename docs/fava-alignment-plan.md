# Fava interface alignment plan

OrangeCount aligns to the pinned Fava 1.30.12 standard surface through a selective frontend transplant, not a clean-room visual approximation. Fava is the authority for observable interface composition and behavior; OrangeCount and Beancount v3 remain the authority for accounting and query semantics.

The authoritative implementation sequence, model ownership, route gates, and completion rules are in [fava-frontend-transplant-plan.md](fava-frontend-transplant-plan.md).

## Acceptance method

Prerequisite Phase 0 will establish a development-only OCI reference environment that runs Fava 1.30.12 with the deterministic synthetic reference ledger. It will generate English Chromium light/dark and desktop/narrow baseline candidates, which become authoritative only after explicit approval by the user (product owner). OrangeCount must then have zero unapproved visual differences; remaining differences require an entry in [fava-approved-deviations.md](fava-approved-deviations.md) and the same user approval.

`docs/fava-route-state-manifest.md` is the canonical coverage and status
registry. Each listed route must independently pass four gates before cutover:

1. adapter contract and OrangeCount/v3 semantics;
2. task behavior, keyboard/focus, error recovery, and write safety;
3. English visual fidelity plus Simplified Chinese structural fidelity;
4. performance, accessibility, offline/CSP, provenance/license, and release quality.

Private-ledger observations remain transient. Committed screenshots and browser evidence may use only the deterministic synthetic reference ledger.

## Prerequisite and seven delivery waves

- **Prerequisite Phase 0 — Reference and evidence:** requested Herdr agents, OCI reference runner, canonical route/state manifest, dense synthetic ledger, four-cell English baseline, deviation registry and provenance tooling.
1. **Golden slice:** actual Fava-derived shell plus a complete Income Statement vertical slice over the private Go adapter.
2. **Core reports:** Balance Sheet and Trial Balance, natural currencies, intervals/conversion, hierarchy charts, printing and unavailable states.
3. **Journal and accounts:** Go-rendered Fava-compatible Journal markup, complete FQL/time filters, account detail, budgets, statements and filtered export.
4. **Secondary pages:** Holdings, Commodities, Documents, Events, Statistics, Errors and Help.
5. **Query and export:** Fava Query experience over BeanQuery with CSV/XLSX/ODS.
6. **Reviewed authoring:** Editor, Source Slice, Add Entry, Import, Options and mutating Document operations through the atomic write/snapshot path.
7. **Final hardening:** complete language/theme/viewport/browser matrix, performance, accessibility, provenance and legacy removal.

## Scope boundaries

- Included: every built-in user-visible Fava 1.30.12 page, modal, keyboard path, option, budget projection, export, statement/document operation, editor and importer workflow that does not require an excluded runtime.
- Excluded: Python importer execution, Python/Beancount runtime, third-party plugins and extension pages, multi-user hosting, and public Fava HTTP API compatibility.
- OrangeCount-exclusive pages are frozen and isolated from the Fava standard navigation until migration completion.
- Editor, Import, Add Entry and document writes remain explicit, reviewed, atomic, recoverable and revalidated before snapshot publication.
