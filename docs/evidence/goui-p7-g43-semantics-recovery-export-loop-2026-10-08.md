# G43 semantics-recovery package export loop

Date: 2026-10-08 (Asia/Singapore). This supplements the visible semantics and holdings pass with export/readback evidence from the same current arm64 package.

## Package and isolation

- Source app: `/tmp/orangecount-goui-g43-semantics-recovery-20261008/OrangeCount Go UI Development.app`; version `0.0.0-g43-semantics-recovery`, arm64, minimum macOS 13.0, unsigned and unnotarized.
- ZIP SHA-256: `89c51ed0ba1de7d08aaebe905ed2cf43d2c0c4ae571a49aebba7936af4e2c3bd`; executable SHA-256: `310f976ad5f1838637e588652c5d44e113037f468fbac518e9a83a9b9db3c17b`.
- Re-ran `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g43-semantics-recovery-20261008 VERSION=0.0.0-g43-semantics-recovery GOARCH=arm64`; plist, ZIP checksum, arm64 architecture and macOS minimum passed.
- Visible testing used the isolated copy `/tmp/orangecount-goui-g43-semantics-recovery-visible-20261008/OrangeCount Go UI Semantics Recovery QA.app`, whose executable hash matches the package. It ran with `ORANGECOUNT_QA=1` and a separate owner-only user-data directory. All generated exports stayed in `/tmp/orangecount-goui-g43-semantics-recovery-visible-20261008/exports`.

## Query export and readback

The packaged app opened the public `testdata/fixtures/native/cashflow.bean` and showed 3 accounts, 2 transactions and 5 postings. In Query, the explicit query `SELECT 1 AS value, 'current-package' AS source LIMIT 1` returned one typed row (`1`, `current-package`).

Using the native Save Panel, the app saved `cashflow-query.csv` and `cashflow-query.ocanalysis`. The CSV parsed as header `value,source` followed by `1,current-package` (SHA-256 `59fdbe556a4a4ce48611face4ed9f3504f0a19d085c429a49962aa4580cf5490`). The analysis archive contains `context.json`, `schema.json`, `cells.ndjson` and `data.csv`; its context records the query text, snapshot, legacy profile and complete one-row capture (SHA-256 `a3df06d4d1ba071d54fa1c8e50350272a9310c4fd0e635dc8c5014c840ef4f91`).

The app then opened that file with its Query page “打开离线分析文件…” control. The visible page identified it as a format-1 typed query capture and displayed the same query and row. The status explicitly said the capture was loaded without re-executing the query; the active public ledger remained open.

## Journal exports

The Journal page showed 2 matching source transactions and 4 posting rows. The native Save Panel wrote `cashflow-transactions--2026-01-03.csv`; parsing found 11 columns and exactly 4 data rows (SHA-256 `ae18a127880b608339fb5446d18b89dafc287923a0f89ffa7be48b6f94be1638`). The page also saved `cashflow-transactions--2026-01-03.ocanalysis`; archive inspection found the four expected members, format 2, `view=journal`, 11 typed columns, 4 result rows and 2 source records (SHA-256 `572a4831fca85d36c1a09d9e91d59775d54105dd8db01179c5f0b89a42f5801b`).

## Additional pages in the same package

In a second launch of the same QA bundle and user-data directory, the welcome page listed only the two public fixtures used in this run and did not reopen either automatically. `cashflow.bean` again showed 3 accounts, 2 transactions, 5 postings, 700 USD net worth, 1,000 USD income and 300 USD expenses.

- Balance Sheet showed 2 complete-projection rows and `Assets 700 USD`; its CSV saved as `cashflow-balance-sheet--2026-01-03.csv`, parsed to the `Assets` and `Assets:Bank` rows (SHA-256 `220c9236608511e2ba0646ed342f32158502484a79cb6615079aa72c9212d280`).
- Income Statement showed 4 complete-projection rows (`Income -1000 USD`, `Expenses 300 USD`); its CSV saved as `cashflow-tree--2026-01-03.csv` and parsed to `Income`, `Income:Salary`, `Expenses` and `Expenses:Food` (SHA-256 `25a63c3a3d67aca7ba982fc2d323a666516ff1b15f92f5b2e3ce74f0ff6be49f`).
- The `Assets:Bank` account detail showed a 700 USD ending balance, 2 source transactions and their typed postings/source locations.
- Commodity catalogue showed the single USD commodity and no local quote pair; the Price History page correctly showed no history without a local quote.
- All Directives showed 5 records with source locations. Attachments showed no document directives; Attachment Roots showed only the explicit default fixture directory. The real native directory chooser opened and was cancelled; the roots page remained unchanged with no additional directory.

The two report CSVs, like the earlier query and Journal exports, were saved to the isolated temporary export directory and parsed after saving. The second Cmd+Q was followed by an exact process-table check; no QA executable remained.

The Query page's offline reader is specifically for query captures and calls `ReadQueryAnalysis`; it rejects a journal report archive because report captures use the separate format-2 `analysis.ReadReport` contract. I attempted that cross-type open during the check, saw the explicit “invalid query analysis container” error, and retained the prior query capture. This was not counted as a report-reader pass; the journal archive was independently inspected with its report schema/context and typed rows.

Finally, Cmd+Q quit the isolated visible app. A process-table check found no remaining `OrangeCount` process. These checks are local development evidence only; they do not cover old-preference migration, the full dev/QA isolation matrix, G37/G38 legacy equivalence and human IME/VoiceOver checks, or signed/notarized distribution.
