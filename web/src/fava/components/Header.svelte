<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { escape_for_regex } from "../lib/regex";
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
