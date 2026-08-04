<script lang="ts">
  import type { AdapterClient } from "../adapter-client";

  export let adapter: AdapterClient;
  export let route: string;
  export let query: Record<string, string> = {};

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
</script>

{#if loading}
  <section class="state-panel" role="status">Loading…</section>
{:else if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else if route === "help" && data?.sections}
  <div class="headerline"><h2>Help</h2></div>
  {#each data.sections as section (section.id)}
    <details open>
      <summary>{section.title}</summary>
      <div>{section.body}</div>
    </details>
  {/each}
{:else if route === "source" && data?.content !== undefined}
  <div class="headerline"><h2>{data.path}</h2></div>
  <pre class="source-content">{data.content}</pre>
{:else if route === "source" && data?.paths}
  <div class="headerline"><h2>Source files</h2></div>
  <ul class="source-list">
    {#each data.paths as path (path)}<li><a href={`/source?path=${encodeURIComponent(path)}`}>{path}</a></li>{/each}
  </ul>
{:else if route === "options"}
  <div class="headerline"><h2>Options</h2></div>
  <table><thead><tr><th>Option</th><th>Value</th></tr></thead><tbody>
    {#each objectEntries(data?.options) as [key, value] (key)}<tr><th scope="row">{key}</th><td>{String(value)}</td></tr>{/each}
  </tbody></table>
{:else}
  <div class="headerline"><h2>{route}</h2></div>
  <pre>{JSON.stringify(data, null, 2)}</pre>
{/if}

<style>
  details,
  .source-list,
  table,
  pre {
    max-width: 70rem;
  }

  .source-list {
    padding-left: 1.5rem;
  }

  pre {
    padding: 0.75rem;
    overflow: auto;
    background: var(--code-background);
  }
</style>
