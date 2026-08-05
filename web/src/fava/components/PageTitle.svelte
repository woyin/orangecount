<script lang="ts">
  import { pageLabel, routeHref } from "../router.mjs";
  import { translations, type Locale } from "../../translations";

  export let route: string;
  export let account = "";
  export let locale = "en";
  export let onNavigate: (href: string) => void = () => {};

  const translationKeys: Record<string, string> = {
    income_statement: "incomeStatement", balance_sheet: "balanceSheet", trial_balance: "trialBalance",
    journal: "journal", query: "query", holdings: "holdings", commodities: "commodities",
    documents: "documents", events: "events", statistics: "statistics", editor: "editor",
    import: "import", options: "options", help: "help", source: "source", diagnostics: "diagnostics",
    errors: "diagnostics",
  };
  $: catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
  $: title = route === "account" && account ? account : catalog[translationKeys[route] || ""] || pageLabel(route);
  $: segments = route === "account" && account ? account.split(":") : [];
</script>

<strong id="page-title">
  {#if segments.length}
    {#each segments as segment, index}
      {#if index < segments.length - 1}
        <a
          class="account-crumb"
          href={routeHref("account", { account: segments.slice(0, index + 1).join(":") })}
          on:click|preventDefault={() => onNavigate(routeHref("account", { account: segments.slice(0, index + 1).join(":") }))}
        >{segment}</a><span class="crumb-sep">›</span>
      {:else}
        {segment}
      {/if}
    {/each}
  {:else}
    {title}
  {/if}
</strong>

<style>
  strong::before {
    margin: 0 10px;
    font-weight: normal;
    content: "›";
    opacity: 0.5;
  }

  .account-crumb {
    color: inherit;
    font-weight: normal;
  }

  .crumb-sep {
    margin: 0 0.35em;
    font-weight: normal;
    opacity: 0.5;
  }
</style>
