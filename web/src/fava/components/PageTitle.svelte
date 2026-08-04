<script lang="ts">
  import { pageLabel } from "../router.mjs";
  import { translations, type Locale } from "../../translations";

  export let route: string;
  export let account = "";
  export let locale = "en";

  const translationKeys: Record<string, string> = {
    income_statement: "incomeStatement", balance_sheet: "balanceSheet", trial_balance: "trialBalance",
    journal: "journal", query: "query", holdings: "holdings", commodities: "commodities",
    documents: "documents", events: "events", statistics: "statistics", editor: "editor",
    import: "import", options: "options", help: "help", source: "source", diagnostics: "diagnostics",
    errors: "diagnostics",
  };
  $: catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
  $: title = route === "account" && account ? account : catalog[translationKeys[route] || ""] || pageLabel(route);
</script>

<strong id="page-title">{title}</strong>

<style>
  strong::before {
    margin: 0 10px;
    font-weight: normal;
    content: "›";
    opacity: 0.5;
  }
</style>
