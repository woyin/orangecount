<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import GenericReport from "./GenericReport.svelte";
  import type { TableReport } from "./types";

  export let entriesByType: [string, number][];
  export let postings: TableReport;
  export let locale = "en";
  export let renderCommas = false;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: total = entriesByType.reduce((sum, [, count]) => sum + count, 0);
</script>

<div class="left">
  <h3>{t("postingsPerAccount")}</h3>
  <GenericReport report={postings} title="" {locale} {renderCommas} />
</div>

<div class="left">
  <h3>{t("entriesPerType")}</h3>
  <table class="entries-by-type">
    <thead>
      <tr><th>{t("type")}</th><th class="num">{t("entriesCount")}</th></tr>
    </thead>
    <tbody>
      {#each entriesByType as [type, count] (type)}
        <tr><td>{type}</td><td class="num">{count}</td></tr>
      {/each}
    </tbody>
    <tfoot>
      <tr><td>{t("total")}</td><td class="num">{total}</td></tr>
    </tfoot>
  </table>
</div>

<style>
  .entries-by-type .num {
    text-align: right;
  }

  .entries-by-type tfoot td {
    font-weight: 500;
  }
</style>
