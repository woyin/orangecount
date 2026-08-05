import { writable } from "svelte/store";

export const DEFAULT_LOCALE = "en";
export const DEFAULT_THEME = "system";

function stored(key, fallback, values) {
  try {
    const value = localStorage.getItem(key);
    return values.includes(value) ? value : fallback;
  } catch {
    return fallback;
  }
}

export function initialShellState(route) {
  return {
    route,
    locale: stored("orangecount-locale", DEFAULT_LOCALE, ["en", "zh-CN"]),
    theme: stored("orangecount-theme", DEFAULT_THEME, ["system", "dark", "light"]),
    loading: false,
    error: null,
    sidebarOpen: false,
    ledgerTitle: "OrangeCount",
    accounts: [],
    tags: [],
    links: [],
    payees: [],
    years: [],
    operatingCurrencies: [],
    renderCommas: false,
    query: {},
    revision: 0,
    errors: [],
  };
}

export function reduceShellState(state, action) {
  switch (action.type) {
    case "route":
      return { ...state, route: action.route, query: action.query || {}, sidebarOpen: false, error: null };
    case "locale":
      return { ...state, locale: action.locale === "zh-CN" ? "zh-CN" : DEFAULT_LOCALE };
    case "theme":
      return { ...state, theme: ["system", "dark", "light"].includes(action.theme) ? action.theme : DEFAULT_THEME };
    case "account":
      return { ...state, account: action.account || "" };
    case "query":
      return { ...state, query: { ...state.query, ...action.query } };
    case "menu":
      return { ...state, sidebarOpen: action.open ?? !state.sidebarOpen };
    case "loading":
      return { ...state, loading: Boolean(action.value), error: action.value ? null : state.error };
    case "error":
      return { ...state, loading: false, error: action.message || "The local adapter could not load this view." };
    case "clear-error":
      return { ...state, error: null };
    case "bootstrap":
      return {
        ...state,
        ledgerTitle: action.ledgerTitle || state.ledgerTitle,
        accounts: Array.isArray(action.accounts) ? action.accounts : state.accounts,
        tags: Array.isArray(action.tags) ? action.tags : state.tags,
        links: Array.isArray(action.links) ? action.links : state.links,
        payees: Array.isArray(action.payees) ? action.payees : state.payees,
        years: Array.isArray(action.years) ? action.years : state.years,
        operatingCurrencies: Array.isArray(action.operatingCurrencies) ? action.operatingCurrencies : state.operatingCurrencies,
        renderCommas: typeof action.renderCommas === "boolean" ? action.renderCommas : state.renderCommas,
        locale: action.locale === "zh-CN" ? "zh-CN" : state.locale,
        theme: ["system", "dark", "light"].includes(action.theme) ? action.theme : state.theme,
        errors: Array.isArray(action.errors) ? action.errors : state.errors,
        error: null,
        loading: false,
        revision: state.revision + 1,
      };
    default:
      return state;
  }
}

export function createShellStore(initial) {
  const store = writable(initial);
  return {
    subscribe: store.subscribe,
    dispatch(action) {
      store.update((state) => reduceShellState(state, action));
    },
  };
}
