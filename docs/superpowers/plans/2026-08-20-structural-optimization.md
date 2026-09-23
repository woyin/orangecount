# OrangeCount Structural Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a UI-independent reviewed-write module, migrate every source-ledger mutation to it, improve Go file locality, and establish complete frontend quality gates without changing observable behavior.

**Architecture:** `internal/authoring` owns ledger proposals, canonical entry serialization, single-file publication, full-graph validation, rollback, and conditional revert. `quickentry` and `web/favaadapter` depend on authoring; `web` maps transport concerns and retains attachment/options behavior. Existing Go packages remain intact while oversized Web and CLI files are split by workflow.

**Tech Stack:** Go 1.22+, standard library only at runtime, Svelte 5, TypeScript 5.6, Node 22, node:test, Playwright, Make.

---

## Global constraints

- Preserve all CLI exit codes and output formats.
- Preserve private Fava-shaped request/response JSON and HTTP status behavior except that verified source drift returns conflict instead of overwriting data.
- Preserve Beancount and dialect semantics.
- Use a clean cutover: no type aliases, forwarding serializers, deprecated methods, or duplicate write paths remain.
- Do not introduce filesystem/store interfaces.
- Do not commit unless the user explicitly requests it; use diff checkpoints instead.

### Task 1: Restore enforceable quality gates

**Files:**
- Modify: `internal/report/pivot.go`
- Modify: `web/package.json`
- Modify: `web/package-lock.json`
- Modify: `Makefile`

- [ ] **Step 1: Reproduce the Go format failure**

Run: `make fmt`

Expected: FAIL and list `internal/report/pivot.go` as requiring gofmt.

- [ ] **Step 2: Format only the failing Go file**

Run: `gofmt -w internal/report/pivot.go`

- [ ] **Step 3: Verify the Go format gate**

Run: `make fmt`

Expected: PASS with exit code 0.

- [ ] **Step 4: Reproduce the frontend type failure**

Run: `npm --prefix web exec -- tsc --noEmit`

Expected: FAIL with `TS2688: Cannot find type definition file for 'node'`.

- [ ] **Step 5: Establish the frontend module/type contract**

Update `web/package.json`:

```json
{
  "type": "module",
  "scripts": {
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "@types/node": "22.15.30"
  }
}
```

Keep every existing script and dependency unchanged. Run `npm --prefix web install --package-lock-only` so `web/package-lock.json` exactly records the pinned development dependency without upgrading unrelated packages.

- [ ] **Step 6: Add the Make gate**

Update the phony list and Web checks:

```make
.PHONY: ... web-test web-typecheck web-check ...

web-typecheck:
	$(NPM) --prefix web run typecheck

web-check: web-test web-typecheck check
```

- [ ] **Step 7: Verify frontend checks are clean**

Run:

```sh
npm --prefix web install
npm --prefix web test
npm --prefix web run typecheck
npm --prefix web run build:check
npm --prefix web run check:phase0
```

Expected: all commands pass; unit tests report 34 passing tests; Node emits no `MODULE_TYPELESS_PACKAGE_JSON` warnings.

### Task 2: Introduce the Ledger Proposal model

**Files:**
- Create: `internal/authoring/proposal.go`
- Create: `internal/authoring/proposal_test.go`
- Modify: `internal/quickentry/compiler.go`
- Modify: `internal/quickentry/coverage_test.go`
- Modify: `internal/web/favaadapter/addentry.go`
- Modify: `internal/web/favaadapter/favaadapter_test.go`

- [ ] **Step 1: Write failing proposal serialization tests**

Create tests that specify the final interface:

```go
func TestSerializeEntriesRendersTransactionBalanceAndNote(t *testing.T) {
    text, err := SerializeEntries([]Entry{
        {
            Type: "transaction", Date: "2026-08-06", Payee: `Bob's "Cafe"`, Narration: "Lunch",
            Tags: []string{"meal"}, Links: []string{"trip"},
            Postings: []Posting{
                {Account: "Assets:Bank:Cash", Amount: "-2.50", Currency: "USD"},
                {Account: "Expenses:Food"},
            },
        },
        {Type: "balance", Date: "2026-08-07", Account: "Assets:Bank:Cash", Amount: "10", Currency: "USD"},
        {Type: "note", Date: "2026-08-08", Account: "Assets:Bank:Cash", Comment: "checked"},
    })
    if err != nil { t.Fatal(err) }
    want := "2026-08-06 * \"Bob's \\\"Cafe\\\"\" \"Lunch\" #meal ^trip\n" +
        "  Assets:Bank:Cash -2.50 USD\n" +
        "  Expenses:Food\n\n" +
        "2026-08-07 balance Assets:Bank:Cash 10 USD\n\n" +
        "2026-08-08 note Assets:Bank:Cash \"checked\""
    if text != want { t.Fatalf("text=%q want=%q", text, want) }
}

func TestSerializeEntriesRejectsInvalidProposal(t *testing.T) {
    cases := []Entry{
        {Type: "transaction", Date: "2026-8-6", Narration: "x", Postings: []Posting{{Account: "Assets:A:B"}}},
        {Type: "transaction", Date: "2026-08-06", Flag: "?", Narration: "x", Postings: []Posting{{Account: "Assets:A:B"}}},
        {Type: "balance", Date: "2026-08-06", Account: "assets:bad", Amount: "1", Currency: "USD"},
        {Type: "note", Date: "2026-08-06", Account: "Assets:A:B", Comment: "line\nbreak"},
    }
    for _, entry := range cases {
        if _, err := SerializeEntries([]Entry{entry}); err == nil {
            t.Fatalf("accepted invalid entry: %+v", entry)
        }
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/authoring`

Expected: FAIL because the package/types/functions do not exist.

- [ ] **Step 3: Implement the domain types and serializer**

Create `proposal.go` with these public types:

```go
package authoring

type Entry struct {
    Type      string    `json:"type"`
    Date      string    `json:"date"`
    Flag      string    `json:"flag,omitempty"`
    Payee     string    `json:"payee,omitempty"`
    Narration string    `json:"narration,omitempty"`
    Account   string    `json:"account,omitempty"`
    Amount    string    `json:"amount,omitempty"`
    Currency  string    `json:"currency,omitempty"`
    Comment   string    `json:"comment,omitempty"`
    Tags      []string  `json:"tags,omitempty"`
    Links     []string  `json:"links,omitempty"`
    Postings  []Posting `json:"postings,omitempty"`
}

type Posting struct {
    Account  string `json:"account"`
    Amount   string `json:"amount,omitempty"`
    Currency string `json:"currency,omitempty"`
}

func SerializeEntries(entries []Entry) (string, error)
```

Move the validated serializer implementation from `internal/web/favaadapter/addentry.go` without changing accepted values or rendered text. Keep validation helpers private to `authoring`.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/authoring`

Expected: PASS.

- [ ] **Step 5: Move Quick Entry to the domain model**

Change `quickentry.LineResult.Entry` to `*authoring.Entry`, build `authoring.Posting` values, and call `authoring.SerializeEntries`. Update tests to import `internal/authoring` instead of `internal/web/favaadapter`.

- [ ] **Step 6: Move adapter serialization tests and delete adapter ownership**

Delete `NewEntry`, `NewPosting`, and serializer functions from `favaadapter/addentry.go`. Delete or move their tests from `favaadapter_test.go`; no aliases remain.

- [ ] **Step 7: Verify the clean dependency direction**

Run:

```sh
go test ./internal/authoring ./internal/quickentry ./internal/web/favaadapter
go list -deps ./internal/quickentry
```

Expected: tests pass; dependency output contains `orangecount/internal/authoring` and does not contain `orangecount/internal/web/favaadapter`.

### Task 3: Implement reviewed publication and conditional revert

**Files:**
- Create: `internal/authoring/change.go`
- Create: `internal/authoring/writer.go`
- Create: `internal/authoring/writer_test.go`

- [ ] **Step 1: Write failing change-construction tests**

Specify append and replace behavior:

```go
func TestAppendNormalizesOneBlankSeparator(t *testing.T) {
    change, err := Append("main.bean", "2000-01-02 note Assets:Cash \"x\"")
    if err != nil { t.Fatal(err) }
    got := change.apply([]byte("2000-01-01 open Assets:Cash USD\n"))
    want := []byte("2000-01-01 open Assets:Cash USD\n\n2000-01-02 note Assets:Cash \"x\"\n")
    if !bytes.Equal(got, want) { t.Fatalf("got=%q want=%q", got, want) }
}

func TestReplacePreservesExactSubmittedBytes(t *testing.T) {
    change, err := Replace("main.bean", []byte("option \"title\" \"new\"\n"))
    if err != nil { t.Fatal(err) }
    if got := change.apply([]byte("old")); !bytes.Equal(got, []byte("option \"title\" \"new\"\n")) {
        t.Fatalf("got=%q", got)
    }
}
```

Use package `authoring` tests so private `apply` can be tested without exporting mechanics.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/authoring -run 'TestAppend|TestReplace'`

Expected: FAIL because `Change`, `Append`, and `Replace` do not exist.

- [ ] **Step 3: Implement immutable Change constructors**

```go
type changeMode uint8
const (
    appendChange changeMode = iota + 1
    replaceChange
)

type Change struct {
    target  string
    mode    changeMode
    content []byte
}

func Append(target, block string) (Change, error)
func Replace(target string, content []byte) (Change, error)
func (c Change) Target() string
func (c Change) apply(current []byte) []byte
```

Reject blank targets and blank append blocks. Defensive-copy input/output bytes. Append must produce exactly one blank line before the block and exactly one final newline.

- [ ] **Step 4: Write failing Writer behavior tests**

Use real temp files and `snapshot.Build`/`snapshot.NewStore` to specify:

```go
func TestWriterPublishesAndRevertsAppend(t *testing.T)
func TestWriterRejectsStaleSnapshot(t *testing.T)
func TestWriterRejectsDiskDrift(t *testing.T)
func TestWriterRollsBackInvalidLedger(t *testing.T)
func TestWriterRejectsTargetOutsideGraph(t *testing.T)
func TestWriterRejectsRevertAfterLaterPublication(t *testing.T)
func TestWriterSerializesConcurrentPublications(t *testing.T)
```

Assertions must cover file bytes, current snapshot ID, diagnostics, backup path, and receipt availability.

- [ ] **Step 5: Verify RED**

Run: `go test ./internal/authoring -run Writer`

Expected: FAIL because `Writer`, `Publish`, `Revert`, and receipt types do not exist.

- [ ] **Step 6: Implement Writer**

Public interface:

```go
var (
    ErrSnapshotChanged = errors.New("snapshot changed")
    ErrSourceChanged   = errors.New("source file changed")
    ErrTargetMissing   = errors.New("target file unavailable")
    ErrValidation      = errors.New("ledger validation failed")
)

type Result struct {
    Build   snapshot.BuildResult
    Backup  string
    Receipt *Receipt
}

type Receipt struct {
    target              string
    before, after       []byte
    publishedSnapshotID string
}

type Writer struct {
    mu    sync.Mutex
    store *snapshot.Store
}

func NewWriter(store *snapshot.Store) (*Writer, error)
func (w *Writer) Publish(expectedSnapshotID string, change Change) (Result, error)
func (w *Writer) Revert(expectedSnapshotID string, receipt *Receipt) (Result, error)
```

Implementation invariants:

- Resolve the target by display path from the current graph.
- Compare `os.ReadFile(file.Path)` with `file.Data` before applying a change.
- Write `file.Path + ".orangecount.bak"` before every replacement.
- Use a temp file in the same directory, copy permissions, `Sync`, close, then rename.
- On invalid reload, restore the original bytes and reload the prior entry path.
- Revert requires both `expectedSnapshotID == receipt.publishedSnapshotID` and disk bytes equal `receipt.after`.
- Return wrapped sentinel errors so callers can use `errors.Is`.

- [ ] **Step 7: Verify GREEN and race safety**

Run:

```sh
go test ./internal/authoring
go test -race ./internal/authoring
```

Expected: PASS with zero races.

### Task 4: Migrate every source-ledger write caller

**Files:**
- Modify: `internal/web/server.go`
- Modify: `internal/web/server_fava.go`
- Modify: `internal/web/quickentry_handlers.go`
- Modify: `internal/web/quickentry_previews.go`
- Modify: `internal/web/import_previews.go` only if proposal storage types require it
- Modify: `internal/web/favaadapter/registry.go`
- Modify: `internal/web/server_test.go`
- Modify: `internal/web/quickentry_test.go`
- Modify: `internal/web/coverage_test.go`
- Modify: `internal/web/server_helpers_test.go`

- [ ] **Step 1: Add failing Web conflict tests**

Add tests proving existing endpoints do not overwrite disk drift:

```go
func TestEditorSaveRejectsSourceDrift(t *testing.T)
func TestFavaAddEntriesRejectsSourceDrift(t *testing.T)
func TestQuickUndoUsesConditionalReceipt(t *testing.T)
```

Each test builds a valid current snapshot, changes the target file directly without reloading the store, submits the endpoint request, expects HTTP 409, and asserts the external bytes remain untouched.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/web -run 'SourceDrift|ConditionalReceipt'`

Expected: at least the drift tests fail because the current handler overwrites or uses the old custom undo path.

- [ ] **Step 3: Inject one authoring Writer**

Add to `Server`:

```go
authoring *authoring.Writer
quickMu   sync.Mutex
```

Remove `writeMu`. In `NewServer`, construct `authoring.NewWriter(config.Store)` and fail if construction fails.

- [ ] **Step 4: Migrate Editor Replace**

Replace `replaceGraphFile` with:

```go
change, err := authoring.Replace(request.Path, []byte(request.Content))
result, err := s.authoring.Publish(current.ID, change)
```

Map `ErrSnapshotChanged`, `ErrSourceChanged`, and `ErrTargetMissing` to 409/404 as appropriate; map `ErrValidation` to 422; map I/O failures to 500. Preserve response fields.

- [ ] **Step 5: Migrate append workflows**

- Add Entry decodes `[]authoring.Entry`, calls `authoring.SerializeEntries`, creates `authoring.Append`, and publishes.
- Quick Entry preview stores `[]authoring.Entry`; commit serializes, creates Append, publishes, and stores the returned receipt under `quickMu`.
- Import commit creates Append from normalized imported source.
- Quick Profile save creates Append from its canonical custom directive.

No caller concatenates current file bytes.

- [ ] **Step 6: Migrate Quick Entry undo**

Read and clear the latest receipt under `quickMu` only after successful revert. Call:

```go
result, err := s.authoring.Revert(current.ID, receipt)
```

Remove suffix matching, direct `os.ReadFile`, backup creation, `atomicWrite`, and direct store reload from the handler.

- [ ] **Step 7: Delete duplicate write implementation**

Delete `replaceGraphFile` and `atomicWrite` from `internal/web/server.go`. Move their behavioral tests to `internal/authoring/writer_test.go`; no Web test references private filesystem helpers.

- [ ] **Step 8: Update adapter registry ownership text**

Change the Add Entries owner to `internal/authoring.SerializeEntries + internal/authoring.Writer`; keep the route private and authority unchanged.

- [ ] **Step 9: Verify migrated behavior**

Run:

```sh
go test ./internal/web ./internal/quickentry ./internal/authoring
go test -race ./internal/web ./internal/authoring
```

Expected: PASS; new drift tests return 409 and preserve bytes.

### Task 5: Improve Web file locality inside package `web`

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/reports_http.go`
- Create: `internal/web/editor_http.go`
- Create: `internal/web/documents_http.go`
- Create: `internal/web/imports_http.go`
- Create: `internal/web/options_help_http.go`
- Create: `internal/web/assets_http.go`
- Create: `internal/web/http_helpers.go`

- [ ] **Step 1: Record the package behavioral baseline**

Run: `go test ./internal/web`

Expected: PASS.

- [ ] **Step 2: Move report/query code without editing behavior**

Move `handleReport` through report/date-filter helpers, `handleQuery`, and `redactQueryPaths` into `reports_http.go`. Preserve declarations byte-for-byte except imports and gofmt.

- [ ] **Step 3: Verify report/query behavior**

Run: `go test ./internal/web -run 'Report|Query|Filter|Date|Holdings'`

Expected: PASS.

- [ ] **Step 4: Move editor/source code**

Move `handleSource`, editor request types, editor handlers, and `graphFile` into `editor_http.go`.

Run: `go test ./internal/web -run 'Source|Editor'`

Expected: PASS.

- [ ] **Step 5: Move attachment code**

Move upload/move handlers and path-containment helpers into `documents_http.go`.

Run: `go test ./internal/web -run 'Document|Upload|Move|Path'`

Expected: PASS.

- [ ] **Step 6: Move import code**

Move import handlers, preview/commit types, CSV conversion, directive rejection, and import row projection into `imports_http.go`.

Run: `go test ./internal/web -run 'Import|CSV'`

Expected: PASS.

- [ ] **Step 7: Move remaining focused groups**

- Options/help handlers to `options_help_http.go`.
- App/style/document asset handlers to `assets_http.go`.
- Same-origin, JSON decode/encode, API errors, locale, and diagnostic response helpers to `http_helpers.go`.

Keep `server.go` limited to embedded assets, Config, Server state, construction, route registration, listener lifecycle, status, and diagnostics.

- [ ] **Step 8: Verify package locality refactor**

Run:

```sh
gofmt -w internal/web/*.go
go test ./internal/web
go vet ./internal/web
```

Expected: PASS. `server.go` contains no import/CSV/document/editor implementation.

### Task 6: Improve CLI file locality inside package `main`

**Files:**
- Modify: `cmd/orangecount/main.go`
- Create: `cmd/orangecount/check.go`
- Create: `cmd/orangecount/query.go`
- Create: `cmd/orangecount/serve.go`
- Create: `cmd/orangecount/help.go`
- Create: `cmd/orangecount/convert.go`
- Create: `cmd/orangecount/port.go`

- [ ] **Step 1: Record CLI behavior**

Run: `go test ./cmd/orangecount`

Expected: PASS.

- [ ] **Step 2: Move command implementations by responsibility**

- `check.go`: `runCheck`, human/JSON diagnostic rendering helpers.
- `query.go`: query flags, validation, CSV/JSON output.
- `serve.go`: serve flags, initial snapshot, watch loop, server startup.
- `help.go`: help argument normalization and guide rendering.
- `convert.go`: export/dialectize and converted-file writes.
- `port.go`: port owner inspection, termination, parsing, and wait helpers.

Leave `main.go` with version/default constants, `main`, `run`, `runWithInput`, top-level usage, and shared tiny helpers only when used by multiple commands.

- [ ] **Step 3: Verify each moved command through existing tests**

Run:

```sh
gofmt -w cmd/orangecount/*.go
go test ./cmd/orangecount
go vet ./cmd/orangecount
```

Expected: PASS with unchanged output assertions.

### Task 7: Align domain and status documentation

**Files:**
- Modify: `CONTEXT.md`
- Modify: `README.md`
- Modify: `docs/adr/0045-ship-a-beancount-v3-superset-dialect.md`
- Modify: `CHANGELOG.md` only if its existing format records unreleased internal refactors

- [ ] **Step 1: Group the glossary without implementation detail**

Add headings under `## Language`:

```md
### Ledger and compatibility
### Workbench and Fava parity
### Authoring and Quick Entry
### Diagnostics and repair guidance
### Dialect language
```

Move existing term blocks intact to the correct cluster. Shorten terms that currently specify package names, library choices, phase numbers, or temporary migration mechanics; retain those facts in their ADRs.

- [ ] **Step 2: Correct README behavior claims**

- Replace “embedded read-only UI” with wording that distinguishes read-only analysis surfaces from explicit reviewed authoring workflows.
- Replace “runtime ... never edits them” with “never edits without an explicit reviewed write action; failed writes restore the prior source and snapshot.”
- Remove branch-only wording for the dialect and describe it as available on current `main`.

- [ ] **Step 3: Update ADR-0045 status, not history**

Add concise status frontmatter or a first paragraph sentence stating the experiment graduated to `main` on or before 2026-08-20. Keep considered options, original rationale, and consequences unchanged.

- [ ] **Step 4: Validate docs for contradictions**

Search for:

```text
feature/dialect-superset
main keeps the strict v3 contract
never edits them
embedded read-only UI
web-only mutation
```

Expected: no stale claims remain outside explicitly historical quoted context.

### Task 8: Update structural indexes

**Files:**
- Create or modify only index files required by the installed project multi-level index workflow.

- [ ] **Step 1: Load index language and workflow configuration**

Use Simplified Chinese fallback because `.claude/locale-config.json` is absent. Read the installed update workflow and adapter instructions before writing any index file.

- [ ] **Step 2: Run incremental index update for structural changes**

Supply the final created/moved files under `internal/authoring`, `internal/web`, and `cmd/orangecount`. Do not index generated assets, coverage files, or `node_modules`.

- [ ] **Step 3: Run index consistency check**

Expected: every created source file is represented once, links resolve, and no index points to removed declarations.

### Task 9: Full verification and review

**Files:**
- Inspect all changed files; modify only to fix verified failures or review findings.

- [ ] **Step 1: Run focused authoring behavior**

```sh
go test ./internal/authoring ./internal/quickentry ./internal/web

go test -race ./internal/authoring ./internal/web
```

Expected: PASS, zero races.

- [ ] **Step 2: Run full Go gates**

```sh
make fmt
make vet
make test
make race
make license
make build
```

Expected: every target passes.

- [ ] **Step 3: Compare coverage**

```sh
go test -coverprofile=/tmp/orangecount-structural-cover.out ./...
go tool cover -func=/tmp/orangecount-structural-cover.out
```

Acceptance: total statement coverage does not regress below the 90.5% baseline; every new authoring error/rollback branch has direct behavioral coverage.

- [ ] **Step 4: Run full frontend gates**

```sh
npm --prefix web test
npm --prefix web run typecheck
npm --prefix web run build:check
npm --prefix web run check:phase0
```

Expected: 34/34 or more unit tests pass, typecheck passes, deterministic build reports four staging files, route/provenance checks pass, and no module-type warnings appear.

- [ ] **Step 5: Smoke the real binary and Web write path**

- Build `bin/orangecount`.
- Create a temporary valid ledger outside the repository.
- Launch `orangecount serve --addr 127.0.0.1:0` through the process hub.
- Browser-drive Quick Entry preview, publish, and undo.
- Verify the published transaction appears after commit, disappears after undo, and the ledger remains valid.

- [ ] **Step 6: Inspect scope and request review**

Check that no generated frontend asset, fixture, private ledger data, unrelated dependency, public route, or accounting result changed. Dispatch a reviewer against the final working diff and this plan. Fix all Critical and Important findings, then rerun the affected focused and full gates.
