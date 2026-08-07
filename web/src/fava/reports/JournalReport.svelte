<script lang="ts">
  import { keyboardShortcut, type KeySpec } from "../keyboard-shortcuts";
  import { DateColumn, Sorter, StringColumn, type SortColumn, type SortOrder } from "../sort/index";
  import { formatAmount, type JournalAmount, type JournalEntry, type JournalReport } from "./types";

  export let report: JournalReport;
  export let renderCommas = false;
  /** Rendered on the account page, which adds a running-balance column. */
  export let runningBalances: Map<JournalEntry, string> | null = null;
  /** Account filter in effect: swaps the Price column for Fava's Change. */
  export let accountFilter = "";

  // Fava drives journal visibility entirely from `show-*` classes on the list,
  // so the chips only toggle class names and never re-request the report. The
  // defaults match Fava's own initial set.
  const chips: { label: string; cls: string; title: string; shortcut: KeySpec; children?: string[] }[] = [
    { label: "Open", cls: "show-open", title: "Toggle Open entries", shortcut: "s o" },
    { label: "Close", cls: "show-close", title: "Toggle Close entries", shortcut: "s c" },
    { label: "Transaction", cls: "show-transaction", title: "Toggle Transaction entries", shortcut: "s t", children: ["show-cleared", "show-pending", "show-other"] },
    { label: "*", cls: "show-cleared", title: "Cleared transactions", shortcut: "t c" },
    { label: "!", cls: "show-pending", title: "Pending transactions", shortcut: "t p" },
    { label: "x", cls: "show-other", title: "Other transactions", shortcut: "t o" },
    { label: "Balance", cls: "show-balance", title: "Toggle Balance entries", shortcut: "s b" },
    { label: "Note", cls: "show-note", title: "Toggle Note entries", shortcut: "s n" },
    { label: "Document", cls: "show-document", title: "Toggle Document entries", shortcut: "s d" },
    { label: "Pad", cls: "show-pad", title: "Toggle Pad entries", shortcut: "s p" },
    { label: "Query", cls: "show-query", title: "Toggle Query entries", shortcut: "s q" },
    { label: "Custom", cls: "show-custom", title: "Toggle Custom entries", shortcut: "s C" },
    { label: "Metadata", cls: "show-metadata", title: "Toggle metadata", shortcut: "m" },
    { label: "Postings", cls: "show-postings", title: "Toggle postings", shortcut: "p" },
  ];
  let active = new Set([
    "show-transaction", "show-cleared", "show-pending",
    "show-balance", "show-note", "show-document",
    "show-query", "show-custom",
  ]);
  function toggleChip(chip: (typeof chips)[number]) {
    const next = new Set(active);
    const flip = (cls: string) => {
      if (next.has(cls)) next.delete(cls); else next.add(cls);
    };
    flip(chip.cls);
    // Fava also toggles all subtype entries together with their supertype.
    chip.children?.forEach(flip);
    active = next;
  }
  $: listClasses = ["flex-table", "journal", ...active].join(" ");

  // Per-entry expansion, matching Fava's `.show-full-entry` row modifier.
  let expanded = new Set<JournalEntry>();
  function toggleEntry(entry: JournalEntry) {
    const next = new Set(expanded);
    if (next.has(entry)) next.delete(entry); else next.add(entry);
    expanded = next;
  }

  function flagClass(entry: JournalEntry): string {
    if (entry.type !== "transaction") return "";
    if (entry.flag === "*") return "cleared";
    if (entry.flag === "!") return "pending";
    return "other";
  }

  function amountText(amount: JournalAmount | undefined): string {
    if (!amount) return "";
    return `${formatAmount(amount.number, renderCommas)} ${amount.currency}`;
  }

  function changeText(change: JournalAmount[] | undefined): string {
    if (!change?.length) return "";
    return change.map((amount) => amountText(amount)).join(", ");
  }

  function accountHref(account: string): string {
    return `/account/${encodeURIComponent(account)}`;
  }

  /** The description cell differs per directive type, the way Fava's does. */
  function describe(entry: JournalEntry): string {
    switch (entry.type) {
      case "open": return entry.extra?.currencies ?? "";
      case "pad": return entry.extra?.source_account ?? "";
      case "query": return entry.extra?.query ?? "";
      case "custom": return entry.extra?.values ?? "";
      case "event": return entry.extra?.value ?? "";
      default: return entry.narration ?? "";
    }
  }

  // Fava persists the journal sort as a [column, order] tuple in
  // localStorage ("journal-sort-order") and defaults to newest first.
  type JournalSortColumn = "date" | "flag" | "narration";
  const sortColumns: Record<JournalSortColumn, SortColumn<JournalEntry>> = {
    date: new DateColumn<JournalEntry>("date"),
    flag: new StringColumn<JournalEntry>("flag", (entry) => entry.flag ?? ""),
    narration: new StringColumn<JournalEntry>("narration", (entry) => `${entry.payee ?? ""} ${describe(entry)}`.trim()),
  };

  function storedSort(): [JournalSortColumn, SortOrder] {
    try {
      const parsed = JSON.parse(localStorage.getItem("journal-sort-order") || "") as unknown;
      if (Array.isArray(parsed) && parsed[0] in sortColumns && (parsed[1] === "asc" || parsed[1] === "desc")) {
        return [parsed[0] as JournalSortColumn, parsed[1]];
      }
    } catch {
      // storage is optional; fall through to the default
    }
    return ["date", "desc"];
  }

  const [initialSortColumn, initialSortOrder] = storedSort();
  let sorter = new Sorter(sortColumns[initialSortColumn], initialSortOrder);

  function setSortColumn(column: JournalSortColumn) {
    sorter = sorter.switchColumn(sortColumns[column]);
    try {
      localStorage.setItem("journal-sort-order", JSON.stringify([column, sorter.order]));
    } catch {
      // storage is optional; sorting still works in-memory
    }
  }

  $: sortedEntries = sorter.sort(report.entries);
</script>

<form class="flex-row journal-chips">
  {#each chips as chip (chip.cls)}
    <button
      type="button"
      class:inactive={!active.has(chip.cls)}
      title={chip.title}
      aria-pressed={active.has(chip.cls)}
      use:keyboardShortcut={chip.shortcut}
      onclick={() => toggleChip(chip)}
    >{chip.label}</button>
  {/each}
  <span class="spacer"></span>
  <a class="button" href="/api/v1/reports/journal?format=csv">Export CSV</a>
</form>

<ol class={listClasses}>
  <li class="head">
    <p>
      <button
        type="button"
        class="datecell unset"
        data-order={sorter.column.name === "date" ? sorter.order : undefined}
        onclick={() => setSortColumn("date")}
      >Date</button>
      <button
        type="button"
        class="flag unset"
        data-order={sorter.column.name === "flag" ? sorter.order : undefined}
        onclick={() => setSortColumn("flag")}
      >F</button>
      <button
        type="button"
        class="description unset"
        data-order={sorter.column.name === "narration" ? sorter.order : undefined}
        onclick={() => setSortColumn("narration")}
      >Payee/Narration</button>
      <span class="num">Units</span>
      <span class="num">Cost</span>
      <span class="num">{runningBalances ? "Balance" : accountFilter ? "Change" : "Price"}</span>
    </p>
  </li>
  {#each sortedEntries as entry, index (entry.type + entry.date + index)}
    <li class="{entry.type} {flagClass(entry)}" class:show-full-entry={expanded.has(entry)}>
      <p>
        <span class="datecell">{entry.date}</span>
        <span class="flag">{entry.flag ?? ""}</span>
        <span class="description">
          {#if entry.account}
            <a href={accountHref(entry.account)}>{entry.account}</a>
          {/if}
          {#if entry.payee}
            <strong class="payee">{entry.payee}</strong><span class="separator"></span>
          {/if}
          {describe(entry)}
          {#each entry.tags ?? [] as tag (tag)}<span class="tag">#{tag}</span>{/each}
          {#each entry.links ?? [] as link (link)}<span class="link">^{link}</span>{/each}
          {#each entry.filenames ?? [] as filename (filename)}<span class="filename">{filename}</span>{/each}
          {#if entry.entry_hash}
            <a class="context-link" href="#context-{entry.entry_hash}" title="Context" aria-label="Context">⋮</a>
          {/if}
        </span>
        {#if entry.postings?.length}
          <!-- Fava shows one dot per posting; clicking expands the entry. -->
          <span
            class="indicators"
            role="button"
            tabindex="0"
            title="Toggle postings"
            onclick={() => toggleEntry(entry)}
            onkeydown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); toggleEntry(entry); } }}
          >{#each entry.postings as posting, dot (dot)}<span class={posting.flag === "!" ? "pending" : ""}></span>{/each}</span>
        {:else}
          <span class="indicators"></span>
        {/if}
        {#if entry.amount}
          <span class="num bal" title={entry.amount.currency}>{amountText(entry.amount)}</span>
          <span class="change num"></span>
          <span class="change num">{accountFilter ? changeText(entry.change) : ""}</span>
        {:else if runningBalances}
          <span class="num"></span>
          <span class="num"></span>
          <span class="num">{runningBalances.get(entry) ?? ""}</span>
        {:else if accountFilter}
          <span class="num"></span>
          <span class="num"></span>
          <span class="num change">{changeText(entry.change)}</span>
        {/if}
      </p>
      {#if entry.postings?.length}
        <ul class="postings">
          {#each entry.postings as posting, postingIndex (posting.account + postingIndex)}
            <li>
              <p>
                <span class="datecell"></span>
                <span class="flag">{posting.flag ?? ""}</span>
                <span class="description"><a href={accountHref(posting.account)}>{posting.account}</a></span>
                <span class="num">{amountText(posting.units)}</span>
                <span class="num">{amountText(posting.cost)}</span>
                <span class="num">{amountText(posting.price)}</span>
              </p>
            </li>
          {/each}
        </ul>
      {/if}
      {#if entry.metadata?.length}
        <dl class="metadata">
          {#each entry.metadata as meta (meta.key)}
            <dt>{meta.key}:</dt>
            <dd>{meta.value}</dd>
          {/each}
        </dl>
      {/if}
    </li>
  {/each}
</ol>

<style>
  .journal-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .journal-chips .spacer {
    flex: 1;
  }

  .context-link {
    padding: 0 0.35em;
    font-weight: bold;
    color: var(--text-color-lightest);
    text-decoration: none;
  }

  .context-link:hover {
    color: var(--link-color);
  }

  .head button {
    position: relative;
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
    background: transparent;
    border: 0;
  }
</style>
