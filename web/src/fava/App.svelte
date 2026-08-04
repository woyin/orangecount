<script lang="ts">
  import { onMount } from "svelte";
  import { createAdapterClient } from "./adapter-client";
  import ErrorBoundary from "./components/ErrorBoundary.svelte";
  import Header from "./components/Header.svelte";
  import LoadingBoundary from "./components/LoadingBoundary.svelte";
  import ReportOutlet from "./components/ReportOutlet.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import { parseRoute, updateQuery } from "./router.mjs";
  import { createShellStore, initialShellState } from "./state.mjs";

  const initialRoute = parseRoute(window.location.href);
  const shell = createShellStore({
    ...initialShellState(initialRoute.route),
    account: initialRoute.account,
    query: initialRoute.query,
  });
  const adapter = createAdapterClient();

  $: current = $shell;
  $: document.documentElement.lang = current.locale;
  $: document.documentElement.dataset.theme = current.theme === "system" ? "" : current.theme;

  function navigate(href: string) {
    const target = new URL(href, window.location.href);
    const next = parseRoute(target.href);
    window.history.pushState({}, "", target.href);
    shell.dispatch({ type: "route", route: next.route, query: next.query });
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

  function setLocale(locale: string) {
    shell.dispatch({ type: "locale", locale });
  }

  function setTheme(theme: string) {
    shell.dispatch({ type: "theme", theme });
  }

  async function bootstrap() {
    shell.dispatch({ type: "loading", value: true });
    try {
      const payload = await adapter.bootstrap();
      shell.dispatch({ type: "bootstrap", ledgerTitle: payload.ledger_title, locale: payload.locale, theme: payload.theme });
    } catch (error) {
      shell.dispatch({ type: "error", message: error instanceof Error ? error.message : "The local adapter could not load this view." });
    }
  }

  onMount(() => {
    const onPopState = () => {
      const next = parseRoute(window.location.href);
      shell.dispatch({ type: "route", route: next.route, query: next.query });
      shell.dispatch({ type: "account", account: next.account });
    };
    window.addEventListener("popstate", onPopState);
    void bootstrap();
    return () => window.removeEventListener("popstate", onPopState);
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
  locale={current.locale}
  theme={current.theme}
  time={current.query.time || ""}
  accountFilter={current.query.account || ""}
  filter={current.query.filter || ""}
  onNavigate={navigate}
  onLocale={setLocale}
  onTheme={setTheme}
  onTime={setTime}
  onAccount={setAccount}
  onQuery={setQuery}
/>
<Sidebar route={current.route} open={current.sidebarOpen} onMenu={() => shell.dispatch({ type: "menu" })} onNavigate={navigate} />
<article id="main-content" tabindex="-1">
  <LoadingBoundary active={current.loading}>
    <ErrorBoundary message={current.error} onRetry={retry}>
      <ReportOutlet
        adapter={adapter}
        route={current.route}
        query={{ ...current.query, ...(current.account ? { account: current.account } : {}) }}
      />
    </ErrorBoundary>
  </LoadingBoundary>
</article>

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
