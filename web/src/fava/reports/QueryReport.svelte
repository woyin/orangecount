<script lang="ts">
  import { onMount } from "svelte";
  import type { EditorState } from "@codemirror/state";
  import type { EditorView } from "@codemirror/view";
  import type { AdapterClient } from "../adapter-client";
  import { init_query_editor, replace_contents } from "../codemirror/bql";
  import LineChart from "../charts/LineChart.svelte";
  import GenericReport from "./GenericReport.svelte";
  import { parseTableReport, type DecimalWire, type TableReport } from "./types";

  export let adapter: AdapterClient;
  export let query: Record<string, string> = {};

  let queryText = query.query_string || "SELECT account, balance FROM accounts ORDER BY account";
  let appliedRouteQuery = query.query_string ?? "";
  let result: TableReport | null = null;
  let loading = false;
  let error = "";
  let showCharts = true;
  let editorHost: HTMLDivElement;
  let editor: EditorView | null = null;

  function onDocChanges(state: EditorState) {
    queryText = state.sliceDoc();
  }

  function syncEditorFromRoute(text: string) {
    if (editor) {
      editor.dispatch(replace_contents(editor.state, text));
    }
  }

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

  $: {
    const routed = query.query_string ?? "";
    if (routed && routed !== appliedRouteQuery) {
      appliedRouteQuery = routed;
      queryText = routed;
      syncEditorFromRoute(routed);
      void run();
    }
  }

  onMount(() => {
    editor = init_query_editor(
      queryText,
      onDocChanges,
      "...enter a BQL query. 'help' to list available commands.",
      () => run,
    );
    editorHost.appendChild(editor.dom);
    return () => editor?.destroy();
  });

  // Mirrors upstream getQueryChart's date+Inventory branch: chart a result
  // with exactly two columns where the first is a date and the second an
  // amount. The shell has no Inventory dtype, so amounts are sniffed from
  // PresentedDecimal / number / numeric-string values.
  const isoDate = /^\d{4}-\d{2}-\d{2}$/;

  function numberValue(value: unknown): number {
    if (typeof value === "number") return Number.isFinite(value) ? value : NaN;
    if (typeof value === "string") {
      if (value.includes("/")) {
        const [numerator, denominator] = value.split("/").map(Number);
        return denominator ? numerator / denominator : NaN;
      }
      return Number(value);
    }
    if (Array.isArray(value)) {
      let total = 0;
      for (const item of value) {
        const parsed = numberValue(item);
        if (!Number.isFinite(parsed)) return NaN;
        total += parsed;
      }
      return total;
    }
    if (value && typeof value === "object") {
      const wire = value as DecimalWire;
      if (typeof wire.display === "string") return numberValue(wire.display);
    }
    return NaN;
  }

  function displayValue(value: unknown): string {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      const wire = value as DecimalWire;
      if (typeof wire.display === "string") return wire.display;
    }
    if (Array.isArray(value)) return value.map(displayValue).join(" + ");
    return String(value);
  }

  $: chartPoints = (() => {
    if (!result || result.columns.length !== 2 || !result.rows.length) return [];
    const [dateColumn, valueColumn] = result.columns;
    const points: { date: string; display: string; value: number }[] = [];
    for (const row of result.rows) {
      const rawDate = row[dateColumn];
      if (typeof rawDate !== "string" || !isoDate.test(rawDate)) return [];
      const value = numberValue(row[valueColumn]);
      if (!Number.isFinite(value)) return [];
      points.push({ date: rawDate, display: displayValue(row[valueColumn]), value });
    }
    return points;
  })();

  onMount(() => { void run(); });
</script>

<div class="headerline">
  <h2>Query</h2>
</div>
<form class="query-form" on:submit|preventDefault={run}>
  <label for="query-editor">BeanQuery</label>
  <div id="query-editor" bind:this={editorHost} aria-label="BeanQuery"></div>
  <button type="submit" disabled={loading}>{loading ? "Running…" : "Run query"}</button>
</form>
{#if error}
  <p class="error-panel" role="alert">{error}</p>
{:else if result}
  <p><a class="button" href={`/api/v1/query?q=${encodeURIComponent(queryText)}&format=csv`}>Export CSV</a></p>
  {#if chartPoints.length}
    <div class="flex-row">
      <span class="spacer"></span>
      <button type="button" class="show-charts" on:click={() => (showCharts = !showCharts)}>{showCharts ? "▼" : "◀"}</button>
    </div>
    {#if showCharts}
      <LineChart points={chartPoints} />
    {/if}
  {/if}
  <GenericReport report={result} title="Query result" />
{/if}

<style>
  .query-form {
    display: grid;
    gap: 0.5rem;
    max-width: 70rem;
    margin-bottom: 1rem;
  }

  .flex-row {
    display: flex;
    gap: var(--flex-gap, 0.5rem);
    margin-bottom: var(--flex-gap, 0.5rem);
  }

  .spacer {
    flex: 1;
  }

  #query-editor {
    min-height: 6rem;
    border: 1px solid var(--border);
  }
  #query-editor :global(.cm-scroller) { font-family: var(--font-family-editor); }
  #query-editor :global(.cm-editor) { width: 100%; }

  .error-panel {
    padding: 0.5rem;
    color: var(--error);
    border: 1px solid var(--error);
  }
</style>
