<script lang="ts">
  import type { AdapterClient } from "../adapter-client";
  import GenericReport from "./GenericReport.svelte";
  import JournalReport from "./JournalReport.svelte";
  import { parseTableReport, type TableReport } from "./types";

  export let adapter: AdapterClient;
  export let account: string;

  let balance: TableReport | null = null;
  let journal: TableReport | null = null;
  let error = "";

  async function load() {
    try {
      const [accountValue, journalValue] = await Promise.all([
        adapter.load("account", { account }),
        adapter.load("journal", { account }),
      ]);
      balance = parseTableReport(accountValue);
      journal = parseTableReport(journalValue);
    } catch (value) {
      error = value instanceof Error ? value.message : "The account report could not be loaded.";
    }
  }

  load();
</script>

{#if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else}
  <div class="headerline"><h2>{account}</h2><span class="muted">Account detail</span></div>
  {#if balance}<GenericReport report={balance} title="Balance" />{/if}
  {#if journal}<JournalReport report={journal} {account} />{/if}
{/if}
