<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { NumberColumn, Sorter, StringColumn, UnsortedColumn, type SortColumn } from "../sort/index";
  import SortHeader from "../sort/SortHeader.svelte";
  import { routeHref } from "../router.mjs";
  import GenericReport from "./GenericReport.svelte";
  import type { TableReport } from "./types";

  interface UpdateActivityRow {
    account: string;
    last_entry_date: string;
    entry_hash: string;
    balances: Record<string, string>;
  }

  export let entriesByType: [string, number][];
  export let postings: TableReport;
  export let updateActivity: UpdateActivityRow[] = [];
  export let locale = "en";
  export let renderCommas = false;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: total = entriesByType.reduce((sum, [, count]) => sum + count, 0);

  $: entryColumns = [
    new StringColumn<[string, number]>(t("type"), (row) => row[0]),
    new NumberColumn<[string, number]>(t("entriesCount"), (row) => row[1]),
  ] as SortColumn<[string, number]>[];
  $: entrySorter = new Sorter<[string, number]>(entryColumns[0], "asc");
  $: sortedEntries = entrySorter.sort(entriesByType);

  // Mirrors Fava's UpdateActivity table: account link, last-entry link into
  // the context modal, and an unsorted multi-currency balance column.
  $: activityColumns = [
    new StringColumn<UpdateActivityRow>(t("account"), (row) => row.account),
    new StringColumn<UpdateActivityRow>(t("updateActivityLastEntry"), (row) => row.last_entry_date),
    new UnsortedColumn<UpdateActivityRow>(t("accountBalance")),
  ] as SortColumn<UpdateActivityRow>[];
  $: activitySorter = new Sorter<UpdateActivityRow>(activityColumns[0], "asc");
  $: sortedActivity = activitySorter.sort(updateActivity);
</script>

<div class="left">
  <h3>{t("postingsPerAccount")}</h3>
  <GenericReport report={postings} title="" {locale} {renderCommas} />
</div>

<div class="left">
  <h3>{t("entriesPerType")}</h3>
  <table class="entries-by-type">
    <thead>
      <tr>
        <SortHeader bind:sorter={entrySorter} column={entryColumns[0]} />
        <SortHeader bind:sorter={entrySorter} column={entryColumns[1]} numeric />
      </tr>
    </thead>
    <tbody>
      {#each sortedEntries as [type, count] (type)}
        <tr><td>{type}</td><td class="num">{count}</td></tr>
      {/each}
    </tbody>
    <tfoot>
      <tr><td>{t("total")}</td><td class="num">{total}</td></tr>
    </tfoot>
  </table>
</div>

{#if updateActivity.length}
  <div class="left">
    <h3>{t("updateActivity")}</h3>
    <table class="update-activity">
      <thead>
        <tr>
          <SortHeader bind:sorter={activitySorter} column={activityColumns[0]} />
          <SortHeader bind:sorter={activitySorter} column={activityColumns[1]} />
          <SortHeader bind:sorter={activitySorter} column={activityColumns[2]} />
        </tr>
      </thead>
      <tbody>
        {#each sortedActivity as row (row.account)}
          <tr>
            <td class="account"><a href={routeHref("account", { account: row.account })}>{row.account}</a></td>
            <td><a href={`#context-${row.entry_hash}`}>{row.last_entry_date}</a></td>
            <td class="num">
              {#each Object.entries(row.balances) as [currency, amount] (currency)}
                <span>{amount} {currency}</span><br />
              {/each}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<style>
  .entries-by-type .num,
  .update-activity .num {
    text-align: right;
  }

  .entries-by-type tfoot td {
    font-weight: 500;
  }

  .update-activity td.account {
    white-space: nowrap;
  }
</style>
