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
  /** User-defined `query` directives, sorted by name; shown in the sidebar. */
  user_queries: { name: string; query_string: string }[];
  /** Configured attachment roots (serve --document-root); empty disables
   * document uploads. */
  document_roots: string[];
  errors: unknown[];
  /** Per-account display lifecycle data (balance, up-to-date status, last entry). */
  account_details: Record<string, { balance_string: string; close_date?: string; uptodate_status?: string; last_entry?: string }>;
  mtime?: string;
}

export interface RepairExample {
  before: string;
  after: string;
  note: string;
}

export interface RepairGuide {
  code: string;
  topic: string;
  phase: string;
  short_action: string;
  what: string;
  why: string;
  inspect: string[];
  safe_steps: string[];
  example: RepairExample;
  revalidate: string;
}

export interface DiagnosticContextLine {
  line: number;
  content: string;
}

export interface DiagnosticContext {
  available: boolean;
  path?: string;
  focus_line?: number;
  lines?: DiagnosticContextLine[];
  reason?: string;
}

export interface AdapterClient {
  bootstrap(): Promise<BootstrapPayload>;
  changed(): Promise<boolean>;
  load(route: string, query?: Record<string, string>): Promise<unknown>;
  guide(code: string, locale?: string): Promise<RepairGuide>;
  diagnosticContext(path: string, line: number): Promise<DiagnosticContext>;
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
  user_queries?: { name: string; query_string: string }[];
  document_roots?: string[];
  account_details?: Record<string, { balance_string: string; close_date?: string; uptodate_status?: string; last_entry?: string }>;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function requiredString(value: unknown, field: string): string {
  if (typeof value !== "string" || value.trim() === "") throw new Error(`Invalid repair guidance: ${field} is required`);
  return value;
}

function stringList(value: unknown, field: string): string[] {
  if (!Array.isArray(value) || !value.every((item) => typeof item === "string" && item.trim() !== "")) {
    throw new Error(`Invalid repair guidance: ${field} must be a non-empty string list`);
  }
  return value as string[];
}

function parseRepairGuide(value: unknown): RepairGuide {
  if (!isRecord(value) || !isRecord(value.example)) throw new Error("Invalid repair guidance response");
  const example = value.example;
  return {
    code: requiredString(value.code, "code"),
    topic: requiredString(value.topic, "topic"),
    phase: requiredString(value.phase, "phase"),
    short_action: requiredString(value.short_action, "short_action"),
    what: requiredString(value.what, "what"),
    why: requiredString(value.why, "why"),
    inspect: stringList(value.inspect, "inspect"),
    safe_steps: stringList(value.safe_steps, "safe_steps"),
    example: {
      before: requiredString(example.before, "example.before"),
      after: requiredString(example.after, "example.after"),
      note: requiredString(example.note, "example.note"),
    },
    revalidate: requiredString(value.revalidate, "revalidate"),
  };
}

function parseDiagnosticContext(value: unknown): DiagnosticContext {
  if (!isRecord(value) || typeof value.available !== "boolean") throw new Error("Invalid diagnostic context response");
  if (value.path !== undefined && typeof value.path !== "string") throw new Error("Invalid diagnostic context: path must be a string");
  if (value.focus_line !== undefined && (!Number.isInteger(value.focus_line) || Number(value.focus_line) <= 0)) throw new Error("Invalid diagnostic context: focus_line must be positive");
  if (!value.available) {
    if (value.reason !== undefined && typeof value.reason !== "string") throw new Error("Invalid diagnostic context: reason must be a string");
    return value as DiagnosticContext;
  }
  if (!Array.isArray(value.lines) || !value.lines.every((item) => isRecord(item) && Number.isInteger(item.line) && Number(item.line) > 0 && typeof item.content === "string")) {
    throw new Error("Invalid diagnostic context: lines are malformed");
  }
  return value as DiagnosticContext;
}

function bootstrapPayload(wire: BootstrapWire, mtime = ""): BootstrapPayload {
  const title = wire.options?.title?.trim() || "OrangeCount";
  const locale = wire.fava_options?.locale === "zh-CN" ? "zh-CN" : "en";
  return {
    ledger_title: title,
    locale,
    locales: ["en", "zh-CN"],
    theme: "system",
    routes: ["income_statement", "balance_sheet", "trial_balance", "journal", "query", "holdings", "commodities", "documents", "events", "statistics", "editor", "import", "options", "help", "diagnostics", "account"],
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
    user_queries: wire.user_queries || [],
    document_roots: wire.document_roots || [],
    account_details: wire.account_details || {},
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

  async function getContext(path: string, line: number): Promise<DiagnosticContext> {
    const params = new URLSearchParams({ path, line: String(line) });
    const response = await fetcher(`/api/v1/diagnostics/context?${params}`, { headers: { Accept: "application/json" } });
    const payload = await response.json() as DiagnosticContext & { error?: string };
    if (!response.ok) throw new Error(payload.error || `Diagnostic context request failed (${response.status})`);
    return parseDiagnosticContext(payload);
  }

  return {
    bootstrap: async () => {
      const wire = await get<BootstrapWire>("ledger_data");
      return bootstrapPayload(wire, lastMtime);
    },
    changed: async () => get<boolean>("changed", lastMtime ? { mtime: lastMtime } : {}),
    guide: async (code, locale = "en") => parseRepairGuide(await get<unknown>("help", { topic: `diagnostics/${code}`, locale })),
    diagnosticContext: getContext,
    load: (route, query = {}) => {
      const treeRoutes = new Set(["income_statement", "balance_sheet", "trial_balance"]);
      const directRoutes = new Set(["options", "help", "diagnostics", "source", "editor", "import", "journal", "entry-context"]);
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
    user_queries: [],
    document_roots: [],
    errors: [],
  };
  return {
    bootstrap: async () => bootstrap,
    changed: async () => false,
    guide: async (code) => ({ code, topic: `diagnostics/${code}`, phase: "recheck-after-semantic", short_action: "Guide unavailable in shell-only mode", what: "Guide unavailable in shell-only mode", why: "", inspect: [], safe_steps: [], example: { before: "", after: "", note: "" }, revalidate: "" }),
    diagnosticContext: async () => ({ available: false, reason: "Context unavailable in shell-only mode" }),
    load: async () => ({ state: "shell-only", rows: [] }),
  };
}
