# G43 development bundle GUI export and restore loop

Date: 2026-10-08 (Asia/Singapore). This records a visible GUI pass in the current isolated development bundle; it is local development evidence, not release or full-product acceptance.

## Bundle

- App: `/tmp/orangecount-goui-g43-gui-20261008/OrangeCount Go UI Development.app`
- Version `0.0.0-g43`, bundle ID `org.orangecount.desktop.goui.dev`, arm64, minimum macOS 13.0.
- Source revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`, dirty tree; Go 1.27.1; MyGo `v0.2.16-0.20261006170038-869768d02d91`; CGO disabled; unsigned and unnotarized.
- ZIP: `OrangeCount-Go-UI-Development-0.0.0-g43-arm64.zip`; SHA-256 `307770defb1956b0128a1d29b2b94aeafd7d73f068f9fa2e874b0c813c08f03c`.
- `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g43-gui-20261008 VERSION=0.0.0-g43 GOARCH=arm64`: passed; plist, ZIP checksum, architecture and macOS minimum accepted.

## Visible packaged-app workflow

The exact `.app` above was launched, with its development identity isolation indicator visible. It opened only the public fixture `testdata/fixtures/native/cashflow.bean`; the app displayed 3 accounts, 2 transactions, 5 postings, and dashboard totals of USD 700 net worth, USD 1,000 income and USD 300 expenses.

In Query, `SELECT 1 AS v LIMIT 1` was explicitly entered and run. The packaged app displayed one typed result (`v = 1`), query fingerprint, snapshot/profile context, and both export actions. Using the actual macOS Save Panel, the query CSV was saved to a dedicated temporary directory. Its bytes were `v\n1\n`.

The Journal page showed 2 source transactions and a current-page export of 4 postings. Its CSV was saved through the native Save Panel and parsed back: 11 columns and exactly 4 posting rows. The Journal report analysis file was also saved through the native Save Panel; inspecting the archive showed `context.json`, `schema.json`, `cells.ndjson`, and `data.csv`, format 2, view `journal`, 11 schema columns, 4 typed rows and 2 source records.

The query analysis capture was then saved through the actual query Save Panel and reopened using the app’s `打开离线分析文件…` control. The packaged app visibly showed `query-result.ocanalysis`, format 1, the captured query and typed result `v = 1`; its status states that the capture was loaded without rerunning the query. The active public ledger stayed open.

The same packaged app opened its Attachment Roots page and presented a real native `NSOpenPanel` for adding a folder. Cancel closed the panel and the page still showed no additional roots.

## Same-workspace error recovery

The packaged app opened a private-mode `0600` copy of the public fixture under `/tmp/orangecount-goui-g43-recovery-20261008/main.bean`. Replacing that copy with the public syntax-error fixture caused the app to detect the file change, show `E-PARSE-STRING` at line 1, column 40, and retain the last valid dashboard snapshot and values. Restoring the original public fixture automatically returned the same workspace to “账本已读取” with no diagnostics and the expected 3 accounts, 2 transactions and 5 postings; no manual reopen was needed. The test did not write to the repository fixture or a user ledger.

After reopening the public fixture, Cmd+Q returned the visible app window to Welcome. The app inventory still reported the development bundle as running, so process termination is not counted as verified in this pass. The temporary ledger's development recent-path entry was removed from the development preference file without changing the other entries.

All generated files were placed in `/tmp/orangecount-goui-g43-exports-20261008`, separate from the repository, app bundle and user ledger locations.

## Remaining G43 scope

This closes the packaged read/query/CSV/report-analysis/query-analysis loop on this machine. It does not establish preference migration and dev/QA isolation across a clean-user matrix, first-launch installation on another machine, or G37/G38 IME, VoiceOver and complete legacy GUI equivalence. Those gates remain open; the bundle remains unsigned development software.
