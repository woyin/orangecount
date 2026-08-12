<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { Sorter, StringColumn, type SortColumn } from "../sort/index";
  import SortHeader from "../sort/SortHeader.svelte";
  import type { AdapterClient, DiagnosticContext, RepairGuide } from "../adapter-client";

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
  let expandedDiagnostic = "";
  let guideCache: Record<string, RepairGuide> = {};
  let guideErrors: Record<string, string> = {};
  let contextCache: Record<string, DiagnosticContext> = {};
  let contextLoading = "";
  let loadedLocale = locale;

  // Locale changes do not reload the whole shell revision. Drop lazy-loaded
  // localized content so the next expansion requests the selected language.
  $: if (locale !== loadedLocale) {
    loadedLocale = locale;
    guideCache = {};
    guideErrors = {};
    contextCache = {};
    const openRow = diagnosticRows.find((value: DiagnosticRow) => `${value.code}:${value.path}:${value.line}` === expandedDiagnostic);
    if (openRow) void loadGuide(openRow);
  }

  async function load() {
    loading = true;
    error = "";
    try {
      const request = route === "help" ? { ...query, locale, ...(helpPage.startsWith("diagnostics/") ? { topic: helpPage } : {}) } : query;
      data = await adapter.load(route, request);
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

  type DiagnosticRow = { code: string; severity: string; path: string; line: number; column: number; message: string };
  $: diagnosticRows = route === "diagnostics" && Array.isArray(data) ? data.map((value: any): DiagnosticRow => ({
    code: String(value?.code ?? value?.Code ?? value?.type ?? ""),
    severity: String(value?.severity ?? value?.Severity ?? "error"),
    path: String(value?.path ?? value?.Path ?? value?.span?.path ?? value?.source?.filename ?? ""),
    line: Number(value?.line ?? value?.span?.start_line ?? value?.source?.lineno ?? 0),
    column: Number(value?.column ?? value?.span?.start_column ?? 0),
    message: String(value?.message ?? value?.Message ?? ""),
  })) : [];
  $: fixFirstRows = diagnosticRows.filter((value: DiagnosticRow) => phaseFor(value.code) !== "recheck-after-semantic");
  $: recheckRows = diagnosticRows.filter((value: DiagnosticRow) => phaseFor(value.code) === "recheck-after-semantic");

  function phaseFor(code: string): string {
    if (code.startsWith("E-INCLUDE") || code.startsWith("E-SOURCE")) return "fix-first-source";
    if (code.startsWith("E-PARSE")) return "fix-first-syntax";
    return "recheck-after-semantic";
  }

  async function loadGuide(row: DiagnosticRow) {
    if (row.severity !== "error" || !row.code || guideCache[row.code]) return;
    try {
      guideCache = { ...guideCache, [row.code]: await adapter.guide(row.code, locale) };
    } catch (value) {
      guideErrors = { ...guideErrors, [row.code]: value instanceof Error ? value.message : t("noGuidance") };
    }
  }

  async function openDiagnostic(row: DiagnosticRow) {
    if (row.severity !== "error") return;
    const key = `${row.code}:${row.path}:${row.line}`;
    expandedDiagnostic = expandedDiagnostic === key ? "" : key;
    if (expandedDiagnostic !== key) return;
    await loadGuide(row);
  }

  async function loadContext(row: DiagnosticRow) {
    const key = `${row.path}:${row.line}`;
    contextLoading = key;
    try {
      contextCache = { ...contextCache, [key]: await adapter.diagnosticContext(row.path, row.line) };
    } catch (value) {
      contextCache = { ...contextCache, [key]: { available: false, reason: value instanceof Error ? value.message : t("contextUnavailable") } };
    } finally {
      if (contextLoading === key) contextLoading = "";
    }
  }

  function renderDiagnosticGroup(title: string, rows: DiagnosticRow[]) {
    return { title, rows };
  }
</script>

{#if loading}
  <section class="state-panel" role="status">Loading…</section>
{:else if error}
  <section class="state-panel error-panel" role="alert">{error}</section>
{:else if route === "help" && activeHelpSection}
  <div class="headerline"><h2>{activeHelpSection.title}</h2></div>
  <p><a href="/help">‹ {t("help")}</a></p>
  <div class="help-body">{activeHelpSection.body}</div>
{:else if route === "help" && data?.code}
  <div class="headerline"><h2>{data.code}</h2></div>
  <p><a href="/help">‹ {t("help")}</a></p>
  <article class="repair-guide">
    <p class="repair-action"><strong>{data.short_action}</strong></p>
    <h3>{t("whatHappened")}</h3><p>{data.what}</p>
    <h3>{t("whyBlocks")}</h3><p>{data.why}</p>
    <h3>{t("whereToInspect")}</h3>
    <ul>{#each data.inspect ?? [] as item}<li>{item}</li>{/each}</ul>
    <h3>{t("safeChecks")}</h3>
    <ul>{#each data.safe_steps ?? [] as item}<li>{item}</li>{/each}</ul>
    <h3>{t("genericExample")}</h3>
    <div class="example-grid"><div><h4>{t("before")}</h4><pre>{data.example?.before}</pre></div><div><h4>{t("after")}</h4><pre>{data.example?.after}</pre></div></div>
    <p class="muted">{data.example?.note}</p>
    <h3>{t("nextStep")}</h3><p>{data.revalidate}</p>
  </article>
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
{:else if route === "diagnostics"}
  <div class="headerline"><h2>{t("diagnostics")}</h2></div>
  {#if diagnosticRows.length === 0}
    <p>{t("noErrors")}</p>
  {:else}
    <p class="muted">{t("repairOrderHint")}</p>
    {#each [renderDiagnosticGroup(t("fixFirst"), fixFirstRows), renderDiagnosticGroup(t("recheckAfter"), recheckRows)] as group (group.title)}
      {#if group.rows.length}
        <section class="diagnostic-group"><h3>{group.title}</h3>
          {#each group.rows as row (row.code + row.path + row.line)}
            {@const key = `${row.code}:${row.path}:${row.line}`}
            <article class:error-diagnostic={row.severity === "error"} class="diagnostic-card">
              <header><div><code>{row.code}</code> <span class="muted">{row.path}:{row.line}:{row.column}</span></div><p>{row.message}</p></header>
              {#if row.severity === "error"}<button type="button" aria-expanded={expandedDiagnostic === key} on:click={() => openDiagnostic(row)}>{expandedDiagnostic === key ? t("hideGuidance") : t("learnHowToFix")}</button>{/if}
              {#if row.severity === "error" && expandedDiagnostic === key}
                {#if guideErrors[row.code]}<p class="error-panel" role="alert">{guideErrors[row.code]}</p>
                {:else if !guideCache[row.code]}<p class="muted" role="status">{t("loadingGuide")}</p>
                {:else}
                  {@const guide = guideCache[row.code]}
                  <div class="repair-guide compact">
                    <p class="repair-action"><strong>{guide.short_action}</strong></p>
                    {#if row.path}<p><a href={`/source?path=${encodeURIComponent(row.path)}`}>{row.path}:{row.line}:{row.column}</a></p>{/if}
                    <h4>{t("whatHappened")}</h4><p>{guide.what}</p>
                    <h4>{t("whyBlocks")}</h4><p>{guide.why}</p>
                    <h4>{t("whereToInspect")}</h4><p>{guide.inspect.join(" ")}</p>
                    <h4>{t("safeChecks")}</h4><p>{guide.safe_steps.join(" ")}</p>
                    <h4>{t("genericExample")}</h4><div class="example-grid"><pre>{guide.example.before}</pre><pre>{guide.example.after}</pre></div>
                    <p class="muted">{guide.example.note}</p><h4>{t("nextStep")}</h4><p>{guide.revalidate}</p>
                    <p><a href={`/help/${encodeURIComponent(guide.topic)}`}>{t("helpTopic")}</a></p>
                    {#if row.path && row.line}
                      <button type="button" on:click={() => loadContext(row)} disabled={contextLoading === `${row.path}:${row.line}`}>{contextLoading === `${row.path}:${row.line}` ? t("loading") : t("showLocalContext")}</button>
                      {@const context = contextCache[`${row.path}:${row.line}`]}
                      {#if context}
                        {#if context.available}<pre class="source-content context-snippet">{#each context.lines ?? [] as line}{line.line}: {line.content}{"\n"}{/each}</pre>
                        {:else}<p class="muted" role="status">{context.reason ?? t("contextUnavailable")}</p>{/if}
                      {/if}
                    {/if}
                  </div>
                {/if}
              {/if}
            </article>
          {/each}
        </section>
      {/if}
    {/each}
  {/if}
{:else}
  <div class="headerline"><h2>{route}</h2></div>
  <pre>{JSON.stringify(data, null, 2)}</pre>
{/if}

<style>
  .source-list,
  .help-index,
  .help-body,
  table,
  pre {
    max-width: 70rem;
  }

  .diagnostic-group { max-width: 70rem; margin: 1rem 0 1.5rem; }
  .diagnostic-card { padding: 0.75rem; margin: 0.5rem 0; border: 1px solid var(--border); }
  .diagnostic-card header p { margin: 0.35rem 0 0.75rem; }
  .diagnostic-card code { color: var(--text-color-lightest); }
  .repair-guide { max-width: 70rem; }
  .repair-guide.compact { padding-top: 0.75rem; }
  .repair-action { color: var(--link-color); }
  .repair-guide h3, .repair-guide h4 { margin-bottom: 0.25rem; }
  .repair-guide p, .repair-guide ul { margin-top: 0.25rem; }
  .example-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.75rem; }
  .example-grid pre { margin: 0; }
  .context-snippet { border-left: 3px solid var(--accent, #d97706); }
  @media (max-width: 40rem) { .example-grid { grid-template-columns: 1fr; } }

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
