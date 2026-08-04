<script lang="ts">
  import { ROUTES, pageLabel, routeHref } from "../router.mjs";

  export let route: string;
  export let open = false;
  export let onMenu: () => void;
  export let onNavigate: (href: string) => void;

  const sections = [
    ["", ["income_statement", "balance_sheet", "trial_balance", "journal", "query", "account"]],
    ["", ["holdings", "commodities", "documents", "events", "statistics"]],
    ["", ["editor", "import", "options", "help"]],
  ];
  const known = new Set([...ROUTES, "account"]);
</script>

{#if open}
  <div class="overlay" onclick={onMenu} aria-hidden="true"></div>
{/if}
<div class:active={open} class="aside-buttons">
  <button id="menu-toggle" type="button" aria-controls="sidebar" aria-expanded={open} aria-label="Menu" onclick={onMenu}>☰</button>
  <a class="button" href="#add-transaction" aria-label="Add transaction">+</a>
</div>
<aside id="sidebar" class:active={open} aria-label="Primary navigation">
  {#each sections as [heading, items]}
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
            >{pageLabel(item)}</a>
          </li>
        {/if}
      {/each}
    </ul>
  {/each}
</aside>

<style>
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
