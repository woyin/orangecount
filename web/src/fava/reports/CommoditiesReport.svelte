<script lang="ts">
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
  // price table per pair plus a line chart; the chart is still out of scope
  // (S1), so only the tables are rendered here.
  $: pairs = (() => {
    const byPair = new Map<string, PriceRow[]>();
    for (const row of report.rows) {
      const base = typeof row.currency === "string" ? row.currency : "";
      const quote = typeof row.quote_currency === "string" ? row.quote_currency : "";
      const key = `${base} / ${quote}`;
      const list = byPair.get(key) ?? [];
      list.push({
        date: typeof row.date === "string" ? row.date : "",
        amount: row.amount as DecimalWire | undefined,
      });
      byPair.set(key, list);
    }
    return [...byPair.entries()];
  })();
</script>

{#if pairs.length}
  {#each pairs as [pair, prices] (pair)}
    <div class="left">
      <h3>{pair}</h3>
      <PriceTable {prices} {locale} {renderCommas} />
    </div>
  {/each}
{:else}
  <p>{t("noPrices")}</p>
{/if}

