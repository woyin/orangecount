<script lang="ts">
  import ReportChart from "../charts/ReportChart.svelte";
  import TreeTable from "../tree-table/TreeTable.svelte";
  import type { TreeReport } from "./types";

  export let report: TreeReport;

  $: firstColumn = report.trees.slice(0, 2);
  $: secondColumn = report.trees.slice(2);
  $: end = report.date_range?.end ?? null;
</script>

{#if report.charts.length}
  <div class="report-charts">
    {#each report.charts as chart (chart.title)}
      <ReportChart {chart} />
    {/each}
  </div>
{/if}

<div class="row">
  <div class="column">
    {#each firstColumn as tree (tree.account)}
      <TreeTable {tree} {end} />
    {/each}
  </div>
  <div class="column">
    {#each secondColumn as tree (tree.account)}
      <TreeTable {tree} {end} />
    {/each}
  </div>
</div>
