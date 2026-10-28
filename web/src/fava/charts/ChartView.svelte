<script lang="ts">
  import ReportChart from "./ReportChart.svelte";
  import ModernChart from "./ModernChart.svelte";
  import { translations, type Locale } from "../../translations";
  import type { ReportChart as ReportChartData } from "../reports/types";

  // ChartView is the chart-rendering seam. The parity ReportChart is the
  // default: it is the ported Fava Chart.svelte, so a standard route matches the
  // Fava baseline it is measured against. The modern d3 layer stays available
  // behind ?chart_layer=modern, which keeps the switch-back that makes it an
  // approved presentation preference instead of silent drift.
  //
  // ADR-0041 had promoted the modern layer to the default and removed this
  // fallback; that also invalidated FD-0006's basis, because a presentation
  // change that cannot be switched back to parity is not a valid deviation.

  export let chart: ReportChartData;
  export let locale = "en";
  // Currencies toggled off in the shared control bar. Both chart layers read the
  // same list so the legend and the plot never disagree.
  export let hiddenCurrencies: string[] = [];

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: isTimeSeries = chart.kind === "bar" || chart.kind === "stacked-bar" || chart.kind === "line";
  // Opt-in modern layer, read from the URL so the choice is bookmarkable and
  // never changes ledger semantics.
  $: modernLayer =
    typeof window !== "undefined" &&
    new URLSearchParams(window.location.search).get("chart_layer") === "modern";
  // Report metadata is semantic and English internally. Localize only the
  // presentation copy that this OC-specific chart adds at the rendering seam.
  $: displayedChart = chart.measure === "average-cost"
    ? { ...chart, title: t("averageCostEvolution"), unit: t("costPerUnit") }
    : chart;
</script>

{#if isTimeSeries && modernLayer}
  <ModernChart chart={displayedChart} {locale} />
{:else}
  <ReportChart chart={displayedChart} {locale} {hiddenCurrencies} />
{/if}
