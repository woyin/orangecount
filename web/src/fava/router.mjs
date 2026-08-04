export const ROUTES = Object.freeze([
  "income_statement",
  "balance_sheet",
  "trial_balance",
  "journal",
  "query",
  "holdings",
  "commodities",
  "documents",
  "events",
  "statistics",
  "editor",
  "import",
  "options",
  "help",
]);

const PATHS = Object.freeze({
  income_statement: "/income_statement",
  balance_sheet: "/balance_sheet",
  trial_balance: "/trial_balance",
  journal: "/journal",
  query: "/query",
  holdings: "/holdings",
  commodities: "/commodities",
  documents: "/documents",
  events: "/events",
  statistics: "/statistics",
  editor: "/editor",
  import: "/import",
  options: "/options",
  help: "/help",
});

const QUERY_KEYS = Object.freeze(["time", "account", "filter", "conversion", "interval"]);

function pathWithoutTrailingSlash(pathname) {
  if (pathname.length > 1 && pathname.endsWith("/")) return pathname.slice(0, -1);
  return pathname || "/";
}

function decodeAccount(value) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function parseRoute(input, basePath = "") {
  const url = input instanceof URL ? new URL(input.href) : new URL(input, "https://orange-count.invalid");
  let pathname = pathWithoutTrailingSlash(url.pathname);
  if (basePath && pathname.startsWith(basePath)) {
    pathname = pathWithoutTrailingSlash(pathname.slice(basePath.length));
  }
  let route = pathname === "/" ? "income_statement" : Object.entries(PATHS).find(([, path]) => path === pathname)?.[0] || "journal";
  let account = "";
  const accountPrefix = "/account/";
  if (pathname.startsWith(accountPrefix)) {
    route = "account";
    account = decodeAccount(pathname.slice(accountPrefix.length));
  }
  const query = {};
  for (const key of QUERY_KEYS) {
    const value = url.searchParams.get(key);
    if (value) query[key] = value;
  }
  return { route, account, query, pathname };
}

export function routeHref(route, { account = "", query = {} } = {}) {
  let pathname = PATHS[route] || "/journal";
  if (route === "account" && account) pathname = `/account/${encodeURIComponent(account)}`;
  const params = new URLSearchParams();
  for (const key of QUERY_KEYS) {
    if (query[key]) params.set(key, query[key]);
  }
  const suffix = params.toString();
  return `${pathname}${suffix ? `?${suffix}` : ""}`;
}

export function updateQuery(current, changes) {
  const parsed = parseRoute(current);
  const query = { ...parsed.query };
  for (const key of QUERY_KEYS) {
    if (Object.hasOwn(changes, key)) {
      if (changes[key]) query[key] = changes[key];
      else delete query[key];
    }
  }
  return routeHref(parsed.route, { account: parsed.account, query });
}

export function pageLabel(route) {
  const labels = {
    income_statement: "Income Statement",
    balance_sheet: "Balance Sheet",
    trial_balance: "Trial Balance",
    journal: "Journal",
    query: "Query",
    holdings: "Holdings",
    commodities: "Commodities",
    documents: "Documents",
    events: "Events",
    statistics: "Statistics",
    editor: "Editor",
    import: "Import",
    options: "Options",
    help: "Help",
    account: "Account",
  };
  return labels[route] || "Journal";
}
