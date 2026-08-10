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
  <label class="header-select">
    <span>{t("conversion")}</span>
    <select id="conversion" value={conversion} on:change={(event) => onConversion((event.currentTarget as HTMLSelectElement).value)}>
      <option value="at_cost">{t("atCost")}</option>
      <option value="market_value">{t("marketValue")}</option>
      <option value="units">Units</option>
      <option value="currency">{t("currency")}</option>
    </select>
  </label>
  <label class="header-select">
    <span>{t("interval")}</span>
    <select id="interval" value={interval} on:change={(event) => onInterval((event.currentTarget as HTMLSelectElement).value)}>
      <option value="month">{t("monthly")}</option>
      <option value="quarter">{t("quarterly")}</option>
      <option value="year">{t("yearly")}</option>
    </select>
  </label>
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
