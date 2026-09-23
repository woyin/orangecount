# OrangeCount Structural Optimization Design

## Outcome

Restructure OrangeCount around a UI-independent reviewed-write module, then improve file locality and quality gates without changing accounting semantics, CLI output, private Fava-shaped HTTP payloads, source-ledger formats, or frontend behavior.

## Evidence

- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.
- Go statement coverage is 90.5%.
- `make fmt` fails because `internal/report/pivot.go` is not gofmt-clean.
- Frontend unit tests pass, but Node reparses TypeScript as ES modules because `web/package.json` lacks `"type": "module"`.
- `npm exec -- tsc --noEmit` fails because `web/tsconfig.json` requires Node types while `@types/node` is absent.
- `internal/quickentry` imports `internal/web/favaadapter` for `NewEntry`, `NewPosting`, and serialization. A domain compiler therefore depends on a Web adapter model.
- `internal/web/server.go` owns HTTP routing, report projection, editor mutation, atomic file replacement, attachment operations, import conversion, options, help, and assets in 1,868 lines.
- Quick Entry commit uses the shared replacement path, but Quick Entry undo reimplements backup, write, rebuild, and rollback in an HTTP handler.
- README claims the runtime never edits ledgers and that the dialect is branch-only, while `main` contains reviewed authoring workflows and dialect commands.

## Decisions

1. The first structural seam is source-ledger authoring, not the parser/evaluator/query core.
2. The canonical term is **Reviewed write workflow**: a UI-independent, atomic, recoverable, full-graph-revalidated source-ledger publication.
3. A **Ledger proposal** is domain data, not an HTTP request. It has no accounting effect before publication.
4. The first implementation supports one target source file per publication. Every publication still validates the complete include graph.
5. The authoring module owns conditional rollback. The Web session only owns the product policy that at most the latest Quick Entry publication is undoable.
6. The authoring module depends directly on `*snapshot.Store` and the standard filesystem. No hypothetical filesystem or repository interfaces are introduced.
7. Existing behavior and transport contracts are preserved through a clean cutover. No compatibility aliases remain after all callers migrate.
8. Large core files are not split merely by line count. Their interfaces are deep and their responsibilities remain cohesive. Reconsider only when a concrete change crosses multiple concerns or measured complexity rises.

## Target architecture

```text
source ─→ ledger ─→ snapshot ─→ query/report
                    ↑
               authoring
              ↑         ↑
        quickentry   web/favaadapter
                         ↑
                        web

cmd/orangecount coordinates snapshot, query, authoring, and web.
```

### `internal/authoring`

A deep module with two capabilities.

#### Ledger proposal model

- `Entry` and `Posting` represent proposed transaction, balance, and note directives.
- JSON tags remain on the types because the private Fava-shaped adapter decodes directly into them, but HTTP status, route, and envelope concerns never enter the module.
- `SerializeEntries` is the sole serializer for these proposal types.
- `Change` is an immutable single-target operation created by `Append(target, block)` or `Replace(target, content)`.
- Append formatting is owned by the module, so callers cannot disagree about blank lines or stale source bytes.

#### Reviewed publication

- `Writer` owns the write mutex and a concrete `*snapshot.Store`.
- `Publish(expectedSnapshotID, Change)`:
  1. locks the complete transaction;
  2. loads the current published snapshot and verifies the expected ID;
  3. resolves the visible target within the current include graph;
  4. verifies on-disk bytes still equal the snapshot's bytes;
  5. builds replacement bytes from the current file and the `Change`;
  6. writes a recoverable backup;
  7. atomically replaces the file;
  8. reloads and validates the complete include graph;
  9. restores the old bytes and prior valid snapshot on failure;
  10. returns a receipt only after successful publication.
- `Revert(expectedSnapshotID, Receipt)` uses the same lock and protocol. It succeeds only when the current snapshot and target bytes still match the publication represented by the receipt.
- Domain sentinel errors distinguish snapshot conflict, source-file drift, unavailable target, invalid proposal, validation failure, and I/O failure. Web maps these errors to existing HTTP status classes.

### Web migration

- `Server` receives one `*authoring.Writer` created from its snapshot store.
- Add Entry, Quick Entry, Editor, Import, and Quick Profile create `authoring.Change` values and call `Publish`.
- Quick Entry stores the latest receipt under a small Web-owned mutex and calls `Revert` for undo.
- Attachment upload/move and local display options remain in `internal/web`; they are not source-ledger publication.
- `favaadapter` retains projection and private transport concerns only.

### File locality

After the clean cutover, split existing Go package files without adding package seams:

- `internal/web/server.go` keeps server lifecycle and route registration.
- Report/query HTTP parsing moves to `reports_http.go`.
- Editor and source handlers move to `editor_http.go`.
- Attachment handlers move to `documents_http.go`.
- Import handlers and CSV conversion move to `imports_http.go`.
- Options/help/assets and shared HTTP helpers move to focused files.
- `cmd/orangecount/main.go` keeps `main`, top-level dispatch, version, and usage.
- Check, query, serve, conversion, help, and port-conflict code move to command-focused files in the same `main` package.

These are locality refactors, not new interfaces. Existing package tests remain the behavioral surface.

### Frontend quality gate

- Add `"type": "module"` to `web/package.json`.
- Add a pinned Node 22-compatible `@types/node` development dependency.
- Add `npm run typecheck` and include it in `make web-check`.
- Do not split Svelte modules until type checking is green and a specific component change demonstrates mixed responsibilities. Provenance-managed Fava-derived files are not mechanically churned.

## Data flow

### Add Entry / Quick Entry / Import / Quick Profile

```text
transport or compiler input
  → domain validation
  → authoring.Entry or canonical directive block
  → authoring.Append(target, block)
  → Writer.Publish(expected snapshot)
  → complete graph reload
  → new immutable snapshot or full rollback
```

### Editor

```text
editor content
  → authoring.Replace(target, full content)
  → Writer.Publish(expected snapshot)
  → complete graph reload
  → new immutable snapshot or full rollback
```

### Quick Entry undo

```text
latest session receipt
  → Writer.Revert(current snapshot, receipt)
  → complete graph reload
  → prior file bytes restored or no change
```

## Error and concurrency invariants

- A stale expected snapshot never writes.
- On-disk drift from the expected snapshot never writes.
- A target outside the current include graph never writes.
- A validation failure restores original bytes and keeps the previous valid snapshot published.
- A failed rollback restores the post-publication bytes and keeps the post-publication snapshot published.
- Two concurrent publish/revert operations serialize through one writer lock.
- A receipt cannot revert a later unrelated publication.
- HTTP response schemas and diagnostic payloads remain unchanged.

## Testing strategy

TDD applies to every new authoring behavior.

1. Proposal serialization tests move from `favaadapter` and first fail against the absent authoring interface.
2. Writer tests use real temporary files and a real `snapshot.Store`:
   - successful append;
   - successful full replacement;
   - stale snapshot conflict;
   - external disk drift conflict;
   - invalid target rejection;
   - validation rollback;
   - conditional revert success;
   - revert rejected after a later publication;
   - concurrent publication serialization.
3. Existing Web tests characterize unchanged JSON/status behavior while callers migrate.
4. Existing CLI tests protect output and exit codes during file moves.
5. Frontend unit tests, type checking, deterministic build, route manifest, and provenance checks are required.
6. Final gates: gofmt, vet, Go tests, race tests, coverage comparison, frontend tests, typecheck, static gates, deterministic build, binary build, and a browser-driven Quick Entry publish/undo smoke test.

## Documentation and domain model

- Keep `CONTEXT.md` implementation-free and group terms by Ledger, Authoring, Workbench, Repair Guidance, and Dialect language clusters.
- Remove stale migration-state wording from glossary terms; implementation decisions stay in ADRs.
- Correct README claims about write behavior and dialect availability.
- Update ADR-0045 status wording to reflect that the dialect is now present on `main`; do not rewrite its historical trade-off record.
- Add or update project indexes only after structural file changes are final.

## Non-goals

- No accounting semantic changes.
- No public HTTP API commitment.
- No persistent database, remote service, or new runtime dependency.
- No multi-file atomic transaction protocol.
- No parser/evaluator/query rewrite.
- No filesystem abstraction with a single implementation.
- No broad Svelte redesign or visual change.
