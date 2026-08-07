# Approved Fava deviations

This registry is the only place where OrangeCount may accept an observable difference from the pinned Fava 1.30.12 visual baseline or behavior. An implementing agent may propose a deviation and produce evidence, but only the user, acting as product owner, may approve it.

A deviation is valid only when required by OrangeCount's accounting semantic authority, security, data integrity, privacy, or accessibility obligations. Convenience, modernization, product preference, implementation cost, and subjective improvement are not valid reasons.

## Status vocabulary

- `proposed` — evidence exists, but the difference still fails its route gate.
- `approved` — the user (product owner) accepted the bounded difference.
- `expired` — the approval no longer applies and the route fails until reviewed.
- `removed` — OrangeCount once differed but now matches Fava.

## Registry

### FD-0001 — Import file list and per-entry extract/review rely on Python importers

| Field | Value |
| --- | --- |
| Status | proposed |
| Route and state | R-IMPORT (`import`) |
| Fava baseline | Import page "Importable Files" list (`FileList.svelte`) + per-entry "Extract" modal (`Extract.svelte`) driven by Python importers |
| OrangeCount behavior | Import is a single-file local-buffer workflow (upload/paste one file, pick Source path/Adapter/Target, Preview, Commit); no server-side import directory, no file list, no per-entry extract/review modal |
| Category | semantics |
| Reason | OrangeCount is a Go-native EagerLedger that approved excluding the Python importer ecosystem (decision D3). Upstream file attribution (identify/file_account/file_date) and per-entry extraction (extract into structured directives) live in user-supplied Python importer classes (beangulp, fava/core/ingest.py); the Go runtime cannot equivalently reproduce importer file recognition, entry extraction, or account inference, and commit writes appended text rather than per-entry add_entries. Reproducing importer intelligence in Go has no bounded recognition correctness. |
| Scope | import route: file list, per-entry extract/review modal, importer-driven auto-attribution; the local upload + generic beancount/csv adapter preview→commit flow is retained |
| Tests | M2-import-upload smoke (DataTransfer upload → local buffer load, path/adapter backfill, Preview diagnostics, Commit stays disabled until valid) |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | none yet |
| Expiry condition | a Fava upgrade that changes the Import contract, or an OrangeCount decision to embed a supported importer runtime |
| Baseline impact | alternate expectation for the Import page |

No other deviations are currently approved. Existing differences in the prototype and legacy UI are migration gaps, not approved deviations.

## Entry template

```markdown
### FD-0001 — Short name

| Field | Value |
| --- | --- |
| Status | proposed / approved / expired / removed |
| Route and state | route-manifest identifier |
| Fava baseline | baseline path and bounded region |
| OrangeCount behavior | precise observable difference |
| Category | semantics / security / data integrity / privacy / accessibility |
| Reason | why matching Fava would violate the named obligation |
| Scope | exact themes, viewports, locales, controls, and states covered |
| Tests | contract, browser, visual, and safety evidence |
| Owner | implementing agent or maintainer |
| Approver | user (product owner) only |
| Approved evidence | review reference and date |
| Expiry condition | event that requires re-review, including a Fava upgrade |
| Baseline impact | masks, alternate expectation, or none |
```

## Rules

1. Baseline candidates are never promoted merely because OrangeCount changed.
2. Masks must be limited to the entry's exact nondeterministic region; a deviation is not permission for a global visual threshold.
3. A semantic deviation must preserve visible access to exact OrangeCount/v3 results rather than hiding the conflict.
4. A security, integrity, or privacy deviation must keep the Fava information hierarchy wherever the protected behavior allows.
5. Accessibility changes should preserve ordinary visual composition and document keyboard/focus differences.
6. Upgrading the pinned Fava version expires every approval unless the entry explicitly proves that its reason and scope remain unchanged.
