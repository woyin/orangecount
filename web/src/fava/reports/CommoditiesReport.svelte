<script lang="ts">
  import LineChart from "../charts/LineChart.svelte";
  import { translations, type Locale } from "../../translations";
  import PriceTable from "./PriceTable.svelte";
  import type { DecimalWire, TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";
  export let renderCommas = false;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  interface PriceRow {
    date: string;
    amount: DecimalWire | undefined;
  }

  // Fava groups the commodities page by base/quote pair with one sortable
  // price table per pair and a switchable line chart per pair.
  $: pairs = (() => {
    const byPair = new Map<string, { base: string; quote: string; prices: PriceRow[] }>();
    for (const row of report.rows) {
      const base = typeof row.currency === "string" ? row.currency : "";
      const quote = typeof row.quote_currency === "string" ? row.quote_currency : "";
      const key = `${base} / ${quote}`;
      const entry = byPair.get(key) ?? { base, quote, prices: [] };
      entry.prices.push({
        date: typeof row.date === "string" ? row.date : "",
        amount: row.amount as DecimalWire | undefined,
      });
      byPair.set(key, entry);
    }
    return [...byPair.entries()];
  })();

  function numberValue(amount: DecimalWire | undefined): number {
    if (!amount?.display) return NaN;
    if (amount.display.includes("/")) {
      const [numerator, denominator] = amount.display.split("/").map(Number);
      return denominator ? numerator / denominator : NaN;
    }
    return Number(amount.display);
  }

  function chartPoints(prices: PriceRow[]) {
    return prices
      .map((price) => ({
        date: price.date,
        display: price.amount?.display ?? "",
        value: numberValue(price.amount),
      }))
      .filter((point) => point.date && Number.isFinite(point.value));
  }

  function tipFor(base: string, quote: string) {
    return (point: { date: string; display: string }) =>
      `1 ${base} = ${point.display} ${quote}\n${point.date}`;
  }

  // Upstream remembers the last active chart name across navigation; the shell
  // persists the selection to localStorage so it survives.
  let activePair = "";
  try {
    activePair = localStorage.getItem("commodities-active-pair") || "";
  } catch {
    // storage is optional; default to the first pair
  }
  $: active = pairs.find(([pair]) => pair === activePair) ?? pairs[0];
  $: try { localStorage.setItem("commodities-active-pair", active[0]); } catch { /* storage optional */ }

  // Upstream hides charts behind the `charts=false` URL parameter; the shell
  // router has no URL-parameter plumbing for it, so the toggle is local.
  let showCharts = true;

  // Upstream's lineChartMode store: "line" or "area", persisted across
  // navigation.
  let chartMode: "line" | "area" = "line";
  try {
    const stored = localStorage.getItem("commodities-chart-mode");
    if (stored === "line" || stored === "area") chartMode = stored;
  } catch {
    // storage is optional; default to line
  }
  function toggleMode() {
    chartMode = chartMode === "line" ? "area" : "line";
    try { localStorage.setItem("commodities-chart-mode", chartMode); } catch { /* storage optional */ }
  }
</script>

{#if pairs.length}
  <div class="flex-row">
    <span class="spacer"></span>
    <button type="button" class="chart-mode" on:click={toggleMode} title="Toggle line/area">{chartMode === "line" ? "Line" : "Area"}</button>
    <button type="button" class="show-charts" on:click={() => (showCharts = !showCharts)}>{showCharts ? "▼" : "◀"}</button>
  </div>
  {#if showCharts && active}
    <LineChart points={chartPoints(active[1].prices)} formatTip={tipFor(active[1].base, active[1].quote)} mode={chartMode} />
    <div class="chart-switcher">
      {#each pairs as [pair] (pair)}
        <button
          type="button"
          class="unset"
          class:selected={active && pair === active[0]}
          on:click={() => (activePair = pair)}
        >{pair}</button>
      {/each}
    </div>
  {/if}

  {#each pairs as [pair, { prices }] (pair)}
    <div class="left">
      <h3>{pair}</h3>
      <PriceTable {prices} {locale} {renderCommas} />
    </div>
  {/each}
{:else}
  <p>{t("noPrices")}</p>
{/if}

<style>
  .flex-row {
    display: flex;
    gap: var(--flex-gap, 0.5rem);
    margin-bottom: var(--flex-gap, 0.5rem);
  }

  .spacer {
    flex: 1;
  }

  .chart-switcher {
    margin-bottom: 1em;
    color: var(--text-color-lightest);
    text-align: center;
  }

  .chart-switcher button {
    padding: 0 0.5em;
  }

  .chart-switcher button + button {
    border-left: 1px solid var(--text-color-lighter);
  }

  .chart-switcher button.selected,
  .chart-switcher button:hover {
    color: var(--text-color-lighter);
  }
</style>
