<!--
Copyright 2026 OrangeCount contributors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
-->

# OrangeCount

OrangeCount is an offline, Go-native reader and validator for compatible
Beancount v3 ledgers. The source ledger remains a user-owned `.bean` include
graph; OrangeCount builds an immutable snapshot and exposes read-only reports,
queries, diagnostics, and a local web workbench.

## v0.1 scope

The v0.1 implementation covers the v3 core syntax and accounting needed for a
personal ledger, including:

- includes, options, metadata, tags and links, amounts, exact decimal
  arithmetic, transactions and postings;
- account lifecycle, currencies, balance assertions and tolerances, pads,
  prices, costs/lots, and deterministic inventory booking; and
- current query workbench behavior, deterministic account/journal/trial
  balance, balance sheet, income statement, holdings, price, event, document,
  and diagnostic reports.

The embedded read-only UI provides overview, account, journal, report,
holdings, price, document, source, diagnostics, and query views. English (`en`)
and Simplified Chinese (`zh-CN`) are shipped locales; changing the display
locale does not change ledger semantics.

## Deliberate boundaries

OrangeCount is not a drop-in replacement for every Beancount or Fava feature.
In particular, v0.1 does not promise:

- Python plugin execution or the Fava plugin ecosystem, HTTP API, or pixel
  parity;
- bank/broker importers, a budget model, a persistent database, or an editor
  that writes ledger files; or
- behavior for v2-only syntax or extensions outside the supported v3 core.

Plugin declarations are preserved and reported as migration diagnostics; no
plugin code is executed. Attachments are served only from explicitly supplied,
normalized document roots.

## Requirements and build

Building requires Go 1.22 or newer and `make`. The binary has no Python, Node,
database, or network service dependency at runtime; the checked-in browser
assets are embedded in the Go binary.

```sh
make fmt       # require gofmt-clean source
make vet       # static analysis
make test      # unit and package tests
make race      # race detector tests
make license   # Apache/notice and dependency policy checks
make build     # bin/orangecount

# Release-style local gate:
make fmt vet test race license build
```

## CLI

Build first with `make build`, then point commands at an entry ledger:

```sh
./bin/orangecount check --locale en ledger/main.bean
./bin/orangecount check --locale zh-CN --json ledger/main.bean

./bin/orangecount query --format json ledger/main.bean \
  "SELECT account, sum(number) AS total FROM postings \
   WHERE currency = 'USD' GROUP BY account ORDER BY total DESC"

./bin/orangecount serve --locale en --addr 127.0.0.1:0 ledger/main.bean
```

`check` returns a non-zero status for an invalid snapshot. `query` also
supports `--format csv` (or `--csv`). `serve` is loopback-only: the default
`127.0.0.1:0` asks the operating system for a free port, and startup prints
the actual URL, for example `http://127.0.0.1:54321`. Non-loopback addresses
are rejected. Stop the local session with Ctrl-C.

The web UI has an English/Simplified Chinese selector (or `?locale=zh-CN`),
and the CLI accepts `--locale en|zh-CN` for rendered diagnostics.

## Privacy and offline behavior

The runtime reads source files locally, never edits them, makes no outbound
requests, loads no remote scripts, and binds the web server only to loopback.
Diagnostics and structured logs redact sensitive fields by default; the
`serve --sensitive-logs` flag is an explicit local debugging choice. A failed
reload leaves the last valid snapshot available.

The optional development differential harness under `tools/reference/` uses
`uv` and Beancount v3 only as an external oracle. Install `uv`, then set
`ORANGECOUNT_PRIVATE_LEDGER` to an absolute ledger path outside this checkout:

```sh
ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean \
  ./tools/reference/differential.sh
```

The harness is never a runtime dependency, refuses repository paths, keeps
intermediate files temporary, and emits only normalized counts and diagnostic
classes—not ledger text, paths, accounts, or amounts. Known intentional
boundaries are tracked in `docs/compatibility-ledger.json`.

## License and clean-room statement

OrangeCount is released under the Apache License 2.0; see [LICENSE](LICENSE)
and [NOTICE](NOTICE). It is an independent clean-room Go implementation of
the public Beancount v3 language and accounting behavior. Beancount is a
separate project and is used only as an optional development-time reference;
its source is not copied, linked, or bundled into OrangeCount.
