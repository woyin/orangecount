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
