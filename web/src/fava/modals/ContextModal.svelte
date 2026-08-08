<script lang="ts">
  import { onMount } from "svelte";
  import type { AdapterClient } from "../adapter-client";
  import { formatAmount, type DecimalWire } from "../reports/types";
  import { translations, type Locale } from "../../translations";

  export let adapter: AdapterClient;
  export let locale = "en";

  interface JournalAmount {
    number: DecimalWire;
    currency: string;
  }

  interface EntryContextPayload {
    entry: {
      type: string;
      date: string;
      file?: string;
      span?: string;
      payee?: string;
      narration?: string;
      account?: string;
      flag?: string;
    };
    source_slice: string;
    sha256sum: string;
    balances_before?: Record<string, JournalAmount[]>;
    balances_after?: Record<string, JournalAmount[]>;
  }

  let shown = false;
  let entryHash = "";
  let context: EntryContextPayload | null = null;
  let error = "";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function sync() {
    const hash = window.location.hash;
    const next = hash.startsWith("#context-") ? hash.slice("#context-".length) : "";
    if (next !== entryHash) {
      entryHash = next;
      context = null;
      error = "";
      if (entryHash) void load();
    }
    shown = entryHash !== "";
  }

  async function load() {
    try {
      context = (await adapter.load("entry-context", { entry_hash: entryHash })) as EntryContextPayload;
    } catch (value) {
      error = value instanceof Error ? value.message : "The entry context could not be loaded.";
    }
  }

  function close() {
    window.history.replaceState({}, "", window.location.pathname + window.location.search);
    shown = false;
    entryHash = "";
    context = null;
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      close();
    }
  }

  function sourceHref(path: string): string {
    return `/source?path=${encodeURIComponent(path)}`;
  }

  function amountText(amount: JournalAmount): string {
    return `${formatAmount(amount.number)} ${amount.currency}`;
  }

  function balanceLines(balances?: Record<string, JournalAmount[]>): string[] {
    if (!balances) return [];
    const lines: string[] = [];
    for (const account of Object.keys(balances).sort()) {
      const amounts = (balances[account] ?? []).map(amountText).join(", ");
      if (amounts) lines.push(`${account}: ${amounts}`);
    }
    return lines;
  }

  onMount(() => {
    sync();
    window.addEventListener("hashchange", sync);
    document.addEventListener("keydown", onKeydown);
    return () => {
      window.removeEventListener("hashchange", sync);
      document.removeEventListener("keydown", onKeydown);
    };
  });
</script>

{#if shown}
  <div class="context-backdrop" role="presentation" on:click={close}>
    <div class="context-modal" role="dialog" aria-modal="true" aria-label={t("context")} on:click|stopPropagation>
      <h3>{t("context")}</h3>
      {#if error}
        <p class="error" role="alert">{error}</p>
      {:else if context}
        <p class="location">
          {#if context.entry.file}
            <a href={sourceHref(context.entry.file)}>{context.entry.file}</a>
          {/if}
          {#if context.entry.span}<span class="span">{context.entry.span}</span>{/if}
        </p>
        <p class="summary">
          {context.entry.date}
          {#if context.entry.flag}{context.entry.flag}{/if}
          {#if context.entry.payee}{context.entry.payee}{/if}
          {#if context.entry.narration}"{context.entry.narration}"{/if}
          {#if context.entry.account}{context.entry.account}{/if}
        </p>
        {#if balanceLines(context.balances_before).length || balanceLines(context.balances_after).length}
          <dl class="balances">
            {#if balanceLines(context.balances_before).length}
              <dt>Balances before</dt>
              {#each balanceLines(context.balances_before) as line (line)}<dd>{line}</dd>{/each}
            {/if}
            {#if balanceLines(context.balances_after).length}
              <dt>Balances after</dt>
              {#each balanceLines(context.balances_after) as line (line)}<dd>{line}</dd>{/each}
            {/if}
          </dl>
        {/if}
        <pre class="source">{context.source_slice}</pre>
      {:else}
        <p class="loading">{t("loading")}</p>
      {/if}
    </div>
  </div>
{/if}

<style>
  .context-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-index-floating-ui);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 10vh;
    background: var(--overlay-wrapper-background);
  }

  .context-modal {
    width: min(46rem, 92vw);
    max-height: 75vh;
    padding: 1em 1.25em;
    overflow-y: auto;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
    box-shadow: var(--box-shadow-overlay);
  }

  .context-modal h3 {
    margin: 0 0 0.5em;
    font-size: 1.1em;
    font-weight: bold;
  }

  .context-modal a {
    color: var(--link-color);
  }

  .location .span {
    margin-left: 0.5em;
    color: var(--text-color-lightest);
  }

  .summary {
    margin-bottom: 0.5em;
  }

  .balances {
    margin: 0 0 0.5em;
    padding: 0.5em;
    font-family: var(--font-family-editor);
    background: var(--sidebar-background);
    border: 1px solid var(--border);
  }

  .balances dt {
    font-weight: bold;
    margin-top: 0.25em;
  }

  .balances dt:first-child {
    margin-top: 0;
  }

  .balances dd {
    margin: 0 0 0 1em;
  }

  .source {
    padding: 0.5em;
    overflow-x: auto;
    font-family: var(--font-family-editor);
    white-space: pre;
    background: var(--sidebar-background);
    border: 1px solid var(--border);
  }

  .error {
    color: var(--error);
  }

  .loading {
    color: var(--text-color-lightest);
  }
</style>
