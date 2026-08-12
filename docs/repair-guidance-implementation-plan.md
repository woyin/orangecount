# Repair guidance: design and implementation plan

## Outcome

Make every released, error-severity OrangeCount diagnostic understandable and
actionable without modifying a ledger. A ledger owner can move from a
diagnostic to a localized explanation, the relevant local source context, a
safe generic example, and a revalidation step. The web interface is the full
experience; the CLI exposes the same topic and a short next action.

This plan implements the domain language in [`CONTEXT.md`](../CONTEXT.md):
repair guidance, complete blocking-guidance coverage, repair order, local
repair context, static first-version guidance, and the repair-guidance release
gate.

## Confirmed product decisions

| Decision | Chosen behavior |
| --- | --- |
| Initial scope | Every currently released `error` diagnostic code; warnings, plugin migration, queries, reports, and imports follow later. |
| Advice boundary | Guidance is non-prescriptive: it may identify the construct to inspect and show generic before/after snippets, but never guesses accounts, amounts, commodities, or intent, and never produces a patch or write action. |
| Presentation | Web: expandable full guidance. CLI: existing stable diagnostics plus a concise action and a stable help topic. Both use the same catalogue. |
| Locales | `en` and `zh-CN` ship complete guidance. Beancount directives, account placeholders, and error codes remain canonical. |
| Ordering | Show source-graph, encoding, and parser problems as “fix first”; retain all diagnostics and source ordering within each group; tell the user to revalidate before acting on subsequent semantic problems. |
| Context | A web user may request the failing line plus one adjacent line on either side and related locations. It is local, on demand, and never appears in diagnostics JSON, URLs, logs, CLI output, or bundled help content. |
| Offline | Guidance is versioned and embedded in the binary. It makes no remote request and has no runtime AI or third-party documentation dependency. |
| First-version facts | No ledger-specific computed explanation (for example, a calculated balancing amount). Such facts are a later, per-code addition after semantic and privacy validation. |

## Scope and non-goals

### In scope

- Guidance for the following released error codes:

  - Source graph: `E-INCLUDE-CYCLE`, `E-INCLUDE-READ`.
  - Source encoding and parsing: `E-SOURCE-UTF8`, `E-PARSE-DATE`,
    `E-PARSE-DIRECTIVE`, `E-PARSE-EXPECTED`, `E-PARSE-TOKEN`, and
    `E-PARSE-STRING`.
  - Account lifecycle and currencies: `E-EVAL-OPEN`, `E-EVAL-REOPEN`,
    `E-EVAL-CLOSE`, `E-EVAL-POSTING`, and `E-EVAL-CURRENCY`.
  - Transaction construction: `E-EVAL-UNBALANCED`, `E-EVAL-INFER`.
  - Assertions and inventory: `E-EVAL-BALANCE`, `E-EVAL-PAD`,
    `E-EVAL-TOLERANCE`, and `E-EVAL-INVENTORY`.
  - Configuration: `E-EVAL-OPTION`.

- Details in both the current embedded UI and the transplanted Fava UI while
  the `ORANGECOUNT_TRANSPLANTED_UI` flag still selects between them.
- A local help deep link for every topic and an explicit CLI help command.
- Fixture-based Go, frontend, and browser/HTTP coverage described below.

### Not in scope

- Automatic fixes, editor write shortcuts, generated patches, or any
  modification of the reviewed-write workflow.
- Executing plugins, supporting a new Beancount dialect, or changing
  diagnostic codes, severity, parser recovery, or evaluator semantics.
- Runtime network documentation, telemetry, LLM calls, or copying a ledger
  into a help topic, URL, log, JSON diagnostic, or visual fixture.
- Guidance for warning-level diagnostics and non-ledger problems in this
  release.

## Design

### The repair-guidance module

Add `internal/repairguidance`. It is the deep module for this feature: its
callers ask for one code and locale, while it owns topic IDs, coverage,
localization, content anatomy, repair-order classification, and the rules that
make an unpublished code fail validation.

Its small interface should be shaped around immutable values rather than UI
rendering:

```go
type Guide struct {
    Code, Topic, Phase string
    ShortAction         string
    What, Why           string
    Inspect, SafeSteps  []string
    Example             Example
    Revalidate          string
}

func Lookup(code, locale string) (Guide, bool)
func Order(code string) RepairPhase
func ValidateCoverage(releasedErrorCodes []string) error
```

`Guide` is presentation-neutral. The module owns the English and Simplified
Chinese strings, fixed generic snippets, and code-to-phase mapping. It accepts
only code and locale; it cannot read a ledger, calculate an adjustment, write a
file, or make a request. This gives the CLI, the two web clients, and tests one
interface and one source of truth. Internal parsing of authored catalogue data
may be private to the module.

Expose a diagnostic-package function or immutable catalogue view that returns
the released error codes. Do not make `repairguidance` scrape a private message
map or maintain an unverified second list. A test calls
`ValidateCoverage(diagnostic.ReleasedErrorCodes())`, so adding a released error
code requires guidance in both locales.

The diagnostic wire types remain intentionally small. Do **not** add guide
prose or source excerpts to `diagnostic.Diagnostic`, `RenderJSON`, or the
existing `/api/v1/diagnostics` response. The web obtains a guide by code, and
context only through the dedicated on-demand operation below.

### Topic and help contract

Use the stable topic format `diagnostics/<CODE>`, for example
`diagnostics/E-EVAL-UNBALANCED`. The canonical browser URL is
`/help/diagnostics/E-EVAL-UNBALANCED`; it contains no ledger identifiers.

Extend the local help projection to return either its existing index or a
single typed topic. It must draw all diagnostic topics from
`repairguidance.Lookup`, rather than copy strings into `internal/web`. The
legacy `/api/v1/help` and private Fava adapter must return the same guide
shape. Add `orangecount help diagnostics/<CODE> --locale en|zh-CN` to render a
full terminal-safe topic, and append a single concise `help:` topic/action to
human-readable `orangecount check` diagnostics. Preserve existing JSON output
exactly; scripts continue to receive only the stable diagnostic schema.

Unknown, warning, and malformed help topics return a localized local-help
“not found” response, never an external link. The generic Help index should
include a Diagnostics entry and may list code topics by error family.

### Repair context contract

The current source endpoint reads only the last valid snapshot. That is not
sufficient when the newest reload failed—the precise time at which repair
context is most valuable. Extend the snapshot build/store path to retain a
read-only reference to the source graph of the **latest attempted build**,
whether it published a snapshot or not. This is in-memory local data only; it
does not change source ownership or publish an invalid snapshot.

Add a read-only, same-origin-independent GET operation such as
`/api/v1/diagnostics/context?path=<display-path>&line=<line>`. Its interface is
strict:

- resolve `path` only through the latest-attempt graph's display-path map;
  reject arbitrary and absolute filesystem paths;
- return at most the focus line plus one previous and one next line, numbered,
  with the display-safe path and requested line;
- return a localized availability reason (rather than source content) if the
  source cannot be read, including an include-read error;
- return no raw absolute path and make no log entry containing source text;
- accept no `start`, `end`, arbitrary range, or search query.

The guide detail calls this endpoint only after an explicit user action such as
“Show local context.” A source-location link still opens the existing source
or editor route. Related locations use the same display-safe lookup. This
separate seam keeps source text out of the generic diagnostics transport.

### Web presentation

Both UI variants consume the same guide and context projections.

1. **Diagnostics list.** Partition errors into `fix-first` (include, UTF-8,
   parser) and `recheck-after` (evaluation/option) groups. Retain every row and
   stable path/line ordering inside a group. Explain that the grouping is a
   repair sequence, not a causality claim.
2. **Row detail.** A row exposes its code, short action, a “Learn how to fix”
   link, and a keyboard-operable expand/collapse detail. The detail renders the
   five agreed sections: what happened, why it blocks, where to inspect,
   safe checks/changes, and generic example plus revalidation.
3. **Local context.** The closed detail contains no source data. On request it
   fetches and displays the bounded local context, provides source/related
   location links, and retains focus/ARIA state correctly.
4. **Help.** `/help/diagnostics/<CODE>` presents the same full topic without a
   particular ledger. The existing generic help route remains intact.

Implement the experience in `web/src/fava/reports/ErrorsReport.svelte` and
the utility/help route first, including types in `adapter-client.ts` or a
dedicated repair-guidance client module. Mirror the minimum grouped list,
details, local context action, and local help links in
`internal/web/assets/app.js` until the legacy asset is removed. Update the
transplant asset through its normal deterministic build path—never hand-edit
`internal/web/assets/transplanted/app.js`.

### Privacy, safety, and accessibility invariants

- A guide is static content; it contains only generic placeholders and sample
  directives.
- No ledger-specific values appear in `Guide`, help URLs, CLI output,
  `RenderJSON`, diagnostics endpoints, or logs.
- The context endpoint is loopback-only by server construction, bounded to
  graph members and three lines, and invoked only on user request.
- Details use native `details/summary` or equivalent buttons with announced
  expanded state; links and context controls work via keyboard; context loading
  and unavailable state are announced without moving focus unexpectedly.
- Fava visual-parity work uses only the synthetic fixture. No private source
  excerpt is captured in a screenshot or committed fixture.

## Implementation sequence

### Phase 0 — catalogue and coverage foundation

1. Make released diagnostic codes discoverable from `internal/diagnostic`
   without changing existing messages or output formats.
2. Create `internal/repairguidance` with typed guides, phases, topic IDs,
   English and Simplified Chinese authored content, and every code in the
   scope table.
3. Add pure tests for lookup, locale completeness, anatomy completeness,
   canonical code preservation, phase classification, topic stability, and
   the coverage guard.
4. Add a fixture manifest with one minimal input per code and expected code,
   phase, and revalidation outcome. Reuse existing parser/evaluator fixtures
   where they already express the condition; add focused synthetic ledgers only
   for missing cases.

Exit: a newly released error code fails tests until it has complete bilingual
guidance, and no guide depends on source text or runtime I/O.

### Phase 1 — local projections and CLI

1. Extend build/store state to retain the latest attempted graph safely.
2. Implement the bounded context resolver and its HTTP handler with graph
   membership, path-redaction, line-bound, unavailable-source, and failed
   reload tests.
3. Extend help projection/handler and Fava adapter help route to look up a
   diagnostic topic by stable ID; retain the existing help index contract.
4. Add the `help` CLI command and human `check` short action/topic line.
   Maintain byte-for-byte compatibility of `check --json`.
5. Test locale selection, unknown topic behavior, CLI redaction, and no
   network side effect.

Exit: CLI and HTTP consumers can retrieve identical guide content by code;
failed reload diagnostics can obtain only bounded local context.

### Phase 2 — web integration

1. Add typed adapter-client guide/context operations and tests for malformed
   payload rejection.
2. Implement grouped, expandable diagnostics in the transplanted
   `ErrorsReport`, its standalone Help topic, keyboard/focus handling, and
   localized copy.
3. Add the corresponding minimal behavior to the current embedded UI so the
   runtime flag cannot create a feature/security discrepancy.
4. Wire source and related-location navigation using display paths only.
5. Build the transplanted asset with `make web-build-embedded`; commit only
   the deterministic generated asset expected by repository policy.

Exit: either UI selection renders full guidance and on-demand context for the
same diagnostics, with no ledger content present before the context action.

### Phase 3 — end-to-end release gate

1. Execute every fixture through parser/evaluator/snapshot and assert guide,
   phase, web detail, CLI topic, and local help URL consistency.
2. For one representative case in each error family, apply the documented
   generic correction to a synthetic ledger and assert successful revalidation.
   Cases requiring owner judgment must instead prove that the guidance stays
   non-prescriptive.
3. Add HTTP/browser tests proving context is absent from diagnostics JSON,
   HTML before request, URLs, and logs; after the request, it is bounded and
   display-path safe.
4. Run offline/network interception coverage on diagnostics and help routes.
5. Run the existing Go and web quality gates plus the relevant Fava visual
   fixture cases in both locales/themes.

Exit: the repair-guidance release gate in `CONTEXT.md` is demonstrably met.

## Acceptance matrix

| Concern | Required proof |
| --- | --- |
| Coverage | A test compares `diagnostic.ReleasedErrorCodes()` with the repair catalogue and requires all five sections plus `en`/`zh-CN` content. |
| Correct topic | Each triggering fixture maps to its exact code, phase, short action, and `diagnostics/<CODE>` topic. |
| CLI compatibility | Human `check` adds only the concise action/topic; `check --json` remains the existing diagnostic array schema. `help` renders locally in both locales. |
| Repair ordering | A mixed parse/evaluation fixture shows all rows, with parser/source rows first and preserved ordering within groups. |
| Context privacy | The diagnostics payload and help route contain no source excerpt; the context route returns only graph-member display paths and three or fewer lines. |
| Failed reload | After a valid snapshot then invalid source change, the UI retrieves context from the latest attempted graph while reports retain the prior valid snapshot. |
| Offline | Browser/network tests observe no outbound request when opening details, help, or context. |
| Accessible UI | Keyboard tests cover opening/closing details, context loading/unavailable feedback, and source/help navigation; visual checks cover narrow and desktop layouts. |
| No semantic drift | Existing parser/evaluator/snapshot tests pass unchanged; guidance introduces no write path or accounting result change. |

## Delivery risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Two UI variants drift | Keep guide/context server projections UI-neutral and add shared fixture contract tests; make legacy parity an explicit Phase 2 exit criterion. |
| A new code lacks help | Derive coverage from the diagnostic catalogue and make the test mandatory. |
| Invalid reload has no context | Retain the latest attempted source graph separately from the last valid snapshot, with explicit lifecycle tests. |
| Source leakage | Never enrich diagnostic wire values; use a narrow, on-demand context resolver that accepts only display paths and bounded lines. |
| Guidance accidentally becomes advice to alter books | Review every topic against the non-prescriptive rule and test for placeholders/generic snippets rather than ledger-derived values. |
| Transplant generated asset is edited by hand | Change Svelte/TypeScript source, use the established embedded build, and verify provenance/build checks. |

## Definition of done

The feature is done only when every scoped code has complete bilingual
guidance, all projections and both UI selections use that same catalogue, the
latest attempted graph safely supports bounded local context, and all
acceptance-matrix checks pass. The follow-on backlog starts with warning and
plugin-migration guidance, then individually validated ledger-specific
explanatory facts; neither is part of this release.
