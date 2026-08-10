<!-- This file is derived from Fava 1.30.12 (commit #aa7538e8971252c9efc52c8a516a3a77d604553f),
which is Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name> and
distributed under the MIT License. Adapted for OrangeCount; see NOTICE and
web/provenance-manifest.json. The MIT notice is reproduced here:

  Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name>

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in all
  copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE. -->

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
    { label: "Document", cls: "show-document", title: "Toggle Document entries", shortcut: "s d", children: ["show-discovered", "show-linked"] },
    { label: "D", cls: "show-discovered", title: "Documents with a #discovered tag", shortcut: "d d" },
    { label: "L", cls: "show-linked", title: "Documents with a #linked tag", shortcut: "d l" },
    { label: "Pad", cls: "show-pad", title: "Toggle Pad entries", shortcut: "s p" },
    { label: "Query", cls: "show-query", title: "Toggle Query entries", shortcut: "s q" },
    { label: "Custom", cls: "show-custom", title: "Toggle Custom entries", shortcut: "s C" },
    { label: "Metadata", cls: "show-metadata", title: "Toggle metadata", shortcut: "m" },
    { label: "Postings", cls: "show-postings", title: "Toggle postings", shortcut: "p" },
  ];
  let active = new Set([
    "show-transaction", "show-cleared", "show-pending",
    "show-balance", "show-note", "show-document",
    "show-discovered", "show-linked",
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

  function hasTag(entry: JournalEntry, tag: string): boolean {
    return (entry.tags ?? []).includes(tag);
  }

  // Upstream flag_to_type: the class is only rendered when a flag exists.
  function postingFlagClass(flag: string | undefined): string {
    if (!flag) return "";
    if (flag === "*") return "cleared";
    if (flag === "!") return "pending";
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

  // Mirrors upstream JournalTable's ondragenter: mark the row's description
  // cell as a document droptarget, deriving the account from the first
  // account link in the row (a transaction's first posting for transactions).
  function onDragEnter(event: DragEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const description = target?.closest("p > .description");
    const li = description?.closest("li");
    const transfer = event.dataTransfer;
    if (!(description instanceof Element) || !li || !transfer || !transfer.types.includes("Files")) return;
    const link = li.querySelector("a[href^='/account/']");
    if (!link) return;
    const account = decodeURIComponent(link.getAttribute("href")?.slice("/account/".length) ?? "");
    if (!account) return;
    description.setAttribute("data-account-name", account);
    description.classList.add("droptarget", "dragover");
    event.preventDefault();
  }
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

<ol class={listClasses} ondragenter={onDragEnter}>
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
    <li
      class="{entry.type} {flagClass(entry)}"
      class:linked={hasTag(entry, "linked")}
      class:discovered={hasTag(entry, "discovered")}
      class:show-full-entry={expanded.has(entry)}
    >
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
          {#if entry.type === "custom" && entry.custom_values?.length}
            <strong class="custom-type">{entry.narration}</strong>
            {#each entry.custom_values as cv (cv.dtype + cv.value)}
              {#if cv.dtype === "account"}<a href={accountHref(cv.value)}>{cv.value}</a>{:else}<span class="custom-value">{cv.value}</span>{/if}
            {/each}
          {:else}
            {describe(entry)}
          {/if}
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
          >
            {#each entry.metadata ?? [] as meta (meta.key)}<span class="metadata-indicator" title="{meta.key}: {meta.value}">{meta.key.slice(0, 2)}</span>{/each}
            {#each entry.postings as posting, dot (dot)}
              <span class={postingFlagClass(posting.flag)}></span>
              {#each posting.metadata ?? [] as meta (meta.key)}<span class="metadata-indicator" title="{meta.key}: {meta.value}">{meta.key.slice(0, 2)}</span>{/each}
            {/each}
          </span>
        {:else}
          <span class="indicators">
            {#each entry.metadata ?? [] as meta (meta.key)}<span class="metadata-indicator" title="{meta.key}: {meta.value}">{meta.key.slice(0, 2)}</span>{/each}
          </span>
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
            <li class={postingFlagClass(posting.flag)}>
              <p>
                <span class="datecell"></span>
                <span class="flag">{posting.flag ?? ""}</span>
                <span class="description"><a href={accountHref(posting.account)}>{posting.account}</a></span>
                <span class="num">{amountText(posting.units)}</span>
                <span class="num">{amountText(posting.cost)}</span>
                <span class="num">{amountText(posting.price)}</span>
              </p>
              {#if posting.metadata?.length}
                <dl class="metadata">
                  {#each posting.metadata as meta (meta.key)}
                    <dt>{meta.key}:</dt>
                    <dd>{meta.value}</dd>
                  {/each}
                </dl>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
      {#if entry.metadata?.length}
        <dl class="metadata">
          {#each entry.metadata as meta (meta.key)}
            <dt>{meta.key}:</dt>
            <!-- Fava links document metadata to its statement view; the shell
                 has no statement route, so a document key links to the
                 document file served by /documents/ instead. -->
            <dd>{#if meta.key.startsWith("document")}<a class="filename" data-remote target="_blank" rel="noopener noreferrer" href={`/documents/${encodeURIComponent(meta.value.replace(/^"|"$/g, ""))}`}>{meta.value.replace(/^"|"$/g, "")}</a>{:else}{meta.value}{/if}</dd>
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
