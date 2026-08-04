# Sanitized browser and visual regression harness

This is a development-only Playwright harness. It does not participate in Go
asset generation, `make build`, or the embedded runtime. Every run starts a
fresh OrangeCount process using only `testdata/fixtures/fava-visual/` and its
explicit document root.

## Tooling and installation

The harness is isolated under `web/`:

- `web/package.json` and `web/package-lock.json` pin
  `@playwright/test` to `1.52.0` and restrict the supported Node range to
  Node 20 or Node 22.
- `web/playwright.config.mjs` defines deterministic desktop and narrow
  Chromium projects, English locale, UTC, reduced motion, one worker, an
  offline request guard, and sanitized snapshot locations.
- `web/scripts/run-visual.mjs` starts OrangeCount on a fresh ephemeral
  loopback port, passes only the synthetic fixture, runs Playwright, and
  terminates the process group.
- `web/tests/visual/` contains high-impact behavior and screenshot tests.

Install exactly once in the development environment:

```sh
npm --prefix web ci
npm exec --prefix web playwright install chromium
```

The browser download is a development prerequisite only. Node and Chromium
are never required by Go release builds.

## Commands

Run the harness with its temporary server and both viewport projects:

```sh
npm --prefix web run visual:test
```

Regenerate sanitized baselines only after visual review:

```sh
npm --prefix web run visual:update
```

The runner uses `go run ./cmd/orangecount serve` by default. A prebuilt
binary can be supplied without changing the release build:

```sh
ORANGECOUNT_BIN=/path/to/orangecount npm --prefix web run visual:test
```

Baselines are stored only under `testdata/visual-baselines/`. Temporary test
results and browser state use a uniquely named system temporary directory and
are not committed.

## Current blocker

The approved installation was attempted with Node `24.14.1`. Playwright
`1.52.0` hung before test discovery even with an empty isolated configuration.
The harness now fails fast with exit status `78` and an explicit message from
both the runner and config; it does not start a server first. Use Node 22 LTS
for the commands above without changing the system Node. No alternate Node
version was installed or probed in this task.

The Chromium binary download completed, but browser tests were not executed
under the blocked Node runtime and no screenshot baselines were generated.

## Coverage

`web/tests/visual/fava-high-impact.spec.mjs` covers OrangeCount against the
sanitized fixture for:

- shell, active navigation, global controls, and narrow responsive menu;
- grouped Journal rows, posting collapse, date filters, and URL state;
- balance sheet and trial balance account trees and hierarchy chart controls;
- account detail, account-scoped journal, and running balance.

`web/tests/visual/fava-reference.spec.mjs` is an explicit, opt-in reference
suite for a newly started Fava 1.30.12 process. It is skipped unless
`FAVA_BASE_URL` is supplied, and refuses the known private-server port and
private-ledger marker. The reference process must be started separately with
the same synthetic fixture; no existing local session is accepted.

Starting the external Fava reference was attempted with the available pinned
source environment. It could not build its Beancount dependency because the
host only provides an older Bison, so the reference suite remains skipped.
No Fava server was started or contacted.

## Privacy and determinism rules

- The only ledger input is `testdata/fixtures/fava-visual/main.bean`.
- The only attachment root is that fixture's `documents/` directory.
- The browser uses a fresh context, fixed desktop/narrow viewports, CSS font
  selection, UTC, English, reduced motion, disabled animation/caret, and one
  worker.
- Requests to any origin other than the temporary test server fail.
- No trace, video, raw DOM, network response, private path, or attachment
  content is retained.
- Reference screenshots, when eventually enabled, must be generated only from
  the same synthetic fixture and reviewed before commit.
- Existing local servers, browser profiles, sessions, screenshots, and ledger
  processes are never inspected or reused.

The existing offline checks remain separate:

```sh
go test ./internal/compat ./internal/source ./internal/web
node internal/web/assets/journal_sorting_test.mjs
node internal/web/assets/chart_helpers_test.mjs
```
