<script lang="ts">
  import type { AdapterClient } from "../adapter-client";
  import AccountReport from "../reports/AccountReport.svelte";
  import GenericReport from "../reports/GenericReport.svelte";
  import HoldingsReport from "../reports/HoldingsReport.svelte";
  import ImportReport from "../reports/ImportReport.svelte";
  import JournalReport from "../reports/JournalReport.svelte";
  import QueryReport from "../reports/QueryReport.svelte";
  import EditorReport from "../reports/EditorReport.svelte";
  import EventsReport from "../reports/EventsReport.svelte";
  import TreeReport from "../reports/TreeReport.svelte";
  import UtilityReport from "../reports/UtilityReport.svelte";
  import { notify_err } from "../notifications";
  import { pageLabel } from "../router.mjs";
  import { parseJournalReport, parseTableReport, parseTreeReport, type JournalReport as JournalReportData, type TableReport, type TreeReport as TreeReportData } from "../reports/types";

  export let adapter: AdapterClient;
  export let route: string;
  export let query: Record<string, string> = {};
  export let locale = "en";
  export let theme = "system";
  export let operatingCurrencies: string[] = [];
  export let renderCommas = false;
  export let onLocale: (value: string) => void = () => {};
  export let onTheme: (value: string) => void = () => {};

  let loadedKey = "";
  let loading = false;
  let error: string | null = null;
  let report: TreeReportData | null = null;
  let table: TableReport | null = null;
  let journal: JournalReportData | null = null;

  $: requestKey = `${route}?${new URLSearchParams(query).toString()}`;
  $: if (requestKey !== loadedKey) {
    loadedKey = requestKey;
    void load(requestKey);
  }

  async function load(key: string) {
    loading = true;
    error = null;
    report = null;
    table = null;
    journal = null;
    if (["query", "options", "help", "diagnostics", "source", "editor", "import"].includes(route) || !["income_statement", "balance_sheet", "trial_balance", "accounts", "journal", "holdings", "holdings_by_account", "holdings_by_currency", "holdings_by_root_account", "holdings_by_commodity", "commodities", "events", "documents", "statistics", "errors"].includes(route)) {
      loading = false;
      return;
    }
    try {
      const payload = await adapter.load(route, query);
      if (key !== requestKey) return;
      if (["income_statement", "balance_sheet", "trial_balance"].includes(route)) {
        report = parseTreeReport(payload);
      } else if (route === "journal") {
        journal = parseJournalReport(payload);
      } else {
        table = parseTableReport(payload);
      }
    } catch (value) {
      if (key !== requestKey) return;
      notify_err(value);
      error = value instanceof Error ? value.message : "The report could not be loaded.";
    } finally {
      if (key === requestKey) loading = false;
    }
  }
</script>

{#if loading}
  <section class="state-panel" role="status" aria-live="polite">Loading report…</section>
{:else if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else if route === "query"}
  <QueryReport {adapter} />
{:else if route === "account"}
  <AccountReport adapter={adapter} {query} {locale} {renderCommas} />
{:else if route === "editor"}
  <EditorReport {adapter} />
{:else if route === "import"}
  <ImportReport {adapter} />
{:else if ["options", "help", "diagnostics", "source"].includes(route)}
  <UtilityReport {adapter} {route} query={query} {locale} {theme} {onLocale} {onTheme} />
{:else if report}
  <TreeReport {report} {locale} {operatingCurrencies} {renderCommas} />
{:else if journal}
  <JournalReport report={journal} {renderCommas} />
{:else if table && (route === "holdings" || route.startsWith("holdings_by_"))}
  <HoldingsReport report={table} {route} {locale} {renderCommas} />
{:else if table && route === "events"}
  <EventsReport report={table} {locale} />
{:else if table}
  <GenericReport report={table} title={pageLabel(route)} {route} {locale} {renderCommas} />
{:else}
  <section class="route-placeholder">
    <p class="headerline"><strong>Fava-aligned shell</strong></p>
    <h2>{route}</h2>
    <p>This route is staged until its OrangeCount adapter contract is implemented.</p>
  </section>
{/if}
