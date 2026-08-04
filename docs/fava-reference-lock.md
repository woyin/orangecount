<!--
Copyright 2026 OrangeCount contributors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
-->

# Fava 1.30.12 reference lock

This record pins the Fava upstream source used as the parity reference for the
frontend transplant (ADR-0031, ADR-0036). The full checkout lives outside this
repository in a read-only location; this repository never vendors the Fava
application, tests, or dependency tree.

## Pinned reference

| Field | Value |
| --- | --- |
| Release | Fava 1.30.12 |
| Tag | `v1.30.12` (annotated tag) |
| Tag object | `af81e32a1fcfced0fe04c77373c41df5d31d78e4` |
| Tagged commit (peeled) | `aa7538e8971252c9efc52c8a516a3a77d604553f` |
| Commit subject | `deps & lint` |
| Committer | Jakob Schnitzer `<mail@jakobschnitzer.de>` |
| Commit date | `2026-02-17T20:24:11+01:00` (author `2026-02-17T20:08:34+01:00`) |
| Upstream URL | `https://github.com/beancount/fava.git` |
| Tag source | `https://github.com/beancount/fava/tree/v1.30.12` |
| Retrieval date (UTC) | `2026-08-03` |
| Mirror checkout (external) | `$HOME/.orangecount/fava-1.30.12-reference` |

Verification performed at retrieval: `git ls-remote` returned
`aa7538e8971252c9efc52c8a516a3a77d604553f` as the peeled target of
`refs/tags/v1.30.12`; a fresh `git clone --single-branch --branch v1.30.12`
was detached at exactly that commit; `git rev-parse HEAD` equals the peeled
commit. The mirror `.git` directory is made read-only after clone.

## License hashes (at the pinned commit)

| File | SHA-256 |
| --- | --- |
| `LICENSE` (Fava MIT license text) | `143ab9400e5ecf3fe1ec07725e1781ec60c24fa2765a6bd22c6a0eda6770d567` |
| `frontend/package-lock.json` (frontend runtime + dev dependency lock) | `99ce58b73d05f7a73ad7a451ba2b93d42e6219cc82b127d373d1171ae3036be5` |
| `frontend/package.json` (frontend manifest) | `c01de9f28b6045cf582fc816bd1965e27f4e10d8a4450ca39a28c86ace277418` |
| `uv.lock` (Python dependency lock) | `d877199bbabc06685ad83c42700088ef88ee5029a194160d9e3d115f53fd9711` |
| `pyproject.toml` (Python package metadata) | `491eba314ad0735140d4f574ec66e56087a97f588a86444e996f99b42ecd0bac` |
| `frontend/src/codemirror/tree-sitter-beancount.wasm` | `4dd6d7c3b3ce760c870de0224227eaf26b0d2b72d3ef8181b31320b035dcce45` |

## Attribution facts (Fava is MIT)

- `LICENSE` is The MIT License (MIT), `Copyright (c) 2015-2016 Dominik Aumayr
  <dominik@aumayr.name>`, 1108 bytes.
- `AUTHORS` additionally records Jakob Schnitzer as maintainer and Martin Blais
  as the author of Beancount. Derived files must retain the MIT notice and the
  copyright line; the third-party notice inventory (see
  `docs/fava-provenance-inventory.md`) records each imported unit.
- The Python package declares `license = "MIT"` in `pyproject.toml`.

## Dependency license posture (summary; full inventory in source-inventory doc)

Frontend `package-lock.json` (npm, `lockfileVersion 3`) contains 372 direct and
transitive packages; license fields observed: MIT 294, ISC 31, Apache-2.0 16,
BSD-2-Clause 15, BSD-3-Clause 5, BlueOak-1.0.0 4, MIT-0 4, OFL-1.1 3,
CC0-1.0 1, Python-2.0 1, and one package with no license field
(`svg-tags@1.0.0`, dev-only transitive dependency of stylelint). Each adopted
frontend unit inherits its direct dependencies (Svelte, d3-*, @codemirror/*,
@lezer/*, @fontsource/*, @ungap/custom-elements, web-tree-sitter); all are
MIT/ISC/Apache-2.0 with no copyleft field observed. The Python lock is
reference-only (Fava backend is not transplanted).

## Controlled visual-environment lock (Prerequisite Phase 0)

The first complete candidate capture recorded the following immutable
execution values in `testdata/visual-candidates/fava-reference/environment-lock.json`:

| Field | Captured value |
| --- | --- |
| OCI image ID | `sha256:02c702c12363ce300e7f8ae2c2392edf9fd55bda3e908e07592f1229fb72e7eb` |
| Python | `3.12.8` |
| Beancount | `3.2.3` |
| Bison | `3.8.2` |
| Node / npm | `22.18.0` / `10.9.3` |
| Playwright | `1.52.0` |
| Chromium | `151.0.7922.71` (Debian bookworm package) |
| Fonts | Fava-pinned Fira Sans/Fira Mono/Source Code Pro; Debian `fonts-noto-cjk` fallback |
| Locale / timezone | `en-US` / `UTC` |
| Browser settings | reduced motion; device scale factor `1`; desktop `1280x800`; narrow `520x800` |
| Fixture content SHA-256 | `522ebabb292ce6dcebbe20a699c05248a4389f086307f602ea3a8a19b78cdea8` |

The source lock above and the OCI image are development-only authority inputs.
The runner records these values on every capture, and
`check-reference-output.mjs` verifies the image/hash/matrix relationship. The
candidate evidence does not approve or replace `testdata/visual-baselines/`.
The image is defined by `tools/fava-reference/Dockerfile` and is rebuilt when
that file or its copied harness inputs change.

## Reference integrity policy

- Do not fetch, rebase, or modify the mirror; treat it as read-only.
- A later Fava upgrade is a separate project requiring a new inventory and a
  new lock record (ADR-0036).
- Private-ledger observation is transient only and never stored in this
  repository (ADR-0027).
