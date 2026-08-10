<script lang="ts">
  import ReportChart from "./ReportChart.svelte";
  import ModernChart from "./ModernChart.svelte";
  import { translations, type Locale } from "../../translations";
  import type { ReportChart as ReportChartData } from "../reports/types";

  // ChartView is the chart-rendering seam. Time-series charts (bar/line) render
  // through the modern d3-backed ModernChart, which is the standard presentation
  // for the income-statement, balance-sheet, and account routes. Hierarchy
  // charts (treemap/sunburst/icicle) are out of scope for the modern layer and
  // still render through the parity ReportChart. See ADR-0040/0041.

  export let chart: ReportChartData;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: isTimeSeries = chart.kind === "bar" || chart.kind === "stacked-bar" || chart.kind === "line";
  // Report metadata is semantic and English internally. Localize only the
  // presentation copy that this OC-specific chart adds at the rendering seam.
  $: displayedChart = chart.measure === "average-cost"
    ? { ...chart, title: t("averageCostEvolution"), unit: t("costPerUnit") }
    : chart;
</script>

{#if isTimeSeries}
  <ModernChart chart={displayedChart} {locale} />
{:else}
  <ReportChart chart={displayedChart} {locale} />
{/if}
