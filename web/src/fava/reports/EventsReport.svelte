<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  // Fava groups the events page by event type with one table per group,
  // sorted by date descending; the adapter delivers a flat table.
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
    for (const list of byType.values()) {
      list.sort((a, b) => b.date.localeCompare(a.date));
    }
    return [...byType.entries()];
  })();
</script>

{#if groups.length}
  {#each groups as [type, events] (type)}
    <div class="left">
      <h3>{t("eventHeading")} {type}</h3>
      <table class="events-table">
        <thead>
          <tr><th>{t("date")}</th><th>{t("description")}</th></tr>
        </thead>
        <tbody>
          {#each events as event (event.date + event.description)}
            <tr><td>{event.date}</td><td>{event.description}</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/each}
{:else}
  <p>{t("noEvents")}</p>
{/if}

<style>
  .events-table {
    max-width: 40rem;
  }
</style>
