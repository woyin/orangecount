<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { formatAmount, type DecimalWire, type TableReport } from "./types";

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

  // Fava groups the commodities page by base/quote pair with one price table
  // per pair (newest first) plus a line chart; the chart is still out of
  // scope (S1), so only the tables are rendered here.
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
    for (const list of byPair.values()) {
      list.sort((a, b) => b.date.localeCompare(a.date));
    }
    return [...byPair.entries()];
  })();
</script>

{#if pairs.length}
  {#each pairs as [pair, prices] (pair)}
    <div class="left">
      <h3>{pair}</h3>
      <table class="prices-table">
        <thead>
          <tr><th>{t("date")}</th><th class="num">{t("price")}</th></tr>
        </thead>
        <tbody>
          {#each prices as price (price.date)}
            <tr>
              <td>{price.date}</td>
              <td class="num">{formatAmount(price.amount, renderCommas)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/each}
{:else}
  <p>{t("noPrices")}</p>
{/if}

<style>
  .prices-table {
    max-width: 30rem;
  }

  .prices-table .num {
    text-align: right;
  }
</style>
