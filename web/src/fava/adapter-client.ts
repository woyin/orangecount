/**
 * Narrow client boundary for the private OrangeCount Fava-shaped adapter.
 *
 * The endpoint names are internal client contracts, not Fava's public API.
 * The loopback handlers may change transport details without making this
 * client a public compatibility promise.
 */

export interface BootstrapPayload {
  ledger_title: string;
  locale: string;
  locales: string[];
  theme: string;
  routes: string[];
  accounts: string[];
  currencies: string[];
  tags: string[];
  links: string[];
  payees: string[];
  years: string[];
  /** Declared operating currencies, in ledger order; these get their own
   * report columns while every other currency shares an "Other" column. */
  operating_currencies: string[];
  /** Ledger `render_commas` option: group thousands in displayed amounts. */
  render_commas: boolean;
  errors: unknown[];
  mtime?: string;
}

export interface AdapterClient {
  bootstrap(): Promise<BootstrapPayload>;
  changed(): Promise<boolean>;
  load(route: string, query?: Record<string, string>): Promise<unknown>;
}

export const PRIVATE_ADAPTER_BASE = "/__orangecount/fava";

interface AdapterEnvelope<T> {
  data: T;
  mtime?: string;
}

interface BootstrapWire {
  accounts: string[];
  currencies: string[];
  tags?: string[];
  links?: string[];
  payees?: string[];
  years?: string[];
  errors: unknown[];
  options?: Record<string, string>;
  fava_options?: Record<string, string>;
}

function bootstrapPayload(wire: BootstrapWire, mtime = ""): BootstrapPayload {
  const title = wire.options?.title?.trim() || "OrangeCount";
  const locale = wire.fava_options?.locale === "zh-CN" ? "zh-CN" : "en";
  return {
    ledger_title: title,
    locale,
    locales: ["en", "zh-CN"],
    theme: "system",
    routes: ["income_statement", "balance_sheet", "trial_balance", "journal", "query", "holdings", "commodities", "documents", "events", "statistics", "editor", "import", "options", "help", "account"],
    accounts: wire.accounts || [],
    currencies: wire.currencies || [],
    tags: wire.tags || [],
    links: wire.links || [],
    payees: wire.payees || [],
    years: wire.years || [],
    // The evaluator joins repeated operating_currency declarations into one
    // space-separated value, preserving declaration order.
    operating_currencies: (wire.options?.operating_currency || "").split(/\s+/).filter(Boolean),
    render_commas: (wire.options?.render_commas || "").toUpperCase() === "TRUE",
    errors: wire.errors || [],
    mtime,
  };
}

export function createAdapterClient(
  fetcher: typeof fetch = fetch,
  base = PRIVATE_ADAPTER_BASE,
): AdapterClient {
  let lastMtime = "";

  async function get<T>(resource: string, query: Record<string, string> = {}): Promise<T> {
    const params = new URLSearchParams(query);
    const response = await fetcher(`${base}/${resource}${params.size ? `?${params}` : ""}`, {
      headers: { Accept: "application/json" },
    });
    const payload = await response.json() as AdapterEnvelope<T> & { error?: string };
    if (!response.ok) throw new Error(payload.error || `Adapter request failed (${response.status})`);
    if (payload.mtime) lastMtime = payload.mtime;
    return payload.data;
  }

  return {
    bootstrap: async () => {
      const wire = await get<BootstrapWire>("ledger_data");
      return bootstrapPayload(wire, lastMtime);
    },
    changed: async () => get<boolean>("changed", lastMtime ? { mtime: lastMtime } : {}),
    load: (route, query = {}) => {
      const treeRoutes = new Set(["income_statement", "balance_sheet", "trial_balance"]);
      const directRoutes = new Set(["options", "help", "diagnostics", "source", "editor", "import", "journal"]);
      const resource = treeRoutes.has(route) || directRoutes.has(route)
        ? route
        : route.startsWith("holdings_by_")
          ? "reports/holdings"
          : `reports/${route}`;
      const params = route.startsWith("holdings_by_")
        ? { ...query, aggregation: route.slice("holdings_".length) }
        : query;
      return get(resource, params);
    },
  };
}

/** A deterministic shell-only adapter for isolated frontend unit/prototype work. */
export function createSyntheticAdapter(): AdapterClient {
  const bootstrap: BootstrapPayload = {
    ledger_title: "OrangeCount",
    locale: "en",
    locales: ["en", "zh-CN"],
    theme: "system",
    routes: ["journal", "balance_sheet", "trial_balance", "account"],
    accounts: [],
    currencies: ["USD"],
    tags: [],
    links: [],
    payees: [],
    years: [],
    operating_currencies: ["USD"],
    render_commas: false,
    errors: [],
  };
  return {
    bootstrap: async () => bootstrap,
    changed: async () => false,
    load: async () => ({ state: "shell-only", rows: [] }),
  };
}
