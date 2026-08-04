<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { ROUTES, pageLabel, routeHref } from "../router.mjs";

  export let route: string;
  export let open = false;
  export let errors: unknown[] = [];
  export let locale = "en";
  export let onMenu: () => void;
  export let onNavigate: (href: string) => void;

  const sections = [
    ["", ["income_statement", "balance_sheet", "trial_balance", "journal", "query", "account"]],
    ["", ["holdings", "commodities", "documents", "events", "statistics"]],
    ["", ["editor", "import", "options", "help"]],
  ];
  const known = new Set([...ROUTES, "account"]);
  const keys: Record<string, string> = { income_statement: "incomeStatement", balance_sheet: "balanceSheet", trial_balance: "trialBalance", journal: "journal", query: "query", holdings: "holdings", commodities: "commodities", documents: "documents", events: "events", statistics: "statistics", editor: "editor", import: "import", options: "options", help: "help", account: "accounts" };
  function label(routeName: string): string { const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale]; return catalog[keys[routeName] || ""] || pageLabel(routeName); }
</script>

{#if open}
  <div class="overlay" onclick={onMenu} aria-hidden="true"></div>
{/if}
<div class:active={open} class="aside-buttons">
  <button id="menu-toggle" type="button" aria-controls="sidebar" aria-expanded={open} aria-label="Menu" onclick={onMenu}>☰</button>
  <a class="button" href="#add-transaction" aria-label="Add transaction">+</a>
</div>
<aside id="sidebar" class:active={open} aria-label="Primary navigation">
  {#each sections as [heading, items], sectionIndex}
    <ul class="navigation" aria-label={heading || "Reports"}>
      {#each items as item}
        {#if known.has(item)}
          <li>
            <a
              href={routeHref(item)}
              class:selected={route === item}
              aria-current={route === item ? "page" : undefined}
              data-route={item}
              onclick={(event) => { event.preventDefault(); onNavigate(routeHref(item)); }}
            >{label(item)}</a>
          </li>
        {/if}
      {/each}
    </ul>
    {#if sectionIndex === sections.length - 1 && errors.length}
      <ul class="navigation">
        <li><a href={routeHref("errors")} class:selected={route === "errors"} aria-current={route === "errors" ? "page" : undefined} onclick={(event) => { event.preventDefault(); onNavigate(routeHref("errors")); }}>Errors ({errors.length})</a></li>
      </ul>
    {/if}
  {/each}
</aside>

<style>
  aside {
    grid-area: aside;
    padding-top: 0.5rem;
    margin: 0;
    overflow-y: auto;
    color: var(--sidebar-color);
    background-color: var(--sidebar-background);
    border-right: 1px solid var(--sidebar-border);
  }

  .aside-buttons {
    display: none;
  }

  @media (width <= 767px) {
    :global(:root) {
      --aside-width: 200px;
    }

    aside {
      position: fixed;
      top: 0;
      bottom: 0;
      z-index: var(--z-index-floating-ui);
      width: 200px;
      margin-left: -200px;
      transition: var(--transitions);
    }

    .overlay {
      position: fixed;
      inset: 0;
      z-index: var(--z-index-floating-ui);
      cursor: pointer;
      background: var(--overlay-wrapper-background);
      transition: var(--transitions);
    }

    aside.active {
      margin-left: 0;
    }

    .aside-buttons {
      position: fixed;
      top: 0;
      left: 0;
      z-index: var(--z-index-floating-ui);
      display: flex;
      flex-direction: column;
      transition: var(--transitions);
    }

    .active.aside-buttons {
      left: 200px;
    }

    .aside-buttons > * {
      width: 42px;
      height: 42px;
      color: var(--mobile-button-text);
      text-align: center;
      background-color: var(--sidebar-background);
      border: 1px solid var(--sidebar-border);
    }

    .aside-buttons a {
      font-size: 28px;
    }
  }

  @media print {
    aside,
    .aside-buttons {
      display: none;
    }
  }

  .navigation {
    padding-bottom: 0.5rem;
    margin: 0;
  }

  .navigation + .navigation {
    padding-top: 0.5rem;
    border-top: 1px solid var(--sidebar-border);
  }

  .navigation a.selected,
  .navigation a:hover {
    color: var(--sidebar-hover-color);
    background-color: var(--sidebar-border);
  }

  .navigation li {
    display: flex;
    flex-wrap: wrap;
  }

  .navigation a {
    display: block;
    flex: 1;
    padding: 0.25em 0.5em 0.25em 1em;
    color: inherit;
  }

  .aside-buttons button {
    font-size: 1.2rem;
  }
</style>
