# Fava-aligned English UX baseline

This is the observable UX specification for the OrangeCount Fava frontend
transplant. The pinned Fava 1.30.12 source and approved synthetic-ledger
screenshots are the visual authority; OrangeCount must have no unapproved
visual difference under controlled conditions. This document describes the
route/state outcomes and structural anchors without storing private Fava data,
DOM, HTTP responses, or private-ledger evidence. Canonical route/state
coverage and acceptance status live in
`docs/fava-route-state-manifest.md`; this document supplies the shared
observable requirements for those entries.

## Baseline frame

| Frame | Layout baseline | Required outcome |
| --- | --- | --- |
| Desktop | Persistent application bar, visible primary navigation, and a dense content region with aligned controls and tables | The current route, period state, filters, and primary action are visible without losing navigation context |
| Narrow | Application bar remains usable; primary navigation becomes an explicit menu; tables and charts reflow or provide an intentional horizontal/detail alternative | No clipped controls or inaccessible actions; the same route and URL-backed state remain usable |
| English | English labels, dates, numbers, diagnostics, and keyboard help | Copy is stable enough for visual comparison; labels do not depend on browser locale |
| Offline | Local styles, charts, fixture documents, and data requests only | No remote asset, font, market-data, or telemetry request is required |

## Strict visual matrix

Every in-scope route must obtain approved English Chromium baselines for
desktop/light, desktop/dark, narrow/light, and narrow/dark during Prerequisite
Phase 0 and its route wave. No approved baseline exists yet. Layout,
typography, color, spacing,
density, control position, table/chart composition, and state structure must
match the Fava visual baseline unless an entry in
`docs/fava-approved-deviations.md` explicitly permits the bounded difference.
Rasterization noise may be handled by a narrowly documented comparison rule;
it is not permission for a global similarity threshold or broad mask.

Simplified Chinese uses the same route/state manifest and components. It must
preserve information hierarchy, table relationships, control and focus order,
keyboard behavior, wrapping, overflow safety, and responsive transitions, but
translated text is not compared pixel-for-pixel with the English baseline.

## Route and state table

| Route | Loaded state | Empty or unavailable state | Error or stale state |
| --- | --- | --- | --- |
| `/` | Resolve the applicable Fava default-page option and enter that standard route with ledger title and global state intact | Use the documented Fava fallback when the configured page is unavailable | Preserve the prior valid route and surface bootstrap/default-option failure |
| `/journal` | Transactions are grouped by date/source identity; postings are collapsed below each header | Explain that no transaction matches the filters | Keep filters and show an actionable load error |
| `/balance_sheet` | Account tree, natural-currency columns, totals, and hierarchy chart are visible | Explain that the selected period has no balances | Keep the last valid report and identify unavailable valuation data |
| `/income_statement` | Income and expense trees, period controls, totals, and chart are visible | Explain that the selected period has no activity | Preserve the prior report when a refresh fails |
| `/trial_balance` | Debit/credit or balance columns, account tree, currency legend, and chart/table fallback are visible | Explain that there are no report rows | Show an unavailable-data card instead of silently removing the chart |
| Account detail | Account title, filters, running-balance journal, and balance/changes chart are visible | Explain that the account has no activity in the selected period | Preserve account identity and show the report error |
| `/holdings` | Lots, units, cost, valuation status, and as-of controls are visible | Explain that no surviving lots exist | Distinguish unavailable price from unavailable conversion |
| `/commodities` | Commodity metadata, price history, filters, and empty-state affordance are visible | Explain that no commodity matches | Show a recoverable report error |
| `/documents` | Document rows expose safe attachment controls and source navigation | Explain that no documents match | Deny unsafe or missing attachments without exposing filesystem details |
| `/events` | Event rows, date filters, and source navigation are visible | Explain that no events match | Keep the filter state and show the error |
| `/statistics` | Deterministic metrics plus an accessible chart/table alternative are visible | Explain that there is no data to chart | Show an actionable calculation error |
| `/query` | Editor, run/save/export controls, and typed result table are visible | Explain that the query returned no rows | A parse/evaluation error must not erase the last successful result |
| `/editor` | File tree, buffer, diagnostics, validate, save, and revert controls are visible | Explain that the selected source has no diagnostics | Failed validation leaves the previous snapshot and editable content recoverable |
| `/import` | Local source selection, adapter/mapping controls, preview rows, diff, review, and commit are visible | Explain that the source contains no candidates | Invalid candidates remain in preview with diagnostics; no implicit commit occurs |
| `/options` | Every applicable built-in Fava option is grouped and has its real interface effect; excluded capabilities are explicit approved deviations | Explain when an optional value is unset | Reject invalid changes without changing the saved option |
| `/errors` | Conditional navigation, diagnostics, source anchors, and Fava budget/FQL/import errors are visible | Remove the conditional navigation entry when there are no diagnostics | Preserve the last valid diagnostic set and identify refresh failure |
| `/help/<slug>` | Searchable help and keyboard-shortcut sections are visible | Explain that no help topic matches | Preserve the prior route and search text if loading fails |
| Global modals | Add Entry, Context, Export, notification, document, and confirmation surfaces preserve Fava composition and focus behavior | Explain unavailable actions without rendering a broken modal | Close safely, restore focus, and retain uncommitted user input where recovery is possible |

## Shared controls

| Control group | Desktop presentation | Narrow presentation | State and URL rule |
| --- | --- | --- | --- |
| Application bar | Current page title, ledger title, period/time control, global account/text filters, currency preference, locale/theme controls | Keep the title and menu action visible; move secondary controls into a reachable panel | Persist user-visible route state in the URL where it changes a report or journal |
| Primary navigation | Stable grouped links with an accessible active state | Explicit menu button, focusable links, and return-to-content behavior | Direct route entry and reload produce the same selected route |
| Report toolbar | Period/as-of, valuation, currency preference, tree/chart, sort, export/print | Wrap controls in a predictable order; keep primary action first | Reset restores the deterministic default without changing ledger semantics |
| Journal toolbar | Date range, account/text, tags/links, payee/narration, flags, directive-kind badges, pagination | Controls wrap or enter a labelled filter panel; transaction expansion remains reachable | Filtering never changes transaction grouping; clear/reset is explicit |
| Multi-currency table | One account row with dynamic natural-currency columns; blank cells mean no holding, not an unavailable report | Preserve account identity while allowing detail or intentional horizontal scrolling | Currency preference affects single-value summaries/charts, never hides natural holdings |
| Hierarchy chart | Treemap, sunburst, or icicle control with currency legend and table fallback | Chart controls remain keyboard reachable; table fallback is available without hover | A zero or unconvertible series becomes an explanatory card, not a missing card |
| Write controls | Validate before save/commit; show review, backup, and rollback status | Keep destructive or publishing actions explicit and separated from preview | A failed write never publishes a new snapshot |

## Keyboard table

| Key or action | Context | Expected result |
| --- | --- | --- |
| `Tab` / `Shift+Tab` | Every route | Move through controls in visual reading order with a visible focus indicator |
| `Enter` / `Space` | Navigation, buttons, menu items, chart series, tree rows | Activate the focused control without requiring a pointer |
| `Escape` | Narrow menu, filter panel, modal, expanded transaction | Close the transient surface and return focus to its opener |
| `Arrow keys` | Account tree, hierarchy chart legend/series, select-like controls | Move within the focused group without losing the selected route state |
| `Home` / `End` | Journal and report pagination or result tables | Move to the first or last deterministic page/row group where supported |
| `Ctrl+Enter` | Query and editor | Run or validate the current content without publishing a write |
| `Ctrl+S` | Editor | Open the explicit save/review path; never silently write invalid content |
| `Enter` on a table header | Sortable report, journal, query result | Toggle typed ascending/descending sort and expose `aria-sort` |
| `Enter` on a transaction or account | Journal, report, account tree | Expand details or navigate to account detail while retaining bookmarkable state |

## High-impact visual acceptance

1. **Natural currencies:** report tables group by account and expose all natural
   currency columns present in the fixture. A missing conversion affects only
   the converted summary/chart and is labelled; it does not remove the row.
2. **Journal grouping:** one transaction header carries date, flag, narration,
   source affordance, and posting count. Its postings expand beneath it with
   indentation and no repeated transaction header content.
3. **Account detail:** an account link opens a dedicated view with running
   balance, account-scoped journal, and a balance/changes chart with a table
   alternative.
4. **Hierarchy charts:** the account tree and chart share the same selected
   currency and deterministic ordering. Treemap, sunburst, and icicle states
   remain structurally present when data is unavailable.
5. **Privacy:** approved screenshots may be committed only when generated from
   the deterministic synthetic reference ledger in the controlled reference
   environment. Never commit private-ledger screenshots, raw private browser
   output, private source text, private attachment content, or private local-
   session URLs.
