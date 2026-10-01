# Development checks

For the next implementation cycle, read the
[Go/native design](go-beancount-native-design-2026-09-30.md),
[development plan](go-beancount-native-plan-2026-09-30.md), and
[implementation handbook](go-beancount-native-implementation-2026-09-30.md).
The handbook distinguishes existing commands from proposed flags and checks;
first check the [current progress](go-beancount-native-progress.md), confirm B01
is still applicable, and preserve the current working-tree implementation.

OrangeCount targets Go 1.22 or newer. The module's `go.mod` is the source of
truth for language/toolchain compatibility; release builds use `-trimpath` and
an empty build ID for reproducible output.

The reproducible local checks are:

```sh
make fmt
make vet
make test
make race
make build
```

`tools/reference/` contains an optional `uv` environment for differential
checks against Beancount v3. It is never a runtime dependency. Set
`ORANGECOUNT_PRIVATE_LEDGER` to an absolute path outside this checkout before
running the harness; it rejects repository paths and reports only redacted
summary counts.


## Native macOS development

The native app uses SwiftUI/Charts and a Go C archive, built directly with
`swiftc`; Xcode.app is not required on the verified local Command Line Tools
installation. Swift 6 is required for these build scripts. Core Go language
compatibility remains declared by `go.mod`; native CI pins Go 1.27.1.

- `make mac-test`: real Go → C ABI → Swift decoding and business assertions,
  plus controlled asynchronous `LedgerStore` state and lifetime tests and a
  real-client source-error/recovery flow with isolated temporary fixtures.
- `make mac-check`: the above, an optimized app, ad hoc signature verification,
  and agreement of the final Mach-O and plist deployment targets.
- `make mac-package`: rebuild the app, then produce the development zip and
  checksum under `dist/macos/`.
- `make mac-benchmark`: generate the deterministic 52,500-transaction ledger
  and report open/read/poll times, JSON sizes and Swift decode costs, first
  with mixed cash currencies and then a separate USD-cash variant with
  populated at-cost charts (the same transaction and lot counts).
  Output includes chart availability and point counts, as well as holdings
  groups/events/history, selected lots/history points and consumed-lot pages
  with independent payload and projection/decode samples. Run after checks
  finish, without other builds. To reuse the baseline, run
  `sh apps/macos/benchmark.sh build/macos-baseline/large.bean` after `mac-test`.
- `sh apps/macos/client-probes.sh build/macos-baseline/large.bean 100`: after
  `mac-test`, measure superseded-request cancellation and 100 real-client
  open/read/close cycles, verify closed handles, and sample process RSS.
  This is a bounded command-line experiment, not a GUI memory or longevity test.
- `python3 apps/macos/process-soak.py build/macos-benchmark/usd.bean`: after
  preparing the QA bundle, observe its process for 60 edits at 10-second
  intervals. It edits a disposable copy of the supplied single-file synthetic
  ledger and stores hashes, RSS samples, and a `leaks --noContent` report under
  a unique `build/macos-baseline/app-soak.*` directory. Use `--cycles 1` for a
  short probe smoke check. It terminates its own process after observation;
  this does not exercise UI Quit or assert visible data, completed rebuilds,
  frame latency, or absence of leaks. Compare allocator findings with a minimal
  framework baseline and inspect trends before drawing memory conclusions.
- To measure automatic reload stages, use a single-file synthetic input:
  `go run ./tools/nativeprobe -ledger build/macos-benchmark/usd.bean -trace build/macos-baseline/watched-reloads.trace -cycles 5`.
  The command edits only its temporary copy, requires one new valid snapshot
  per edit, and records `native.edit`, `desktop.watch`, `desktop.build`,
  `desktop.publish`, and `native.observed` execution-trace events. With the
  pinned Go 1.27 toolchain, `go tool trace -d=parsed` prints their timestamps.
  Trace regions carry no ledger paths or content. Report detection, debounce,
  build, publication and observation separately; do not attribute the whole
  file-save delay to the build or to publication.

`apps/macos/toolchain.sh` is shared by app and test builds. Use
`ORANGECOUNT_GO`, `ORANGECOUNT_SWIFTC`, and `DEVELOPER_DIR` to select installed
toolchains explicitly. `ORANGECOUNT_MIN_MACOS` defaults to 13.0;
`ORANGECOUNT_VERSION` defaults to 0.1.0. The generated `build-info.txt` records
the actual compiler/SDK versions and source revision. Changing the deployment
target is a build configuration change, not proof of runtime compatibility.

Use only synthetic fixtures for unattended app checks. For example:

```sh
ORANGECOUNT_TEST_MODE=1 build/OrangeCount.app/Contents/MacOS/OrangeCount \
  testdata/fixtures/native/cashflow.bean
```

Test mode isolates recent-ledger preferences. Omitting the path shows the
welcome screen. Normal launches may restore a valid recent file; neither
mode falls back to a repository fixture. For GUI automation, prepare a separate
bundle after `make app` and launch `build/OrangeCount-QA.app`:

```sh
sh apps/macos/prepare-qa.sh
# Optional presentation overrides, confined to the QA bundle:
ORANGECOUNT_QA_APPEARANCE=dark sh apps/macos/prepare-qa.sh
```

Quit the previous QA process before replacing the bundle. Its separate bundle
identifier and `OrangeCountTestMode` metadata preserve isolation even when a UI
tool relaunches it without environment variables. The QA clone omits file-type
registrations so it does not compete with installed ledger viewers. `light` and `dark` are the
supported appearance overrides. These overrides do not alter system preferences.
Only edit disposable copies of fixtures under `build/macos-qa/`.

Window interaction, source-file navigation, presentation, accessibility, and
target-machine installation are recorded separately from compilation or
process survival in the native verification report and acceptance checklist.
Source navigation respects the default external editor. If `.bean` is associated
with OrangeCount itself, it opens TextEdit instead; the app displays the source
line without promising editor-specific line navigation.

The native CI workflow is separate from Linux CLI release cross-compilation.
No Developer ID signing, notarization or public upload is performed by the
local development packaging command.
