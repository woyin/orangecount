<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { formatAmount, type DecimalWire } from "./types";
  import { DateColumn, NumberColumn, Sorter, type SortColumn } from "../sort/index";
  import SortHeader from "../sort/SortHeader.svelte";

  interface PriceRow {
    date: string;
    amount: DecimalWire | undefined;
  }

  export let prices: PriceRow[];
  export let locale = "en";
  export let renderCommas = false;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function priceValue(row: PriceRow): number {
    const display = row.amount?.display ?? "";
    const parsed = Number.parseFloat(display.replace(/,/g, ""));
    return Number.isFinite(parsed) ? parsed : 0;
  }

  $: columns = [
    new DateColumn<PriceRow>(t("date")),
    new NumberColumn<PriceRow>(t("price"), priceValue),
  ] as SortColumn<PriceRow>[];
  $: sorter = new Sorter<PriceRow>(columns[0], "desc");
  $: sorted = sorter.sort(prices);
</script>

<table class="prices-table">
  <thead>
    <tr>
      <SortHeader bind:sorter column={columns[0]} />
      <SortHeader bind:sorter column={columns[1]} numeric />
    </tr>
  </thead>
  <tbody>
    {#each sorted as price (price.date)}
      <tr>
        <td>{price.date}</td>
        <td class="num">{formatAmount(price.amount, renderCommas)}</td>
      </tr>
    {/each}
  </tbody>
</table>

<style>
  .prices-table {
    max-width: 30rem;
  }

  .prices-table .num {
    text-align: right;
  }
</style>
