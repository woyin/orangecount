<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import PageTitle from "./PageTitle.svelte";

  export let ledgerTitle: string;
  export let route: string;
  export let account = "";
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
</script>

<header>
  <h1>
    <a class="ledger-title" href="/" on:click|preventDefault={() => onNavigate("/")}>{ledgerTitle}</a>
    <PageTitle {route} {account} {locale} />
  </h1>
  <span class="spacer"></span>
  <form class="flex-row" aria-label="Global filters" on:submit|preventDefault>
    <input
      id="global-time"
      type="text"
      value={time}
      placeholder="Time"
      aria-label="Time"
      on:change={(event) => onTime((event.currentTarget as HTMLInputElement).value)}
    />
    <input
      id="global-account"
      type="text"
      value={accountFilter}
      placeholder="Account"
      aria-label="Account"
      on:change={(event) => onAccount((event.currentTarget as HTMLInputElement).value)}
    />
    <input
      id="global-filter"
      type="text"
      value={filter}
      placeholder="Filter by tag, payee, ..."
      aria-label="Filter by tag, payee, or narration"
      on:change={(event) => onQuery((event.currentTarget as HTMLInputElement).value)}
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
