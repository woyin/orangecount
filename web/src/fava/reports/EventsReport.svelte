<script lang="ts">
  import ScatterPlot from "../charts/ScatterPlot.svelte";
  import { translations, type Locale } from "../../translations";
  import EventTable from "./EventTable.svelte";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  // Fava groups the events page by event type with one sortable table per
  // group; the adapter delivers a flat table.
  $: groups = (() => {
    const byType = new Map<string, { date: string; description: string }[]>();
    for (const row of report.rows) {
      const type = typeof row.type === "string" ? row.type : "";
      const list = byType.get(type) ?? [];
      list.push({
        date: typeof row.date === "string" ? row.date : "",
        description: typeof row.value === "string" ? row.value : String(row.value ?? ""),
      });
      byType.set(type, list);
    }
    return [...byType.entries()];
  })();

  $: chartEvents = report.rows.map((row) => ({
    date: typeof row.date === "string" ? row.date : "",
    type: typeof row.type === "string" ? row.type : "",
    description: typeof row.value === "string" ? row.value : String(row.value ?? ""),
  }));

  // Upstream hides charts behind the `charts=false` URL parameter; the shell
  // router has no URL-parameter plumbing for it, so the toggle is local.
  let showCharts = true;
</script>

{#if groups.length}
  <div class="flex-row">
    <span class="spacer"></span>
    <button type="button" class="show-charts" on:click={() => (showCharts = !showCharts)}>{showCharts ? "▼" : "◀"}</button>
  </div>
  {#if showCharts}
    <ScatterPlot events={chartEvents} />
    <div class="chart-switcher">
      <button type="button" class="unset selected">{t("events")}</button>
    </div>
  {/if}

  {#each groups as [type, events] (type)}
    <div class="left">
      <h3>{t("eventHeading")} {type}</h3>
      <EventTable {events} {locale} />
    </div>
  {/each}
{:else}
  <p>{t("noEvents")}</p>
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

  .chart-switcher button.selected,
  .chart-switcher button:hover {
    color: var(--text-color-lighter);
  }
</style>
