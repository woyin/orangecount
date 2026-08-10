<!-- This file is derived from Fava 1.30.12 (commit #aa7538e8971252c9efc52c8a516a3a77d604553f),
which is Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name> and
distributed under the MIT License. Adapted for OrangeCount; see NOTICE and
web/provenance-manifest.json. The MIT notice is reproduced here:

  Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name>

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in all
  copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE. -->

<script lang="ts">
 import ChartView from "../charts/ChartView.svelte";
 import TreeTable from "../tree-table/TreeTable.svelte";
  import type { TreeReport } from "./types";

 export let report: TreeReport;
 export let locale = "en";
 export let operatingCurrencies: string[] = [];
 export let renderCommas = false;

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
   <ChartView {chart} {locale} />
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
