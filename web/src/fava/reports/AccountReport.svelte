<script lang="ts">
  import type { AdapterClient } from "../adapter-client";
  import { translations, type Locale } from "../../translations";
 import ChartView from "../charts/ChartView.svelte";
  import GenericReport from "./GenericReport.svelte";
  import JournalReport from "./JournalReport.svelte";
  import { parseJournalReport, parseTableReport, type JournalReport as JournalReportData, type TableReport } from "./types";

  export let adapter: AdapterClient;
  export let query: Record<string, string>;
  export let locale = "en";
  export let renderCommas = false;
 export let accountDetails: Record<string, { balance_string: string; close_date?: string; uptodate_status?: string; last_entry?: string }> = {};
 export let chartLayer: "parity" | "modern" = "parity";
 export let onChartLayer: (value: string) => void = () => {};

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: account = query.account || "";
  // Fava switches the account page between the balance tree, per-interval
  // changes, and per-interval balances with the `r` query parameter.
  $: reportType = query.r === "changes" || query.r === "balances" ? query.r : "journal";

  function intervalLabelFor(value: string): string {
    if (value === "quarter") return t("quarterly");
    if (value === "year") return t("yearly");
    return t("monthly");
  }

  $: intervalLabel = intervalLabelFor(query.interval || "month");

  // Fava renders the account title as a breadcrumb where every ancestor is a
  // link to that account level; only the leaf segment of each is shown.
  function ancestors(name: string): string[] {
    const result: string[] = [];
    let index = name.indexOf(":");
    while (index !== -1) {
      result.push(name.slice(0, index));
      index = name.indexOf(":", index + 1);
    }
    if (name !== "") result.push(name);
    return result;
  }

  function leaf(name: string): string {
    const parentEnd = name.lastIndexOf(":");
    return parentEnd > 0 ? name.slice(parentEnd + 1) : name;
  }

  function accountHref(name: string): string {
    return `/account/${encodeURIComponent(name)}`;
  }

  $: parts = ancestors(account);

  // Per-account up-to-date status (green = latest is a passing balance check,
  // yellow = latest is a transaction; red is unreachable under valid-only
  // serving). Displayed as a small dot beside the breadcrumb, like Fava.
  $: uptodate = accountDetails[account]?.uptodate_status ?? "";
  $: statusTitle = uptodate === "green" ? "The last entry is a passing balance check." : uptodate === "yellow" ? "The last entry is not a balance check." : "";

  let balance: TableReport | null = null;
  let intervals: TableReport | null = null;
  let journal: JournalReportData | null = null;
  let error = "";
  let requestKey = "";

  // The account route returns a chart alongside the balance table; render it
  // above the balance tree the way Fava shows the account chart.
  $: chart = balance?.chart ?? null;

  function sectionHref(mode: string): string {
    const params = new URLSearchParams();
    for (const key of ["time", "interval"]) {
      if (query[key]) params.set(key, query[key]);
    }
    if (mode) params.set("r", mode);
    const suffix = params.size ? `?${params.toString()}` : "";
    return `/account/${encodeURIComponent(account)}${suffix}`;
  }

  // The journal arrives newest-first, so the first entry's date is the most
  // recent activity inside the current filters.
  $: lastEntry = journal && journal.entries.length ? journal.entries[0].date : "";
  $: lastEntryHash = journal && journal.entries.length && journal.entries[0].entry_hash ? journal.entries[0].entry_hash : "";

  $: if (requestKey !== JSON.stringify(query)) {
    requestKey = JSON.stringify(query);
    void load(requestKey);
  }

  async function load(key: string) {
    try {
      const requests: Promise<unknown>[] = [
        adapter.load("account", query),
        adapter.load("journal", query),
      ];
      if (reportType !== "journal") {
        requests.push(adapter.load("account", { ...query, r: reportType }));
      }
      const values = await Promise.all(requests);
      if (key !== requestKey) return;
      balance = parseTableReport(values[0]);
      journal = parseJournalReport(values[1]);
      intervals = reportType !== "journal" ? parseTableReport(values[2]) : null;
      error = "";
    } catch (value) {
      if (key !== requestKey) return;
      error = value instanceof Error ? value.message : "The account report could not be loaded.";
    }
  }
</script>

{#if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else}
  <div class="headerline"><h2 class="account-breadcrumb"><span class="droptarget" data-account-name={account}>{#each parts as name, index (name)}<a href={accountHref(name)} title={name}>{leaf(name)}</a>{#if index < parts.length - 1}<span class="sep">:</span>{/if}{/each}{#if uptodate}<span class="status-indicator status-{uptodate}" title={statusTitle}></span>{/if}{#if lastEntry}<span class="last-activity">({t("lastEntry")} {#if lastEntryHash}<a href="#context-{lastEntryHash}">{lastEntry}</a>{:else}{lastEntry}{/if})</span>{/if}</span></h2></div>
  <div class="headerline sections">
    <h3>{#if reportType !== "journal"}<a href={sectionHref("")}>{t("accountBalance")}</a>{:else}{t("accountBalance")}{/if}</h3>
    <h3>{#if reportType !== "changes"}<a href={sectionHref("changes")}>{t("changes")} ({intervalLabel})</a>{:else}{t("changes")} ({intervalLabel}){/if}</h3>
    <h3>{#if reportType !== "balances"}<a href={sectionHref("balances")}>{t("balances")} ({intervalLabel})</a>{:else}{t("balances")} ({intervalLabel}){/if}</h3>
  </div>
  {#if reportType === "changes" || reportType === "balances"}
    {#if intervals}<GenericReport report={intervals} title={`${reportType === "changes" ? t("changes") : t("balances")} (${intervalLabel})`} {locale} {renderCommas} />{/if}
  {:else if balance}
   {#if chart}<ChartView {chart} {locale} {chartLayer} {onChartLayer} />{/if}
    <GenericReport report={balance} title="Balance" {locale} {renderCommas} />
  {/if}
  {#if journal}<JournalReport report={journal} {renderCommas} accountFilter={account} />{/if}
{/if}

<style>
  .account-breadcrumb .droptarget {
    padding: 0.35em 0.5em;
    margin: -0.35em -0.5em;
  }

  .account-breadcrumb a {
    color: unset;
  }

  .account-breadcrumb .sep {
    color: var(--text-color-lightest);
  }

  .account-breadcrumb .last-activity {
    display: inline-block;
    margin-left: 10px;
    font-size: 12px;
    font-weight: normal;
    opacity: 0.8;
  }

  .sections {
    display: flex;
    gap: 1.5em;
    align-items: baseline;
  }

  .sections h3 {
    margin: 0;
    font-size: 1em;
  }

  .sections a {
    color: var(--link-color);
  }

  .status-indicator {
    display: inline-block;
    width: 10px;
    height: 10px;
    margin-left: 10px;
    border-radius: 10px;
  }

  .status-green {
    background: var(--status-green, #2e7d32);
  }

  .status-yellow {
    background: var(--status-yellow, #fbc02d);
  }
</style>
