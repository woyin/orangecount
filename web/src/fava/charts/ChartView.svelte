<script lang="ts">
  import ReportChart from "./ReportChart.svelte";
  import ModernChart from "./ModernChart.svelte";
  import type { ReportChart as ReportChartData } from "../reports/types";

  // ChartView is the chart-rendering seam. Time-series charts (bar/line) render
  // through the modern d3-backed ModernChart, which is the standard presentation
  // for the income-statement, balance-sheet, and account routes. Hierarchy
  // charts (treemap/sunburst/icicle) are out of scope for the modern layer and
  // still render through the parity ReportChart. See ADR-0040/0041.

  export let chart: ReportChartData;
  export let locale = "en";

  $: isTimeSeries = chart.kind === "bar" || chart.kind === "stacked-bar" || chart.kind === "line";
</script>

{#if isTimeSeries}
  <ModernChart {chart} {locale} />
{:else}
  <ReportChart {chart} {locale} />
{/if}
