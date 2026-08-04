<script lang="ts">
  import { pageLabel } from "../router.mjs";

  export let ledgerTitle: string;
  export let route: string;
  export let account = "";
  export let locale: string;
  export let theme: string;
  export let menuOpen = false;
  export let time = "all";
  export let filter = "";
  export let onMenu: () => void;
  export let onNavigate: (href: string) => void;
  export let onLocale: (value: string) => void;
  export let onTheme: (value: string) => void;
  export let onTime: (value: string) => void;
  export let onQuery: (value: string) => void;
</script>

<header class="topbar">
  <button id="menu-toggle" class="menu-toggle" type="button" aria-controls="sidebar" aria-expanded={menuOpen} aria-label="Menu" on:click={onMenu}>
    <span aria-hidden="true">☰</span>
  </button>
  <a class="brand" href="/" on:click|preventDefault={() => onNavigate("/")}>{ledgerTitle}</a>
  <span class="brand-separator" aria-hidden="true">›</span>
  <span class="brand-page">{account || pageLabel(route)}</span>
  <div class="global-filters" role="search" aria-label="Global filters">
    <label>Time
      <select id="global-time" value={time} on:change={(event) => onTime((event.currentTarget as HTMLSelectElement).value)}>
        <option value="all">All time</option>
        <option value="year">This year</option>
        <option value="month">This month</option>
      </select>
    </label>
    <input id="global-filter" type="search" value={filter} aria-label="Filter by tag, payee, or narration" placeholder="Filter by tag, payee, or narration" on:input={(event) => onQuery((event.currentTarget as HTMLInputElement).value)} />
  </div>
  <label class="select-control">Language
    <select id="locale" value={locale} on:change={(event) => onLocale((event.currentTarget as HTMLSelectElement).value)}>
      <option value="en">English</option>
      <option value="zh-CN">简体中文</option>
    </select>
  </label>
  <label class="select-control">Theme
    <select id="theme" value={theme} on:change={(event) => onTheme((event.currentTarget as HTMLSelectElement).value)}>
      <option value="system">System</option>
      <option value="dark">Dark</option>
      <option value="light">Light</option>
    </select>
  </label>
</header>
