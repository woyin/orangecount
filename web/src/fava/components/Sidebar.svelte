<script lang="ts">
  import { ROUTES, pageLabel, routeHref } from "../router.mjs";

  export let route: string;
  export let open = false;
  export let onNavigate: (href: string) => void;

  const sections = [
    ["Reports", ["income_statement", "balance_sheet", "trial_balance", "journal"]],
    ["Explore", ["query", "holdings", "commodities", "documents", "events", "statistics"]],
    ["Tools", ["editor", "import", "options", "help"]],
  ];
  const known = new Set(ROUTES);
</script>

<aside id="sidebar" class:open class="sidebar" aria-label="Primary navigation">
  {#each sections as [heading, items]}
    <section class="nav-section" aria-labelledby={`nav-${heading}`}>
      <h2 id={`nav-${heading}`}>{heading}</h2>
      <nav>
        {#each items as item}
          {#if known.has(item)}
            <a href={routeHref(item)} class:active={route === item} aria-current={route === item ? "page" : undefined} data-route={item} on:click|preventDefault={() => onNavigate(routeHref(item))}>
              {pageLabel(item)}
            </a>
          {/if}
        {/each}
      </nav>
    </section>
  {/each}
</aside>
