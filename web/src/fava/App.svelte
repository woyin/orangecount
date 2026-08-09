<script lang="ts">
  import { onMount } from "svelte";
  import { createAdapterClient } from "./adapter-client";
  import ErrorBoundary from "./components/ErrorBoundary.svelte";
  import Header from "./components/Header.svelte";
  import LoadingBoundary from "./components/LoadingBoundary.svelte";
  import ReportOutlet from "./components/ReportOutlet.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import AddEntryModal from "./modals/AddEntryModal.svelte";
  import ContextModal from "./modals/ContextModal.svelte";
  import DocumentUploadModal from "./modals/DocumentUploadModal.svelte";
  import ExportModal from "./modals/ExportModal.svelte";
  import { initGlobalKeyboardShortcuts } from "./keyboard-shortcuts";
  import { notify, notify_err } from "./notifications";
  import { parseRoute, updateQuery } from "./router.mjs";
  import { createShellStore, initialShellState } from "./state.mjs";
  import { translations, type Locale } from "../translations";

  const initialRoute = parseRoute(window.location.href);
  const shell = createShellStore({
    ...initialShellState(initialRoute.route),
    account: initialRoute.account,
    helpPage: initialRoute.helpPage,
    query: initialRoute.query,
  });
  const adapter = createAdapterClient();

  $: current = $shell;
  $: document.documentElement.lang = current.locale;
  $: document.documentElement.dataset.theme = current.theme === "system" ? "" : current.theme;
  $: document.documentElement.style.colorScheme = current.theme === "system" ? "light dark" : current.theme;

  function navigate(href: string) {
    const target = new URL(href, window.location.href);
    const next = parseRoute(target.href);
    window.history.pushState({}, "", target.href);
    shell.dispatch({ type: "route", route: next.route, query: next.query, helpPage: next.helpPage });
    shell.dispatch({ type: "account", account: next.account });
  }

  function setQuery(value: string) {
    const href = updateQuery(window.location.href, { filter: value });
    const target = new URL(href, window.location.href);
    window.history.replaceState({}, "", target.href);
    shell.dispatch({ type: "query", query: { filter: value } });
  }

  function setAccount(value: string) {
    const href = updateQuery(window.location.href, { account: value });
    const target = new URL(href, window.location.href);
    window.history.replaceState({}, "", target.href);
    shell.dispatch({ type: "query", query: { account: value } });
  }

  function setTime(value: string) {
    const href = updateQuery(window.location.href, { time: value });
    const target = new URL(href, window.location.href);
    window.history.replaceState({}, "", target.href);
    shell.dispatch({ type: "query", query: { time: value } });
  }

  function setConversion(value: string) {
    const href = updateQuery(window.location.href, { conversion: value });
    const target = new URL(href, window.location.href);
    window.history.replaceState({}, "", target.href);
    shell.dispatch({ type: "query", query: { conversion: value } });
  }

 function setInterval(value: string) {
   const href = updateQuery(window.location.href, { interval: value });
   const target = new URL(href, window.location.href);
   window.history.replaceState({}, "", target.href);
   shell.dispatch({ type: "query", query: { interval: value } });
 }

  function setLocale(locale: string) {
    try { localStorage.setItem("orangecount-locale", locale); } catch { /* storage is optional */ }
    shell.dispatch({ type: "locale", locale });
  }

  function setTheme(theme: string) {
    try { localStorage.setItem("orangecount-theme", theme); } catch { /* storage is optional */ }
    shell.dispatch({ type: "theme", theme });
  }

  async function bootstrap() {
    shell.dispatch({ type: "loading", value: true });
    try {
      const payload = await adapter.bootstrap();
      shell.dispatch({ type: "bootstrap", ledgerTitle: payload.ledger_title, locale: payload.locale, theme: payload.theme, accounts: payload.accounts, tags: payload.tags, links: payload.links, payees: payload.payees, years: payload.years, userQueries: payload.user_queries, documentRoots: payload.document_roots, errors: payload.errors, operatingCurrencies: payload.operating_currencies, renderCommas: payload.render_commas, accountDetails: payload.account_details });
    } catch (error) {
      notify_err(error);
      shell.dispatch({ type: "error", message: error instanceof Error ? error.message : "The local adapter could not load this view." });
    }
  }

  onMount(() => {
    initGlobalKeyboardShortcuts();
    const onPopState = () => {
      const next = parseRoute(window.location.href);
      shell.dispatch({ type: "route", route: next.route, query: next.query, helpPage: next.helpPage });
      shell.dispatch({ type: "account", account: next.account });
    };
    window.addEventListener("popstate", onPopState);
    void bootstrap();
    const poll = window.setInterval(async () => {
      try {
        if (await adapter.changed()) {
          // The reload happens either way; the toast exists so the refresh is
          // perceptible, and clicking it forces one more pass.
          const catalog = translations[($shell.locale === "zh-CN" ? "zh-CN" : "en") as Locale];
          notify(catalog.fileChangeDetected ?? "File change detected. Click to reload.", "warning", () => { void bootstrap(); });
          await bootstrap();
        }
      } catch {
        // A transient poll failure must not replace the last usable report.
      }
    }, 5000);
    return () => {
      window.removeEventListener("popstate", onPopState);
      window.clearInterval(poll);
    };
  });

  function retry() {
    void bootstrap();
  }
</script>

<svelte:head>
  <title>{current.ledgerTitle} › {current.account || current.route}</title>
  <meta name="description" content="OrangeCount local ledger interface" />
</svelte:head>

<Header
  ledgerTitle={current.ledgerTitle}
  route={current.route}
  account={current.account}
  accounts={current.accounts}
  tags={current.tags}
  links={current.links}
  payees={current.payees}
  years={current.years}
  locale={current.locale}
  time={current.query.time || ""}
  accountFilter={current.query.account || ""}
  filter={current.query.filter || ""}
  onNavigate={navigate}
  onReload={retry}
  onTime={setTime}
  onAccount={setAccount}
  conversion={current.query.conversion || "at_cost"}
  interval={current.query.interval || "month"}
  onConversion={setConversion}
  onInterval={setInterval}
  onQuery={setQuery}
/>
<Sidebar route={current.route} open={current.sidebarOpen} errors={current.errors} locale={current.locale} accounts={current.accounts} userQueries={current.userQueries} onMenu={() => shell.dispatch({ type: "menu" })} onNavigate={navigate} />
<article id="main-content" tabindex="-1">
  <LoadingBoundary active={current.loading}>
    <ErrorBoundary message={current.error} onRetry={retry}>
      {#key current.revision}
        <ReportOutlet
          adapter={adapter}
          route={current.route}
          helpPage={current.helpPage || ""}
          locale={current.locale}
          theme={current.theme}
          operatingCurrencies={current.operatingCurrencies}
          renderCommas={current.renderCommas}
          accounts={current.accounts}
         accountDetails={current.accountDetails}
         onLocale={setLocale}
          onTheme={setTheme}
          query={{ ...current.query, ...(current.account ? { account: current.account } : {}) }}
        />
      {/key}
    </ErrorBoundary>
  </LoadingBoundary>
</article>
<ExportModal locale={current.locale} />
<ContextModal {adapter} locale={current.locale} />
<AddEntryModal locale={current.locale} payees={current.payees} onSaved={() => void bootstrap()} />
<DocumentUploadModal locale={current.locale} documentRoots={current.documentRoots} accounts={current.accounts} onUploaded={() => void bootstrap()} />

<style>
  :global(.route-placeholder) {
    max-width: 70rem;
  }

  :global(.route-placeholder p) {
    color: var(--text-color-lighter);
  }

  :global(.state-panel) {
    max-width: 70rem;
    padding: 1rem;
    margin-bottom: 1rem;
    border: 1px solid var(--border);
  }

  :global(.error-panel) {
    color: var(--error);
    border-color: var(--error);
  }

  :global(.loading-panel) {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  :global(.spinner) {
    width: 1rem;
    height: 1rem;
    border: 2px solid var(--border);
    border-top-color: var(--link-color);
    border-radius: 50%;
    animation: spinner 1s linear infinite;
  }

  @keyframes spinner {
    to { transform: rotate(360deg); }
  }
</style>
