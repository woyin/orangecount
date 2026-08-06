<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { DateColumn, Sorter, StringColumn, type SortColumn } from "../sort/index";
  import SortHeader from "../sort/SortHeader.svelte";

  interface EventRow {
    date: string;
    description: string;
  }

  export let events: EventRow[];
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: columns = [
    new DateColumn<EventRow>(t("date")),
    new StringColumn<EventRow>(t("description"), (event) => event.description),
  ] as SortColumn<EventRow>[];
  $: sorter = new Sorter<EventRow>(columns[0], "desc");
  $: sorted = sorter.sort(events);
</script>

<table class="events-table">
  <thead>
    <tr>
      {#each columns as column (column.name)}
        <SortHeader bind:sorter {column} />
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each sorted as event (event.date + event.description)}
      <tr><td>{event.date}</td><td>{event.description}</td></tr>
    {/each}
  </tbody>
</table>

<style>
  .events-table {
    max-width: 40rem;
  }
</style>
