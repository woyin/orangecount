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
 // Fava's ConversionAndInterval renders two SelectComboboxes inside the chart
 // control bar; the header has neither.
 export let conversion = "at_cost";
 export let interval = "month";
 export let onConversion: (value: string) => void = () => {};
 export let onInterval: (value: string) => void = () => {};
 // TrialBalance.svelte renders its trees without column wrappers.
 export let singleColumn = false;

  // The adapter returns one chart per measure. Fava shows a single chart with
  // a row of links to switch measures rather than stacking them all, so the
  // report table stays on the first screen.
  let selected = 0;
  $: if (selected >= report.charts.length) selected = 0;
  $: chart = report.charts[selected];

  // Fava's Chart.svelte owns one currency legend above the plot and uses it as a
  // per-series on/off switch. The dots must come from the chart's own series so
  // both layers agree on the colour assignment order.
  let hiddenCurrencies: string[] = [];
  $: visibleCurrencies = [...new Set(chart?.series?.map((series) => series.label) ?? [])];
  function toggleCurrency(label: string) {
    hiddenCurrencies = hiddenCurrencies.includes(label)
      ? hiddenCurrencies.filter((value) => value !== label)
      : [...hiddenCurrencies, label];
  }
  // Fava derives the dot colour from a currency scale; a stable hash keeps the
  // assignment identical across renders and across the two chart layers.
  function currencyColor(label: string): string {
    let hash = 0;
    for (let index = 0; index < label.length; index++) {
      hash = (hash * 31 + label.charCodeAt(index)) >>> 0;
    }
    const hue = hash % 360;
    return `hsl(${hue} 62% 48%)`;
  }

  $: firstColumn = report.trees.slice(0, 2);
  $: secondColumn = report.trees.slice(2);
  $: end = report.date_range?.end ?? null;
</script>

{#if chart}
 <div class="report-charts">
   <!-- Fava's Chart.svelte puts a single control bar above the plot: the currency
        legend and the mode switch on the left, conversion and interval selects on
        the right, and a show/hide toggle. Rendering it here keeps the vertical
        composition (bar, plot, legend) identical to the baseline. -->
   <div class="chart-controls flex-row">
     {#if visibleCurrencies.length}
       <div class="chart-currency-legend">
         {#each visibleCurrencies as currency (currency)}
           <button
             type="button"
             class="currency-dot"
             class:inactive={hiddenCurrencies.includes(currency)}
             aria-pressed={!hiddenCurrencies.includes(currency)}
             title={currency}
             onclick={() => toggleCurrency(currency)}
           ><i style={`background-color: ${currencyColor(currency)}`}></i><span>{currency}</span></button>
         {/each}
       </div>
     {/if}
     <span class="spacer"></span>
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
     <label class="chart-select">
       <select value={conversion} onchange={(event) => onConversion((event.currentTarget as HTMLSelectElement).value)} aria-label="Conversion">
         <option value="at_cost">At Cost</option>
         <option value="at_value">At Market Value</option>
         <option value="units">Units</option>
         {#each operatingCurrencies as currency (currency)}
           <option value={currency}>{currency}</option>
         {/each}
       </select>
     </label>
     <label class="chart-select">
       <select value={interval} onchange={(event) => onInterval((event.currentTarget as HTMLSelectElement).value)} aria-label="Interval">
         <option value="day">Daily</option>
         <option value="week">Weekly</option>
         <option value="month">Monthly</option>
         <option value="quarter">Quarterly</option>
         <option value="year">Yearly</option>
       </select>
     </label>
   </div>
   <ChartView {chart} {locale} {hiddenCurrencies} />
  </div>
{/if}

<!-- Fava splits income statement and balance sheet into two columns with
     slice(0, 2) / slice(2), but its TrialBalance.svelte renders every tree as a
     direct child of .row with no column wrappers. Matching each route's own
     composition is what keeps the report heights aligned. -->
{#if singleColumn}
  <div class="row">
    {#each report.trees as tree (tree.account)}
      <TreeTable {tree} {end} {operatingCurrencies} {renderCommas} />
    {/each}
  </div>
{:else}
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
{/if}

<style>
  .chart-controls {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: var(--flex-gap, 0.5rem);
    color: var(--text-color-lighter);
    min-height: 1.6rem;
  }

  .chart-controls .spacer {
    flex: 1 1 auto;
  }

  .chart-currency-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .chart-currency-legend .currency-dot {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    background: none;
    border: none;
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
    font-variant-numeric: tabular-nums;
  }

  .chart-currency-legend .currency-dot i {
    display: inline-block;
    width: 0.7rem;
    height: 0.7rem;
    border-radius: 50%;
  }

  .chart-currency-legend .currency-dot.inactive span {
    text-decoration: line-through;
  }

  .chart-currency-legend .currency-dot.inactive i {
    filter: grayscale();
  }

  .chart-select select {
    border: 1px solid var(--input-border, var(--border));
    background: var(--input-bg, var(--background));
    color: inherit;
    font: inherit;
    padding: 0.1em 0.25em;
  }

  .chart-meta-inline {
    font-size: 0.85em;
    color: var(--text-color-lighter);
  }

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
