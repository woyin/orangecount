<script lang="ts">
  import { onMount } from "svelte";
  import type { AdapterClient } from "../adapter-client";
  import GenericReport from "./GenericReport.svelte";
  import { parseTableReport, type TableReport } from "./types";

  export let adapter: AdapterClient;

  let queryText = "SELECT account, balance FROM accounts ORDER BY account";
  let result: TableReport | null = null;
  let loading = false;
  let error = "";

  async function run() {
    loading = true;
    error = "";
    try {
      result = parseTableReport(await adapter.load("query", { query_string: queryText }));
    } catch (value) {
      error = value instanceof Error ? value.message : "The query could not be evaluated.";
    } finally {
      loading = false;
    }
  }

  onMount(() => { void run(); });
</script>

<div class="headerline">
  <h2>Query</h2>
</div>
<form class="query-form" on:submit|preventDefault={run}>
  <label for="query-editor">BeanQuery</label>
  <textarea id="query-editor" bind:value={queryText} spellcheck="false" rows="4"></textarea>
  <button type="submit" disabled={loading}>{loading ? "Running…" : "Run query"}</button>
</form>
{#if error}
  <p class="error-panel" role="alert">{error}</p>
{:else if result}
  <p><a class="button" href={`/api/v1/query?q=${encodeURIComponent(queryText)}&format=csv`}>Export CSV</a></p>
  <GenericReport report={result} title="Query result" />
{/if}

<style>
  .query-form {
    display: grid;
    gap: 0.5rem;
    max-width: 70rem;
    margin-bottom: 1rem;
  }

  textarea {
    width: 100%;
    font-family: var(--font-family-editor);
    resize: vertical;
  }

  .error-panel {
    padding: 0.5rem;
    color: var(--error);
    border: 1px solid var(--error);
  }
</style>
