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
  import { translations, type Locale } from "../../translations";
  import { escape_for_regex } from "../lib/regex";
  import { keyboardShortcut } from "../keyboard-shortcuts";
  import AutocompleteInput from "./AutocompleteInput.svelte";
  import PageTitle from "./PageTitle.svelte";

  export let ledgerTitle: string;
  export let route: string;
  export let account = "";
  export let accounts: string[] = [];
  export let tags: string[] = [];
  export let links: string[] = [];
  export let payees: string[] = [];
  export let years: string[] = [];
  export let locale: string;
  export let time = "";
  export let accountFilter = "";
  export let filter = "";
  export let conversion = "at_cost";
  export let interval = "month";
  export let onNavigate: (href: string) => void;
  export let onReload: () => void;
  export let onTime: (value: string) => void;
  export let onAccount: (value: string) => void;
  export let onQuery: (value: string) => void;
  export let onConversion: (value: string) => void;
  export let onInterval: (value: string) => void;

  function t(key: string): string {
    return translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale][key] || translations.en[key] || key;
  }

  $: filterSuggestions = [
    ...tags.map((tag) => `#${tag}`),
    ...links.map((link) => `^${link}`),
    ...payees.map((payee) => `payee:"${escape_for_regex(payee)}"`),
  ];

  function valueExtractor(value: string, input: HTMLInputElement): string {
    const match = /\S*$/.exec(value.slice(0, input.selectionStart ?? undefined));
    return match?.[0] ?? value;
  }

  function valueSelector(value: string, input: HTMLInputElement): string {
    const selectionStart = input.selectionStart ?? 0;
    const match = /\S*$/.exec(input.value.slice(0, selectionStart));
    const matchLength = match?.[0]?.length;
    return matchLength !== undefined
      ? `${input.value.slice(0, selectionStart - matchLength)}${value}${input.value.slice(selectionStart)}`
      : value;
  }

  let timeDraft = time;
  $: timeDraft = time;
  let accountDraft = accountFilter;
  $: accountDraft = accountFilter;
  let filterDraft = filter;
  $: filterDraft = filter;
</script>

<header>
  <!-- Fava renders its ledger icon before the title; it is part of the header
       composition the baseline captures. -->
  <svg class="header-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="25" viewBox="0 0 24 27" aria-hidden="true">
    <path fill="#FFF" d="M14 17.854c0 1.537-.498 2.86-1.493 3.97-.995 1.11-2.288 1.796-3.88 2.057v2.64c0 .14-.044.26-.134.35-.09.09-.205.14-.345.14H6.122c-.13 0-.242-.04-.337-.14-.095-.09-.143-.21-.143-.34v-2.64c-.66-.09-1.298-.24-1.913-.46-.62-.22-1.13-.44-1.53-.67-.4-.22-.77-.46-1.11-.72-.34-.26-.58-.45-.7-.57-.13-.12-.21-.21-.26-.27-.17-.21-.18-.42-.03-.62l1.54-2.03c.07-.1.18-.16.34-.18.15-.02.27.03.36.14l.03.03c1.13 1 2.34 1.63 3.64 1.89.37.08.74.12 1.11.12.81 0 1.52-.21 2.14-.64.61-.43.92-1.04.92-1.83 0-.28-.08-.55-.23-.8-.15-.25-.32-.46-.51-.63-.19-.17-.48-.36-.88-.56-.4-.21-.73-.37-.99-.48-.26-.12-.66-.28-1.2-.49l-.93-.38c-.23-.09-.54-.22-.93-.4-.39-.17-.7-.33-.94-.46s-.52-.31-.85-.53c-.33-.22-.6-.43-.8-.64-.21-.202-.42-.45-.65-.74-.23-.28-.41-.572-.53-.87-.13-.295-.23-.63-.32-1-.06-.4-.1-.79-.1-1.2 0-1.39.49-2.6 1.47-3.65S4.07 3.5 5.64 3.2V.48c0-.13.05-.243.143-.34.097-.09.21-.14.34-.14H8.15c.14 0 .254.045.344.136.09.09.135.206.135.346v2.652c.57.06 1.12.176 1.65.346.53.17.97.34 1.3.505.33.166.65.354.95.565.3.21.49.357.58.437.09.08.16.15.22.21.17.182.19.373.07.574l-1.21 2.2c-.08.15-.2.23-.35.24-.14.04-.28 0-.41-.1-.03-.03-.1-.09-.22-.18s-.31-.22-.59-.4c-.28-.17-.57-.33-.88-.48-.31-.14-.69-.27-1.12-.39-.44-.11-.86-.17-1.28-.17-.95 0-1.73.22-2.33.65-.6.44-.9.99-.9 1.68 0 .26.04.5.13.72.08.22.23.43.44.63s.41.36.59.5.464.29.84.47c.374.18.677.31.907.41.23.09.58.23 1.05.41.53.2.94.36 1.22.48s.66.29 1.14.53c.48.24.86.45 1.13.64.28.19.59.44.93.76.35.31.61.63.8.95.19.33.345.71.475 1.16.13.44.192.92.192 1.42z"/>
  </svg>
  <h1>
    <a class="ledger-title" href="/" on:click|preventDefault={() => onNavigate("/")}>{ledgerTitle}</a>
    <PageTitle {route} {account} {locale} {onNavigate} />
  </h1>
  <button
    type="button"
    class="reload-page"
    title={`${t("reload")} (r)`}
    aria-label={t("reload")}
    use:keyboardShortcut={"r"}
    on:click={() => onReload()}
  >&#8635;</button>
  <span class="spacer"></span>
  <form class="flex-row" aria-label="Global filters" on:submit|preventDefault>
    <AutocompleteInput
      value={timeDraft}
      on:change={(event) => { timeDraft = event.detail; }}
      placeholder="Time"
      suggestions={years}
      key="f t"
      clearButton={true}
      setSize={true}
      onBlur={() => onTime(timeDraft)}
      onSelect={() => onTime(timeDraft)}
      onEnter={() => onTime(timeDraft)}
    />
    <AutocompleteInput
      value={accountDraft}
      on:change={(event) => { accountDraft = event.detail; }}
      placeholder="Account"
      suggestions={accounts}
      key="f a"
      clearButton={true}
      setSize={true}
      onBlur={() => onAccount(accountDraft)}
      onSelect={() => onAccount(accountDraft)}
      onEnter={() => onAccount(accountDraft)}
    />
    <AutocompleteInput
      value={filterDraft}
      on:change={(event) => { filterDraft = event.detail; }}
      placeholder="Filter by tag, payee, ..."
      suggestions={filterSuggestions}
      key="f f"
      clearButton={true}
      setSize={true}
      {valueExtractor}
      {valueSelector}
      onBlur={() => onQuery(filterDraft)}
      onSelect={() => onQuery(filterDraft)}
      onEnter={() => onQuery(filterDraft)}
    />
  </form>
  <!-- Fava's Header.svelte ends at the filter form. Conversion and interval are
       SelectComboboxes inside the chart control bar (ConversionAndInterval), so
       they must not sit here: at narrow widths they wrap and add ~27px to the
       header, which shifts every route's content down. See TreeReport.svelte. -->
</header>

<style>
  h1 {
    display: inline-block;
    padding: 0.5rem;
    margin: 0;
    overflow: hidden;
    font-size: 16px;
    font-weight: normal;
  }

  .ledger-title {
    color: inherit;
  }

  .reload-page {
    padding: 0.25rem 0.4rem;
    font-size: 16px;
    color: var(--header-color);
    background-color: transparent;
    border: none;
    cursor: pointer;
  }

  .spacer {
    flex: 1;
  }

  form > :global(span) {
    max-width: 18rem;
  }

  .header-icon { flex: none; margin: 0 0.25rem 0 0.5rem; }

  .header-select {
    display: flex;
    gap: 0.25rem;
    align-items: center;
    white-space: nowrap;
  }

  .header-select select {
    color: var(--header-color);
    background-color: var(--header-background);
    border: 1px solid var(--header-placeholder-background);
  }

  @media (width <= 767px) {
    .header-select span {
      position: absolute;
      width: 1px;
      height: 1px;
      padding: 0;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
      white-space: nowrap;
      border: 0;
    }

    form {
      width: 100%;
    }

    form :global(input) {
      min-width: 0;
      flex: 1;
    }
  }
</style>
