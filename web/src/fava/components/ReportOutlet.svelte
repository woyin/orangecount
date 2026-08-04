<script lang="ts">
  import type { AdapterClient } from "../adapter-client";
  import AccountReport from "../reports/AccountReport.svelte";
  import GenericReport from "../reports/GenericReport.svelte";
  import ImportReport from "../reports/ImportReport.svelte";
  import JournalReport from "../reports/JournalReport.svelte";
  import QueryReport from "../reports/QueryReport.svelte";
  import EditorReport from "../reports/EditorReport.svelte";
  import TreeReport from "../reports/TreeReport.svelte";
  import UtilityReport from "../reports/UtilityReport.svelte";
  import { pageLabel } from "../router.mjs";
  import { parseTableReport, parseTreeReport, type TableReport, type TreeReport as TreeReportData } from "../reports/types";

  export let adapter: AdapterClient;
  export let route: string;
  export let query: Record<string, string> = {};

  let loadedKey = "";
  let loading = false;
  let error: string | null = null;
  let report: TreeReportData | null = null;
  let table: TableReport | null = null;

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
    if (["query", "options", "help", "diagnostics", "source", "editor", "import"].includes(route) || !["income_statement", "balance_sheet", "trial_balance", "accounts", "account", "journal", "holdings", "holdings_by_account", "holdings_by_currency", "holdings_by_root_account", "holdings_by_commodity", "commodities", "events", "documents", "statistics", "errors"].includes(route)) {
      loading = false;
      return;
    }
    try {
      const payload = await adapter.load(route, query);
      if (key !== requestKey) return;
      if (["income_statement", "balance_sheet", "trial_balance"].includes(route)) {
        report = parseTreeReport(payload);
      } else {
        table = parseTableReport(payload);
      }
    } catch (value) {
      if (key !== requestKey) return;
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
  <AccountReport adapter={adapter} account={query.account || ""} />
{:else if route === "editor"}
  <EditorReport {adapter} />
{:else if route === "import"}
  <ImportReport {adapter} />
{:else if ["options", "help", "diagnostics", "source"].includes(route)}
  <UtilityReport {adapter} {route} />
{:else if report}
  <TreeReport {report} />
{:else if table && route === "journal"}
  <JournalReport report={table} />
{:else if table}
  <GenericReport report={table} title={pageLabel(route)} />
{:else}
  <section class="route-placeholder">
    <p class="headerline"><strong>Fava-aligned shell</strong></p>
    <h2>{route}</h2>
    <p>This route is staged until its OrangeCount adapter contract is implemented.</p>
  </section>
{/if}
