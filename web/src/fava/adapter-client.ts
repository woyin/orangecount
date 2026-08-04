/**
 * Narrow client boundary for the private OrangeCount Fava-shaped adapter.
 *
 * The endpoint names are internal client contracts, not Fava's public API.
 * P3 supplies the loopback handlers and may change transport details without
 * making this client a public compatibility promise.
 */

export interface BootstrapPayload {
  ledger_title: string;
  locale: string;
  locales: string[];
  theme: string;
  routes: string[];
  accounts: string[];
  currencies: string[];
  errors: unknown[];
}

export interface AdapterClient {
  bootstrap(): Promise<BootstrapPayload>;
  load(route: string, query?: Record<string, string>): Promise<unknown>;
}

export const PRIVATE_ADAPTER_BASE = "/__orangecount/fava";

interface AdapterEnvelope<T> {
  data: T;
  mtime?: string;
}

export function createAdapterClient(
  fetcher: typeof fetch = fetch,
  base = PRIVATE_ADAPTER_BASE,
): AdapterClient {
  async function get<T>(resource: string, query: Record<string, string> = {}): Promise<T> {
    const params = new URLSearchParams(query);
    const response = await fetcher(`${base}/${resource}${params.size ? `?${params}` : ""}`, {
      headers: { Accept: "application/json" },
    });
    const payload = await response.json() as AdapterEnvelope<T> & { error?: string };
    if (!response.ok) throw new Error(payload.error || `Adapter request failed (${response.status})`);
    return payload.data;
  }

  return {
    bootstrap: () => get<BootstrapPayload>("ledger_data"),
    load: (route, query = {}) => get(route, query),
  };
}

/** A deterministic shell-only adapter used until P3 publishes bootstrap data. */
export function createSyntheticAdapter(): AdapterClient {
  const bootstrap: BootstrapPayload = {
    ledger_title: "OrangeCount",
    locale: "en",
    locales: ["en", "zh-CN"],
    theme: "system",
    routes: ["journal", "balance_sheet", "trial_balance", "account"],
    accounts: [],
    currencies: ["USD"],
    errors: [],
  };
  return {
    bootstrap: async () => bootstrap,
    load: async () => ({ state: "shell-only", rows: [] }),
  };
}
