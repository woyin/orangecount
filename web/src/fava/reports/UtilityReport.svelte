<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { Sorter, StringColumn, type SortColumn } from "../sort/index";
  import SortHeader from "../sort/SortHeader.svelte";
  import type { AdapterClient } from "../adapter-client";

  export let adapter: AdapterClient;
  export let route: string;
  export let helpPage = "";
  export let query: Record<string, string> = {};
  export let locale = "en";
  export let theme = "system";
  export let onLocale: (value: string) => void = () => {};
  export let onTheme: (value: string) => void = () => {};

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  let loading = true;
  let error = "";
  let data: any = null;

  async function load() {
    loading = true;
    error = "";
    try {
      data = await adapter.load(route, query);
    } catch (value) {
      error = value instanceof Error ? value.message : "The page could not be loaded.";
    } finally {
      loading = false;
    }
  }

  load();

  function objectEntries(value: unknown): [string, unknown][] {
    return value && typeof value === "object" && !Array.isArray(value) ? Object.entries(value as Record<string, unknown>) : [];
  }

  $: colorSchemes = [
    ["system", `⚙️ ${t("system")}`],
    ["dark", `🌙 ${t("dark")}`],
    ["light", `☀️ ${t("light")}`],
  ] as [string, string][];

  // The adapter's fava_options carries the ledger-declared subset; locale and
  // theme are shell-managed here (decision D2), so merge them in and sort by
  // key like upstream's default Sorter state.
  $: favaRows = (() => {
    const extras = objectEntries(data?.fava_options)
      .filter(([key]) => key !== "locale" && key !== "theme")
      .map(([key, value]) => [key, String(value)] as [string, string]);
    return [...extras, ["locale", locale], ["theme", theme]].sort(([a], [b]) => a.localeCompare(b));
  })();

  $: beancountRows = objectEntries(data?.options)
    .map(([key, value]) => [key, String(value)] as [string, string])
    .sort(([a], [b]) => a.localeCompare(b));

  // Upstream renders both options tables with sortable Key/Value headers.
  $: optionColumns = [
    new StringColumn<[string, string]>(t("optionKey"), (row) => row[0]),
    new StringColumn<[string, string]>(t("optionValue"), (row) => row[1]),
  ] as SortColumn<[string, string]>[];
  $: favaSorter = new Sorter<[string, string]>(optionColumns[0], "asc");
  $: beancountSorter = new Sorter<[string, string]>(optionColumns[0], "asc");
  $: sortedFavaRows = favaSorter.sort(favaRows);
  $: sortedBeancountRows = beancountSorter.sort(beancountRows);

  // Like upstream, help lives under /help/<page>; the bare /help route is the
  // index listing every topic.
  $: helpSections = Array.isArray(data?.sections) ? data.sections : [];
  $: activeHelpSection = helpSections.find((section: { id: string }) => section.id === helpPage);
</script>

{#if loading}
  <section class="state-panel" role="status">Loading…</section>
{:else if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else if route === "help" && activeHelpSection}
  <div class="headerline"><h2>{activeHelpSection.title}</h2></div>
  <p><a href="/help">‹ {t("help")}</a></p>
  <div class="help-body">{activeHelpSection.body}</div>
{:else if route === "help" && data?.sections}
  <div class="headerline"><h2>Help</h2></div>
  <ul class="help-index">
    {#each helpSections as section (section.id)}
      <li><a href={`/help/${encodeURIComponent(section.id)}`}>{section.title}</a></li>
    {/each}
  </ul>
{:else if route === "source" && data?.content !== undefined}
  <div class="headerline"><h2>{data.path}</h2></div>
  <pre class="source-content">{data.content}</pre>
{:else if route === "source" && data?.paths}
  <div class="headerline"><h2>Source files</h2></div>
  <ul class="source-list">
    {#each data.paths as path (path)}<li><a href={`/source?path=${encodeURIComponent(path)}`}>{path}</a></li>{/each}
  </ul>
{:else if route === "options"}
  <div class="headerline"><h2>{t("options")}</h2></div>
  <h3>{t("colorScheme")}</h3>
  <p>
    <span class="mode-switch" role="radiogroup" aria-label={t("colorScheme")}>
      {#each colorSchemes as [value, label] (value)}
        <label class="button" class:muted={theme !== value}>
          <input type="radio" name="color-scheme" value={value} checked={theme === value} on:change={() => onTheme(value)} />
          {label}
        </label>
      {/each}
    </span>
  </p>
  <h3>{t("favaOptions")} <a href="/help/options">({t("help")})</a></h3>
  <table class="options-table">
    <thead>
      <tr>
        <SortHeader bind:sorter={favaSorter} column={optionColumns[0]} />
        <SortHeader bind:sorter={favaSorter} column={optionColumns[1]} />
      </tr>
    </thead>
    <tbody>
      {#each sortedFavaRows as [key, value] (key)}
        <tr>
          <td>{key}</td>
          <td>
            {#if key === "locale"}
              <select id="fava-option-locale" value={locale} on:change={(event) => onLocale((event.currentTarget as HTMLSelectElement).value)}>
                <option value="en">English</option>
                <option value="zh-CN">简体中文</option>
              </select>
            {:else}
              <pre>{value}</pre>
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  <h3>{t("beancountOptions")}</h3>
  <table class="options-table">
    <thead>
      <tr>
        <SortHeader bind:sorter={beancountSorter} column={optionColumns[0]} />
        <SortHeader bind:sorter={beancountSorter} column={optionColumns[1]} />
      </tr>
    </thead>
    <tbody>
      {#each sortedBeancountRows as [key, value] (key)}
        <tr><td>{key}</td><td><pre>{value}</pre></td></tr>
      {/each}
    </tbody>
  </table>
{:else}
  <div class="headerline"><h2>{route}</h2></div>
  <pre>{JSON.stringify(data, null, 2)}</pre>
{/if}

<style>
  details,
  .source-list,
  .help-index,
  .help-body,
  table,
  pre {
    max-width: 70rem;
  }

  .help-index {
    padding-left: 1.5rem;
  }

  .source-list {
    padding-left: 1.5rem;
  }

  .mode-switch input {
    display: none;
  }

  .mode-switch label + label {
    margin-left: 0.125rem;
  }

  .options-table td:nth-child(1) {
    font-weight: 500;
  }

  .options-table pre {
    padding: 0;
    margin: 0;
    background: transparent;
  }

  pre {
    padding: 0.75rem;
    overflow: auto;
    background: var(--code-background);
  }
</style>
