<script lang="ts">
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let account = "";

  interface Posting {
    account: string;
    units: unknown;
    currency: unknown;
    flag: unknown;
    file: unknown;
    span: unknown;
    running: string;
  }

  interface Group {
    key: string;
    date: unknown;
    kind: unknown;
    flag: unknown;
    payee: unknown;
    narration: unknown;
    tags: unknown;
    links: unknown;
    postings: Posting[];
  }

  let expanded = new Set<string>();
  let kindFilter = "";

  function text(value: unknown): string {
    if (value && typeof value === "object" && "display" in value && typeof value.display === "string") return value.display;
    if (Array.isArray(value)) return value.join(", ");
    if (value && typeof value === "object") return JSON.stringify(value);
    return value == null ? "" : String(value);
  }

  function groupRows(rows: Record<string, unknown>[]): Group[] {
    const groups: Group[] = [];
    const byKey = new Map<string, Group>();
    const running = new Map<string, number>();
    for (const row of rows) {
      const key = [row.date, row.file, row.span, row.payee, row.narration, row.kind].map(text).join("\u0000");
      let group = byKey.get(key);
      if (!group) {
        group = { key, date: row.date, kind: row.kind, flag: row.flag, payee: row.payee, narration: row.narration, tags: row.tags, links: row.links, postings: [] };
        byKey.set(key, group);
        groups.push(group);
      }
      const currency = text(row.currency);
      const amount = Number(text(row.units));
      if (account && text(row.account) === account && currency && Number.isFinite(amount)) running.set(currency, (running.get(currency) || 0) + amount);
      group.postings.push({ account: text(row.account), units: row.units, currency: row.currency, flag: row.flag, file: row.file, span: row.span, running: account && text(row.account) === account && currency ? String(running.get(currency) || 0) : "" });
    }
    return groups;
  }

  $: filteredRows = kindFilter ? report.rows.filter((row) => text(row.kind).toLowerCase() === kindFilter.toLowerCase() || text(row.flag) === kindFilter) : report.rows;
  $: groups = groupRows(filteredRows);

  function isExpanded(key: string): boolean {
    return expanded.has(key);
  }

  function toggle(key: string) {
    const next = new Set(expanded);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    expanded = next;
  }
</script>

<div class="headerline">
  <h2>Journal</h2>
  <span class="muted">{groups.length} transactions · {filteredRows.length} postings</span>
  <a class="button" href="/api/v1/reports/journal?format=csv">Export CSV</a>
</div>
<div class="journal-filters" aria-label="Entry type filters">
  <button class:active={!kindFilter} type="button" onclick={() => kindFilter = ""}>All</button>
  {#each ["transaction", "open", "close", "balance", "note", "document", "query", "custom", "pad", "price"] as kind (kind)}
    <button class:active={kindFilter === kind} type="button" onclick={() => kindFilter = kind}>{kind}</button>
  {/each}
</div>
<div class="journal-table-wrapper">
  <table class="journal-table">
    <thead>
      <tr><th>Date</th><th>Transaction</th><th>Account</th><th class="num">Amount</th><th>Currency</th>{#if account}<th class="num">Running balance</th>{/if}</tr>
    </thead>
    <tbody>
      {#each groups as group (group.key)}
        <tr class="journal-transaction-row">
          <th scope="row">{text(group.date)}</th>
          <td colspan={account ? 5 : 4}>
            <button class="journal-transaction-toggle link" type="button" aria-expanded={isExpanded(group.key)} onclick={() => toggle(group.key)}>
              {isExpanded(group.key) ? "▾" : "▸"} {text(group.flag)} {text(group.payee)} {text(group.narration)}
            </button>
            {#if text(group.tags)} <span class="journal-meta">#{text(group.tags)}</span>{/if}
            {#if text(group.links)} <span class="journal-meta">^{text(group.links)}</span>{/if}
          </td>
        </tr>
        {#if isExpanded(group.key)}
          {#each group.postings as posting, index (group.key + index)}
            <tr class="journal-posting-row">
              <td></td>
              <td></td>
              <td><a href={`/account/${encodeURIComponent(posting.account)}`}>{posting.account}</a></td>
              <td class="num">{text(posting.units)}</td>
              <td>{text(posting.currency)}</td>
              {#if account}<td class="num">{posting.running}</td>{/if}
            </tr>
          {/each}
        {/if}
      {:else}
        <tr><td colspan={account ? 6 : 5}>No journal entries.</td></tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .journal-filters { display: flex; flex-wrap: wrap; gap: .25rem; margin-bottom: .75rem; }
  .journal-filters button { padding: .2rem .5rem; color: var(--text-color); background: var(--background-darker); border: 1px solid var(--border); }
  .journal-filters button.active { color: var(--button-color); background: var(--button-background); }

  .muted,
  .journal-meta {
    color: var(--text-color-lightest);
  }

  .journal-meta {
    margin-left: 0.5rem;
  }

  .journal-table-wrapper {
    overflow-x: auto;
  }

  .journal-table {
    min-width: 48rem;
  }

  .journal-transaction-row th,
  .journal-transaction-row td {
    background-color: var(--background-darker);
  }

  .journal-posting-row td {
    border-top: 0;
  }
</style>
