<script lang="ts">
  import { onMount } from "svelte";
  import { createSyntheticAdapter } from "./adapter-client";
  import ErrorBoundary from "./components/ErrorBoundary.svelte";
  import Header from "./components/Header.svelte";
  import LoadingBoundary from "./components/LoadingBoundary.svelte";
  import PageTitle from "./components/PageTitle.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import { parseRoute, routeHref, updateQuery } from "./router.mjs";
  import { createShellStore, initialShellState } from "./state.mjs";

  const initialRoute = parseRoute(window.location.href);
  const shell = createShellStore({
    ...initialShellState(initialRoute.route),
    account: initialRoute.account,
    query: initialRoute.query,
  });
  const adapter = createSyntheticAdapter();

  $: current = $shell;
  $: document.documentElement.dataset.theme = current.theme === "system" ? "" : current.theme;
  $: document.documentElement.lang = current.locale;

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

  function setTime(value: string) {
    const href = updateQuery(window.location.href, { time: value === "all" ? "" : value });
    const target = new URL(href, window.location.href);
    window.history.replaceState({}, "", target.href);
    shell.dispatch({ type: "query", query: { time: value === "all" ? "" : value } });
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
      shell.dispatch({ type: "bootstrap", ledgerTitle: payload.ledger_title });
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
  <meta name="description" content="OrangeCount local ledger interface shell" />
</svelte:head>

<div class="application-shell">
  <Header
    ledgerTitle={current.ledgerTitle}
    route={current.route}
    account={current.account}
    locale={current.locale}
    theme={current.theme}
    menuOpen={current.sidebarOpen}
    time={current.query.time || "all"}
    filter={current.query.filter || ""}
    onMenu={() => shell.dispatch({ type: "menu" })}
    onNavigate={navigate}
    onLocale={setLocale}
    onTheme={setTheme}
    onTime={setTime}
    onQuery={setQuery}
  />
  <div class="shell-body">
    <Sidebar route={current.route} open={current.sidebarOpen} onNavigate={navigate} />
    <main id="main-content" tabindex="-1">
      <div class="page-heading">
        <PageTitle route={current.route} account={current.account} />
        <p class="subtitle">Read-only local ledger view</p>
      </div>
      <div id="app-content">
        <LoadingBoundary active={current.loading}>
          <ErrorBoundary message={current.error} onRetry={retry}>
            <section class="route-placeholder" aria-labelledby="route-heading">
              <p class="eyebrow">Fava-aligned shell</p>
              <h2 id="route-heading">{current.account || current.route}</h2>
              <p>This route is ready for its private OrangeCount adapter contract.</p>
              <p class="adapter-note">Data access is intentionally deferred until the P3 adapter is available.</p>
            </section>
          </ErrorBoundary>
        </LoadingBoundary>
      </div>
    </main>
  </div>
</div>
