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

  // Fava renders the account title as a breadcrumb where every ancestor is a
  // link to that account level; only the leaf segment of each is shown.
  function ancestors(name: string): string[] {
    const result: string[] = [];
    let index = name.indexOf(":");
    while (index !== -1) {
      result.push(name.slice(0, index));
      index = name.indexOf(":", index + 1);
    }
    if (name !== "") result.push(name);
    return result;
  }

  function leaf(name: string): string {
    const parentEnd = name.lastIndexOf(":");
    return parentEnd > 0 ? name.slice(parentEnd + 1) : name;
  }

  function accountHref(name: string): string {
    return `/account/${encodeURIComponent(name)}`;
  }

  $: parts = ancestors(account);

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
  <div class="headerline"><h2 class="account-breadcrumb">{#each parts as name, index (name)}<a href={accountHref(name)} title={name}>{leaf(name)}</a>{#if index < parts.length - 1}<span class="sep">:</span>{/if}{/each}</h2></div>
  {#if balance?.chart}
    <ReportChart chart={balance.chart} {locale} />
  {/if}
  {#if balance}<GenericReport report={balance} title="Balance" {locale} {renderCommas} />{/if}
  {#if journal}<JournalReport report={journal} {renderCommas} accountFilter={account} />{/if}
{/if}

<style>
  .account-breadcrumb a {
    color: unset;
  }

  .account-breadcrumb .sep {
    color: var(--text-color-lightest);
  }
</style>
