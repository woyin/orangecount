<script lang="ts">
 import ChartView from "../charts/ChartView.svelte";
 import TreeTable from "../tree-table/TreeTable.svelte";
  import type { TreeReport } from "./types";

 export let report: TreeReport;
 export let locale = "en";
 export let operatingCurrencies: string[] = [];
 export let renderCommas = false;
 export let chartLayer: "parity" | "modern" = "parity";
 export let onChartLayer: (value: string) => void = () => {};

  // The adapter returns one chart per measure. Fava shows a single chart with
  // a row of links to switch measures rather than stacking them all, so the
  // report table stays on the first screen.
  let selected = 0;
  $: if (selected >= report.charts.length) selected = 0;
  $: chart = report.charts[selected];

  $: firstColumn = report.trees.slice(0, 2);
  $: secondColumn = report.trees.slice(2);
  $: end = report.date_range?.end ?? null;
</script>

{#if chart}
 <div class="report-charts">
   <ChartView {chart} {locale} {chartLayer} {onChartLayer} />
   {#if report.charts.length > 1}
      <nav class="chart-picker" aria-label={chart.title}>
        {#each report.charts as option, index (option.title)}
          <button
            type="button"
            class="unset"
            class:selected={index === selected}
            aria-pressed={index === selected}
            onclick={() => (selected = index)}
          >{option.title}</button>
        {/each}
      </nav>
    {/if}
  </div>
{/if}

<div class="row">
  <div class="column">
    {#each firstColumn as tree (tree.account)}
      <TreeTable {tree} {end} {operatingCurrencies} {renderCommas} />
    {/each}
  </div>
  <div class="column">
    {#each secondColumn as tree (tree.account)}
      <TreeTable {tree} {end} {operatingCurrencies} {renderCommas} />
    {/each}
  </div>
</div>

<style>
  .chart-picker {
    display: flex;
    flex-wrap: wrap;
    gap: 0 1rem;
    justify-content: center;
    margin-bottom: 1rem;
  }

  .chart-picker button {
    color: var(--link-color);
    cursor: pointer;
  }

  .chart-picker button.selected {
    font-weight: 500;
    color: var(--text-color);
    cursor: default;
  }
</style>
