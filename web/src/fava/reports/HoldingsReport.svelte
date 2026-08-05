<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { routeHref } from "../router.mjs";
  import GenericReport from "./GenericReport.svelte";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let route: string;
  export let locale = "en";
  export let renderCommas = false;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  // The tab set mirrors Fava's holdings headerline (one h3 per aggregation,
  // active tab plain, the rest links). OrangeCount keeps its two additional
  // OC-extension aggregations here rather than hiding data the adapter serves.
  const tabs = [
    { route: "holdings", key: "holdings" },
    { route: "holdings_by_account", key: "holdingsByAccount" },
    { route: "holdings_by_currency", key: "holdingsByCurrency" },
    { route: "holdings_by_root_account", key: "holdingsByRootAccount" },
    { route: "holdings_by_commodity", key: "holdingsByCommodity" },
  ];

  $: aggregation = route.startsWith("holdings_by_") ? route.slice("holdings_".length) : "";
  $: csvHref = aggregation
    ? `/api/v1/reports/holdings?format=csv&aggregation=${encodeURIComponent(aggregation)}`
    : "/api/v1/reports/holdings?format=csv";

  function columnLabel(column: string): string {
    return column.split("_").map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part)).join(" ");
  }

  // The adapter emits snake_case field names; Fava's tables show readable
  // headers, so rename both the column list and each row's keys.
  $: labeled = (() => {
    const rename = new Map(report.columns.map((column) => [column, columnLabel(column)]));
    return {
      ...report,
      columns: report.columns.map((column) => rename.get(column) ?? column),
      rows: report.rows.map((row) =>
        Object.fromEntries(Object.entries(row).map(([key, value]) => [rename.get(key) ?? key, value])),
      ),
    };
  })();
</script>

<div class="headerline">
  {#each tabs as tab (tab.route)}
    <h3>
      {#if tab.route === route}
        {t(tab.key)}
      {:else}
        <a href={routeHref(tab.route)}>{t(tab.key)}</a>
      {/if}
    </h3>
  {/each}
  <a class="button" href={csvHref}>{t("exportCSV")}</a>
</div>
<GenericReport report={labeled} title="" {locale} {renderCommas} />
