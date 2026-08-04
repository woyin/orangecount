<script lang="ts">
  import type { AdapterClient } from "../adapter-client";
  import TreeReport from "../reports/TreeReport.svelte";
  import { parseTreeReport, type TreeReport as TreeReportData } from "../reports/types";

  export let adapter: AdapterClient;
  export let route: string;
  export let query: Record<string, string> = {};

  let loadedKey = "";
  let loading = false;
  let error: string | null = null;
  let report: TreeReportData | null = null;

  $: requestKey = `${route}?${new URLSearchParams(query).toString()}`;
  $: if (requestKey !== loadedKey) {
    loadedKey = requestKey;
    void load(requestKey);
  }

  async function load(key: string) {
    loading = true;
    error = null;
    report = null;
    if (!["income_statement", "balance_sheet", "trial_balance"].includes(route)) {
      loading = false;
      return;
    }
    try {
      const payload = await adapter.load(route, query);
      if (key !== requestKey) return;
      report = parseTreeReport(payload);
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
{:else if report}
  <TreeReport {report} />
{:else}
  <section class="route-placeholder">
    <p class="headerline"><strong>Fava-aligned shell</strong></p>
    <h2>{route}</h2>
    <p>This route is staged until its OrangeCount adapter contract is implemented.</p>
  </section>
{/if}
