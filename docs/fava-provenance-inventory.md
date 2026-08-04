<!--
Copyright 2026 OrangeCount contributors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
-->

# Fava provenance inventory

Every Fava-derived file or asset inside this repository must be recorded in
this inventory **before** it is added, per ADR-0030 and the Fava frontend
transplant plan ("a derived file cannot be added without inventory evidence").
Original OrangeCount files never appear here. This is a template; entries are filled in by the owning implementation-wave
agents when they actually import or adapt Fava units.
Until a row exists for a file, the file must not be imported.

## Attribution baseline

- Upstream: Fava `v1.30.12`, commit
  `aa7538e8971252c9efc52c8a516a3a77d604553f` (see
  `docs/fava-reference-lock.md`).
- Upstream license: MIT, `Copyright (c) 2015-2016 Dominik Aumayr
  <dominik@aumayr.name>`. Every derived file keeps the upstream copyright and
  MIT notice and adds the OrangeCount Apache-2.0 notice below it.
- The repository `NOTICE` file is updated to list "Fava (MIT)" as a
  third-party component with its copyright line and source URL.

## Decision vocabulary

| Decision | Meaning |
| --- | --- |
| `copied` | File imported unchanged from the pinned Fava commit (still needs a row). |
| `adapted` | File imported and modified for the Go adapter / OrangeCount boundary. |
| `rewritten` | File replaced by an independent OrangeCount implementation with equivalent behavior. It needs no derivative-license notice when no upstream code is reused, but still requires a traceability row and justification. |
| `excluded` | File intentionally not adopted; record the reason. |

## Row template (one row per derived file)

```markdown
| Field | Value |
| --- | --- |
| Upstream path | `frontend/src/...` (at the pinned commit) |
| Upstream revision | `aa7538e8971252c9efc52c8a516a3a77d604553f` |
| OrangeCount path | `web/...` or `internal/web/assets/...` |
| Decision | `copied` / `adapted` / `rewritten` / `excluded` |
| Local modifications | exact list of behavioral changes vs upstream |
| Copyright holder | Dominik Aumayr (Fava) + OrangeCount contributors |
| MIT notice placement | header of the file and `NOTICE` section |
| Dependency licenses | resolved from `frontend/package-lock.json` at the pinned commit; record the specific dep versions adopted |
| Contract rows | link to `docs/fava-contract-map.md` rows that this file consumes |
| Notes | anything a reviewer must verify before merge |
```

## Current inventory

The completed P2 phase-1 attempt used the pinned Fava source only as a behavior
and dependency reference. It copied no Fava implementation code, CSS, fonts,
icons, or runtime assets. The shell units below are therefore clean-room
prototype rewrites, not accepted implementation of the Fava frontend
transplant or rendering-fidelity goal. Wave 1 replaces or relocates them and
adds selected upstream-derived units with full provenance before a route may
cut over.

The rows remain as an honest inventory of the current tree. A `rewritten` row
never grants visual acceptance and may not be used to avoid an `adapted`
source unit selected by the authoritative transplant plan.

| Upstream path | OrangeCount path | Decision | Local modifications | MIT notice location | Dependency license result |
| --- | --- | --- | --- | --- | --- |
| `frontend/package.json`, `frontend/package-lock.json` | `web/package.json`, `web/package-lock.json` | rewritten | Selected `svelte@5.11.3`, `esbuild@0.27.0`, `esbuild-svelte@0.9.0`, and `typescript@5.6.2`; added Playwright tooling from P1; no Fava runtime dependency | Not applicable: new manifest and lock, no upstream source copied | Direct build dependencies are from the approved Fava dependency families; resolved licenses remain in the npm lock |
| `frontend/build.ts` | `web/build.mjs`, `web/scripts/check-build.mjs` | rewritten | Independent esbuild build into `web/staging/fava-shell`; no Fava output path, runtime asset, or CDN; deterministic hash check added | Not applicable: independent build scripts | esbuild/esbuild-svelte are development-only |
| `frontend/src/app.ts`, `frontend/src/router.ts`, `frontend/src/helpers.ts` | `web/src/fava/main.ts`, `web/src/fava/router.mjs`, `web/src/fava/adapter-client.ts` | rewritten | New mount, route parser, URL serializer, and narrow private OrangeCount adapter boundary; Fava Python API is not used | Not applicable: no upstream implementation copied | Svelte compiler only; no runtime network dependency |
| `frontend/src/sidebar/Header.svelte`, `AsideContents.svelte`, `PageTitle.svelte`, `sidebar/index.ts` | `web/src/fava/components/Header.svelte`, `Sidebar.svelte`, `PageTitle.svelte` | rewritten | Independent shell components; removed multi-ledger, extensions, and Fava backend assumptions; retained accessible navigation and responsive menu behavior | Not applicable: clean-room rewrite | Svelte only |
| `frontend/src/stores/url.ts`, `stores/color_scheme.ts`, `stores/options.ts`, `stores/filters.ts` | `web/src/fava/state.mjs` | rewritten | Small route/locale/theme/loading/error state reducer; supported locale/theme values only | Not applicable: clean-room rewrite | Svelte store is development/build dependency |
| `frontend/css/base.css`, `layout.css`, `style.css`, `components.css` | `web/src/fava/styles/shell.css` | rewritten | New OrangeCount shell tokens and responsive styles; no Fava CSS or font asset copied | Not applicable: clean-room stylesheet | No font or remote asset dependency |
| `frontend/src/reports/route.svelte.ts`, `ReportLoadError.svelte` | `web/src/fava/components/LoadingBoundary.svelte`, `ErrorBoundary.svelte` | rewritten | Common shell-only loading/error states targeting the future private adapter | Not applicable: clean-room rewrite | Svelte only |

## Workflow

1. `adapted`/`copied` files require a filled row above and an updated `NOTICE`
   entry in the same change.
2. `rewritten` files do not reuse upstream code; they still require a row so
   the exclusion/adoption matrix stays complete.
3. Prerequisite Phase 0 must extend `make license` and add a provenance guard that fails when
   a selected, adapted, or Fava-influenced unit under `web/` or
   `internal/web/assets/` has no traceability row, required notice, upstream
   hash, or contract mapping. The current `make license` does not yet enforce
   this rule.
