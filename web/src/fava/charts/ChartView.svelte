<script lang="ts">
  import ReportChart from "./ReportChart.svelte";
  import ModernChart from "./ModernChart.svelte";
  import { translations, type Locale } from "../../translations";
  import type { ReportChart as ReportChartData } from "../reports/types";

  // ChartView is the seam between the parity chart layer and the modern chart
  // layer. It renders the parity ReportChart by default and switches to the
  // ModernChart only when chart_layer=modern AND the chart is a bar/line
  // time-series. Hierarchy charts (treemap/sunburst/icicle) are out of scope for
  // the modern layer (first phase) and always use parity. The parity layer is
  // never modified by this switch — see ADR-0040.

  export let chart: ReportChartData;
  export let locale = "en";
  export let chartLayer: "parity" | "modern" = "parity";
  export let onChartLayer: (value: string) => void = () => {};

  $: catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
  function label(key: string): string { return catalog[key] ?? translations.en[key] ?? key; }

  $: isTimeSeries = chart.kind === "bar" || chart.kind === "stacked-bar" || chart.kind === "line";
  $: useModern = chartLayer === "modern" && isTimeSeries;
</script>

{#if isTimeSeries}
  <div class="chart-layer-bar">
    <button
      type="button"
      class="chart-layer-toggle"
      class:active={useModern}
      aria-pressed={useModern}
      title={useModern ? label("modernChartLayer") : label("parityChartLayer")}
      on:click={() => onChartLayer(useModern ? "" : "modern")}
    >{label(useModern ? "useParityChart" : "useModernChart")}</button>
  </div>
{/if}

{#if useModern}
  <ModernChart {chart} {locale} />
{:else}
  <ReportChart {chart} {locale} />
{/if}

<style>
  .chart-layer-bar { display: flex; justify-content: flex-end; margin-bottom: 0.25rem; }
  .chart-layer-toggle { background: none; border: 1px solid var(--border); color: var(--text-color-lighter); padding: 0.1rem 0.5rem; font-size: 0.75rem; cursor: pointer; border-radius: 3px; font-variant-numeric: tabular-nums; }
  .chart-layer-toggle:hover { color: var(--text-color); border-color: var(--text-color-lightest); }
  .chart-layer-toggle.active { color: var(--series-0, #2563eb); border-color: var(--series-0, #2563eb); }
</style>
