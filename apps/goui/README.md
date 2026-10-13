# OrangeCount Go UI for macOS — development preview

This is the MyGo native Go UI client under active development. It is an unsigned, unnotarized preview and is not the production macOS replacement yet. The current task ledger records remaining GUI, input-method, VoiceOver, performance, parity, installation, signing, and handoff gates in [`docs/mygo-native-desktop-tasks-2026-10-07.md`](../../docs/mygo-native-desktop-tasks-2026-10-07.md).

## Requirements

- Build and run on macOS 13 or later.
- Use the Go version declared in `go.mod` (currently Go 1.27.1).
- Package output supports `arm64` and `amd64`; this does not mean both architectures or all macOS versions have completed release acceptance.

## Build and test

Run these commands from the repository root:

```sh
make goui-test
make goui-build
make goui-check
```

The desktop module is independent from the root Go module. Its scripts set `GOWORK=off` and `CGO_ENABLED=0`; `goui-check` runs the Go UI tests and builds the macOS executable into `apps/goui/build/`.

For opt-in macOS GUI/Accessibility/input-client scenarios against public
fixtures, run `make goui-native-e2e`. This starts an isolated MyGo test app and
opens native windows; it does not open another application or read a private
ledger. To repeat the four navigation, empty-ledger, multi-currency, and IME
cases ten times:

```sh
./apps/goui/native-e2e.sh -run '^TestNative(CashflowOpenAndAccessibleNavigation|EmptyLedgerStatementShowsNoActivity|MultiCurrencyBalancesKeepCurrenciesSeparate|QueryEditorPreservesMarkedComposition)$' -count=10
```

The **View / 显示** menu changes text size for the current ledger: Increase,
Decrease, and Reset (**放大文字 / 缩小文字 / 恢复默认文字大小**). On macOS the
shortcuts are Cmd+Plus, Cmd+Minus, and Cmd+0. Supported sizes are 100%, 125%,
and 150%; the selected size is saved independently per ledger and restores when
that ledger is opened again. The scale composes with the desktop accessibility
text-size preference.

For an opt-in read-only comparison of the current core, Go UI workspace, and page renderers against one external ledger:

```sh
ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean \
ORANGECOUNT_PRIVATE_PROFILE=legacy \
make goui-private-read
```

Choose `legacy` or `v3` to match the profile being evaluated; omitting the profile selects `legacy`. The probe opens the entry through both core and Go UI sessions, compares snapshot-bound responses, and renders ten report/workbench pages with MyGo's headless UI tester. It also executes a fixed read-only query and compares query CSV, `.ocanalysis`, and journal-report CSV bytes in memory. It logs only page/export IDs, formats, byte counts, a one-way snapshot digest, profile, and rendered-node counts. It does not write to the ledger or user preferences and never prints the entry path, values, query text, or diagnostics. This is a read-only core/workspace/UI-rendering check; it does not replace the real GUI and VoiceOver gate.

On macOS, a separate explicit opt-in can take the first readable ledger from the production app's recent list without placing its path in a shell command. It reads from the real `org.orangecount.desktop` defaults domain, compares the same projections, and keeps the same path/value/diagnostic redaction:

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=0 ORANGECOUNT_TEST_PRIVATE_RECENT_LEDGER=1 \
  go test ./internal/app \
  -run '^TestPrivateLedgerUIReadOnlyProjectionFromProductionRecent$' -count=1 -v
```

This opens one of the user's recent ledgers read-only in memory; it does not write the ledger, preferences, or query presets. Leave the opt-in unset during routine tests.

## Run the isolated QA app

```sh
make goui-qa
```

This builds and launches `apps/goui/build/OrangeCount Go UI QA.app`. Open a public `.bean` fixture from the welcome page or the app's Open command. The QA bundle uses `org.orangecount.desktop.goui.qa`, a separate user-data identity, and does not migrate or read the legacy production UserDefaults domain. Use public fixtures for QA; this isolation does not make the unsigned preview a security boundary.

For an isolated packaged-app run in either QA or development mode, set
`ORANGECOUNT_ISOLATED_USER_DATA_DIR` to a new absolute private directory before
launching the executable. The app creates the directory with owner-only
permissions, refuses a relative or shared directory, and skips legacy defaults
migration for that run. This lets packaged GUI checks use the actual development
mode without reading or changing production preferences.

## Configure the source editor

In the app's **File / 文件** menu, choose **Choose Source Editor… / 选择来源编辑器…**
and select an external `.app` bundle. The preference stores that app's bundle
identifier, not its path, so moving or removing the app may make it unavailable
until you select it again. Choose **Use System Default for Sources / 使用系统默认应用打开来源**
to clear this preference. The selected editor is used by source-file actions
throughout the current app identity; each ledger window still applies its own
file-access policy.

Choose **Export Preferences Backup… / 导出偏好备份…** from the File menu to
save a point-in-time JSON copy of the app's versioned preferences. It includes
recent ledger paths, per-ledger reading settings, attachment-folder bookmarks,
and app settings; it excludes ledger contents, query drafts, and query results.
The save panel shows this scope before export, and the file is written with
owner-only permissions. The backup does not restore itself; first-run restore
acceptance remains part of the migration gate.

## Create a development package

```sh
OUT_DIR=/tmp/orangecount-goui-preview VERSION=0.0.0-dev make goui-package
```

The output directory contains:

- `OrangeCount Go UI Development.app`
- `OrangeCount-Go-UI-Development-<version>-<architecture>.zip`
- the ZIP's `.sha256` file

The package script refuses to overwrite an existing app, ZIP, or checksum. Choose a new `OUT_DIR` or `VERSION` for another build. Verify a ZIP from its output directory with:

```sh
cd /tmp/orangecount-goui-preview
shasum -a 256 -c OrangeCount-Go-UI-Development-0.0.0-dev-*.zip.sha256
```

From the repository root, verify the package after building it:

```sh
./apps/goui/verify-package.sh /tmp/orangecount-goui-preview 0.0.0-dev arm64
```

The verifier checks the ZIP checksum and extracted app, bundle/version and
minimum macOS metadata, Mach-O architecture and deployment target, CGO/Swift
linkage, and that the bundled license list exactly matches the resolved Go
module graph.

From the repository root, launch the bundle against the public fixture without changing its file association:

```sh
cd "$(git rev-parse --show-toplevel)"
open -n -a "/tmp/orangecount-goui-preview/OrangeCount Go UI Development.app" \
  "$PWD/testdata/fixtures/native/cashflow.bean"
```

The development bundle ID is `org.orangecount.desktop.goui.dev`, distinct from the QA identity. Its preferences and saved query presets are kept under the development app's own user-data location.

## Development-package limits

The app minimum is macOS 13.0, but the package is **not signed or notarized** and has not completed clean-machine installation, target-version, full GUI/VoiceOver, or upgrade acceptance. macOS may warn about or refuse an unsigned app. Do not treat this preview as a formal release or replace the existing production app with it. There is no automatic updater, installed `.bean` file association, or release signature in this package.
