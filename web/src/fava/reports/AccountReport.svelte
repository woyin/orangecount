<script lang="ts">
  import type { AdapterClient } from "../adapter-client";
  import ReportChart from "../charts/ReportChart.svelte";
  import GenericReport from "./GenericReport.svelte";
  import JournalReport from "./JournalReport.svelte";
  import { parseJournalReport, parseTableReport, type JournalReport as JournalReportData, type TableReport } from "./types";

  export let adapter: AdapterClient;
  export let query: Record<string, string>;
  export let locale = "en";
  export let renderCommas = false;

  $: account = query.account || "";

  let balance: TableReport | null = null;
  let journal: JournalReportData | null = null;
  let error = "";
  let requestKey = "";

  $: if (requestKey !== JSON.stringify(query)) {
    requestKey = JSON.stringify(query);
    void load(requestKey);
  }

  async function load(key: string) {
    try {
      const [accountValue, journalValue] = await Promise.all([
        adapter.load("account", query),
        adapter.load("journal", query),
      ]);
      if (key !== requestKey) return;
      balance = parseTableReport(accountValue);
      journal = parseJournalReport(journalValue);
      error = "";
    } catch (value) {
      if (key !== requestKey) return;
      error = value instanceof Error ? value.message : "The account report could not be loaded.";
    }
  }
</script>

{#if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else}
  <div class="headerline"><h2>{account}</h2></div>
  {#if balance?.chart}
    <ReportChart chart={balance.chart} {locale} />
  {/if}
  {#if balance}<GenericReport report={balance} title="Balance" {locale} {renderCommas} />{/if}
  {#if journal}<JournalReport report={journal} {renderCommas} />{/if}
{/if}
